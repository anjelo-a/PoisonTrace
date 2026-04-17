package pipeline

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"poisontrace/internal/counterparties"
	"poisontrace/internal/helius"
	"poisontrace/internal/runs"
	"poisontrace/internal/storage"
	"poisontrace/internal/transactions"
)

type walletRunnerStoreStub struct {
	mu sync.Mutex

	nextWalletID       int64
	nextWalletSyncRun  int64
	walletLastSyncedAt map[int64]time.Time

	insertedTransfers map[string]struct{}
	linkedTransfers   map[string]struct{}
	counterparties    map[string]bool

	progresses []storage.WalletSyncProgress
	finalized  []struct {
		status       runs.WalletStatus
		incomplete   bool
		unknown      string
		errorCode    string
		errorMessage string
	}

	dustThresholds []storage.DustThresholdRecord
}

func newWalletRunnerStoreStub() *walletRunnerStoreStub {
	return &walletRunnerStoreStub{
		nextWalletID:       1,
		nextWalletSyncRun:  100,
		walletLastSyncedAt: make(map[int64]time.Time),
		insertedTransfers:  make(map[string]struct{}),
		linkedTransfers:    make(map[string]struct{}),
		counterparties:     make(map[string]bool),
		dustThresholds: []storage.DustThresholdRecord{
			{
				AssetKey:   "SOL",
				AmountRaw:  "100",
				ActiveFrom: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}
}

func (s *walletRunnerStoreStub) CreateIngestionRun(context.Context, time.Time) (int64, error) {
	return 0, fmt.Errorf("not used")
}

func (s *walletRunnerStoreStub) FinalizeIngestionRun(context.Context, int64, runs.RunStatus, time.Time, runs.Counters, string) error {
	return fmt.Errorf("not used")
}

func (s *walletRunnerStoreStub) EnsureWallet(context.Context, string) (int64, error) {
	return s.nextWalletID, nil
}

func (s *walletRunnerStoreStub) CreateWalletSyncRun(context.Context, int64, int64, runs.WalletSyncWindow, time.Time) (int64, error) {
	return s.nextWalletSyncRun, nil
}

func (s *walletRunnerStoreStub) UpdateWalletSyncProgress(_ context.Context, _ int64, progress storage.WalletSyncProgress) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.progresses = append(s.progresses, progress)
	return nil
}

func (s *walletRunnerStoreStub) FinalizeWalletSyncRun(_ context.Context, _ int64, status runs.WalletStatus, _ time.Time, incompleteWindow bool, unknownGateReason, errorCode, errorMessage, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.finalized = append(s.finalized, struct {
		status       runs.WalletStatus
		incomplete   bool
		unknown      string
		errorCode    string
		errorMessage string
	}{
		status:       status,
		incomplete:   incompleteWindow,
		unknown:      unknownGateReason,
		errorCode:    errorCode,
		errorMessage: errorMessage,
	})
	return nil
}

func (s *walletRunnerStoreStub) UpdateWalletLastSyncedAt(_ context.Context, walletID int64, syncedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.walletLastSyncedAt[walletID] = syncedAt.UTC()
	return nil
}

func (s *walletRunnerStoreStub) UpsertNormalizedTransfers(_ context.Context, transfers []transactions.NormalizedTransfer) (int, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inserted := 0
	for _, tr := range transfers {
		k := tr.Signature + "|" + tr.TransferFingerprint
		if _, ok := s.insertedTransfers[k]; ok {
			continue
		}
		s.insertedTransfers[k] = struct{}{}
		inserted++
	}
	return inserted, len(transfers) - inserted, nil
}

func (s *walletRunnerStoreStub) LinkWalletTransfer(_ context.Context, walletID int64, relationType counterparties.RelationType, transfer transactions.NormalizedTransfer) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := fmt.Sprintf("%d|%s|%s|%s", walletID, relationType, transfer.Signature, transfer.TransferFingerprint)
	if _, ok := s.linkedTransfers[k]; ok {
		return false, nil
	}
	s.linkedTransfers[k] = struct{}{}
	return true, nil
}

func (s *walletRunnerStoreStub) UpsertCounterpartyEvent(_ context.Context, event counterparties.Event) (bool, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := fmt.Sprintf("%d|%s", event.FocalWalletID, event.CounterpartyAddress)
	if !s.counterparties[k] {
		s.counterparties[k] = true
		return true, false, nil
	}
	return false, true, nil
}

