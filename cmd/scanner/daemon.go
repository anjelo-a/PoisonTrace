package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"poisontrace/internal/config"
	"poisontrace/internal/helius"
	"poisontrace/internal/pipeline"
	"poisontrace/internal/storage"
	"poisontrace/internal/walletsource"
)

const (
	freePlanRPCDefaultRPS      = 5
	freePlanEnhancedDefaultRPS = 1
	enhancedTransactionCredits = 100
	standardRPCCredits         = 1
	daemonLockKey              = 0x50545DAE
)

var errDailyCreditBudgetExhausted = errors.New("daily Helius credit budget exhausted")

type daemonOptions struct {
	once                 bool
	workDir              string
	cycleInterval        time.Duration
	dailyCreditBudget    int
	rpcRPS               int
	enhancedRPS          int
	maxHeliusRetries     int
	targetCount          int
	runMaxTXPages        int
	runMaxTXPerWallet    int
	runMaxConcurrent     int
	maxBlocks            int
	blockLookback        int
	maxTXPerBlock        int
	maxScrapedWallets    int
	maxSeedWallets       int
	maxCandidates        int
	candidateMaxPages    int
	sourceMaxTXPerWallet int
	maxAcceptedTX        int
	minNativeLamports    int64
	scanWindowDays       int
	baselineLookbackDays int
	maxSameTimestampTX   int
	maxTransfersPerTX    int
	maxUnknownDustSPL    int
	minScanInboundDust   int
	deepDiveTopN         int
	deepDiveMaxPages     int
	deepDiveMaxTX        int
	deepDiveMinScore     int
	maxNoisyInstructions int
	minOutbound          int
}

func daemonCmd(cfg config.Config, args []string) {
	opts, err := parseDaemonOptions(cfg, args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "daemon config error: %v\n", err)
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "database connection error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	store := storage.NewPostgresStore(db)
	if err := store.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "database ping error: %v\n", err)
		os.Exit(1)
	}
	release, err := acquireDaemonLock(ctx, db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "daemon lock error: %v\n", err)
		os.Exit(1)
	}
	defer release()

	rpcLimiter := newRequestLimiter(opts.rpcRPS)
	enhancedLimiter := newRequestLimiter(opts.enhancedRPS)
	budget := newDailyCreditBudget(opts.dailyCreditBudget)

	for {
		cycleCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.RunTimeoutSeconds*2)*time.Second)
		err := runDaemonCycle(cycleCtx, cfg, store, opts, rpcLimiter, enhancedLimiter, budget)
		cancel()
		if err != nil {
			if errors.Is(err, errDailyCreditBudgetExhausted) {
				log.Printf("daemon budget exhausted; sleeping until next UTC day: %v", err)
				if sleepErr := sleepUntilNextUTCDay(ctx); sleepErr != nil {
					return
				}
			} else {
				log.Printf("daemon cycle failed: %v", err)
			}
		}
		if opts.once {
			return
		}
		if err := sleepContext(ctx, opts.cycleInterval); err != nil {
			return
		}
	}
}

