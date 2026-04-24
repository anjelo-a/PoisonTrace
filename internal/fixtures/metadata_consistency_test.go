package fixtures

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFixtureMetadataConsistency(t *testing.T) {
	root := fixturesRootForTest()
	caseIDs, err := ListCaseIDs(root)
	if err != nil {
		t.Fatalf("list fixture cases: %v", err)
	}

	for _, caseID := range caseIDs {
		caseID := caseID
		t.Run(caseID, func(t *testing.T) {
			fx, err := LoadCase(root, caseID)
			if err != nil {
				t.Fatalf("load case: %v", err)
			}

			expectedCandidates, err := readExpectedCandidates(fx.CaseDir)
			if err != nil {
				t.Fatalf("read expected candidates: %v", err)
			}

			hasExpectedCandidates := len(expectedCandidates) > 0
			expectedInScope := fx.Meta.ExpectedInScope
			expectedMissReason := strings.TrimSpace(fx.Meta.ExpectedMissReason)

			if expectedInScope && !hasExpectedCandidates {
				t.Fatalf("expected_in_scope=true requires at least one candidate row")
			}
			if !expectedInScope && hasExpectedCandidates {
				t.Fatalf("expected_in_scope=false requires zero candidate rows")
			}
			if expectedInScope && expectedMissReason != "" {
				t.Fatalf("expected_in_scope=true must not set expected_miss_reason (got %q)", expectedMissReason)
			}
			if !expectedInScope && expectedMissReason == "" {
				t.Fatalf("expected_in_scope=false must set expected_miss_reason")
			}
		})
	}
}

func readExpectedCandidates(caseDir string) ([]PoisoningCandidateRecord, error) {
	path := filepath.Join(caseDir, "expected", "poisoning_candidates.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var out []PoisoningCandidateRecord
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return out, nil
}
