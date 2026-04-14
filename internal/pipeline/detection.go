package pipeline

import (
	"strings"

	"poisontrace/internal/transactions"
)

type GateState string

const (
	GatePass    GateState = "pass"
	GateFail    GateState = "fail"
	GateUnknown GateState = "unknown"
)

type CandidateGate struct {
	NormalizationResolved GateState
	PoisoningEligible     GateState
	Inbound               GateState
	ZeroOrDust            GateState
	NewCounterparty       GateState
	BaselineComplete      GateState
	LookalikeMatch        GateState
	RecencyValid          GateState
	AddressInequality     GateState
	MinInjectionCountMet  GateState
}

type CandidateDecision struct {
	CanEmit           bool
	IncompleteWindow  bool
	UnknownGateReason string
}

func EvaluateCandidate(g CandidateGate) CandidateDecision {
	required := []struct {
		name  string
		state GateState
	}{
		{name: "normalization_resolved", state: g.NormalizationResolved},
		{name: "poisoning_eligible", state: g.PoisoningEligible},
		{name: "inbound", state: g.Inbound},
		{name: "zero_or_dust", state: g.ZeroOrDust},
		{name: "is_new_counterparty", state: g.NewCounterparty},
		{name: "baseline_complete", state: g.BaselineComplete},
		{name: "lookalike_match", state: g.LookalikeMatch},
		{name: "recency_valid", state: g.RecencyValid},
		{name: "address_inequality", state: g.AddressInequality},
		{name: "min_injection_count_met", state: g.MinInjectionCountMet},
	}

	var unknown []string
	for _, gate := range required {
		if gate.state == GateUnknown {
			unknown = append(unknown, gate.name)
		}
	}
	if len(unknown) > 0 {
		return CandidateDecision{
			CanEmit:           false,
			IncompleteWindow:  true,
			UnknownGateReason: "unknown_required_gates:" + strings.Join(unknown, ","),
		}
	}
	for _, gate := range required {
		if gate.state != GatePass {
			return CandidateDecision{CanEmit: false}
		}
	}
	return CandidateDecision{CanEmit: true}
}

func CanEmitCandidate(g CandidateGate) bool {
	return EvaluateCandidate(g).CanEmit
}

func IsZeroOrDust(t transactions.NormalizedTransfer) bool {
	return t.AmountRaw == "0" || t.DustStatus == transactions.DustTrue
}
