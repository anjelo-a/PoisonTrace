package helius

import (
	"context"
	"errors"
	"fmt"
)

type Client interface {
	FetchEnhancedPage(ctx context.Context, walletAddress string, before string) (EnhancedPage, error)
}

// StubClient is a compile-time placeholder for Phase 0 scaffolding.
// Replace with real HTTP implementation in Phase 1 execution.
type StubClient struct{}

func (StubClient) FetchEnhancedPage(_ context.Context, walletAddress string, _ string) (EnhancedPage, error) {
	if walletAddress == "" {
		return EnhancedPage{}, errors.New("wallet address is required")
	}
	return EnhancedPage{}, fmt.Errorf("helius client not implemented")
}
