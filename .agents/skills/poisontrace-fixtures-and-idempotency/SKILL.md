---
name: poisontrace-fixtures-and-idempotency
description: Apply when changing transfer fingerprints, uniqueness constraints, upserts, rerun behavior, fixture suites, or CI checks for unknown-gate blocking and incomplete-window enforcement.
---

# PoisonTrace Fixtures and Idempotency

Use this skill for deterministic persistence, deduplication safety, and fixture-backed verification.

## Use when

- Editing DB uniqueness constraints, conflict handling, or monotonic timestamp updates.
- Changing transfer fingerprint composition or candidate identity semantics.
- Updating fixture corpus, fixture assertions, or CI checks for poisoning-gate safety.
- Touching rerun behavior that could duplicate transfers, links, counterparties, or candidates.

## Do not use when

- Changes are primarily runtime orchestration and timeout/retry handling (use `poisontrace-runtime-guards`).
- Changes are primarily poisoning-gate decision logic (use `poisontrace-poisoning-gates`).

## Required checks

1. Canonical transfer dedup remains `UNIQUE(signature, transfer_fingerprint)`.
2. Candidate dedup remains `UNIQUE(wallet_sync_run_id, signature, transfer_index)`.
3. Upserts are deterministic and rerun-safe; counts derive from persisted outcomes.
4. Monotonic fields preserve `first_seen_at = LEAST(...)` and `last_seen_at = GREATEST(...)` semantics where applicable.
5. Fixture suite asserts no candidate is emitted when any required gate is unknown.
6. Fixture suite asserts `wallet_sync_run.incomplete_window = true` when unknown required gates occur.
7. Fixture coverage includes unresolved owner, missing threshold (unknown dust), truncated baseline, shared signature across wallets, and repeated injections (>=2 gate).
8. Routine CI remains fixture-based; live Helius checks stay manual/sparse.

## Validation expectations

- Run migrations/DB tests when uniqueness or schema constraints change.
- Run relevant tests in `internal/pipeline`, `internal/storage`, and fixture-driven suites.

## Delivery reminder

- If repository files changed in the turn, final response must include:
- `Branch: <branch-name>`
- `Commit: <commit-subject-line>`
- If no repository files changed, final response must include:
- `No branch/commit required (no file changes).`
