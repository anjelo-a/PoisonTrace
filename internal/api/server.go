package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"poisontrace/internal/config"
	"poisontrace/internal/storage"
)

type ReadRepository interface {
	GetOverviewMetrics(ctx context.Context, since time.Time) (storage.OverviewMetricsRecord, error)
	ListOverviewCandidates(ctx context.Context, limit int) ([]storage.OverviewCandidateRecord, error)
	ListCandidates(ctx context.Context, limit, offset int) ([]storage.CandidateListRecord, int, error)
	ListTransactions(ctx context.Context, limit, offset int) ([]storage.TransactionListRecord, int, error)
	ListRuns(ctx context.Context, limit, offset int) ([]storage.IngestionRunListRecord, int, error)
	ListWalletSyncRuns(ctx context.Context, limit, offset int) ([]storage.WalletSyncListRecord, int, error)
	ListCounterparties(ctx context.Context, limit, offset int) ([]storage.CounterpartyListRecord, int, error)
}

type Server struct {
	repo ReadRepository
	cfg  config.Config
}

func NewServer(repo ReadRepository, cfg config.Config) *Server {
	return &Server{repo: repo, cfg: cfg}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/overview", s.handleOverview)
	mux.HandleFunc("/api/candidates", s.handleCandidates)
	mux.HandleFunc("/api/transactions", s.handleTransactions)
	mux.HandleFunc("/api/runs", s.handleRuns)
	mux.HandleFunc("/api/wallet-sync", s.handleWalletSync)
	mux.HandleFunc("/api/counterparties", s.handleCounterparties)
	mux.HandleFunc("/api/settings", s.handleSettings)
	mux.HandleFunc("/healthz", s.handleHealth)
	return withCORS(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	since := time.Now().Add(-time.Duration(s.cfg.ScanWindowDays) * 24 * time.Hour)
	metrics, err := s.repo.GetOverviewMetrics(ctx, since)
	if err != nil {
		writeError(w, err)
		return
	}
	recent, err := s.repo.ListOverviewCandidates(ctx, 5)
	if err != nil {
		writeError(w, err)
		return
	}

	lastScan := ""
	if metrics.LastScanUpdateAt != nil {
		lastScan = metrics.LastScanUpdateAt.UTC().Format(time.RFC3339)
	}

	passRate := 0.0
	if metrics.TransactionsScanned > 0 {
		passRate = float64(metrics.PassedTransactions) / float64(metrics.TransactionsScanned) * 100
	}

	type recentItem struct {
		WalletSyncRunID        int64  `json:"walletSyncRunId"`
		FocalWallet            string `json:"focalWallet"`
		Signature              string `json:"signature"`
		TransferIndex          int    `json:"transferIndex"`
		BlockTime              string `json:"blockTime"`
		SuspiciousCounterparty string `json:"suspiciousCounterparty"`
		RepeatInjectionCount   int    `json:"repeatInjectionCount"`
		RecencyDays            int    `json:"recencyDays"`
	}

	items := make([]recentItem, 0, len(recent))
	for _, rec := range recent {
		items = append(items, recentItem{
			WalletSyncRunID:        rec.WalletSyncRunID,
			FocalWallet:            rec.FocalWallet,
			Signature:              rec.Signature,
			TransferIndex:          rec.TransferIndex,
			BlockTime:              rec.BlockTime.UTC().Format(time.RFC3339),
			SuspiciousCounterparty: rec.SuspiciousCounterparty,
			RepeatInjectionCount:   rec.RepeatInjectionCount,
			RecencyDays:            rec.RecencyDays,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"metrics": map[string]any{
			"candidatesEmitted":   metrics.CandidatesEmitted,
			"unknownGateBlocks":   metrics.UnknownGateBlocks,
			"transactionsScanned": metrics.TransactionsScanned,
			"passedTransactions":  metrics.PassedTransactions,
			"lastScanUpdateAt":    lastScan,
			"scanWindowLabel":     strconv.Itoa(s.cfg.ScanWindowDays) + "d",
			"passRatePct":         passRate,
		},
		"recentCandidates": items,
	})
}

func (s *Server) handleCandidates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	page, pageSize := parsePage(r)
	offset := (page - 1) * pageSize
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, total, err := s.repo.ListCandidates(ctx, pageSize, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, rec := range rows {
		items = append(items, map[string]any{
			"walletSyncRunId":          rec.WalletSyncRunID,
			"focalWallet":              rec.FocalWallet,
			"signature":                rec.Signature,
			"transferIndex":            rec.TransferIndex,
			"blockTime":                rec.BlockTime.UTC().Format(time.RFC3339),
			"suspiciousCounterparty":   rec.SuspiciousCounterparty,
			"matchedLegitCounterparty": rec.MatchedLegitCounterparty,
			"repeatInjectionCount":     rec.RepeatInjectionCount,
			"recencyDays":              rec.RecencyDays,
			"unknownGateReason":        rec.UnknownGateReason,
			"incompleteWindow":         rec.IncompleteWindow,
		})
	}
	writePaged(w, items, total, page, pageSize)
}

func (s *Server) handleTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	page, pageSize := parsePage(r)
	offset := (page - 1) * pageSize
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, total, err := s.repo.ListTransactions(ctx, pageSize, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, rec := range rows {
		items = append(items, map[string]any{
			"focalWallet":         rec.FocalWallet,
			"signature":           rec.Signature,
			"transferIndex":       rec.TransferIndex,
			"blockTime":           rec.BlockTime.UTC().Format(time.RFC3339),
			"normalizationStatus": rec.NormalizationStatus,
			"poisoningEligible":   rec.PoisoningEligible,
			"relationType":        rec.RelationType,
			"dustStatus":          rec.DustStatus,
			"amountRaw":           rec.AmountRaw,
			"assetType":           rec.AssetType,
		})
	}
	writePaged(w, items, total, page, pageSize)
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	page, pageSize := parsePage(r)
	offset := (page - 1) * pageSize
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, total, err := s.repo.ListRuns(ctx, pageSize, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, rec := range rows {
		completed := ""
		if rec.CompletedAt != nil {
			completed = rec.CompletedAt.UTC().Format(time.RFC3339)
		}
		items = append(items, map[string]any{
			"runId":                rec.ID,
			"status":               rec.Status,
			"startedAt":            rec.StartedAt.UTC().Format(time.RFC3339),
			"completedAt":          completed,
			"walletsProcessed":     rec.WalletsProcessed,
			"walletsFailed":        rec.WalletsFailed,
			"walletsSkipped":       rec.WalletsSkipped,
			"incompleteWindows":    rec.IncompleteWindows,
			"truncationWalletRate": rec.TruncationWalletRate,
			"unknownGatePresent":   rec.UnknownGatePresent,
		})
	}
	writePaged(w, items, total, page, pageSize)
}

func (s *Server) handleWalletSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	page, pageSize := parsePage(r)
	offset := (page - 1) * pageSize
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, total, err := s.repo.ListWalletSyncRuns(ctx, pageSize, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, rec := range rows {
		if rec.IncompleteWindow && strings.TrimSpace(rec.UnknownGateReason) == "" {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "wallet sync row missing unknown_gate_reason while incomplete_window=true",
			})
			return
		}
		items = append(items, map[string]any{
			"walletSyncRunId":     rec.WalletSyncRunID,
			"runId":               rec.IngestionRunID,
			"focalWallet":         rec.FocalWallet,
			"status":              rec.Status,
			"baselineStartAt":     rec.BaselineStartAt.UTC().Format(time.RFC3339),
			"baselineEndAt":       rec.BaselineEndAt.UTC().Format(time.RFC3339),
			"scanStartAt":         rec.ScanStartAt.UTC().Format(time.RFC3339),
			"scanEndAt":           rec.ScanEndAt.UTC().Format(time.RFC3339),
			"baselineComplete":    rec.BaselineComplete,
			"incompleteWindow":    rec.IncompleteWindow,
			"unknownGateReason":   rec.UnknownGateReason,
			"transactionsFetched": rec.TransactionsFetched,
			"truncationReason":    rec.TruncationReason,
			"updatedAt":           rec.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	writePaged(w, items, total, page, pageSize)
}

func (s *Server) handleCounterparties(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	page, pageSize := parsePage(r)
	offset := (page - 1) * pageSize
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, total, err := s.repo.ListCounterparties(ctx, pageSize, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, rec := range rows {
		lastOut := ""
		if rec.LastOutboundAt != nil {
			lastOut = rec.LastOutboundAt.UTC().Format(time.RFC3339)
		}
		items = append(items, map[string]any{
			"focalWallet":         rec.FocalWallet,
			"counterpartyAddress": rec.CounterpartyAddress,
			"firstSeenAt":         rec.FirstSeenAt.UTC().Format(time.RFC3339),
			"lastSeenAt":          rec.LastSeenAt.UTC().Format(time.RFC3339),
			"inboundCount":        rec.InboundCount,
			"outboundCount":       rec.OutboundCount,
			"lastOutboundAt":      lastOut,
			"candidateLinks":      rec.CandidateLinks,
		})
	}
	writePaged(w, items, total, page, pageSize)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"maxWalletsPerRun":         s.cfg.MaxWalletsPerRun,
		"maxTXPagesPerWallet":      s.cfg.MaxTXPagesPerWallet,
		"maxTXPerWallet":           s.cfg.MaxTXPerWallet,
		"maxConcurrentWallets":     s.cfg.MaxConcurrentWallets,
		"walletSyncTimeoutSeconds": s.cfg.WalletSyncTimeoutSeconds,
		"runTimeoutSeconds":        s.cfg.RunTimeoutSeconds,
		"maxHeliusRetries":         s.cfg.MaxHeliusRetries,
		"heliusRequestDelayMS":     s.cfg.HeliusRequestDelayMS,
		"baselineLookbackDays":     s.cfg.BaselineLookbackDays,
		"scanWindowDays":           s.cfg.ScanWindowDays,
		"lookalikeRecencyDays":     s.cfg.LookalikeRecencyDays,
		"lookalikePrefixMin":       s.cfg.LookalikePrefixMin,
		"lookalikeSuffixMin":       s.cfg.LookalikeSuffixMin,
		"lookalikeSingleSideMin":   s.cfg.LookalikeSingleSideMin,
		"minInjectionCount":        s.cfg.MinInjectionCount,
	})
}

func parsePage(r *http.Request) (int, int) {
	page := parseIntDefault(r.URL.Query().Get("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := parseIntDefault(r.URL.Query().Get("page_size"), 50)
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 500 {
		pageSize = 500
	}
	return page, pageSize
}

func parseIntDefault(raw string, fallback int) int {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return v
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

func writeError(w http.ResponseWriter, err error) {
	if errors.Is(err, context.DeadlineExceeded) {
		writeJSON(w, http.StatusGatewayTimeout, map[string]string{"error": "request timeout"})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}

func writePaged(w http.ResponseWriter, items any, total, page, pageSize int) {
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "page": page, "pageSize": pageSize})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
