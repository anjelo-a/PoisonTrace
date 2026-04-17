package pipeline

import (
	"math/big"
	"sort"
	"strings"
	"time"

	"poisontrace/internal/counterparties"
	"poisontrace/internal/transactions"
)

type WalletTransferObservation struct {
	Transfer            transactions.NormalizedTransfer
	RelationType        counterparties.RelationType
	CounterpartyAddress string
}

type CandidateMaterializeParams struct {
	BaselineComplete       bool
	LookalikeRecencyDays   int
	LookalikePrefixMin     int
	LookalikeSuffixMin     int
	LookalikeSingleSideMin int
	MinInjectionCount      int
}

type PoisoningCandidate struct {
	Signature                string
	TransferIndex            int
	SuspiciousCounterparty   string
	MatchedLegitCounterparty string
	TokenMint                string
	AmountRaw                string
	BlockTime                time.Time
	IsZeroValue              bool
	IsDust                   bool
	IsNewCounterparty        bool
	IsInbound                bool
	LegitLastSeenAt          time.Time
	RecencyDays              int
	RepeatInjectionCount     int
	IncompleteWindow         bool
	UnknownGateReason        string
	MatchRuleVersion         string
}

type CandidateMaterializationResult struct {
	Candidates        []PoisoningCandidate
	IncompleteWindow  bool
	UnknownGateReason string
}

func MaterializeCandidates(baseline []WalletTransferObservation, scan []WalletTransferObservation, p CandidateMaterializeParams) CandidateMaterializationResult {
	res := CandidateMaterializationResult{
		Candidates: make([]PoisoningCandidate, 0),
	}
	if err := ValidateCandidateMaterializeParams(p); err != nil {
		res.IncompleteWindow = true
		res.UnknownGateReason = "invalid_candidate_params:" + err.Error()
		return res
	}

	preWindowSeen := buildPreWindowInteractionSet(baseline)
	legitBaseline := buildLegitBaselineCounterparties(baseline)
	inboundStats := buildInboundCounterpartyStats(scan)

	unknownGates := make(map[string]struct{})
	for _, obs := range scan {
		baseGates := evaluateBaseEmissionGates(obs)
		gates := CandidateGate{
			NormalizationResolved: baseGates.NormalizationResolved,
			PoisoningEligible:     baseGates.PoisoningEligible,
			AssetTypeSupported:    baseGates.AssetTypeSupported,
			Inbound:               baseGates.Inbound,
			ZeroOrDust:            gateZeroOrDust(obs.Transfer),
			NewCounterparty:       gateNewCounterparty(p.BaselineComplete, preWindowSeen[obs.CounterpartyAddress]),
			BaselineComplete:      gateBool(p.BaselineComplete),
		}

		matchedLegit, recencyDays, recencyGate, lookalikeGate := evaluateLookalikeAndRecency(obs, legitBaseline, p)
		gates.LookalikeMatch = lookalikeGate
		gates.RecencyValid = recencyGate
		gates.AddressInequality = gateAddressInequality(obs.CounterpartyAddress, matchedLegit, lookalikeGate)
		gates.MinInjectionCountMet = gateMinInjectionCount(inboundStats[obs.CounterpartyAddress], p.MinInjectionCount)

		decision := EvaluateCandidate(gates)
		if decision.IncompleteWindow {
			res.IncompleteWindow = true
			addUnknownGates(unknownGates, decision.UnknownGateReason)
		}
		if !decision.CanEmit {
			continue
		}

		isZero, _ := isAmountZero(obs.Transfer.AmountRaw)
		repeatCount := inboundStats[obs.CounterpartyAddress].KnownQualifying
		candidate := PoisoningCandidate{
			Signature:                obs.Transfer.Signature,
			TransferIndex:            obs.Transfer.TransferIndex,
			SuspiciousCounterparty:   obs.CounterpartyAddress,
			MatchedLegitCounterparty: matchedLegit,
			TokenMint:                obs.Transfer.TokenMint,
			AmountRaw:                obs.Transfer.AmountRaw,
			BlockTime:                obs.Transfer.BlockTime.UTC(),
			IsZeroValue:              isZero,
			IsDust:                   obs.Transfer.DustStatus == transactions.DustTrue,
			IsNewCounterparty:        true,
			IsInbound:                true,
			LegitLastSeenAt:          legitBaseline[matchedLegit],
			RecencyDays:              recencyDays,
			RepeatInjectionCount:     repeatCount,
			IncompleteWindow:         false,
			UnknownGateReason:        "",
			MatchRuleVersion:         "phase1-v1",
		}
		res.Candidates = append(res.Candidates, candidate)
	}

	if res.IncompleteWindow {
		res.UnknownGateReason = buildUnknownGateReason(unknownGates)
	}
	return res
}

