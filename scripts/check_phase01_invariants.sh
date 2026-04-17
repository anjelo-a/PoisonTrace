#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "[phase01-check] 1/5 shared gate helper guardrails"
go test ./internal/pipeline -run 'TestCandidateBaseGateHelpersAreSharedAcrossCodePaths'

echo "[phase01-check] 2/5 runtime entry-point fail-fast validation"
go test ./internal/pipeline -run 'TestWalletExecutionRunnerRejectsMissingRequiredInputs|TestRunWalletCoreSyncRejectsInvalidLookalikeThresholds|TestValidateCoreSyncParamsRejectsInvalidRuntimeBounds|TestMaterializeCandidatesFailsClosedOnInvalidParams'

echo "[phase01-check] 3/5 numeric and URL boundary validation"
go test ./internal/pipeline ./internal/helius ./internal/config -run 'TestRetryBackoffClampsLargeAttemptsWithoutOverflow|TestNewHTTPClientRequiresAbsoluteHTTPBaseURL|TestValidateRejectsMalformedURLs'

echo "[phase01-check] 4/5 unknown-gate fail-closed behavior"
go test ./internal/pipeline ./internal/fixtures -run 'TestEvaluateCandidate_BlocksOnUnknownRequiredGate|TestMaterializeCandidatesMinInjectionUnknownWhenDustUnknownCouldMeetThreshold|TestReplayCanonicalFixtures/(baseline_truncated_newness_unknown|missing_threshold_dust_unknown|two_injection_gate_with_unknown_second)$'

echo "[phase01-check] 5/5 min-injection base-eligibility gating"
go test ./internal/pipeline -run 'TestMaterializeCandidatesMinInjectionIgnoresNonBaseGateInboundEvents'

echo "[phase01-check] all Phase 0-1 checklist guardrails passed"
