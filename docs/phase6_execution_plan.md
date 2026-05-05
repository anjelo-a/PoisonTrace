# Phase 6 Comprehensive Implementation Execution Plan: Operational Hardening

This document operationalizes Phase 6 into deterministic runtime hardening work with explicit reliability evidence gates.

## 1) Objective and Exit Criteria

Goal:
- Guarantee safe behavior under repeated, partial, and interrupted execution with deterministic failure classification and auditable recovery evidence.

Phase 6 is complete only when all are true:
1. Terminal failure paths are classified into canonical `failure_class` values with stable `failure_reasons`.
2. Cancellation, timeout, retry exhaustion, and lock conflict paths always finalize run + wallet status with bounded tail contexts.
3. Reruns after interruption remain idempotent (no duplicate canonical transfer/candidate/link/counterparty rows).
4. Operational observability artifacts are exportable deterministically from persisted DB state.
5. Phase 0-1 invariants and fixture/CI guardrails continue to pass unchanged.

## 2) Locked Contracts (Required Before Code Merge)

Before implementation merge, lock these decisions in repo history:
1. Failure taxonomy contract version and backward-compatibility policy.
2. Reason token grammar and deterministic merge order.
3. Lock TTL and stale takeover policy (expiry threshold + takeover annotation token).
4. Retry envelope defaults (attempt cap, backoff cap, jitter contract).
5. Tail finalization time budgets for wallet and run finalization paths.
6. Operational export schema fields and ordering guarantees.
7. Phase 6 acceptance fixture set and strict pass criteria.

## 3) Workstreams

### 3.1 Failure Classification Layer (Canonical + Persisted)

Implement shared internal contract:
- `timeout`
- `canceled`
- `retry_exhausted`
- `lock_conflict`
- `persistence_error`
- `normalization_error`
- `upstream_error`
- `unknown_required_gate`

Required changes:
1. Add typed enum/constants shared by pipeline, storage, read API, and export.
2. Replace ad hoc free-form terminal classification strings with canonical `failure_class`.
3. Persist stable `failure_reasons` as deterministic semicolon-delimited tokens:
- trim whitespace
- dedupe
- lexicographic ordering
- stable output for identical inputs
4. Ensure run-level status rollups derive from persisted wallet outcomes and canonical classes.

### 3.2 Interruption Recovery and Finalization Guarantees

Required changes:
1. Enforce bounded finalization tail contexts for wallet and ingestion run completion paths.
2. Ensure context cancellation and timeout always persist terminal status, `failure_class`, and reason chain.
3. Add stale-lock recovery:
- validate lock TTL
- allow safe reacquire when expired
- persist lock takeover token in reasons/notes for auditability
4. Preserve idempotent rerun behavior after interruption; never duplicate canonical persisted rows.

### 3.3 Retry and Backoff Runtime Hardening

Required changes:
1. Consolidate retry boundaries across fetch/pagination paths.
2. Enforce bounded attempts + capped exponential backoff + deterministic jitter.
3. Persist retry-exhaustion counters + classifications at wallet and run levels.
4. Mark truncation/incomplete-window linkage explicitly on retry exhaustion and timeout paths.
5. Add guardrails preventing pagination/retry hidden unbounded loops.

### 3.4 Operational Observability (DB-First, Read-Only)

Required changes:
1. Add run and wallet operational snapshot queries:
- failure-class distribution
- retry-exhaustion distribution
- timeout/cancel rates
- truncation/incomplete-window rates
- lock-conflict counts
2. Extend dataset export with `operational_health` artifacts:
- `operational_health_runs.jsonl`
- `operational_health_wallet_sync.csv`
- manifest hashes with deterministic ordering
3. Keep observability strictly read-side. No detection gate changes.

### 3.5 Read APIs and CLI Surfaces

Required changes:
1. Add:
- `GET /api/ops/runs`
- `GET /api/ops/wallet-sync`
- optional `GET /api/ops/failures`
2. Ensure response schemas are deterministic and sourced directly from persisted fields.
3. Add Phase 6 validation command/script to produce machine-readable checks for:
- interruption safety finalization
- rerun idempotency invariants
- classification coverage

## 4) Suggested Implementation Order

1. Add shared failure contract types + reason merge utility.
2. Thread failure classification through wallet runner/orchestrator finalization.
3. Harden lock recovery + retry/backoff loop guardrails.
4. Persist classification + counters in storage layer.
5. Add read-side ops repository queries + API handlers.
6. Extend dataset export + manifest entries for operational health artifacts.
7. Add/extend unit + integration + idempotency tests.
8. Add Phase 6 checks script and integrate into CI guardrail entrypoint.

## 5) Test Plan (Mandatory)

### 5.1 Unit
1. Failure classifier mapping for all known terminal paths.
2. `failure_reasons` merge determinism (order, dedupe, stable output).
3. Retry/backoff cap behavior and bounded-loop enforcement.

### 5.2 Integration
1. Forced wallet timeout finalizes with persisted class/reasons/incomplete markers.
2. Forced run timeout finalizes all reachable statuses in bounded tail context.
3. Retry exhaustion persists counters + truncation/incomplete linkage.
4. Lock conflict and lock-expiry takeover are safe and auditable.

### 5.3 Recovery/Idempotency
1. Interrupted run + rerun: no duplicates for canonical transfer/candidate identities.
2. Partial persistence + rerun converges to stable counters/statuses.
3. Operational export artifacts are byte-reproducible on unchanged DB state.

### 5.4 Exit Acceptance
1. No uncategorized terminal failure modes.
2. Recovery after interruption is bounded, safe, and auditable.
3. Reruns after failures remain idempotent.
4. All existing Phase 0-1 fixture gates remain passing.

## 6) Evidence Pack Requirements

For Phase 6 sign-off, archive:
1. Config/profile matrix used for canary, standard, stress runs.
2. Test reports for unit/integration/idempotency suites.
3. Forced-failure scenario logs:
- forced timeout
- forced retry exhaustion
- interruption + rerun
- stale-lock takeover
4. Ops API snapshots for runs/wallet-sync/failures windows.
5. Dataset export directories including operational health artifacts and manifest hashes.
6. Deterministic reproducibility comparison report across unchanged DB state.

## 7) Rollout Stages

1. Stage A (Canary):
- small wallet set
- forced interruption scenarios included
- zero invariant regressions required

2. Stage B (Standard):
- normal bounds
- repeated runs with induced partial failures
- stable finalization and classifications required

3. Stage C (Stress):
- upper-bound caps
- bounded degradation only
- no silent drops, no uncategorized terminal failures

4. Phase 6 sign-off:
- closeout document complete with evidence links and explicit PASS/BLOCKED verdict

## 8) Failure Handling Playbook (Operational)

If instability appears:
1. Halt scale-up.
2. Capture run/wallet persisted status rows and failure-class distributions.
3. Preserve export + manifest + API snapshots before any remediation rerun.
4. Classify root cause under canonical taxonomy.
5. Apply bounded mitigation and rerun canary profile.
6. Block release until acceptance gates re-pass.