func buildPreWindowInteractionSet(baseline []WalletTransferObservation) map[string]bool {
	out := make(map[string]bool, len(baseline))
	for _, obs := range baseline {
		if strings.TrimSpace(obs.CounterpartyAddress) == "" {
			continue
		}
		out[obs.CounterpartyAddress] = true
	}
	return out
}

func buildLegitBaselineCounterparties(baseline []WalletTransferObservation) map[string]time.Time {
	out := make(map[string]time.Time)
	for _, obs := range baseline {
		if obs.RelationType != counterparties.RelationSender {
			continue
		}
		if strings.TrimSpace(obs.CounterpartyAddress) == "" {
			continue
		}
		if obs.Transfer.NormalizationStatus != transactions.NormalizationResolved || !obs.Transfer.PoisoningEligible {
			continue
		}
		// Legitimate baseline is outbound non-dust only.
		if obs.Transfer.DustStatus != transactions.DustFalse {
			continue
		}

		lastSeen := out[obs.CounterpartyAddress]
		if lastSeen.IsZero() || obs.Transfer.BlockTime.After(lastSeen) {
			out[obs.CounterpartyAddress] = obs.Transfer.BlockTime.UTC()
		}
	}
	return out
}

type inboundCounterpartyStats struct {
	KnownQualifying  int
	UnknownPotential int
}

type candidateBaseEmissionGates struct {
	NormalizationResolved GateState
	PoisoningEligible     GateState
	AssetTypeSupported    GateState
	Inbound               GateState
}

func buildInboundCounterpartyStats(scan []WalletTransferObservation) map[string]inboundCounterpartyStats {
	out := make(map[string]inboundCounterpartyStats)
	for _, obs := range scan {
		if !evaluateBaseEmissionGates(obs).canQualifyForMinInjection() {
			continue
		}
		cp := strings.TrimSpace(obs.CounterpartyAddress)
		if cp == "" {
			continue
		}

		stats := out[cp]
		switch gateZeroOrDust(obs.Transfer) {
		case GatePass:
			stats.KnownQualifying++
		case GateUnknown:
			stats.UnknownPotential++
		}
		out[cp] = stats
	}
	return out
}

func evaluateBaseEmissionGates(obs WalletTransferObservation) candidateBaseEmissionGates {
	return candidateBaseEmissionGates{
		NormalizationResolved: gateNormalizationResolved(obs.Transfer),
		PoisoningEligible:     gateBool(obs.Transfer.PoisoningEligible),
		AssetTypeSupported:    gateSupportedAssetType(obs.Transfer.AssetType),
		Inbound:               gateRelationInbound(obs.RelationType),
	}
}

func (g candidateBaseEmissionGates) canQualifyForMinInjection() bool {
	return g.Inbound == GatePass &&
		g.NormalizationResolved == GatePass &&
		g.PoisoningEligible == GatePass &&
		g.AssetTypeSupported == GatePass
}

func gateNormalizationResolved(tr transactions.NormalizedTransfer) GateState {
	if tr.NormalizationStatus == transactions.NormalizationResolved {
		return GatePass
	}
	return GateFail
}

func gateSupportedAssetType(assetType transactions.AssetType) GateState {
	switch assetType {
	case transactions.AssetTypeNativeSOL, transactions.AssetTypeSPLFungible:
		return GatePass
	default:
		return GateFail
	}
}

func gateRelationInbound(relation counterparties.RelationType) GateState {
	if relation == counterparties.RelationReceiver {
		return GatePass
	}
	return GateFail
}

func gateZeroOrDust(tr transactions.NormalizedTransfer) GateState {
	isZero, known := isAmountZero(tr.AmountRaw)
	if isZero && known {
		return GatePass
	}
	switch tr.DustStatus {
	case transactions.DustTrue:
		return GatePass
	case transactions.DustFalse:
		return GateFail
	default:
		return GateUnknown
	}
}

