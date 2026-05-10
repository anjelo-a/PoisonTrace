package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type OverviewMetricsRecord struct {
	CandidatesEmitted   int
	UnknownGateBlocks   int
	TransactionsScanned int
	PassedTransactions  int
	LastScanUpdateAt    *time.Time
}

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
	IncompleteWindow         bool
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
	WalletSyncRunID       int64
	IngestionRunID        int64
	FocalWallet           string
	Status                string
	BaselineStartAt       time.Time
	BaselineEndAt         time.Time
	ScanStartAt           time.Time
	ScanEndAt             time.Time
	BaselineComplete      bool
	IncompleteWindow      bool
	UnknownGateReason     string
	TransactionsFetched   int
	UnsupportedAssetCount int
	UnknownGateBlockCount int
	CandidateBlockCount   int
	TruncationReason      string
	UpdatedAt             time.Time
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

type CandidateExplanationRecord struct {
	WalletSyncRunID          int64
	IngestionRunID           int64
	FocalWallet              string
	Signature                string
	TransferIndex            int
	BlockTime                time.Time
	SuspiciousCounterparty   string
	MatchedLegitCounterparty string
	ScanStartAt              time.Time
	ScanEndAt                time.Time
	BaselineStartAt          time.Time
	BaselineEndAt            time.Time
	BaselineComplete         bool
	IncompleteWindow         bool
	UnknownGateReason        string
	RelationType             string
	AssetType                string
	NormalizationStatus      string
	PoisoningEligible        bool
	SourceOwner              string
	DestinationOwner         string
	FromTokenAccount         string
	ToTokenAccount           string
	TokenMint                string
	AmountRaw                string
	DustStatus               string
	IsDust                   bool
	IsZeroValue              bool
	IsInbound                bool
	IsNewCounterparty        bool
	RecencyDays              int
	RepeatInjectionCount     int
	LookalikePrefixMatch     int
	LookalikeSuffixMatch     int
	MatchRuleVersion         string
	LegitLastSeenAt          time.Time
	CandidateCreatedAt       time.Time
	WalletTransactionID      int64
	TransactionID            int64
	CounterpartyID           int64
}

type WalletInspectionSummaryRecord struct {
	RunID                   int64
	WalletSyncRunID         int64
	FocalWallet             string
	CandidateCount          int
	UnknownGateBlockCount   int
	IncompleteWindow        bool
	UnknownGateReason       string
	TruncationReason        string
	BaselineComplete        bool
	ScanStartAt             time.Time
	ScanEndAt               time.Time
	BaselineStartAt         time.Time
	BaselineEndAt           time.Time
	TransactionsFetched     int
	PoisoningCandidatesSeen int
}

