package exports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"poisontrace/internal/storage"
)

const schemaVersion = "phase5-v1"

type DatasetSource interface {
	ListIngestionRunsForExport(ctx context.Context, filter storage.ExportFilter) ([]storage.IngestionRunExportRecord, error)
	ListWalletSyncRunsForExport(ctx context.Context, filter storage.ExportFilter) ([]storage.WalletSyncRunExportRecord, error)
	ListPoisoningCandidatesForExport(ctx context.Context, filter storage.ExportFilter) ([]storage.PoisoningCandidateExportRecord, error)
	ListCandidateExplanationsForExport(ctx context.Context, filter storage.ExportFilter) ([]storage.CandidateExplanationExportRecord, error)
	ListWalletInspectionSummaryForExport(ctx context.Context, filter storage.ExportFilter) ([]storage.WalletInspectionSummaryExportRecord, error)
}

type ExportOptions struct {
	OutDir string
	Filter storage.ExportFilter
}

type Manifest struct {
	SchemaVersion string         `json:"schema_version"`
	GeneratedAt   string         `json:"generated_at"`
	SourceFilters SourceFilters  `json:"source_filters"`
	Files         []ManifestFile `json:"files"`
}

type SourceFilters struct {
	RunID         *int64 `json:"run_id,omitempty"`
	StartedAtFrom string `json:"started_at_from,omitempty"`
	StartedAtTo   string `json:"started_at_to,omitempty"`
}

type ManifestFile struct {
	Name     string `json:"name"`
	RowCount int    `json:"row_count"`
	SHA256   string `json:"sha256"`
}

type ExportResult struct {
	Manifest Manifest
}

func ExportDataset(ctx context.Context, source DatasetSource, opts ExportOptions) (ExportResult, error) {
	if strings.TrimSpace(opts.OutDir) == "" {
		return ExportResult{}, fmt.Errorf("out dir is required")
	}
	if source == nil {
		return ExportResult{}, fmt.Errorf("dataset source is required")
	}
	if opts.Filter.RunID == nil && (opts.Filter.StartedAtFrom == nil || opts.Filter.StartedAtTo == nil) {
		return ExportResult{}, fmt.Errorf("export filter requires either run_id or both started_at_from and started_at_to")
	}

	ingestionRuns, err := source.ListIngestionRunsForExport(ctx, opts.Filter)
	if err != nil {
		return ExportResult{}, err
	}
	walletSyncRuns, err := source.ListWalletSyncRunsForExport(ctx, opts.Filter)
	if err != nil {
		return ExportResult{}, err
	}
	candidates, err := source.ListPoisoningCandidatesForExport(ctx, opts.Filter)
	if err != nil {
		return ExportResult{}, err
	}
	sortIngestionRuns(ingestionRuns)
	sortWalletSyncRuns(walletSyncRuns)
	sortPoisoningCandidates(candidates)
	candidateExplanations, err := source.ListCandidateExplanationsForExport(ctx, opts.Filter)
	if err != nil {
		return ExportResult{}, err
	}
	walletSummaries, err := source.ListWalletInspectionSummaryForExport(ctx, opts.Filter)
	if err != nil {
		return ExportResult{}, err
	}
	sortCandidateExplanations(candidateExplanations)
	sortWalletInspectionSummary(walletSummaries)

	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return ExportResult{}, fmt.Errorf("create out dir: %w", err)
	}

	artifacts := []struct {
		name    string
		rows    int
		payload []byte
	}{
		{name: "ingestion_runs.jsonl", rows: len(ingestionRuns)},
		{name: "wallet_sync_runs.jsonl", rows: len(walletSyncRuns)},
		{name: "poisoning_candidates.jsonl", rows: len(candidates)},
		{name: "candidate_explanations.jsonl", rows: len(candidateExplanations)},
		{name: "candidate_explanations.csv", rows: len(candidateExplanations)},
		{name: "wallet_inspection_summary.csv", rows: len(walletSummaries)},
	}

	artifacts[0].payload, err = encodeJSONL(ingestionRuns)
	if err != nil {
		return ExportResult{}, fmt.Errorf("encode ingestion runs: %w", err)
	}
	artifacts[1].payload, err = encodeJSONL(walletSyncRuns)
	if err != nil {
		return ExportResult{}, fmt.Errorf("encode wallet sync runs: %w", err)
	}
	artifacts[2].payload, err = encodeJSONL(candidates)
	if err != nil {
		return ExportResult{}, fmt.Errorf("encode poisoning candidates: %w", err)
	}
	artifacts[3].payload, err = encodeJSONL(candidateExplanations)
	if err != nil {
		return ExportResult{}, fmt.Errorf("encode candidate explanations: %w", err)
	}
	artifacts[4].payload, err = encodeCandidateExplanationsCSV(candidateExplanations)
	if err != nil {
		return ExportResult{}, fmt.Errorf("encode candidate explanations csv: %w", err)
	}
	artifacts[5].payload, err = encodeWalletInspectionSummaryCSV(walletSummaries)
	if err != nil {
		return ExportResult{}, fmt.Errorf("encode wallet inspection summary csv: %w", err)
	}

	manifest := Manifest{
		SchemaVersion: schemaVersion,
		GeneratedAt:   deriveGeneratedAt(ingestionRuns),
		SourceFilters: buildSourceFilters(opts.Filter),
		Files:         make([]ManifestFile, 0, len(artifacts)),
	}

	for _, artifact := range artifacts {
		path := filepath.Join(opts.OutDir, artifact.name)
		if err := os.WriteFile(path, artifact.payload, 0o644); err != nil {
			return ExportResult{}, fmt.Errorf("write %s: %w", artifact.name, err)
		}
		h := sha256.Sum256(artifact.payload)
		manifest.Files = append(manifest.Files, ManifestFile{
			Name:     artifact.name,
			RowCount: artifact.rows,
			SHA256:   hex.EncodeToString(h[:]),
		})
	}
	sort.Slice(manifest.Files, func(i, j int) bool {
		return manifest.Files[i].Name < manifest.Files[j].Name
	})

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return ExportResult{}, fmt.Errorf("encode manifest: %w", err)
	}
	manifestBytes = append(manifestBytes, '\n')
	if err := os.WriteFile(filepath.Join(opts.OutDir, "report_manifest.json"), manifestBytes, 0o644); err != nil {
		return ExportResult{}, fmt.Errorf("write report_manifest.json: %w", err)
	}

	return ExportResult{Manifest: manifest}, nil
}

