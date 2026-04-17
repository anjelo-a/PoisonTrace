package fixtures

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"poisontrace/internal/helius"
	"poisontrace/internal/pipeline"
	"poisontrace/internal/transactions"
)

const (
	defaultMaxTXPagesPerWallet    = 20
	defaultMaxTXPerWallet         = 1500
	defaultMaxHeliusRetries       = 1
	defaultLookalikeRecencyDays   = 30
	defaultLookalikePrefixMin     = 4
	defaultLookalikeSuffixMin     = 4
	defaultLookalikeSingleSideMin = 6
	defaultMinInjectionCount      = 2
)

var CanonicalCaseIDs = []string{
	"baseline_truncated_newness_unknown",
	"spl_unresolved_owner_non_poisoning_ready",
	"missing_threshold_dust_unknown",
	"same_signature_multiple_wallets",
	"repeat_inbound_two_injections_pass",
	"single_injection_fail_min_count",
	"lookalike_prefix_only_pass",
	"lookalike_suffix_only_pass",
	"legit_baseline_outbound_non_dust_only",
	"rate_limited_then_retry_success",
	"wallet_timeout_partial",
	"max_tx_cap_truncated",
	"duplicate_event_across_pages",
	"out_of_order_events_same_signature",
	"scan_boundary_exact_timestamp",
	"partial_owner_present",
	"self_transfer_owner_level",
	"two_injection_gate_with_unknown_second",
	"multi_legit_match_tiebreak",
}

type Meta struct {
	CaseID                 string            `json:"case_id"`
	Description            string            `json:"description"`
	FocalWallet            string            `json:"focal_wallet,omitempty"`
	FocalWallets           []string          `json:"focal_wallets,omitempty"`
	BaselineStart          time.Time         `json:"baseline_start"`
	BaselineEnd            time.Time         `json:"baseline_end"`
	ScanStart              time.Time         `json:"scan_start"`
	ScanEnd                time.Time         `json:"scan_end"`
	ExpectedInScope        bool              `json:"expected_in_scope"`
	ExpectedMissReason     string            `json:"expected_miss_reason,omitempty"`
	MaxTXPagesPerWallet    int               `json:"max_tx_pages_per_wallet,omitempty"`
	MaxTXPerWallet         int               `json:"max_tx_per_wallet,omitempty"`
	MaxHeliusRetries       int               `json:"max_helius_retries,omitempty"`
	HeliusRequestDelayMS   int               `json:"helius_request_delay_ms,omitempty"`
	LookalikeRecencyDays   int               `json:"lookalike_recency_days,omitempty"`
	LookalikePrefixMin     int               `json:"lookalike_prefix_min,omitempty"`
	LookalikeSuffixMin     int               `json:"lookalike_suffix_min,omitempty"`
	LookalikeSingleSideMin int               `json:"lookalike_single_side_min,omitempty"`
	MinInjectionCount      int               `json:"min_injection_count,omitempty"`
	TimeoutMS              int               `json:"timeout_ms,omitempty"`
	DustThresholds         []DustThreshold   `json:"dust_thresholds,omitempty"`
	FetchScript            []FetchScriptStep `json:"fetch_script,omitempty"`
}

type DustThreshold struct {
	AssetKey   string     `json:"asset_key"`
	AmountRaw  string     `json:"amount_raw"`
	ActiveFrom time.Time  `json:"active_from"`
	ActiveTo   *time.Time `json:"active_to,omitempty"`
}