func parseDaemonOptions(cfg config.Config, args []string) (daemonOptions, error) {
	fs := flag.NewFlagSet("daemon", flag.ExitOnError)
	var cycleInterval daemonDuration
	cycleInterval.Duration = 24 * time.Hour
	opts := daemonOptions{}
	fs.BoolVar(&opts.once, "once", false, "run one daemon cycle and exit")
	fs.StringVar(&opts.workDir, "work-dir", "artifacts/daemon", "directory for generated wallet files")
	fs.Var(&cycleInterval, "cycle-interval", "duration between cycles, for example 24h or 6h")
	fs.IntVar(&opts.dailyCreditBudget, "daily-credit-budget", 45000, "maximum estimated Helius credits used by the daemon per UTC day")
	fs.IntVar(&opts.rpcRPS, "rpc-rps", freePlanRPCDefaultRPS, "maximum Helius RPC requests per second")
	fs.IntVar(&opts.enhancedRPS, "enhanced-rps", freePlanEnhancedDefaultRPS, "maximum Helius Enhanced API requests per second")
	fs.IntVar(&opts.maxHeliusRetries, "max-helius-retries", minInt(cfg.MaxHeliusRetries, 1), "maximum retries per Helius Enhanced API request in daemon cycles")
	fs.IntVar(&opts.targetCount, "target-count", 5, "accepted wallets per cycle")
	fs.IntVar(&opts.runMaxTXPages, "run-max-tx-pages", minInt(cfg.MaxTXPagesPerWallet, 5), "maximum Helius pages per wallet during daemon scanner runs")
	fs.IntVar(&opts.runMaxTXPerWallet, "run-max-tx-per-wallet", minInt(cfg.MaxTXPerWallet, 400), "maximum transactions per wallet during daemon scanner runs")
	fs.IntVar(&opts.runMaxConcurrent, "run-max-concurrent-wallets", 1, "maximum concurrent wallets during daemon scanner runs")
	fs.IntVar(&opts.maxBlocks, "max-blocks", 20, "maximum getBlock calls per source cycle")
	fs.IntVar(&opts.blockLookback, "block-lookback", 250, "maximum slots to inspect per source cycle")
	fs.IntVar(&opts.maxTXPerBlock, "max-tx-per-block", 200, "maximum transactions inspected per block")
	fs.IntVar(&opts.maxScrapedWallets, "max-scraped-wallets", 500, "maximum wallets discovered from block scraping")
	fs.IntVar(&opts.maxSeedWallets, "max-seed-wallets", 120, "maximum scraped seed wallets to score")
	fs.IntVar(&opts.maxCandidates, "max-candidates", 40, "maximum discovered candidate wallets to score")
	fs.IntVar(&opts.candidateMaxPages, "candidate-max-pages", 3, "maximum Helius pages sampled per candidate wallet")
	fs.IntVar(&opts.sourceMaxTXPerWallet, "source-max-tx-per-wallet", 300, "maximum transactions sampled per source candidate")
	fs.IntVar(&opts.maxAcceptedTX, "max-accepted-tx", 300, "maximum sampled transactions allowed for accepted wallets")
	fs.Int64Var(&opts.minNativeLamports, "min-native-lamports", 10000000, "minimum lamports for native SOL transfer seeds")
	fs.IntVar(&opts.scanWindowDays, "scan-window-days", cfg.ScanWindowDays, "scan window days per daemon run")
	fs.IntVar(&opts.baselineLookbackDays, "baseline-lookback-days", cfg.BaselineLookbackDays, "baseline lookback days per daemon run")
	fs.IntVar(&opts.maxSameTimestampTX, "max-same-timestamp-tx", 5, "maximum transactions with identical timestamp allowed")
	fs.IntVar(&opts.maxTransfersPerTX, "max-transfers-per-tx", 4, "maximum owner-level transfers involving candidate in one transaction")
	fs.IntVar(&opts.maxUnknownDustSPL, "max-unknown-dust-spl", 0, "maximum non-zero SPL transfers without a configured dust threshold allowed")
	fs.IntVar(&opts.minScanInboundDust, "min-scan-inbound-dust", 1, "minimum zero/dust inbound transfers in the scan window required for daemon sourced wallets")
	fs.IntVar(&opts.deepDiveTopN, "deep-dive-top-n", 5, "number of high-signal capped wallets to retry with deeper sampling")
	fs.IntVar(&opts.deepDiveMaxPages, "deep-dive-max-pages", 8, "maximum Helius pages for deep-dive wallet sampling")
	fs.IntVar(&opts.deepDiveMaxTX, "deep-dive-max-tx", 900, "maximum sampled transactions for deep-dive wallet sampling")
	fs.IntVar(&opts.deepDiveMinScore, "deep-dive-min-score", 30, "minimum source score required to qualify for deep-dive retries")
	fs.IntVar(&opts.maxNoisyInstructions, "max-noisy-instructions", 0, "maximum non-transfer instruction types allowed in scraped transactions")
	fs.IntVar(&opts.minOutbound, "min-outbound", 1, "minimum resolved outbound transfers required")
	if err := fs.Parse(args); err != nil {
		return daemonOptions{}, err
	}
	opts.cycleInterval = cycleInterval.Duration
	return opts, validateDaemonOptions(cfg, opts)
}

