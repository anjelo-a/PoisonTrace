package helius

import "time"

type EnhancedPage struct {
	Transactions []EnhancedTransaction `json:"transactions"`
	Before       string                `json:"before,omitempty"`
}

type EnhancedTransaction struct {
	Signature        string           `json:"signature"`
	Slot             int64            `json:"slot"`
	TimestampUnix    int64            `json:"timestamp"`
	TransactionError any              `json:"transactionError"`
	NativeTransfers  []NativeTransfer `json:"nativeTransfers"`
	TokenTransfers   []TokenTransfer  `json:"tokenTransfers"`
}

func (t EnhancedTransaction) BlockTimeUTC() time.Time {
	return time.Unix(t.TimestampUnix, 0).UTC()
}

func (t EnhancedTransaction) IsSuccess() bool {
	return t.TransactionError == nil
}

type NativeTransfer struct {
	FromUserAccount string `json:"fromUserAccount"`
	ToUserAccount   string `json:"toUserAccount"`
	Amount          string `json:"amount"`
}

type TokenTransfer struct {
	FromUserAccount  string      `json:"fromUserAccount"`
	ToUserAccount    string      `json:"toUserAccount"`
	FromTokenAccount string      `json:"fromTokenAccount"`
	ToTokenAccount   string      `json:"toTokenAccount"`
	Mint             string      `json:"mint"`
	TokenAmount      TokenAmount `json:"tokenAmount"`
	TokenStandard    string      `json:"tokenStandard"`
	InstructionIndex *int        `json:"instructionIndex,omitempty"`
	InnerIndex       *int        `json:"innerInstructionIndex,omitempty"`
}

type TokenAmount struct {
	Amount   string `json:"amount"`
	Decimals *int   `json:"decimals"`
}
