package walletsource

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"poisontrace/internal/helius"
	"poisontrace/internal/pipeline"
	"poisontrace/internal/transactions"
)

type Options struct {
	SeedWalletFile    string
	SeedWallets       []string
	ScrapeStats       map[string]ScrapedWallet
	ScoreSeedWallets  bool
	DiscoverNeighbors bool
	OutPath           string
	RejectedOutPath   string
	ScanStart         time.Time
	ScanEnd           time.Time
	BaselineLookback  time.Duration

	TargetCount             int
	MaxSeedWallets          int
	MaxCandidates           int
	SeedMaxPages            int
	CandidateMaxPages       int
	MaxTXPerWallet          int
	MaxRetries              int
	RequestDelay            time.Duration
	MinOutbound             int
	MaxAcceptedTX           int
	MaxSameTimestampTX      int
	MaxTransfersPerTX       int
	MaxUnknownDustSPL       int
	MinScanInboundDust      int
	MinUniqueDustRecipients int
	KnownDustAssetKeys      []string
	DeepDiveTopN            int
	DeepDiveMaxPages        int
	DeepDiveMaxTX           int
	DeepDiveMinScore        int
	SourceMode              string
}

const (
	SourceModeVictimInboundDust    = "victim_inbound_dust"
	SourceModeAttackerOutboundDust = "attacker_outbound_dust"
)

type Result struct {
	Accepted []AcceptedWallet
	Rejected []RejectedWallet
}

type AcceptedWallet struct {
	Address                         string
	ScrapeCount                     int
	ScrapeOutbound                  int
	ScrapeInbound                   int
	SampleTransactions              int
	OutboundTransfers               int
	InboundTransfers                int
	UniqueCounterparties            int
	LegitOutboundCounterparties     int
	ScanInboundDustTransfers        int
	UniqueDustRecipients            int
	RepeatedInboundDustCounterparts int
	LookalikeInboundDustMatches     int
	SourceScore                     int
	DiscoveredFromSeeds             int
}

type RejectedWallet struct {
	Address                         string
	Reason                          string
	ScrapeCount                     int
	ScrapeOutbound                  int
	ScrapeInbound                   int
	SampleTransactions              int
	OutboundTransfers               int
	InboundTransfers                int
	UniqueCounterparties            int
	LegitOutboundCounterparties     int
	ScanInboundDustTransfers        int
	UniqueDustRecipients            int
	RepeatedInboundDustCounterparts int
	LookalikeInboundDustMatches     int
	SourceScore                     int
	DiscoveredFromSeeds             int
}

type candidateDiscovery struct {
	address string
	seeds   map[string]struct{}
}

type walletStats struct {
	address                         string
	scrapeCount                     int
	scrapeOutbound                  int
	scrapeInbound                   int
	sampleTransactions              int
	outboundTransfers               int
	inboundTransfers                int
	uniqueCounterparties            int
	legitOutboundCounterparties     int
	scanInboundDustTransfers        int
	uniqueDustRecipients            int
	repeatedInboundDustCounterparts int
	lookalikeInboundDustMatches     int
	sourceScore                     int
	maxSameTimestampTX              int
	maxTransfersPerTX               int
	unknownDustSPL                  int
	discoveredFromSeeds             int
}

type deepDiveCandidate struct {
	discovery      candidateDiscovery
	initialStats   walletStats
	truncationCode string
}

