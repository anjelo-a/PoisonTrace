package helius

import (
	"strings"
	"testing"
)

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
			wantErr: "scheme must be http or https",
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
