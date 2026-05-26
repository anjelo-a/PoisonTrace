package walletsource

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type RPC interface {
	GetSlot(ctx context.Context) (int64, error)
	GetBlock(ctx context.Context, slot int64) (RPCBlock, error)
}

type HeliusRPCClient struct {
	endpoint   string
	httpClient *http.Client
}

func NewHeliusRPCClient(apiKey string, timeout time.Duration) (*HeliusRPCClient, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("api key is required")
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	u := url.URL{
		Scheme:   "https",
		Host:     "mainnet.helius-rpc.com",
		RawQuery: url.Values{"api-key": []string{apiKey}}.Encode(),
	}
	return &HeliusRPCClient{
		endpoint:   u.String(),
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse[T any] struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      int       `json:"id"`
	Result  T         `json:"result"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e rpcError) Error() string {
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

func (c *HeliusRPCClient) GetSlot(ctx context.Context) (int64, error) {
	return rpcCall[int64](ctx, c, "getSlot", []any{map[string]string{"commitment": "finalized"}})
}

func (c *HeliusRPCClient) GetBlock(ctx context.Context, slot int64) (RPCBlock, error) {
	params := []any{
		slot,
		map[string]any{
			"encoding":                       "jsonParsed",
			"transactionDetails":             "full",
			"rewards":                        false,
			"maxSupportedTransactionVersion": 0,
		},
	}
	return rpcCall[RPCBlock](ctx, c, "getBlock", params)
}

func rpcCall[T any](ctx context.Context, c *HeliusRPCClient, method string, params any) (T, error) {
	var zero T
	if c == nil || c.httpClient == nil || c.endpoint == "" {
		return zero, fmt.Errorf("rpc client is not initialized")
	}
	body, err := json.Marshal(rpcRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: params})
	if err != nil {
		return zero, fmt.Errorf("encode rpc request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return zero, fmt.Errorf("build rpc request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return zero, fmt.Errorf("rpc request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return zero, fmt.Errorf("read rpc response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return zero, fmt.Errorf("rpc http status %d", resp.StatusCode)
	}
	var decoded rpcResponse[T]
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return zero, fmt.Errorf("decode rpc response: %w", err)
	}
	if decoded.Error != nil {
		return zero, *decoded.Error
	}
	return decoded.Result, nil
}

type RPCBlock struct {
	BlockTime    *int64               `json:"blockTime"`
	Transactions []RPCBlockTxEnvelope `json:"transactions"`
}

type RPCBlockTxEnvelope struct {
	Transaction RPCTransaction `json:"transaction"`
	Meta        RPCMeta        `json:"meta"`
}

type RPCTransaction struct {
	Signatures []string   `json:"signatures"`
	Message    RPCMessage `json:"message"`
}

type RPCMessage struct {
	AccountKeys  []RPCAccountKey  `json:"accountKeys"`
	Instructions []RPCInstruction `json:"instructions"`
}

type RPCAccountKey struct {
	Pubkey   string `json:"pubkey"`
	Signer   bool   `json:"signer"`
	Writable bool   `json:"writable"`
	Source   string `json:"source"`
}

type RPCInstruction struct {
	Program string                `json:"program"`
	Parsed  *RPCParsedInstruction `json:"parsed,omitempty"`
}

type RPCParsedInstruction struct {
	Type string `json:"type"`
}

type RPCMeta struct {
	Err any `json:"err"`
}

type ScrapeOptions struct {
	StartSlot            int64
	BlockLookback        int
	MaxBlocks            int
	MaxWallets           int
	MaxTXPerBlock        int
	MaxNoisyInstructions int
}

func ScrapeRecentWallets(ctx context.Context, rpc RPC, opts ScrapeOptions) ([]string, error) {
	if rpc == nil {
		return nil, fmt.Errorf("rpc client is required")
	}
	opts = withScrapeDefaults(opts)
	startSlot := opts.StartSlot
	if startSlot == 0 {
		latest, err := rpc.GetSlot(ctx)
		if err != nil {
			return nil, fmt.Errorf("get latest slot: %w", err)
		}
		startSlot = latest
	}

	counts := make(map[string]int)
	for checked := 0; checked < opts.BlockLookback && len(counts) < opts.MaxWallets; checked++ {
		if opts.MaxBlocks > 0 && checked >= opts.MaxBlocks {
			break
		}
		slot := startSlot - int64(checked)
		if slot <= 0 {
			break
		}
		block, err := rpc.GetBlock(ctx, slot)
		if err != nil {
			continue
		}
		txLimit := len(block.Transactions)
		if opts.MaxTXPerBlock > 0 && txLimit > opts.MaxTXPerBlock {
			txLimit = opts.MaxTXPerBlock
		}
		for _, tx := range block.Transactions[:txLimit] {
			if tx.Meta.Err != nil {
				continue
			}
			if !quietTransferLike(tx.Transaction.Message.Instructions, opts.MaxNoisyInstructions) {
				continue
			}
			for _, key := range tx.Transaction.Message.AccountKeys {
				if !key.Signer || !key.Writable || key.Source == "lookupTable" {
					continue
				}
				if strings.TrimSpace(key.Pubkey) == "" {
					continue
				}
				counts[key.Pubkey]++
			}
		}
	}

	type scored struct {
		address string
		count   int
	}
	scoredWallets := make([]scored, 0, len(counts))
	for address, count := range counts {
		scoredWallets = append(scoredWallets, scored{address: address, count: count})
	}
	sort.Slice(scoredWallets, func(i, j int) bool {
		if scoredWallets[i].count == scoredWallets[j].count {
			return scoredWallets[i].address < scoredWallets[j].address
		}
		return scoredWallets[i].count > scoredWallets[j].count
	})

	out := make([]string, 0, len(scoredWallets))
	for _, wallet := range scoredWallets {
		out = append(out, wallet.address)
		if opts.MaxWallets > 0 && len(out) >= opts.MaxWallets {
			break
		}
	}
	return out, nil
}

func withScrapeDefaults(opts ScrapeOptions) ScrapeOptions {
	if opts.BlockLookback <= 0 {
		opts.BlockLookback = 100
	}
	if opts.MaxBlocks <= 0 {
		opts.MaxBlocks = opts.BlockLookback
	}
	if opts.MaxWallets <= 0 {
		opts.MaxWallets = 100
	}
	if opts.MaxTXPerBlock <= 0 {
		opts.MaxTXPerBlock = 200
	}
	if opts.MaxNoisyInstructions < 0 {
		opts.MaxNoisyInstructions = 0
	}
	return opts
}

func quietTransferLike(instructions []RPCInstruction, maxNoisy int) bool {
	if len(instructions) == 0 {
		return false
	}
	transferCount := 0
	noisyCount := 0
	for _, ix := range instructions {
		ixType := strings.ToLower(strings.TrimSpace(parsedInstructionType(ix)))
		switch ixType {
		case "transfer", "transferchecked":
			transferCount++
		case "createaccount", "closeaccount", "syncnative":
		case "setcomputeunitlimit", "setcomputeunitprice":
			noisyCount++
		default:
			if ixType != "" {
				noisyCount++
			}
		}
	}
	return transferCount > 0 && noisyCount <= maxNoisy
}

func parsedInstructionType(ix RPCInstruction) string {
	if ix.Parsed != nil {
		return ix.Parsed.Type
	}
	return ix.Program
}
