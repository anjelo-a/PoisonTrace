package config

import "testing"

func validConfig() Config {
	return Config{
		DatabaseURL:              "postgres://postgres:postgres@localhost:5432/poisontrace?sslmode=disable",
		HeliusAPIKey:             "helius_key_real",
		HeliusBaseURL:            "https://api.helius.xyz/v0",
		MaxWalletsPerRun:         25,
		MaxTXPagesPerWallet:      20,
		MaxTXPerWallet:           1500,
		MaxConcurrentWallets:     2,
		WalletSyncTimeoutSeconds: 180,
		RunTimeoutSeconds:        900,
		HeliusRequestDelayMS:     150,
		MaxHeliusRetries:         4,
		BaselineLookbackDays:     90,
		ScanWindowDays:           7,
		LookalikeRecencyDays:     30,
		LookalikePrefixMin:       4,
		LookalikeSuffixMin:       4,
		LookalikeSingleSideMin:   6,
		MinInjectionCount:        2,
		DustThresholdsSeedPath:   "data/seeds/asset_thresholds.seed.sql",
	}
}

func TestValidateAcceptsValidConfig(t *testing.T) {
	cfg := validConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}
}

func TestValidateRejectsPlaceholderHeliusKey(t *testing.T) {
	cfg := validConfig()
	cfg.HeliusAPIKey = "replace_me"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for placeholder helius key")
	}
}

func TestValidateRejectsRunTimeoutLessThanWalletTimeout(t *testing.T) {
	cfg := validConfig()
	cfg.RunTimeoutSeconds = 120
	cfg.WalletSyncTimeoutSeconds = 180
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when run timeout is less than wallet timeout")
	}
}

func TestValidateRejectsBaselineNotGreaterThanScanWindow(t *testing.T) {
	cfg := validConfig()
	cfg.BaselineLookbackDays = 7
	cfg.ScanWindowDays = 7
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when baseline window is not greater than scan window")
	}
}

func TestValidateRejectsNonHTTPSHeliusBaseURL(t *testing.T) {
	cfg := validConfig()
	cfg.HeliusBaseURL = "http://api.helius.xyz/v0"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when HELIUS_BASE_URL is not https")
	}
}
