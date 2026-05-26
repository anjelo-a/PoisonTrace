package main

import (
	"testing"
	"time"

	"poisontrace/internal/config"
)

func TestDaemonDefaultsFitFreePlanBudget(t *testing.T) {
	cfg := config.Config{
		MaxWalletsPerRun:         10,
		MaxTXPagesPerWallet:      20,
		MaxTXPerWallet:           1500,
		MaxConcurrentWallets:     2,
		RunTimeoutSeconds:        900,
		BaselineLookbackDays:     90,
		ScanWindowDays:           7,
		HeliusRequestDelayMS:     150,
		MaxHeliusRetries:         4,
		LookalikeRecencyDays:     30,
		LookalikePrefixMin:       4,
		LookalikeSuffixMin:       4,
		LookalikeSingleSideMin:   6,
		MinInjectionCount:        2,
		DustThresholdsSeedPath:   "data/seeds/asset_thresholds.seed.sql",
		WalletSyncTimeoutSeconds: 180,
	}

	opts, err := parseDaemonOptions(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if opts.rpcRPS != freePlanRPCDefaultRPS {
		t.Fatalf("unexpected rpc rps %d", opts.rpcRPS)
	}
	if opts.enhancedRPS != freePlanEnhancedDefaultRPS {
		t.Fatalf("unexpected enhanced rps %d", opts.enhancedRPS)
	}
	if opts.cycleInterval != 24*time.Hour {
		t.Fatalf("unexpected cycle interval %s", opts.cycleInterval)
	}
	if got := estimateDaemonCycleCredits(opts); got > opts.dailyCreditBudget {
		t.Fatalf("estimate %d exceeds daily budget %d", got, opts.dailyCreditBudget)
	}
}

func TestDaemonRejectsFreePlanLimitOverflow(t *testing.T) {
	cfg := config.Config{
		MaxWalletsPerRun:     10,
		MaxTXPagesPerWallet:  5,
		MaxTXPerWallet:       400,
		MaxConcurrentWallets: 1,
		BaselineLookbackDays: 90,
		ScanWindowDays:       7,
	}

	if _, err := parseDaemonOptions(cfg, []string{"--enhanced-rps", "3"}); err == nil {
		t.Fatal("expected enhanced-rps above free-plan limit to fail")
	}
	if _, err := parseDaemonOptions(cfg, []string{"--rpc-rps", "11"}); err == nil {
		t.Fatal("expected rpc-rps above free-plan limit to fail")
	}
}
