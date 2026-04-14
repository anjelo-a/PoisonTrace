package pipeline

import (
	"strings"
	"testing"
	"time"

	"poisontrace/internal/counterparties"
	"poisontrace/internal/transactions"
)

func obs(sig string, idx int, at time.Time, relation counterparties.RelationType, cp string, amount string, dust transactions.DustStatus) WalletTransferObservation {
	return WalletTransferObservation{
		RelationType:        relation,
		CounterpartyAddress: cp,
		Transfer: transactions.NormalizedTransfer{
			Signature:           sig,
			TransferIndex:       idx,
			BlockTime:           at,
			AmountRaw:           amount,
			DustStatus:          dust,
			TokenMint:           "So11111111111111111111111111111111111111112",
			AssetType:           transactions.AssetTypeNativeSOL,
			NormalizationStatus: transactions.NormalizationResolved,
			PoisoningEligible:   true,
		},
	}
}

func defaultMaterializeParams() CandidateMaterializeParams {
	return CandidateMaterializeParams{
		BaselineComplete:       true,
		LookalikeRecencyDays:   30,
		LookalikePrefixMin:     4,
		LookalikeSuffixMin:     4,
		LookalikeSingleSideMin: 6,
		MinInjectionCount:      2,
	}
}

func TestMaterializeCandidatesEmitsWithAllGatesAndMinTwo(t *testing.T) {
	params := defaultMaterializeParams()
	baseAt := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	scanAt1 := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	scanAt2 := time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC)

	baseline := []WalletTransferObservation{
		obs("b1", 0, baseAt, counterparties.RelationSender, "LegitABCDxyzz", "10000", transactions.DustFalse),
	}
	scan := []WalletTransferObservation{
		obs("s1", 0, scanAt1, counterparties.RelationReceiver, "LegitWXYZxyzz", "0", transactions.DustFalse),
		obs("s2", 1, scanAt2, counterparties.RelationReceiver, "LegitWXYZxyzz", "0", transactions.DustFalse),
	}

	result := MaterializeCandidates(baseline, scan, params)
	if result.IncompleteWindow {
		t.Fatalf("did not expect incomplete window, reason=%s", result.UnknownGateReason)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(result.Candidates))
	}
	for _, c := range result.Candidates {
		if c.RepeatInjectionCount != 2 {
			t.Fatalf("expected repeat injection count 2, got %d", c.RepeatInjectionCount)
		}
		if c.SuspiciousCounterparty != "LegitWXYZxyzz" || c.MatchedLegitCounterparty != "LegitABCDxyzz" {
			t.Fatalf("unexpected candidate counterparties: %#v", c)
		}
	}
}

func TestMaterializeCandidatesBlocksWhenBaselineIncompleteAndMarksUnknown(t *testing.T) {
	params := defaultMaterializeParams()
	params.BaselineComplete = false
	at := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	scan := []WalletTransferObservation{
		obs("s1", 0, at, counterparties.RelationReceiver, "LegitWXYZxyzz", "0", transactions.DustFalse),
		obs("s2", 1, at.Add(2*time.Hour), counterparties.RelationReceiver, "LegitWXYZxyzz", "0", transactions.DustFalse),
	}

	result := MaterializeCandidates(nil, scan, params)
	if len(result.Candidates) != 0 {
		t.Fatalf("expected no candidates when baseline incomplete, got %d", len(result.Candidates))
	}
	if !result.IncompleteWindow {
		t.Fatal("expected incomplete window due unknown required gates")
	}
	if !strings.Contains(result.UnknownGateReason, "is_new_counterparty") {
		t.Fatalf("expected unknown gate reason to include new counterparty, got %q", result.UnknownGateReason)
	}
}

func TestMaterializeCandidatesMinInjectionUnknownWhenDustUnknownCouldMeetThreshold(t *testing.T) {
	params := defaultMaterializeParams()
	baseAt := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	scanAt := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	baseline := []WalletTransferObservation{
		obs("b1", 0, baseAt, counterparties.RelationSender, "LegitABCDxyzz", "10000", transactions.DustFalse),
	}
	scan := []WalletTransferObservation{
		obs("s1", 0, scanAt, counterparties.RelationReceiver, "LegitWXYZxyzz", "0", transactions.DustFalse),
		obs("s2", 1, scanAt.Add(1*time.Hour), counterparties.RelationReceiver, "LegitWXYZxyzz", "5", transactions.DustUnknown),
	}

	result := MaterializeCandidates(baseline, scan, params)
	if len(result.Candidates) != 0 {
		t.Fatalf("expected no emitted candidates under min-count unknown, got %d", len(result.Candidates))
	}
	if !result.IncompleteWindow {
		t.Fatal("expected incomplete window for unknown min injection gate")
	}
	if !strings.Contains(result.UnknownGateReason, "min_injection_count_met") {
		t.Fatalf("expected unknown reason to include min injection gate, got %q", result.UnknownGateReason)
	}
}

func TestMaterializeCandidatesDoesNotTreatInboundOnlyBaselineAsLegit(t *testing.T) {
	params := defaultMaterializeParams()
	at := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	baseline := []WalletTransferObservation{
		obs("b1", 0, at.Add(-24*time.Hour), counterparties.RelationReceiver, "LegitABCDxyzz", "10000", transactions.DustFalse),
	}
	scan := []WalletTransferObservation{
		obs("s1", 0, at, counterparties.RelationReceiver, "LegitWXYZxyzz", "0", transactions.DustFalse),
		obs("s2", 1, at.Add(2*time.Hour), counterparties.RelationReceiver, "LegitWXYZxyzz", "0", transactions.DustFalse),
	}

	result := MaterializeCandidates(baseline, scan, params)
	if result.IncompleteWindow {
		t.Fatalf("did not expect incomplete window, reason=%s", result.UnknownGateReason)
	}
	if len(result.Candidates) != 0 {
		t.Fatalf("expected no candidates without legit outbound baseline, got %d", len(result.Candidates))
	}
}
