package fixtures

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

type CorpusValidationOptions struct {
	StrictMissReason bool
}

type CorpusValidationSummary struct {
	TotalCases                 int     `json:"total_cases"`
	PassedCases                int     `json:"passed_cases"`
	FailedCases                int     `json:"failed_cases"`
	ExpectedInScopeCases       int     `json:"expected_in_scope_cases"`
	DetectedInScopeCases       int     `json:"detected_in_scope_cases"`
	ExpectedOutOfScopeCases    int     `json:"expected_out_of_scope_cases"`
	OutOfScopeWithCandidates   int     `json:"out_of_scope_with_candidates"`
	CaseLevelRecall            float64 `json:"case_level_recall"`
	CaseLevelFalsePositiveRate float64 `json:"case_level_false_positive_rate"`
}

type CorpusValidationCaseResult struct {
	CaseID                string   `json:"case_id"`
	ExpectedInScope       bool     `json:"expected_in_scope"`
	ActualInScope         bool     `json:"actual_in_scope"`
	ExpectedMissReason    string   `json:"expected_miss_reason,omitempty"`
	MissReasonSupported   bool     `json:"miss_reason_supported"`
	MissReasonMatched     bool     `json:"miss_reason_matched"`
	ObservedMissSignals   []string `json:"observed_miss_signals,omitempty"`
	ReplayMatchedExpected bool     `json:"replay_matched_expected"`
	Passed                bool     `json:"passed"`
	Errors                []string `json:"errors,omitempty"`
}

type CorpusValidationReport struct {
	FixturesRoot     string                       `json:"fixtures_root"`
	StrictMissReason bool                         `json:"strict_miss_reason"`
	Summary          CorpusValidationSummary      `json:"summary"`
	Cases            []CorpusValidationCaseResult `json:"cases"`
}

func ValidateCorpus(ctx context.Context, fixturesRoot string, opts CorpusValidationOptions) (CorpusValidationReport, error) {
	caseIDs, err := ListCaseIDs(fixturesRoot)
	if err != nil {
		return CorpusValidationReport{}, err
	}
	results := make([]CorpusValidationCaseResult, 0, len(caseIDs))
	summary := CorpusValidationSummary{TotalCases: len(caseIDs)}

	for _, caseID := range caseIDs {
		res := CorpusValidationCaseResult{CaseID: caseID, Passed: true}

		fx, err := LoadCase(fixturesRoot, caseID)
		if err != nil {
			res.Passed = false
			res.Errors = append(res.Errors, fmt.Sprintf("load_case_failed:%v", err))
			results = append(results, res)
			continue
		}

		res.ExpectedInScope = fx.Meta.ExpectedInScope
		res.ExpectedMissReason = strings.TrimSpace(fx.Meta.ExpectedMissReason)
		if res.ExpectedInScope {
			summary.ExpectedInScopeCases++
		} else {
			summary.ExpectedOutOfScopeCases++
		}

		out, err := Replay(ctx, fx)
		if err != nil {
			res.Passed = false
			res.Errors = append(res.Errors, fmt.Sprintf("replay_failed:%v", err))
			results = append(results, res)
			continue
		}

		if err := CompareExpected(fx, out); err != nil {
			res.Passed = false
			res.ReplayMatchedExpected = false
			res.Errors = append(res.Errors, fmt.Sprintf("expected_mismatch:%v", err))
		} else {
			res.ReplayMatchedExpected = true
		}

		res.ActualInScope = len(out.PoisoningCandidates) > 0
		if res.ActualInScope {
			summary.DetectedInScopeCases++
		}

		if res.ExpectedInScope && !res.ActualInScope {
			res.Passed = false
			res.Errors = append(res.Errors, "expected_in_scope_case_missing_candidate")
		}
		if !res.ExpectedInScope && res.ActualInScope {
			res.Passed = false
			summary.OutOfScopeWithCandidates++
			res.Errors = append(res.Errors, "out_of_scope_case_emitted_candidate")
		}

		if !res.ExpectedInScope && !res.ActualInScope && res.ExpectedMissReason != "" {
			supported, matched, signals := evaluateMissReasonEvidence(res.ExpectedMissReason, out)
			res.MissReasonSupported = supported
			res.MissReasonMatched = matched
			res.ObservedMissSignals = signals
			if opts.StrictMissReason && (!supported || !matched) {
				res.Passed = false
				if !supported {
					res.Errors = append(res.Errors, "expected_miss_reason_not_supported")
				} else {
					res.Errors = append(res.Errors, "expected_miss_reason_not_matched")
				}
			}
		}

		results = append(results, res)
	}

	sort.Slice(results, func(i, j int) bool { return results[i].CaseID < results[j].CaseID })
	for _, res := range results {
		if res.Passed {
			summary.PassedCases++
		} else {
			summary.FailedCases++
		}
	}
	if summary.ExpectedInScopeCases > 0 {
		summary.CaseLevelRecall = float64(summary.DetectedInScopeCases) / float64(summary.ExpectedInScopeCases)
	}
	if summary.ExpectedOutOfScopeCases > 0 {
		summary.CaseLevelFalsePositiveRate = float64(summary.OutOfScopeWithCandidates) / float64(summary.ExpectedOutOfScopeCases)
	}

	return CorpusValidationReport{
		FixturesRoot:     fixturesRoot,
		StrictMissReason: opts.StrictMissReason,
		Summary:          summary,
		Cases:            results,
	}, nil
}