func (s *walletRunnerStoreStub) InsertPoisoningCandidates(_ context.Context, _ int64, _ int64, candidates []storage.CandidateRecord) (int, error) {
	return len(candidates), nil
}

func (s *walletRunnerStoreStub) ListDustThresholds(context.Context, time.Time, time.Time) ([]storage.DustThresholdRecord, error) {
	return s.dustThresholds, nil
}

func TestWalletExecutionRunner_SucceededPath(t *testing.T) {
	baselineEnd := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	scanStart := baselineEnd
	scanEnd := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)

	client := &scriptedClient{
		responses: []helius.EnhancedPage{
			{Transactions: []helius.EnhancedTransaction{nativeTx("b1", time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC), "walletA", "LegitABCDxyzz", "1000")}},
			{},
			{Transactions: []helius.EnhancedTransaction{
				nativeTx("s1", time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC), "LegitWXYZxyzz", "walletA", "0"),
				nativeTx("s2", time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC), "LegitWXYZxyzz", "walletA", "0"),
			}},
			{},
		},
	}

	store := newWalletRunnerStoreStub()
	runner := &WalletExecutionRunner{
		cfg:    testConfig(),
		store:  store,
		client: client,
	}

	report, err := runner.RunWallet(context.Background(), "walletA", RunParams{
		ScanStart:            scanStart,
		ScanEnd:              scanEnd,
		BaselineLookbackDays: 90,
		IngestionRunID:       55,
	}, WalletRunLimits{MaxTXPagesPerWallet: 5, MaxTXPerWallet: 100, MaxHeliusRetries: 1})
	if err != nil {
		t.Fatalf("run wallet failed: %v", err)
	}
	if report.WalletStatus != runs.WalletStatusSucceeded {
		t.Fatalf("expected succeeded wallet status, got %s", report.WalletStatus)
	}
	if report.Counters.TransactionsInserted == 0 || report.Counters.TransactionsLinked == 0 {
		t.Fatalf("expected persisted transfer counters, got %+v", report.Counters)
	}
	if len(store.progresses) != 1 {
		t.Fatalf("expected one progress update, got %d", len(store.progresses))
	}
	if store.progresses[0].PoisoningCandidatesInserted != report.Counters.PoisoningCandidatesInserted {
		t.Fatalf("expected progress candidate counter=%d, got %+v", report.Counters.PoisoningCandidatesInserted, store.progresses[0])
	}
	if store.progresses[0].IncompleteWindow {
		t.Fatalf("expected complete window, got %+v", store.progresses[0])
	}
	if len(store.finalized) != 1 || store.finalized[0].status != runs.WalletStatusSucceeded {
		t.Fatalf("expected succeeded finalize, got %+v", store.finalized)
	}
	if _, ok := store.walletLastSyncedAt[store.nextWalletID]; !ok {
		t.Fatal("expected wallet last_synced_at update")
	}
}

func TestWalletExecutionRunner_PartialOnBaselineTruncation(t *testing.T) {
	scanStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	scanEnd := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)

	client := &scriptedClient{
		responses: []helius.EnhancedPage{
			{
				Transactions: []helius.EnhancedTransaction{nativeTx("b1", time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC), "walletA", "LegitABCDxyzz", "1000")},
				Before:       "next",
			},
			{Transactions: []helius.EnhancedTransaction{
				nativeTx("s1", time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC), "LegitWXYZxyzz", "walletA", "0"),
				nativeTx("s2", time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC), "LegitWXYZxyzz", "walletA", "0"),
			}},
			{},
		},
	}

	store := newWalletRunnerStoreStub()
	runner := &WalletExecutionRunner{
		cfg:    testConfig(),
		store:  store,
		client: client,
	}

	report, err := runner.RunWallet(context.Background(), "walletA", RunParams{
		ScanStart:            scanStart,
		ScanEnd:              scanEnd,
		BaselineLookbackDays: 90,
		IngestionRunID:       88,
	}, WalletRunLimits{MaxTXPagesPerWallet: 1, MaxTXPerWallet: 100, MaxHeliusRetries: 1})
	if err != nil {
		t.Fatalf("run wallet failed: %v", err)
	}
	if report.WalletStatus != runs.WalletStatusPartial {
		t.Fatalf("expected partial wallet status, got %s", report.WalletStatus)
	}
	if len(store.progresses) != 1 || !store.progresses[0].IncompleteWindow {
		t.Fatalf("expected incomplete progress, got %+v", store.progresses)
	}
	if len(store.finalized) != 1 || !store.finalized[0].incomplete {
		t.Fatalf("expected finalized incomplete=true, got %+v", store.finalized)
	}
	if store.finalized[0].unknown == "" {
		t.Fatal("expected unknown gate reason on finalized partial wallet")
	}
}

