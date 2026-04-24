package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"poisontrace/internal/config"
	"poisontrace/internal/fixtures"
	"poisontrace/internal/helius"
	"poisontrace/internal/pipeline"
	"poisontrace/internal/storage"

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
	case "replay-fixture":
		replayFixtureCmd(os.Args[2:])
	case "validate-corpus":
		validateCorpusCmd(os.Args[2:])
	default:
		printUsage()
		os.Exit(2)
	}
}

func runCmd(cfg config.Config, args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	walletFile := fs.String("wallets", "", "path to wallet list (one address per line)")
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

func printUsage() {
	fmt.Println(`PoisonTrace scanner

Usage:
  scanner run --wallets <path> --scan-start <RFC3339> --scan-end <RFC3339> [--baseline-lookback-days N]
  scanner replay-fixture --fixture <case_id> [--fixtures-root data/fixtures] [--write-expected]
  scanner validate-corpus [--fixtures-root data/fixtures] [--report-out path] [--strict-miss-reason]`)
}