func validateDaemonOptions(cfg config.Config, opts daemonOptions) error {
	if opts.workDir == "" {
		return fmt.Errorf("work-dir is required")
	}
	if opts.cycleInterval < time.Minute && !opts.once {
		return fmt.Errorf("cycle-interval must be >= 1m")
	}
	if opts.dailyCreditBudget < 1 {
		return fmt.Errorf("daily-credit-budget must be >= 1")
	}
	if opts.rpcRPS < 1 || opts.rpcRPS > 10 {
		return fmt.Errorf("rpc-rps must be between 1 and 10 for Helius Free")
	}
	if opts.enhancedRPS < 1 || opts.enhancedRPS > 2 {
		return fmt.Errorf("enhanced-rps must be between 1 and 2 for Helius Free")
	}
	if opts.maxHeliusRetries < 0 || opts.maxHeliusRetries > cfg.MaxHeliusRetries {
		return fmt.Errorf("max-helius-retries must be between 0 and MAX_HELIUS_RETRIES")
	}
	if opts.targetCount < 1 || opts.targetCount > cfg.MaxWalletsPerRun {
		return fmt.Errorf("target-count must be between 1 and MAX_WALLETS_PER_RUN")
	}
	if opts.runMaxTXPages < 1 || opts.runMaxTXPages > cfg.MaxTXPagesPerWallet {
		return fmt.Errorf("run-max-tx-pages must be between 1 and MAX_TX_PAGES_PER_WALLET")
	}
	if opts.runMaxTXPerWallet < 1 || opts.runMaxTXPerWallet > cfg.MaxTXPerWallet {
		return fmt.Errorf("run-max-tx-per-wallet must be between 1 and MAX_TX_PER_WALLET")
	}
	if opts.runMaxConcurrent < 1 || opts.runMaxConcurrent > cfg.MaxConcurrentWallets || opts.runMaxConcurrent > opts.targetCount {
		return fmt.Errorf("run-max-concurrent-wallets must be between 1 and both MAX_CONCURRENT_WALLETS and target-count")
	}
	if opts.maxBlocks < 1 || opts.blockLookback < 1 || opts.maxTXPerBlock < 1 {
		return fmt.Errorf("block scrape bounds must be >= 1")
	}
	if opts.maxScrapedWallets < 1 || opts.maxSeedWallets < 1 || opts.maxCandidates < 1 {
		return fmt.Errorf("source wallet bounds must be >= 1")
	}
	if opts.candidateMaxPages < 1 || opts.sourceMaxTXPerWallet < 1 || opts.maxAcceptedTX < 1 {
		return fmt.Errorf("source sampling bounds must be >= 1")
	}
	if opts.deepDiveTopN < 0 || opts.deepDiveMaxPages < 1 || opts.deepDiveMaxTX < 1 || opts.deepDiveMinScore < 1 {
		return fmt.Errorf("deep-dive bounds must satisfy top-n>=0, max-pages>=1, max-tx>=1, min-score>=1")
	}
	if opts.scanWindowDays < 1 || opts.baselineLookbackDays <= opts.scanWindowDays {
		return fmt.Errorf("baseline-lookback-days must be greater than scan-window-days")
	}
	if estimateDaemonCycleCredits(cfg, opts) > opts.dailyCreditBudget {
		return fmt.Errorf("daily-credit-budget is below one cycle worst-case estimate (%d credits)", estimateDaemonCycleCredits(cfg, opts))
	}
	return nil
}

