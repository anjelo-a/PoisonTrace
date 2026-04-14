package transactions

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func BuildTransferFingerprint(signature, instructionRef, srcOwner, dstOwner, mint, amountRaw string, assetType AssetType) string {
	seed := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s", signature, instructionRef, srcOwner, dstOwner, mint, amountRaw, assetType)
	sum := sha256.Sum256([]byte(seed))
	return "sha256:" + hex.EncodeToString(sum[:])
}
