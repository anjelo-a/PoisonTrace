package fixtures

import (
	"context"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
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
		t.Fatalf("canonical fixture inventory mismatch\nwant=%v\ngot=%v", want, caseIDs)
	}
}

func TestApplyMetaDefaults_MaxHeliusRetriesDefaultsWhenOmitted(t *testing.T) {
	meta := validMetaForDefaults()
	if err := applyMetaDefaults(&meta, metaFieldPresence{}); err != nil {
		t.Fatalf("apply defaults: %v", err)
	}
	if meta.MaxHeliusRetries != defaultMaxHeliusRetries {
		t.Fatalf("expected omitted max_helius_retries to default to %d, got %d", defaultMaxHeliusRetries, meta.MaxHeliusRetries)
	}
}

func TestApplyMetaDefaults_MaxHeliusRetriesAllowsExplicitZero(t *testing.T) {
	meta := validMetaForDefaults()
	meta.MaxHeliusRetries = 0
	if err := applyMetaDefaults(&meta, metaFieldPresence{MaxHeliusRetries: true}); err != nil {
		t.Fatalf("apply defaults: %v", err)
	}
	if meta.MaxHeliusRetries != 0 {
		t.Fatalf("expected explicit max_helius_retries=0 to remain 0, got %d", meta.MaxHeliusRetries)
	}
}

func validMetaForDefaults() Meta {
	baselineStart := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	baselineEnd := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)
	scanEnd := time.Date(2026, time.February, 8, 0, 0, 0, 0, time.UTC)
	return Meta{
		FocalWallet:   "Focal11111111111111111111111111111111111111111",
		BaselineStart: baselineStart,
		BaselineEnd:   baselineEnd,
		ScanStart:     baselineEnd,
		ScanEnd:       scanEnd,
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
