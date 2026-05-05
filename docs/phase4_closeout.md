# Phase 4 Closeout

Date: 2026-05-05
Owner: Codex + Anjelo

## Scope

- Profiles executed: `canary_low`, `standard_low`, `stress_low`
- Scan window: `2026-04-28T00:00:00Z` to `2026-05-05T00:00:00Z`
- Wallet set: 3 focal wallets (see `docs/phase4_profile_matrix.md`)
- Ingestion runs used for sign-off evidence: `6`, `7`, `8`

## Exit Criteria Results

1. Batch stability under caps: `MET`
- Runs completed as `partially_succeeded` with wallet-level isolation and no run-level crash.
- Wallet status pattern was stable across all stages: `partial:1,succeeded:2`.

2. Dataset reproducibility: `MET`
- Repro checks passed for all runs.
- Evidence:
  - `artifacts/phase4/repro_run_6/repro_check.txt`
  - `artifacts/phase4/repro_run_7/repro_check.txt`
  - `artifacts/phase4/repro_run_8/repro_check.txt`

3. Truncation/cost envelope measured: `MET`
- `truncation_wallet_count=0`, `truncation_wallet_rate=0.0` for runs `6`, `7`, `8`.
- Per-run counters persisted in `ingestion_runs` and export manifests.

4. Candidate distribution analyzable from persisted outputs: `MET`
- Candidate count was `0` across the selected window and wallet set.
- This is analyzable and reproducible from persisted gating signals; no hidden logic path used.

5. Fail-closed unknown-gate handling preserved: `MET`
- Incomplete windows had persisted reason (`unknown_required_gates:zero_or_dust`).
- Integrity checks confirmed no missing required reason metadata.

## Evidence Artifacts

- Preflight:
  - `artifacts/phase4/preflight/corpus_validation_strict.json`
  - `artifacts/phase4/preflight/bounds.env.snapshot`
- Integrity checks:
  - `artifacts/phase4/run_6/integrity_check.txt`
  - `artifacts/phase4/run_7/integrity_check.txt`
  - `artifacts/phase4/run_8/integrity_check.txt`
- Reproducibility checks:
  - `artifacts/phase4/repro_run_6/repro_check.txt`
  - `artifacts/phase4/repro_run_7/repro_check.txt`
  - `artifacts/phase4/repro_run_8/repro_check.txt`
- Dataset exports:
  - `artifacts/phase4/canary_low_run_6/`
  - `artifacts/phase4/standard_low_run_7/`
  - `artifacts/phase4/stress_low_run_8/`

## Metrics Summary

From ingestion runs `6`, `7`, `8`:
- wallets requested: `3` each run
- wallets processed: `3` each run
- wallets failed: `0` each run
- transactions fetched: `15` each run
- transactions inserted: `50` (run 6), `0` (run 7), `0` (run 8)
- poisoning candidates inserted: `0` each run
- truncation wallet count: `0` each run
- truncation wallet rate: `0.0` each run

Normalization snapshot across persisted transactions:
- `resolved=46`
- `failed=4`
- `dust_status true=4`
- `dust_status unknown=46`

## Notes

- Run 6 inserted net-new transfers; runs 7 and 8 demonstrated idempotent reruns (no duplicate insertions) while preserving identical export reproducibility under unchanged source filters.
- Prior exploratory runs (`3`, `4`, `5`) used high-throughput system/program addresses and were excluded from sign-off because they forced full truncation behavior; they remain useful as stress/truncation references only.

## Final Verdict

- `PASS`
- Phase 4 execution criteria are met for bounded batch execution, reproducible exports, and fail-closed gating evidence.
