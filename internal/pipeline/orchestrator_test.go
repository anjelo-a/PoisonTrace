package pipeline

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"poisontrace/internal/config"
	"poisontrace/internal/runs"
	"poisontrace/internal/storage"
)

type lockRepoStub struct {
	mu          sync.Mutex
	locked      map[string]bool
	tokens      map[string]string
	acquired    []string
	acquiredTTL []int
	released    []string
}

type runRepoStub struct {
	mu            sync.Mutex
	nextID        int64
	finalized     bool
	finalStatus   runs.RunStatus
	finalCounters runs.Counters
}

func (r *runRepoStub) CreateIngestionRun(_ context.Context, _ time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.nextID == 0 {
		r.nextID = 1
	}
	return r.nextID, nil
}

func (r *runRepoStub) FinalizeIngestionRun(_ context.Context, _ int64, status runs.RunStatus, _ time.Time, counters runs.Counters, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.finalized = true
	r.finalStatus = status
	r.finalCounters = counters
	return nil
}

func (r *runRepoStub) CreateWalletSyncRun(context.Context, int64, int64, runs.WalletSyncWindow, time.Time) (int64, error) {
	return 0, errors.New("not used")
}

func (r *runRepoStub) UpdateWalletSyncProgress(context.Context, int64, storage.WalletSyncProgress) error {
	return errors.New("not used")
}

func (r *runRepoStub) FinalizeWalletSyncRun(context.Context, int64, runs.WalletStatus, time.Time, bool, string, string, string, string) error {
	return errors.New("not used")
}

func newLockRepoStub() *lockRepoStub {
	return &lockRepoStub{
		locked: make(map[string]bool),
		tokens: make(map[string]string),
	}
}

func (l *lockRepoStub) AcquireWalletLock(_ context.Context, walletAddress string, ttlSeconds int) (bool, string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.acquired = append(l.acquired, walletAddress)
	l.acquiredTTL = append(l.acquiredTTL, ttlSeconds)
	if l.locked[walletAddress] {
		return false, "", nil
	}
	l.locked[walletAddress] = true
	token := walletAddress + "-token"
	l.tokens[walletAddress] = token
	return true, token, nil
}

func (l *lockRepoStub) ReleaseWalletLock(_ context.Context, walletAddress string, holderToken string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.tokens[walletAddress] != holderToken {
		return nil
	}
	delete(l.locked, walletAddress)
	delete(l.tokens, walletAddress)
	l.released = append(l.released, walletAddress)
	return nil
}

func testConfig() config.Config {
	return config.Config{
		MaxWalletsPerRun:         10,
		MaxTXPagesPerWallet:      20,
		MaxTXPerWallet:           1500,
		MaxConcurrentWallets:     2,
		WalletSyncTimeoutSeconds: 1,
		RunTimeoutSeconds:        5,
		HeliusRequestDelayMS:     25,
		MaxHeliusRetries:         2,
		BaselineLookbackDays:     90,
		LookalikeRecencyDays:     30,
		LookalikePrefixMin:       4,
		LookalikeSuffixMin:       4,
		LookalikeSingleSideMin:   6,
		MinInjectionCount:        2,
	}
}

func writeWalletFile(t *testing.T, wallets []string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "wallets.txt")
	content := strings.Join(wallets, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write wallet file: %v", err)
	}
	return path
}

func TestRunUsesWalletLockAndRunnerLimits(t *testing.T) {
	lockRepo := newLockRepoStub()
	cfg := testConfig()
	var gotLimits WalletRunLimits
	var called bool

	orch := NewOrchestrator(
		cfg,
		WithWalletLockRepository(lockRepo),
		WithWalletRunner(func(_ context.Context, _ string, _ RunParams, limits WalletRunLimits) (WalletRunReport, error) {
			called = true
			gotLimits = limits
			return WalletRunReport{}, nil
		}),
	)

	err := orch.Run(context.Background(), RunParams{
		WalletFile:           writeWalletFile(t, []string{"walletA"}),
		ScanStart:            time.Now().UTC().Add(-1 * time.Hour),
		ScanEnd:              time.Now().UTC(),
		BaselineLookbackDays: 90,
	})
	if err != nil {
		t.Fatalf("run returned unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected wallet runner to be called")
	}
	if gotLimits.MaxTXPagesPerWallet != cfg.MaxTXPagesPerWallet || gotLimits.MaxTXPerWallet != cfg.MaxTXPerWallet {
		t.Fatal("wallet runner did not receive configured bounds")
	}
	if len(lockRepo.acquired) != 1 || len(lockRepo.released) != 1 {
		t.Fatalf("expected one lock acquire/release, got acquired=%d released=%d", len(lockRepo.acquired), len(lockRepo.released))
	}
	wantTTL := cfg.WalletSyncTimeoutSeconds + walletLockTTLTailSeconds
	if len(lockRepo.acquiredTTL) != 1 || lockRepo.acquiredTTL[0] != wantTTL {
		t.Fatalf("expected lock ttl=%d, got %+v", wantTTL, lockRepo.acquiredTTL)
	}
}

