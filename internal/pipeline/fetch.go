package pipeline

import (
	"context"
	"errors"
	"fmt"
	"time"

	"poisontrace/internal/helius"
)

type FetchWindowResult struct {
	Transactions   []helius.EnhancedTransaction
	Partial        bool
	TruncationCode string
	RetryExhausted bool
}

type FetchWindowParams struct {
	Start        time.Time
	End          time.Time
	MaxPages     int
	MaxTx        int
	MaxRetries   int
	RequestDelay time.Duration
}

func FetchEnhancedWindow(ctx context.Context, client helius.Client, walletAddress string, p FetchWindowParams) (FetchWindowResult, error) {
	if client == nil {
		return FetchWindowResult{}, fmt.Errorf("helius client is required")
	}
	if p.MaxPages < 1 {
		return FetchWindowResult{}, fmt.Errorf("max pages must be >= 1")
	}
	if p.MaxTx < 1 {
		return FetchWindowResult{}, fmt.Errorf("max tx must be >= 1")
	}
	if !p.Start.Before(p.End) {
		return FetchWindowResult{}, fmt.Errorf("window must satisfy start < end")
	}

	result := FetchWindowResult{Transactions: make([]helius.EnhancedTransaction, 0, p.MaxTx)}
	before := ""
	pageCount := 0

	// Bounded pagination loop: exits on explicit page/tx caps, window boundary, or provider exhaustion.
	for {
		if pageCount >= p.MaxPages {
			result.Partial = true
			result.TruncationCode = "max_tx_pages_per_wallet_reached"
			return result, nil
		}
		if len(result.Transactions) >= p.MaxTx {
			result.Partial = true
			result.TruncationCode = "max_tx_per_wallet_reached"
			return result, nil
		}
		if pageCount > 0 && p.RequestDelay > 0 {
			if err := sleepContext(ctx, p.RequestDelay); err != nil {
				return result, err
			}
		}

		page, err := fetchPageWithRetry(ctx, client, walletAddress, before, p.MaxRetries)
		if err != nil {
			var exhausted retryExhaustedError
			if errors.As(err, &exhausted) {
				result.Partial = true
				result.RetryExhausted = true
				result.TruncationCode = "helius_retry_exhausted"
				return result, nil
			}
			return result, err
		}
		pageCount++
		if len(page.Transactions) == 0 {
			return result, nil
		}

		reachedBeforeWindow := false
		for _, tx := range page.Transactions {
			blockTime := tx.BlockTimeUTC()
			if !blockTime.Before(p.End) {
				continue
			}
			if blockTime.Before(p.Start) {
				reachedBeforeWindow = true
				continue
			}
			result.Transactions = append(result.Transactions, tx)
			if len(result.Transactions) >= p.MaxTx {
				result.Partial = true
				result.TruncationCode = "max_tx_per_wallet_reached"
				return result, nil
			}
		}

		if reachedBeforeWindow {
			return result, nil
		}

		nextBefore := page.Before
		if nextBefore == "" {
			nextBefore = page.Transactions[len(page.Transactions)-1].Signature
		}
		if nextBefore == "" || nextBefore == before {
			return result, nil
		}
		before = nextBefore
	}
}

type retryExhaustedError struct {
	last error
}

func (e retryExhaustedError) Error() string {
	if e.last == nil {
		return "helius retry exhausted"
	}
	return fmt.Sprintf("helius retry exhausted: %v", e.last)
}

func fetchPageWithRetry(ctx context.Context, client helius.Client, walletAddress, before string, maxRetries int) (helius.EnhancedPage, error) {
	if maxRetries < 0 {
		maxRetries = 0
	}
	attempt := 0
	for {
		page, err := client.FetchEnhancedPage(ctx, walletAddress, before)
		if err == nil {
			return page, nil
		}
		if !helius.IsRetryable(err) {
			return helius.EnhancedPage{}, err
		}
		if attempt >= maxRetries {
			return helius.EnhancedPage{}, retryExhaustedError{last: err}
		}
		backoff := retryBackoff(attempt)
		if sleepErr := sleepContext(ctx, backoff); sleepErr != nil {
			return helius.EnhancedPage{}, sleepErr
		}
		attempt++
	}
}

func retryBackoff(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	const (
		baseBackoff     = 200 * time.Millisecond
		maxBackoff      = 5 * time.Second
		maxBackoffShift = 5
	)

	shift := attempt
	if shift > maxBackoffShift {
		shift = maxBackoffShift
	}

	// Exponential backoff with bounded growth.
	backoff := baseBackoff << shift
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	// Deterministic jitter smooths retry bursts without introducing non-reproducible tests.
	jitterSeed := attempt % 97
	jitter := time.Duration((jitterSeed*37)%97) * time.Millisecond
	return backoff + jitter
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
