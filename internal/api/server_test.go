package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"poisontrace/internal/config"
	"poisontrace/internal/storage"
)

type fakeRepo struct {
	candidates []storage.CandidateListRecord
	walletSync []storage.WalletSyncListRecord
}

func (f fakeRepo) GetOverviewMetrics(ctx context.Context, since time.Time) (storage.OverviewMetricsRecord, error) {
	return storage.OverviewMetricsRecord{}, nil
}
func (f fakeRepo) ListOverviewCandidates(ctx context.Context, limit int) ([]storage.OverviewCandidateRecord, error) {
	return nil, nil
}
func (f fakeRepo) ListCandidates(ctx context.Context, limit, offset int) ([]storage.CandidateListRecord, int, error) {
	return f.candidates, len(f.candidates), nil
}
func (f fakeRepo) ListTransactions(ctx context.Context, limit, offset int) ([]storage.TransactionListRecord, int, error) {
	return nil, 0, nil
}
func (f fakeRepo) ListRuns(ctx context.Context, limit, offset int) ([]storage.IngestionRunListRecord, int, error) {
	return nil, 0, nil
}
func (f fakeRepo) ListWalletSyncRuns(ctx context.Context, limit, offset int) ([]storage.WalletSyncListRecord, int, error) {
	return f.walletSync, len(f.walletSync), nil
}
func (f fakeRepo) ListCounterparties(ctx context.Context, limit, offset int) ([]storage.CounterpartyListRecord, int, error) {
	return nil, 0, nil
}
func (f fakeRepo) GetCandidateExplanation(ctx context.Context, walletSyncRunID int64, signature string, transferIndex int) (storage.CandidateExplanationRecord, bool, error) {
	return storage.CandidateExplanationRecord{}, false, nil
}
func (f fakeRepo) ListCandidateExplanationsForRun(ctx context.Context, runID int64, limit, offset int) ([]storage.CandidateExplanationRecord, int, error) {
	return nil, 0, nil
}
func (f fakeRepo) ListWalletInspectionSummaryForRun(ctx context.Context, runID int64, limit, offset int) ([]storage.WalletInspectionSummaryRecord, int, error) {
	return nil, 0, nil
}

func TestCandidatesAPI_NoUnknownOrIncompleteCandidates(t *testing.T) {
	now := time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)
	repo := fakeRepo{candidates: []storage.CandidateListRecord{
		{
			WalletSyncRunID:          1,
			FocalWallet:              "wallet",
			Signature:                "sig",
			TransferIndex:            0,
			BlockTime:                now,
			SuspiciousCounterparty:   "sus",
			MatchedLegitCounterparty: "legit",
			RepeatInjectionCount:     2,
			RecencyDays:              1,
			UnknownGateReason:        "",
			IncompleteWindow:         false,
		},
	}}

	s := NewServer(repo, config.Config{})
	req := httptest.NewRequest(http.MethodGet, "/api/candidates", nil)
	res := httptest.NewRecorder()
	s.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	var body struct {
		Items []struct {
			UnknownGateReason string `json:"unknownGateReason"`
			IncompleteWindow  bool   `json:"incompleteWindow"`
		} `json:"items"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(body.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(body.Items))
	}
	if body.Items[0].UnknownGateReason != "" || body.Items[0].IncompleteWindow {
		t.Fatalf("expected emitted candidates to have no unknown gate and no incomplete window")
	}
}

func TestRunsAPI_PostStartsManualRun(t *testing.T) {
	now := time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)
	var got RunStartRequest
	s := NewServer(fakeRepo{}, config.Config{
		DatabaseURL:              "postgres://postgres:postgres@localhost:5432/poisontrace?sslmode=disable",
		HeliusBaseURL:            "https://api.helius.xyz/v0",
		MaxWalletsPerRun:         5,
		MaxTXPagesPerWallet:      1,
		MaxTXPerWallet:           1,
		MaxConcurrentWallets:     1,
		WalletSyncTimeoutSeconds: 10,
		RunTimeoutSeconds:        10,
		BaselineLookbackDays:     30,
		ScanWindowDays:           7,
		LookalikeRecencyDays:     30,
		LookalikePrefixMin:       4,
		LookalikeSuffixMin:       4,
		LookalikeSingleSideMin:   6,
		MinInjectionCount:        2,
		DustThresholdsSeedPath:   "db/seeds/asset_thresholds.seed.sql",
	}).WithRunStarter(func(ctx context.Context, req RunStartRequest) (RunStartResult, error) {
		got = req
		return RunStartResult{
			RunID:                99,
			WalletCount:          len(req.WalletAddresses),
			ScanStart:            req.ScanStart,
			ScanEnd:              req.ScanEnd,
			BaselineLookbackDays: req.BaselineLookbackDays,
		}, nil
	})

	body := []byte(`{"addresses":"walletA\nwalletB walletA","scanStart":"2026-05-01T00:00:00Z","scanEnd":"2026-05-02T00:00:00Z","baselineLookbackDays":30}`)
	req := httptest.NewRequest(http.MethodPost, "/api/runs", bytes.NewReader(body))
	res := httptest.NewRecorder()
	s.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", res.Code, res.Body.String())
	}
	if len(got.WalletAddresses) != 2 || got.WalletAddresses[0] != "walletA" || got.WalletAddresses[1] != "walletB" {
		t.Fatalf("unexpected wallet addresses: %#v", got.WalletAddresses)
	}
	if !got.ScanStart.Equal(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)) || !got.ScanEnd.Equal(now.Add(-10*time.Hour)) {
		t.Fatalf("unexpected scan window: %s - %s", got.ScanStart, got.ScanEnd)
	}
}

func TestWalletSyncAPI_IncompleteWindowMissingReasonReturnsError(t *testing.T) {
	now := time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)
	repo := fakeRepo{walletSync: []storage.WalletSyncListRecord{
		{
			WalletSyncRunID:     1,
			IngestionRunID:      9,
			FocalWallet:         "wallet",
			Status:              "partial",
			BaselineStartAt:     now,
			BaselineEndAt:       now,
			ScanStartAt:         now,
			ScanEndAt:           now,
			BaselineComplete:    false,
			IncompleteWindow:    true,
			UnknownGateReason:   "",
			TransactionsFetched: 10,
			TruncationReason:    "timeout",
			UpdatedAt:           now,
		},
	}}

	s := NewServer(repo, config.Config{})
	req := httptest.NewRequest(http.MethodGet, "/api/wallet-sync", nil)
	res := httptest.NewRecorder()
	s.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", res.Code)
	}
}

func TestWalletSyncAPI_UsesRunIDAndReasonAsIs(t *testing.T) {
	now := time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)
	repo := fakeRepo{walletSync: []storage.WalletSyncListRecord{
		{
			WalletSyncRunID:     1,
			IngestionRunID:      9,
			FocalWallet:         "wallet",
			Status:              "partial",
			BaselineStartAt:     now,
			BaselineEndAt:       now,
			ScanStartAt:         now,
			ScanEndAt:           now,
			BaselineComplete:    false,
			IncompleteWindow:    true,
			UnknownGateReason:   "baseline_truncated",
			TransactionsFetched: 10,
			TruncationReason:    "timeout",
			UpdatedAt:           now,
		},
	}}

	s := NewServer(repo, config.Config{})
	req := httptest.NewRequest(http.MethodGet, "/api/wallet-sync", nil)
	res := httptest.NewRecorder()
	s.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	var body struct {
		Items []struct {
			RunID             int64  `json:"runId"`
			IncompleteWindow  bool   `json:"incompleteWindow"`
			UnknownGateReason string `json:"unknownGateReason"`
		} `json:"items"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(body.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(body.Items))
	}
	if body.Items[0].RunID != 9 {
		t.Fatalf("expected runId=9, got %d", body.Items[0].RunID)
	}
	if body.Items[0].UnknownGateReason != "baseline_truncated" {
		t.Fatalf("expected preserved unknownGateReason, got %q", body.Items[0].UnknownGateReason)
	}
}