func Source(ctx context.Context, client helius.Client, opts Options) (Result, error) {
	if client == nil {
		return Result{}, fmt.Errorf("helius client is required")
	}
	if opts.SeedWalletFile == "" && len(opts.SeedWallets) == 0 {
		return Result{}, fmt.Errorf("seed wallet file or seed wallets are required")
	}
	if opts.OutPath == "" {
		return Result{}, fmt.Errorf("out path is required")
	}
	if opts.RejectedOutPath == "" {
		return Result{}, fmt.Errorf("rejected out path is required")
	}
	if !opts.ScanStart.Before(opts.ScanEnd) {
		return Result{}, fmt.Errorf("scan window must satisfy start < end")
	}
	opts = withDefaults(opts)

	seeds, err := loadSeedWallets(opts.SeedWalletFile, opts.SeedWallets)
	if err != nil {
		return Result{}, err
	}
	if len(seeds) == 0 {
		return Result{}, fmt.Errorf("seed wallet file has no wallets")
	}
	if opts.MaxSeedWallets > 0 && len(seeds) > opts.MaxSeedWallets {
		seeds = seeds[:opts.MaxSeedWallets]
	}

	discovered := make(map[string]*candidateDiscovery)
	seedSet := make(map[string]struct{}, len(seeds))
	windowStart := opts.ScanStart.Add(-opts.BaselineLookback).UTC()
	for _, seed := range seeds {
		seedSet[seed] = struct{}{}
		if opts.ScoreSeedWallets {
			addDiscovered(discovered, seed, nil, seed)
		}
		if !opts.DiscoverNeighbors {
			continue
		}
		page, fetchErr := pipeline.FetchEnhancedWindow(ctx, client, seed, pipeline.FetchWindowParams{
			Start:        windowStart,
			End:          opts.ScanEnd.UTC(),
			MaxPages:     opts.SeedMaxPages,
			MaxTx:        opts.MaxTXPerWallet,
			MaxRetries:   opts.MaxRetries,
			RequestDelay: opts.RequestDelay,
		})
		if fetchErr != nil {
			return Result{}, fmt.Errorf("fetch seed wallet %s: %w", seed, fetchErr)
		}
		for _, tx := range page.Transactions {
			transfers, normalizeErr := transactions.NormalizeEnhancedTx(tx)
			if normalizeErr != nil {
				continue
			}
			for _, tr := range transfers {
				addDiscovered(discovered, seed, seedSet, tr.SourceOwnerAddress)
				addDiscovered(discovered, seed, seedSet, tr.DestinationOwnerAddress)
			}
		}
	}

	candidates := sortedDiscoveries(discovered)
	if opts.MaxCandidates > 0 && len(candidates) > opts.MaxCandidates {
		candidates = candidates[:opts.MaxCandidates]
	}
	knownDustThresholds := knownDustThresholdMap(opts.KnownDustAssetKeys)

	result := Result{
		Accepted: make([]AcceptedWallet, 0, opts.TargetCount),
		Rejected: make([]RejectedWallet, 0, len(candidates)),
	}
	eligible := make([]walletStats, 0, len(candidates))
	deepDiveQueue := make([]deepDiveCandidate, 0)
	for _, c := range candidates {
		page, fetchErr := pipeline.FetchEnhancedWindow(ctx, client, c.address, pipeline.FetchWindowParams{
			Start:        windowStart,
			End:          opts.ScanEnd.UTC(),
			MaxPages:     opts.CandidateMaxPages,
			MaxTx:        opts.MaxTXPerWallet,
			MaxRetries:   opts.MaxRetries,
			RequestDelay: opts.RequestDelay,
		})
		if fetchErr != nil {
			scrape := opts.ScrapeStats[c.address]
			result.Rejected = append(result.Rejected, RejectedWallet{
				Address:             c.address,
				Reason:              "fetch_error",
				ScrapeCount:         scrape.Count,
				ScrapeOutbound:      scrape.Outbound,
				ScrapeInbound:       scrape.Inbound,
				DiscoveredFromSeeds: len(c.seeds),
			})
			continue
		}
		stats := summarizeWallet(c.address, page.Transactions, opts.ScanStart.UTC(), knownDustThresholds, opts.SourceMode)
		if scrape, ok := opts.ScrapeStats[c.address]; ok {
			stats.scrapeCount = scrape.Count
			stats.scrapeOutbound = scrape.Outbound
			stats.scrapeInbound = scrape.Inbound
		}
		stats.discoveredFromSeeds = len(c.seeds)
		stats.sourceScore = sourceScore(stats)

		if page.Partial {
			if shouldDeepDive(stats, page.TruncationCode, opts) {
				deepDiveQueue = append(deepDiveQueue, deepDiveCandidate{
					discovery:      c,
					initialStats:   stats,
					truncationCode: page.TruncationCode,
				})
				continue
			}
			result.Rejected = append(result.Rejected, rejectedFromStats(stats, "activity_cap_reached:"+page.TruncationCode))
			continue
		}
		if page.RetryExhausted {
			result.Rejected = append(result.Rejected, rejectedFromStats(stats, "helius_retry_exhausted"))
			continue
		}
		if reason, ok := statsRejectReason(stats, opts); ok {
			result.Rejected = append(result.Rejected, rejectedFromStats(stats, reason))
			continue
		}

		eligible = append(eligible, stats)
	}
	if len(deepDiveQueue) > 0 && opts.DeepDiveTopN > 0 {
		sort.SliceStable(deepDiveQueue, func(i, j int) bool {
			if deepDiveQueue[i].initialStats.sourceScore == deepDiveQueue[j].initialStats.sourceScore {
				return deepDiveQueue[i].initialStats.address < deepDiveQueue[j].initialStats.address
			}
			return deepDiveQueue[i].initialStats.sourceScore > deepDiveQueue[j].initialStats.sourceScore
		})
		for idx, cand := range deepDiveQueue {
			if idx >= opts.DeepDiveTopN {
				result.Rejected = append(result.Rejected, rejectedFromStats(cand.initialStats, "activity_cap_reached:"+cand.truncationCode))
				continue
			}
			if opts.TargetCount > 0 && len(eligible) >= opts.TargetCount {
				result.Rejected = append(result.Rejected, rejectedFromStats(cand.initialStats, "target_count_reached"))
				continue
			}
			reFetched, err := pipeline.FetchEnhancedWindow(ctx, client, cand.discovery.address, pipeline.FetchWindowParams{
				Start:        windowStart,
				End:          opts.ScanEnd.UTC(),
				MaxPages:     opts.DeepDiveMaxPages,
				MaxTx:        opts.DeepDiveMaxTX,
				MaxRetries:   opts.MaxRetries,
				RequestDelay: opts.RequestDelay,
			})
			if err != nil {
				result.Rejected = append(result.Rejected, rejectedFromStats(cand.initialStats, "deep_dive_fetch_error"))
				continue
			}
			reStats := summarizeWallet(cand.discovery.address, reFetched.Transactions, opts.ScanStart.UTC(), knownDustThresholds, opts.SourceMode)
			if scrape, ok := opts.ScrapeStats[cand.discovery.address]; ok {
				reStats.scrapeCount = scrape.Count
				reStats.scrapeOutbound = scrape.Outbound
				reStats.scrapeInbound = scrape.Inbound
			}
			reStats.discoveredFromSeeds = len(cand.discovery.seeds)
			reStats.sourceScore = sourceScore(reStats)

			if reFetched.Partial {
				result.Rejected = append(result.Rejected, rejectedFromStats(reStats, "activity_cap_reached:"+reFetched.TruncationCode))
				continue
			}
			if reFetched.RetryExhausted {
				result.Rejected = append(result.Rejected, rejectedFromStats(reStats, "helius_retry_exhausted"))
				continue
			}
			if reason, ok := statsRejectReason(reStats, opts); ok {
				result.Rejected = append(result.Rejected, rejectedFromStats(reStats, reason))
				continue
			}
			eligible = append(eligible, reStats)
		}
	}

	sort.SliceStable(eligible, func(i, j int) bool {
		if eligible[i].sourceScore == eligible[j].sourceScore {
			if eligible[i].scanInboundDustTransfers == eligible[j].scanInboundDustTransfers {
				if eligible[i].legitOutboundCounterparties == eligible[j].legitOutboundCounterparties {
					return eligible[i].address < eligible[j].address
				}
				return eligible[i].legitOutboundCounterparties > eligible[j].legitOutboundCounterparties
			}
			return eligible[i].scanInboundDustTransfers > eligible[j].scanInboundDustTransfers
		}
		return eligible[i].sourceScore > eligible[j].sourceScore
	})
	for i, stats := range eligible {
		if opts.TargetCount > 0 && i >= opts.TargetCount {
			result.Rejected = append(result.Rejected, rejectedFromStats(stats, "target_count_reached"))
			continue
		}
		result.Accepted = append(result.Accepted, acceptedFromStats(stats))
	}

	if err := writeAccepted(opts.OutPath, result.Accepted); err != nil {
		return Result{}, err
	}
	if err := writeRejected(opts.RejectedOutPath, result.Rejected); err != nil {
		return Result{}, err
	}
	return result, nil
}

