package exports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"poisontrace/internal/storage"
)

type stubSource struct {
	ingestion []storage.IngestionRunExportRecord
	wallets   []storage.WalletSyncRunExportRecord
	cand      []storage.PoisoningCandidateExportRecord
}

func (s *stubSource) ListIngestionRunsForExport(context.Context, storage.ExportFilter) ([]storage.IngestionRunExportRecord, error) {
	return append([]storage.IngestionRunExportRecord{}, s.ingestion...), nil
}

func (s *stubSource) ListWalletSyncRunsForExport(context.Context, storage.ExportFilter) ([]storage.WalletSyncRunExportRecord, error) {
	return append([]storage.WalletSyncRunExportRecord{}, s.wallets...), nil
}

func (s *stubSource) ListPoisoningCandidatesForExport(context.Context, storage.ExportFilter) ([]storage.PoisoningCandidateExportRecord, error) {
	return append([]storage.PoisoningCandidateExportRecord{}, s.cand...), nil
}

func TestExportDatasetDeterministicAcrossInputOrder(t *testing.T) {
	completed := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	runID := int64(99)
	filter := storage.ExportFilter{RunID: &runID}

	base := &stubSource{
		ingestion: []storage.IngestionRunExportRecord{{ID: 99, Status: "succeeded", StartedAt: completed.Add(-time.Hour), CompletedAt: &completed, TruncationWalletRate: "0.50000000"}},
		wallets: []storage.WalletSyncRunExportRecord{
			{WalletSyncRunID: 2, IngestionRunID: 99, FocalWallet: "WalletB", ScanStartAt: completed},
			{WalletSyncRunID: 1, IngestionRunID: 99, FocalWallet: "WalletA", ScanStartAt: completed},
		},
		cand: []storage.PoisoningCandidateExportRecord{
			{IngestionRunID: 99, WalletSyncRunID: 2, FocalWallet: "WalletB", Signature: "sigB", TransferIndex: 1, BlockTime: completed.Add(2 * time.Minute)},
			{IngestionRunID: 99, WalletSyncRunID: 1, FocalWallet: "WalletA", Signature: "sigA", TransferIndex: 0, BlockTime: completed.Add(time.Minute)},
		},
	}
	shuffled := &stubSource{
		ingestion: append([]storage.IngestionRunExportRecord{}, base.ingestion...),
		wallets:   []storage.WalletSyncRunExportRecord{base.wallets[1], base.wallets[0]},
		cand:      []storage.PoisoningCandidateExportRecord{base.cand[1], base.cand[0]},
	}

	out1 := filepath.Join(t.TempDir(), "a")
	out2 := filepath.Join(t.TempDir(), "b")
	if _, err := ExportDataset(context.Background(), base, ExportOptions{OutDir: out1, Filter: filter}); err != nil {
		t.Fatalf("export 1: %v", err)
	}
	if _, err := ExportDataset(context.Background(), shuffled, ExportOptions{OutDir: out2, Filter: filter}); err != nil {
		t.Fatalf("export 2: %v", err)
	}

	files := []string{"ingestion_runs.jsonl", "wallet_sync_runs.jsonl", "poisoning_candidates.jsonl", "manifest.json"}
	for _, name := range files {
		left, err := os.ReadFile(filepath.Join(out1, name))
		if err != nil {
			t.Fatalf("read left %s: %v", name, err)
		}
		right, err := os.ReadFile(filepath.Join(out2, name))
		if err != nil {
			t.Fatalf("read right %s: %v", name, err)
		}
		if string(left) != string(right) {
			t.Fatalf("expected identical file %s across exports", name)
		}
	}
}

func TestExportDatasetManifestHashesMatchFileBytes(t *testing.T) {
	runID := int64(100)
	out := t.TempDir()
	res, err := ExportDataset(context.Background(), &stubSource{
		ingestion: []storage.IngestionRunExportRecord{{ID: 100, Status: "succeeded", StartedAt: time.Date(2026, 4, 24, 0, 0, 0, 0, time.UTC), TruncationWalletRate: "0"}},
	}, ExportOptions{OutDir: out, Filter: storage.ExportFilter{RunID: &runID}})
	if err != nil {
		t.Fatalf("export dataset: %v", err)
	}
	if res.Manifest.SchemaVersion != schemaVersion {
		t.Fatalf("unexpected schema version: %s", res.Manifest.SchemaVersion)
	}

	for _, file := range res.Manifest.Files {
		payload, err := os.ReadFile(filepath.Join(out, file.Name))
		if err != nil {
			t.Fatalf("read exported file: %v", err)
		}
		h := sha256.Sum256(payload)
		if file.SHA256 != hex.EncodeToString(h[:]) {
			t.Fatalf("hash mismatch for %s", file.Name)
		}
	}
}

func TestExportDatasetPreservesIncompleteWindowSignals(t *testing.T) {
	runID := int64(77)
	out := t.TempDir()
	_, err := ExportDataset(context.Background(), &stubSource{
		ingestion: []storage.IngestionRunExportRecord{{ID: 77, Status: "partially_succeeded", StartedAt: time.Date(2026, 4, 24, 0, 0, 0, 0, time.UTC), TruncationWalletRate: "1.00000000"}},
		wallets: []storage.WalletSyncRunExportRecord{{
			WalletSyncRunID:   5,
			IngestionRunID:    77,
			FocalWallet:       "WalletZ",
			IncompleteWindow:  true,
			UnknownGateReason: "unknown_required_gates:zero_or_dust",
			TruncationReason:  "scan_truncation:max_tx_cap",
		}},
	}, ExportOptions{OutDir: out, Filter: storage.ExportFilter{RunID: &runID}})
	if err != nil {
		t.Fatalf("export dataset: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(out, "wallet_sync_runs.jsonl"))
	if err != nil {
		t.Fatalf("read wallet_sync_runs.jsonl: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected one wallet sync row, got %d", len(lines))
	}
	var row storage.WalletSyncRunExportRecord
	if err := json.Unmarshal([]byte(lines[0]), &row); err != nil {
		t.Fatalf("decode row: %v", err)
	}
	if !row.IncompleteWindow {
		t.Fatal("expected incomplete_window=true")
	}
	if row.UnknownGateReason == "" || row.TruncationReason == "" {
		t.Fatalf("expected unknown/truncation reason to be preserved, got %+v", row)
	}
}
