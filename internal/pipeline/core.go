package pipeline

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"poisontrace/internal/counterparties"
	"poisontrace/internal/helius"
	"poisontrace/internal/transactions"
)

type CoreSyncParams struct {
	FocalWalletAddress     string
	BaselineStart          time.Time
	BaselineEnd            time.Time
	ScanStart              time.Time
	ScanEnd                time.Time
	MaxTXPagesPerWallet    int
	MaxTXPerWallet         int
	MaxHeliusRetries       int
	HeliusRequestDelay     time.Duration
	LookalikeRecencyDays   int
	LookalikePrefixMin     int
	LookalikeSuffixMin     int
	LookalikeSingleSideMin int
	MinInjectionCount      int
	ClassifyDust           func(tr transactions.NormalizedTransfer) transactions.DustStatus
}

type CoreSyncResult struct {
	BaselineObservations []WalletTransferObservation
	ScanObservations     []WalletTransferObservation
	Counterparties       map[string]counterparties.Counterparty
	Candidates           []PoisoningCandidate
	BaselineComplete     bool
	IncompleteWindow     bool
	UnknownGateReason    string
	BaselineTruncation   string
	ScanTruncation       string
}

func RunWalletCoreSync(ctx context.Context, client helius.Client, p CoreSyncParams) (CoreSyncResult, error) {
	if strings.TrimSpace(p.FocalWalletAddress) == "" {
		return CoreSyncResult{}, fmt.Errorf("focal wallet address is required")
	}
	if !p.BaselineStart.Before(p.BaselineEnd) || !p.ScanStart.Before(p.ScanEnd) {
		return CoreSyncResult{}, fmt.Errorf("invalid baseline/scan windows")
	}
	if !p.BaselineEnd.Equal(p.ScanStart) {
		return CoreSyncResult{}, fmt.Errorf("baseline end must equal scan start")
	}

	baselineFetch, err := FetchEnhancedWindow(ctx, client, p.FocalWalletAddress, FetchWindowParams{
		Start:        p.BaselineStart,
		End:          p.BaselineEnd,
		MaxPages:     p.MaxTXPagesPerWallet,
		MaxTx:        p.MaxTXPerWallet,
		MaxRetries:   p.MaxHeliusRetries,
		RequestDelay: p.HeliusRequestDelay,
	})
	if err != nil {
		return CoreSyncResult{}, fmt.Errorf("fetch baseline enhanced tx: %w", err)
	}
	scanFetch, err := FetchEnhancedWindow(ctx, client, p.FocalWalletAddress, FetchWindowParams{
		Start:        p.ScanStart,
		End:          p.ScanEnd,
		MaxPages:     p.MaxTXPagesPerWallet,
		MaxTx:        p.MaxTXPerWallet,
		MaxRetries:   p.MaxHeliusRetries,
		RequestDelay: p.HeliusRequestDelay,
	})
	if err != nil {
		return CoreSyncResult{}, fmt.Errorf("fetch scan enhanced tx: %w", err)
	}

	baselineObs, err := normalizeAndMapByWallet(p.FocalWalletAddress, baselineFetch.Transactions, p.ClassifyDust)
	if err != nil {
		return CoreSyncResult{}, fmt.Errorf("normalize baseline transfers: %w", err)
	}
	scanObs, err := normalizeAndMapByWallet(p.FocalWalletAddress, scanFetch.Transactions, p.ClassifyDust)
	if err != nil {
		return CoreSyncResult{}, fmt.Errorf("normalize scan transfers: %w", err)
	}

	cpState := make(map[string]counterparties.Counterparty)
	applyObservationsToCounterparties(cpState, baselineObs)
	applyObservationsToCounterparties(cpState, scanObs)

	baselineComplete := !baselineFetch.Partial
	materialized := MaterializeCandidates(baselineObs, scanObs, CandidateMaterializeParams{
		BaselineComplete:       baselineComplete,
		LookalikeRecencyDays:   p.LookalikeRecencyDays,
		LookalikePrefixMin:     p.LookalikePrefixMin,
		LookalikeSuffixMin:     p.LookalikeSuffixMin,
		LookalikeSingleSideMin: p.LookalikeSingleSideMin,
		MinInjectionCount:      p.MinInjectionCount,
	})

	incomplete := baselineFetch.Partial || scanFetch.Partial || materialized.IncompleteWindow
	reason := mergeReasons(
		materialized.UnknownGateReason,
		truncationReason("baseline", baselineFetch.TruncationCode),
		truncationReason("scan", scanFetch.TruncationCode),
	)

	return CoreSyncResult{
		BaselineObservations: baselineObs,
		ScanObservations:     scanObs,
		Counterparties:       cpState,
		Candidates:           materialized.Candidates,
		BaselineComplete:     baselineComplete,
		IncompleteWindow:     incomplete,
		UnknownGateReason:    reason,
		BaselineTruncation:   baselineFetch.TruncationCode,
		ScanTruncation:       scanFetch.TruncationCode,
	}, nil
}

func normalizeAndMapByWallet(focalWallet string, txs []helius.EnhancedTransaction, classifyDust func(tr transactions.NormalizedTransfer) transactions.DustStatus) ([]WalletTransferObservation, error) {
	out := make([]WalletTransferObservation, 0, len(txs))
	for _, tx := range txs {
		normalized, err := transactions.NormalizeEnhancedTx(tx)
		if err != nil {
			return nil, err
		}
		for _, tr := range normalized {
			if classifyDust != nil {
				tr.DustStatus = classifyDust(tr)
			}
			mapping, ok := counterparties.MapWalletRelation(focalWallet, tr)
			if !ok {
				continue
			}
			out = append(out, WalletTransferObservation{
				Transfer:            tr,
				RelationType:        mapping.RelationType,
				CounterpartyAddress: mapping.CounterpartyAddress,
			})
		}
	}
	return out, nil
}

func applyObservationsToCounterparties(state map[string]counterparties.Counterparty, observations []WalletTransferObservation) {
	for _, obs := range observations {
		cpAddr := strings.TrimSpace(obs.CounterpartyAddress)
		if cpAddr == "" {
			continue
		}
		cp := state[cpAddr]
		if cp.CounterpartyAddress == "" {
			cp.CounterpartyAddress = cpAddr
		}
		cp = counterparties.ApplyEvent(cp, counterparties.Event{
			CounterpartyAddress: cpAddr,
			RelationType:        obs.RelationType,
			OccurredAt:          obs.Transfer.BlockTime.UTC(),
		})
		state[cpAddr] = cp
	}
}

func truncationReason(window, code string) string {
	if strings.TrimSpace(code) == "" {
		return ""
	}
	return window + "_truncation:" + code
}

func mergeReasons(reasons ...string) string {
	uniq := make(map[string]struct{})
	for _, reason := range reasons {
		reason = strings.TrimSpace(reason)
		if reason == "" {
			continue
		}
		uniq[reason] = struct{}{}
	}
	if len(uniq) == 0 {
		return ""
	}
	keys := make([]string, 0, len(uniq))
	for k := range uniq {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ";")
}
