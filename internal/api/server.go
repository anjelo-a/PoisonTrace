package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

type SettingsStore interface {
	GetConfigOverride(ctx context.Context) (storage.ConfigOverrideRecord, bool, error)
	UpsertConfigOverride(ctx context.Context, rec storage.ConfigOverrideRecord) error
}

type ExportJobStore interface {
	CreateExportJob(ctx context.Context, runID int64, outDir string) (int64, error)
	UpdateExportJobStatus(ctx context.Context, jobID int64, status string, errMsg *string, startedAt, completedAt *time.Time) error
	GetExportJob(ctx context.Context, jobID int64) (storage.ExportJobRecord, bool, error)
	ListExportJobsForRun(ctx context.Context, runID int64, limit int) ([]storage.ExportJobRecord, error)
}

type OpsStore interface {
	ListOpsRunHealth(ctx context.Context, limit int, offset int) ([]storage.OpsRunHealthRecord, int, error)
	ListFailureClassCounts(ctx context.Context) ([]storage.FailureClassCountRecord, error)
}

type Server struct {
	repo         ReadRepository
	cfg          config.Config
	exportSource exportspkg.DatasetSource
	exportRoot   string
	settings     SettingsStore
	exportJobs   ExportJobStore
	ops          OpsStore
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
	if ss, ok := any(repo).(SettingsStore); ok {
		s.settings = ss
	}
	if ej, ok := any(repo).(ExportJobStore); ok {
		s.exportJobs = ej
	}
	if ops, ok := any(repo).(OpsStore); ok {
		s.ops = ops
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
	mux.HandleFunc("/api/exports/jobs", s.handleExportJobs)
	mux.HandleFunc("/api/exports/jobs/", s.handleExportJobByID)
	mux.HandleFunc("/api/exports/files", s.handleExportFiles)
	mux.HandleFunc("/api/exports/download", s.handleExportDownload)
	mux.HandleFunc("/api/ops/runs", s.handleOpsRuns)
	mux.HandleFunc("/api/ops/failures", s.handleOpsFailures)
	mux.HandleFunc("/api/transactions", s.handleTransactions)
	mux.HandleFunc("/api/runs", s.handleRuns)
	mux.HandleFunc("/api/wallet-sync", s.handleWalletSync)
	mux.HandleFunc("/api/counterparties", s.handleCounterparties)
	mux.HandleFunc("/api/settings", s.handleSettings)
	mux.HandleFunc("/healthz", s.handleHealth)
	return withCORS(withAuth(mux, s.cfg.APIBearerToken))
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
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "wallet sync row missing unknown_gate_reason while incomplete_window=true"})
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
	switch r.Method {
	case http.MethodGet:
		s.handleSettingsGet(w, r)
	case http.MethodPut, http.MethodPatch:
		s.handleSettingsWrite(w, r)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleSettingsGet(w http.ResponseWriter, r *http.Request) {
	override, _, err := s.loadSettingsOverride(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"maxWalletsPerRun":         withOverrideInt(s.cfg.MaxWalletsPerRun, override.MaxWalletsPerRun),
		"maxTXPagesPerWallet":      withOverrideInt(s.cfg.MaxTXPagesPerWallet, override.MaxTXPagesPerWallet),
		"maxTXPerWallet":           withOverrideInt(s.cfg.MaxTXPerWallet, override.MaxTXPerWallet),
		"maxConcurrentWallets":     withOverrideInt(s.cfg.MaxConcurrentWallets, override.MaxConcurrentWallets),
		"walletSyncTimeoutSeconds": withOverrideInt(s.cfg.WalletSyncTimeoutSeconds, override.WalletSyncTimeoutSeconds),
		"runTimeoutSeconds":        withOverrideInt(s.cfg.RunTimeoutSeconds, override.RunTimeoutSeconds),
		"maxHeliusRetries":         withOverrideInt(s.cfg.MaxHeliusRetries, override.MaxHeliusRetries),
		"heliusRequestDelayMS":     withOverrideInt(s.cfg.HeliusRequestDelayMS, override.HeliusRequestDelayMS),
		"baselineLookbackDays":     withOverrideInt(s.cfg.BaselineLookbackDays, override.BaselineLookbackDays),
		"scanWindowDays":           withOverrideInt(s.cfg.ScanWindowDays, override.ScanWindowDays),
		"lookalikeRecencyDays":     withOverrideInt(s.cfg.LookalikeRecencyDays, override.LookalikeRecencyDays),
		"lookalikePrefixMin":       withOverrideInt(s.cfg.LookalikePrefixMin, override.LookalikePrefixMin),
		"lookalikeSuffixMin":       withOverrideInt(s.cfg.LookalikeSuffixMin, override.LookalikeSuffixMin),
		"lookalikeSingleSideMin":   withOverrideInt(s.cfg.LookalikeSingleSideMin, override.LookalikeSingleSideMin),
		"minInjectionCount":        withOverrideInt(s.cfg.MinInjectionCount, override.MinInjectionCount),
	})
}

