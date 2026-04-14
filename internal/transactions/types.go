package transactions

import "time"

type NormalizationStatus string

const (
	NormalizationResolved         NormalizationStatus = "resolved"
	NormalizationUnresolvedOwner  NormalizationStatus = "unresolved_owner"
	NormalizationUnsupportedAsset NormalizationStatus = "unsupported_asset"
	NormalizationFailed           NormalizationStatus = "failed"
)

type DustStatus string

const (
	DustTrue    DustStatus = "true"
	DustFalse   DustStatus = "false"
	DustUnknown DustStatus = "unknown"
)

type AssetType string

const (
	AssetTypeNativeSOL   AssetType = "native_sol"
	AssetTypeSPLFungible AssetType = "spl_fungible"
	AssetTypeOther       AssetType = "other"
)

type NormalizedTransfer struct {
	Signature               string
	TransferIndex           int
	TransferFingerprint     string
	Slot                    int64
	BlockTime               time.Time
	SourceOwnerAddress      string
	DestinationOwnerAddress string
	SourceTokenAccount      string
	DestinationTokenAccount string
	AmountRaw               string
	TokenMint               string
	AssetType               AssetType
	AssetKey                string
	Decimals                *int
	NormalizationStatus     NormalizationStatus
	NormalizationReasonCode string
	PoisoningEligible       bool
	DustStatus              DustStatus
	IsSuccess               bool
}
