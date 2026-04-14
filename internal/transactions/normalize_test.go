package transactions

import (
	"testing"
	"time"

	"poisontrace/internal/helius"
)

func TestNormalizeEnhancedTxSPLUnresolvedOwnerHandling(t *testing.T) {
	tx := helius.EnhancedTransaction{
		Signature:     "sig1",
		TimestampUnix: time.Now().Unix(),
		TokenTransfers: []helius.TokenTransfer{
			{
				FromUserAccount:  "ownerA",
				ToUserAccount:    "",
				FromTokenAccount: "tokenA",
				ToTokenAccount:   "tokenB",
				Mint:             "mintA",
				TokenAmount:      helius.TokenAmount{Amount: "1"},
				TokenStandard:    "Fungible",
			},
		},
	}

	transfers, err := NormalizeEnhancedTx(tx)
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	if len(transfers) != 1 {
		t.Fatalf("expected 1 transfer, got %d", len(transfers))
	}
	got := transfers[0]
	if got.NormalizationStatus != NormalizationUnresolvedOwner || got.PoisoningEligible {
		t.Fatalf("expected unresolved owner + ineligible, got status=%s eligible=%v", got.NormalizationStatus, got.PoisoningEligible)
	}
	if got.NormalizationReasonCode != "missing_spl_owner_endpoint" {
		t.Fatalf("unexpected reason: %s", got.NormalizationReasonCode)
	}
}

func TestNormalizeEnhancedTxSPLOwnerEqualsTokenAccountIsUnresolved(t *testing.T) {
	tx := helius.EnhancedTransaction{
		Signature:     "sig2",
		TimestampUnix: time.Now().Unix(),
		TokenTransfers: []helius.TokenTransfer{
			{
				FromUserAccount:  "tokenA",
				ToUserAccount:    "ownerB",
				FromTokenAccount: "tokenA",
				ToTokenAccount:   "tokenB",
				Mint:             "mintA",
				TokenAmount:      helius.TokenAmount{Amount: "1"},
				TokenStandard:    "Fungible",
			},
		},
	}

	transfers, err := NormalizeEnhancedTx(tx)
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	got := transfers[0]
	if got.NormalizationStatus != NormalizationUnresolvedOwner {
		t.Fatalf("expected unresolved owner, got %s", got.NormalizationStatus)
	}
	if got.NormalizationReasonCode != "owner_equals_token_account_source" {
		t.Fatalf("unexpected reason: %s", got.NormalizationReasonCode)
	}
}

func TestNormalizeEnhancedTxUnsupportedAssetIsMarkedOther(t *testing.T) {
	tx := helius.EnhancedTransaction{
		Signature:     "sig3",
		TimestampUnix: time.Now().Unix(),
		TokenTransfers: []helius.TokenTransfer{
			{
				FromUserAccount:  "ownerA",
				ToUserAccount:    "ownerB",
				FromTokenAccount: "tokenA",
				ToTokenAccount:   "tokenB",
				Mint:             "mintA",
				TokenAmount:      helius.TokenAmount{Amount: "1"},
				TokenStandard:    "NonFungible",
			},
		},
	}

	transfers, err := NormalizeEnhancedTx(tx)
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	got := transfers[0]
	if got.NormalizationStatus != NormalizationUnsupportedAsset || got.PoisoningEligible {
		t.Fatalf("expected unsupported + ineligible, got status=%s eligible=%v", got.NormalizationStatus, got.PoisoningEligible)
	}
	if got.AssetType != AssetTypeOther {
		t.Fatalf("expected asset type other for unsupported asset, got %s", got.AssetType)
	}
}

func TestNormalizeEnhancedTxNativeSelfTransferIsNonEligible(t *testing.T) {
	tx := helius.EnhancedTransaction{
		Signature:     "sig4",
		TimestampUnix: time.Now().Unix(),
		NativeTransfers: []helius.NativeTransfer{
			{
				FromUserAccount: "ownerA",
				ToUserAccount:   "ownerA",
				Amount:          "1",
			},
		},
	}

	transfers, err := NormalizeEnhancedTx(tx)
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	got := transfers[0]
	if got.NormalizationStatus != NormalizationResolved {
		t.Fatalf("expected resolved status for owner-level self transfer, got %s", got.NormalizationStatus)
	}
	if got.PoisoningEligible {
		t.Fatal("expected self transfer to be poisoning-ineligible")
	}
	if got.NormalizationReasonCode != "self_transfer_owner_level" {
		t.Fatalf("unexpected reason: %s", got.NormalizationReasonCode)
	}
}
