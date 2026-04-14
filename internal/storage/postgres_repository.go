package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"poisontrace/internal/runs"
	"poisontrace/internal/transactions"
)

func (s *PostgresStore) CreateIngestionRun(ctx context.Context, startedAt time.Time) (int64, error) {
	const q = `
INSERT INTO ingestion_runs (status, started_at)
VALUES ($1, $2)
RETURNING id`
	var id int64
	if err := s.DB.QueryRowContext(ctx, q, runs.RunStatusRunning, startedAt.UTC()).Scan(&id); err != nil {
		return 0, fmt.Errorf("create ingestion run: %w", err)
	}
	return id, nil
}

func (s *PostgresStore) FinalizeIngestionRun(ctx context.Context, runID int64, status runs.RunStatus, completedAt time.Time, notes string) error {
	const q = `
UPDATE ingestion_runs
SET status = $2,
    completed_at = $3,
    notes = $4
WHERE id = $1`
	res, err := s.DB.ExecContext(ctx, q, runID, status, completedAt.UTC(), nullableText(notes))
	if err != nil {
		return fmt.Errorf("finalize ingestion run %d: %w", runID, err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("finalize ingestion run %d rows affected: %w", runID, err)
	}
	if rows == 0 {
		return fmt.Errorf("finalize ingestion run %d: not found", runID)
	}
	return nil
}

func (s *PostgresStore) CreateWalletSyncRun(ctx context.Context, runID, walletID int64, window runs.WalletSyncWindow, startedAt time.Time) (int64, error) {
	const q = `
INSERT INTO wallet_sync_runs (
  wallet_id,
  ingestion_run_id,
  status,
  started_at,
  baseline_start_at,
  baseline_end_at,
  scan_start_at,
  scan_end_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id`
	var id int64
	if err := s.DB.QueryRowContext(
		ctx,
		q,
		walletID,
		runID,
		runs.WalletStatusRunning,
		startedAt.UTC(),
		window.BaselineStart.UTC(),
		window.BaselineEnd.UTC(),
		window.ScanStart.UTC(),
		window.ScanEnd.UTC(),
	).Scan(&id); err != nil {
		return 0, fmt.Errorf("create wallet sync run for wallet %d: %w", walletID, err)
	}
	return id, nil
}

func (s *PostgresStore) FinalizeWalletSyncRun(ctx context.Context, walletSyncRunID int64, status runs.WalletStatus, completedAt time.Time, incompleteWindow bool, unknownGateReason, notes string) error {
	const q = `
UPDATE wallet_sync_runs
SET status = $2,
    completed_at = $3,
    incomplete_window = $4,
    unknown_gate_reason = $5,
    notes = $6
WHERE id = $1`
	reason := nullableText(unknownGateReason)
	if !incompleteWindow {
		reason = nil
	}
	res, err := s.DB.ExecContext(ctx, q, walletSyncRunID, status, completedAt.UTC(), incompleteWindow, reason, nullableText(notes))
	if err != nil {
		return fmt.Errorf("finalize wallet sync run %d: %w", walletSyncRunID, err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("finalize wallet sync run %d rows affected: %w", walletSyncRunID, err)
	}
	if rows == 0 {
		return fmt.Errorf("finalize wallet sync run %d: not found", walletSyncRunID)
	}
	return nil
}

func (s *PostgresStore) UpsertNormalizedTransfers(ctx context.Context, transfers []transactions.NormalizedTransfer) (inserted int, updated int, err error) {
	if len(transfers) == 0 {
		return 0, 0, nil
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("begin upsert normalized transfers: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO transactions (
  signature,
  transfer_index,
  transfer_fingerprint,
  slot,
  block_time,
  source_owner_address,
  destination_owner_address,
  source_token_account,
  destination_token_account,
  amount_raw,
  token_mint,
  asset_type,
  asset_key,
  decimals,
  normalization_status,
  normalization_reason_code,
  poisoning_eligible,
  dust_status,
  is_success
)
VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
)
ON CONFLICT (signature, transfer_fingerprint) DO UPDATE SET
  transfer_index = EXCLUDED.transfer_index,
  slot = EXCLUDED.slot,
  block_time = EXCLUDED.block_time,
  source_owner_address = EXCLUDED.source_owner_address,
  destination_owner_address = EXCLUDED.destination_owner_address,
  source_token_account = EXCLUDED.source_token_account,
  destination_token_account = EXCLUDED.destination_token_account,
  amount_raw = EXCLUDED.amount_raw,
  token_mint = EXCLUDED.token_mint,
  asset_type = EXCLUDED.asset_type,
  asset_key = EXCLUDED.asset_key,
  decimals = EXCLUDED.decimals,
  normalization_status = EXCLUDED.normalization_status,
  normalization_reason_code = EXCLUDED.normalization_reason_code,
  poisoning_eligible = EXCLUDED.poisoning_eligible,
  dust_status = EXCLUDED.dust_status,
  is_success = EXCLUDED.is_success
RETURNING (xmax = 0)`)
	if err != nil {
		return 0, 0, fmt.Errorf("prepare transfer upsert: %w", err)
	}
	defer stmt.Close()

	for _, tr := range transfers {
		var wasInserted bool
		if err = stmt.QueryRowContext(
			ctx,
			tr.Signature,
			tr.TransferIndex,
			tr.TransferFingerprint,
			tr.Slot,
			tr.BlockTime.UTC(),
			nullableText(tr.SourceOwnerAddress),
			nullableText(tr.DestinationOwnerAddress),
			nullableText(tr.SourceTokenAccount),
			nullableText(tr.DestinationTokenAccount),
			tr.AmountRaw,
			nullableText(tr.TokenMint),
			tr.AssetType,
			tr.AssetKey,
			nullableInt(tr.Decimals),
			tr.NormalizationStatus,
			nullableText(tr.NormalizationReasonCode),
			tr.PoisoningEligible,
			tr.DustStatus,
			tr.IsSuccess,
		).Scan(&wasInserted); err != nil {
			return 0, 0, fmt.Errorf("upsert transfer signature=%s fingerprint=%s: %w", tr.Signature, tr.TransferFingerprint, err)
		}
		if wasInserted {
			inserted++
		} else {
			updated++
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("commit transfer upserts: %w", err)
	}
	return inserted, updated, nil
}

func (s *PostgresStore) AcquireWalletLock(ctx context.Context, walletAddress string, ttlSeconds int) (bool, error) {
	if walletAddress == "" {
		return false, fmt.Errorf("wallet address is required")
	}
	if ttlSeconds < 1 {
		return false, fmt.Errorf("ttlSeconds must be >= 1")
	}

	holder := fmt.Sprintf("pid:%d", time.Now().UnixNano())
	const q = `
INSERT INTO wallet_locks (wallet_address, acquired_at, acquired_until, holder_token, updated_at)
VALUES ($1, NOW(), NOW() + ($2 * INTERVAL '1 second'), $3, NOW())
ON CONFLICT (wallet_address) DO UPDATE SET
  acquired_at = NOW(),
  acquired_until = NOW() + ($2 * INTERVAL '1 second'),
  holder_token = EXCLUDED.holder_token,
  updated_at = NOW()
WHERE wallet_locks.acquired_until <= NOW()`

	res, err := s.DB.ExecContext(ctx, q, walletAddress, ttlSeconds, holder)
	if err != nil {
		return false, fmt.Errorf("acquire wallet lock %s: %w", walletAddress, err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("acquire wallet lock %s rows affected: %w", walletAddress, err)
	}
	return rows > 0, nil
}

func (s *PostgresStore) ReleaseWalletLock(ctx context.Context, walletAddress string) error {
	if walletAddress == "" {
		return fmt.Errorf("wallet address is required")
	}
	const q = `DELETE FROM wallet_locks WHERE wallet_address = $1`
	if _, err := s.DB.ExecContext(ctx, q, walletAddress); err != nil {
		return fmt.Errorf("release wallet lock %s: %w", walletAddress, err)
	}
	return nil
}

func nullableText(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func nullableInt(v *int) any {
	if v == nil {
		return nil
	}
	return sql.NullInt64{
		Int64: int64(*v),
		Valid: true,
	}
}
