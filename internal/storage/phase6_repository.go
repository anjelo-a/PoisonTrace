package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type ConfigOverrideRecord struct {
	MaxWalletsPerRun         *int
	MaxTXPagesPerWallet      *int
	MaxTXPerWallet           *int
	MaxConcurrentWallets     *int
	WalletSyncTimeoutSeconds *int
	RunTimeoutSeconds        *int
	MaxHeliusRetries         *int
	HeliusRequestDelayMS     *int
	BaselineLookbackDays     *int
	ScanWindowDays           *int
	LookalikeRecencyDays     *int
	LookalikePrefixMin       *int
	LookalikeSuffixMin       *int
	LookalikeSingleSideMin   *int
	MinInjectionCount        *int
	UpdatedAt                *time.Time
}

type ExportJobRecord struct {
	ID           int64
	RunID        int64
	Status       string
	OutDir       string
	ErrorMessage string
	CreatedAt    time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time
}

type OpsRunHealthRecord struct {
	RunID                int64
	Status               string
	StartedAt            time.Time
	CompletedAt          *time.Time
	WalletsRequested     int
	WalletsProcessed     int
	WalletsFailed        int
	WalletsSkipped       int
	TruncationWalletRate string
	RetryExhaustedCount  int
}

type FailureClassCountRecord struct {
	FailureClass string
	Count        int
}

func (s *PostgresStore) GetConfigOverride(ctx context.Context) (ConfigOverrideRecord, bool, error) {
	const q = `
SELECT max_wallets_per_run,
       max_tx_pages_per_wallet,
       max_tx_per_wallet,
       max_concurrent_wallets,
       wallet_sync_timeout_seconds,
       run_timeout_seconds,
       max_helius_retries,
       helius_request_delay_ms,
       baseline_lookback_days,
       scan_window_days,
       lookalike_recency_days,
       lookalike_prefix_min,
       lookalike_suffix_min,
       lookalike_single_side_min,
       min_injection_count,
       updated_at
FROM app_config_overrides
WHERE id = 1`
	var rec ConfigOverrideRecord
	err := s.DB.QueryRowContext(ctx, q).Scan(
		&rec.MaxWalletsPerRun,
		&rec.MaxTXPagesPerWallet,
		&rec.MaxTXPerWallet,
		&rec.MaxConcurrentWallets,
		&rec.WalletSyncTimeoutSeconds,
		&rec.RunTimeoutSeconds,
		&rec.MaxHeliusRetries,
		&rec.HeliusRequestDelayMS,
		&rec.BaselineLookbackDays,
		&rec.ScanWindowDays,
		&rec.LookalikeRecencyDays,
		&rec.LookalikePrefixMin,
		&rec.LookalikeSuffixMin,
		&rec.LookalikeSingleSideMin,
		&rec.MinInjectionCount,
		&rec.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return ConfigOverrideRecord{}, false, nil
		}
		return ConfigOverrideRecord{}, false, fmt.Errorf("get config override: %w", err)
	}
	return rec, true, nil
}

func (s *PostgresStore) UpsertConfigOverride(ctx context.Context, rec ConfigOverrideRecord) error {
	const q = `
INSERT INTO app_config_overrides (
  id,
  max_wallets_per_run,
  max_tx_pages_per_wallet,
  max_tx_per_wallet,
  max_concurrent_wallets,
  wallet_sync_timeout_seconds,
  run_timeout_seconds,
  max_helius_retries,
  helius_request_delay_ms,
  baseline_lookback_days,
  scan_window_days,
  lookalike_recency_days,
  lookalike_prefix_min,
  lookalike_suffix_min,
  lookalike_single_side_min,
  min_injection_count,
  updated_at
)
VALUES (1, $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15, NOW())
ON CONFLICT (id) DO UPDATE SET
  max_wallets_per_run = EXCLUDED.max_wallets_per_run,
  max_tx_pages_per_wallet = EXCLUDED.max_tx_pages_per_wallet,
  max_tx_per_wallet = EXCLUDED.max_tx_per_wallet,
  max_concurrent_wallets = EXCLUDED.max_concurrent_wallets,
  wallet_sync_timeout_seconds = EXCLUDED.wallet_sync_timeout_seconds,
  run_timeout_seconds = EXCLUDED.run_timeout_seconds,
  max_helius_retries = EXCLUDED.max_helius_retries,
  helius_request_delay_ms = EXCLUDED.helius_request_delay_ms,
  baseline_lookback_days = EXCLUDED.baseline_lookback_days,
  scan_window_days = EXCLUDED.scan_window_days,
  lookalike_recency_days = EXCLUDED.lookalike_recency_days,
  lookalike_prefix_min = EXCLUDED.lookalike_prefix_min,
  lookalike_suffix_min = EXCLUDED.lookalike_suffix_min,
  lookalike_single_side_min = EXCLUDED.lookalike_single_side_min,
  min_injection_count = EXCLUDED.min_injection_count,
  updated_at = NOW()`
	if _, err := s.DB.ExecContext(ctx, q,
		rec.MaxWalletsPerRun,
		rec.MaxTXPagesPerWallet,
		rec.MaxTXPerWallet,
		rec.MaxConcurrentWallets,
		rec.WalletSyncTimeoutSeconds,
		rec.RunTimeoutSeconds,
		rec.MaxHeliusRetries,
		rec.HeliusRequestDelayMS,
		rec.BaselineLookbackDays,
		rec.ScanWindowDays,
		rec.LookalikeRecencyDays,
		rec.LookalikePrefixMin,
		rec.LookalikeSuffixMin,
		rec.LookalikeSingleSideMin,
		rec.MinInjectionCount,
	); err != nil {
		return fmt.Errorf("upsert config override: %w", err)
	}
	return nil
}

