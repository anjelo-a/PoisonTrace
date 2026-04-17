package fixtures

import (
	"context"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
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
			assertUnknownGateGuardrails(t, caseID, out)
		})
	}
}

func assertUnknownGateGuardrails(t *testing.T, caseID string, out ReplayOutput) {
	t.Helper()

	candidatesByWallet := make(map[string]int, len(out.PoisoningCandidates))
	for _, candidate := range out.PoisoningCandidates {
		candidatesByWallet[candidate.FocalWallet]++
	}

	for _, run := range out.WalletSyncRuns {
		if !strings.Contains(run.UnknownGateReason, "unknown_required_gates:") {
			continue
		}
		if !run.IncompleteWindow {
			t.Fatalf("%s/%s: expected incomplete_window=true when unknown required gates occur", caseID, run.FocalWallet)
		}
		if run.PoisoningCandidatesInserted != 0 {
			t.Fatalf("%s/%s: expected no emitted candidates under unknown required gate, got wallet_sync_run poisoning_candidates_inserted=%d", caseID, run.FocalWallet, run.PoisoningCandidatesInserted)
		}
		if got := candidatesByWallet[run.FocalWallet]; got != 0 {
			t.Fatalf("%s/%s: expected no emitted candidate rows under unknown required gate, got %d", caseID, run.FocalWallet, got)
		}
	}
}
