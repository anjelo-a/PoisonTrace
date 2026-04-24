# PoisonTrace Architecture (Phase 0–2)

## 1. System Overview

PoisonTrace is a scanner-first Solana pipeline that detects probable wallet poisoning injections.

Operational constraints:
- ingest chain data in bounded batches
- normalize transfers into wallet-owner endpoints
- build wallet-scoped historical counterparties
- materialize probable poisoning injection candidates only when required gates pass

This is not a user product and not a generic risk platform in Phase 0–2.

## 2. Detection Target Definition

A probable poisoning injection candidate is a new inbound counterparty sending a zero-value or dust transfer to a focal wallet, where the suspicious counterparty resembles a legitimate historical counterparty and has at least 2 qualifying injections in the scan window.

Hard policy:
- If any required gate is `UNKNOWN`, no candidate is emitted, `incomplete_window = true`, and the reason is persisted.

## 3. Data Flow

1. Ingestion run starts with bounded wallet batch.
2. For each wallet, run two-window sync:
- baseline window (historical context)
- scan window (candidate search)
3. Helius Enhanced Transactions are fetched page-by-page with bounded retries.
4. Normalizer extracts deterministic transfer events:
- canonical key: `(signature, transfer_fingerprint)`
- reporting order key: `transfer_index`
- endpoint mapping to owner wallets
5. Storage layer upserts:
- global transactions
- wallet-to-transaction relations
- wallet-scoped counterparties
6. Detector materializes `poisoning_candidates` for the wallet sync.
7. Run/wallet summaries persist counters, status, and failure reasons.

## 4. Why Global `transactions` + `wallet_transactions`

Rationale:
1. One transaction can affect multiple wallets.
2. Global transaction table avoids duplicating transaction payload per focal wallet.
3. `wallet_transactions` provides focal-wallet relation context (`sender`/`receiver`) without denormalizing chain data.
4. This split supports idempotency, storage efficiency, and cross-wallet auditability.
5. It also enables the required cross-wallet fixture case where same signature appears in multiple wallet syncs.

## 5. Solana Endpoint Normalization Model

Key Solana constraint:
- SPL transfers often move between token accounts, not owner wallets.

Poisoning logic must compare owner wallets, not token accounts.

Rules:
1. Native SOL:
- use wallet endpoints directly.

2. SPL fungible:
- require `fromUserAccount` and `toUserAccount` from Helius enhanced transfer fields.
- token-account addresses are persisted only as supporting trace fields.

3. If owner cannot be resolved:
- persist transfer with `normalization_status = unresolved_owner`
- set `poisoning_eligible = false`
- count reason in run metrics
- exclude from candidate evaluation

4. Partial-owner and self-transfer safeguards:
- one-sided owner presence is treated as unresolved
- owner==token account equality is treated as unresolved unless explicitly owner-level in source fields
- owner self-transfer events do not contribute to counterparties or poisoning detection

## 6. Core Persistence Model

Primary tables:
- `wallets`
- `transactions` (global normalized transfer events)
- `wallet_transactions` (wallet relation context)
- `counterparties` (wallet-scoped history and direction stats)
- `ingestion_runs`
- `wallet_sync_runs`
- `asset_thresholds`
- `poisoning_candidates`

Key uniqueness:
- transactions: `UNIQUE(signature, transfer_fingerprint)` (canonical dedup)
- transactions: `transfer_index` retained for deterministic ordering and joins
- wallet links: `UNIQUE(wallet_id, transaction_id, relation_type)`
- counterparties: `UNIQUE(focal_wallet_id, counterparty_address)`
- detections: `UNIQUE(wallet_sync_run_id, signature, transfer_index)`

## 7. Candidate Gating Semantics

Required conditions:
1. `normalization_status = resolved`
2. `poisoning_eligible = true`
3. inbound relation (`receiver`)
4. zero or dust transfer
5. `is_new_counterparty = true` with `baseline_complete = true`
6. lookalike similarity passes:
- `(prefix >= 4 AND suffix >= 4) OR (prefix >= 6) OR (suffix >= 6)`
7. recency gate:
- `0 < suspicious.block_time - legit.last_outbound_at <= LOOKALIKE_RECENCY_DAYS`
8. inequality guard:
- `suspicious_counterparty != matched_legit_counterparty`
9. minimum-injection gate:
- at least 2 qualifying inbound events from same focal+counterparty in current scan window

Unknown-gate fail-closed rule:
- if any required gate is `UNKNOWN`, candidate is not emitted,
- `wallet_sync_run.incomplete_window = true`,
- an `unknown_gate_reason` must be persisted.

## 8. Baseline and Newness Semantics

Legitimate baseline counterparty:
- outbound from focal wallet
- non-dust transfer
- resolved/eligible transfer
- baseline window only

New counterparty:
- no interaction in either direction before scan window start
- valid only when baseline completeness is true
- otherwise `unknown`

## 9. Failure-Containment Model

