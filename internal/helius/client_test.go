package helius

import (
	"context"
	"errors"
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

func TestFetchEnhancedPageRedactsAPIKeyOnTransportError(t *testing.T) {
	t.Parallel()

	const apiKey = "pt_test_secret_key"
	client, err := NewHTTPClient("https://api.helius.xyz/v0", apiKey, 0)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, &url.Error{
			Op:  req.Method,
			URL: req.URL.String(),
			Err: errors.New("dial tcp: lookup api.helius.xyz: no such host"),
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
	if !strings.Contains(msg, "api-key=REDACTED") {
		t.Fatalf("expected redacted api key marker in error, got %q", msg)
	}
	if !IsRetryable(err) {
		t.Fatalf("expected transport error to remain retryable, got %q", msg)
	}
}
