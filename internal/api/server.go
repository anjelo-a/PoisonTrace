package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"poisontrace/internal/config"
	exportspkg "poisontrace/internal/exports"
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
	GetCandidateExplanation(ctx context.Context, walletSyncRunID int64, signature string, transferIndex int) (storage.CandidateExplanationRecord, bool, error)
	ListCandidateExplanationsForRun(ctx context.Context, runID int64, limit, offset int) ([]storage.CandidateExplanationRecord, int, error)
	ListWalletInspectionSummaryForRun(ctx context.Context, runID int64, limit, offset int) ([]storage.WalletInspectionSummaryRecord, int, error)
}

type Server struct {
	repo         ReadRepository
	cfg          config.Config
	exportSource exportspkg.DatasetSource
	exportRoot   string
}

func NewServer(repo ReadRepository, cfg config.Config) *Server {
	s := &Server{
		repo:       repo,
		cfg:        cfg,
		exportRoot: filepath.Join("artifacts", "web_exports"),
	}
	if ds, ok := any(repo).(exportspkg.DatasetSource); ok {
		s.exportSource = ds
	}
	return s
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/overview", s.handleOverview)
	mux.HandleFunc("/api/candidates", s.handleCandidates)
	mux.HandleFunc("/api/candidates/", s.handleCandidateDetail)
	mux.HandleFunc("/api/reports/candidates", s.handleCandidateReports)
	mux.HandleFunc("/api/reports/wallets", s.handleWalletReports)
	mux.HandleFunc("/api/exports/generate", s.handleExportGenerate)
	mux.HandleFunc("/api/exports/files", s.handleExportFiles)
	mux.HandleFunc("/api/exports/download", s.handleExportDownload)
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

func (s *Server) handleCandidateDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/candidates/"), "/")
	if len(parts) != 3 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expected path: /api/candidates/:walletSyncRunId/:signature/:transferIndex"})
		return
	}
	walletSyncRunID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || walletSyncRunID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid walletSyncRunId"})
		return
	}
	signature := strings.TrimSpace(parts[1])
	if signature == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid signature"})
		return
	}
	transferIndex, err := strconv.Atoi(parts[2])
	if err != nil || transferIndex < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid transferIndex"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	rec, ok, err := s.repo.GetCandidateExplanation(ctx, walletSyncRunID, signature, transferIndex)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "candidate not found"})
		return
	}
	writeJSON(w, http.StatusOK, candidateExplanationPayload(rec))
}

func (s *Server) handleCandidateReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	runID, ok := parseRunID(w, r)
	if !ok {
		return
	}
	page, pageSize := parsePage(r)
	offset := (page - 1) * pageSize
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	rows, total, err := s.repo.ListCandidateExplanationsForRun(ctx, runID, pageSize, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, rec := range rows {
		items = append(items, candidateExplanationPayload(rec))
	}
	writePaged(w, items, total, page, pageSize)
}

func (s *Server) handleWalletReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	runID, ok := parseRunID(w, r)
	if !ok {
		return
	}
	page, pageSize := parsePage(r)
	offset := (page - 1) * pageSize
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	rows, total, err := s.repo.ListWalletInspectionSummaryForRun(ctx, runID, pageSize, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, rec := range rows {
		items = append(items, map[string]any{
			"runId":                 rec.RunID,
			"walletSyncRunId":       rec.WalletSyncRunID,
			"focalWallet":           rec.FocalWallet,
			"candidateCount":        rec.CandidateCount,
			"unknownGateBlockCount": rec.UnknownGateBlockCount,
			"incompleteWindow":      rec.IncompleteWindow,
			"unknownGateReason":     rec.UnknownGateReason,
			"truncationReason":      rec.TruncationReason,
			"baselineComplete":      rec.BaselineComplete,
			"scanStartAt":           rec.ScanStartAt.UTC().Format(time.RFC3339),
			"scanEndAt":             rec.ScanEndAt.UTC().Format(time.RFC3339),
			"baselineStartAt":       rec.BaselineStartAt.UTC().Format(time.RFC3339),
			"baselineEndAt":         rec.BaselineEndAt.UTC().Format(time.RFC3339),
			"transactionsFetched":   rec.TransactionsFetched,
			"sourceReferences": map[string]any{
				"walletSyncRunId": rec.WalletSyncRunID,
				"runId":           rec.RunID,
			},
		})
	}
	writePaged(w, items, total, page, pageSize)
}

