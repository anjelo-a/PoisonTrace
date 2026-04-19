package transactions

import (
	"fmt"
	"strings"

	"poisontrace/internal/helius"
)

// NormalizeEnhancedTx expands a Helius enhanced transaction into deterministic transfer events.
func NormalizeEnhancedTx(tx helius.EnhancedTransaction) ([]NormalizedTransfer, error) {
	if tx.Signature == "" {
		return nil, fmt.Errorf("normalize tx: missing signature")
	}

	out := make([]NormalizedTransfer, 0, len(tx.NativeTransfers)+len(tx.TokenTransfers))
	idx := 0

	for _, nt := range tx.NativeTransfers {
		src := strings.TrimSpace(nt.FromUserAccount)
		dst := strings.TrimSpace(nt.ToUserAccount)
		status := NormalizationResolved
		reason := ""
		eligible := true
		if src == "" || dst == "" {
			status = NormalizationFailed
			reason = "missing_native_owner_endpoint"
			eligible = false
		}
		if src != "" && src == dst && status == NormalizationResolved {
			reason = "self_transfer_owner_level"
			eligible = false
		}

		tr := NormalizedTransfer{
			Signature:               tx.Signature,
			TransferIndex:           idx,
			TransferFingerprint:     BuildTransferFingerprint(tx.Signature, fmt.Sprintf("native:%d", idx), src, dst, "", nt.Amount, AssetTypeNativeSOL),
			Slot:                    tx.Slot,
			BlockTime:               tx.BlockTimeUTC(),
			SourceOwnerAddress:      src,
			DestinationOwnerAddress: dst,
			AmountRaw:               nt.Amount,
			AssetType:               AssetTypeNativeSOL,
			AssetKey:                "SOL",
			NormalizationStatus:     status,
			NormalizationReasonCode: reason,
			PoisoningEligible:       eligible,
			DustStatus:              DustUnknown,
			IsSuccess:               tx.IsSuccess(),
		}
		out = append(out, tr)
		idx++
	}

	for _, tt := range tx.TokenTransfers {
		src := strings.TrimSpace(tt.FromUserAccount)
		dst := strings.TrimSpace(tt.ToUserAccount)
		status := NormalizationResolved
		reason := ""
		eligible := true
		assetType := AssetTypeSPLFungible

		if tt.TokenStandard != "Fungible" && tt.TokenStandard != "fungible" && tt.TokenStandard != "FUNGIBLE" {
			status = NormalizationUnsupportedAsset
			reason = "unsupported_token_standard"
			eligible = false
			assetType = AssetTypeOther
		} else {
			// SPL poisoning logic requires owner-level endpoints. Ambiguous owner/token-account forms fail closed.
			if src == "" || dst == "" {
				status = NormalizationUnresolvedOwner
				reason = "missing_spl_owner_endpoint"
				eligible = false
			}
			if src != "" && src == tt.FromTokenAccount {
				status = NormalizationUnresolvedOwner
				reason = "owner_equals_token_account_source"
				eligible = false
			}
			if dst != "" && dst == tt.ToTokenAccount {
				status = NormalizationUnresolvedOwner
				reason = "owner_equals_token_account_destination"
				eligible = false
			}
			if src != "" && dst != "" && src == dst && status == NormalizationResolved {
				reason = "self_transfer_owner_level"
				eligible = false
			}
		}

		instructionRef := "spl"
		if tt.InstructionIndex != nil {
			instructionRef = fmt.Sprintf("spl:%d", *tt.InstructionIndex)
			if tt.InnerIndex != nil {
				instructionRef = fmt.Sprintf("%s:%d", instructionRef, *tt.InnerIndex)
			}
		} else {
			// Fallback keeps fingerprints deterministic when provider instruction metadata is missing.
			instructionRef = fmt.Sprintf("spl_fallback:%d", idx)
		}

		tr := NormalizedTransfer{
			Signature:               tx.Signature,
			TransferIndex:           idx,
			TransferFingerprint:     BuildTransferFingerprint(tx.Signature, instructionRef, src, dst, tt.Mint, tt.TokenAmount.Amount, assetType),
			Slot:                    tx.Slot,
			BlockTime:               tx.BlockTimeUTC(),
			SourceOwnerAddress:      src,
			DestinationOwnerAddress: dst,
			SourceTokenAccount:      tt.FromTokenAccount,
			DestinationTokenAccount: tt.ToTokenAccount,
			AmountRaw:               tt.TokenAmount.Amount,
			TokenMint:               tt.Mint,
			AssetType:               assetType,
			AssetKey:                tt.Mint,
			Decimals:                tt.TokenAmount.Decimals,
			NormalizationStatus:     status,
			NormalizationReasonCode: reason,
			PoisoningEligible:       eligible,
			DustStatus:              DustUnknown,
			IsSuccess:               tx.IsSuccess(),
		}
		out = append(out, tr)
		idx++
	}

	return out, nil
}
