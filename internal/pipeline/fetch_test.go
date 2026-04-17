package pipeline

import (
	"context"
	"fmt"
	"testing"
	"time"

	"poisontrace/internal/helius"
)

type fetchStubClient struct {
	callCount int
	pages     map[string]helius.EnhancedPage
	errs      map[int]error
}

func (s *fetchStubClient) FetchEnhancedPage(_ context.Context, _ string, before string) (helius.EnhancedPage, error) {
	s.callCount++
	if err, ok := s.errs[s.callCount]; ok {
		return helius.EnhancedPage{}, err
	}
	if page, ok := s.pages[before]; ok {
		return page, nil
	}
	return helius.EnhancedPage{}, nil
}

func mkTx(sig string, ts time.Time) helius.EnhancedTransaction {
	return helius.EnhancedTransaction{Signature: sig, TimestampUnix: ts.Unix()}
}

func TestFetchEnhancedWindowRetriesAndCollectsWithinWindow(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)

	client := &fetchStubClient{
		pages: map[string]helius.EnhancedPage{
			"": {
				Transactions: []helius.EnhancedTransaction{
					mkTx("too_new", end.Add(2*time.Hour)),
					mkTx("in_1", end.Add(-2*time.Hour)),
					mkTx("in_2", end.Add(-24*time.Hour)),
				},
				Before: "cursor_1",
			},
			"cursor_1": {
				Transactions: []helius.EnhancedTransaction{
					mkTx("in_3", end.Add(-48*time.Hour)),
					mkTx("too_old", start.Add(-1*time.Hour)),
				},
				Before: "cursor_2",
			},
		},
		errs: map[int]error{
			1: fmt.Errorf("temporary: %w", helius.StatusError{StatusCode: 429}),
		},
	}

	result, err := FetchEnhancedWindow(context.Background(), client, "walletA", FetchWindowParams{
		Start:        start,
		End:          end,
		MaxPages:     5,
		MaxTx:        10,
		MaxRetries:   2,
		RequestDelay: 0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Partial {
		t.Fatal("did not expect partial result")
	}
	if result.RetryExhausted {
		t.Fatal("did not expect retry exhaustion")
	}
	if len(result.Transactions) != 3 {
		t.Fatalf("expected 3 in-window txs, got %d", len(result.Transactions))
	}
	if client.callCount < 3 {
		t.Fatalf("expected retry + multiple page fetches, got callCount=%d", client.callCount)
	}
}

func TestFetchEnhancedWindowMarksPartialOnPageCap(t *testing.T) {
	now := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	client := &fetchStubClient{
		pages: map[string]helius.EnhancedPage{
			"": {Transactions: []helius.EnhancedTransaction{mkTx("a", now.Add(-time.Hour))}, Before: "next"},
		},
	}

	result, err := FetchEnhancedWindow(context.Background(), client, "walletA", FetchWindowParams{
		Start:        now.Add(-7 * 24 * time.Hour),
		End:          now,
		MaxPages:     1,
		MaxTx:        10,
		MaxRetries:   1,
		RequestDelay: 0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Partial {
		t.Fatal("expected partial result")
	}
	if result.TruncationCode != "max_tx_pages_per_wallet_reached" {
		t.Fatalf("unexpected truncation code: %s", result.TruncationCode)
	}
}

func TestFetchEnhancedWindowMarksPartialOnRetryExhaustion(t *testing.T) {
	client := &fetchStubClient{
		pages: map[string]helius.EnhancedPage{},
		errs: map[int]error{
			1: fmt.Errorf("retryable: %w", helius.StatusError{StatusCode: 503}),
			2: fmt.Errorf("retryable: %w", helius.StatusError{StatusCode: 503}),
		},
	}

	result, err := FetchEnhancedWindow(context.Background(), client, "walletA", FetchWindowParams{
		Start:        time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		End:          time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
		MaxPages:     2,
		MaxTx:        10,
		MaxRetries:   1,
		RequestDelay: 0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Partial || !result.RetryExhausted {
		t.Fatal("expected retry exhausted partial result")
	}
	if result.TruncationCode != "helius_retry_exhausted" {
		t.Fatalf("unexpected truncation code: %s", result.TruncationCode)
	}
}

func TestRetryBackoffClampsLargeAttemptsWithoutOverflow(t *testing.T) {
	hugeAttempt := int(^uint(0) >> 1)
	backoff := retryBackoff(hugeAttempt)
	if backoff <= 0 {
		t.Fatalf("expected positive backoff for large attempt, got %s", backoff)
	}
	if backoff > 5*time.Second+96*time.Millisecond {
		t.Fatalf("expected capped backoff with jitter, got %s", backoff)
	}
}
