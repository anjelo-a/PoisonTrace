package storage

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type ExportFilter struct {
	RunID         *int64
	StartedAtFrom *time.Time
	StartedAtTo   *time.Time
}

type IngestionRunExportRecord struct {
	ID                          int64
	Status                      string
	StartedAt                   time.Time
	CompletedAt                 *time.Time
	WalletsRequested            int
	WalletsProcessed            int
	WalletsFailed               int
	WalletsSkipped              int
	TruncationWalletCount       int
	TruncationWalletRate        string
	TransactionsFetched         int
	TransactionsInserted        int
	TransactionsLinked          int
	TransactionsFailedNormalize int
	OwnerUnresolvedCount        int
	DecimalsUnresolvedCount     int
	CounterpartiesCreated       int
	CounterpartiesUpdated       int
	PoisoningCandidatesInserted int
	RetryExhaustedCount         int
	Notes                       string
}

type WalletSyncRunExportRecord struct {
	WalletSyncRunID             int64
	IngestionRunID              int64
	FocalWallet                 string
	Status                      string
	StartedAt                   time.Time
	CompletedAt                 *time.Time
	BaselineStartAt             time.Time
	BaselineEndAt               time.Time
	ScanStartAt                 time.Time
	ScanEndAt                   time.Time
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
	ErrorCode                   string
	ErrorMessage                string
	Notes                       string
}

