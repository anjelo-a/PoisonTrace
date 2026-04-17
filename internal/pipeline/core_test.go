package pipeline

import (
	"context"
	"strings"
	"testing"
	"time"

	"poisontrace/internal/helius"
	"poisontrace/internal/transactions"
)

type scriptedClient struct {
	responses []helius.EnhancedPage
	errs      []error
	idx       int
}

func (s *scriptedClient) FetchEnhancedPage(_ context.Context, _ string, _ string) (helius.EnhancedPage, error) {
	i := s.idx
	s.idx++
	if i < len(s.errs) && s.errs[i] != nil {
		return helius.EnhancedPage{}, s.errs[i]
	}
	if i < len(s.responses) {
		return s.responses[i], nil
	}
	return helius.EnhancedPage{}, nil
}

func nativeTx(sig string, at time.Time, from, to, amount string) helius.EnhancedTransaction {
	return helius.EnhancedTransaction{
		Signature:     sig,
		TimestampUnix: at.Unix(),
		NativeTransfers: []helius.NativeTransfer{
			{
				FromUserAccount: from,
				ToUserAccount:   to,
				Amount:          amount,
			},
		},
	}
}

func TestRunWalletCoreSyncEndToEndCandidateEmission(t *testing.T) {
	baselineStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	baselineEnd := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	scanStart := baselineEnd
	scanEnd := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)

	client := &scriptedClient{
		responses: []helius.EnhancedPage{
			{
				Transactions: []helius.EnhancedTransaction{
					nativeTx("b1", time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC), "walletA", "LegitABCDxyzz", "1000"),
				},
			},
			{}, // baseline second page empty
			{
				Transactions: []helius.EnhancedTransaction{
					nativeTx("s1", time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC), "LegitWXYZxyzz", "walletA", "0"),
					nativeTx("s2", time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC), "LegitWXYZxyzz", "walletA", "0"),
				},
			},
			{}, // scan second page empty
		},
	}

	res, err := RunWalletCoreSync(context.Background(), client, CoreSyncParams{
		FocalWalletAddress:     "walletA",
		BaselineStart:          baselineStart,
		BaselineEnd:            baselineEnd,
		ScanStart:              scanStart,
		ScanEnd:                scanEnd,
		MaxTXPagesPerWallet:    5,
		MaxTXPerWallet:         100,
		MaxHeliusRetries:       1,
		HeliusRequestDelay:     0,
		LookalikeRecencyDays:   30,
		LookalikePrefixMin:     4,
		LookalikeSuffixMin:     4,
		LookalikeSingleSideMin: 6,
		MinInjectionCount:      2,
		ClassifyDust: func(tr transactions.NormalizedTransfer) transactions.DustStatus {
			if tr.AmountRaw == "0" {
				return transactions.DustTrue
			}
			return transactions.DustFalse
		},
	})
	if err != nil {
		t.Fatalf("core sync failed: %v", err)
	}
	if !res.BaselineComplete {
		t.Fatal("expected complete baseline")
	}
	if res.IncompleteWindow {
		t.Fatalf("did not expect incomplete window, reason=%s", res.UnknownGateReason)
	}
	if len(res.Candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(res.Candidates))
	}
	cp := res.Counterparties["LegitWXYZxyzz"]
	if cp.InboundCount != 2 {
		t.Fatalf("expected suspicious inbound counterparty count 2, got %d", cp.InboundCount)
	}
}

func TestRunWalletCoreSyncBaselineTruncationForcesIncompleteAndSuppressesCandidates(t *testing.T) {
	baselineStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	baselineEnd := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	scanStart := baselineEnd
	scanEnd := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)

	client := &scriptedClient{
		responses: []helius.EnhancedPage{
			{
				Transactions: []helius.EnhancedTransaction{
					nativeTx("b1", time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC), "walletA", "LegitABCDxyzz", "1000"),
				},
				Before: "next",
			},
			{
				Transactions: []helius.EnhancedTransaction{
					nativeTx("s1", time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC), "LegitWXYZxyzz", "walletA", "0"),
					nativeTx("s2", time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC), "LegitWXYZxyzz", "walletA", "0"),
				},
			},
			{},
		},
	}

	res, err := RunWalletCoreSync(context.Background(), client, CoreSyncParams{
		FocalWalletAddress:     "walletA",
		BaselineStart:          baselineStart,
		BaselineEnd:            baselineEnd,
		ScanStart:              scanStart,
		ScanEnd:                scanEnd,
		MaxTXPagesPerWallet:    1,
		MaxTXPerWallet:         100,
		MaxHeliusRetries:       1,
		HeliusRequestDelay:     0,
		LookalikeRecencyDays:   30,
		LookalikePrefixMin:     4,
		LookalikeSuffixMin:     4,
		LookalikeSingleSideMin: 6,
		MinInjectionCount:      2,
		ClassifyDust: func(tr transactions.NormalizedTransfer) transactions.DustStatus {
			if tr.AmountRaw == "0" {
				return transactions.DustTrue
			}
			return transactions.DustFalse
		},
	})
	if err != nil {
		t.Fatalf("core sync failed: %v", err)
	}
	if res.BaselineComplete {
		t.Fatal("expected baseline to be incomplete under page cap")
	}
	if !res.IncompleteWindow {
		t.Fatal("expected incomplete window when baseline is truncated")
	}
	if len(res.Candidates) != 0 {
		t.Fatalf("expected candidate suppression, got %d", len(res.Candidates))
	}
	if res.BaselineTruncation == "" || res.UnknownGateReason == "" {
		t.Fatalf("expected truncation and reason to be persisted, got truncation=%q reason=%q", res.BaselineTruncation, res.UnknownGateReason)
	}
}

func TestRunWalletCoreSyncRejectsInvalidLookalikeThresholds(t *testing.T) {
	t.Parallel()

	baseParams := CoreSyncParams{
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
			name: "recency_non_positive",
			mutate: func(p *CoreSyncParams) {
				p.LookalikeRecencyDays = 0
			},
			wantInErr: "lookalike recency days must be >= 1",
		},
		{
			name: "threshold_non_positive",
			mutate: func(p *CoreSyncParams) {
				p.LookalikePrefixMin = -1
			},
			wantInErr: "lookalike thresholds must be >= 1",
		},
		{
			name: "phase1_minimums_not_met",
			mutate: func(p *CoreSyncParams) {
				p.LookalikePrefixMin = 3
			},
			wantInErr: "lookalike thresholds must satisfy phase1 minimums",
		},
		{
			name: "min_injection_non_positive",
			mutate: func(p *CoreSyncParams) {
				p.MinInjectionCount = 1
			},
			wantInErr: "min injection count must be >= 2",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			params := baseParams
			tc.mutate(&params)
			_, err := RunWalletCoreSync(context.Background(), nil, params)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tc.wantInErr) {
				t.Fatalf("expected error containing %q, got %q", tc.wantInErr, err.Error())
			}
		})
	}
}

func TestRunWalletCoreSyncRejectsNilClient(t *testing.T) {
	t.Parallel()

	_, err := RunWalletCoreSync(context.Background(), nil, CoreSyncParams{
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
	})
	if err == nil {
		t.Fatal("expected nil client validation error")
	}
	if !strings.Contains(err.Error(), "helius client is required") {
		t.Fatalf("expected helius client error, got %q", err.Error())
	}
}
