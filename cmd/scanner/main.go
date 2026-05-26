package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"poisontrace/internal/api"
	"poisontrace/internal/config"
	"poisontrace/internal/exports"
	"poisontrace/internal/fixtures"
	"poisontrace/internal/helius"
	"poisontrace/internal/pipeline"
	"poisontrace/internal/storage"
	"poisontrace/internal/walletsource"

	_ "github.com/lib/pq"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "run":
		cfg, err := config.LoadFromEnv()
		if err != nil {
			fmt.Fprintf(os.Stderr, "config error: %v\n", err)
			os.Exit(1)
		}
		runCmd(cfg, os.Args[2:])
	case "source-wallets":
		cfg, err := config.LoadFromEnv()
		if err != nil {
			fmt.Fprintf(os.Stderr, "config error: %v\n", err)
			os.Exit(1)
		}
		sourceWalletsCmd(cfg, os.Args[2:])
	case "daemon":
		cfg, err := config.LoadFromEnv()
		if err != nil {
			fmt.Fprintf(os.Stderr, "config error: %v\n", err)
			os.Exit(1)
		}
		daemonCmd(cfg, os.Args[2:])
	case "replay-fixture":
		replayFixtureCmd(os.Args[2:])
	case "validate-corpus":
		validateCorpusCmd(os.Args[2:])
	case "export-dataset":
		cfg, err := config.LoadFromEnv()
		if err != nil {
			fmt.Fprintf(os.Stderr, "config error: %v\n", err)
			os.Exit(1)
		}
		exportDatasetCmd(cfg, os.Args[2:])
	case "serve-api":
		cfg, err := config.LoadReadOnlyAPIFromEnv()
		if err != nil {
			fmt.Fprintf(os.Stderr, "config error: %v\n", err)
			os.Exit(1)
		}
		serveAPICmd(cfg, os.Args[2:])
	default:
		printUsage()
		os.Exit(2)
	}
}

func runCmd(cfg config.Config, args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	walletFile := fs.String("wallets", "", "path to wallet list (one address per line)")
	skipWalletFile := fs.String("skip-wallets", "", "optional path to wallet list to exclude from this run")
	scanStart := fs.String("scan-start", "", "scan window start in RFC3339")
	scanEnd := fs.String("scan-end", "", "scan window end in RFC3339")
	baselineLookbackDays := fs.Int("baseline-lookback-days", cfg.BaselineLookbackDays, "baseline lookback days")
	_ = fs.Parse(args)

	if *walletFile == "" || *scanStart == "" || *scanEnd == "" {
		fmt.Fprintln(os.Stderr, "missing required flags: --wallets --scan-start --scan-end")
		fs.Usage()
		os.Exit(2)
	}

	startAt, err := time.Parse(time.RFC3339, *scanStart)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid --scan-start: %v\n", err)
		os.Exit(2)
	}
	endAt, err := time.Parse(time.RFC3339, *scanEnd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid --scan-end: %v\n", err)
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.RunTimeoutSeconds)*time.Second)
	defer cancel()

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

	heliusClient, err := helius.NewHTTPClient(cfg.HeliusBaseURL, cfg.HeliusAPIKey, 15*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "helius client error: %v\n", err)
		os.Exit(1)
	}

	orch := pipeline.NewOrchestrator(
		cfg,
		pipeline.WithRunRepository(store),
		pipeline.WithWalletLockRepository(store),
		pipeline.WithWalletRunner(pipeline.NewWalletExecutionRunner(cfg, heliusClient, store)),
	)
	err = orch.Run(ctx, pipeline.RunParams{
		WalletFile:            *walletFile,
		SkipWalletFile:        *skipWalletFile,
		ScanStart:             startAt.UTC(),
		ScanEnd:               endAt.UTC(),
		BaselineLookbackDays:  *baselineLookbackDays,
		RequestedByCLICommand: "scanner run",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "run failed: %v\n", err)
		os.Exit(1)
	}
}