func SourceDiscovered(ctx context.Context, client helius.Client, seeds []string, opts Options) (Result, error) {
	opts.SeedWallets = seeds
	opts.SeedWalletFile = ""
	opts.ScoreSeedWallets = true
	opts.DiscoverNeighbors = false
	return Source(ctx, client, opts)
}

func withDefaults(opts Options) Options {
	if opts.TargetCount <= 0 {
		opts.TargetCount = 30
	}
	if opts.MaxSeedWallets <= 0 {
		opts.MaxSeedWallets = 10
	}
	if opts.MaxCandidates <= 0 {
		opts.MaxCandidates = 100
	}
	if opts.SeedMaxPages <= 0 {
		opts.SeedMaxPages = 3
	}
	if opts.CandidateMaxPages <= 0 {
		opts.CandidateMaxPages = 5
	}
	if opts.MaxTXPerWallet <= 0 {
		opts.MaxTXPerWallet = 400
	}
	if opts.MinOutbound <= 0 {
		opts.MinOutbound = 1
	}
	if opts.MinUniqueDustRecipients <= 0 {
		opts.MinUniqueDustRecipients = 10
	}
	if opts.MaxAcceptedTX <= 0 {
		opts.MaxAcceptedTX = opts.MaxTXPerWallet
	}
	if opts.MaxSameTimestampTX <= 0 {
		opts.MaxSameTimestampTX = 5
	}
	if opts.MaxTransfersPerTX <= 0 {
		opts.MaxTransfersPerTX = 4
	}
	if opts.DeepDiveTopN < 0 {
		opts.DeepDiveTopN = 0
	}
	if opts.DeepDiveMaxPages <= 0 {
		opts.DeepDiveMaxPages = opts.CandidateMaxPages + 2
	}
	if opts.DeepDiveMaxTX <= 0 {
		opts.DeepDiveMaxTX = opts.MaxTXPerWallet + 200
	}
	if opts.DeepDiveMinScore <= 0 {
		opts.DeepDiveMinScore = 40
	}
	if len(opts.KnownDustAssetKeys) == 0 {
		opts.KnownDustAssetKeys = DefaultKnownDustAssetKeys()
	}
	if !opts.ScoreSeedWallets && !opts.DiscoverNeighbors {
		opts.DiscoverNeighbors = true
	}
	opts.SourceMode = normalizeSourceMode(opts.SourceMode)
	return opts
}

func normalizeSourceMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case "", SourceModeVictimInboundDust:
		return SourceModeVictimInboundDust
	case SourceModeAttackerOutboundDust:
		return SourceModeAttackerOutboundDust
	default:
		return SourceModeVictimInboundDust
	}
}

func DefaultKnownDustAssetKeys() []string {
	thresholds := DefaultKnownDustThresholds()
	keys := make([]string, 0, len(thresholds))
	for key := range thresholds {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func DefaultKnownDustThresholds() map[string]int64 {
	return map[string]int64{
		"SOL": 1000,
		"So11111111111111111111111111111111111111112":  1000,
		"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v": 1000,
		"Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB": 1000,
	}
}

func addDiscovered(discovered map[string]*candidateDiscovery, seed string, seedSet map[string]struct{}, address string) {
	address = strings.TrimSpace(address)
	if address == "" {
		return
	}
	if seedSet != nil {
		if _, ok := seedSet[address]; ok {
			return
		}
	}
	rec, ok := discovered[address]
	if !ok {
		rec = &candidateDiscovery{address: address, seeds: make(map[string]struct{})}
		discovered[address] = rec
	}
	rec.seeds[seed] = struct{}{}
}

func sortedDiscoveries(discovered map[string]*candidateDiscovery) []candidateDiscovery {
	out := make([]candidateDiscovery, 0, len(discovered))
	for _, c := range discovered {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].seeds) == len(out[j].seeds) {
			return out[i].address < out[j].address
		}
		return len(out[i].seeds) > len(out[j].seeds)
	})
	return out
}

func readWalletLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open wallet file %s: %w", path, err)
	}
	defer f.Close()

	seen := make(map[string]struct{})
	out := make([]string, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line == "" {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		out = append(out, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read wallet file %s: %w", path, err)
	}
	return out, nil
}

func loadSeedWallets(path string, explicit []string) ([]string, error) {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(explicit))
	for _, wallet := range explicit {
		wallet = strings.TrimSpace(wallet)
		if wallet == "" {
			continue
		}
		if _, ok := seen[wallet]; ok {
			continue
		}
		seen[wallet] = struct{}{}
		out = append(out, wallet)
	}
	if path == "" {
		return out, nil
	}
	fromFile, err := readWalletLines(path)
	if err != nil {
		return nil, err
	}
	for _, wallet := range fromFile {
		if _, ok := seen[wallet]; ok {
			continue
		}
		seen[wallet] = struct{}{}
		out = append(out, wallet)
	}
	return out, nil
}

func summarizeWallet(address string, txs []helius.EnhancedTransaction, scanStart time.Time, knownDustThresholds map[string]int64, sourceMode string) walletStats {
	stats := walletStats{address: address, sampleTransactions: len(txs)}
	counterparties := make(map[string]struct{})
	legitOutbound := make(map[string]struct{})
	scanInboundDustByCounterparty := make(map[string]int)
	sameTimestamp := make(map[int64]int)
	for _, tx := range txs {
		sameTimestamp[tx.TimestampUnix]++
		if sameTimestamp[tx.TimestampUnix] > stats.maxSameTimestampTX {
			stats.maxSameTimestampTX = sameTimestamp[tx.TimestampUnix]
		}

		transfers, err := transactions.NormalizeEnhancedTx(tx)
		if err != nil {
			continue
		}
		transfersInTx := 0
		for _, tr := range transfers {
			if !tr.PoisoningEligible || tr.NormalizationStatus != transactions.NormalizationResolved {
				continue
			}
			switch {
			case tr.SourceOwnerAddress == address:
				stats.outboundTransfers++
				transfersInTx++
				if tr.DestinationOwnerAddress != "" && tr.DestinationOwnerAddress != address {
					counterparties[tr.DestinationOwnerAddress] = struct{}{}
					if tx.BlockTimeUTC().Before(scanStart) && sourceDustStatus(tr, knownDustThresholds) == transactions.DustFalse {
						legitOutbound[tr.DestinationOwnerAddress] = struct{}{}
					}
				}
				if transferNeedsKnownDustThreshold(tr, knownDustThresholds) {
					stats.unknownDustSPL++
				}
			case tr.DestinationOwnerAddress == address:
				stats.inboundTransfers++
				transfersInTx++
				if tr.SourceOwnerAddress != "" && tr.SourceOwnerAddress != address {
					counterparties[tr.SourceOwnerAddress] = struct{}{}
					if sourceMode == SourceModeVictimInboundDust && !tx.BlockTimeUTC().Before(scanStart) && sourceDustStatus(tr, knownDustThresholds) == transactions.DustTrue {
						scanInboundDustByCounterparty[tr.SourceOwnerAddress]++
					}
				}
				if transferNeedsKnownDustThreshold(tr, knownDustThresholds) {
					stats.unknownDustSPL++
				}
			}
			if sourceMode == SourceModeAttackerOutboundDust &&
				tr.SourceOwnerAddress == address &&
				tr.DestinationOwnerAddress != "" &&
				tr.DestinationOwnerAddress != address &&
				!tx.BlockTimeUTC().Before(scanStart) &&
				sourceDustStatus(tr, knownDustThresholds) == transactions.DustTrue {
				scanInboundDustByCounterparty[tr.DestinationOwnerAddress]++
			}
		}
		if transfersInTx > stats.maxTransfersPerTX {
			stats.maxTransfersPerTX = transfersInTx
		}
	}
	stats.uniqueCounterparties = len(counterparties)
	stats.legitOutboundCounterparties = len(legitOutbound)
	for counterparty, count := range scanInboundDustByCounterparty {
		stats.scanInboundDustTransfers += count
		if count >= 2 {
			stats.repeatedInboundDustCounterparts++
		}
		if hasLookalikeMatch(counterparty, legitOutbound) {
			stats.lookalikeInboundDustMatches++
		}
	}
	stats.uniqueDustRecipients = len(scanInboundDustByCounterparty)
	stats.sourceScore = sourceScore(stats)
	return stats
}

