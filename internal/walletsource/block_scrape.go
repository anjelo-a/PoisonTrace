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
	Type string         `json:"type"`
	Info map[string]any `json:"info"`
}

func (i *RPCInstruction) UnmarshalJSON(data []byte) error {
	var raw struct {
		Program string          `json:"program"`
		Parsed  json.RawMessage `json:"parsed"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	i.Program = raw.Program
	trimmed := bytes.TrimSpace(raw.Parsed)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}
	if trimmed[0] != '{' {
		return nil
	}
	var parsed RPCParsedInstruction
	if err := json.Unmarshal(trimmed, &parsed); err != nil {
		return nil
	}
	i.Parsed = &parsed
	return nil
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
	MinNativeLamports    int64
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

	stats := make(map[string]blockWalletStats)
	for checked := 0; checked < opts.BlockLookback; checked++ {
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
		inspected := 0
		for _, tx := range block.Transactions {
			if tx.Meta.Err != nil {
				continue
			}
			if !quietTransferLike(tx.Transaction.Message.Instructions, opts.MaxNoisyInstructions) {
				continue
			}
			inspected++
			if opts.MaxTXPerBlock > 0 && inspected > opts.MaxTXPerBlock {
				break
			}
			observations, hasParsedTransfer := parsedTransferOwnerObservations(tx.Transaction.Message.Instructions, opts.MinNativeLamports)
			if len(observations) == 0 && !hasParsedTransfer {
				for _, address := range signerWallets(tx.Transaction.Message.AccountKeys) {
					observations = append(observations, blockWalletObservation{address: address, outbound: true})
				}
			}
			for _, obs := range observations {
				rec := stats[obs.address]
				rec.count++
				if obs.outbound {
					rec.outbound++
				} else {
					rec.inbound++
				}
				stats[obs.address] = rec
			}
		}
	}

	counts := make(map[string]int)
	for address, stat := range stats {
		if stat.outbound == 0 || stat.inbound > stat.outbound {
			continue
		}
		counts[address] = stat.count
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
		return scoredWallets[i].count < scoredWallets[j].count
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

type blockWalletStats struct {
	count    int
	outbound int
	inbound  int
}

type blockWalletObservation struct {
	address  string
	outbound bool
}

func parsedTransferOwnerObservations(instructions []RPCInstruction, minNativeLamports int64) ([]blockWalletObservation, bool) {
	seen := make(map[blockWalletObservation]struct{})
	out := make([]blockWalletObservation, 0)
	hasParsedTransfer := false
	for _, ix := range instructions {
		ixType := strings.ToLower(strings.TrimSpace(parsedInstructionType(ix)))
		if ixType != "transfer" && ixType != "transferchecked" {
			continue
		}
		if ix.Parsed == nil || ix.Parsed.Info == nil {
			continue
		}
		hasParsedTransfer = true
		for _, obs := range transferOwnerObservations(ix, minNativeLamports) {
			if _, ok := seen[obs]; ok {
				continue
			}
			seen[obs] = struct{}{}
			out = append(out, obs)
		}
	}
	return out, hasParsedTransfer
}

func transferOwnerObservations(ix RPCInstruction, minNativeLamports int64) []blockWalletObservation {
	observations := make([]blockWalletObservation, 0, 2)
	authority, _ := ix.Parsed.Info["authority"].(string)
	authority = strings.TrimSpace(authority)
	if authority != "" {
		observations = append(observations, blockWalletObservation{address: authority, outbound: true})
	}
	if strings.EqualFold(ix.Program, "system") {
		if lamports, ok := parsedLamports(ix.Parsed.Info["lamports"]); ok && lamports < minNativeLamports {
			return observations
		}
		source, _ := ix.Parsed.Info["source"].(string)
		source = strings.TrimSpace(source)
		if source != "" {
			observations = append(observations, blockWalletObservation{address: source, outbound: true})
		}
		destination, _ := ix.Parsed.Info["destination"].(string)
		destination = strings.TrimSpace(destination)
		if destination != "" {
			observations = append(observations, blockWalletObservation{address: destination, outbound: false})
		}
	}
	return observations
}

func parsedLamports(value any) (int64, bool) {
	switch v := value.(type) {
	case float64:
		return int64(v), true
	case int64:
		return v, true
	case int:
		return int64(v), true
	case json.Number:
		n, err := v.Int64()
		return n, err == nil
	default:
		return 0, false
	}
}

func signerWallets(keys []RPCAccountKey) []string {
	out := make([]string, 0)
	for _, key := range keys {
		if !key.Signer || !key.Writable || key.Source == "lookupTable" {
			continue
		}
		if strings.TrimSpace(key.Pubkey) == "" {
			continue
		}
		out = append(out, key.Pubkey)
	}
	return out
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
	if opts.MinNativeLamports < 0 {
		opts.MinNativeLamports = 0
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