func encodeCandidateExplanationsCSV(rows []storage.CandidateExplanationExportRecord) ([]byte, error) {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	header := []string{
		"ingestion_run_id", "wallet_sync_run_id", "focal_wallet", "signature", "transfer_index", "block_time",
		"suspicious_counterparty", "matched_legit_counterparty", "relation_type", "asset_type", "normalization_status",
		"poisoning_eligible", "source_owner", "destination_owner", "from_token_account", "to_token_account",
		"token_mint", "amount_raw", "dust_status", "is_dust", "is_zero_value", "is_inbound", "is_new_counterparty",
		"recency_days", "repeat_injection_count", "baseline_start_at", "baseline_end_at", "scan_start_at", "scan_end_at",
		"baseline_complete", "incomplete_window", "unknown_gate_reason", "match_rule_version",
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}
	for _, r := range rows {
		row := []string{
			fmt.Sprintf("%d", r.IngestionRunID),
			fmt.Sprintf("%d", r.WalletSyncRunID),
			r.FocalWallet,
			r.Signature,
			fmt.Sprintf("%d", r.TransferIndex),
			r.BlockTime.UTC().Format(time.RFC3339),
			r.SuspiciousCounterparty,
			r.MatchedLegitCounterparty,
			r.RelationType,
			r.AssetType,
			r.NormalizationStatus,
			fmt.Sprintf("%t", r.PoisoningEligible),
			r.SourceOwner,
			r.DestinationOwner,
			r.FromTokenAccount,
			r.ToTokenAccount,
			r.TokenMint,
			r.AmountRaw,
			r.DustStatus,
			fmt.Sprintf("%t", r.IsDust),
			fmt.Sprintf("%t", r.IsZeroValue),
			fmt.Sprintf("%t", r.IsInbound),
			fmt.Sprintf("%t", r.IsNewCounterparty),
			fmt.Sprintf("%d", r.RecencyDays),
			fmt.Sprintf("%d", r.RepeatInjectionCount),
			r.BaselineStartAt.UTC().Format(time.RFC3339),
			r.BaselineEndAt.UTC().Format(time.RFC3339),
			r.ScanStartAt.UTC().Format(time.RFC3339),
			r.ScanEndAt.UTC().Format(time.RFC3339),
			fmt.Sprintf("%t", r.BaselineComplete),
			fmt.Sprintf("%t", r.IncompleteWindow),
			r.UnknownGateReason,
			r.MatchRuleVersion,
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func encodeWalletInspectionSummaryCSV(rows []storage.WalletInspectionSummaryExportRecord) ([]byte, error) {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	header := []string{
		"ingestion_run_id", "wallet_sync_run_id", "focal_wallet", "candidate_count", "unknown_gate_block_count",
		"incomplete_window", "unknown_gate_reason", "truncation_reason", "baseline_complete",
		"baseline_start_at", "baseline_end_at", "scan_start_at", "scan_end_at", "transactions_fetched", "poisoning_candidates_inserted",
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}
	for _, r := range rows {
		row := []string{
			fmt.Sprintf("%d", r.IngestionRunID),
			fmt.Sprintf("%d", r.WalletSyncRunID),
			r.FocalWallet,
			fmt.Sprintf("%d", r.CandidateCount),
			fmt.Sprintf("%d", r.UnknownGateBlockCount),
			fmt.Sprintf("%t", r.IncompleteWindow),
			r.UnknownGateReason,
			r.TruncationReason,
			fmt.Sprintf("%t", r.BaselineComplete),
			r.BaselineStartAt.UTC().Format(time.RFC3339),
			r.BaselineEndAt.UTC().Format(time.RFC3339),
			r.ScanStartAt.UTC().Format(time.RFC3339),
			r.ScanEndAt.UTC().Format(time.RFC3339),
			fmt.Sprintf("%d", r.TransactionsFetched),
			fmt.Sprintf("%d", r.PoisoningCandidatesSeen),
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func encodeJSONL[T any](rows []T) ([]byte, error) {
	if len(rows) == 0 {
		return []byte{}, nil
	}
	buf := make([]byte, 0, len(rows)*64)
	for _, row := range rows {
		line, err := json.Marshal(row)
		if err != nil {
			return nil, err
		}
		buf = append(buf, line...)
		buf = append(buf, '\n')
	}
	return buf, nil
}

func deriveGeneratedAt(runs []storage.IngestionRunExportRecord) string {
	if len(runs) == 0 {
		return time.Unix(0, 0).UTC().Format(time.RFC3339Nano)
	}
	maxAt := time.Unix(0, 0).UTC()
	for _, run := range runs {
		candidate := run.StartedAt.UTC()
		if run.CompletedAt != nil {
			candidate = run.CompletedAt.UTC()
		}
		if candidate.After(maxAt) {
			maxAt = candidate
		}
	}
	return maxAt.Format(time.RFC3339Nano)
}

func buildSourceFilters(filter storage.ExportFilter) SourceFilters {
	out := SourceFilters{RunID: filter.RunID}
	if filter.StartedAtFrom != nil {
		out.StartedAtFrom = filter.StartedAtFrom.UTC().Format(time.RFC3339)
	}
	if filter.StartedAtTo != nil {
		out.StartedAtTo = filter.StartedAtTo.UTC().Format(time.RFC3339)
	}
	return out
}

func sortIngestionRuns(rows []storage.IngestionRunExportRecord) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].StartedAt.Equal(rows[j].StartedAt) {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].StartedAt.Before(rows[j].StartedAt)
	})
}

func sortWalletSyncRuns(rows []storage.WalletSyncRunExportRecord) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].FocalWallet == rows[j].FocalWallet {
			if rows[i].ScanStartAt.Equal(rows[j].ScanStartAt) {
				return rows[i].WalletSyncRunID < rows[j].WalletSyncRunID
			}
			return rows[i].ScanStartAt.Before(rows[j].ScanStartAt)
		}
		return rows[i].FocalWallet < rows[j].FocalWallet
	})
}

