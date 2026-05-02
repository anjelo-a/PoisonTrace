package storage

import (
	"context"
	"fmt"
	"time"
)

type OverviewCandidateRecord struct {
	WalletSyncRunID        int64
	FocalWallet            string
	Signature              string
	TransferIndex          int
	BlockTime              time.Time
	SuspiciousCounterparty string
	RepeatInjectionCount   int
	RecencyDays            int
}

type OverviewMetricsRecord struct {
	CandidatesEmitted   int
	UnknownGateBlocks   int
	TransactionsScanned int
	PassedTransactions  int
	LastScanUpdateAt    *time.Time
}

type CandidateListRecord struct {
	WalletSyncRunID          int64
	FocalWallet              string
	Signature                string
	TransferIndex            int
	BlockTime                time.Time
	SuspiciousCounterparty   string
	MatchedLegitCounterparty string
	RepeatInjectionCount     int
	RecencyDays              int
	UnknownGateReason        string
}

type IngestionRunListRecord struct {
	ID                   int64
	Status               string
	StartedAt            time.Time
	CompletedAt          *time.Time
	WalletsProcessed     int
	WalletsFailed        int
	WalletsSkipped       int
	IncompleteWindows    int
	TruncationWalletRate string
	UnknownGatePresent   bool
}

type WalletSyncListRecord struct {
	WalletSyncRunID     int64
	IngestionRunID      int64
	FocalWallet         string
	Status              string
	BaselineStartAt     time.Time
	BaselineEndAt       time.Time
	ScanStartAt         time.Time
	ScanEndAt           time.Time
	BaselineComplete    bool
	IncompleteWindow    bool
	UnknownGateReason   string
	TransactionsFetched int
	TruncationReason    string
	UpdatedAt           time.Time
}

type TransactionListRecord struct {
	FocalWallet         string
	Signature           string
	TransferIndex       int
	BlockTime           time.Time
	NormalizationStatus string
	PoisoningEligible   bool
	RelationType        string
	DustStatus          string
	AmountRaw           string
	AssetType           string
}

type CounterpartyListRecord struct {
	FocalWallet         string
	CounterpartyAddress string
	FirstSeenAt         time.Time
	LastSeenAt          time.Time
	InboundCount        int64
	OutboundCount       int64
	LastOutboundAt      *time.Time
	CandidateLinks      int
}

func (s *PostgresStore) GetOverviewMetrics(ctx context.Context, since time.Time) (OverviewMetricsRecord, error) {
	const q = `
SELECT
  (SELECT COUNT(*)
   FROM poisoning_candidates pc
   WHERE pc.block_time >= $1) AS candidates_emitted,
  (SELECT COUNT(*)
   FROM wallet_sync_runs wsr
   WHERE wsr.incomplete_window = TRUE
     AND COALESCE(NULLIF(BTRIM(wsr.unknown_gate_reason), ''), '') <> ''
     AND wsr.scan_end_at >= $1) AS unknown_gate_blocks,
  (SELECT COALESCE(SUM(wsr.transactions_fetched), 0)
   FROM wallet_sync_runs wsr
   WHERE wsr.scan_end_at >= $1) AS transactions_scanned,
  (SELECT COALESCE(COUNT(*), 0)
   FROM transactions t
   JOIN wallet_transactions wt ON wt.transaction_id = t.id
   JOIN wallet_sync_runs wsr ON wsr.wallet_id = wt.wallet_id
   WHERE wsr.scan_end_at >= $1
     AND t.normalization_status = 'resolved'
     AND t.poisoning_eligible = TRUE) AS passed_transactions,
  (SELECT MAX(COALESCE(wsr.completed_at, wsr.started_at))
   FROM wallet_sync_runs wsr) AS last_scan_update_at`

	var out OverviewMetricsRecord
	if err := s.DB.QueryRowContext(ctx, q, since.UTC()).Scan(
		&out.CandidatesEmitted,
		&out.UnknownGateBlocks,
		&out.TransactionsScanned,
		&out.PassedTransactions,
		&out.LastScanUpdateAt,
	); err != nil {
		return OverviewMetricsRecord{}, fmt.Errorf("get overview metrics: %w", err)
	}
	return out, nil
}