func (s *Server) handleSettingsWrite(w http.ResponseWriter, r *http.Request) {
	if s.settings == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "settings write is not configured"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	type payload struct {
		MaxWalletsPerRun         *int `json:"maxWalletsPerRun"`
		MaxTXPagesPerWallet      *int `json:"maxTXPagesPerWallet"`
		MaxTXPerWallet           *int `json:"maxTXPerWallet"`
		MaxConcurrentWallets     *int `json:"maxConcurrentWallets"`
		WalletSyncTimeoutSeconds *int `json:"walletSyncTimeoutSeconds"`
		RunTimeoutSeconds        *int `json:"runTimeoutSeconds"`
		MaxHeliusRetries         *int `json:"maxHeliusRetries"`
		HeliusRequestDelayMS     *int `json:"heliusRequestDelayMS"`
		BaselineLookbackDays     *int `json:"baselineLookbackDays"`
		ScanWindowDays           *int `json:"scanWindowDays"`
		LookalikeRecencyDays     *int `json:"lookalikeRecencyDays"`
		LookalikePrefixMin       *int `json:"lookalikePrefixMin"`
		LookalikeSuffixMin       *int `json:"lookalikeSuffixMin"`
		LookalikeSingleSideMin   *int `json:"lookalikeSingleSideMin"`
		MinInjectionCount        *int `json:"minInjectionCount"`
	}
	var in payload
	if err := json.Unmarshal(body, &in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	rec := storage.ConfigOverrideRecord{
		MaxWalletsPerRun:         in.MaxWalletsPerRun,
		MaxTXPagesPerWallet:      in.MaxTXPagesPerWallet,
		MaxTXPerWallet:           in.MaxTXPerWallet,
		MaxConcurrentWallets:     in.MaxConcurrentWallets,
		WalletSyncTimeoutSeconds: in.WalletSyncTimeoutSeconds,
		RunTimeoutSeconds:        in.RunTimeoutSeconds,
		MaxHeliusRetries:         in.MaxHeliusRetries,
		HeliusRequestDelayMS:     in.HeliusRequestDelayMS,
		BaselineLookbackDays:     in.BaselineLookbackDays,
		ScanWindowDays:           in.ScanWindowDays,
		LookalikeRecencyDays:     in.LookalikeRecencyDays,
		LookalikePrefixMin:       in.LookalikePrefixMin,
		LookalikeSuffixMin:       in.LookalikeSuffixMin,
		LookalikeSingleSideMin:   in.LookalikeSingleSideMin,
		MinInjectionCount:        in.MinInjectionCount,
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := s.settings.UpsertConfigOverride(ctx, rec); err != nil {
		writeError(w, err)
		return
	}
	s.handleSettingsGet(w, r)
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
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
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
	res, err := exportspkg.ExportDataset(ctx, s.exportSource, exportspkg.ExportOptions{OutDir: outDir, Filter: filter})
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
		files = append(files, fileItem{Name: f.Name, RowCount: f.RowCount, SHA256: f.SHA256, DownloadURL: fmt.Sprintf("/api/exports/download?run_id=%d&file=%s", runID, f.Name)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"runId": runID, "outDir": outDir, "schemaVersion": res.Manifest.SchemaVersion, "generatedAt": res.Manifest.GeneratedAt, "files": files})
}

func (s *Server) handleExportJobs(w http.ResponseWriter, r *http.Request) {
	if s.exportJobs == nil || s.exportSource == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "export jobs are not configured"})
		return
	}
	switch r.Method {
	case http.MethodPost:
		runID, ok := parseRunID(w, r)
		if !ok {
			return
		}
		outDir := filepath.Join(s.exportRoot, fmt.Sprintf("run_%d", runID))
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		jobID, err := s.exportJobs.CreateExportJob(ctx, runID, outDir)
		if err != nil {
			writeError(w, err)
			return
		}
		go s.runExportJob(jobID, runID, outDir)
		writeJSON(w, http.StatusAccepted, map[string]any{"jobId": jobID, "runId": runID, "status": "queued"})
	case http.MethodGet:
		runID, ok := parseRunID(w, r)
		if !ok {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		rows, err := s.exportJobs.ListExportJobsForRun(ctx, runID, 50)
		if err != nil {
			writeError(w, err)
			return
		}
		items := make([]map[string]any, 0, len(rows))
		for _, rec := range rows {
			items = append(items, exportJobPayload(rec))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleExportJobByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if s.exportJobs == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "export jobs are not configured"})
		return
	}
	idRaw := strings.TrimPrefix(r.URL.Path, "/api/exports/jobs/")
	jobID, err := strconv.ParseInt(strings.TrimSpace(idRaw), 10, 64)
	if err != nil || jobID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid job id"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	rec, ok, err := s.exportJobs.GetExportJob(ctx, jobID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
		return
	}
	writeJSON(w, http.StatusOK, exportJobPayload(rec))
}

