package pipeline

import (
	"fmt"
	"strings"
)

func ValidateCoreSyncParams(p CoreSyncParams) error {
	if strings.TrimSpace(p.FocalWalletAddress) == "" {
		return fmt.Errorf("focal wallet address is required")
	}
	if !p.BaselineStart.Before(p.BaselineEnd) || !p.ScanStart.Before(p.ScanEnd) {
		return fmt.Errorf("invalid baseline/scan windows")
	}
	if !p.BaselineEnd.Equal(p.ScanStart) {
		return fmt.Errorf("baseline end must equal scan start")
	}
	if p.MaxTXPagesPerWallet < 1 {
		return fmt.Errorf("max tx pages per wallet must be >= 1")
	}
	if p.MaxTXPerWallet < 1 {
		return fmt.Errorf("max tx per wallet must be >= 1")
	}
	if p.MaxHeliusRetries < 0 {
		return fmt.Errorf("max helius retries must be >= 0")
	}
	if p.HeliusRequestDelay < 0 {
		return fmt.Errorf("helius request delay must be >= 0")
	}

	return validateLookalikeAndInjectionParams(
		p.LookalikeRecencyDays,
		p.LookalikePrefixMin,
		p.LookalikeSuffixMin,
		p.LookalikeSingleSideMin,
		p.MinInjectionCount,
	)
}

func ValidateCandidateMaterializeParams(p CandidateMaterializeParams) error {
	return validateLookalikeAndInjectionParams(
		p.LookalikeRecencyDays,
		p.LookalikePrefixMin,
		p.LookalikeSuffixMin,
		p.LookalikeSingleSideMin,
		p.MinInjectionCount,
	)
}

func validateLookalikeAndInjectionParams(recencyDays, prefixMin, suffixMin, singleSideMin, minInjectionCount int) error {
	if recencyDays < 1 {
		return fmt.Errorf("lookalike recency days must be >= 1")
	}
	if prefixMin < 1 || suffixMin < 1 || singleSideMin < 1 {
		return fmt.Errorf("lookalike thresholds must be >= 1")
	}
	if prefixMin < 4 || suffixMin < 4 || singleSideMin < 6 {
		return fmt.Errorf("lookalike thresholds must satisfy phase1 minimums: prefix>=4, suffix>=4, single-side>=6")
	}
	if minInjectionCount < 2 {
		return fmt.Errorf("min injection count must be >= 2")
	}
	return nil
}
