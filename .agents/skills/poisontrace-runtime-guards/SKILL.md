---
name: poisontrace-runtime-guards
description: Apply when changing scanner orchestration, retries, backoff, timeouts, bounded caps, concurrency, lock behavior, partial/failed status handling, or any path that could silently drop data.
---

# PoisonTrace Runtime Guards

Use this skill for runtime control-flow safety in Phase 0-1.

## Use when

- Editing wallet/run orchestration loops, retries, pagination, or concurrency.
- Changing timeout, cancellation, truncation, or lock-acquisition behavior.
- Updating status assignment (`partial`, `failed`, `skipped_budget`) or run notes/counters.

## Do not use when

- The task is primarily poisoning candidate gate logic (use `poisontrace-poisoning-gates`).
- The task is mainly fixture corpus or idempotency-key validation (use `poisontrace-fixtures-and-idempotency`).

## Required checks

1. All operational limits come from config, not hardcoded literals.
2. Execution remains bounded by run and wallet deadlines, page/tx caps, and concurrency caps.
3. Retry policy is limited to transient failures (429/5xx/network timeout) with bounded backoff and jitter.
4. Retry exhaustion and truncation paths persist explicit reason metadata and counters.
5. No normalization or ingestion failure path silently drops records.
6. Baseline completeness is never marked true when truncated by timeout, cap, retry exhaustion, or cancellation.
7. Wallet-level failures do not cancel sibling wallets unless parent context is canceled.

## Validation expectations

- Run impacted unit tests in `internal/pipeline`, `internal/helius`, and `internal/runs`.
- Verify failure states remain auditable via persisted status, error code/message, and notes.

## Delivery reminder

- If repository files changed in the turn, final response must include:
- `Branch: <branch-name>`
- `Commit: <commit-subject-line>`
- If no repository files changed, final response must include:
- `No branch/commit required (no file changes).`
