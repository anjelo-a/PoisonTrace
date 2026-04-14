# SKILLS.md

## Purpose

Engineering standards for PoisonTrace Phase 0–1 implementation in Go.

## Go Coding Standards

1. Use Go 1.22+ style with explicit context propagation.
2. Keep packages narrowly scoped:
- `internal/config`
- `internal/helius`
- `internal/transactions`
- `internal/counterparties`
- `internal/pipeline`
- `internal/runs`
- `internal/storage`
3. Return typed errors with wrapped context (`fmt.Errorf("...: %w", err)`).
4. Avoid package-level mutable globals except immutable config constants.
5. Keep domain types explicit (do not pass raw maps across layers).
6. Never use floating-point math for token amounts or dust checks.
7. Use UTC only for window and recency comparisons.

## Error Handling Rules (No Silent Failures)

1. Every normalization skip must produce:
- `normalization_status`
- `reason_code`
- run counter increment
2. Every retry exhaustion must be surfaced in wallet/run notes and counters.
3. Unknown decision states must be explicit enums, not implicit null logic.
4. Never `continue` on parse/normalize failure without persisting reason metadata.

## Idempotent DB Write Rules

1. Use deterministic unique keys:
- transactions: `UNIQUE(signature, transfer_fingerprint)`
- wallet links: `UNIQUE(wallet_id, transaction_id, relation_type)`
- counterparties: `UNIQUE(focal_wallet_id, counterparty_address)`
- candidates: `UNIQUE(wallet_sync_run_id, signature, transfer_index)`
2. Use `INSERT ... ON CONFLICT ... DO UPDATE` for counters/timestamps.
3. Keep updates monotonic where applicable:
- `first_seen_at = LEAST(existing, incoming)`
- `last_seen_at = GREATEST(existing, incoming)`
4. Reruns must not inflate counts due to duplicate event replay.
5. Transfer dedup must use canonical event fingerprint:
- compute `transfer_fingerprint = hash(signature, instruction_ref, source_owner, destination_owner, token_mint, amount_raw, asset_type)`.
- persist `transfer_index` for ordering/reporting.
- canonical uniqueness is `UNIQUE(signature, transfer_fingerprint)`.
- `transfer_index` is deterministic output ordering, not primary dedup identity.

## Strict Normalization Contract

1. Native SOL:
- owner endpoints direct
- `asset_key = SOL`

2. SPL fungible:
- owner endpoints from `fromUserAccount/toUserAccount`
- token accounts stored only as trace fields

3. Unresolved owner:
- persist row
- `normalization_status = unresolved_owner`
- `poisoning_eligible = false`

4. Unsupported/non-fungible:
- `normalization_status = unsupported_asset`
- excluded from poisoning candidate logic

5. Dust classification:
- threshold lookup by `asset_key`
- missing threshold => `is_dust = unknown`
- unknown dust cannot satisfy candidate gate

6. Self-transfer handling:
- if normalized source owner equals destination owner, mark as non-counterparty event and exclude from poisoning detection.

## Concurrency Pattern (WaitGroup + Semaphore)

1. Use worker fanout bounded by `MAX_CONCURRENT_WALLETS`.
2. Pattern:
- `sem := make(chan struct{}, maxConcurrent)`
- `var wg sync.WaitGroup`
- acquire semaphore before wallet sync
- `defer` release semaphore and `wg.Done()`
3. Every wallet sync runs with context timeout (`WALLET_SYNC_TIMEOUT_SECONDS`).
4. Top-level run uses parent context timeout (`RUN_TIMEOUT_SECONDS`).
5. Wallet failures do not cancel sibling wallets unless run timeout/cancel is reached.
6. Wallet sync must hold a DB-backed single-writer lock (advisory lock or equivalent) for the focal wallet.
7. Lock acquisition failure must set wallet status to `skipped_budget` or explicit lock-skip code, never silent retry loops.

## Retry and Timeout Standards

1. Retry only transient classes (429/5xx/network timeout).
2. Use bounded exponential backoff with jitter.
3. Respect both wallet and run context deadlines.
4. On retry exhaustion:
- wallet status => `partial` if progress persisted, else `failed`
- record `error_code`, `error_message`, counters

## Config-Driven Limits (Mandatory)

No hardcoded operational limits in code paths.
Must come from config:
- wallet/page/tx caps
- concurrency
- delays
- retries
- dust thresholds source
- lookalike params
- recency days
- min injection count

## Fixture-Based Testing Rules

1. Routine smoke tests must use fixtures.
2. Live Helius checks are manual/sparse only.
3. Fixture coverage must include:
- unresolved owner
- missing threshold => unknown dust
- truncated baseline
- cross-wallet shared signature
- repeated injections (>=2 gate)
4. Known poisoning corpus must be replayed from deterministic fixture set.

## Code Review Checklist (Required)

1. Does any code path emit candidates when a required gate can be unknown?
2. Are unresolved/unsupported transfers persisted with reason codes?
3. Are token accounts ever used as poisoning counterparties? (must be no)
4. Are DB writes idempotent under rerun and partial retry?
5. Are bounded caps/timeouts enforced in all loops?
6. Are partial vs failed statuses assigned by policy, not convenience?
7. Are baseline completeness and newness semantics correct?
8. Is recency directionality correct (`legit < suspicious`)?
9. Is min-2 injection gate applied on same focal+counterparty within scan window?
10. Are fixtures updated for every behavior change?