func sourceWalletsCmd(cfg config.Config, args []string) {
	fs := flag.NewFlagSet("source-wallets", flag.ExitOnError)
	seedWalletFile := fs.String("seed-wallets", "", "path to seed wallet list used to discover counterparties")
	scrapeBlocks := fs.Bool("scrape-blocks", false, "discover seed wallets from recent finalized Solana blocks")
	startSlot := fs.Int64("start-slot", 0, "slot to start block scraping from (default latest finalized slot)")
	blockLookback := fs.Int("block-lookback", 100, "maximum slots to inspect when scraping blocks")
	maxBlocks := fs.Int("max-blocks", 100, "maximum getBlock calls when scraping blocks")
	maxScrapedWallets := fs.Int("max-scraped-wallets", 100, "maximum wallets discovered from block scraping")
	maxTXPerBlock := fs.Int("max-tx-per-block", 200, "maximum transactions inspected per block")
	maxNoisyInstructions := fs.Int("max-noisy-instructions", 0, "maximum non-transfer instruction types allowed in scraped transactions")
	minNativeLamports := fs.Int64("min-native-lamports", 10000000, "minimum lamports for native SOL transfer seeds")
	outPath := fs.String("out", "", "path to write accepted wallet addresses")
	rejectedOutPath := fs.String("rejected-out", "", "path to write rejected wallet TSV with reasons")
	scanStart := fs.String("scan-start", "", "scan window start in RFC3339")
	scanEnd := fs.String("scan-end", "", "scan window end in RFC3339")
	baselineLookbackDays := fs.Int("baseline-lookback-days", cfg.BaselineLookbackDays, "baseline lookback days")
	targetCount := fs.Int("target-count", 30, "maximum accepted wallet count")
	maxSeedWallets := fs.Int("max-seed-wallets", 10, "maximum seed wallets to sample")
	maxCandidates := fs.Int("max-candidates", 100, "maximum discovered candidates to score")
	seedMaxPages := fs.Int("seed-max-pages", 3, "maximum Helius pages per seed wallet")
	candidateMaxPages := fs.Int("candidate-max-pages", cfg.MaxTXPagesPerWallet, "maximum Helius pages per candidate wallet")
	maxTXPerWallet := fs.Int("max-tx-per-wallet", cfg.MaxTXPerWallet, "maximum transactions sampled per wallet")
	minOutbound := fs.Int("min-outbound", 1, "minimum resolved outbound transfers required")
	maxAcceptedTX := fs.Int("max-accepted-tx", cfg.MaxTXPerWallet, "maximum sampled transactions allowed for accepted wallets")
	maxSameTimestampTX := fs.Int("max-same-timestamp-tx", 5, "maximum transactions with identical timestamp allowed")
	maxTransfersPerTX := fs.Int("max-transfers-per-tx", 4, "maximum owner-level transfers involving candidate in one transaction")
	maxUnknownDustSPL := fs.Int("max-unknown-dust-spl", 0, "maximum non-zero SPL transfers without a configured dust threshold allowed")
	minScanInboundDust := fs.Int("min-scan-inbound-dust", 0, "minimum zero/dust inbound transfers in the scan window required for sourced wallets")
	deepDiveTopN := fs.Int("deep-dive-top-n", 0, "number of high-signal capped wallets to retry with deeper sampling")
	deepDiveMaxPages := fs.Int("deep-dive-max-pages", 0, "maximum Helius pages for deep-dive wallet sampling (0 uses default)")
	deepDiveMaxTX := fs.Int("deep-dive-max-tx", 0, "maximum sampled transactions for deep-dive wallet sampling (0 uses default)")
	deepDiveMinScore := fs.Int("deep-dive-min-score", 0, "minimum source score required to qualify for deep-dive retries (0 uses default)")
	knownDustAssets := fs.String("known-dust-assets", strings.Join(walletsource.DefaultKnownDustAssetKeys(), ","), "comma-separated asset keys with configured dust thresholds")
	_ = fs.Parse(args)

	if !*scrapeBlocks && *seedWalletFile == "" {
		fmt.Fprintln(os.Stderr, "missing required seed source: provide --seed-wallets or --scrape-blocks")
		fs.Usage()
		os.Exit(2)
	}
	if *outPath == "" || *rejectedOutPath == "" || *scanStart == "" || *scanEnd == "" {
		fmt.Fprintln(os.Stderr, "missing required flags: --out --rejected-out --scan-start --scan-end")
		fs.Usage()
		os.Exit(2)
	}

	startAt, err := time.Parse(time.RFC3339, *scanStart)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid --scan-start: %v\n", err)
		os.Exit(2)
	}
	endAt, err := time.Parse(time.RFC3339, *scanEnd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid --scan-end: %v\n", err)
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.RunTimeoutSeconds)*time.Second)
	defer cancel()

	heliusClient, err := helius.NewHTTPClient(cfg.HeliusBaseURL, cfg.HeliusAPIKey, 15*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "helius client error: %v\n", err)
		os.Exit(1)
	}

	var scrapedWallets []string
	if *scrapeBlocks {
		rpcClient, rpcErr := walletsource.NewHeliusRPCClient(cfg.HeliusAPIKey, 15*time.Second)
		if rpcErr != nil {
			fmt.Fprintf(os.Stderr, "helius rpc client error: %v\n", rpcErr)
			os.Exit(1)
		}
		scrapedWallets, err = walletsource.ScrapeRecentWallets(ctx, rpcClient, walletsource.ScrapeOptions{
			StartSlot:            *startSlot,
			BlockLookback:        *blockLookback,
			MaxBlocks:            *maxBlocks,
			MaxWallets:           *maxScrapedWallets,
			MaxTXPerBlock:        *maxTXPerBlock,
			MaxNoisyInstructions: *maxNoisyInstructions,
			MinNativeLamports:    *minNativeLamports,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "scrape wallets failed: %v\n", err)
			os.Exit(1)
		}
		if len(scrapedWallets) == 0 && *seedWalletFile == "" {
			fmt.Fprintln(os.Stderr, "scrape wallets found no seed wallets")
			os.Exit(1)
		}
		fmt.Printf("scraped seed wallets: %d\n", len(scrapedWallets))
	}

	result, err := walletsource.Source(ctx, heliusClient, walletsource.Options{
		SeedWalletFile:     *seedWalletFile,
		SeedWallets:        scrapedWallets,
		ScoreSeedWallets:   *scrapeBlocks,
		DiscoverNeighbors:  !*scrapeBlocks,
		OutPath:            *outPath,
		RejectedOutPath:    *rejectedOutPath,
		ScanStart:          startAt.UTC(),
		ScanEnd:            endAt.UTC(),
		BaselineLookback:   time.Duration(*baselineLookbackDays) * 24 * time.Hour,
		TargetCount:        *targetCount,
		MaxSeedWallets:     *maxSeedWallets,
		MaxCandidates:      *maxCandidates,
		SeedMaxPages:       *seedMaxPages,
		CandidateMaxPages:  *candidateMaxPages,
		MaxTXPerWallet:     *maxTXPerWallet,
		MaxRetries:         cfg.MaxHeliusRetries,
		RequestDelay:       time.Duration(cfg.HeliusRequestDelayMS) * time.Millisecond,
		MinOutbound:        *minOutbound,
		MaxAcceptedTX:      *maxAcceptedTX,
		MaxSameTimestampTX: *maxSameTimestampTX,
		MaxTransfersPerTX:  *maxTransfersPerTX,
		MaxUnknownDustSPL:  *maxUnknownDustSPL,
		MinScanInboundDust: *minScanInboundDust,
		KnownDustAssetKeys: splitCSV(*knownDustAssets),
		DeepDiveTopN:       *deepDiveTopN,
		DeepDiveMaxPages:   *deepDiveMaxPages,
		DeepDiveMaxTX:      *deepDiveMaxTX,
		DeepDiveMinScore:   *deepDiveMinScore,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "source wallets failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("accepted wallets: %d -> %s\n", len(result.Accepted), *outPath)
	fmt.Printf("rejected wallets: %d -> %s\n", len(result.Rejected), *rejectedOutPath)
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func replayFixtureCmd(args []string) {
	fs := flag.NewFlagSet("replay-fixture", flag.ExitOnError)
	fixture := fs.String("fixture", "", "fixture case id under data/fixtures")
	fixturesRoot := fs.String("fixtures-root", "data/fixtures", "fixtures root directory")
	writeExpected := fs.Bool("write-expected", false, "write replay output to expected/*.json instead of validating")
	_ = fs.Parse(args)
	if *fixture == "" {
		fmt.Fprintln(os.Stderr, "missing required flag: --fixture")
		fs.Usage()
		os.Exit(2)
	}

	fx, err := fixtures.LoadCase(*fixturesRoot, *fixture)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load fixture failed: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	out, err := fixtures.Replay(ctx, fx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "replay fixture failed: %v\n", err)
		os.Exit(1)
	}

	if *writeExpected {
		if err := fixtures.WriteExpected(fx, out); err != nil {
			fmt.Fprintf(os.Stderr, "write expected failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("fixture expected files updated: %s\n", *fixture)
		return
	}

	if err := fixtures.CompareExpected(fx, out); err != nil {
		fmt.Fprintf(os.Stderr, "fixture mismatch: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf(
		"fixture replay ok: %s (wallets=%d tx_inserted=%d candidates=%d)\n",
		*fixture,
		len(out.WalletSyncRuns),
		out.IngestionRunDelta.TransactionsInserted,
		out.IngestionRunDelta.PoisoningCandidatesInserted,
	)
}

func validateCorpusCmd(args []string) {
	fs := flag.NewFlagSet("validate-corpus", flag.ExitOnError)
	fixturesRoot := fs.String("fixtures-root", "data/fixtures", "fixtures root directory")
	reportOut := fs.String("report-out", "", "optional path to write JSON validation report")
	strictMissReason := fs.Bool("strict-miss-reason", false, "fail run when expected_miss_reason is unsupported or not evidenced")
	_ = fs.Parse(args)

	report, err := fixtures.ValidateCorpus(context.Background(), *fixturesRoot, fixtures.CorpusValidationOptions{
		StrictMissReason: *strictMissReason,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "validate corpus failed: %v\n", err)
		os.Exit(1)
	}

	if *reportOut != "" {
		raw, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "encode report failed: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*reportOut, append(raw, '\n'), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "write report failed: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf(
		"corpus validation: cases=%d passed=%d failed=%d recall=%.3f false_positive_rate=%.3f\n",
		report.Summary.TotalCases,
		report.Summary.PassedCases,
		report.Summary.FailedCases,
		report.Summary.CaseLevelRecall,
		report.Summary.CaseLevelFalsePositiveRate,
	)
	if report.Summary.FailedCases > 0 {
		for _, c := range report.Cases {
			if c.Passed {
				continue
			}
			fmt.Printf("  FAIL %s: %s\n", c.CaseID, strings.Join(c.Errors, ","))
		}
		os.Exit(1)
	}
}

func exportDatasetCmd(cfg config.Config, args []string) {
	fs := flag.NewFlagSet("export-dataset", flag.ExitOnError)
	outDir := fs.String("out-dir", "", "output directory for JSONL artifacts + manifest")
	runID := fs.Int64("run-id", 0, "ingestion run id to export")
	startedAtFrom := fs.String("started-at-from", "", "inclusive ingestion_run.started_at lower bound (RFC3339)")
	startedAtTo := fs.String("started-at-to", "", "exclusive ingestion_run.started_at upper bound (RFC3339)")
	_ = fs.Parse(args)

	if *outDir == "" {
		fmt.Fprintln(os.Stderr, "missing required flag: --out-dir")
		fs.Usage()
		os.Exit(2)
	}

	var filter storage.ExportFilter
	if *runID > 0 {
		filter.RunID = runID
	}
	if *startedAtFrom != "" || *startedAtTo != "" {
		if *startedAtFrom == "" || *startedAtTo == "" {
			fmt.Fprintln(os.Stderr, "both --started-at-from and --started-at-to are required for time-window export")
			os.Exit(2)
		}
		from, err := time.Parse(time.RFC3339, *startedAtFrom)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --started-at-from: %v\n", err)
			os.Exit(2)
		}
		to, err := time.Parse(time.RFC3339, *startedAtTo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --started-at-to: %v\n", err)
			os.Exit(2)
		}
		if !from.Before(to) {
			fmt.Fprintln(os.Stderr, "invalid range: started-at-from must be before started-at-to")
			os.Exit(2)
		}
		filter.StartedAtFrom = &from
		filter.StartedAtTo = &to
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

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

	result, err := exports.ExportDataset(ctx, store, exports.ExportOptions{
		OutDir: *outDir,
		Filter: filter,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "export dataset failed: %v\n", err)
		os.Exit(1)
	}

	for _, file := range result.Manifest.Files {
		fmt.Printf("exported %s rows=%d sha256=%s\n", file.Name, file.RowCount, file.SHA256)
	}
	fmt.Printf("manifest: %s/report_manifest.json\n", strings.TrimRight(*outDir, "/"))
}

func printUsage() {
	fmt.Println(`PoisonTrace scanner

Usage:
  scanner run --wallets <path> --scan-start <RFC3339> --scan-end <RFC3339> [--baseline-lookback-days N]
  scanner source-wallets (--seed-wallets <path> | --scrape-blocks) --out <path> --rejected-out <path> --scan-start <RFC3339> --scan-end <RFC3339>
  scanner daemon [--once] [--cycle-interval 24h] [--daily-credit-budget 45000]
  scanner replay-fixture --fixture <case_id> [--fixtures-root data/fixtures] [--write-expected]
  scanner validate-corpus [--fixtures-root data/fixtures] [--report-out path] [--strict-miss-reason]
  scanner export-dataset --out-dir <dir> [--run-id N | --started-at-from <RFC3339> --started-at-to <RFC3339>]
  scanner serve-api [--addr :8080]`)
}

func serveAPICmd(cfg config.Config, args []string) {
	fs := flag.NewFlagSet("serve-api", flag.ExitOnError)
	addr := fs.String("addr", ":8080", "HTTP bind address")
	_ = fs.Parse(args)

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "database connection error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	store := storage.NewPostgresStore(db)
	if err := store.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "database ping error: %v\n", err)
		os.Exit(1)
	}

	srv := api.NewServer(store, cfg).WithRunStarter(func(ctx context.Context, req api.RunStartRequest) (api.RunStartResult, error) {
		runCfg := cfg
		if override, ok, err := store.GetConfigOverride(ctx); err != nil {
			return api.RunStartResult{}, err
		} else if ok {
			runCfg = applyStoredConfigOverride(runCfg, override)
		}
		if err := runCfg.Validate(); err != nil {
			return api.RunStartResult{}, err
		}

		heliusClient, err := helius.NewHTTPClient(runCfg.HeliusBaseURL, runCfg.HeliusAPIKey, 15*time.Second)
		if err != nil {
			return api.RunStartResult{}, err
		}
		runID, err := store.CreateIngestionRun(ctx, time.Now().UTC())
		if err != nil {
			return api.RunStartResult{}, err
		}
		orch := pipeline.NewOrchestrator(
			runCfg,
			pipeline.WithRunRepository(store),
			pipeline.WithWalletLockRepository(store),
			pipeline.WithWalletRunner(pipeline.NewWalletExecutionRunner(runCfg, heliusClient, store)),
		)
		go func() {
			runCtx, cancel := context.WithTimeout(context.Background(), time.Duration(runCfg.RunTimeoutSeconds)*time.Second)
			defer cancel()
			err := orch.Run(runCtx, pipeline.RunParams{
				WalletAddresses:       req.WalletAddresses,
				ScanStart:             req.ScanStart.UTC(),
				ScanEnd:               req.ScanEnd.UTC(),
				BaselineLookbackDays:  req.BaselineLookbackDays,
				RequestedByCLICommand: req.RequestedBy,
				IngestionRunID:        runID,
			})
			if err != nil {
				log.Printf("manual run %d failed: %v", runID, err)
			}
		}()
		return api.RunStartResult{
			RunID:                runID,
			WalletCount:          len(req.WalletAddresses),
			ScanStart:            req.ScanStart.UTC(),
			ScanEnd:              req.ScanEnd.UTC(),
			BaselineLookbackDays: req.BaselineLookbackDays,
		}, nil
	})
	log.Printf("api server listening on %s", *addr)

	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- httpServer.ListenAndServe()
	}()

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-sigCtx.Done():
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "api server shutdown failed: %v\n", err)
			os.Exit(1)
		}
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "api server failed: %v\n", err)
			os.Exit(1)
		}
	}
}