func runDaemonCycle(ctx context.Context, cfg config.Config, store *storage.PostgresStore, opts daemonOptions, rpcLimiter, enhancedLimiter *requestLimiter, budget *dailyCreditBudget) error {
	estimate := estimateDaemonCycleCredits(cfg, opts)
	if err := budget.Reserve(estimate); err != nil {
		return err
	}

	cycleID := time.Now().UTC().Format("20060102T150405Z")
	cycleDir := filepath.Join(opts.workDir, cycleID)
	if err := os.MkdirAll(cycleDir, 0o755); err != nil {
		return fmt.Errorf("create daemon cycle dir: %w", err)
	}

	scanEnd := time.Now().UTC()
	scanStart := scanEnd.Add(-time.Duration(opts.scanWindowDays) * 24 * time.Hour)
	acceptedPath := filepath.Join(cycleDir, "accepted_wallets.txt")
	rejectedPath := filepath.Join(cycleDir, "rejected_wallets.tsv")

	log.Printf("daemon cycle %s starting: target_wallets=%d estimated_credits=%d rpc_rps=%d enhanced_rps=%d", cycleID, opts.targetCount, estimate, opts.rpcRPS, opts.enhancedRPS)

	rpcClient, err := walletsource.NewHeliusRPCClient(cfg.HeliusAPIKey, 15*time.Second)
	if err != nil {
		return fmt.Errorf("helius rpc client: %w", err)
	}
	scrapedWallets, err := walletsource.ScrapeRecentWallets(ctx, limitedRPC{inner: rpcClient, limiter: rpcLimiter}, walletsource.ScrapeOptions{
		BlockLookback:        opts.blockLookback,
		MaxBlocks:            opts.maxBlocks,
		MaxWallets:           opts.maxScrapedWallets,
		MaxTXPerBlock:        opts.maxTXPerBlock,
		MaxNoisyInstructions: opts.maxNoisyInstructions,
		MinNativeLamports:    opts.minNativeLamports,
	})
	if err != nil {
		return fmt.Errorf("scrape wallets: %w", err)
	}
	if len(scrapedWallets) == 0 {
		log.Printf("daemon cycle %s found no scraped wallets", cycleID)
		return nil
	}

	enhancedClient, err := helius.NewHTTPClient(cfg.HeliusBaseURL, cfg.HeliusAPIKey, 15*time.Second)
	if err != nil {
		return fmt.Errorf("helius enhanced client: %w", err)
	}
	limitedEnhanced := limitedEnhancedClient{inner: enhancedClient, limiter: enhancedLimiter}
	sourceResult, err := walletsource.Source(ctx, limitedEnhanced, walletsource.Options{
		SeedWallets:        scrapedWallets,
		ScoreSeedWallets:   true,
		DiscoverNeighbors:  false,
		OutPath:            acceptedPath,
		RejectedOutPath:    rejectedPath,
		ScanStart:          scanStart,
		ScanEnd:            scanEnd,
		BaselineLookback:   time.Duration(opts.baselineLookbackDays) * 24 * time.Hour,
		TargetCount:        opts.targetCount,
		MaxSeedWallets:     opts.maxSeedWallets,
		MaxCandidates:      opts.maxCandidates,
		CandidateMaxPages:  opts.candidateMaxPages,
		MaxTXPerWallet:     opts.sourceMaxTXPerWallet,
		MaxRetries:         opts.maxHeliusRetries,
		RequestDelay:       time.Duration(cfg.HeliusRequestDelayMS) * time.Millisecond,
		MinOutbound:        opts.minOutbound,
		MaxAcceptedTX:      opts.maxAcceptedTX,
		MaxSameTimestampTX: opts.maxSameTimestampTX,
		MaxTransfersPerTX:  opts.maxTransfersPerTX,
		MaxUnknownDustSPL:  opts.maxUnknownDustSPL,
		MinScanInboundDust: opts.minScanInboundDust,
		DeepDiveTopN:       opts.deepDiveTopN,
		DeepDiveMaxPages:   opts.deepDiveMaxPages,
		DeepDiveMaxTX:      opts.deepDiveMaxTX,
		DeepDiveMinScore:   opts.deepDiveMinScore,
	})
	if err != nil {
		return fmt.Errorf("source wallets: %w", err)
	}
	if len(sourceResult.Accepted) == 0 {
		log.Printf("daemon cycle %s accepted no wallets; rejected=%d", cycleID, len(sourceResult.Rejected))
		return nil
	}

	runCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.RunTimeoutSeconds)*time.Second)
	defer cancel()
	runCfg := cfg
	runCfg.MaxWalletsPerRun = opts.targetCount
	runCfg.MaxTXPagesPerWallet = opts.runMaxTXPages
	runCfg.MaxTXPerWallet = opts.runMaxTXPerWallet
	runCfg.MaxConcurrentWallets = opts.runMaxConcurrent
	runCfg.MaxHeliusRetries = opts.maxHeliusRetries

	orch := pipeline.NewOrchestrator(
		runCfg,
		pipeline.WithRunRepository(store),
		pipeline.WithWalletLockRepository(store),
		pipeline.WithWalletRunner(pipeline.NewWalletExecutionRunner(runCfg, limitedEnhanced, store)),
	)
	err = orch.Run(runCtx, pipeline.RunParams{
		WalletFile:            acceptedPath,
		ScanStart:             scanStart,
		ScanEnd:               scanEnd,
		BaselineLookbackDays:  opts.baselineLookbackDays,
		RequestedByCLICommand: "scanner daemon",
	})
	if err != nil {
		return fmt.Errorf("scanner run: %w", err)
	}
	log.Printf("daemon cycle %s complete: accepted=%d rejected=%d dir=%s", cycleID, len(sourceResult.Accepted), len(sourceResult.Rejected), cycleDir)
	return nil
}