func TestBuildDustClassifier_EffectiveTimeAwareThresholds(t *testing.T) {
	store := newWalletRunnerStoreStub()
	store.dustThresholds = []storage.DustThresholdRecord{
		{
			AssetKey:   "SOL",
			AmountRaw:  "100",
			ActiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			ActiveTo:   ptrTime(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)),
		},
		{
			AssetKey:   "SOL",
			AmountRaw:  "10",
			ActiveFrom: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	runner := &WalletExecutionRunner{store: store}
	classify, err := runner.buildDustClassifier(
		context.Background(),
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("buildDustClassifier failed: %v", err)
	}

	beforeCutover := classify(transactions.NormalizedTransfer{
		AssetKey:   "SOL",
		AmountRaw:  "50",
		BlockTime:  time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		DustStatus: transactions.DustUnknown,
	})
	if beforeCutover != transactions.DustTrue {
		t.Fatalf("expected pre-cutover SOL threshold to classify dust=true, got %s", beforeCutover)
	}

	onCutover := classify(transactions.NormalizedTransfer{
		AssetKey:  "SOL",
		AmountRaw: "10",
		BlockTime: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	})
	if onCutover != transactions.DustTrue {
		t.Fatalf("expected cutover timestamp to use new threshold, got %s", onCutover)
	}

	afterCutover := classify(transactions.NormalizedTransfer{
		AssetKey:  "SOL",
		AmountRaw: "50",
		BlockTime: time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC),
	})
	if afterCutover != transactions.DustFalse {
		t.Fatalf("expected post-cutover SOL threshold to classify dust=false, got %s", afterCutover)
	}

	noRule := classify(transactions.NormalizedTransfer{
		AssetKey:  "UNKNOWN_MINT",
		AmountRaw: "1",
		BlockTime: time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC),
	})
	if noRule != transactions.DustUnknown {
		t.Fatalf("expected missing threshold to classify dust=unknown, got %s", noRule)
	}
}

func TestBuildDustClassifier_RejectsInvalidThresholdRows(t *testing.T) {
	baseStart := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	baseEnd := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		thresholds  []storage.DustThresholdRecord
		wantErrPart string
	}{
		{
			name: "empty_asset_key",
			thresholds: []storage.DustThresholdRecord{
				{
					AssetKey:   " ",
					AmountRaw:  "100",
					ActiveFrom: baseStart,
				},
			},
			wantErrPart: "asset_key is empty",
		},
		{
			name: "invalid_amount",
			thresholds: []storage.DustThresholdRecord{
				{
					AssetKey:   "SOL",
					AmountRaw:  "not_numeric",
					ActiveFrom: baseStart,
				},
			},
			wantErrPart: "dust_amount_raw_threshold",
		},
		{
			name: "invalid_window",
			thresholds: []storage.DustThresholdRecord{
				{
					AssetKey:   "SOL",
					AmountRaw:  "100",
					ActiveFrom: baseStart,
					ActiveTo:   ptrTime(baseStart),
				},
			},
			wantErrPart: "active_to must be greater than active_from",
		},
		{
			name: "overlapping_windows",
			thresholds: []storage.DustThresholdRecord{
				{
					AssetKey:   "SOL",
					AmountRaw:  "100",
					ActiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					ActiveTo:   ptrTime(time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)),
				},
				{
					AssetKey:   "SOL",
					AmountRaw:  "10",
					ActiveFrom: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			wantErrPart: "overlapping dust threshold windows",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			store := newWalletRunnerStoreStub()
			store.dustThresholds = tc.thresholds
			runner := &WalletExecutionRunner{store: store}
			_, err := runner.buildDustClassifier(context.Background(), baseStart, baseEnd)
			if err == nil {
				t.Fatal("expected buildDustClassifier to fail")
			}
			if !strings.Contains(err.Error(), tc.wantErrPart) {
				t.Fatalf("expected error containing %q, got %q", tc.wantErrPart, err.Error())
			}
		})
	}
}

func ptrTime(v time.Time) *time.Time {
	return &v
}
