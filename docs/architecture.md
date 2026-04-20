# PoisonTrace Architecture (Phase 0–1)

## 1. System Overview

PoisonTrace is a scanner-first Solana pipeline that detects probable wallet poisoning injections.

Operational constraints:
- ingest chain data in bounded batches
- normalize transfers into wallet-owner endpoints
- build wallet-scoped historical counterparties
- materialize probable poisoning injection candidates only when required gates pass

This is not a user product and not a generic risk platform in Phase 0–1.

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

Phase 0–1 is bounded by configured caps and windows.
Benefits:
- predictable cost and runtime
- safer local-first iteration

Tradeoff:
- incomplete history can produce `UNKNOWN` gate states.
- `UNKNOWN` required gates block candidate emission.
