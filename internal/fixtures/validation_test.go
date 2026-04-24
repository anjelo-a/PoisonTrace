package fixtures

import (
	"context"
	"testing"
)

func TestValidateCorpusCanonicalFixtures(t *testing.T) {
	report, err := ValidateCorpus(context.Background(), fixturesRootForTest(), CorpusValidationOptions{})
	if err != nil {
		t.Fatalf("validate corpus: %v", err)
	}
	if report.Summary.TotalCases == 0 {
		t.Fatalf("expected non-empty fixture corpus")
	}
	if report.Summary.FailedCases != 0 {
		t.Fatalf("expected zero failed cases, got %d", report.Summary.FailedCases)
	}
	if report.Summary.ExpectedInScopeCases == 0 {
		t.Fatalf("expected at least one in-scope case")
	}
	if report.Summary.CaseLevelRecall <= 0 {
		t.Fatalf("expected positive recall, got %f", report.Summary.CaseLevelRecall)
	}
}

func TestValidateCorpusCanonicalFixtures_StrictMissReason(t *testing.T) {
	report, err := ValidateCorpus(context.Background(), fixturesRootForTest(), CorpusValidationOptions{
		StrictMissReason: true,
	})
	if err != nil {
		t.Fatalf("validate corpus (strict): %v", err)
	}
	if report.Summary.TotalCases == 0 {
		t.Fatalf("expected non-empty fixture corpus")
	}
	if report.Summary.FailedCases != 0 {
		t.Fatalf("expected zero failed cases under strict miss reason, got %d", report.Summary.FailedCases)
	}
}

func TestEvaluateMissReasonEvidence_KnownSignals(t *testing.T) {
	fx, err := LoadCase(fixturesRootForTest(), "missing_threshold_dust_unknown")
	if err != nil {
		t.Fatalf("load case: %v", err)
	}
	out, err := Replay(context.Background(), fx)
	if err != nil {
		t.Fatalf("replay case: %v", err)
	}
	supported, matched, _ := evaluateMissReasonEvidence("unknown_dust_threshold", out)
	if !supported {
		t.Fatalf("expected reason to be supported")
	}
	if !matched {
		t.Fatalf("expected reason to match observed signals")
	}
}
