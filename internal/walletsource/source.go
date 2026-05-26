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
	ScoreSeedWallets  bool
	DiscoverNeighbors bool
	OutPath           string
	RejectedOutPath   string
	ScanStart         time.Time
	ScanEnd           time.Time
	BaselineLookback  time.Duration

	TargetCount        int
	MaxSeedWallets     int
	MaxCandidates      int
	SeedMaxPages       int
	CandidateMaxPages  int
	MaxTXPerWallet     int
	MaxRetries         int
	RequestDelay       time.Duration
	MinOutbound        int
	MaxAcceptedTX      int
	MaxSameTimestampTX int
	MaxTransfersPerTX  int
	MaxUnknownDustSPL  int
	KnownDustAssetKeys []string
}

type Result struct {
	Accepted []AcceptedWallet
	Rejected []RejectedWallet
}

type AcceptedWallet struct {
	Address              string
	SampleTransactions   int
	OutboundTransfers    int
	InboundTransfers     int
	UniqueCounterparties int
	DiscoveredFromSeeds  int
}

type RejectedWallet struct {
	Address              string
	Reason               string
	SampleTransactions   int
	OutboundTransfers    int
	InboundTransfers     int
	UniqueCounterparties int
	DiscoveredFromSeeds  int
}

type candidateDiscovery struct {
	address string
	seeds   map[string]struct{}
}

type walletStats struct {
	address              string
	sampleTransactions   int
	outboundTransfers    int
	inboundTransfers     int
	uniqueCounterparties int
	maxSameTimestampTX   int
	maxTransfersPerTX    int
	unknownDustSPL       int
	discoveredFromSeeds  int
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
	knownDustAssetKeys := knownDustAssetSet(opts.KnownDustAssetKeys)

	result := Result{
		Accepted: make([]AcceptedWallet, 0, opts.TargetCount),
		Rejected: make([]RejectedWallet, 0, len(candidates)),
	}
	for _, c := range candidates {
		if opts.TargetCount > 0 && len(result.Accepted) >= opts.TargetCount {
			result.Rejected = append(result.Rejected, RejectedWallet{
				Address:             c.address,
				Reason:              "target_count_reached",
				DiscoveredFromSeeds: len(c.seeds),
			})
			continue
		}

		page, fetchErr := pipeline.FetchEnhancedWindow(ctx, client, c.address, pipeline.FetchWindowParams{
			Start:        windowStart,
			End:          opts.ScanEnd.UTC(),
			MaxPages:     opts.CandidateMaxPages,
			MaxTx:        opts.MaxTXPerWallet,
			MaxRetries:   opts.MaxRetries,
			RequestDelay: opts.RequestDelay,
		})
		if fetchErr != nil {
			result.Rejected = append(result.Rejected, RejectedWallet{
				Address:             c.address,
				Reason:              "fetch_error",
				DiscoveredFromSeeds: len(c.seeds),
			})
			continue
		}
		stats := summarizeWallet(c.address, page.Transactions, knownDustAssetKeys)
		stats.discoveredFromSeeds = len(c.seeds)

		if page.Partial {
			result.Rejected = append(result.Rejected, rejectedFromStats(stats, "activity_cap_reached:"+page.TruncationCode))
			continue
		}
		if page.RetryExhausted {
			result.Rejected = append(result.Rejected, rejectedFromStats(stats, "helius_retry_exhausted"))
			continue
		}
		if stats.sampleTransactions == 0 {
			result.Rejected = append(result.Rejected, rejectedFromStats(stats, "no_sample_transactions"))
			continue
		}
		if stats.sampleTransactions > opts.MaxAcceptedTX {
			result.Rejected = append(result.Rejected, rejectedFromStats(stats, "too_many_sample_transactions"))
			continue
		}
		if stats.outboundTransfers < opts.MinOutbound {
			result.Rejected = append(result.Rejected, rejectedFromStats(stats, "insufficient_outbound_history"))
			continue
		}
		if stats.maxSameTimestampTX > opts.MaxSameTimestampTX {
			result.Rejected = append(result.Rejected, rejectedFromStats(stats, "bursty_same_timestamp_activity"))
			continue
		}
		if stats.maxTransfersPerTX > opts.MaxTransfersPerTX {
			result.Rejected = append(result.Rejected, rejectedFromStats(stats, "batch_transfer_activity"))
			continue
		}
		if stats.unknownDustSPL > opts.MaxUnknownDustSPL {
			result.Rejected = append(result.Rejected, rejectedFromStats(stats, "unknown_dust_spl_activity"))
			continue
		}

		result.Accepted = append(result.Accepted, AcceptedWallet{
			Address:              stats.address,
			SampleTransactions:   stats.sampleTransactions,
			OutboundTransfers:    stats.outboundTransfers,
			InboundTransfers:     stats.inboundTransfers,
			UniqueCounterparties: stats.uniqueCounterparties,
			DiscoveredFromSeeds:  len(c.seeds),
		})
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
	if opts.MaxAcceptedTX <= 0 {
		opts.MaxAcceptedTX = opts.MaxTXPerWallet
	}
	if opts.MaxSameTimestampTX <= 0 {
		opts.MaxSameTimestampTX = 5
	}
	if opts.MaxTransfersPerTX <= 0 {
		opts.MaxTransfersPerTX = 4
	}
	if len(opts.KnownDustAssetKeys) == 0 {
		opts.KnownDustAssetKeys = DefaultKnownDustAssetKeys()
	}
	if !opts.ScoreSeedWallets && !opts.DiscoverNeighbors {
		opts.DiscoverNeighbors = true
	}
	return opts
}

func DefaultKnownDustAssetKeys() []string {
	return []string{
		"SOL",
		"So11111111111111111111111111111111111111112",
		"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
		"Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB",
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

func summarizeWallet(address string, txs []helius.EnhancedTransaction, knownDustAssetKeys map[string]struct{}) walletStats {
	stats := walletStats{address: address, sampleTransactions: len(txs)}
	counterparties := make(map[string]struct{})
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
				}
				if transferNeedsKnownDustThreshold(tr, knownDustAssetKeys) {
					stats.unknownDustSPL++
				}
			case tr.DestinationOwnerAddress == address:
				stats.inboundTransfers++
				transfersInTx++
				if tr.SourceOwnerAddress != "" && tr.SourceOwnerAddress != address {
					counterparties[tr.SourceOwnerAddress] = struct{}{}
				}
				if transferNeedsKnownDustThreshold(tr, knownDustAssetKeys) {
					stats.unknownDustSPL++
				}
			}
		}
		if transfersInTx > stats.maxTransfersPerTX {
			stats.maxTransfersPerTX = transfersInTx
		}
	}
	stats.uniqueCounterparties = len(counterparties)
	return stats
}

