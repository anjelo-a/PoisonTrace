package pipeline

import "testing"

func passGateSet() CandidateGate {
	return CandidateGate{
		NormalizationResolved: GatePass,
		PoisoningEligible:     GatePass,
		Inbound:               GatePass,
		ZeroOrDust:            GatePass,
		NewCounterparty:       GatePass,
		BaselineComplete:      GatePass,
		LookalikeMatch:        GatePass,
		RecencyValid:          GatePass,
		AddressInequality:     GatePass,
		MinInjectionCountMet:  GatePass,
	}
}

func TestEvaluateCandidate_EmitsWhenAllRequiredGatesPass(t *testing.T) {
	decision := EvaluateCandidate(passGateSet())
	if !decision.CanEmit {
		t.Fatal("expected candidate emission when all gates pass")
	}
	if decision.IncompleteWindow {
		t.Fatal("did not expect incomplete window when all gates pass")
	}
}

func TestEvaluateCandidate_BlocksOnUnknownRequiredGate(t *testing.T) {
	gates := passGateSet()
	gates.NewCounterparty = GateUnknown

	decision := EvaluateCandidate(gates)
	if decision.CanEmit {
		t.Fatal("expected candidate emission to be blocked when a required gate is unknown")
	}
	if !decision.IncompleteWindow {
		t.Fatal("expected incomplete window when a required gate is unknown")
	}
	if decision.UnknownGateReason == "" {
		t.Fatal("expected unknown gate reason to be persisted")
	}
}

func TestEvaluateCandidate_BlocksWithoutUnknownWhenGateFails(t *testing.T) {
	gates := passGateSet()
	gates.LookalikeMatch = GateFail

	decision := EvaluateCandidate(gates)
	if decision.CanEmit {
		t.Fatal("expected candidate emission to be blocked when a required gate fails")
	}
	if decision.IncompleteWindow {
		t.Fatal("did not expect incomplete window when no required gate is unknown")
	}
	if decision.UnknownGateReason != "" {
		t.Fatal("did not expect unknown gate reason when no required gate is unknown")
	}
}
