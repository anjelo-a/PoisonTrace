package helius

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

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

func (n *NativeTransfer) UnmarshalJSON(data []byte) error {
	type wire NativeTransfer
	var aux struct {
		wire
		Amount json.RawMessage `json:"amount"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*n = NativeTransfer(aux.wire)
	amount, err := decodeFlexibleJSONNumber(aux.Amount)
	if err != nil {
		return fmt.Errorf("decode native transfer amount: %w", err)
	}
	n.Amount = amount
	return nil
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

func (t *TokenAmount) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		t.Amount = ""
		t.Decimals = nil
		return nil
	}
	if strings.HasPrefix(trimmed, "{") {
		type wire TokenAmount
		var aux wire
		if err := json.Unmarshal(data, &aux); err != nil {
			return err
		}
		t.Amount = aux.Amount
		t.Decimals = aux.Decimals
		return nil
	}
	amount, err := decodeFlexibleJSONNumber(data)
	if err != nil {
		return fmt.Errorf("decode token amount: %w", err)
	}
	t.Amount = amount
	t.Decimals = nil
	return nil
}

func decodeFlexibleJSONNumber(raw json.RawMessage) (string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return "", nil
	}
	if strings.HasPrefix(trimmed, "\"") {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return "", err
		}
		return strings.TrimSpace(s), nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var n json.Number
	if err := dec.Decode(&n); err != nil {
		return "", err
	}
	return strings.TrimSpace(n.String()), nil
}