func evaluateMissReasonEvidence(expectedReason string, out ReplayOutput) (supported bool, matched bool, signals []string) {
	signals = collectObservedMissSignals(out)
	hasSignal := make(map[string]struct{}, len(signals))
	for _, s := range signals {
		hasSignal[s] = struct{}{}
	}
	has := func(prefix string) bool { return hasSignalPrefix(hasSignal, prefix) }
	notHas := func(prefix string) bool { return !has(prefix) }

	switch strings.TrimSpace(expectedReason) {
	case "baseline_truncated":
		return true, has("truncation:baseline_truncation:"), signals
	case "scan_truncated":
		return true, has("truncation:scan_truncation:"), signals
	case "wallet_timeout":
		return true, has("unknown_gate:wallet_timeout"), signals
	case "unknown_dust_threshold":
		return true, has("unknown_gate:zero_or_dust") && has("dust_status:unknown"), signals
	case "unknown_second_injection":
		return true, has("unknown_gate:min_injection_count_met") && has("unknown_gate:zero_or_dust"), signals
	case "unresolved_owner":
		return true, has("normalization_status:unresolved_owner"), signals
	case "partial_owner_unresolved":
		// Some fixtures model this as complete suppression before a normalized row exists.
		return true, has("normalization_reason:missing_spl_owner_endpoint") || (has("candidate:none") && has("transactions_fetched:0")), signals
	case "self_transfer":
		return true, has("normalization_reason:self_transfer_owner_level"), signals
	case "duplicate_no_new_signal":
		return true, has("candidate:none") && notHas("unknown_gate:") && notHas("truncation:"), signals
	case "insufficient_injections":
		return true, has("candidate:none") && has("dust_status:true") && notHas("unknown_gate:") && notHas("truncation:"), signals
	case "min_injection_not_met":
		return true, has("candidate:none") && has("dust_status:true") && notHas("unknown_gate:") && notHas("truncation:"), signals
	case "no_legit_outbound_baseline":
		return true, has("candidate:none") && has("dust_status:false") && notHas("unknown_gate:") && notHas("truncation:"), signals
	default:
		return false, false, signals
	}
}

func hasSignalPrefix(signals map[string]struct{}, prefix string) bool {
	for signal := range signals {
		if strings.HasPrefix(signal, prefix) {
			return true
		}
	}
	return false
}

func collectObservedMissSignals(out ReplayOutput) []string {
	signalSet := make(map[string]struct{})

	if len(out.PoisoningCandidates) > 0 {
		signalSet["candidate:emitted"] = struct{}{}
	} else {
		signalSet["candidate:none"] = struct{}{}
	}

	for _, run := range out.WalletSyncRuns {
		signalSet[fmt.Sprintf("transactions_fetched:%d", run.TransactionsFetched)] = struct{}{}
		for _, token := range splitReasonTokens(run.UnknownGateReason) {
			if strings.HasPrefix(token, "unknown_required_gates:") {
				names := strings.TrimPrefix(token, "unknown_required_gates:")
				for _, gate := range strings.Split(names, ",") {
					gate = strings.TrimSpace(gate)
					if gate != "" {
						signalSet["unknown_gate:"+gate] = struct{}{}
					}
				}
				continue
			}
			signalSet["unknown_reason:"+token] = struct{}{}
		}
		for _, token := range splitReasonTokens(run.TruncationReason) {
			signalSet["truncation:"+token] = struct{}{}
		}
		if !run.BaselineComplete {
			signalSet["baseline_complete:false"] = struct{}{}
		}
	}

	for _, tr := range out.NormalizedTransfers {
		if strings.TrimSpace(tr.NormalizationStatus) != "" {
			signalSet["normalization_status:"+tr.NormalizationStatus] = struct{}{}
		}
		if strings.TrimSpace(tr.NormalizationReasonCode) != "" {
			signalSet["normalization_reason:"+tr.NormalizationReasonCode] = struct{}{}
		}
		if strings.TrimSpace(tr.DustStatus) != "" {
			signalSet["dust_status:"+tr.DustStatus] = struct{}{}
		}
	}

	signals := make([]string, 0, len(signalSet))
	for signal := range signalSet {
		signals = append(signals, signal)
	}
	sort.Strings(signals)
	return signals
}

func splitReasonTokens(reason string) []string {
	parts := strings.Split(reason, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