func (s *PostgresStore) ListOverviewCandidates(ctx context.Context, limit int) ([]OverviewCandidateRecord, error) {
	const q = `
SELECT pc.wallet_sync_run_id,
       w.address,
       pc.signature,
       pc.transfer_index,
       pc.block_time,
       pc.suspicious_counterparty,
       pc.repeat_injection_count,
       pc.recency_days
FROM poisoning_candidates pc
JOIN wallets w ON w.id = pc.focal_wallet_id
ORDER BY pc.block_time DESC, pc.signature DESC, pc.transfer_index DESC
LIMIT $1`

	rows, err := s.DB.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("list overview candidates: %w", err)
	}
	defer rows.Close()

	out := make([]OverviewCandidateRecord, 0)
	for rows.Next() {
		var rec OverviewCandidateRecord
		if err := rows.Scan(
			&rec.WalletSyncRunID,
			&rec.FocalWallet,
			&rec.Signature,
			&rec.TransferIndex,
			&rec.BlockTime,
			&rec.SuspiciousCounterparty,
			&rec.RepeatInjectionCount,
			&rec.RecencyDays,
		); err != nil {
			return nil, fmt.Errorf("scan overview candidate row: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate overview candidate rows: %w", err)
	}
	return out, nil
}

func (s *PostgresStore) ListCandidates(ctx context.Context, limit int, offset int) ([]CandidateListRecord, int, error) {
	const countQ = `SELECT COUNT(*) FROM poisoning_candidates`
	var total int
	if err := s.DB.QueryRowContext(ctx, countQ).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count candidates: %w", err)
	}

	const q = `
SELECT pc.wallet_sync_run_id,
       w.address,
       pc.signature,
       pc.transfer_index,
       pc.block_time,
       pc.suspicious_counterparty,
       pc.matched_legit_counterparty,
       pc.repeat_injection_count,
       pc.recency_days,
       COALESCE(pc.unknown_gate_reason, '')
FROM poisoning_candidates pc
JOIN wallets w ON w.id = pc.focal_wallet_id
ORDER BY pc.block_time DESC, pc.signature DESC, pc.transfer_index DESC
LIMIT $1 OFFSET $2`

	rows, err := s.DB.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list candidates: %w", err)
	}
	defer rows.Close()

	out := make([]CandidateListRecord, 0)
	for rows.Next() {
		var rec CandidateListRecord
		if err := rows.Scan(
			&rec.WalletSyncRunID,
			&rec.FocalWallet,
			&rec.Signature,
			&rec.TransferIndex,
			&rec.BlockTime,
			&rec.SuspiciousCounterparty,
			&rec.MatchedLegitCounterparty,
			&rec.RepeatInjectionCount,
			&rec.RecencyDays,
			&rec.UnknownGateReason,
		); err != nil {
			return nil, 0, fmt.Errorf("scan candidate row: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate candidate rows: %w", err)
	}
	return out, total, nil
}

func (s *PostgresStore) ListTransactions(ctx context.Context, limit int, offset int) ([]TransactionListRecord, int, error) {
	const countQ = `SELECT COUNT(*) FROM wallet_transactions`
	var total int
	if err := s.DB.QueryRowContext(ctx, countQ).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count transactions: %w", err)
	}

	const q = `
SELECT w.address,
       t.signature,
       t.transfer_index,
       t.block_time,
       t.normalization_status,
       t.poisoning_eligible,
       wt.relation_type,
       t.dust_status,
       t.amount_raw::TEXT,
       t.asset_type
FROM wallet_transactions wt
JOIN wallets w ON w.id = wt.wallet_id
JOIN transactions t ON t.id = wt.transaction_id
ORDER BY t.block_time DESC, t.signature DESC, t.transfer_index DESC
LIMIT $1 OFFSET $2`

	rows, err := s.DB.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	out := make([]TransactionListRecord, 0)
	for rows.Next() {
		var rec TransactionListRecord
		if err := rows.Scan(
			&rec.FocalWallet,
			&rec.Signature,
			&rec.TransferIndex,
			&rec.BlockTime,
			&rec.NormalizationStatus,
			&rec.PoisoningEligible,
			&rec.RelationType,
			&rec.DustStatus,
			&rec.AmountRaw,
			&rec.AssetType,
		); err != nil {
			return nil, 0, fmt.Errorf("scan transaction row: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate transaction rows: %w", err)
	}
	return out, total, nil
}