func (s *Server) runExportJob(jobID int64, runID int64, outDir string) {
	now := time.Now().UTC()
	_ = s.exportJobs.UpdateExportJobStatus(context.Background(), jobID, "running", nil, &now, nil)
	filter := storage.ExportFilter{RunID: &runID}
	_, err := exportspkg.ExportDataset(context.Background(), s.exportSource, exportspkg.ExportOptions{OutDir: outDir, Filter: filter})
	done := time.Now().UTC()
	if err != nil {
		msg := err.Error()
		_ = s.exportJobs.UpdateExportJobStatus(context.Background(), jobID, "failed", &msg, nil, &done)
		return
	}
	_ = s.exportJobs.UpdateExportJobStatus(context.Background(), jobID, "succeeded", nil, nil, &done)
}

func exportJobPayload(rec storage.ExportJobRecord) map[string]any {
	started := ""
	if rec.StartedAt != nil {
		started = rec.StartedAt.UTC().Format(time.RFC3339)
	}
	completed := ""
	if rec.CompletedAt != nil {
		completed = rec.CompletedAt.UTC().Format(time.RFC3339)
	}
	return map[string]any{
		"jobId":        rec.ID,
		"runId":        rec.RunID,
		"status":       rec.Status,
		"outDir":       rec.OutDir,
		"errorMessage": rec.ErrorMessage,
		"createdAt":    rec.CreatedAt.UTC().Format(time.RFC3339),
		"startedAt":    started,
		"completedAt":  completed,
	}
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
			writeJSON(w, http.StatusOK, map[string]any{"runId": runID, "files": []any{}})
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
		info, infoErr := e.Info()
		if infoErr != nil {
			continue
		}
		name := e.Name()
		files = append(files, fileItem{Name: name, SizeBytes: info.Size(), ModifiedAt: info.ModTime().UTC().Format(time.RFC3339), DownloadURL: fmt.Sprintf("/api/exports/download?run_id=%d&file=%s", runID, name)})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	writeJSON(w, http.StatusOK, map[string]any{"runId": runID, "files": files})
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

func (s *Server) handleOpsRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if s.ops == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "ops read model is not configured"})
		return
	}
	page, pageSize := parsePage(r)
	offset := (page - 1) * pageSize
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	rows, total, err := s.ops.ListOpsRunHealth(ctx, pageSize, offset)
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
			"runId":                rec.RunID,
			"status":               rec.Status,
			"startedAt":            rec.StartedAt.UTC().Format(time.RFC3339),
			"completedAt":          completed,
			"walletsRequested":     rec.WalletsRequested,
			"walletsProcessed":     rec.WalletsProcessed,
			"walletsFailed":        rec.WalletsFailed,
			"walletsSkipped":       rec.WalletsSkipped,
			"truncationWalletRate": rec.TruncationWalletRate,
			"retryExhaustedCount":  rec.RetryExhaustedCount,
		})
	}
	writePaged(w, items, total, page, pageSize)
}

func (s *Server) handleOpsFailures(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if s.ops == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "ops read model is not configured"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	rows, err := s.ops.ListFailureClassCounts(ctx)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, rec := range rows {
		items = append(items, map[string]any{"failureClass": rec.FailureClass, "count": rec.Count})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
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

func (s *Server) loadSettingsOverride(ctx context.Context) (storage.ConfigOverrideRecord, bool, error) {
	if s.settings == nil {
		return storage.ConfigOverrideRecord{}, false, nil
	}
	c, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.settings.GetConfigOverride(c)
}

func withOverrideInt(base int, override *int) int {
	if override == nil {
		return base
	}
	return *override
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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withAuth(next http.Handler, bearerToken string) http.Handler {
	token := strings.TrimSpace(bearerToken)
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) || strings.TrimSpace(strings.TrimPrefix(auth, prefix)) != token {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
