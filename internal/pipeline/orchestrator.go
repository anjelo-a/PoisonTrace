package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"poisontrace/internal/config"
	"poisontrace/internal/runs"
	"poisontrace/internal/storage"
	"poisontrace/internal/wallets"
)

type Orchestrator struct {
	cfg          config.Config
	runRepo      storage.RunRepository
	walletLocks  storage.WalletLockRepository
	walletRunner WalletRunnerFunc
}

type RunParams struct {
	WalletFile            string
	ScanStart             time.Time
	ScanEnd               time.Time
	BaselineLookbackDays  int
	RequestedByCLICommand string
	IngestionRunID        int64
}

type WalletRunLimits struct {
	MaxTXPagesPerWallet int
	MaxTXPerWallet      int
	MaxHeliusRetries    int
	HeliusRequestDelay  time.Duration
}

type WalletRunReport struct {
	WalletStatus runs.WalletStatus
	Counters     runs.Counters
}

type WalletRunnerFunc func(ctx context.Context, walletAddress string, p RunParams, limits WalletRunLimits) (WalletRunReport, error)

var ErrWalletAlreadyLocked = errors.New("wallet sync lock is currently held")

type Option func(*Orchestrator)

func WithRunRepository(repo storage.RunRepository) Option {
	return func(o *Orchestrator) {
		o.runRepo = repo
	}
}

func WithWalletLockRepository(repo storage.WalletLockRepository) Option {
	return func(o *Orchestrator) {
		o.walletLocks = repo
	}
}

func WithWalletRunner(runner WalletRunnerFunc) Option {
	return func(o *Orchestrator) {
		o.walletRunner = runner
	}
}

func NewOrchestrator(cfg config.Config, opts ...Option) *Orchestrator {
	orch := &Orchestrator{cfg: cfg}
	for _, opt := range opts {
		opt(orch)
	}
	return orch
}

type walletOutcome struct {
	address string
	report  WalletRunReport
	err     error
}

func (o *Orchestrator) Run(ctx context.Context, p RunParams) error {
	if !p.ScanStart.Before(p.ScanEnd) {
		return fmt.Errorf("scan window must satisfy scan_start < scan_end")
	}
	if p.BaselineLookbackDays < 1 {
		p.BaselineLookbackDays = o.cfg.BaselineLookbackDays
	}
	if p.BaselineLookbackDays <= 0 {
		return fmt.Errorf("baseline lookback days must be >= 1")
	}

	runCtx, cancelRun := context.WithTimeout(ctx, time.Duration(o.cfg.RunTimeoutSeconds)*time.Second)
	defer cancelRun()

	walletList, err := wallets.LoadAddressesFromFile(p.WalletFile, o.cfg.MaxWalletsPerRun)
	if err != nil {
		return err
	}
	if len(walletList) == 0 {
		return fmt.Errorf("no wallets to process")
	}

	if o.walletRunner == nil {
		return fmt.Errorf("wallet runner is not configured")
	}

	total := runs.Counters{WalletsRequested: len(walletList)}
	if o.runRepo != nil {
		runID, createErr := o.runRepo.CreateIngestionRun(runCtx, time.Now().UTC())
		if createErr != nil {
			return fmt.Errorf("create ingestion run: %w", createErr)
		}
		p.IngestionRunID = runID
	}

	sem := make(chan struct{}, o.cfg.MaxConcurrentWallets)
	var wg sync.WaitGroup
	outcomeCh := make(chan walletOutcome, len(walletList))

	for _, addr := range walletList {
		addr := addr
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-runCtx.Done():
				outcomeCh <- walletOutcome{address: addr, err: runCtx.Err()}
				return
			}
			defer func() { <-sem }()

			walletCtx, cancel := context.WithTimeout(runCtx, time.Duration(o.cfg.WalletSyncTimeoutSeconds)*time.Second)
			defer cancel()
			report, runErr := o.runWallet(walletCtx, addr, p)
			outcomeCh <- walletOutcome{address: addr, report: report, err: runErr}
		}()
	}

	wg.Wait()
	close(outcomeCh)

	errs := make([]string, 0, len(walletList))
	partialWallets := 0
	for outcome := range outcomeCh {
		total.TransactionsFetched += outcome.report.Counters.TransactionsFetched
		total.TransactionsInserted += outcome.report.Counters.TransactionsInserted
		total.TransactionsLinked += outcome.report.Counters.TransactionsLinked
		total.TransactionsFailedNormalize += outcome.report.Counters.TransactionsFailedNormalize
		total.OwnerUnresolvedCount += outcome.report.Counters.OwnerUnresolvedCount
		total.DecimalsUnresolvedCount += outcome.report.Counters.DecimalsUnresolvedCount
		total.CounterpartiesCreated += outcome.report.Counters.CounterpartiesCreated
		total.CounterpartiesUpdated += outcome.report.Counters.CounterpartiesUpdated
		total.RetryExhaustedCount += outcome.report.Counters.RetryExhaustedCount

		switch {
		case outcome.err == nil:
			total.WalletsProcessed++
			if outcome.report.WalletStatus == runs.WalletStatusPartial {
				partialWallets++
			}
		case errors.Is(outcome.err, ErrWalletAlreadyLocked):
			total.WalletsSkipped++
			errs = append(errs, fmt.Sprintf("wallet %s: %v", outcome.address, outcome.err))
		default:
			total.WalletsFailed++
			errs = append(errs, fmt.Sprintf("wallet %s: %v", outcome.address, outcome.err))
		}
	}

	notes := strings.Join(errs, "; ")
	runStatus := deriveRunStatus(runCtx.Err(), total, len(errs), partialWallets)
	if o.runRepo != nil {
		if finalizeErr := o.runRepo.FinalizeIngestionRun(context.Background(), p.IngestionRunID, runStatus, time.Now().UTC(), total, notes); finalizeErr != nil {
			if len(errs) == 0 {
				return fmt.Errorf("finalize ingestion run: %w", finalizeErr)
			}
			errs = append(errs, fmt.Sprintf("finalize ingestion run: %v", finalizeErr))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("run completed with wallet errors: %s", strings.Join(errs, "; "))
	}
	if runCtx.Err() != nil {
		return runCtx.Err()
	}
	return nil
}