func (s *PostgresStore) ListRuns(ctx context.Context, limit int, offset int) ([]IngestionRunListRecord, int, error) {
	const countQ = `SELECT COUNT(*) FROM ingestion_runs`
	var total int
	if err := s.DB.QueryRowContext(ctx, countQ).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count runs: %w", err)
	}

	const q = `
SELECT ir.id,
       ir.status,
       ir.started_at,
       ir.completed_at,
       ir.wallets_processed,
       ir.wallets_failed,
       ir.wallets_skipped,
       (
         SELECT COUNT(*)
         FROM wallet_sync_runs wsr
         WHERE wsr.ingestion_run_id = ir.id
           AND wsr.incomplete_window = TRUE
       ) AS incomplete_windows,
       ir.truncation_wallet_rate::TEXT,
       EXISTS (
         SELECT 1
         FROM wallet_sync_runs wsr
         WHERE wsr.ingestion_run_id = ir.id
           AND COALESCE(NULLIF(BTRIM(wsr.unknown_gate_reason), ''), '') <> ''
       ) AS unknown_gate_present
FROM ingestion_runs ir
ORDER BY ir.started_at DESC, ir.id DESC
LIMIT $1 OFFSET $2`

	rows, err := s.DB.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list runs: %w", err)
	}
	defer rows.Close()

	out := make([]IngestionRunListRecord, 0)
	for rows.Next() {
		var rec IngestionRunListRecord
		if err := rows.Scan(
			&rec.ID,
			&rec.Status,
			&rec.StartedAt,
			&rec.CompletedAt,
			&rec.WalletsProcessed,
			&rec.WalletsFailed,
			&rec.WalletsSkipped,
			&rec.IncompleteWindows,
			&rec.TruncationWalletRate,
			&rec.UnknownGatePresent,
		); err != nil {
			return nil, 0, fmt.Errorf("scan run row: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate run rows: %w", err)
	}
	return out, total, nil
}

func (s *PostgresStore) ListWalletSyncRuns(ctx context.Context, limit int, offset int) ([]WalletSyncListRecord, int, error) {
	const countQ = `SELECT COUNT(*) FROM wallet_sync_runs`
	var total int
	if err := s.DB.QueryRowContext(ctx, countQ).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count wallet sync runs: %w", err)
	}

	const q = `
SELECT wsr.id,
       wsr.ingestion_run_id,
       w.address,
       wsr.status,
       wsr.baseline_start_at,
       wsr.baseline_end_at,
       wsr.scan_start_at,
       wsr.scan_end_at,
       wsr.baseline_complete,
       wsr.incomplete_window,
       COALESCE(wsr.unknown_gate_reason, ''),
       wsr.transactions_fetched,
       COALESCE(wsr.truncation_reason, ''),
       COALESCE(wsr.completed_at, wsr.started_at)
FROM wallet_sync_runs wsr
JOIN wallets w ON w.id = wsr.wallet_id
ORDER BY wsr.scan_end_at DESC, wsr.id DESC
LIMIT $1 OFFSET $2`

	rows, err := s.DB.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list wallet sync runs: %w", err)
	}
	defer rows.Close()

	out := make([]WalletSyncListRecord, 0)
	for rows.Next() {
		var rec WalletSyncListRecord
		if err := rows.Scan(
			&rec.WalletSyncRunID,
			&rec.IngestionRunID,
			&rec.FocalWallet,
			&rec.Status,
			&rec.BaselineStartAt,
			&rec.BaselineEndAt,
			&rec.ScanStartAt,
			&rec.ScanEndAt,
			&rec.BaselineComplete,
			&rec.IncompleteWindow,
			&rec.UnknownGateReason,
			&rec.TransactionsFetched,
			&rec.TruncationReason,
			&rec.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan wallet sync row: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate wallet sync rows: %w", err)
	}
	return out, total, nil
}

func (s *PostgresStore) ListCounterparties(ctx context.Context, limit int, offset int) ([]CounterpartyListRecord, int, error) {
	const countQ = `SELECT COUNT(*) FROM counterparties`
	var total int
	if err := s.DB.QueryRowContext(ctx, countQ).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count counterparties: %w", err)
	}

	const q = `
SELECT w.address,
       cp.counterparty_address,
       cp.first_seen_at,
       cp.last_seen_at,
       cp.inbound_count,
       cp.outbound_count,
       cp.last_outbound_at,
       (
         SELECT COUNT(*)
         FROM poisoning_candidates pc
         WHERE pc.focal_wallet_id = cp.focal_wallet_id
           AND pc.suspicious_counterparty = cp.counterparty_address
       ) AS candidate_links
FROM counterparties cp
JOIN wallets w ON w.id = cp.focal_wallet_id
ORDER BY cp.last_seen_at DESC, cp.id DESC
LIMIT $1 OFFSET $2`

	rows, err := s.DB.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list counterparties: %w", err)
	}
	defer rows.Close()

	out := make([]CounterpartyListRecord, 0)
	for rows.Next() {
		var rec CounterpartyListRecord
		if err := rows.Scan(
			&rec.FocalWallet,
			&rec.CounterpartyAddress,
			&rec.FirstSeenAt,
			&rec.LastSeenAt,
			&rec.InboundCount,
			&rec.OutboundCount,
			&rec.LastOutboundAt,
			&rec.CandidateLinks,
		); err != nil {
			return nil, 0, fmt.Errorf("scan counterparty row: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate counterparties rows: %w", err)
	}
	return out, total, nil
}