func estimateDaemonCycleCredits(cfg config.Config, opts daemonOptions) int {
	rpcCalls := 1 + opts.maxBlocks
	sourceEnhancedCalls := opts.maxCandidates * opts.candidateMaxPages
	deepDiveEnhancedCalls := opts.deepDiveTopN * opts.deepDiveMaxPages
	runEnhancedCalls := opts.targetCount * opts.runMaxTXPages * 2
	enhancedAttempts := opts.maxHeliusRetries + 1
	return rpcCalls*standardRPCCredits + (sourceEnhancedCalls+deepDiveEnhancedCalls+runEnhancedCalls)*enhancedAttempts*enhancedTransactionCredits
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func acquireDaemonLock(ctx context.Context, db *sql.DB) (func(), error) {
	var ok bool
	if err := db.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", daemonLockKey).Scan(&ok); err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("another scanner daemon already holds the database lock")
	}
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = db.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", daemonLockKey)
	}, nil
}

type daemonDuration struct {
	time.Duration
}

func (d *daemonDuration) String() string {
	return d.Duration.String()
}

func (d *daemonDuration) Set(value string) error {
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return err
	}
	d.Duration = parsed
	return nil
}

type requestLimiter struct {
	interval time.Duration
	mu       sync.Mutex
	next     time.Time
}

func newRequestLimiter(rps int) *requestLimiter {
	if rps < 1 {
		rps = 1
	}
	return &requestLimiter{interval: time.Second / time.Duration(rps)}
}

func (l *requestLimiter) Wait(ctx context.Context) error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	now := time.Now()
	when := now
	if l.next.After(now) {
		when = l.next
	}
	l.next = when.Add(l.interval)
	l.mu.Unlock()
	return sleepUntil(ctx, when)
}

type dailyCreditBudget struct {
	limit int
	mu    sync.Mutex
	day   string
	used  int
}

func newDailyCreditBudget(limit int) *dailyCreditBudget {
	return &dailyCreditBudget{limit: limit}
}

func (b *dailyCreditBudget) Reserve(credits int) error {
	if b == nil || b.limit <= 0 {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	day := time.Now().UTC().Format("2006-01-02")
	if b.day != day {
		b.day = day
		b.used = 0
	}
	if b.used+credits > b.limit {
		return fmt.Errorf("%w: used=%d requested=%d limit=%d", errDailyCreditBudgetExhausted, b.used, credits, b.limit)
	}
	b.used += credits
	return nil
}

type limitedEnhancedClient struct {
	inner   helius.Client
	limiter *requestLimiter
}

func (c limitedEnhancedClient) FetchEnhancedPage(ctx context.Context, walletAddress string, before string) (helius.EnhancedPage, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return helius.EnhancedPage{}, err
	}
	return c.inner.FetchEnhancedPage(ctx, walletAddress, before)
}

type limitedRPC struct {
	inner   walletsource.RPC
	limiter *requestLimiter
}

func (r limitedRPC) GetSlot(ctx context.Context) (int64, error) {
	if err := r.limiter.Wait(ctx); err != nil {
		return 0, err
	}
	return r.inner.GetSlot(ctx)
}

func (r limitedRPC) GetBlock(ctx context.Context, slot int64) (walletsource.RPCBlock, error) {
	if err := r.limiter.Wait(ctx); err != nil {
		return walletsource.RPCBlock{}, err
	}
	return r.inner.GetBlock(ctx, slot)
}

func sleepContext(ctx context.Context, d time.Duration) error {
	return sleepUntil(ctx, time.Now().Add(d))
}

func sleepUntil(ctx context.Context, at time.Time) error {
	delay := time.Until(at)
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func sleepUntilNextUTCDay(ctx context.Context) error {
	now := time.Now().UTC()
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	return sleepUntil(ctx, next)
}
