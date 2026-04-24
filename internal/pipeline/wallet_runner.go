package pipeline

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"poisontrace/internal/config"
	"poisontrace/internal/counterparties"
	"poisontrace/internal/helius"
	"poisontrace/internal/runs"
	"poisontrace/internal/storage"
	"poisontrace/internal/transactions"
)

type WalletExecutionStore interface {
	storage.RunRepository
	storage.WalletRepository
	storage.TransactionRepository
	storage.WalletTransactionRepository
	storage.CounterpartyRepository
	storage.CandidateRepository
	storage.DustThresholdRepository
}

type WalletExecutionRunner struct {
	cfg    config.Config
	store  WalletExecutionStore
	client helius.Client
}

func NewWalletExecutionRunner(cfg config.Config, client helius.Client, store WalletExecutionStore) WalletRunnerFunc {
	r := &WalletExecutionRunner{
		cfg:    cfg,
		store:  store,
		client: client,
	}
	return r.RunWallet
}

func (r *WalletExecutionRunner) RunWallet(ctx context.Context, walletAddress string, p RunParams, limits WalletRunLimits) (report WalletRunReport, err error) {
	report = WalletRunReport{WalletStatus: runs.WalletStatusFailed}
	if strings.TrimSpace(walletAddress) == "" {
		return report, fmt.Errorf("wallet address is required")
	}
	if p.IngestionRunID == 0 {
		return report, fmt.Errorf("ingestion run id is required for wallet execution")
	}
	if r.store == nil {
		return report, fmt.Errorf("wallet execution store is required")
	}
	if r.client == nil {
		return report, fmt.Errorf("helius client is required")
	}

	window := runs.BuildWindow(p.ScanStart, p.ScanEnd, p.BaselineLookbackDays)
	walletID, err := r.store.EnsureWallet(ctx, walletAddress)
	if err != nil {
		return report, fmt.Errorf("ensure wallet: %w", err)
	}

	walletSyncRunID, err := r.store.CreateWalletSyncRun(ctx, p.IngestionRunID, walletID, window, time.Now().UTC())
	if err != nil {
		return report, fmt.Errorf("create wallet sync run: %w", err)
	}

	var progress storage.WalletSyncProgress
	status := runs.WalletStatusFailed
	errorCode := ""
	errorMessage := ""
	notes := ""
	progressPersisted := false

	defer func() {
		// Finalization runs in a bounded tail context so status is persisted even if the main wallet ctx is canceled.
		incomplete := progress.IncompleteWindow
		reason := progress.UnknownGateReason
		if incomplete && strings.TrimSpace(reason) == "" {
			reason = "unknown_required_gates:runner_incomplete_without_reason"
		}
		finalizeCtx, finalizeCancel := context.WithTimeout(context.WithoutCancel(ctx), time.Duration(walletFinalizeTimeoutSeconds)*time.Second)
		defer finalizeCancel()
		if finalizeErr := r.store.FinalizeWalletSyncRun(
			finalizeCtx,
			walletSyncRunID,
			status,
			time.Now().UTC(),
			incomplete,
			reason,
			errorCode,
			errorMessage,
			notes,
		); finalizeErr != nil {
			if err == nil {
				err = fmt.Errorf("finalize wallet sync run: %w", finalizeErr)
			} else {
				err = fmt.Errorf("%w; finalize wallet sync run: %v", err, finalizeErr)
			}
		}
		report.WalletStatus = status
		report.IncompleteWindow = progress.IncompleteWindow
		report.TruncationReason = progress.TruncationReason
		report.TruncationObserved = strings.TrimSpace(progress.TruncationReason) != ""
	}()

	classifyDust, err := r.buildDustClassifier(ctx, window.BaselineStart, window.ScanEnd)
	if err != nil {
		errorCode = "dust_threshold_load_failed"
		errorMessage = err.Error()
		status = runs.WalletStatusFailed
		return report, fmt.Errorf("build dust classifier: %w", err)
	}

	coreRes, err := RunWalletCoreSync(ctx, r.client, CoreSyncParams{
		FocalWalletAddress:     walletAddress,
		BaselineStart:          window.BaselineStart,
		BaselineEnd:            window.BaselineEnd,
		ScanStart:              window.ScanStart,
		ScanEnd:                window.ScanEnd,
		MaxTXPagesPerWallet:    limits.MaxTXPagesPerWallet,
		MaxTXPerWallet:         limits.MaxTXPerWallet,
		MaxHeliusRetries:       limits.MaxHeliusRetries,
		HeliusRequestDelay:     limits.HeliusRequestDelay,
		LookalikeRecencyDays:   r.cfg.LookalikeRecencyDays,
		LookalikePrefixMin:     r.cfg.LookalikePrefixMin,
		LookalikeSuffixMin:     r.cfg.LookalikeSuffixMin,
		LookalikeSingleSideMin: r.cfg.LookalikeSingleSideMin,
		MinInjectionCount:      r.cfg.MinInjectionCount,
		ClassifyDust:           classifyDust,
	})
	if err != nil {
		errorCode = "core_sync_failed"
		errorMessage = err.Error()
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			status = runs.WalletStatusTimedOut
			progress.IncompleteWindow = true
			progress.UnknownGateReason = mergeReasons(progress.UnknownGateReason, "unknown_required_gates:wallet_runner_timeout")
		} else if runs.IsPartial(progressPersisted, true) {
			status = runs.WalletStatusPartial
			progress.IncompleteWindow = true
			progress.UnknownGateReason = mergeReasons(progress.UnknownGateReason, "unknown_required_gates:wallet_runner_failure")
		} else {
			status = runs.WalletStatusFailed
		}
		return report, fmt.Errorf("run wallet core sync: %w", err)
	}

	report.Counters.TransactionsFetched = coreRes.TransactionsFetched
	report.Counters.TransactionsFailedNormalize = coreRes.TransactionsFailedNormalize
	report.Counters.OwnerUnresolvedCount = coreRes.OwnerUnresolvedCount
	report.Counters.DecimalsUnresolvedCount = coreRes.DecimalsUnresolvedCount
	if coreRes.RetryExhausted {
		report.Counters.RetryExhaustedCount = 1
	}

	allTransfers := dedupeTransfers(append(append([]transactions.NormalizedTransfer{}, coreRes.BaselineTransfers...), coreRes.ScanTransfers...))
	inserted, _, err := r.store.UpsertNormalizedTransfers(ctx, allTransfers)
	if err != nil {
		errorCode = "transfer_upsert_failed"
		errorMessage = err.Error()
		status = runs.WalletStatusFailed
		return report, fmt.Errorf("upsert normalized transfers: %w", err)
	}
	report.Counters.TransactionsInserted = inserted

	// Persistence order is intentional: transfers -> wallet links -> counterparties -> candidates.
	allObservations := append(append([]WalletTransferObservation{}, coreRes.BaselineObservations...), coreRes.ScanObservations...)
	for _, obs := range allObservations {
		linked, linkErr := r.store.LinkWalletTransfer(ctx, walletID, obs.RelationType, obs.Transfer)
		if linkErr != nil {
			errorCode = "wallet_transfer_link_failed"
			errorMessage = linkErr.Error()
			status = runs.WalletStatusPartial
			progress.IncompleteWindow = true
			progress.UnknownGateReason = mergeReasons(progress.UnknownGateReason, "unknown_required_gates:wallet_link_persistence_failed")
			return report, fmt.Errorf("link wallet transfer: %w", linkErr)
		}
		if !linked {
			continue
		}
		report.Counters.TransactionsLinked++

		created, updated, cpErr := r.store.UpsertCounterpartyEvent(ctx, counterparties.Event{
			FocalWalletID:       walletID,
			CounterpartyAddress: obs.CounterpartyAddress,
			RelationType:        obs.RelationType,
			OccurredAt:          obs.Transfer.BlockTime.UTC(),
		})
		if cpErr != nil {
			errorCode = "counterparty_upsert_failed"
			errorMessage = cpErr.Error()
			status = runs.WalletStatusPartial
			progress.IncompleteWindow = true
			progress.UnknownGateReason = mergeReasons(progress.UnknownGateReason, "unknown_required_gates:counterparty_persistence_failed")
			return report, fmt.Errorf("upsert counterparty event: %w", cpErr)
		}
		if created {
			report.Counters.CounterpartiesCreated++
		}
		if updated {
			report.Counters.CounterpartiesUpdated++
		}
	}

	candidateRecords := make([]storage.CandidateRecord, 0, len(coreRes.Candidates))
	for _, c := range coreRes.Candidates {
		candidateRecords = append(candidateRecords, storage.CandidateRecord{
			Signature:                c.Signature,
			TransferIndex:            c.TransferIndex,
			SuspiciousCounterparty:   c.SuspiciousCounterparty,
			MatchedLegitCounterparty: c.MatchedLegitCounterparty,
			TokenMint:                c.TokenMint,
			AmountRaw:                c.AmountRaw,
			BlockTime:                c.BlockTime,
			IsZeroValue:              c.IsZeroValue,
			IsDust:                   c.IsDust,
			IsNewCounterparty:        c.IsNewCounterparty,
			IsInbound:                c.IsInbound,
			LegitLastSeenAt:          c.LegitLastSeenAt,
			RecencyDays:              c.RecencyDays,
			RepeatInjectionCount:     c.RepeatInjectionCount,
			IncompleteWindow:         c.IncompleteWindow,
			UnknownGateReason:        c.UnknownGateReason,
			MatchRuleVersion:         c.MatchRuleVersion,
		})
	}

	insertedCandidates, err := r.store.InsertPoisoningCandidates(ctx, walletSyncRunID, walletID, candidateRecords)
	if err != nil {
		errorCode = "candidate_insert_failed"
		errorMessage = err.Error()
		status = runs.WalletStatusPartial
		progress.IncompleteWindow = true
		progress.UnknownGateReason = mergeReasons(progress.UnknownGateReason, "unknown_required_gates:candidate_persistence_failed")
		return report, fmt.Errorf("insert poisoning candidates: %w", err)
	}
	report.Counters.PoisoningCandidatesInserted = insertedCandidates

	// Progress is derived from persisted outcomes and core result semantics, not in-memory assumptions.
	progress = storage.WalletSyncProgress{
		BaselineComplete:            coreRes.BaselineComplete,
		IncompleteWindow:            coreRes.IncompleteWindow,
		UnknownGateReason:           coreRes.UnknownGateReason,
		TruncationReason:            mergeTruncationReason(coreRes.BaselineTruncation, coreRes.ScanTruncation),
		TransactionsFetched:         coreRes.TransactionsFetched,
		TransactionsInserted:        report.Counters.TransactionsInserted,
		TransactionsLinked:          report.Counters.TransactionsLinked,
		TransactionsFailedNormalize: coreRes.TransactionsFailedNormalize,
		CounterpartiesCreated:       report.Counters.CounterpartiesCreated,
		CounterpartiesUpdated:       report.Counters.CounterpartiesUpdated,
		PoisoningCandidatesInserted: insertedCandidates,
	}
	if progress.TruncationReason != "" {
		progress.IncompleteWindow = true
		progress.UnknownGateReason = mergeReasons(progress.UnknownGateReason, truncationReason("baseline", coreRes.BaselineTruncation), truncationReason("scan", coreRes.ScanTruncation))
	}
	if progress.IncompleteWindow && strings.TrimSpace(progress.UnknownGateReason) == "" {
		progress.UnknownGateReason = "unknown_required_gates:incomplete_window"
	}

	if err := r.store.UpdateWalletSyncProgress(ctx, walletSyncRunID, progress); err != nil {
		errorCode = "wallet_sync_progress_update_failed"
		errorMessage = err.Error()
		status = runs.WalletStatusPartial
		progress.IncompleteWindow = true
		progress.UnknownGateReason = mergeReasons(progress.UnknownGateReason, "unknown_required_gates:wallet_progress_persistence_failed")
		return report, fmt.Errorf("update wallet sync progress: %w", err)
	}
	progressPersisted = true

	if err := r.store.UpdateWalletLastSyncedAt(ctx, walletID, window.ScanEnd.UTC()); err != nil {
		errorCode = "wallet_last_synced_update_failed"
		errorMessage = err.Error()
		status = runs.WalletStatusPartial
		progress.IncompleteWindow = true
		progress.UnknownGateReason = mergeReasons(progress.UnknownGateReason, "unknown_required_gates:wallet_metadata_update_failed")
		return report, fmt.Errorf("update wallet last_synced_at: %w", err)
	}

	if progress.IncompleteWindow {
		status = runs.WalletStatusPartial
	} else {
		status = runs.WalletStatusSucceeded
	}
	notes = fmt.Sprintf("candidates_inserted=%d", insertedCandidates)

	return report, nil
}