func knownDustAssetSet(keys []string) map[string]struct{} {
	out := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = struct{}{}
	}
	return out
}

func transferNeedsKnownDustThreshold(tr transactions.NormalizedTransfer, knownDustAssetKeys map[string]struct{}) bool {
	if tr.AssetType != transactions.AssetTypeSPLFungible {
		return false
	}
	if amountRawIsZero(tr.AmountRaw) {
		return false
	}
	_, ok := knownDustAssetKeys[tr.AssetKey]
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

func rejectedFromStats(stats walletStats, reason string) RejectedWallet {
	return RejectedWallet{
		Address:              stats.address,
		Reason:               reason,
		SampleTransactions:   stats.sampleTransactions,
		OutboundTransfers:    stats.outboundTransfers,
		InboundTransfers:     stats.inboundTransfers,
		UniqueCounterparties: stats.uniqueCounterparties,
		DiscoveredFromSeeds:  stats.discoveredFromSeeds,
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
	b.WriteString("address\treason\tsample_transactions\toutbound_transfers\tinbound_transfers\tunique_counterparties\tdiscovered_from_seeds\n")
	for _, r := range rejected {
		fmt.Fprintf(
			&b,
			"%s\t%s\t%d\t%d\t%d\t%d\t%d\n",
			r.Address,
			r.Reason,
			r.SampleTransactions,
			r.OutboundTransfers,
			r.InboundTransfers,
			r.UniqueCounterparties,
			r.DiscoveredFromSeeds,
		)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
