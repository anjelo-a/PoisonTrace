# Phase 3 Closeout (Strict Checklist Pass)

Date:
- 2026-04-24

Scope:
- Validation and tuning closeout for Phase 3.
- No detection-gate expansion; all fail-closed invariants remain unchanged.

## 1) Known Poisoning Corpus Scope (Locked)

Source of truth:
- Canonical fixture corpus under `data/fixtures/*` with `meta.json` fields:
- `expected_in_scope`
- `expected_miss_reason` (required for out-of-scope cases)

Corpus inventory:
- Total fixture cases: 19
- In-scope cases: 7
- Out-of-scope cases: 12

In-scope case IDs:
1. `lookalike_prefix_only_pass`
2. `lookalike_suffix_only_pass`
3. `multi_legit_match_tiebreak`
4. `out_of_order_events_same_signature`
5. `rate_limited_then_retry_success`
6. `repeat_inbound_two_injections_pass`
7. `scan_boundary_exact_timestamp`

Out-of-scope case IDs:
1. `baseline_truncated_newness_unknown`
2. `duplicate_event_across_pages`
3. `legit_baseline_outbound_non_dust_only`
4. `max_tx_cap_truncated`
5. `missing_threshold_dust_unknown`
6. `partial_owner_present`
7. `same_signature_multiple_wallets`
8. `self_transfer_owner_level`
9. `single_injection_fail_min_count`
10. `spl_unresolved_owner_non_poisoning_ready`
11. `two_injection_gate_with_unknown_second`
12. `wallet_timeout_partial`

## 2) Phase 3 Pass Targets (Locked)

Targets for phase exit:
1. Fixture replay parity: `0` expected-file mismatches.
2. Known in-scope recall: `>= 1.00` (case-level).
3. Out-of-scope false-positive rate: `<= 0.00` (case-level).
4. Strict miss-reason evidence pass: `100%` for out-of-scope cases with expected miss reason.
5. Reproducibility: strict validation report must be byte-identical across reruns.

## 3) Threshold Tuning Cycles (Completed)

### Cycle A: Baseline Threshold Evaluation

Configuration:
- `LOOKALIKE_RECENCY_DAYS=30`
- `LOOKALIKE_PREFIX_MIN=4`
- `LOOKALIKE_SUFFIX_MIN=4`
- `LOOKALIKE_SINGLE_SIDE_MIN=6`
- `MIN_INJECTION_COUNT=2`

Command:
- `go run ./cmd/scanner validate-corpus --fixtures-root data/fixtures --report-out ./phase3_baseline_report.json`

Result:
- `cases=19 passed=19 failed=0 recall=1.000 false_positive_rate=0.000`

Decision:
- No threshold change required.

Rationale:
- Targets met with full deterministic parity and no out-of-scope emissions.

### Cycle B: Strict Miss-Reason Evidence Evaluation

Command:
- `go run ./cmd/scanner validate-corpus --fixtures-root data/fixtures --strict-miss-reason --report-out ./phase3_strict_report.json`

Result:
- `cases=19 passed=19 failed=0 recall=1.000 false_positive_rate=0.000`

Decision:
- No threshold change required.

Rationale:
- Misses/blocks are explainable from persisted signals under strict validation mode.

### Cycle C: Reproducibility Verification

Command:
- `go run ./cmd/scanner validate-corpus --fixtures-root data/fixtures --strict-miss-reason --report-out ./phase3_strict_report_rerun.json`
- `shasum -a 256 phase3_strict_report.json phase3_strict_report_rerun.json`

Result:
- Hashes match exactly:
- `51916883035c83c0d5e93472093813a983d21336948889a46671ce2cc60703c7`

Decision:
- Validation outcomes are reproducible.

## 4) Strict Exit Checklist

1. Target recall is reached for known in-scope cases: `MET`
- Evidence: strict/baseline reports show `detected_in_scope_cases=7` of `expected_in_scope_cases=7`.

2. False positives are explainable from persisted evidence: `MET`
- Evidence: strict corpus validation passed with `failed_cases=0`; out-of-scope emissions `0`.

3. Threshold behavior is stable across reruns: `MET`
- Evidence: baseline and strict runs both pass with identical summary metrics.

4. Validation outcomes are reproducible: `MET`
- Evidence: strict report rerun is byte-identical (same SHA-256 hash).

Phase 3 closeout verdict:
- `PASS`
