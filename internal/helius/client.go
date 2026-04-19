package helius

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

type Client interface {
	FetchEnhancedPage(ctx context.Context, walletAddress string, before string) (EnhancedPage, error)
}

type HTTPClient struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *http.Client
	pageLimit  int
}

func NewHTTPClient(baseURL, apiKey string, timeout time.Duration) (*HTTPClient, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL is required")
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("apiKey is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse baseURL: %w", err)
	}
	if strings.TrimSpace(parsed.Scheme) == "" || strings.TrimSpace(parsed.Host) == "" {
		return nil, fmt.Errorf("baseURL must include scheme and host (e.g. https://api.helius.xyz)")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https":
	default:
		return nil, fmt.Errorf("baseURL scheme must be https")
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &HTTPClient{
		baseURL: parsed,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		pageLimit: 100,
	}, nil
}

func (c *HTTPClient) FetchEnhancedPage(ctx context.Context, walletAddress string, before string) (EnhancedPage, error) {
	walletAddress = strings.TrimSpace(walletAddress)
	if walletAddress == "" {
		return EnhancedPage{}, errors.New("wallet address is required")
	}
	if c == nil || c.baseURL == nil || c.httpClient == nil {
		return EnhancedPage{}, fmt.Errorf("helius client is not initialized")
	}

	u := *c.baseURL
	u.Path = path.Join(c.baseURL.Path, "addresses", walletAddress, "transactions")
	q := u.Query()
	q.Set("limit", fmt.Sprintf("%d", c.pageLimit))
	if before != "" {
		q.Set("before", before)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return EnhancedPage{}, fmt.Errorf("build helius request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return EnhancedPage{}, fmt.Errorf("helius request: %w", redactedError{err: err})
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return EnhancedPage{}, fmt.Errorf("read helius response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return EnhancedPage{}, StatusError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(body)),
		}
	}

	page, err := decodeEnhancedPage(body)
	if err != nil {
		return EnhancedPage{}, fmt.Errorf("decode helius enhanced page: %w", err)
	}
	if page.Before == "" && len(page.Transactions) > 0 {
		page.Before = page.Transactions[len(page.Transactions)-1].Signature
	}
	return page, nil
}

type StatusError struct {
	StatusCode int
	Body       string
}

func (e StatusError) Error() string {
	body := redactSensitiveValues(e.Body)
	if len(body) > 256 {
		body = body[:256]
	}
	if body == "" {
		return fmt.Sprintf("helius status %d", e.StatusCode)
	}
	return fmt.Sprintf("helius status %d: %s", e.StatusCode, body)
}

type redactedError struct {
	err error
}

func (e redactedError) Error() string {
	if e.err == nil {
		return ""
	}
	return redactSensitiveValues(e.err.Error())
}

func (e redactedError) Unwrap() error {
	return e.err
}

var (
	sensitiveQueryPairPattern = regexp.MustCompile(`(?i)\b(api[-_]?key|apikey|token|access[-_]?token|authorization|auth|secret|password)=([^&\s"]+)`)
	authorizationHeaderRegex  = regexp.MustCompile(`(?i)\bauthorization\s*:\s*(?:bearer\s+|basic\s+)?[A-Za-z0-9._~+/\-=]+`)
	sensitiveHeaderPattern    = regexp.MustCompile(`(?i)\b(x-api-key|api-key|x-auth-token)\s*:\s*[^\s,;]+`)
	sensitiveJSONPattern      = regexp.MustCompile(`(?i)"(api[-_]?key|apikey|token|access[-_]?token|authorization|auth|secret|password)"\s*:\s*"[^"]*"`)
	bearerTokenPattern        = regexp.MustCompile(`(?i)\b(Bearer)\s+[A-Za-z0-9._~+/\-=]+`)
)

func redactSensitiveValues(text string) string {
	if text == "" {
		return ""
	}
	redacted := sensitiveQueryPairPattern.ReplaceAllString(text, "${1}=REDACTED")
	redacted = authorizationHeaderRegex.ReplaceAllString(redacted, "authorization: REDACTED")
	redacted = sensitiveHeaderPattern.ReplaceAllStringFunc(redacted, func(match string) string {
		idx := strings.Index(match, ":")
		if idx < 0 {
			return "REDACTED"
		}
		return match[:idx+1] + " REDACTED"
	})
	redacted = sensitiveJSONPattern.ReplaceAllString(redacted, `"$1":"REDACTED"`)
	return bearerTokenPattern.ReplaceAllString(redacted, "$1 REDACTED")
}

func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var statusErr StatusError
	if errors.As(err, &statusErr) {
		return statusErr.StatusCode == http.StatusTooManyRequests || statusErr.StatusCode >= 500
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "eof") ||
		strings.Contains(msg, "timeout")
}

func decodeEnhancedPage(raw []byte) (EnhancedPage, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return EnhancedPage{}, nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var txs []EnhancedTransaction
		if err := json.Unmarshal(raw, &txs); err != nil {
			return EnhancedPage{}, err
		}
		return EnhancedPage{Transactions: txs}, nil
	}

	var wire struct {
		Transactions []EnhancedTransaction `json:"transactions"`
		Before       string                `json:"before"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return EnhancedPage{}, err
	}
	return EnhancedPage{
		Transactions: wire.Transactions,
		Before:       wire.Before,
	}, nil
}
