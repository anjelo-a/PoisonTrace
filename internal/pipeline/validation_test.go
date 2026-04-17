package pipeline

import (
	"strings"
	"testing"
	"time"
)

func TestValidateCoreSyncParamsAcceptsPhase1MinimumBoundaries(t *testing.T) {
	t.Parallel()

	err := ValidateCoreSyncParams(CoreSyncParams{
		FocalWalletAddress:     "walletA",
		BaselineStart:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		BaselineEnd:            time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		ScanStart:              time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		ScanEnd:                time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
		MaxTXPagesPerWallet:    1,
		MaxTXPerWallet:         1,
		MaxHeliusRetries:       0,
		HeliusRequestDelay:     0,
		LookalikeRecencyDays:   1,
		LookalikePrefixMin:     4,
		LookalikeSuffixMin:     4,
		LookalikeSingleSideMin: 6,
		MinInjectionCount:      2,
	})
	if err != nil {
		t.Fatalf("expected valid boundary params, got %v", err)
	}
}

func TestValidateCoreSyncParamsRejectsInvalidRuntimeBounds(t *testing.T) {
	t.Parallel()

	base := CoreSyncParams{
		FocalWalletAddress:     "walletA",
		BaselineStart:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		BaselineEnd:            time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		ScanStart:              time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		ScanEnd:                time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
		MaxTXPagesPerWallet:    5,
		MaxTXPerWallet:         100,
		MaxHeliusRetries:       1,
		HeliusRequestDelay:     0,
		LookalikeRecencyDays:   30,
		LookalikePrefixMin:     4,
		LookalikeSuffixMin:     4,
		LookalikeSingleSideMin: 6,
		MinInjectionCount:      2,
	}

	tests := []struct {
		name      string
		mutate    func(*CoreSyncParams)
		wantInErr string
	}{
		{
			name: "max_pages_non_positive",
			mutate: func(p *CoreSyncParams) {
				p.MaxTXPagesPerWallet = 0
			},
			wantInErr: "max tx pages per wallet must be >= 1",
		},
		{
			name: "max_tx_non_positive",
			mutate: func(p *CoreSyncParams) {
				p.MaxTXPerWallet = 0
			},
			wantInErr: "max tx per wallet must be >= 1",
		},
		{
			name: "negative_retries",
			mutate: func(p *CoreSyncParams) {
				p.MaxHeliusRetries = -1
			},
			wantInErr: "max helius retries must be >= 0",
		},
		{
			name: "negative_delay",
			mutate: func(p *CoreSyncParams) {
				p.HeliusRequestDelay = -1 * time.Millisecond
			},
			wantInErr: "helius request delay must be >= 0",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params := base
			tc.mutate(&params)
			err := ValidateCoreSyncParams(params)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tc.wantInErr) {
				t.Fatalf("expected error containing %q, got %q", tc.wantInErr, err.Error())
			}
		})
	}
}