func (s *PostgresStore) CreateExportJob(ctx context.Context, runID int64, outDir string) (int64, error) {
	const q = `
INSERT INTO export_jobs (run_id, status, out_dir)
VALUES ($1, 'queued', $2)
RETURNING id`
	var id int64
	if err := s.DB.QueryRowContext(ctx, q, runID, outDir).Scan(&id); err != nil {
		return 0, fmt.Errorf("create export job: %w", err)
	}
	return id, nil
}

func (s *PostgresStore) UpdateExportJobStatus(ctx context.Context, jobID int64, status string, errMsg *string, startedAt, completedAt *time.Time) error {
	const q = `
UPDATE export_jobs
SET status = $2,
    error_message = $3,
    started_at = COALESCE($4, started_at),
    completed_at = $5
WHERE id = $1`
	if _, err := s.DB.ExecContext(ctx, q, jobID, status, nullableText(deref(errMsg)), startedAt, completedAt); err != nil {
		return fmt.Errorf("update export job status: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetExportJob(ctx context.Context, jobID int64) (ExportJobRecord, bool, error) {
	const q = `
SELECT id, run_id, status, out_dir, COALESCE(error_message, ''), created_at, started_at, completed_at
FROM export_jobs
WHERE id = $1`
	var rec ExportJobRecord
	if err := s.DB.QueryRowContext(ctx, q, jobID).Scan(
		&rec.ID, &rec.RunID, &rec.Status, &rec.OutDir, &rec.ErrorMessage, &rec.CreatedAt, &rec.StartedAt, &rec.CompletedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return ExportJobRecord{}, false, nil
		}
		return ExportJobRecord{}, false, fmt.Errorf("get export job: %w", err)
	}
	return rec, true, nil
}

func (s *PostgresStore) ListExportJobsForRun(ctx context.Context, runID int64, limit int) ([]ExportJobRecord, error) {
	const q = `
SELECT id, run_id, status, out_dir, COALESCE(error_message, ''), created_at, started_at, completed_at
FROM export_jobs
WHERE run_id = $1
ORDER BY id DESC
LIMIT $2`
	rows, err := s.DB.QueryContext(ctx, q, runID, limit)
	if err != nil {
		return nil, fmt.Errorf("list export jobs: %w", err)
	}
	defer rows.Close()
	out := make([]ExportJobRecord, 0)
	for rows.Next() {
		var rec ExportJobRecord
		if err := rows.Scan(&rec.ID, &rec.RunID, &rec.Status, &rec.OutDir, &rec.ErrorMessage, &rec.CreatedAt, &rec.StartedAt, &rec.CompletedAt); err != nil {
			return nil, fmt.Errorf("scan export job: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate export jobs: %w", err)
	}
	return out, nil
}

func (s *PostgresStore) ListOpsRunHealth(ctx context.Context, limit int, offset int) ([]OpsRunHealthRecord, int, error) {
	var total int
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM ingestion_runs`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count ops runs: %w", err)
	}
	const q = `
SELECT id, status, started_at, completed_at, wallets_requested, wallets_processed, wallets_failed, wallets_skipped, truncation_wallet_rate::TEXT, retry_exhausted_count
FROM ingestion_runs
ORDER BY id DESC
LIMIT $1 OFFSET $2`
	rows, err := s.DB.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list ops runs: %w", err)
	}
	defer rows.Close()
	out := make([]OpsRunHealthRecord, 0)
	for rows.Next() {
		var rec OpsRunHealthRecord
		if err := rows.Scan(&rec.RunID, &rec.Status, &rec.StartedAt, &rec.CompletedAt, &rec.WalletsRequested, &rec.WalletsProcessed, &rec.WalletsFailed, &rec.WalletsSkipped, &rec.TruncationWalletRate, &rec.RetryExhaustedCount); err != nil {
			return nil, 0, fmt.Errorf("scan ops run row: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate ops runs: %w", err)
	}
	return out, total, nil
}

func (s *PostgresStore) ListFailureClassCounts(ctx context.Context) ([]FailureClassCountRecord, error) {
	const q = `
WITH classes AS (
  SELECT CASE
           WHEN status IN ('timed_out') THEN 'timeout'
           WHEN status IN ('cancelled') THEN 'canceled'
           WHEN status IN ('failed') THEN 'failed'
           WHEN status IN ('partially_succeeded') THEN 'partial'
           ELSE 'other'
         END AS failure_class
  FROM ingestion_runs
  UNION ALL
  SELECT CASE
           WHEN status IN ('timed_out') THEN 'timeout'
           WHEN COALESCE(NULLIF(BTRIM(error_code), ''), '') <> '' THEN 'persistence_error'
           WHEN COALESCE(NULLIF(BTRIM(unknown_gate_reason), ''), '') LIKE '%retry_exhausted%' THEN 'retry_exhausted'
           WHEN status IN ('partial') THEN 'partial'
           WHEN status IN ('failed') THEN 'failed'
           ELSE 'other'
         END AS failure_class
  FROM wallet_sync_runs
)
SELECT failure_class, COUNT(*)
FROM classes
GROUP BY failure_class
ORDER BY failure_class`
	rows, err := s.DB.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list failure class counts: %w", err)
	}
	defer rows.Close()
	out := make([]FailureClassCountRecord, 0)
	for rows.Next() {
		var rec FailureClassCountRecord
		if err := rows.Scan(&rec.FailureClass, &rec.Count); err != nil {
			return nil, fmt.Errorf("scan failure class row: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate failure class rows: %w", err)
	}
	return out, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
