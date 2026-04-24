package exports

import (
	"context"
	"crypto/sha256"
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

const schemaVersion = "phase4-v1"

type DatasetSource interface {
	ListIngestionRunsForExport(ctx context.Context, filter storage.ExportFilter) ([]storage.IngestionRunExportRecord, error)
	ListWalletSyncRunsForExport(ctx context.Context, filter storage.ExportFilter) ([]storage.WalletSyncRunExportRecord, error)
	ListPoisoningCandidatesForExport(ctx context.Context, filter storage.ExportFilter) ([]storage.PoisoningCandidateExportRecord, error)
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
	if err := os.WriteFile(filepath.Join(opts.OutDir, "manifest.json"), manifestBytes, 0o644); err != nil {
		return ExportResult{}, fmt.Errorf("write manifest.json: %w", err)
	}

	return ExportResult{Manifest: manifest}, nil
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
