package storage

import (
	"context"
	"time"

	"poisontrace/internal/counterparties"
	"poisontrace/internal/runs"
	"poisontrace/internal/transactions"
)

type WalletSyncProgress struct {
	BaselineComplete            bool
	IncompleteWindow            bool
	UnknownGateReason           string
	TruncationReason            string
	TransactionsFetched         int
	TransactionsInserted        int
	TransactionsLinked          int
	TransactionsFailedNormalize int
	CounterpartiesCreated       int
	CounterpartiesUpdated       int
	PoisoningCandidatesInserted int
}

type CandidateRecord struct {
	Signature                string
	TransferIndex            int
	SuspiciousCounterparty   string
	MatchedLegitCounterparty string
	TokenMint                string
	AmountRaw                string
	BlockTime                time.Time
	IsZeroValue              bool
	IsDust                   bool
	IsNewCounterparty        bool
	IsInbound                bool
	LegitLastSeenAt          time.Time
	RecencyDays              int
	RepeatInjectionCount     int
	IncompleteWindow         bool
	UnknownGateReason        string
	MatchRuleVersion         string
}

type DustThresholdRecord struct {
	AssetKey   string
	AmountRaw  string
	ActiveFrom time.Time
	ActiveTo   *time.Time
}

type RunRepository interface {
	CreateIngestionRun(ctx context.Context, startedAt time.Time) (int64, error)
	FinalizeIngestionRun(ctx context.Context, runID int64, status runs.RunStatus, completedAt time.Time, counters runs.Counters, notes string) error
	CreateWalletSyncRun(ctx context.Context, runID, walletID int64, window runs.WalletSyncWindow, startedAt time.Time) (int64, error)
	UpdateWalletSyncProgress(ctx context.Context, walletSyncRunID int64, progress WalletSyncProgress) error
	FinalizeWalletSyncRun(ctx context.Context, walletSyncRunID int64, status runs.WalletStatus, completedAt time.Time, incompleteWindow bool, unknownGateReason, errorCode, errorMessage, notes string) error
}

type TransactionRepository interface {
	UpsertNormalizedTransfers(ctx context.Context, transfers []transactions.NormalizedTransfer) (inserted int, updated int, err error)
}

type WalletRepository interface {
	EnsureWallet(ctx context.Context, walletAddress string) (int64, error)
	UpdateWalletLastSyncedAt(ctx context.Context, walletID int64, syncedAt time.Time) error
}

type WalletTransactionRepository interface {
	LinkWalletTransfer(ctx context.Context, walletID int64, relationType counterparties.RelationType, transfer transactions.NormalizedTransfer) (linked bool, err error)
}

type CounterpartyRepository interface {
	UpsertCounterpartyEvent(ctx context.Context, event counterparties.Event) (created bool, updated bool, err error)
}

type CandidateRepository interface {
	InsertPoisoningCandidates(ctx context.Context, walletSyncRunID int64, focalWalletID int64, candidates []CandidateRecord) (inserted int, err error)
}

type DustThresholdRepository interface {
	ListDustThresholds(ctx context.Context, startInclusive, endExclusive time.Time) ([]DustThresholdRecord, error)
}

type WalletLockRepository interface {
	AcquireWalletLock(ctx context.Context, walletAddress string, ttlSeconds int) (bool, error)
	ReleaseWalletLock(ctx context.Context, walletAddress string) error
}