func sortPoisoningCandidates(rows []storage.PoisoningCandidateExportRecord) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].FocalWallet == rows[j].FocalWallet {
			if rows[i].BlockTime.Equal(rows[j].BlockTime) {
				if rows[i].Signature == rows[j].Signature {
					return rows[i].TransferIndex < rows[j].TransferIndex
				}
				return rows[i].Signature < rows[j].Signature
			}
			return rows[i].BlockTime.Before(rows[j].BlockTime)
		}
		return rows[i].FocalWallet < rows[j].FocalWallet
	})
}

func sortCandidateExplanations(rows []storage.CandidateExplanationExportRecord) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].FocalWallet == rows[j].FocalWallet {
			if rows[i].BlockTime.Equal(rows[j].BlockTime) {
				if rows[i].Signature == rows[j].Signature {
					return rows[i].TransferIndex < rows[j].TransferIndex
				}
				return rows[i].Signature < rows[j].Signature
			}
			return rows[i].BlockTime.Before(rows[j].BlockTime)
		}
		return rows[i].FocalWallet < rows[j].FocalWallet
	})
}

func sortWalletInspectionSummary(rows []storage.WalletInspectionSummaryExportRecord) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].FocalWallet == rows[j].FocalWallet {
			if rows[i].ScanEndAt.Equal(rows[j].ScanEndAt) {
				return rows[i].WalletSyncRunID < rows[j].WalletSyncRunID
			}
			return rows[i].ScanEndAt.Before(rows[j].ScanEndAt)
		}
		return rows[i].FocalWallet < rows[j].FocalWallet
	})
}