1. Wallet-level isolation: one wallet failure cannot crash entire run.
2. Single-writer wallet lock avoids concurrent wallet race conditions.
3. Bounded retries and explicit timeout controls at wallet and run levels.
4. Partial progress is persisted and labeled (`partial`, `incomplete_window`, truncation reason).
5. No silent normalization drops.

## 10. Bounded Ingestion and Its Implications

Phase 0–2 is bounded by configured caps and windows.
Benefits:
- predictable cost and runtime
- safer local-first iteration

Tradeoff:
- incomplete history can produce `UNKNOWN` gate states.
- `UNKNOWN` required gates block candidate emission.

## 11. Roadmap Blueprint (Post Phase 0–2)

## System Objective

PoisonTrace is a scanner-first Solana wallet poisoning injection detection system.

Probable poisoning injection candidate:
- new inbound counterparty
- zero-value or dust transfer
- lookalike of a known legitimate counterparty
- valid temporal relationship
- at least 2 qualifying injections within scan window

## Global System Rules

Core constraints:
- scanner-first (no user dependency)
- batch processing (foundation through dataset phases)
- bounded ingestion always
- idempotent storage
- wallet-level failure isolation
- no silent data loss
- fail closed on unknown required signals

Required data layers:
1. raw provider response (implicit)
2. normalized transfer events
3. wallet-to-transaction relation
4. counterparties (baseline + scan)
5. materialized poisoning candidates

## Phase Structure

Each phase uses:
- Goal
- Inputs
- Outputs
- Components
- Processing rules
- Failure rules
- Exit criteria

### Phase 0–1: Foundation (Complete)

Goal:
- Establish deterministic, bounded, and poisoning-ready data pipeline.

Inputs:
- Helius Enhanced Transactions.
- Seed wallet list.
- Configured windows, caps, and thresholds.

Outputs:
- Normalized transactions.
- `wallet_transactions`.
- Counterparties.
- `wallet_sync_runs`.
- `ingestion_runs`.

Components:
- Helius client.
- Normalization layer.
- Wallet-owner resolver.
- Relation mapper (`sender` / `receiver`).
- Counterparty builder.
- Run orchestrator.

Processing rules:
- Transfer identity uses canonical fingerprint uniqueness: `UNIQUE(signature, transfer_fingerprint)`.
- Process native SOL and fungible SPL only for poisoning logic.
- Owner resolution is required for poisoning-ready SPL transfers.
- Baseline window and scan window are separated and bounded.
- Legitimate counterparties require outbound non-dust baseline interactions.
- Directionality is explicit via sender/receiver mapping.

Failure rules:
- `unresolved_owner` is persisted but non-poisoning-ready.
- Unknown dust status blocks candidate emission.
- Incomplete baseline blocks "new counterparty" claims.
- Truncation is allowed only as explicit partial status.
- Wallet failures remain isolated.

Exit criteria:
- Idempotent reruns are verified.
- Normalization correctness is fixture-verified.
- Fixture replay is deterministic.
- Known poisoning patterns are reconstructable from persisted records.

### Phase 2: Detection Engine (Complete)

Goal:
- Materialize poisoning candidates using strict gating logic.

Inputs:
- Normalized transactions.
- `wallet_transactions`.
- Counterparties.
- `wallet_sync_runs` (including baseline completeness and windows).

Outputs:
- `poisoning_candidates` materialized table.

Components:
- Lookalike matcher.
- Recency evaluator.
- Injection counter.
- Candidate materializer.

Processing rules:
- Emit candidate only if all required gates pass:
1. `normalization_status = resolved`
2. `asset_type in {native_sol, spl_fungible}`
3. `relation_type = receiver`
4. `amount_raw = 0 OR is_dust = true`
5. `is_new_counterparty = true AND baseline_complete = true`
6. lookalike similarity rule passes
7. `0 < time_gap <= LOOKALIKE_RECENCY_DAYS`
8. `suspicious_counterparty != matched_legit_counterparty`
9. at least 2 qualifying inbound injections in scan window

Failure rules:
- Any required gate `UNKNOWN` blocks candidate emission.
- Incomplete baseline blocks candidate emission.
- Unknown dust status blocks candidate emission.
- Unresolved owner blocks poisoning eligibility.

Exit criteria:
- Candidate results are stable across reruns.
- No duplicate candidate emissions per uniqueness contract.
- Known in-scope poisoning patterns are detected.
- Zero emission occurs when required gates are incomplete.

Current status:
- Implemented in code paths under `internal/pipeline/*` and `internal/storage/*`.
- Covered by deterministic fixture replay and CI guardrail tests.

### Phase 3: Validation and Tuning

Goal:
- Validate detection quality and tune thresholds within the same bounded rule-based model.

Inputs:
- `poisoning_candidates`.
- Known poisoning-case datasets.
- Fixture test cases.

Outputs:
- Tuned threshold configurations.
- Validated detection rule settings.
- Classified misses and false positives.

