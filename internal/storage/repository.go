package storage

import (
	"context"
	"time"

	"poisontrace/internal/runs"
	"poisontrace/internal/transactions"
)

type RunRepository interface {
	CreateIngestionRun(ctx context.Context, startedAt time.Time) (int64, error)
	FinalizeIngestionRun(ctx context.Context, runID int64, status runs.RunStatus, completedAt time.Time, notes string) error
	CreateWalletSyncRun(ctx context.Context, runID, walletID int64, window runs.WalletSyncWindow, startedAt time.Time) (int64, error)
	FinalizeWalletSyncRun(ctx context.Context, walletSyncRunID int64, status runs.WalletStatus, completedAt time.Time, incompleteWindow bool, unknownGateReason, notes string) error
}

type TransactionRepository interface {
	UpsertNormalizedTransfers(ctx context.Context, transfers []transactions.NormalizedTransfer) (inserted int, updated int, err error)
}

type WalletLockRepository interface {
	AcquireWalletLock(ctx context.Context, walletAddress string, ttlSeconds int) (bool, error)
	ReleaseWalletLock(ctx context.Context, walletAddress string) error
}