func applyStoredConfigOverride(cfg config.Config, override storage.ConfigOverrideRecord) config.Config {
	if override.MaxWalletsPerRun != nil {
		cfg.MaxWalletsPerRun = *override.MaxWalletsPerRun
	}
	if override.MaxTXPagesPerWallet != nil {
		cfg.MaxTXPagesPerWallet = *override.MaxTXPagesPerWallet
	}
	if override.MaxTXPerWallet != nil {
		cfg.MaxTXPerWallet = *override.MaxTXPerWallet
	}
	if override.MaxConcurrentWallets != nil {
		cfg.MaxConcurrentWallets = *override.MaxConcurrentWallets
	}
	if override.WalletSyncTimeoutSeconds != nil {
		cfg.WalletSyncTimeoutSeconds = *override.WalletSyncTimeoutSeconds
	}
	if override.RunTimeoutSeconds != nil {
		cfg.RunTimeoutSeconds = *override.RunTimeoutSeconds
	}
	if override.MaxHeliusRetries != nil {
		cfg.MaxHeliusRetries = *override.MaxHeliusRetries
	}
	if override.HeliusRequestDelayMS != nil {
		cfg.HeliusRequestDelayMS = *override.HeliusRequestDelayMS
	}
	if override.BaselineLookbackDays != nil {
		cfg.BaselineLookbackDays = *override.BaselineLookbackDays
	}
	if override.ScanWindowDays != nil {
		cfg.ScanWindowDays = *override.ScanWindowDays
	}
	if override.LookalikeRecencyDays != nil {
		cfg.LookalikeRecencyDays = *override.LookalikeRecencyDays
	}
	if override.LookalikePrefixMin != nil {
		cfg.LookalikePrefixMin = *override.LookalikePrefixMin
	}
	if override.LookalikeSuffixMin != nil {
		cfg.LookalikeSuffixMin = *override.LookalikeSuffixMin
	}
	if override.LookalikeSingleSideMin != nil {
		cfg.LookalikeSingleSideMin = *override.LookalikeSingleSideMin
	}
	if override.MinInjectionCount != nil {
		cfg.MinInjectionCount = *override.MinInjectionCount
	}
	return cfg
}
