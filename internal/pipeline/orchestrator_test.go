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
)

type lockRepoStub struct {
	mu       sync.Mutex
	locked   map[string]bool
	acquired []string
	released []string
}

func newLockRepoStub() *lockRepoStub {
	return &lockRepoStub{locked: make(map[string]bool)}
}

func (l *lockRepoStub) AcquireWalletLock(_ context.Context, walletAddress string, _ int) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.acquired = append(l.acquired, walletAddress)
	if l.locked[walletAddress] {
		return false, nil
	}
	l.locked[walletAddress] = true
	return true, nil
}

func (l *lockRepoStub) ReleaseWalletLock(_ context.Context, walletAddress string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.locked, walletAddress)
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
		WithWalletRunner(func(_ context.Context, _ string, _ RunParams, limits WalletRunLimits) error {
			called = true
			gotLimits = limits
			return nil
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
}

func TestRunReturnsLockErrorWhenWalletAlreadyLocked(t *testing.T) {
	lockRepo := newLockRepoStub()
	lockRepo.locked["walletA"] = true
	cfg := testConfig()

	orch := NewOrchestrator(
		cfg,
		WithWalletLockRepository(lockRepo),
		WithWalletRunner(func(context.Context, string, RunParams, WalletRunLimits) error {
			t.Fatal("runner should not be called when lock is unavailable")
			return nil
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
		WithWalletRunner(func(ctx context.Context, _ string, _ RunParams, _ WalletRunLimits) error {
			<-ctx.Done()
			return ctx.Err()
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