func (s *Server) handleExportGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if s.exportSource == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "export generation is not configured"})
		return
	}
	runID, ok := parseRunID(w, r)
	if !ok {
		return
	}
	outDir := filepath.Join(s.exportRoot, fmt.Sprintf("run_%d", runID))
	filter := storage.ExportFilter{RunID: &runID}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	res, err := exportspkg.ExportDataset(ctx, s.exportSource, exportspkg.ExportOptions{
		OutDir: outDir,
		Filter: filter,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	type fileItem struct {
		Name        string `json:"name"`
		RowCount    int    `json:"rowCount"`
		SHA256      string `json:"sha256"`
		DownloadURL string `json:"downloadUrl"`
	}
	files := make([]fileItem, 0, len(res.Manifest.Files))
	for _, f := range res.Manifest.Files {
		files = append(files, fileItem{
			Name:        f.Name,
			RowCount:    f.RowCount,
			SHA256:      f.SHA256,
			DownloadURL: fmt.Sprintf("/api/exports/download?run_id=%d&file=%s", runID, f.Name),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"runId":         runID,
		"outDir":        outDir,
		"schemaVersion": res.Manifest.SchemaVersion,
		"generatedAt":   res.Manifest.GeneratedAt,
		"files":         files,
	})
}

func (s *Server) handleExportFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	runID, ok := parseRunID(w, r)
	if !ok {
		return
	}
	outDir := filepath.Join(s.exportRoot, fmt.Sprintf("run_%d", runID))
	entries, err := os.ReadDir(outDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeJSON(w, http.StatusOK, map[string]any{
				"runId": runID,
				"files": []any{},
			})
			return
		}
		writeError(w, err)
		return
	}
	type fileItem struct {
		Name        string `json:"name"`
		SizeBytes   int64  `json:"sizeBytes"`
		ModifiedAt  string `json:"modifiedAt"`
		DownloadURL string `json:"downloadUrl"`
	}
	files := make([]fileItem, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		info, infoErr := e.Info()
		if infoErr != nil {
			continue
		}
		files = append(files, fileItem{
			Name:        name,
			SizeBytes:   info.Size(),
			ModifiedAt:  info.ModTime().UTC().Format(time.RFC3339),
			DownloadURL: fmt.Sprintf("/api/exports/download?run_id=%d&file=%s", runID, name),
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	writeJSON(w, http.StatusOK, map[string]any{
		"runId": runID,
		"files": files,
	})
}

func (s *Server) handleExportDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	runID, ok := parseRunID(w, r)
	if !ok {
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("file"))
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file is required"})
		return
	}
	base := filepath.Base(name)
	if base != name {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid file"})
		return
	}
	path := filepath.Join(s.exportRoot, fmt.Sprintf("run_%d", runID), base)
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "file not found"})
			return
		}
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", base))
	http.ServeFile(w, r, path)
}

func parseRunID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("run_id"))
	if raw == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "run_id is required"})
		return 0, false
	}
	runID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || runID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid run_id"})
		return 0, false
	}
	return runID, true
}

func candidateExplanationPayload(rec storage.CandidateExplanationRecord) map[string]any {
	return map[string]any{
		"walletSyncRunId":          rec.WalletSyncRunID,
		"runId":                    rec.IngestionRunID,
		"focalWallet":              rec.FocalWallet,
		"signature":                rec.Signature,
		"transferIndex":            rec.TransferIndex,
		"blockTime":                rec.BlockTime.UTC().Format(time.RFC3339),
		"suspiciousCounterparty":   rec.SuspiciousCounterparty,
		"matchedLegitCounterparty": rec.MatchedLegitCounterparty,
		"relationType":             rec.RelationType,
		"assetType":                rec.AssetType,
		"normalizationStatus":      rec.NormalizationStatus,
		"poisoningEligible":        rec.PoisoningEligible,
		"sourceOwner":              rec.SourceOwner,
		"destinationOwner":         rec.DestinationOwner,
		"fromTokenAccount":         rec.FromTokenAccount,
		"toTokenAccount":           rec.ToTokenAccount,
		"tokenMint":                rec.TokenMint,
		"amountRaw":                rec.AmountRaw,
		"dustStatus":               rec.DustStatus,
		"isDust":                   rec.IsDust,
		"isZeroValue":              rec.IsZeroValue,
		"isInbound":                rec.IsInbound,
		"isNewCounterparty":        rec.IsNewCounterparty,
		"recencyDays":              rec.RecencyDays,
		"repeatInjectionCount":     rec.RepeatInjectionCount,
		"lookalikePrefixMatch":     rec.LookalikePrefixMatch,
		"lookalikeSuffixMatch":     rec.LookalikeSuffixMatch,
		"matchRuleVersion":         rec.MatchRuleVersion,
		"legitLastSeenAt":          rec.LegitLastSeenAt.UTC().Format(time.RFC3339),
		"baselineComplete":         rec.BaselineComplete,
		"incompleteWindow":         rec.IncompleteWindow,
		"unknownGateReason":        rec.UnknownGateReason,
		"scanStartAt":              rec.ScanStartAt.UTC().Format(time.RFC3339),
		"scanEndAt":                rec.ScanEndAt.UTC().Format(time.RFC3339),
		"baselineStartAt":          rec.BaselineStartAt.UTC().Format(time.RFC3339),
		"baselineEndAt":            rec.BaselineEndAt.UTC().Format(time.RFC3339),
		"sourceReferences": map[string]any{
			"walletSyncRunId":     rec.WalletSyncRunID,
			"runId":               rec.IngestionRunID,
			"transactionId":       rec.TransactionID,
			"walletTransactionId": rec.WalletTransactionID,
			"counterpartyId":      rec.CounterpartyID,
		},
	}
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
