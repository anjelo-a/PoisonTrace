---
name: poisontrace-poisoning-gates
description: Apply when changing normalization-to-owner mapping, relation direction, baseline completeness/newness, lookalike matching, recency/min-injection rules, or poisoning candidate emission gates.
---

# PoisonTrace Poisoning Gates

Use this skill for poisoning-candidate eligibility and gate semantics.

## Use when

- Editing candidate emission logic, gate evaluation, or blocked-candidate reason handling.
- Changing normalization status handling that affects poisoning eligibility.
- Updating direction mapping (`sender`/`receiver`), counterparty newness, or baseline completeness behavior.
- Modifying lookalike thresholds, recency windows, or min-injection gating.

## Do not use when

- Changes are mostly runtime retry/timeout/cap orchestration (use `poisontrace-runtime-guards`).
- Changes are mostly fixture management or idempotent upsert strategy (use `poisontrace-fixtures-and-idempotency`).

## Required checks

1. Candidate emission requires all mandatory gates to be known and true.
2. Any required gate in `UNKNOWN` blocks emission, sets `wallet_sync_run.incomplete_window = true`, and persists `unknown_gate_reason`.
3. Poisoning evaluation is inbound only (`relation_type = receiver`).
4. SPL counterparties use owner endpoints (`fromUserAccount`/`toUserAccount`), never token-account addresses as wallet counterparties.
5. Unresolved/unsupported normalization stays persisted with explicit status and excluded eligibility.
6. `is_new_counterparty` is true only with no pre-window interaction in either direction and complete baseline; otherwise unknown when baseline is incomplete.
7. Legit historical counterparties require resolved, poisoning-eligible, non-dust outbound history from focal wallet.
8. Similarity and recency rules preserve directionality (`legit.last_outbound_at < suspicious.block_time`).
9. Min injections gate requires at least two qualifying inbound events for the same `(focal_wallet, suspicious_counterparty)` within the scan window.

## Validation expectations

- Run affected tests in `internal/pipeline` and `internal/transactions`.
- Add or update fixtures whenever gate behavior changes.