func sourceDustStatus(tr transactions.NormalizedTransfer, knownDustThresholds map[string]int64) transactions.DustStatus {
	if amountRawIsZero(tr.AmountRaw) {
		return transactions.DustTrue
	}
	switch tr.AssetType {
	case transactions.AssetTypeNativeSOL, transactions.AssetTypeSPLFungible:
	default:
		return transactions.DustUnknown
	}
	threshold, ok := knownDustThresholds[tr.AssetKey]
	if !ok {
		return transactions.DustUnknown
	}
	if amountRawLTE(tr.AmountRaw, threshold) {
		return transactions.DustTrue
	}
	return transactions.DustFalse
}

func sourceScore(stats walletStats) int {
	score := 0
	score += stats.lookalikeInboundDustMatches * 100
	score += stats.repeatedInboundDustCounterparts * 50
	score += stats.scanInboundDustTransfers * 10
	score += stats.legitOutboundCounterparties * 3
	score += stats.discoveredFromSeeds
	return score
}

func shouldDeepDive(stats walletStats, truncationCode string, opts Options) bool {
	if opts.DeepDiveTopN <= 0 {
		return false
	}
	if truncationCode != "max_tx_pages_per_wallet_reached" && truncationCode != "max_tx_per_wallet_reached" {
		return false
	}
	if stats.scanInboundDustTransfers < opts.MinScanInboundDust {
		return false
	}
	if opts.SourceMode == SourceModeAttackerOutboundDust {
		// In attacker mode, dust-signal wallets should still earn a deep-dive retry even when
		// baseline-style score features are weak under truncated sampling.
		return true
	}
	if stats.sourceScore < opts.DeepDiveMinScore {
		return false
	}
	return true
}

func statsRejectReason(stats walletStats, opts Options) (string, bool) {
	if stats.sampleTransactions == 0 {
		return "no_sample_transactions", true
	}
	if stats.sampleTransactions > opts.MaxAcceptedTX {
		return "too_many_sample_transactions", true
	}
	if stats.outboundTransfers < opts.MinOutbound {
		return "insufficient_outbound_history", true
	}
	if stats.maxSameTimestampTX > opts.MaxSameTimestampTX {
		return "bursty_same_timestamp_activity", true
	}
	if stats.maxTransfersPerTX > opts.MaxTransfersPerTX {
		return "batch_transfer_activity", true
	}
	if stats.unknownDustSPL > opts.MaxUnknownDustSPL {
		return "unknown_dust_spl_activity", true
	}
	if stats.scanInboundDustTransfers < opts.MinScanInboundDust {
		if opts.SourceMode == SourceModeAttackerOutboundDust {
			return "insufficient_outbound_dust_activity", true
		}
		return "insufficient_inbound_dust_activity", true
	}
	if opts.SourceMode == SourceModeAttackerOutboundDust && stats.uniqueDustRecipients < opts.MinUniqueDustRecipients {
		return "insufficient_unique_dust_recipients", true
	}
	return "", false
}

func hasLookalikeMatch(suspicious string, legit map[string]struct{}) bool {
	for candidate := range legit {
		prefix := commonPrefixLength(suspicious, candidate)
		suffix := commonSuffixLength(suspicious, candidate)
		if (prefix >= 4 && suffix >= 4) || prefix >= 6 || suffix >= 6 {
			return true
		}
	}
	return false
}

func knownDustThresholdMap(keys []string) map[string]int64 {
	defaults := DefaultKnownDustThresholds()
	out := make(map[string]int64, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		threshold, ok := defaults[key]
		if !ok {
			continue
		}
		out[key] = threshold
	}
	return out
}

func transferNeedsKnownDustThreshold(tr transactions.NormalizedTransfer, knownDustThresholds map[string]int64) bool {
	if tr.AssetType != transactions.AssetTypeSPLFungible {
		return false
	}
	if amountRawIsZero(tr.AmountRaw) {
		return false
	}
	_, ok := knownDustThresholds[tr.AssetKey]
	return !ok
}

func amountRawIsZero(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" {
		return false
	}
	for _, r := range v {
		if r != '0' {
			return false
		}
	}
	return true
}

func amountRawLTE(v string, threshold int64) bool {
	v = strings.TrimSpace(v)
	if v == "" {
		return false
	}
	for _, r := range v {
		if r < '0' || r > '9' {
			return false
		}
	}
	trimmed := strings.TrimLeft(v, "0")
	if trimmed == "" {
		return true
	}
	thresholdText := fmt.Sprintf("%d", threshold)
	if len(trimmed) != len(thresholdText) {
		return len(trimmed) < len(thresholdText)
	}
	return trimmed <= thresholdText
}