func deriveRunStatus(runErr error, total runs.Counters, errorCount int, partialCount int) runs.RunStatus {
	if errors.Is(runErr, context.DeadlineExceeded) {
		return runs.RunStatusTimedOut
	}
	if errors.Is(runErr, context.Canceled) {
		return runs.RunStatusCancelled
	}
	if errorCount == 0 {
		if partialCount > 0 {
			return runs.RunStatusPartiallySucceeded
		}
		return runs.RunStatusSucceeded
	}
	if total.WalletsProcessed > 0 || total.WalletsSkipped > 0 {
		return runs.RunStatusPartiallySucceeded
	}
	return runs.RunStatusFailed
}

func (o *Orchestrator) runWallet(ctx context.Context, walletAddress string, p RunParams) (WalletRunReport, error) {
	limits := WalletRunLimits{
		MaxTXPagesPerWallet: o.cfg.MaxTXPagesPerWallet,
		MaxTXPerWallet:      o.cfg.MaxTXPerWallet,
		MaxHeliusRetries:    o.cfg.MaxHeliusRetries,
		HeliusRequestDelay:  time.Duration(o.cfg.HeliusRequestDelayMS) * time.Millisecond,
	}
	if limits.MaxTXPagesPerWallet < 1 || limits.MaxTXPerWallet < 1 {
		return WalletRunReport{}, fmt.Errorf("invalid wallet run bounds for %s", walletAddress)
	}

	if o.walletLocks != nil {
		acquired, err := o.walletLocks.AcquireWalletLock(ctx, walletAddress, o.cfg.WalletSyncTimeoutSeconds)
		if err != nil {
			return WalletRunReport{}, fmt.Errorf("acquire lock: %w", err)
		}
		if !acquired {
			return WalletRunReport{}, ErrWalletAlreadyLocked
		}
		defer func() {
			releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = o.walletLocks.ReleaseWalletLock(releaseCtx, walletAddress)
		}()
	}

	select {
	case <-ctx.Done():
		return WalletRunReport{}, ctx.Err()
	default:
		report, err := o.walletRunner(ctx, walletAddress, p, limits)
		if err != nil {
			return report, fmt.Errorf("wallet runner: %w", err)
		}
		return report, nil
	}
}