type FetchScriptStep struct {
	Kind       string `json:"kind,omitempty"`
	File       string `json:"file,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
	Message    string `json:"message,omitempty"`
	SleepMS    int    `json:"sleep_ms,omitempty"`
}

type FixtureCase struct {
	RootDir string
	CaseDir string
	Meta    Meta
	pages   map[string]helius.EnhancedPage
}

type NormalizedTransferRecord struct {
	Signature               string `json:"signature"`
	TransferIndex           int    `json:"transfer_index"`
	TransferFingerprint     string `json:"transfer_fingerprint"`
	Slot                    int64  `json:"slot"`
	BlockTime               string `json:"block_time"`
	SourceOwnerAddress      string `json:"source_owner_address"`
	DestinationOwnerAddress string `json:"destination_owner_address"`
	SourceTokenAccount      string `json:"source_token_account"`
	DestinationTokenAccount string `json:"destination_token_account"`
	AssetType               string `json:"asset_type"`
	AssetKey                string `json:"asset_key"`
	TokenMint               string `json:"token_mint"`
	AmountRaw               string `json:"amount_raw"`
	Decimals                *int   `json:"decimals"`
	NormalizationStatus     string `json:"normalization_status"`
	NormalizationReasonCode string `json:"normalization_reason_code"`
	PoisoningEligible       bool   `json:"poisoning_eligible"`
	DustStatus              string `json:"dust_status"`
	IsSuccess               bool   `json:"is_success"`
}

type WalletTransactionRecord struct {
	FocalWallet         string `json:"focal_wallet"`
	RelationType        string `json:"relation_type"`
	CounterpartyAddress string `json:"counterparty_address"`
	Signature           string `json:"signature"`
	TransferIndex       int    `json:"transfer_index"`
	TransferFingerprint string `json:"transfer_fingerprint"`
}

type CounterpartyRecord struct {
	FocalWallet         string `json:"focal_wallet"`
	CounterpartyAddress string `json:"counterparty_address"`
	FirstSeenAt         string `json:"first_seen_at"`
	LastSeenAt          string `json:"last_seen_at"`
	InteractionCount    int64  `json:"interaction_count"`
	FirstInboundAt      string `json:"first_inbound_at"`
	LastInboundAt       string `json:"last_inbound_at"`
	InboundCount        int64  `json:"inbound_count"`
	FirstOutboundAt     string `json:"first_outbound_at"`
	LastOutboundAt      string `json:"last_outbound_at"`
	OutboundCount       int64  `json:"outbound_count"`
}

type PoisoningCandidateRecord struct {
	FocalWallet                string `json:"focal_wallet"`
	Signature                  string `json:"signature"`
	TransferIndex              int    `json:"transfer_index"`
	SuspiciousCounterparty     string `json:"suspicious_counterparty"`
	MatchedLegitCounterparty   string `json:"matched_legit_counterparty"`
	TokenMint                  string `json:"token_mint"`
	AmountRaw                  string `json:"amount_raw"`
	BlockTime                  string `json:"block_time"`
	IsZeroValue                bool   `json:"is_zero_value"`
	IsDust                     bool   `json:"is_dust"`
	IsNewCounterparty          bool   `json:"is_new_counterparty"`
	IsInbound                  bool   `json:"is_inbound"`
	LegitLastSeenAt            string `json:"legit_last_seen_at"`
	RecencyDays                int    `json:"recency_days"`
	InjectionCountInScanWindow int    `json:"injection_count_in_scan_window"`
	IncompleteWindow           bool   `json:"incomplete_window"`
	UnknownGateReason          string `json:"unknown_gate_reason"`
	MatchRuleVersion           string `json:"match_rule_version"`
}

type WalletSyncRunRecord struct {
	FocalWallet                 string `json:"focal_wallet"`
	Status                      string `json:"status"`
	BaselineComplete            bool   `json:"baseline_complete"`
	IncompleteWindow            bool   `json:"incomplete_window"`
	UnknownGateReason           string `json:"unknown_gate_reason"`
	TruncationReason            string `json:"truncation_reason"`
	TransactionsFetched         int    `json:"transactions_fetched"`
	TransactionsInserted        int    `json:"transactions_inserted"`
	TransactionsLinked          int    `json:"transactions_linked"`
	TransactionsFailedNormalize int    `json:"transactions_failed_to_normalize"`
	OwnerUnresolvedCount        int    `json:"owner_unresolved_count"`
	DecimalsUnresolvedCount     int    `json:"decimals_unresolved_count"`
	CounterpartiesCreated       int    `json:"counterparties_created"`
	CounterpartiesUpdated       int    `json:"counterparties_updated"`
	PoisoningCandidatesInserted int    `json:"poisoning_candidates_inserted"`
	RetryExhausted              bool   `json:"retry_exhausted"`
}

type IngestionRunDeltaRecord struct {
	WalletsRequested            int `json:"wallets_requested"`
	WalletsProcessed            int `json:"wallets_processed"`
	WalletsFailed               int `json:"wallets_failed"`
	WalletsSkipped              int `json:"wallets_skipped"`
	TransactionsFetched         int `json:"transactions_fetched"`
	TransactionsInserted        int `json:"transactions_inserted"`
	TransactionsLinked          int `json:"transactions_linked"`
	TransactionsFailedNormalize int `json:"transactions_failed_to_normalize"`
	OwnerUnresolvedCount        int `json:"owner_unresolved_count"`
	DecimalsUnresolvedCount     int `json:"decimals_unresolved_count"`
	CounterpartiesCreated       int `json:"counterparties_created"`
	CounterpartiesUpdated       int `json:"counterparties_updated"`
	PoisoningCandidatesInserted int `json:"poisoning_candidates_inserted"`
	RetryExhaustedCount         int `json:"retry_exhausted_count"`
}

type ReplayOutput struct {
	NormalizedTransfers []NormalizedTransferRecord
	WalletTransactions  []WalletTransactionRecord
	Counterparties      []CounterpartyRecord
	PoisoningCandidates []PoisoningCandidateRecord
	WalletSyncRuns      []WalletSyncRunRecord
	IngestionRunDelta   IngestionRunDeltaRecord
}

func ListCaseIDs(rootDir string) ([]string, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, fmt.Errorf("read fixtures root %s: %w", rootDir, err)
	}
	caseIDs := make([]string, 0, len(entries))
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		metaPath := filepath.Join(rootDir, ent.Name(), "meta.json")
		if _, err := os.Stat(metaPath); err != nil {
			continue
		}
		caseIDs = append(caseIDs, ent.Name())
	}
	sort.Strings(caseIDs)
	return caseIDs, nil
}

func LoadCase(rootDir, caseID string) (FixtureCase, error) {
	caseDir := filepath.Join(rootDir, caseID)
	metaPath := filepath.Join(caseDir, "meta.json")
	rawMeta, err := os.ReadFile(metaPath)
	if err != nil {
		return FixtureCase{}, fmt.Errorf("read meta for fixture %s: %w", caseID, err)
	}
	var meta Meta
	if err := json.Unmarshal(rawMeta, &meta); err != nil {
		return FixtureCase{}, fmt.Errorf("decode meta for fixture %s: %w", caseID, err)
	}
	if strings.TrimSpace(meta.CaseID) == "" {
		meta.CaseID = caseID
	}
	if strings.TrimSpace(meta.CaseID) != caseID {
		return FixtureCase{}, fmt.Errorf("fixture %s meta case_id mismatch: %s", caseID, meta.CaseID)
	}
	if err := applyMetaDefaults(&meta); err != nil {
		return FixtureCase{}, fmt.Errorf("normalize meta for fixture %s: %w", caseID, err)
	}

	rawDir := filepath.Join(caseDir, "raw")
	pages, err := loadRawPages(rawDir)
	if err != nil {
		return FixtureCase{}, fmt.Errorf("load raw pages for fixture %s: %w", caseID, err)
	}
	return FixtureCase{
		RootDir: rootDir,
		CaseDir: caseDir,
		Meta:    meta,
		pages:   pages,
	}, nil
}

func applyMetaDefaults(meta *Meta) error {
	if len(meta.FocalWallets) == 0 && strings.TrimSpace(meta.FocalWallet) != "" {
		meta.FocalWallets = []string{meta.FocalWallet}
	}
	if len(meta.FocalWallets) == 0 {
		return fmt.Errorf("focal_wallet or focal_wallets is required")
	}
	for i := range meta.FocalWallets {
		meta.FocalWallets[i] = strings.TrimSpace(meta.FocalWallets[i])
		if meta.FocalWallets[i] == "" {
			return fmt.Errorf("focal_wallets contains empty value")
		}
	}
	if meta.MaxTXPagesPerWallet <= 0 {
		meta.MaxTXPagesPerWallet = defaultMaxTXPagesPerWallet
	}
	if meta.MaxTXPerWallet <= 0 {
		meta.MaxTXPerWallet = defaultMaxTXPerWallet
	}
	if meta.MaxHeliusRetries < 0 {
		return fmt.Errorf("max_helius_retries must be >= 0")
	}
	if meta.MaxHeliusRetries == 0 {
		meta.MaxHeliusRetries = defaultMaxHeliusRetries
	}
	if meta.LookalikeRecencyDays <= 0 {
		meta.LookalikeRecencyDays = defaultLookalikeRecencyDays
	}
	if meta.LookalikePrefixMin <= 0 {
		meta.LookalikePrefixMin = defaultLookalikePrefixMin
	}
	if meta.LookalikeSuffixMin <= 0 {
		meta.LookalikeSuffixMin = defaultLookalikeSuffixMin
	}
	if meta.LookalikeSingleSideMin <= 0 {
		meta.LookalikeSingleSideMin = defaultLookalikeSingleSideMin
	}
	if meta.MinInjectionCount <= 0 {
		meta.MinInjectionCount = defaultMinInjectionCount
	}
	if meta.HeliusRequestDelayMS < 0 {
		return fmt.Errorf("helius_request_delay_ms must be >= 0")
	}
	if meta.TimeoutMS < 0 {
		return fmt.Errorf("timeout_ms must be >= 0")
	}
	if !meta.BaselineStart.Before(meta.BaselineEnd) || !meta.ScanStart.Before(meta.ScanEnd) {
		return fmt.Errorf("invalid baseline/scan windows")
	}
	if !meta.BaselineEnd.Equal(meta.ScanStart) {
		return fmt.Errorf("baseline_end must equal scan_start")
	}
	if len(meta.DustThresholds) == 0 {
		meta.DustThresholds = []DustThreshold{
			{
				AssetKey:   "SOL",
				AmountRaw:  "100",
				ActiveFrom: meta.BaselineStart,
			},
		}
	}
	for i := range meta.FetchScript {
		if strings.TrimSpace(meta.FetchScript[i].Kind) == "" {
			meta.FetchScript[i].Kind = "page"
		}
	}
	return nil
}

func loadRawPages(rawDir string) (map[string]helius.EnhancedPage, error) {
	pages := make(map[string]helius.EnhancedPage)
	entries, err := os.ReadDir(rawDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return pages, nil
		}
		return nil, err
	}
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		name := ent.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(rawDir, name))
		if err != nil {
			return nil, err
		}
		page, err := decodeRawPage(raw)
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", name, err)
		}
		pages[name] = page
	}
	return pages, nil
}

func decodeRawPage(raw []byte) (helius.EnhancedPage, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return helius.EnhancedPage{}, nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var txs []helius.EnhancedTransaction
		if err := json.Unmarshal(raw, &txs); err != nil {
			return helius.EnhancedPage{}, err
		}
		page := helius.EnhancedPage{Transactions: txs}
		if len(page.Transactions) > 0 {
			page.Before = page.Transactions[len(page.Transactions)-1].Signature
		}
		return page, nil
	}
	var page helius.EnhancedPage
	if err := json.Unmarshal(raw, &page); err != nil {
		return helius.EnhancedPage{}, err
	}
	if page.Before == "" && len(page.Transactions) > 0 {
		page.Before = page.Transactions[len(page.Transactions)-1].Signature
	}
	return page, nil
}

type scriptedClient struct {
	mu    sync.Mutex
	idx   int
	steps []FetchScriptStep
	pages map[string]helius.EnhancedPage
}

func newScriptedClient(meta Meta, pages map[string]helius.EnhancedPage) *scriptedClient {
	steps := make([]FetchScriptStep, 0)
	if len(meta.FetchScript) > 0 {
		steps = append(steps, meta.FetchScript...)
	} else {
		names := make([]string, 0, len(pages))
		for name := range pages {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			steps = append(steps, FetchScriptStep{Kind: "page", File: name})
		}
	}
	return &scriptedClient{steps: steps, pages: pages}
}

func (c *scriptedClient) nextStep() (FetchScriptStep, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.idx >= len(c.steps) {
		return FetchScriptStep{}, false
	}
	step := c.steps[c.idx]
	c.idx++
	return step, true
}

func (c *scriptedClient) FetchEnhancedPage(ctx context.Context, _ string, _ string) (helius.EnhancedPage, error) {
	step, ok := c.nextStep()
	if !ok {
		return helius.EnhancedPage{}, nil
	}
	if step.SleepMS > 0 {
		timer := time.NewTimer(time.Duration(step.SleepMS) * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return helius.EnhancedPage{}, ctx.Err()
		case <-timer.C:
		}
	}

	kind := strings.ToLower(strings.TrimSpace(step.Kind))
	switch kind {
	case "", "page":
		page, ok := c.pages[step.File]
		if !ok {
			return helius.EnhancedPage{}, fmt.Errorf("fixture script references missing raw page: %s", step.File)
		}
		return page, nil
	case "error":
		msg := strings.TrimSpace(step.Message)
		if msg == "" {
			msg = "scripted fetch error"
		}
		if step.StatusCode != 0 {
			return helius.EnhancedPage{}, fmt.Errorf("%s: %w", msg, helius.StatusError{StatusCode: step.StatusCode, Body: msg})
		}
		return helius.EnhancedPage{}, errors.New(msg)
	default:
		return helius.EnhancedPage{}, fmt.Errorf("unsupported fetch script kind: %s", step.Kind)
	}
}

func Replay(ctx context.Context, fx FixtureCase) (ReplayOutput, error) {
	classifyDust, err := buildDustClassifier(fx.Meta.DustThresholds)
	if err != nil {
		return ReplayOutput{}, fmt.Errorf("build fixture dust classifier: %w", err)
	}

	out := ReplayOutput{
		NormalizedTransfers: make([]NormalizedTransferRecord, 0),
		WalletTransactions:  make([]WalletTransactionRecord, 0),
		Counterparties:      make([]CounterpartyRecord, 0),
		PoisoningCandidates: make([]PoisoningCandidateRecord, 0),
		WalletSyncRuns:      make([]WalletSyncRunRecord, 0),
	}
	out.IngestionRunDelta.WalletsRequested = len(fx.Meta.FocalWallets)

	seenTransfers := make(map[string]struct{})
	seenWalletLinks := make(map[string]struct{})
	seenCandidates := make(map[string]struct{})
	seenCounterparty := make(map[string]bool)

	for _, wallet := range fx.Meta.FocalWallets {
		walletInserted := 0
		walletLinked := 0
		walletCounterpartiesCreated := 0
		walletCounterpartiesUpdated := 0
		walletCandidatesInserted := 0

		walletCtx := ctx
		cancel := func() {}
		if fx.Meta.TimeoutMS > 0 {
			walletCtx, cancel = context.WithTimeout(ctx, time.Duration(fx.Meta.TimeoutMS)*time.Millisecond)
		}

		client := newScriptedClient(fx.Meta, fx.pages)
		coreRes, runErr := pipeline.RunWalletCoreSync(walletCtx, client, pipeline.CoreSyncParams{
			FocalWalletAddress:     wallet,
			BaselineStart:          fx.Meta.BaselineStart,
			BaselineEnd:            fx.Meta.BaselineEnd,
			ScanStart:              fx.Meta.ScanStart,
			ScanEnd:                fx.Meta.ScanEnd,
			MaxTXPagesPerWallet:    fx.Meta.MaxTXPagesPerWallet,
			MaxTXPerWallet:         fx.Meta.MaxTXPerWallet,
			MaxHeliusRetries:       fx.Meta.MaxHeliusRetries,
			HeliusRequestDelay:     time.Duration(fx.Meta.HeliusRequestDelayMS) * time.Millisecond,
			LookalikeRecencyDays:   fx.Meta.LookalikeRecencyDays,
			LookalikePrefixMin:     fx.Meta.LookalikePrefixMin,
			LookalikeSuffixMin:     fx.Meta.LookalikeSuffixMin,
			LookalikeSingleSideMin: fx.Meta.LookalikeSingleSideMin,
			MinInjectionCount:      fx.Meta.MinInjectionCount,
			ClassifyDust:           classifyDust,
		})
		cancel()

		if runErr != nil {
			reason := ""
			status := "failed"
			if errors.Is(runErr, context.DeadlineExceeded) || errors.Is(walletCtx.Err(), context.DeadlineExceeded) {
				status = "timed_out"
				reason = "unknown_required_gates:wallet_timeout"
			}
			out.IngestionRunDelta.WalletsFailed++
			out.WalletSyncRuns = append(out.WalletSyncRuns, WalletSyncRunRecord{
				FocalWallet:       wallet,
				Status:            status,
				BaselineComplete:  false,
				IncompleteWindow:  true,
				UnknownGateReason: reason,
			})
			continue
		}

		out.IngestionRunDelta.WalletsProcessed++
		out.IngestionRunDelta.TransactionsFetched += coreRes.TransactionsFetched
		out.IngestionRunDelta.TransactionsFailedNormalize += coreRes.TransactionsFailedNormalize
		out.IngestionRunDelta.OwnerUnresolvedCount += coreRes.OwnerUnresolvedCount
		out.IngestionRunDelta.DecimalsUnresolvedCount += coreRes.DecimalsUnresolvedCount
		if coreRes.RetryExhausted {
			out.IngestionRunDelta.RetryExhaustedCount++
		}

		for _, tr := range dedupeTransfers(append(append([]transactions.NormalizedTransfer{}, coreRes.BaselineTransfers...), coreRes.ScanTransfers...)) {
			key := tr.Signature + "\x00" + tr.TransferFingerprint
			if _, ok := seenTransfers[key]; ok {
				continue
			}
			seenTransfers[key] = struct{}{}
			walletInserted++
			out.IngestionRunDelta.TransactionsInserted++
			out.NormalizedTransfers = append(out.NormalizedTransfers, normalizeTransferRecord(tr))
		}

		allObs := append(append([]pipeline.WalletTransferObservation{}, coreRes.BaselineObservations...), coreRes.ScanObservations...)
		for _, obs := range allObs {
			linkKey := wallet + "\x00" + string(obs.RelationType) + "\x00" + obs.Transfer.Signature + "\x00" + obs.Transfer.TransferFingerprint
			if _, ok := seenWalletLinks[linkKey]; ok {
				continue
			}
			seenWalletLinks[linkKey] = struct{}{}
			walletLinked++
			out.IngestionRunDelta.TransactionsLinked++
			out.WalletTransactions = append(out.WalletTransactions, WalletTransactionRecord{
				FocalWallet:         wallet,
				RelationType:        string(obs.RelationType),
				CounterpartyAddress: obs.CounterpartyAddress,
				Signature:           obs.Transfer.Signature,
				TransferIndex:       obs.Transfer.TransferIndex,
				TransferFingerprint: obs.Transfer.TransferFingerprint,
			})

			cpAddress := strings.TrimSpace(obs.CounterpartyAddress)
			if cpAddress == "" {
				continue
			}
			cpKey := wallet + "\x00" + cpAddress
			if !seenCounterparty[cpKey] {
				seenCounterparty[cpKey] = true
				walletCounterpartiesCreated++
				out.IngestionRunDelta.CounterpartiesCreated++
			} else {
				walletCounterpartiesUpdated++
				out.IngestionRunDelta.CounterpartiesUpdated++
			}
		}

		counterpartyKeys := make([]string, 0, len(coreRes.Counterparties))
		for cpAddr := range coreRes.Counterparties {
			counterpartyKeys = append(counterpartyKeys, cpAddr)
		}
		sort.Strings(counterpartyKeys)
		for _, cpAddr := range counterpartyKeys {
			cp := coreRes.Counterparties[cpAddr]
			out.Counterparties = append(out.Counterparties, CounterpartyRecord{
				FocalWallet:         wallet,
				CounterpartyAddress: cp.CounterpartyAddress,
				FirstSeenAt:         fmtTS(cp.FirstSeenAt),
				LastSeenAt:          fmtTS(cp.LastSeenAt),
				InteractionCount:    cp.InteractionCount,
				FirstInboundAt:      fmtPtrTS(cp.FirstInboundAt),
				LastInboundAt:       fmtPtrTS(cp.LastInboundAt),
				InboundCount:        cp.InboundCount,
				FirstOutboundAt:     fmtPtrTS(cp.FirstOutboundAt),
				LastOutboundAt:      fmtPtrTS(cp.LastOutboundAt),
				OutboundCount:       cp.OutboundCount,
			})
		}

		for _, candidate := range coreRes.Candidates {
			candidateKey := wallet + "\x00" + candidate.Signature + fmt.Sprintf("\x00%d", candidate.TransferIndex)
			if _, ok := seenCandidates[candidateKey]; ok {
				continue
			}
			seenCandidates[candidateKey] = struct{}{}
			walletCandidatesInserted++
			out.IngestionRunDelta.PoisoningCandidatesInserted++
			out.PoisoningCandidates = append(out.PoisoningCandidates, PoisoningCandidateRecord{
				FocalWallet:                wallet,
				Signature:                  candidate.Signature,
				TransferIndex:              candidate.TransferIndex,
				SuspiciousCounterparty:     candidate.SuspiciousCounterparty,
				MatchedLegitCounterparty:   candidate.MatchedLegitCounterparty,
				TokenMint:                  candidate.TokenMint,
				AmountRaw:                  candidate.AmountRaw,
				BlockTime:                  fmtTS(candidate.BlockTime),
				IsZeroValue:                candidate.IsZeroValue,
				IsDust:                     candidate.IsDust,
				IsNewCounterparty:          candidate.IsNewCounterparty,
				IsInbound:                  candidate.IsInbound,
				LegitLastSeenAt:            fmtTS(candidate.LegitLastSeenAt),
				RecencyDays:                candidate.RecencyDays,
				InjectionCountInScanWindow: candidate.RepeatInjectionCount,
				IncompleteWindow:           candidate.IncompleteWindow,
				UnknownGateReason:          candidate.UnknownGateReason,
				MatchRuleVersion:           candidate.MatchRuleVersion,
			})
		}

		truncationReason := mergeTruncationReason(coreRes.BaselineTruncation, coreRes.ScanTruncation)
		incomplete := coreRes.IncompleteWindow
		unknownReason := strings.TrimSpace(coreRes.UnknownGateReason)
		if truncationReason != "" {
			incomplete = true
			unknownReason = mergeReasons(
				unknownReason,
				truncationReasonFromCode("baseline", coreRes.BaselineTruncation),
				truncationReasonFromCode("scan", coreRes.ScanTruncation),
			)
		}
		if incomplete && unknownReason == "" {
			unknownReason = "unknown_required_gates:incomplete_window"
		}

		status := "succeeded"
		if incomplete {
			status = "partial"
		}
		out.WalletSyncRuns = append(out.WalletSyncRuns, WalletSyncRunRecord{
			FocalWallet:                 wallet,
			Status:                      status,
			BaselineComplete:            coreRes.BaselineComplete,
			IncompleteWindow:            incomplete,
			UnknownGateReason:           unknownReason,
			TruncationReason:            truncationReason,
			TransactionsFetched:         coreRes.TransactionsFetched,
			TransactionsInserted:        walletInserted,
			TransactionsLinked:          walletLinked,
			TransactionsFailedNormalize: coreRes.TransactionsFailedNormalize,
			OwnerUnresolvedCount:        coreRes.OwnerUnresolvedCount,
			DecimalsUnresolvedCount:     coreRes.DecimalsUnresolvedCount,
			CounterpartiesCreated:       walletCounterpartiesCreated,
			CounterpartiesUpdated:       walletCounterpartiesUpdated,
			PoisoningCandidatesInserted: walletCandidatesInserted,
			RetryExhausted:              coreRes.RetryExhausted,
		})
	}

	sortReplayOutput(&out)
	return out, nil
}

func dedupeTransfers(in []transactions.NormalizedTransfer) []transactions.NormalizedTransfer {
	if len(in) == 0 {
		return nil
	}
	out := make([]transactions.NormalizedTransfer, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, tr := range in {
		key := tr.Signature + "\x00" + tr.TransferFingerprint
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, tr)
	}
	return out
}

func normalizeTransferRecord(tr transactions.NormalizedTransfer) NormalizedTransferRecord {
	return NormalizedTransferRecord{
		Signature:               tr.Signature,
		TransferIndex:           tr.TransferIndex,
		TransferFingerprint:     tr.TransferFingerprint,
		Slot:                    tr.Slot,
		BlockTime:               fmtTS(tr.BlockTime),
		SourceOwnerAddress:      tr.SourceOwnerAddress,
		DestinationOwnerAddress: tr.DestinationOwnerAddress,
		SourceTokenAccount:      tr.SourceTokenAccount,
		DestinationTokenAccount: tr.DestinationTokenAccount,
		AssetType:               string(tr.AssetType),
		AssetKey:                tr.AssetKey,
		TokenMint:               tr.TokenMint,
		AmountRaw:               tr.AmountRaw,
		Decimals:                tr.Decimals,
		NormalizationStatus:     string(tr.NormalizationStatus),
		NormalizationReasonCode: tr.NormalizationReasonCode,
		PoisoningEligible:       tr.PoisoningEligible,
		DustStatus:              string(tr.DustStatus),
		IsSuccess:               tr.IsSuccess,
	}
}

func fmtTS(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func fmtPtrTS(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func truncationReasonFromCode(window, code string) string {
	if strings.TrimSpace(code) == "" {
		return ""
	}
	return window + "_truncation:" + code
}

func mergeTruncationReason(baselineCode, scanCode string) string {
	reasons := make([]string, 0, 2)
	if strings.TrimSpace(baselineCode) != "" {
		reasons = append(reasons, truncationReasonFromCode("baseline", baselineCode))
	}
	if strings.TrimSpace(scanCode) != "" {
		reasons = append(reasons, truncationReasonFromCode("scan", scanCode))
	}
	return strings.Join(reasons, ";")
}

func mergeReasons(reasons ...string) string {
	uniq := make(map[string]struct{})
	for _, reason := range reasons {
		reason = strings.TrimSpace(reason)
		if reason == "" {
			continue
		}
		uniq[reason] = struct{}{}
	}
	if len(uniq) == 0 {
		return ""
	}
	keys := make([]string, 0, len(uniq))
	for k := range uniq {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ";")
}

func sortReplayOutput(out *ReplayOutput) {
	sort.Slice(out.NormalizedTransfers, func(i, j int) bool {
		a := out.NormalizedTransfers[i]
		b := out.NormalizedTransfers[j]
		if a.BlockTime != b.BlockTime {
			return a.BlockTime < b.BlockTime
		}
		if a.Signature != b.Signature {
			return a.Signature < b.Signature
		}
		if a.TransferIndex != b.TransferIndex {
			return a.TransferIndex < b.TransferIndex
		}
		return a.TransferFingerprint < b.TransferFingerprint
	})
	sort.Slice(out.WalletTransactions, func(i, j int) bool {
		a := out.WalletTransactions[i]
		b := out.WalletTransactions[j]
		if a.FocalWallet != b.FocalWallet {
			return a.FocalWallet < b.FocalWallet
		}
		if a.Signature != b.Signature {
			return a.Signature < b.Signature
		}
		if a.TransferIndex != b.TransferIndex {
			return a.TransferIndex < b.TransferIndex
		}
		if a.RelationType != b.RelationType {
			return a.RelationType < b.RelationType
		}
		return a.TransferFingerprint < b.TransferFingerprint
	})
	sort.Slice(out.Counterparties, func(i, j int) bool {
		a := out.Counterparties[i]
		b := out.Counterparties[j]
		if a.FocalWallet != b.FocalWallet {
			return a.FocalWallet < b.FocalWallet
		}
		return a.CounterpartyAddress < b.CounterpartyAddress
	})
	sort.Slice(out.PoisoningCandidates, func(i, j int) bool {
		a := out.PoisoningCandidates[i]
		b := out.PoisoningCandidates[j]
		if a.FocalWallet != b.FocalWallet {
			return a.FocalWallet < b.FocalWallet
		}
		if a.Signature != b.Signature {
			return a.Signature < b.Signature
		}
		return a.TransferIndex < b.TransferIndex
	})
	sort.Slice(out.WalletSyncRuns, func(i, j int) bool {
		return out.WalletSyncRuns[i].FocalWallet < out.WalletSyncRuns[j].FocalWallet
	})
}

func buildDustClassifier(thresholds []DustThreshold) (func(tr transactions.NormalizedTransfer) transactions.DustStatus, error) {
	type dustRule struct {
		From      time.Time
		To        *time.Time
		Threshold *big.Int
	}

	index := make(map[string][]dustRule)
	for _, rec := range thresholds {
		asset := strings.TrimSpace(rec.AssetKey)
		if asset == "" {
			return nil, fmt.Errorf("dust threshold asset_key is empty")
		}
		amountRaw := strings.TrimSpace(rec.AmountRaw)
		v, ok := new(big.Int).SetString(amountRaw, 10)
		if !ok || v.Sign() < 0 {
			return nil, fmt.Errorf("invalid dust threshold amount_raw for asset %s: %s", asset, rec.AmountRaw)
		}
		from := rec.ActiveFrom.UTC()
		var to *time.Time
		if rec.ActiveTo != nil {
			toVal := rec.ActiveTo.UTC()
			if !toVal.After(from) {
				return nil, fmt.Errorf("invalid dust threshold window for %s", asset)
			}
			to = &toVal
		}
		index[asset] = append(index[asset], dustRule{From: from, To: to, Threshold: v})
	}
	for asset := range index {
		rules := index[asset]
		sort.Slice(rules, func(i, j int) bool {
			return rules[i].From.Before(rules[j].From)
		})
		for i := 1; i < len(rules); i++ {
			prev := rules[i-1]
			curr := rules[i]
			if prev.To == nil || curr.From.Before(*prev.To) {
				return nil, fmt.Errorf("overlapping dust threshold windows for %s", asset)
			}
		}
		index[asset] = rules
	}
	return func(tr transactions.NormalizedTransfer) transactions.DustStatus {
		amount, ok := new(big.Int).SetString(strings.TrimSpace(tr.AmountRaw), 10)
		if !ok || amount.Sign() < 0 {
			return transactions.DustUnknown
		}
		if amount.Sign() == 0 {
			return transactions.DustTrue
		}
		rules := index[strings.TrimSpace(tr.AssetKey)]
		if len(rules) == 0 {
			return transactions.DustUnknown
		}
		at := tr.BlockTime.UTC()
		for i := len(rules) - 1; i >= 0; i-- {
			rule := rules[i]
			if at.Before(rule.From) {
				continue
			}
			if rule.To != nil && !at.Before(*rule.To) {
				continue
			}
			if amount.Cmp(rule.Threshold) <= 0 {
				return transactions.DustTrue
			}
			return transactions.DustFalse
		}
		return transactions.DustUnknown
	}, nil
}

func CompareExpected(fx FixtureCase, out ReplayOutput) error {
	if err := compareExpectedFile(filepath.Join(fx.CaseDir, "expected", "normalized_transfers.json"), out.NormalizedTransfers); err != nil {
		return err
	}
	if err := compareExpectedFile(filepath.Join(fx.CaseDir, "expected", "wallet_transactions.json"), out.WalletTransactions); err != nil {
		return err
	}
	if err := compareExpectedFile(filepath.Join(fx.CaseDir, "expected", "counterparties.json"), out.Counterparties); err != nil {
		return err
	}
	if err := compareExpectedFile(filepath.Join(fx.CaseDir, "expected", "poisoning_candidates.json"), out.PoisoningCandidates); err != nil {
		return err
	}
	if err := compareExpectedFile(filepath.Join(fx.CaseDir, "expected", "wallet_sync_run.json"), out.WalletSyncRuns); err != nil {
		return err
	}
	if err := compareExpectedFile(filepath.Join(fx.CaseDir, "expected", "ingestion_run_delta.json"), out.IngestionRunDelta); err != nil {
		return err
	}
	return nil
}

func compareExpectedFile(path string, actual any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read expected file %s: %w", path, err)
	}
	expected := reflect.New(reflect.TypeOf(actual))
	if err := json.Unmarshal(raw, expected.Interface()); err != nil {
		return fmt.Errorf("decode expected file %s: %w", path, err)
	}
	if !reflect.DeepEqual(expected.Elem().Interface(), actual) {
		expJSON, _ := json.MarshalIndent(expected.Elem().Interface(), "", "  ")
		actJSON, _ := json.MarshalIndent(actual, "", "  ")
		return fmt.Errorf("expected mismatch for %s\nexpected:\n%s\nactual:\n%s", path, string(expJSON), string(actJSON))
	}
	return nil
}

func WriteExpected(fx FixtureCase, out ReplayOutput) error {
	expectedDir := filepath.Join(fx.CaseDir, "expected")
	if err := os.MkdirAll(expectedDir, 0o755); err != nil {
		return fmt.Errorf("mkdir expected dir: %w", err)
	}
	if err := writeExpectedFile(filepath.Join(expectedDir, "normalized_transfers.json"), out.NormalizedTransfers); err != nil {
		return err
	}
	if err := writeExpectedFile(filepath.Join(expectedDir, "wallet_transactions.json"), out.WalletTransactions); err != nil {
		return err
	}
	if err := writeExpectedFile(filepath.Join(expectedDir, "counterparties.json"), out.Counterparties); err != nil {
		return err
	}
	if err := writeExpectedFile(filepath.Join(expectedDir, "poisoning_candidates.json"), out.PoisoningCandidates); err != nil {
		return err
	}
	if err := writeExpectedFile(filepath.Join(expectedDir, "wallet_sync_run.json"), out.WalletSyncRuns); err != nil {
		return err
	}
	if err := writeExpectedFile(filepath.Join(expectedDir, "ingestion_run_delta.json"), out.IngestionRunDelta); err != nil {
		return err
	}
	return nil
}

func writeExpectedFile(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