Components:
- Validation runner.
- Threshold configuration manager.
- Fixture and known-case harness.

Processing rules:
- Evaluate recall on known cases.
- Evaluate false positives on representative wallet samples.
- Tune lookalike thresholds, recency windows, and dust thresholds through explicit config revisions only.

Failure rules:
- Unexplained false positives block phase exit.
- Unexplained misses block phase exit.
- Inconsistent rerun results block phase exit.

Exit criteria:
- Target recall is reached for known in-scope cases.
- False positives are explainable from persisted evidence.
- Threshold behavior is stable across reruns.
- Validation outcomes are reproducible.

### Phase 4: Batch Scanner and Dataset Generation

Goal:
- Execute bounded scans over wallet sets and generate reproducible poisoning datasets.

Inputs:
- Seed wallet sets.
- Configured limits and windows.
- Validated detection engine configuration.

Outputs:
- Multi-run ingestion artifacts.
- Poisoning candidate datasets.
- Run summaries.

Components:
- Wallet scheduler.
- Batch orchestrator.
- Run summary aggregator.

Processing rules:
- Use deterministic wallet ordering.
- Enforce bounded per-wallet processing.
- Scale batch size incrementally under configured caps.
- Track run-level metrics: wallets processed, transactions processed, candidates emitted, truncation rate.

Failure rules:
- Wallet-level failures are logged and isolated.
- Runs continue unless run-level timeout/cancellation is reached.
- Incomplete windows remain explicitly flagged.

Exit criteria:
- Batch runs remain stable under configured caps.
- Datasets are generated reproducibly.
- Operational cost envelopes are measured.
- Candidate distribution is analyzable from persisted outputs.

### Phase 5: Inspection and Reporting

Goal:
- Make detection outputs explainable and inspectable without changing detection logic.

Inputs:
- `poisoning_candidates`.
- Counterparties.
- Transactions and wallet relation context.

Outputs:
- Inspection queries and views.
- Exportable reports.
- Candidate explanation artifacts.

Components:
- SQL read models.
- Reporting queries.
- Optional minimal read API.

Processing rules:
- Every candidate must be explainable from persisted data.
- No hidden detection logic outside canonical persisted contracts.

Failure rules:
- Unexplainable candidates are invalid outputs.
- Inconsistent explanation paths are invalid outputs.

Exit criteria:
- Every candidate is traceable to stored transfer evidence.
- Explanation paths are consistent and reproducible.
- Datasets are exportable for review and presentation.

### Phase 6: Operational Hardening

Goal:
- Ensure robust behavior under repeated, partial, and interrupted execution.

Inputs:
- Full pipeline implementation.
- Historical batch run outcomes.

Outputs:
- Stable operational behavior.
- Improved observability and failure classification.

Components:
- Retry manager.
- Logging system.
- Metrics collection.
- Failure classification layer.

Processing rules:
- Retries are bounded with backoff.
- Worker failures are recovered without global run collapse.
- Errors are categorized and persisted with consistent run statuses.

Failure rules:
- Silent failures are forbidden.
- Unbounded retry loops are forbidden.
- Single-wallet failures must not crash global runs.

Exit criteria:
- Reruns after failure remain stable and idempotent.
- Failure modes are observable and classified.
- Recovery from interruption is safe and auditable.

### Phase 7: Optional Extensions

Goal:
- Add approved extensions without weakening the core bounded detection pipeline.

Inputs:
- Stable candidate datasets.
- Stable rule-based detection engine.

Outputs:
- Isolated enhancement modules and outputs.
- No-regression validation evidence for core invariants.

Components:
- Optional confidence scoring layer.
- Optional cross-wallet pattern analysis.
- Optional scheduled scan automation.
- Optional ML-based ranking layer (ranking only).

Processing rules:
- Extensions must not alter core Phase 2 candidate gates.
- Extensions must remain bounded and explicitly configurable.

Failure rules:
- Extensions are disabled if they degrade correctness, boundedness, or fail-closed behavior.
- Extensions are rejected if they require scope expansion into generic cross-chain intelligence.

Exit criteria:
- Enhancements remain isolated and reversible.
- Core pipeline behavior is unchanged.
- No regression in detection correctness.

## Final System Definition

PoisonTrace is:
- a bounded, deterministic, idempotent scanner
- ingesting Solana wallet history via Helius Enhanced Transactions
- resolving owner-level transfer behavior
- building wallet-scoped counterparty relationships
- materializing probable poisoning injection candidates

And it guarantees:
- strict gating
- fail-contained execution
- reproducible outputs
- auditable detection logic

## 12. Locked Decisions For The Next Step

The `AGENTS.md` "Required Questions Before Implementation Starts" are explicitly locked for the next step (Validation and Tuning) in:
- [`docs/phase_transition_decisions.md`](./phase_transition_decisions.md)

Lock date:
- 2026-04-24
