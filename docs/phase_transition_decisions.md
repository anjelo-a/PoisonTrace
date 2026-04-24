# Phase Transition Decisions (Locked)

This document locks the required decisions from `AGENTS.md` before starting the next step after Phase 0–2.

Lock date:
- 2026-04-24

Phase 3 closeout evidence:
- [`docs/phase3_closeout.md`](./phase3_closeout.md)

## 1. Scan and baseline window bounds

Locked:
- Baseline window uses `BASELINE_LOOKBACK_DAYS` (default `90`).
- Scan window uses `SCAN_WINDOW_DAYS` (default `7`).
- Windows are contiguous and non-overlapping: `baseline_end == scan_start`.
- Window semantics remain `[start, end)` for both baseline and scan.

References:
- `internal/config/config.go`
- `internal/pipeline/validation.go`
- `data/fixtures/README.md`

## 2. Baseline completeness behavior on truncation

Locked:
- `baseline_complete` is `true` only when baseline fetch is not partial/truncated.
- If baseline stops due to timeout, tx/page cap, retry exhaustion, cancellation, or failure, then:
- `baseline_complete = false`
- `incomplete_window = true`
- truncation/unknown reasons are persisted.

References:
- `internal/pipeline/core.go`
- `internal/pipeline/wallet_runner.go`
- `AGENTS.md` runtime guards

## 3. Dust threshold source for each asset

Locked:
- Dust thresholds are sourced from `asset_thresholds` and seeded from `DUST_THRESHOLDS_SEED_PATH` (default `data/seeds/asset_thresholds.seed.sql`).
- Thresholds are asset-specific (`SOL` or token mint asset key), not global.
- Missing threshold yields `is_dust = unknown`, blocks candidate emission, and marks `incomplete_window = true` with persisted reason.

References:
- `migrations/0002_phase1_detection.sql`
- `.env.example`
- `data/fixtures/missing_threshold_dust_unknown/*`

## 4. Lookalike thresholds and recency limits

Locked:
- Similarity rule: `(prefix >= 4 AND suffix >= 4) OR (prefix >= 6) OR (suffix >= 6)`.
- Recency gate: `0 < suspicious.block_time - legit.last_outbound_at <= LOOKALIKE_RECENCY_DAYS`.
- Defaults: `LOOKALIKE_RECENCY_DAYS=30`, `LOOKALIKE_PREFIX_MIN=4`, `LOOKALIKE_SUFFIX_MIN=4`, `LOOKALIKE_SINGLE_SIDE_MIN=6`.

References:
- `internal/pipeline/candidate_materialize.go`
- `internal/pipeline/validation.go`
- `.env.example`

## 5. Retry, timeout, and partial status semantics

Locked:
- Runtime bounds remain mandatory:
- `MAX_WALLETS_PER_RUN`
- `MAX_TX_PAGES_PER_WALLET`
- `MAX_TX_PER_WALLET`
- `MAX_CONCURRENT_WALLETS`
- `WALLET_SYNC_TIMEOUT_SECONDS`
- `RUN_TIMEOUT_SECONDS`
- `MAX_HELIUS_RETRIES`
- `HELIUS_REQUEST_DELAY_MS`
- Cap/timeout/retry-exhausted outcomes persist partial progress and explicit reasons.
- Wallet-level failures remain isolated and do not collapse the whole run.

References:
- `internal/config/config.go`
- `internal/pipeline/orchestrator.go`
- `internal/pipeline/fetch.go`
- `internal/pipeline/wallet_runner.go`

## 6. Candidate uniqueness and idempotency keys

Locked:
- Transfer canonical identity: `UNIQUE(signature, transfer_fingerprint)`.
- Candidate identity per wallet sync: `UNIQUE(wallet_sync_run_id, signature, transfer_index)`.
- Upserts and reruns must remain deterministic and non-duplicating.

References:
- `migrations/0001_phase0_core.sql`
- `migrations/0002_phase1_detection.sql`
- `internal/storage/postgres_repository.go`

## 7. Fixture pass criteria against known poisoning corpus

Locked:
- Canonical fixture replay must match all `expected/*.json` outputs exactly for each fixture case.
- Unknown required-gate scenarios must emit zero candidates and persist `incomplete_window = true` plus reason metadata.
- For known poisoning corpus fixtures:
- `expected_in_scope = true` requires candidate presence for the case.
- `expected_in_scope = false` requires no candidate emission.
- If a miss is expected, `expected_miss_reason` must be explicit and reproducible from persisted outputs.

References:
- `data/fixtures/README.md`
- `internal/fixtures/replay_test.go`
- `scripts/ci_guardrails.sh`

## Threshold Tuning Policy (Locked)

Locked policy for the next step:
- Tuning is configuration-only (threshold/env/fixture parameters), not heuristic code expansion.
- Any threshold change must include fixture updates and deterministic replay evidence.
- Tuning cannot weaken fail-closed gates, unknown-gate blocking, idempotency contracts, or bounded execution guarantees.
