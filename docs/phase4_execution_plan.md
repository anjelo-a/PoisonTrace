# Phase 4 Comprehensive Execution Plan

This document operationalizes Phase 4 into reproducible, bounded execution steps with explicit evidence gates.

## 1) Objective and Exit Criteria

Goal:
- Run bounded multi-wallet scans and generate reproducible poisoning datasets without violating Phase 0–1 invariants.

Phase 4 is complete only when all are true:
1. Batch runs are stable under configured caps.
2. Dataset exports are byte-reproducible for the same source filters and unchanged DB state.
3. Run-level truncation and cost envelopes are measured and documented.
4. Candidate distribution is analyzable from persisted outputs.
5. Unknown-gate and incomplete-window behavior remains fail-closed.

## 2) Pre-Execution Lock (Required Decisions)

Before first production-scale execution, lock these in a tracked note committed to repo history:
1. Scan window bounds (`scan_start`, `scan_end`).
2. Baseline window and truncation semantics (`baseline_complete=false` on truncation).
3. Dust threshold source and version.
4. Lookalike thresholds and recency limit.
5. Retry, timeout, and partial status semantics.
6. Candidate uniqueness and idempotency keys.
7. Fixture pass criteria version and fixture corpus commit SHA.

Use [`docs/phase4_profile_matrix.template.md`](./phase4_profile_matrix.template.md) as the lock artifact.

## 3) Run Profiles

Define and lock three profiles:
1. `canary`: small wallet set, short window, strict caps.
2. `standard`: medium wallet set, normal production bounds.
3. `stress`: upper-bound caps to validate stability envelopes.

Each profile must set:
- `MAX_WALLETS_PER_RUN`
- `MAX_TX_PAGES_PER_WALLET`
- `MAX_TX_PER_WALLET`
- `MAX_CONCURRENT_WALLETS`
- `WALLET_SYNC_TIMEOUT_SECONDS`
- `RUN_TIMEOUT_SECONDS`
- `MAX_HELIUS_RETRIES`
- `HELIUS_REQUEST_DELAY_MS`

## 4) Per-Batch Execution Workflow

For each batch, run:
1. Config and environment preflight.
2. Fixture gate preflight (`validate-corpus`, strict miss-reason enabled).
3. Bounded batch scan (`scanner run`) with fixed wallet seed and RFC3339 bounds.
4. Canonical dataset export (`export-dataset` by `run-id` or started_at window).
5. Artifact checksum capture.
6. Post-run integrity checks against persisted DB state.
7. Run report publication with metrics, anomalies, and truncation reasons.

Suggested command sequence:

```bash
# 1) preflight
./ops/scripts/phase4_preflight.sh

# 2) run bounded scan
go run ./cmd/scanner run \
  --wallets ./db/seeds/wallets.example.txt \
  --scan-start 2026-04-01T00:00:00Z \
  --scan-end 2026-04-08T00:00:00Z

# 3) export
RUN_ID=<ingestion_run_id>
OUT_DIR=./artifacts/phase4/run_${RUN_ID}
go run ./cmd/scanner export-dataset --out-dir "$OUT_DIR" --run-id "$RUN_ID"

# 4) integrity checks
./ops/scripts/phase4_integrity_check.sh --run-id "$RUN_ID"

# 5) reproducibility check (same source filter, unchanged DB state)
./ops/scripts/phase4_repro_check.sh --run-id "$RUN_ID"
```

## 5) Post-Run Integrity Checks

Mandatory checks per run:
1. No candidate emitted when any required gate is `UNKNOWN`.
2. Every `wallet_sync_runs.incomplete_window=true` row has non-empty `unknown_gate_reason`.
3. `baseline_complete` is never true for truncated/time-capped/timeout wallets.
4. No dedup violations:
- transfers unique on `(signature, transfer_fingerprint)`
- candidates unique on `(wallet_sync_run_id, signature, transfer_index)`
5. Counters reconcile from persisted outcomes.

Implemented helper:
- [`ops/scripts/phase4_integrity_check.sh`](../ops/scripts/phase4_integrity_check.sh)

## 6) Observability and Metrics Pack

Track and preserve per run:
1. Wallets processed, partial, failed, timed out.
2. Transactions fetched and normalization status distribution.
3. Candidates emitted by wallet and gate-path.
4. Truncation metrics:
- `truncation_wallet_count`
- `truncation_wallet_rate`
5. Unknown-gate reason frequency.
6. Retry counts and delay/backoff behavior.
7. Duration and throughput (`wallets/min`, `tx/min`).

Store alongside exported dataset under `artifacts/phase4/...`.

## 7) Reproducibility Protocol

For reproducibility evidence:
1. Re-run the same source filter against unchanged DB state.
2. Export twice to separate directories.
3. Compare SHA-256 hashes for:
- `ingestion_runs.jsonl`
- `wallet_sync_runs.jsonl`
- `poisoning_candidates.jsonl`
- `manifest.json`
4. Any mismatch is a release blocker.

Implemented helper:
- [`ops/scripts/phase4_repro_check.sh`](../ops/scripts/phase4_repro_check.sh)

## 8) Rollout Stages and Gates

1. Stage A: Canary
- Zero invariant violations.
- Reproducible exports.
- Acceptable truncation.

2. Stage B: Standard
- Stable for N consecutive runs (`N=5` recommended).

3. Stage C: Stress
- Bounded degradation only.
- No silent drops.
- No invariant violations.

4. Phase 4 sign-off
- All exit criteria met with retained evidence artifacts.

## 9) Failure Handling Playbook

If instability appears:
1. Stop scale-up.
2. Classify failure (timeout, cap truncation, retry exhaustion, normalization uncertainty, storage).
3. Preserve partial outputs and persisted reason fields.
4. Reduce concurrency/window and rerun canary.
5. Open blocker with root cause and mitigation before proceeding.

## 10) Deliverables Checklist

Required artifacts for Phase 4 closeout:
1. Locked config matrix (canary/standard/stress).
2. Runbook command SOP.
3. Automated integrity check report.
4. Reproducibility evidence (hash comparisons).
5. Phase 4 closeout document with trend summary and sign-off.

Template:
- [`docs/phase4_closeout.template.md`](./phase4_closeout.template.md)