func gateNewCounterparty(baselineComplete bool, hadPreWindowInteraction bool) GateState {
	if !baselineComplete {
		return GateUnknown
	}
	if hadPreWindowInteraction {
		return GateFail
	}
	return GatePass
}

func gateAddressInequality(suspicious, matched string, lookalikeGate GateState) GateState {
	if lookalikeGate == GateUnknown {
		return GateUnknown
	}
	if lookalikeGate != GatePass {
		return GateFail
	}
	if strings.TrimSpace(suspicious) == "" || strings.TrimSpace(matched) == "" {
		return GateFail
	}
	if suspicious == matched {
		return GateFail
	}
	return GatePass
}

func gateMinInjectionCount(stats inboundCounterpartyStats, min int) GateState {
	if min < 2 {
		min = 2
	}
	if stats.KnownQualifying >= min {
		return GatePass
	}
	if stats.KnownQualifying+stats.UnknownPotential < min {
		return GateFail
	}
	return GateUnknown
}

func evaluateLookalikeAndRecency(obs WalletTransferObservation, legit map[string]time.Time, p CandidateMaterializeParams) (matchedLegit string, recencyDays int, recencyGate GateState, lookalikeGate GateState) {
	if !p.BaselineComplete {
		return "", 0, GateUnknown, GateUnknown
	}
	matchedLegit = ""
	bestScore := -1
	for candidate := range legit {
		prefix := commonPrefixLength(obs.CounterpartyAddress, candidate)
		suffix := commonSuffixLength(obs.CounterpartyAddress, candidate)
		if !passesLookalikeThreshold(prefix, suffix, p) {
			continue
		}
		score := prefix + suffix
		if score > bestScore || (score == bestScore && candidate < matchedLegit) {
			bestScore = score
			matchedLegit = candidate
		}
	}
	if matchedLegit == "" {
		return "", 0, GateFail, GateFail
	}
	legitAt := legit[matchedLegit]
	suspiciousAt := obs.Transfer.BlockTime.UTC()
	if legitAt.IsZero() || suspiciousAt.IsZero() {
		return matchedLegit, 0, GateUnknown, GatePass
	}
	delta := suspiciousAt.Sub(legitAt)
	if delta <= 0 {
		return matchedLegit, 0, GateFail, GatePass
	}
	limit := time.Duration(p.LookalikeRecencyDays) * 24 * time.Hour
	if delta > limit {
		return matchedLegit, int(delta.Hours() / 24), GateFail, GatePass
	}
	return matchedLegit, int(delta.Hours() / 24), GatePass, GatePass
}

func passesLookalikeThreshold(prefix, suffix int, p CandidateMaterializeParams) bool {
	return (prefix >= p.LookalikePrefixMin && suffix >= p.LookalikeSuffixMin) ||
		(prefix >= p.LookalikeSingleSideMin) ||
		(suffix >= p.LookalikeSingleSideMin)
}

func commonPrefixLength(a, b string) int {
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	n := 0
	for i := 0; i < limit; i++ {
		if a[i] != b[i] {
			break
		}
		n++
	}
	return n
}

func commonSuffixLength(a, b string) int {
	i := len(a) - 1
	j := len(b) - 1
	n := 0
	for i >= 0 && j >= 0 {
		if a[i] != b[j] {
			break
		}
		n++
		i--
		j--
	}
	return n
}

func isAmountZero(raw string) (bool, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, false
	}
	v, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return false, false
	}
	return v.Sign() == 0, true
}

func gateBool(v bool) GateState {
	if v {
		return GatePass
	}
	return GateFail
}

func addUnknownGates(set map[string]struct{}, reason string) {
	const prefix = "unknown_required_gates:"
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" {
		return
	}
	if strings.HasPrefix(trimmed, prefix) {
		names := strings.Split(strings.TrimPrefix(trimmed, prefix), ",")
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name != "" {
				set[name] = struct{}{}
			}
		}
		return
	}
	set[trimmed] = struct{}{}
}

func buildUnknownGateReason(set map[string]struct{}) string {
	if len(set) == 0 {
		return ""
	}
	names := make([]string, 0, len(set))
	for gateName := range set {
		names = append(names, gateName)
	}
	sort.Strings(names)
	return "unknown_required_gates:" + strings.Join(names, ",")
}