func TestRunReturnsLockErrorWhenWalletAlreadyLocked(t *testing.T) {
	lockRepo := newLockRepoStub()
	lockRepo.locked["walletA"] = true
	cfg := testConfig()

	orch := NewOrchestrator(
		cfg,
		WithWalletLockRepository(lockRepo),
		WithWalletRunner(func(context.Context, string, RunParams, WalletRunLimits) (WalletRunReport, error) {
			t.Fatal("runner should not be called when lock is unavailable")
			return WalletRunReport{}, nil
		}),
	)

	err := orch.Run(context.Background(), RunParams{
		WalletFile:           writeWalletFile(t, []string{"walletA"}),
		ScanStart:            time.Now().UTC().Add(-2 * time.Hour),
		ScanEnd:              time.Now().UTC().Add(-1 * time.Hour),
		BaselineLookbackDays: 90,
	})
	if err == nil {
		t.Fatal("expected run to fail when wallet lock cannot be acquired")
	}
	if !strings.Contains(err.Error(), ErrWalletAlreadyLocked.Error()) {
		t.Fatalf("expected lock error, got: %v", err)
	}
}

func TestRunPropagatesWalletContextTimeout(t *testing.T) {
	lockRepo := newLockRepoStub()
	cfg := testConfig()
	cfg.WalletSyncTimeoutSeconds = 1
	cfg.RunTimeoutSeconds = 10

	orch := NewOrchestrator(
		cfg,
		WithWalletLockRepository(lockRepo),
		WithWalletRunner(func(ctx context.Context, _ string, _ RunParams, _ WalletRunLimits) (WalletRunReport, error) {
			<-ctx.Done()
			return WalletRunReport{}, ctx.Err()
		}),
	)

	start := time.Now()
	err := orch.Run(context.Background(), RunParams{
		WalletFile:           writeWalletFile(t, []string{"walletA"}),
		ScanStart:            time.Now().UTC().Add(-2 * time.Hour),
		ScanEnd:              time.Now().UTC().Add(-1 * time.Hour),
		BaselineLookbackDays: 90,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
		t.Fatalf("expected deadline exceeded, got: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 4*time.Second {
		t.Fatalf("expected wallet timeout bound near 1 second, elapsed=%v", elapsed)
	}
}

func TestRunAggregatesPoisoningCandidateCounters(t *testing.T) {
	cfg := testConfig()
	runRepo := &runRepoStub{}

	orch := NewOrchestrator(
		cfg,
		WithRunRepository(runRepo),
		WithWalletRunner(func(_ context.Context, wallet string, _ RunParams, _ WalletRunLimits) (WalletRunReport, error) {
			report := WalletRunReport{WalletStatus: runs.WalletStatusSucceeded}
			report.Counters.TransactionsFetched = 1
			report.Counters.PoisoningCandidatesInserted = 1
			if wallet == "walletB" {
				report.Counters.PoisoningCandidatesInserted = 2
			}
			return report, nil
		}),
	)

	err := orch.Run(context.Background(), RunParams{
		WalletFile:           writeWalletFile(t, []string{"walletA", "walletB"}),
		ScanStart:            time.Now().UTC().Add(-2 * time.Hour),
		ScanEnd:              time.Now().UTC().Add(-1 * time.Hour),
		BaselineLookbackDays: 90,
	})
	if err != nil {
		t.Fatalf("run returned unexpected error: %v", err)
	}

	if !runRepo.finalized {
		t.Fatal("expected ingestion run to be finalized")
	}
	if runRepo.finalCounters.PoisoningCandidatesInserted != 3 {
		t.Fatalf("expected aggregated poisoning candidate count=3, got %+v", runRepo.finalCounters)
	}
}