func TestOpsWalletSyncAPI_MirrorsWalletSyncPayload(t *testing.T) {
	now := time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)
	repo := fakeRepo{walletSync: []storage.WalletSyncListRecord{
		{
			WalletSyncRunID:     1,
			IngestionRunID:      9,
			FocalWallet:         "wallet",
			Status:              "partial",
			BaselineStartAt:     now,
			BaselineEndAt:       now,
			ScanStartAt:         now,
			ScanEndAt:           now,
			BaselineComplete:    false,
			IncompleteWindow:    true,
			UnknownGateReason:   "unknown_required_gates:zero_or_dust",
			TransactionsFetched: 10,
			TruncationReason:    "timeout",
			UpdatedAt:           now,
		},
	}}

	s := NewServer(repo, config.Config{})
	req := httptest.NewRequest(http.MethodGet, "/api/ops/wallet-sync", nil)
	res := httptest.NewRecorder()
	s.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	var body struct {
		Items []struct {
			RunID int64 `json:"runId"`
		} `json:"items"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].RunID != 9 {
		t.Fatalf("unexpected ops wallet sync payload: %+v", body.Items)
	}
}

func TestReportsCandidatesRequiresRunID(t *testing.T) {
	s := NewServer(fakeRepo{}, config.Config{})
	req := httptest.NewRequest(http.MethodGet, "/api/reports/candidates", nil)
	res := httptest.NewRecorder()
	s.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestCandidateDetailNotFound(t *testing.T) {
	s := NewServer(fakeRepo{}, config.Config{})
	req := httptest.NewRequest(http.MethodGet, "/api/candidates/1/sig/0", nil)
	res := httptest.NewRecorder()
	s.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
}

func TestExportGenerateNotConfigured(t *testing.T) {
	s := NewServer(fakeRepo{}, config.Config{})
	req := httptest.NewRequest(http.MethodGet, "/api/exports/generate?run_id=42", nil)
	res := httptest.NewRecorder()
	s.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", res.Code)
	}
}

func TestExportFilesListsGeneratedArtifacts(t *testing.T) {
	s := NewServer(fakeRepo{}, config.Config{})
	tmp := t.TempDir()
	s.exportRoot = tmp
	runDir := filepath.Join(tmp, "run_42")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "report_manifest.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/exports/files?run_id=42", nil)
	res := httptest.NewRecorder()
	s.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	var body struct {
		RunID int64 `json:"runId"`
		Files []struct {
			Name string `json:"name"`
		} `json:"files"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.RunID != 42 {
		t.Fatalf("expected runId=42, got %d", body.RunID)
	}
	if len(body.Files) != 1 || body.Files[0].Name != "report_manifest.json" {
		t.Fatalf("unexpected files payload: %+v", body.Files)
	}
}