type PoisoningCandidateExportRecord struct {
	IngestionRunID           int64
	WalletSyncRunID          int64
	FocalWallet              string
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

func (s *PostgresStore) ListIngestionRunsForExport(ctx context.Context, filter ExportFilter) ([]IngestionRunExportRecord, error) {
	where, args, err := buildExportFilterWhere(filter, "ir")
	if err != nil {
		return nil, err
	}
	const baseQuery = `
SELECT ir.id,
       ir.status,
       ir.started_at,
       ir.completed_at,
       ir.wallets_requested,
       ir.wallets_processed,
       ir.wallets_failed,
       ir.wallets_skipped,
       ir.truncation_wallet_count,
       ir.truncation_wallet_rate::TEXT,
       ir.transactions_fetched,
       ir.transactions_inserted,
       ir.transactions_linked,
       ir.transactions_failed_to_normalize,
       ir.owner_unresolved_count,
       ir.decimals_unresolved_count,
       ir.counterparties_created,
       ir.counterparties_updated,
       ir.poisoning_candidates_inserted,
       ir.retry_exhausted_count,
       COALESCE(ir.notes, '')
FROM ingestion_runs ir`
	query := baseQuery + where + " ORDER BY ir.id ASC"

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list ingestion runs for export: %w", err)
	}
	defer rows.Close()

	out := make([]IngestionRunExportRecord, 0)
	for rows.Next() {
		var rec IngestionRunExportRecord
		if err := rows.Scan(
			&rec.ID,
			&rec.Status,
			&rec.StartedAt,
			&rec.CompletedAt,
			&rec.WalletsRequested,
			&rec.WalletsProcessed,
			&rec.WalletsFailed,
			&rec.WalletsSkipped,
			&rec.TruncationWalletCount,
			&rec.TruncationWalletRate,
			&rec.TransactionsFetched,
			&rec.TransactionsInserted,
			&rec.TransactionsLinked,
			&rec.TransactionsFailedNormalize,
			&rec.OwnerUnresolvedCount,
			&rec.DecimalsUnresolvedCount,
			&rec.CounterpartiesCreated,
			&rec.CounterpartiesUpdated,
			&rec.PoisoningCandidatesInserted,
			&rec.RetryExhaustedCount,
			&rec.Notes,
		); err != nil {
			return nil, fmt.Errorf("scan ingestion export row: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ingestion export rows: %w", err)
	}
	return out, nil
}

func (s *PostgresStore) ListWalletSyncRunsForExport(ctx context.Context, filter ExportFilter) ([]WalletSyncRunExportRecord, error) {
	where, args, err := buildExportFilterWhere(filter, "ir")
	if err != nil {
		return nil, err
	}
	const baseQuery = `
SELECT wsr.id,
       wsr.ingestion_run_id,
       w.address,
       wsr.status,
       wsr.started_at,
       wsr.completed_at,
       wsr.baseline_start_at,
       wsr.baseline_end_at,
       wsr.scan_start_at,
       wsr.scan_end_at,
       wsr.baseline_complete,
       wsr.incomplete_window,
       COALESCE(wsr.unknown_gate_reason, ''),
       COALESCE(wsr.truncation_reason, ''),
       wsr.transactions_fetched,
       wsr.transactions_inserted,
       wsr.transactions_linked,
       wsr.transactions_failed_to_normalize,
       wsr.counterparties_created,
       wsr.counterparties_updated,
       wsr.poisoning_candidates_inserted,
       COALESCE(wsr.error_code, ''),
       COALESCE(wsr.error_message, ''),
       COALESCE(wsr.notes, '')
FROM wallet_sync_runs wsr
JOIN wallets w ON w.id = wsr.wallet_id
JOIN ingestion_runs ir ON ir.id = wsr.ingestion_run_id`
	query := baseQuery + where + " ORDER BY w.address ASC, wsr.scan_start_at ASC, wsr.id ASC"

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list wallet sync runs for export: %w", err)
	}
	defer rows.Close()

	out := make([]WalletSyncRunExportRecord, 0)
	for rows.Next() {
		var rec WalletSyncRunExportRecord
		if err := rows.Scan(
			&rec.WalletSyncRunID,
			&rec.IngestionRunID,
			&rec.FocalWallet,
			&rec.Status,
			&rec.StartedAt,
			&rec.CompletedAt,
			&rec.BaselineStartAt,
			&rec.BaselineEndAt,
			&rec.ScanStartAt,
			&rec.ScanEndAt,
			&rec.BaselineComplete,
			&rec.IncompleteWindow,
			&rec.UnknownGateReason,
			&rec.TruncationReason,
			&rec.TransactionsFetched,
			&rec.TransactionsInserted,
			&rec.TransactionsLinked,
			&rec.TransactionsFailedNormalize,
			&rec.CounterpartiesCreated,
			&rec.CounterpartiesUpdated,
			&rec.PoisoningCandidatesInserted,
			&rec.ErrorCode,
			&rec.ErrorMessage,
			&rec.Notes,
		); err != nil {
			return nil, fmt.Errorf("scan wallet sync export row: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate wallet sync export rows: %w", err)
	}
	return out, nil
}

func (s *PostgresStore) ListPoisoningCandidatesForExport(ctx context.Context, filter ExportFilter) ([]PoisoningCandidateExportRecord, error) {
	where, args, err := buildExportFilterWhere(filter, "ir")
	if err != nil {
		return nil, err
	}
	const baseQuery = `
SELECT wsr.ingestion_run_id,
       pc.wallet_sync_run_id,
       w.address,
       pc.signature,
       pc.transfer_index,
       pc.suspicious_counterparty,
       pc.matched_legit_counterparty,
       COALESCE(pc.token_mint, ''),
       pc.amount_raw::TEXT,
       pc.block_time,
       pc.is_zero_value,
       pc.is_dust,
       pc.is_new_counterparty,
       pc.is_inbound,
       pc.legit_last_seen_at,
       pc.recency_days,
       pc.repeat_injection_count,
       pc.incomplete_window,
       COALESCE(pc.unknown_gate_reason, ''),
       pc.match_rule_version
FROM poisoning_candidates pc
JOIN wallet_sync_runs wsr ON wsr.id = pc.wallet_sync_run_id
JOIN wallets w ON w.id = pc.focal_wallet_id
JOIN ingestion_runs ir ON ir.id = wsr.ingestion_run_id`
	query := baseQuery + where + " ORDER BY w.address ASC, pc.block_time ASC, pc.signature ASC, pc.transfer_index ASC"

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list poisoning candidates for export: %w", err)
	}
	defer rows.Close()

	out := make([]PoisoningCandidateExportRecord, 0)
	for rows.Next() {
		var rec PoisoningCandidateExportRecord
		if err := rows.Scan(
			&rec.IngestionRunID,
			&rec.WalletSyncRunID,
			&rec.FocalWallet,
			&rec.Signature,
			&rec.TransferIndex,
			&rec.SuspiciousCounterparty,
			&rec.MatchedLegitCounterparty,
			&rec.TokenMint,
			&rec.AmountRaw,
			&rec.BlockTime,
			&rec.IsZeroValue,
			&rec.IsDust,
			&rec.IsNewCounterparty,
			&rec.IsInbound,
			&rec.LegitLastSeenAt,
			&rec.RecencyDays,
			&rec.RepeatInjectionCount,
			&rec.IncompleteWindow,
			&rec.UnknownGateReason,
			&rec.MatchRuleVersion,
		); err != nil {
			return nil, fmt.Errorf("scan poisoning candidate export row: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate poisoning candidate export rows: %w", err)
	}
	return out, nil
}

func buildExportFilterWhere(filter ExportFilter, runAlias string) (string, []any, error) {
	if filter.RunID != nil && (filter.StartedAtFrom != nil || filter.StartedAtTo != nil) {
		return "", nil, fmt.Errorf("export filter: run_id cannot be combined with started_at range")
	}

	clauses := make([]string, 0, 3)
	args := make([]any, 0, 3)
	if filter.RunID != nil {
		args = append(args, *filter.RunID)
		clauses = append(clauses, fmt.Sprintf("%s.id = $%d", runAlias, len(args)))
	}
	if filter.StartedAtFrom != nil {
		args = append(args, filter.StartedAtFrom.UTC())
		clauses = append(clauses, fmt.Sprintf("%s.started_at >= $%d", runAlias, len(args)))
	}
	if filter.StartedAtTo != nil {
		args = append(args, filter.StartedAtTo.UTC())
		clauses = append(clauses, fmt.Sprintf("%s.started_at < $%d", runAlias, len(args)))
	}
	if len(clauses) == 0 {
		return "", args, nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args, nil
}
