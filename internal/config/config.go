package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Config captures all hard operational bounds for Phase 0–1.
type Config struct {
	DatabaseURL              string
	HeliusAPIKey             string
	HeliusBaseURL            string
	MaxWalletsPerRun         int
	MaxTXPagesPerWallet      int
	MaxTXPerWallet           int
	MaxConcurrentWallets     int
	WalletSyncTimeoutSeconds int
	RunTimeoutSeconds        int
	HeliusRequestDelayMS     int
	MaxHeliusRetries         int
	BaselineLookbackDays     int
	ScanWindowDays           int
	LookalikeRecencyDays     int
	LookalikePrefixMin       int
	LookalikeSuffixMin       int
	LookalikeSingleSideMin   int
	MinInjectionCount        int
	DustThresholdsSeedPath   string
}

func LoadFromEnv() (Config, error) {
	maxWalletsPerRun, err := getEnvInt("MAX_WALLETS_PER_RUN", 25)
	if err != nil {
		return Config{}, err
	}
	maxTXPagesPerWallet, err := getEnvInt("MAX_TX_PAGES_PER_WALLET", 20)
	if err != nil {
		return Config{}, err
	}
	maxTXPerWallet, err := getEnvInt("MAX_TX_PER_WALLET", 1500)
	if err != nil {
		return Config{}, err
	}
	maxConcurrentWallets, err := getEnvInt("MAX_CONCURRENT_WALLETS", 2)
	if err != nil {
		return Config{}, err
	}
	walletSyncTimeoutSeconds, err := getEnvInt("WALLET_SYNC_TIMEOUT_SECONDS", 180)
	if err != nil {
		return Config{}, err
	}
	runTimeoutSeconds, err := getEnvInt("RUN_TIMEOUT_SECONDS", 900)
	if err != nil {
		return Config{}, err
	}
	heliusRequestDelayMS, err := getEnvInt("HELIUS_REQUEST_DELAY_MS", 150)
	if err != nil {
		return Config{}, err
	}
	maxHeliusRetries, err := getEnvInt("MAX_HELIUS_RETRIES", 4)
	if err != nil {
		return Config{}, err
	}
	baselineLookbackDays, err := getEnvInt("BASELINE_LOOKBACK_DAYS", 90)
	if err != nil {
		return Config{}, err
	}
	scanWindowDays, err := getEnvInt("SCAN_WINDOW_DAYS", 7)
	if err != nil {
		return Config{}, err
	}
	lookalikeRecencyDays, err := getEnvInt("LOOKALIKE_RECENCY_DAYS", 30)
	if err != nil {
		return Config{}, err
	}
	lookalikePrefixMin, err := getEnvInt("LOOKALIKE_PREFIX_MIN", 4)
	if err != nil {
		return Config{}, err
	}
	lookalikeSuffixMin, err := getEnvInt("LOOKALIKE_SUFFIX_MIN", 4)
	if err != nil {
		return Config{}, err
	}
	lookalikeSingleSideMin, err := getEnvInt("LOOKALIKE_SINGLE_SIDE_MIN", 6)
	if err != nil {
		return Config{}, err
	}
	minInjectionCount, err := getEnvInt("MIN_INJECTION_COUNT", 2)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		DatabaseURL:              os.Getenv("DATABASE_URL"),
		HeliusAPIKey:             os.Getenv("HELIUS_API_KEY"),
		HeliusBaseURL:            getEnv("HELIUS_BASE_URL", "https://api.helius.xyz/v0"),
		MaxWalletsPerRun:         maxWalletsPerRun,
		MaxTXPagesPerWallet:      maxTXPagesPerWallet,
		MaxTXPerWallet:           maxTXPerWallet,
		MaxConcurrentWallets:     maxConcurrentWallets,
		WalletSyncTimeoutSeconds: walletSyncTimeoutSeconds,
		RunTimeoutSeconds:        runTimeoutSeconds,
		HeliusRequestDelayMS:     heliusRequestDelayMS,
		MaxHeliusRetries:         maxHeliusRetries,
		BaselineLookbackDays:     baselineLookbackDays,
		ScanWindowDays:           scanWindowDays,
		LookalikeRecencyDays:     lookalikeRecencyDays,
		LookalikePrefixMin:       lookalikePrefixMin,
		LookalikeSuffixMin:       lookalikeSuffixMin,
		LookalikeSingleSideMin:   lookalikeSingleSideMin,
		MinInjectionCount:        minInjectionCount,
		DustThresholdsSeedPath:   getEnv("DUST_THRESHOLDS_SEED_PATH", "data/seeds/asset_thresholds.seed.sql"),
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	if _, err := url.ParseRequestURI(c.DatabaseURL); err != nil {
		return fmt.Errorf("DATABASE_URL must be a valid URI: %w", err)
	}
	if c.HeliusAPIKey == "" {
		return errors.New("HELIUS_API_KEY is required")
	}
	if strings.EqualFold(strings.TrimSpace(c.HeliusAPIKey), "replace_me") {
		return errors.New("HELIUS_API_KEY must not be a placeholder value")
	}
	if c.HeliusBaseURL == "" {
		return errors.New("HELIUS_BASE_URL is required")
	}
	parsedHeliusBaseURL, err := url.ParseRequestURI(c.HeliusBaseURL)
	if err != nil {
		return fmt.Errorf("HELIUS_BASE_URL must be a valid URI: %w", err)
	}
	if !strings.EqualFold(parsedHeliusBaseURL.Scheme, "https") {
		return errors.New("HELIUS_BASE_URL must use https")
	}
	if c.MaxWalletsPerRun < 1 || c.MaxTXPagesPerWallet < 1 || c.MaxTXPerWallet < 1 {
		return errors.New("wallet/page/tx caps must be >= 1")
	}
	if c.MaxConcurrentWallets < 1 {
		return errors.New("MAX_CONCURRENT_WALLETS must be >= 1")
	}
	if c.MaxConcurrentWallets > c.MaxWalletsPerRun {
		return errors.New("MAX_CONCURRENT_WALLETS must be <= MAX_WALLETS_PER_RUN")
	}
	if c.WalletSyncTimeoutSeconds < 10 || c.RunTimeoutSeconds < 10 {
		return errors.New("timeouts must be >= 10 seconds")
	}
	if c.RunTimeoutSeconds < c.WalletSyncTimeoutSeconds {
		return errors.New("RUN_TIMEOUT_SECONDS must be >= WALLET_SYNC_TIMEOUT_SECONDS")
	}
	if c.HeliusRequestDelayMS < 0 {
		return errors.New("HELIUS_REQUEST_DELAY_MS must be >= 0")
	}
	if c.MaxHeliusRetries < 0 {
		return errors.New("MAX_HELIUS_RETRIES must be >= 0")
	}
	if c.BaselineLookbackDays < 1 || c.ScanWindowDays < 1 {
		return errors.New("window days must be >= 1")
	}
	if c.BaselineLookbackDays <= c.ScanWindowDays {
		return errors.New("BASELINE_LOOKBACK_DAYS must be > SCAN_WINDOW_DAYS")
	}
	if c.LookalikeRecencyDays < 1 {
		return errors.New("LOOKALIKE_RECENCY_DAYS must be >= 1")
	}
	if c.LookalikePrefixMin < 1 || c.LookalikeSuffixMin < 1 || c.LookalikeSingleSideMin < 1 {
		return errors.New("lookalike thresholds must be >= 1")
	}
	if c.LookalikePrefixMin < 4 || c.LookalikeSuffixMin < 4 || c.LookalikeSingleSideMin < 6 {
		return errors.New("lookalike thresholds must satisfy phase1 minimums: prefix>=4, suffix>=4, single-side>=6")
	}
	if c.MinInjectionCount < 2 {
		return errors.New("MIN_INJECTION_COUNT must be >= 2")
	}
	if strings.TrimSpace(c.DustThresholdsSeedPath) == "" {
		return errors.New("DUST_THRESHOLDS_SEED_PATH is required")
	}
	return nil
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func getEnvInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid integer env %s=%q", key, v)
	}
	return n, nil
}
