package helius

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestNewHTTPClientRequiresAbsoluteHTTPBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		baseURL string
		wantErr string
	}{
		{
			name:    "relative URL",
			baseURL: "/v0",
			wantErr: "must include scheme and host",
		},
		{
			name:    "missing scheme",
			baseURL: "api.helius.xyz/v0",
			wantErr: "must include scheme and host",
		},
		{
			name:    "unsupported scheme",
			baseURL: "ftp://api.helius.xyz/v0",
			wantErr: "scheme must be https",
		},
		{
			name:    "http scheme is rejected",
			baseURL: "http://api.helius.xyz/v0",
			wantErr: "scheme must be https",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewHTTPClient(tc.baseURL, "test-api-key", 0)
			if err == nil {
				t.Fatalf("expected error for %q", tc.baseURL)
			}
			if got := err.Error(); got == "" || !strings.Contains(got, tc.wantErr) {
				t.Fatalf("expected error to contain %q, got %q", tc.wantErr, got)
			}
		})
	}
}

func TestNewHTTPClientAcceptsValidHTTPSBaseURL(t *testing.T) {
	t.Parallel()

	client, err := NewHTTPClient("https://api.helius.xyz/v0", "test-api-key", 0)
	if err != nil {
		t.Fatalf("expected valid base URL, got error: %v", err)
	}
	if client == nil || client.baseURL == nil {
		t.Fatal("expected initialized client")
	}
}

func TestFetchEnhancedPageUsesHeaderAuthNotQueryParam(t *testing.T) {
	t.Parallel()

	const apiKey = "pt_test_secret_key"
	client, err := NewHTTPClient("https://api.helius.xyz/v0", apiKey, 0)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.URL.Query().Get("api-key"); got != "" {
			t.Fatalf("expected api-key query param to be empty, got %q", got)
		}
		if got := req.Header.Get("X-API-Key"); got != apiKey {
			t.Fatalf("expected X-API-Key header to be set")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("[]")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})

	if _, err := client.FetchEnhancedPage(context.Background(), "wallet123", ""); err != nil {
		t.Fatalf("fetch enhanced page failed: %v", err)
	}
}

func TestFetchEnhancedPageRedactsSensitiveValuesOnTransportError(t *testing.T) {
	t.Parallel()

	const apiKey = "pt_test_secret_key"
	client, err := NewHTTPClient("https://api.helius.xyz/v0", apiKey, 0)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, &url.Error{
			Op:  req.Method,
			URL: req.URL.String() + "?authorization=Bearer+" + apiKey,
			Err: errors.New("dial tcp: authorization: Bearer " + apiKey + " x-api-key: " + apiKey),
		}
	})

	_, err = client.FetchEnhancedPage(context.Background(), "wallet123", "")
	if err == nil {
		t.Fatal("expected transport error")
	}
	msg := err.Error()
	if strings.Contains(msg, apiKey) {
		t.Fatalf("error leaked api key: %q", msg)
	}
	if !strings.Contains(strings.ToLower(msg), "authorization=redacted") {
		t.Fatalf("expected redacted query marker in error, got %q", msg)
	}
	if !strings.Contains(strings.ToLower(msg), "x-api-key: redacted") {
		t.Fatalf("expected redacted header marker in error, got %q", msg)
	}
	if !IsRetryable(err) {
		t.Fatalf("expected transport error to remain retryable, got %q", msg)
	}
}

func TestRedactSensitiveValuesCoversMultiplePatterns(t *testing.T) {
	t.Parallel()

	input := `api-key=secret1 apikey=secret2 token=secret3 access_token=secret4 authorization: Bearer secret5 x-api-key: secret6 {"api_key":"secret7","password":"secret8"}`
	got := redactSensitiveValues(input)
	for _, secret := range []string{"secret1", "secret2", "secret3", "secret4", "secret5", "secret6", "secret7", "secret8"} {
		if strings.Contains(got, secret) {
			t.Fatalf("redaction leaked secret value %q in %q", secret, got)
		}
	}
}
