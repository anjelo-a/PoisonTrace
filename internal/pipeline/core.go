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
	BaselineTransfers           []transactions.NormalizedTransfer
	ScanTransfers               []transactions.NormalizedTransfer
	BaselineObservations        []WalletTransferObservation
	ScanObservations            []WalletTransferObservation
	Counterparties              map[string]counterparties.Counterparty
	Candidates                  []PoisoningCandidate
	BaselineComplete            bool
	IncompleteWindow            bool
	UnknownGateReason           string
	BaselineTruncation          string
	ScanTruncation              string
	TransactionsFetched         int
	TransactionsFailedNormalize int
	OwnerUnresolvedCount        int
	DecimalsUnresolvedCount     int
	RetryExhausted              bool
}

func RunWalletCoreSync(ctx context.Context, client helius.Client, p CoreSyncParams) (CoreSyncResult, error) {
	if err := ValidateCoreSyncParams(p); err != nil {
		return CoreSyncResult{}, err
	}
	if client == nil {
		return CoreSyncResult{}, fmt.Errorf("helius client is required")
	}

	// Phase flow is two-window by design: baseline establishes history/newness, scan evaluates injections.
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

	baselineNormalized, err := normalizeWindow(p.FocalWalletAddress, baselineFetch.Transactions, p.ClassifyDust)
	if err != nil {
		return CoreSyncResult{}, fmt.Errorf("normalize baseline transfers: %w", err)
	}
	scanNormalized, err := normalizeWindow(p.FocalWalletAddress, scanFetch.Transactions, p.ClassifyDust)
	if err != nil {
		return CoreSyncResult{}, fmt.Errorf("normalize scan transfers: %w", err)
	}

	// Build in-memory counterparty state from normalized relation observations.
	cpState := make(map[string]counterparties.Counterparty)
	applyObservationsToCounterparties(cpState, baselineNormalized.Observations)
	applyObservationsToCounterparties(cpState, scanNormalized.Observations)

	// Baseline completeness is strictly tied to bounded fetch outcomes; truncated baseline is never complete.
	baselineComplete := !baselineFetch.Partial
	materialized := MaterializeCandidates(baselineNormalized.Observations, scanNormalized.Observations, CandidateMaterializeParams{
		BaselineComplete:       baselineComplete,
		LookalikeRecencyDays:   p.LookalikeRecencyDays,
		LookalikePrefixMin:     p.LookalikePrefixMin,
		LookalikeSuffixMin:     p.LookalikeSuffixMin,
		LookalikeSingleSideMin: p.LookalikeSingleSideMin,
		MinInjectionCount:      p.MinInjectionCount,
	})

	// Unknown-gate/truncation reasons are merged so reruns emit deterministic reason strings.
	incomplete := baselineFetch.Partial || scanFetch.Partial || materialized.IncompleteWindow
	reason := mergeReasons(
		materialized.UnknownGateReason,
		truncationReason("baseline", baselineFetch.TruncationCode),
		truncationReason("scan", scanFetch.TruncationCode),
	)

	return CoreSyncResult{
		BaselineTransfers:           baselineNormalized.Transfers,
		ScanTransfers:               scanNormalized.Transfers,
		BaselineObservations:        baselineNormalized.Observations,
		ScanObservations:            scanNormalized.Observations,
		Counterparties:              cpState,
		Candidates:                  materialized.Candidates,
		BaselineComplete:            baselineComplete,
		IncompleteWindow:            incomplete,
		UnknownGateReason:           reason,
		BaselineTruncation:          baselineFetch.TruncationCode,
		ScanTruncation:              scanFetch.TruncationCode,
		TransactionsFetched:         baselineNormalized.FetchedTx + scanNormalized.FetchedTx,
		TransactionsFailedNormalize: baselineNormalized.FailedNormalize + scanNormalized.FailedNormalize,
		OwnerUnresolvedCount:        baselineNormalized.OwnerUnresolved + scanNormalized.OwnerUnresolved,
		DecimalsUnresolvedCount:     baselineNormalized.DecimalsUnresolved + scanNormalized.DecimalsUnresolved,
		RetryExhausted:              baselineFetch.RetryExhausted || scanFetch.RetryExhausted,
	}, nil
}

type normalizeWindowResult struct {
	Transfers          []transactions.NormalizedTransfer
	Observations       []WalletTransferObservation
	FetchedTx          int
	FailedNormalize    int
	OwnerUnresolved    int
	DecimalsUnresolved int
}

func normalizeWindow(focalWallet string, txs []helius.EnhancedTransaction, classifyDust func(tr transactions.NormalizedTransfer) transactions.DustStatus) (normalizeWindowResult, error) {
	out := normalizeWindowResult{
		Transfers:    make([]transactions.NormalizedTransfer, 0),
		Observations: make([]WalletTransferObservation, 0, len(txs)),
		FetchedTx:    len(txs),
	}
	for _, tx := range txs {
		normalized, err := transactions.NormalizeEnhancedTx(tx)
		if err != nil {
			return normalizeWindowResult{}, err
		}
		for _, tr := range normalized {
			if classifyDust != nil {
				tr.DustStatus = classifyDust(tr)
			}
			out.Transfers = append(out.Transfers, tr)
			if tr.NormalizationStatus != transactions.NormalizationResolved {
				out.FailedNormalize++
			}
			if tr.NormalizationStatus == transactions.NormalizationUnresolvedOwner {
				out.OwnerUnresolved++
			}
			if tr.AssetType == transactions.AssetTypeSPLFungible && tr.Decimals == nil {
				out.DecimalsUnresolved++
			}

			mapping, ok := counterparties.MapWalletRelation(focalWallet, tr)
			if !ok {
				continue
			}
			out.Observations = append(out.Observations, WalletTransferObservation{
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
