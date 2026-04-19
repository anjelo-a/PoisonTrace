package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"poisontrace/internal/counterparties"
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

func (s *PostgresStore) FinalizeIngestionRun(ctx context.Context, runID int64, status runs.RunStatus, completedAt time.Time, counters runs.Counters, notes string) error {
	const q = `
UPDATE ingestion_runs
SET status = $2,
    completed_at = $3,
    wallets_requested = $4,
    wallets_processed = $5,
    wallets_failed = $6,
    wallets_skipped = $7,
    transactions_fetched = $8,
    transactions_inserted = $9,
    transactions_linked = $10,
    transactions_failed_to_normalize = $11,
    owner_unresolved_count = $12,
    decimals_unresolved_count = $13,
    counterparties_created = $14,
    counterparties_updated = $15,
    poisoning_candidates_inserted = $16,
    retry_exhausted_count = $17,
    notes = $18
WHERE id = $1`
	res, err := s.DB.ExecContext(
		ctx,
		q,
		runID,
		status,
		completedAt.UTC(),
		counters.WalletsRequested,
		counters.WalletsProcessed,
		counters.WalletsFailed,
		counters.WalletsSkipped,
		counters.TransactionsFetched,
		counters.TransactionsInserted,
		counters.TransactionsLinked,
		counters.TransactionsFailedNormalize,
		counters.OwnerUnresolvedCount,
		counters.DecimalsUnresolvedCount,
		counters.CounterpartiesCreated,
		counters.CounterpartiesUpdated,
		counters.PoisoningCandidatesInserted,
		counters.RetryExhaustedCount,
		nullableText(notes),
	)
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

func (s *PostgresStore) UpdateWalletSyncProgress(ctx context.Context, walletSyncRunID int64, progress WalletSyncProgress) error {
	const q = `
UPDATE wallet_sync_runs
SET baseline_complete = $2,
    incomplete_window = $3,
    unknown_gate_reason = $4,
    truncation_reason = $5,
    transactions_fetched = $6,
    transactions_inserted = $7,
    transactions_linked = $8,
    transactions_failed_to_normalize = $9,
    counterparties_created = $10,
    counterparties_updated = $11,
    poisoning_candidates_inserted = $12
WHERE id = $1`

	reason := nullableText(progress.UnknownGateReason)
	if !progress.IncompleteWindow {
		reason = nil
	}
	truncation := nullableText(progress.TruncationReason)
	if progress.TruncationReason == "" {
		truncation = nil
	}

	res, err := s.DB.ExecContext(
		ctx,
		q,
		walletSyncRunID,
		progress.BaselineComplete,
		progress.IncompleteWindow,
		reason,
		truncation,
		progress.TransactionsFetched,
		progress.TransactionsInserted,
		progress.TransactionsLinked,
		progress.TransactionsFailedNormalize,
		progress.CounterpartiesCreated,
		progress.CounterpartiesUpdated,
		progress.PoisoningCandidatesInserted,
	)
	if err != nil {
		return fmt.Errorf("update wallet sync progress %d: %w", walletSyncRunID, err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update wallet sync progress %d rows affected: %w", walletSyncRunID, err)
	}
	if rows == 0 {
		return fmt.Errorf("update wallet sync progress %d: not found", walletSyncRunID)
	}
	return nil
}

func (s *PostgresStore) FinalizeWalletSyncRun(ctx context.Context, walletSyncRunID int64, status runs.WalletStatus, completedAt time.Time, incompleteWindow bool, unknownGateReason, errorCode, errorMessage, notes string) error {
	const q = `
UPDATE wallet_sync_runs
SET status = $2,
    completed_at = $3,
    incomplete_window = $4,
    unknown_gate_reason = $5,
    error_code = $6,
    error_message = $7,
    notes = $8
WHERE id = $1`
	reason := nullableText(unknownGateReason)
	if !incompleteWindow {
		reason = nil
	}
	res, err := s.DB.ExecContext(
		ctx,
		q,
		walletSyncRunID,
		status,
		completedAt.UTC(),
		incompleteWindow,
		reason,
		nullableText(errorCode),
		nullableText(errorMessage),
		nullableText(notes),
	)
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

func (s *PostgresStore) EnsureWallet(ctx context.Context, walletAddress string) (int64, error) {
	const q = `
INSERT INTO wallets (address)
VALUES ($1)
ON CONFLICT (address) DO UPDATE SET address = EXCLUDED.address
RETURNING id`
	var walletID int64
	if err := s.DB.QueryRowContext(ctx, q, walletAddress).Scan(&walletID); err != nil {
		return 0, fmt.Errorf("ensure wallet %s: %w", walletAddress, err)
	}
	return walletID, nil
}

func (s *PostgresStore) UpdateWalletLastSyncedAt(ctx context.Context, walletID int64, syncedAt time.Time) error {
	const q = `
UPDATE wallets
SET last_synced_at = GREATEST(COALESCE(last_synced_at, to_timestamp(0)), $2)
WHERE id = $1`
	res, err := s.DB.ExecContext(ctx, q, walletID, syncedAt.UTC())
	if err != nil {
		return fmt.Errorf("update wallet %d last_synced_at: %w", walletID, err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update wallet %d last_synced_at rows affected: %w", walletID, err)
	}
	if rows == 0 {
		return fmt.Errorf("update wallet %d last_synced_at: not found", walletID)
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

func (s *PostgresStore) LinkWalletTransfer(ctx context.Context, walletID int64, relationType counterparties.RelationType, transfer transactions.NormalizedTransfer) (linked bool, err error) {
	const q = `
	WITH matched_tx AS (
	  SELECT id
	  FROM transactions
	  WHERE signature = $3
	    AND transfer_fingerprint = $4
	),
	inserted_link AS (
	  INSERT INTO wallet_transactions (wallet_id, transaction_id, relation_type)
	  SELECT $1, matched_tx.id, $2
	  FROM matched_tx
	  ON CONFLICT (wallet_id, transaction_id, relation_type) DO NOTHING
	  RETURNING id
	)
	SELECT EXISTS(SELECT 1 FROM matched_tx), EXISTS(SELECT 1 FROM inserted_link)`
	var txExists bool
	var inserted bool
	if err = s.DB.QueryRowContext(ctx, q, walletID, relationType, transfer.Signature, transfer.TransferFingerprint).Scan(&txExists, &inserted); err != nil {
		return false, fmt.Errorf("link wallet transfer wallet=%d signature=%s fingerprint=%s: %w", walletID, transfer.Signature, transfer.TransferFingerprint, err)
	}
	if !txExists {
		return false, fmt.Errorf("link wallet transfer wallet=%d signature=%s fingerprint=%s: backing transaction not found", walletID, transfer.Signature, transfer.TransferFingerprint)
	}
	return inserted, nil
}

func (s *PostgresStore) UpsertCounterpartyEvent(ctx context.Context, event counterparties.Event) (created bool, updated bool, err error) {
	if event.FocalWalletID == 0 {
		return false, false, fmt.Errorf("counterparty event focal wallet id is required")
	}
	if event.CounterpartyAddress == "" {
		return false, false, fmt.Errorf("counterparty event counterparty address is required")
	}

	var firstInboundAt any
	var lastInboundAt any
	var firstOutboundAt any
	var lastOutboundAt any
	inboundCount := 0
	outboundCount := 0

	switch event.RelationType {
	case counterparties.RelationReceiver:
		firstInboundAt = event.OccurredAt.UTC()
		lastInboundAt = event.OccurredAt.UTC()
		inboundCount = 1
	case counterparties.RelationSender:
		firstOutboundAt = event.OccurredAt.UTC()
		lastOutboundAt = event.OccurredAt.UTC()
		outboundCount = 1
	default:
		return false, false, fmt.Errorf("unsupported relation type %q", event.RelationType)
	}

	const q = `
INSERT INTO counterparties (
  focal_wallet_id,
  counterparty_address,
  first_seen_at,
  last_seen_at,
  interaction_count,
  first_inbound_at,
  last_inbound_at,
  inbound_count,
  first_outbound_at,
  last_outbound_at,
  outbound_count
)
VALUES ($1, $2, $3, $4, 1, $5, $6, $7, $8, $9, $10)
ON CONFLICT (focal_wallet_id, counterparty_address) DO UPDATE SET
  first_seen_at = LEAST(counterparties.first_seen_at, EXCLUDED.first_seen_at),
  last_seen_at = GREATEST(counterparties.last_seen_at, EXCLUDED.last_seen_at),
  interaction_count = counterparties.interaction_count + EXCLUDED.interaction_count,
  first_inbound_at = CASE
    WHEN EXCLUDED.first_inbound_at IS NULL THEN counterparties.first_inbound_at
    WHEN counterparties.first_inbound_at IS NULL THEN EXCLUDED.first_inbound_at
    ELSE LEAST(counterparties.first_inbound_at, EXCLUDED.first_inbound_at)
  END,
  last_inbound_at = CASE
    WHEN EXCLUDED.last_inbound_at IS NULL THEN counterparties.last_inbound_at
    WHEN counterparties.last_inbound_at IS NULL THEN EXCLUDED.last_inbound_at
    ELSE GREATEST(counterparties.last_inbound_at, EXCLUDED.last_inbound_at)
  END,
  inbound_count = counterparties.inbound_count + EXCLUDED.inbound_count,
  first_outbound_at = CASE
    WHEN EXCLUDED.first_outbound_at IS NULL THEN counterparties.first_outbound_at
    WHEN counterparties.first_outbound_at IS NULL THEN EXCLUDED.first_outbound_at
    ELSE LEAST(counterparties.first_outbound_at, EXCLUDED.first_outbound_at)
  END,
  last_outbound_at = CASE
    WHEN EXCLUDED.last_outbound_at IS NULL THEN counterparties.last_outbound_at
    WHEN counterparties.last_outbound_at IS NULL THEN EXCLUDED.last_outbound_at
    ELSE GREATEST(counterparties.last_outbound_at, EXCLUDED.last_outbound_at)
  END,
  outbound_count = counterparties.outbound_count + EXCLUDED.outbound_count,
  updated_at = NOW()
RETURNING (xmax = 0)`

	var wasInserted bool
	if err = s.DB.QueryRowContext(
		ctx,
		q,
		event.FocalWalletID,
		event.CounterpartyAddress,
		event.OccurredAt.UTC(),
		event.OccurredAt.UTC(),
		firstInboundAt,
		lastInboundAt,
		inboundCount,
		firstOutboundAt,
		lastOutboundAt,
		outboundCount,
	).Scan(&wasInserted); err != nil {
		return false, false, fmt.Errorf("upsert counterparty event wallet=%d counterparty=%s: %w", event.FocalWalletID, event.CounterpartyAddress, err)
	}
	if wasInserted {
		return true, false, nil
	}
	return false, true, nil
}

func (s *PostgresStore) InsertPoisoningCandidates(ctx context.Context, walletSyncRunID int64, focalWalletID int64, candidates []CandidateRecord) (inserted int, err error) {
	if len(candidates) == 0 {
		return 0, nil
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin poisoning candidate insert: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO poisoning_candidates (
  wallet_sync_run_id,
  focal_wallet_id,
  signature,
  transfer_index,
  suspicious_counterparty,
  matched_legit_counterparty,
  token_mint,
  amount_raw,
  block_time,
  is_zero_value,
  is_dust,
  is_new_counterparty,
  is_inbound,
  legit_last_seen_at,
  recency_days,
  repeat_injection_count,
  incomplete_window,
  unknown_gate_reason,
  match_rule_version
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
ON CONFLICT (wallet_sync_run_id, signature, transfer_index) DO NOTHING
RETURNING id`)
	if err != nil {
		return 0, fmt.Errorf("prepare poisoning candidate insert: %w", err)
	}
	defer stmt.Close()

	for _, c := range candidates {
		var id int64
		reason := nullableText(c.UnknownGateReason)
		if !c.IncompleteWindow {
			reason = nil
		}
		err = stmt.QueryRowContext(
			ctx,
			walletSyncRunID,
			focalWalletID,
			c.Signature,
			c.TransferIndex,
			c.SuspiciousCounterparty,
			c.MatchedLegitCounterparty,
			nullableText(c.TokenMint),
			c.AmountRaw,
			c.BlockTime.UTC(),
			c.IsZeroValue,
			c.IsDust,
			c.IsNewCounterparty,
			c.IsInbound,
			c.LegitLastSeenAt.UTC(),
			c.RecencyDays,
			c.RepeatInjectionCount,
			c.IncompleteWindow,
			reason,
			c.MatchRuleVersion,
		).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return 0, fmt.Errorf("insert poisoning candidate signature=%s transfer_index=%d: %w", c.Signature, c.TransferIndex, err)
		}
		inserted++
	}

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit poisoning candidate inserts: %w", err)
	}
	return inserted, nil
}

func (s *PostgresStore) ListDustThresholds(ctx context.Context, startInclusive, endExclusive time.Time) ([]DustThresholdRecord, error) {
	const q = `
SELECT asset_key,
       dust_amount_raw_threshold::TEXT,
       active_from,
       active_to
FROM asset_thresholds
WHERE active_from < $2
  AND (active_to IS NULL OR active_to > $1)
ORDER BY asset_key, active_from DESC`

	rows, err := s.DB.QueryContext(ctx, q, startInclusive.UTC(), endExclusive.UTC())
	if err != nil {
		return nil, fmt.Errorf("list dust thresholds: %w", err)
	}
	defer rows.Close()

	out := make([]DustThresholdRecord, 0)
	for rows.Next() {
		var rec DustThresholdRecord
		if err := rows.Scan(&rec.AssetKey, &rec.AmountRaw, &rec.ActiveFrom, &rec.ActiveTo); err != nil {
			return nil, fmt.Errorf("scan dust threshold: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dust thresholds: %w", err)
	}
	return out, nil
}

func (s *PostgresStore) AcquireWalletLock(ctx context.Context, walletAddress string, ttlSeconds int) (acquired bool, holderToken string, err error) {
	if walletAddress == "" {
		return false, "", fmt.Errorf("wallet address is required")
	}
	if ttlSeconds < 1 {
		return false, "", fmt.Errorf("ttlSeconds must be >= 1")
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
		return false, "", fmt.Errorf("acquire wallet lock %s: %w", walletAddress, err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, "", fmt.Errorf("acquire wallet lock %s rows affected: %w", walletAddress, err)
	}
	if rows == 0 {
		return false, "", nil
	}
	return true, holder, nil
}

func (s *PostgresStore) ReleaseWalletLock(ctx context.Context, walletAddress string, holderToken string) error {
	if walletAddress == "" {
		return fmt.Errorf("wallet address is required")
	}
	if holderToken == "" {
		return fmt.Errorf("holder token is required")
	}
	const q = `DELETE FROM wallet_locks WHERE wallet_address = $1 AND holder_token = $2`
	if _, err := s.DB.ExecContext(ctx, q, walletAddress, holderToken); err != nil {
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
