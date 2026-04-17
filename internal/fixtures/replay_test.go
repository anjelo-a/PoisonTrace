package fixtures

import (
	"context"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func fixturesRootForTest() string {
	return filepath.Join("..", "..", "data", "fixtures")
}

func TestCanonicalFixtureInventory(t *testing.T) {
	root := fixturesRootForTest()
	caseIDs, err := ListCaseIDs(root)
	if err != nil {
		t.Fatalf("list cases: %v", err)
	}
	want := append([]string{}, CanonicalCaseIDs...)
	sort.Strings(want)
	if !reflect.DeepEqual(caseIDs, want) {
		t.Fatalf("canonical fixture inventory mismatch\nwant=%v\ngot=%v", CanonicalCaseIDs, caseIDs)
	}
}

func TestReplayCanonicalFixtures(t *testing.T) {
	root := fixturesRootForTest()
	caseIDs, err := ListCaseIDs(root)
	if err != nil {
		t.Fatalf("list cases: %v", err)
	}

	for _, caseID := range caseIDs {
		caseID := caseID
		t.Run(caseID, func(t *testing.T) {
			fx, err := LoadCase(root, caseID)
			if err != nil {
				t.Fatalf("load case: %v", err)
			}
			out, err := Replay(context.Background(), fx)
			if err != nil {
				t.Fatalf("replay: %v", err)
			}
			if err := CompareExpected(fx, out); err != nil {
				t.Fatalf("compare expected: %v", err)
			}

			switch caseID {
			case "baseline_truncated_newness_unknown", "missing_threshold_dust_unknown", "two_injection_gate_with_unknown_second":
				if out.IngestionRunDelta.PoisoningCandidatesInserted != 0 {
					t.Fatalf("expected no emitted candidates under unknown required gate, got %d", out.IngestionRunDelta.PoisoningCandidatesInserted)
				}
				hasIncomplete := false
				for _, ws := range out.WalletSyncRuns {
					if ws.IncompleteWindow {
						hasIncomplete = true
						break
					}
				}
				if !hasIncomplete {
					t.Fatal("expected incomplete_window=true when unknown required gate occurs")
				}
			}
		})
	}
}