type dustRule struct {
	From      time.Time
	To        *time.Time
	Threshold *big.Int
}

func (r *WalletExecutionRunner) buildDustClassifier(ctx context.Context, startInclusive, endExclusive time.Time) (func(tr transactions.NormalizedTransfer) transactions.DustStatus, error) {
	recs, err := r.store.ListDustThresholds(ctx, startInclusive, endExclusive)
	if err != nil {
		return nil, err
	}

	index := make(map[string][]dustRule)
	for _, rec := range recs {
		asset := strings.TrimSpace(rec.AssetKey)
		if asset == "" {
			return nil, fmt.Errorf("invalid dust threshold row: asset_key is empty (active_from=%s)", rec.ActiveFrom.UTC().Format(time.RFC3339))
		}

		v, ok := new(big.Int).SetString(strings.TrimSpace(rec.AmountRaw), 10)
		if !ok || v.Sign() < 0 {
			return nil, fmt.Errorf("invalid dust threshold row for asset %s: dust_amount_raw_threshold=%q", asset, rec.AmountRaw)
		}

		from := rec.ActiveFrom.UTC()
		var to *time.Time
		if rec.ActiveTo != nil {
			toUTC := rec.ActiveTo.UTC()
			if !toUTC.After(from) {
				return nil, fmt.Errorf("invalid dust threshold window for asset %s: active_to must be greater than active_from", asset)
			}
			to = &toUTC
		}
		index[asset] = append(index[asset], dustRule{
			From:      from,
			To:        to,
			Threshold: v,
		})
	}

	for asset := range index {
		sort.Slice(index[asset], func(i, j int) bool {
			return index[asset][i].From.Before(index[asset][j].From)
		})

		rules := index[asset]
		for i := 1; i < len(rules); i++ {
			prev := rules[i-1]
			curr := rules[i]
			if prev.To == nil || curr.From.Before(*prev.To) {
				return nil, fmt.Errorf("overlapping dust threshold windows for asset %s", asset)
			}
		}
	}

	return func(tr transactions.NormalizedTransfer) transactions.DustStatus {
		amount, ok := new(big.Int).SetString(strings.TrimSpace(tr.AmountRaw), 10)
		if !ok || amount.Sign() < 0 {
			return transactions.DustUnknown
		}
		if amount.Sign() == 0 {
			return transactions.DustTrue
		}

		rules := index[strings.TrimSpace(tr.AssetKey)]
		if len(rules) == 0 {
			return transactions.DustUnknown
		}

		at := tr.BlockTime.UTC()
		for i := len(rules) - 1; i >= 0; i-- {
			rule := rules[i]
			if at.Before(rule.From) {
				continue
			}
			if rule.To != nil && !at.Before(*rule.To) {
				continue
			}
			if amount.Cmp(rule.Threshold) <= 0 {
				return transactions.DustTrue
			}
			return transactions.DustFalse
		}
		return transactions.DustUnknown
	}, nil
}

func dedupeTransfers(in []transactions.NormalizedTransfer) []transactions.NormalizedTransfer {
	if len(in) == 0 {
		return nil
	}
	out := make([]transactions.NormalizedTransfer, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, tr := range in {
		key := tr.Signature + "\x00" + tr.TransferFingerprint
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, tr)
	}
	return out
}

func mergeTruncationReason(baselineCode, scanCode string) string {
	reasons := make([]string, 0, 2)
	if strings.TrimSpace(baselineCode) != "" {
		reasons = append(reasons, truncationReason("baseline", baselineCode))
	}
	if strings.TrimSpace(scanCode) != "" {
		reasons = append(reasons, truncationReason("scan", scanCode))
	}
	return strings.Join(reasons, ";")
}
