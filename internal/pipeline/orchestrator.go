package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"poisontrace/internal/config"
	"poisontrace/internal/storage"
	"poisontrace/internal/wallets"
)

type Orchestrator struct {
	cfg          config.Config
	walletLocks  storage.WalletLockRepository
	walletRunner WalletRunnerFunc
}

type RunParams struct {
	WalletFile            string
	ScanStart             time.Time
	ScanEnd               time.Time
	BaselineLookbackDays  int
	RequestedByCLICommand string
}

type WalletRunLimits struct {
	MaxTXPagesPerWallet int
	MaxTXPerWallet      int
	MaxHeliusRetries    int
	HeliusRequestDelay  time.Duration
}

type WalletRunnerFunc func(ctx context.Context, walletAddress string, p RunParams, limits WalletRunLimits) error

var ErrWalletAlreadyLocked = errors.New("wallet sync lock is currently held")

type Option func(*Orchestrator)

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

	sem := make(chan struct{}, o.cfg.MaxConcurrentWallets)
	var wg sync.WaitGroup
	errCh := make(chan error, len(walletList))

	for _, addr := range walletList {
		addr := addr
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-runCtx.Done():
				errCh <- runCtx.Err()
				return
			}
			defer func() { <-sem }()

			walletCtx, cancel := context.WithTimeout(runCtx, time.Duration(o.cfg.WalletSyncTimeoutSeconds)*time.Second)
			defer cancel()
			if err := o.runWallet(walletCtx, addr, p); err != nil {
				errCh <- fmt.Errorf("wallet %s: %w", addr, err)
			}
		}()
	}

	wg.Wait()
	close(errCh)

	errs := make([]string, 0, len(walletList))
	for walletErr := range errCh {
		if walletErr != nil {
			errs = append(errs, walletErr.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("run completed with wallet errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (o *Orchestrator) runWallet(ctx context.Context, walletAddress string, p RunParams) error {
	limits := WalletRunLimits{
		MaxTXPagesPerWallet: o.cfg.MaxTXPagesPerWallet,
		MaxTXPerWallet:      o.cfg.MaxTXPerWallet,
		MaxHeliusRetries:    o.cfg.MaxHeliusRetries,
		HeliusRequestDelay:  time.Duration(o.cfg.HeliusRequestDelayMS) * time.Millisecond,
	}
	if limits.MaxTXPagesPerWallet < 1 || limits.MaxTXPerWallet < 1 {
		return fmt.Errorf("invalid wallet run bounds for %s", walletAddress)
	}

	if o.walletLocks != nil {
		acquired, err := o.walletLocks.AcquireWalletLock(ctx, walletAddress, o.cfg.WalletSyncTimeoutSeconds)
		if err != nil {
			return fmt.Errorf("acquire lock: %w", err)
		}
		if !acquired {
			return ErrWalletAlreadyLocked
		}
		defer func() {
			releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = o.walletLocks.ReleaseWalletLock(releaseCtx, walletAddress)
		}()
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		if o.walletRunner == nil {
			// TODO: Phase 1 implementation will run baseline + scan windows,
			// normalize enhanced transactions, upsert, and materialize candidates.
			return nil
		}
		if err := o.walletRunner(ctx, walletAddress, p, limits); err != nil {
			return fmt.Errorf("wallet runner: %w", err)
		}
		return nil
	}
}
