package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