func (s *PostgresStore) GetOverviewMetrics(ctx context.Context, since time.Time) (OverviewMetricsRecord, error) {
	const q = `
SELECT
  (SELECT COUNT(*)
   FROM poisoning_candidates pc
   WHERE pc.block_time >= $1
     AND pc.incomplete_window = FALSE
     AND COALESCE(NULLIF(BTRIM(pc.unknown_gate_reason), ''), '') = '') AS candidates_emitted,
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
WHERE pc.incomplete_window = FALSE
  AND COALESCE(NULLIF(BTRIM(pc.unknown_gate_reason), ''), '') = ''
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

func (s *PostgresStore) ListCandidates(ctx context.Context, limit, offset int) ([]CandidateListRecord, int, error) {
	const countQ = `
SELECT COUNT(*)
FROM poisoning_candidates pc
WHERE pc.incomplete_window = FALSE
  AND COALESCE(NULLIF(BTRIM(pc.unknown_gate_reason), ''), '') = ''`
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
       COALESCE(pc.unknown_gate_reason, ''),
       pc.incomplete_window
FROM poisoning_candidates pc
JOIN wallets w ON w.id = pc.focal_wallet_id
WHERE pc.incomplete_window = FALSE
  AND COALESCE(NULLIF(BTRIM(pc.unknown_gate_reason), ''), '') = ''
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
			&rec.IncompleteWindow,
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

func (s *PostgresStore) ListTransactions(ctx context.Context, limit, offset int) ([]TransactionListRecord, int, error) {
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

func (s *PostgresStore) ListRuns(ctx context.Context, limit, offset int) ([]IngestionRunListRecord, int, error) {
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

func (s *PostgresStore) ListWalletSyncRuns(ctx context.Context, limit, offset int) ([]WalletSyncListRecord, int, error) {
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
       wsr.unsupported_asset_count,
       wsr.unknown_gate_block_count,
       wsr.candidate_block_count,
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
			&rec.UnsupportedAssetCount,
			&rec.UnknownGateBlockCount,
			&rec.CandidateBlockCount,
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

func (s *PostgresStore) ListCounterparties(ctx context.Context, limit, offset int) ([]CounterpartyListRecord, int, error) {
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
           AND pc.incomplete_window = FALSE
           AND COALESCE(NULLIF(BTRIM(pc.unknown_gate_reason), ''), '') = ''
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

func (s *PostgresStore) GetCandidateExplanation(ctx context.Context, walletSyncRunID int64, signature string, transferIndex int) (CandidateExplanationRecord, bool, error) {
	const q = `
SELECT pc.wallet_sync_run_id,
       wsr.ingestion_run_id,
       w.address,
       pc.signature,
       pc.transfer_index,
       pc.block_time,
       pc.suspicious_counterparty,
       pc.matched_legit_counterparty,
       wsr.scan_start_at,
       wsr.scan_end_at,
       wsr.baseline_start_at,
       wsr.baseline_end_at,
       wsr.baseline_complete,
       pc.incomplete_window,
       COALESCE(NULLIF(BTRIM(pc.unknown_gate_reason), ''), ''),
       wt.relation_type,
       t.asset_type,
       t.normalization_status,
       t.poisoning_eligible,
       COALESCE(t.source_owner_address, ''),
       COALESCE(t.destination_owner_address, ''),
       COALESCE(t.source_token_account, ''),
       COALESCE(t.destination_token_account, ''),
       COALESCE(t.token_mint, ''),
       t.amount_raw::TEXT,
       t.dust_status,
       pc.is_dust,
       pc.is_zero_value,
       pc.is_inbound,
       pc.is_new_counterparty,
       pc.recency_days,
       pc.repeat_injection_count,
       0 AS prefix_match_len,
       0 AS suffix_match_len,
       pc.match_rule_version,
       pc.legit_last_seen_at,
       pc.created_at,
       wt.id,
       t.id,
       cp.id
FROM poisoning_candidates pc
JOIN wallet_sync_runs wsr ON wsr.id = pc.wallet_sync_run_id
JOIN wallets w ON w.id = pc.focal_wallet_id
JOIN transactions t ON t.signature = pc.signature AND t.transfer_index = pc.transfer_index
JOIN wallet_transactions wt ON wt.transaction_id = t.id AND wt.wallet_id = wsr.wallet_id
LEFT JOIN counterparties cp ON cp.focal_wallet_id = pc.focal_wallet_id AND cp.counterparty_address = pc.suspicious_counterparty
WHERE pc.wallet_sync_run_id = $1
  AND pc.signature = $2
  AND pc.transfer_index = $3
LIMIT 1`

	var rec CandidateExplanationRecord
	err := s.DB.QueryRowContext(ctx, q, walletSyncRunID, signature, transferIndex).Scan(
		&rec.WalletSyncRunID,
		&rec.IngestionRunID,
		&rec.FocalWallet,
		&rec.Signature,
		&rec.TransferIndex,
		&rec.BlockTime,
		&rec.SuspiciousCounterparty,
		&rec.MatchedLegitCounterparty,
		&rec.ScanStartAt,
		&rec.ScanEndAt,
		&rec.BaselineStartAt,
		&rec.BaselineEndAt,
		&rec.BaselineComplete,
		&rec.IncompleteWindow,
		&rec.UnknownGateReason,
		&rec.RelationType,
		&rec.AssetType,
		&rec.NormalizationStatus,
		&rec.PoisoningEligible,
		&rec.SourceOwner,
		&rec.DestinationOwner,
		&rec.FromTokenAccount,
		&rec.ToTokenAccount,
		&rec.TokenMint,
		&rec.AmountRaw,
		&rec.DustStatus,
		&rec.IsDust,
		&rec.IsZeroValue,
		&rec.IsInbound,
		&rec.IsNewCounterparty,
		&rec.RecencyDays,
		&rec.RepeatInjectionCount,
		&rec.LookalikePrefixMatch,
		&rec.LookalikeSuffixMatch,
		&rec.MatchRuleVersion,
		&rec.LegitLastSeenAt,
		&rec.CandidateCreatedAt,
		&rec.WalletTransactionID,
		&rec.TransactionID,
		&rec.CounterpartyID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return CandidateExplanationRecord{}, false, nil
		}
		return CandidateExplanationRecord{}, false, fmt.Errorf("get candidate explanation: %w", err)
	}
	return rec, true, nil
}

func (s *PostgresStore) ListCandidateExplanationsForRun(ctx context.Context, runID int64, limit, offset int) ([]CandidateExplanationRecord, int, error) {
	const countQ = `
SELECT COUNT(*)
FROM poisoning_candidates pc
JOIN wallet_sync_runs wsr ON wsr.id = pc.wallet_sync_run_id
WHERE wsr.ingestion_run_id = $1`
	var total int
	if err := s.DB.QueryRowContext(ctx, countQ, runID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count candidate explanations: %w", err)
	}

	const q = `
SELECT pc.wallet_sync_run_id,
       wsr.ingestion_run_id,
       w.address,
       pc.signature,
       pc.transfer_index,
       pc.block_time,
       pc.suspicious_counterparty,
       pc.matched_legit_counterparty,
       wsr.scan_start_at,
       wsr.scan_end_at,
       wsr.baseline_start_at,
       wsr.baseline_end_at,
       wsr.baseline_complete,
       pc.incomplete_window,
       COALESCE(NULLIF(BTRIM(pc.unknown_gate_reason), ''), ''),
       wt.relation_type,
       t.asset_type,
       t.normalization_status,
       t.poisoning_eligible,
       COALESCE(t.source_owner_address, ''),
       COALESCE(t.destination_owner_address, ''),
       COALESCE(t.source_token_account, ''),
       COALESCE(t.destination_token_account, ''),
       COALESCE(t.token_mint, ''),
       t.amount_raw::TEXT,
       t.dust_status,
       pc.is_dust,
       pc.is_zero_value,
       pc.is_inbound,
       pc.is_new_counterparty,
       pc.recency_days,
       pc.repeat_injection_count,
       0 AS prefix_match_len,
       0 AS suffix_match_len,
       pc.match_rule_version,
       pc.legit_last_seen_at,
       pc.created_at,
       wt.id,
       t.id,
       COALESCE(cp.id, 0)
FROM poisoning_candidates pc
JOIN wallet_sync_runs wsr ON wsr.id = pc.wallet_sync_run_id
JOIN wallets w ON w.id = pc.focal_wallet_id
JOIN transactions t ON t.signature = pc.signature AND t.transfer_index = pc.transfer_index
JOIN wallet_transactions wt ON wt.transaction_id = t.id AND wt.wallet_id = wsr.wallet_id
LEFT JOIN counterparties cp ON cp.focal_wallet_id = pc.focal_wallet_id AND cp.counterparty_address = pc.suspicious_counterparty
WHERE wsr.ingestion_run_id = $1
ORDER BY w.address ASC, pc.block_time ASC, pc.signature ASC, pc.transfer_index ASC
LIMIT $2 OFFSET $3`
	rows, err := s.DB.QueryContext(ctx, q, runID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list candidate explanations: %w", err)
	}
	defer rows.Close()

	out := make([]CandidateExplanationRecord, 0)
	for rows.Next() {
		var rec CandidateExplanationRecord
		if err := rows.Scan(
			&rec.WalletSyncRunID,
			&rec.IngestionRunID,
			&rec.FocalWallet,
			&rec.Signature,
			&rec.TransferIndex,
			&rec.BlockTime,
			&rec.SuspiciousCounterparty,
			&rec.MatchedLegitCounterparty,
			&rec.ScanStartAt,
			&rec.ScanEndAt,
			&rec.BaselineStartAt,
			&rec.BaselineEndAt,
			&rec.BaselineComplete,
			&rec.IncompleteWindow,
			&rec.UnknownGateReason,
			&rec.RelationType,
			&rec.AssetType,
			&rec.NormalizationStatus,
			&rec.PoisoningEligible,
			&rec.SourceOwner,
			&rec.DestinationOwner,
			&rec.FromTokenAccount,
			&rec.ToTokenAccount,
			&rec.TokenMint,
			&rec.AmountRaw,
			&rec.DustStatus,
			&rec.IsDust,
			&rec.IsZeroValue,
			&rec.IsInbound,
			&rec.IsNewCounterparty,
			&rec.RecencyDays,
			&rec.RepeatInjectionCount,
			&rec.LookalikePrefixMatch,
			&rec.LookalikeSuffixMatch,
			&rec.MatchRuleVersion,
			&rec.LegitLastSeenAt,
			&rec.CandidateCreatedAt,
			&rec.WalletTransactionID,
			&rec.TransactionID,
			&rec.CounterpartyID,
		); err != nil {
			return nil, 0, fmt.Errorf("scan candidate explanation row: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate candidate explanation rows: %w", err)
	}
	return out, total, nil
}

func (s *PostgresStore) ListWalletInspectionSummaryForRun(ctx context.Context, runID int64, limit, offset int) ([]WalletInspectionSummaryRecord, int, error) {
	const countQ = `SELECT COUNT(*) FROM wallet_sync_runs WHERE ingestion_run_id = $1`
	var total int
	if err := s.DB.QueryRowContext(ctx, countQ, runID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count wallet inspection summaries: %w", err)
	}
	const q = `
SELECT wsr.ingestion_run_id,
       wsr.id,
       w.address,
       (
         SELECT COUNT(*)
         FROM poisoning_candidates pc
         WHERE pc.wallet_sync_run_id = wsr.id
       ) AS candidate_count,
       CASE
         WHEN wsr.incomplete_window = TRUE AND COALESCE(NULLIF(BTRIM(wsr.unknown_gate_reason), ''), '') <> '' THEN 1
         ELSE 0
       END AS unknown_gate_block_count,
       wsr.incomplete_window,
       COALESCE(NULLIF(BTRIM(wsr.unknown_gate_reason), ''), ''),
       COALESCE(NULLIF(BTRIM(wsr.truncation_reason), ''), ''),
       wsr.baseline_complete,
       wsr.scan_start_at,
       wsr.scan_end_at,
       wsr.baseline_start_at,
       wsr.baseline_end_at,
       wsr.transactions_fetched,
       wsr.poisoning_candidates_inserted
FROM wallet_sync_runs wsr
JOIN wallets w ON w.id = wsr.wallet_id
WHERE wsr.ingestion_run_id = $1
ORDER BY w.address ASC, wsr.scan_end_at ASC, wsr.id ASC
LIMIT $2 OFFSET $3`
	rows, err := s.DB.QueryContext(ctx, q, runID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list wallet inspection summaries: %w", err)
	}
	defer rows.Close()

	out := make([]WalletInspectionSummaryRecord, 0)
	for rows.Next() {
		var rec WalletInspectionSummaryRecord
		if err := rows.Scan(
			&rec.RunID,
			&rec.WalletSyncRunID,
			&rec.FocalWallet,
			&rec.CandidateCount,
			&rec.UnknownGateBlockCount,
			&rec.IncompleteWindow,
			&rec.UnknownGateReason,
			&rec.TruncationReason,
			&rec.BaselineComplete,
			&rec.ScanStartAt,
			&rec.ScanEndAt,
			&rec.BaselineStartAt,
			&rec.BaselineEndAt,
			&rec.TransactionsFetched,
			&rec.PoisoningCandidatesSeen,
		); err != nil {
			return nil, 0, fmt.Errorf("scan wallet inspection summary row: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate wallet inspection summary rows: %w", err)
	}
	return out, total, nil
}