func commonPrefixLength(a, b string) int {
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	n := 0
	for i := 0; i < limit; i++ {
		if a[i] != b[i] {
			break
		}
		n++
	}
	return n
}

func commonSuffixLength(a, b string) int {
	i := len(a) - 1
	j := len(b) - 1
	n := 0
	for i >= 0 && j >= 0 {
		if a[i] != b[j] {
			break
		}
		n++
		i--
		j--
	}
	return n
}

func acceptedFromStats(stats walletStats) AcceptedWallet {
	return AcceptedWallet{
		Address:                         stats.address,
		ScrapeCount:                     stats.scrapeCount,
		ScrapeOutbound:                  stats.scrapeOutbound,
		ScrapeInbound:                   stats.scrapeInbound,
		SampleTransactions:              stats.sampleTransactions,
		OutboundTransfers:               stats.outboundTransfers,
		InboundTransfers:                stats.inboundTransfers,
		UniqueCounterparties:            stats.uniqueCounterparties,
		LegitOutboundCounterparties:     stats.legitOutboundCounterparties,
		ScanInboundDustTransfers:        stats.scanInboundDustTransfers,
		UniqueDustRecipients:            stats.uniqueDustRecipients,
		RepeatedInboundDustCounterparts: stats.repeatedInboundDustCounterparts,
		LookalikeInboundDustMatches:     stats.lookalikeInboundDustMatches,
		SourceScore:                     stats.sourceScore,
		DiscoveredFromSeeds:             stats.discoveredFromSeeds,
	}
}

func rejectedFromStats(stats walletStats, reason string) RejectedWallet {
	return RejectedWallet{
		Address:                         stats.address,
		Reason:                          reason,
		ScrapeCount:                     stats.scrapeCount,
		ScrapeOutbound:                  stats.scrapeOutbound,
		ScrapeInbound:                   stats.scrapeInbound,
		SampleTransactions:              stats.sampleTransactions,
		OutboundTransfers:               stats.outboundTransfers,
		InboundTransfers:                stats.inboundTransfers,
		UniqueCounterparties:            stats.uniqueCounterparties,
		LegitOutboundCounterparties:     stats.legitOutboundCounterparties,
		ScanInboundDustTransfers:        stats.scanInboundDustTransfers,
		UniqueDustRecipients:            stats.uniqueDustRecipients,
		RepeatedInboundDustCounterparts: stats.repeatedInboundDustCounterparts,
		LookalikeInboundDustMatches:     stats.lookalikeInboundDustMatches,
		SourceScore:                     stats.sourceScore,
		DiscoveredFromSeeds:             stats.discoveredFromSeeds,
	}
}

func writeAccepted(path string, accepted []AcceptedWallet) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return fmt.Errorf("create accepted output dir: %w", err)
	}
	lines := make([]string, 0, len(accepted))
	for _, a := range accepted {
		lines = append(lines, a.Address)
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

func writeRejected(path string, rejected []RejectedWallet) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return fmt.Errorf("create rejected output dir: %w", err)
	}
	var b strings.Builder
	b.WriteString("address\treason\tscrape_count\tscrape_outbound\tscrape_inbound\tsample_transactions\toutbound_transfers\tinbound_transfers\tunique_counterparties\tlegit_outbound_counterparties\tscan_inbound_dust_transfers\tunique_dust_recipients\trepeated_inbound_dust_counterparties\tlookalike_inbound_dust_matches\tsource_score\tdiscovered_from_seeds\n")
	for _, r := range rejected {
		fmt.Fprintf(
			&b,
			"%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n",
			r.Address,
			r.Reason,
			r.ScrapeCount,
			r.ScrapeOutbound,
			r.ScrapeInbound,
			r.SampleTransactions,
			r.OutboundTransfers,
			r.InboundTransfers,
			r.UniqueCounterparties,
			r.LegitOutboundCounterparties,
			r.ScanInboundDustTransfers,
			r.UniqueDustRecipients,
			r.RepeatedInboundDustCounterparts,
			r.LookalikeInboundDustMatches,
			r.SourceScore,
			r.DiscoveredFromSeeds,
		)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
