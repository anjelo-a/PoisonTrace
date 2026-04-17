# AGENTS.md

## Mission

PoisonTrace Phase 0–1 is a scanner-first Solana poisoning-injection detection pipeline.

Primary outcome:
- Detect probable poisoning injections.
- Do not claim confirmed victim attribution.
- Preserve auditable signals and failure states.

## Scope Boundaries (Strict)

In scope:
- Solana only.
- Helius Enhanced Transactions as source.
- Batch ingestion.
- Historical baseline + bounded scan window.
- Probable poisoning candidate materialization.
- Idempotent reruns.
- Wallet-level failure isolation.

Out of scope:
- End-user product UX.
- Generic cross-chain risk intelligence.
- ML scoring/ranking.
- Campaign clustering.
- Real-time streaming detection.

Any PR that expands scope beyond the above is rejected unless explicitly approved.

## Non-Negotiable Invariants

1. Fail-safe over best-effort ambiguity.
2. Idempotency over convenience.
3. No silent drops.
4. No token-account-as-wallet shortcuts for poisoning logic.
5. Bounded execution (time, concurrency, pages, tx count).
6. Unknown gating state must block candidate emission.
7. Do not ship unreliable or performance-hostile code paths.
8. Do not introduce design or implementation choices that create future operational blockage.

## Required Solana Normalization Rules

1. Native SOL:
- Use wallet/user endpoints directly.

2. SPL fungible:
- Use `fromUserAccount` and `toUserAccount` as owner endpoints.
- Persist token-account addresses only as trace fields.

3. If SPL owner endpoints cannot be resolved:
- Persist transfer with `normalization_status = unresolved_owner`.
- Set `poisoning_eligible = false`.
- Count and report it.
- Never use it for poisoning candidate logic.

4. Unsupported/non-fungible/unknown asset type:
- Persist as `normalization_status = unsupported_asset`.
- `poisoning_eligible = false`.

5. SPL owner validation hardening:
- If only one owner endpoint is present, status is `unresolved_owner`.
- If SPL `fromUserAccount == fromTokenAccount` or `toUserAccount == toTokenAccount`, treat as unresolved unless explicitly proven owner-level in source fields.
- If normalized source owner equals destination owner, classify as self-transfer and exclude from poisoning/counterparty updates.

## Direction Mapping Requirement

For each focal wallet:
- `relation_type = sender` when focal wallet is normalized source owner.
- `relation_type = receiver` when focal wallet is normalized destination owner.

Poisoning candidate logic only evaluates inbound (`receiver`) transfers.

## Candidate Emission Hard Gates (Mandatory)

A transfer may become a poisoning candidate only if all are true:

1. `normalization_status = resolved`.
2. `poisoning_eligible = true`.
3. `asset_type in {native_sol, spl_fungible}`.
4. `relation_type = receiver`.
5. `amount_raw = 0 OR is_dust = true`.
6. `is_new_counterparty = true`.
7. `baseline_complete = true`.
8. Exists matched legit counterparty with similarity rule:
- `(prefix >= 4 AND suffix >= 4) OR (prefix >= 6) OR (suffix >= 6)`.
9. `suspicious_counterparty != matched_legit_counterparty`.
10. Recency gate:
- `0 < suspicious.block_time - legit.last_outbound_at <= LOOKALIKE_RECENCY_DAYS`.
11. Minimum injections gate:
- At least 2 qualifying inbound events from same `(focal_wallet, suspicious_counterparty)` within the current scan window.
- Qualifier is `(amount_raw = 0 OR is_dust = true)`.

If ANY required gate is `UNKNOWN`:
- Candidate MUST NOT be emitted.
- `wallet_sync_run.incomplete_window = true`.
- `unknown_gate_reason` must be persisted.

## New Counterparty Definition

`is_new_counterparty = true` only when:
- No pre-window interaction exists in either direction, and
- `baseline_complete = true`.

If baseline is truncated/incomplete:
- `is_new_counterparty = UNKNOWN`.

## Legitimate Baseline Definition

A legit historical counterparty requires:
- Outbound interaction from focal wallet in baseline window.
- Non-dust transfer (`is_dust = false`).
- Transfer is resolved and poisoning-eligible.

Inbound-only history does not establish legitimacy in Phase 1.

## Bounded Execution Rules

Every run must honor:
- `MAX_WALLETS_PER_RUN`
- `MAX_TX_PAGES_PER_WALLET`
- `MAX_TX_PER_WALLET`
- `MAX_CONCURRENT_WALLETS`
- `WALLET_SYNC_TIMEOUT_SECONDS`
- `RUN_TIMEOUT_SECONDS`
- `MAX_HELIUS_RETRIES`
- `HELIUS_REQUEST_DELAY_MS`

If cap is hit:
- Persist progress.
- Mark wallet sync as `partial`.
- Mark `incomplete_window = true`.
- Preserve truncation reason.

## Idempotency Rules

1. Transfer identity is event-level with canonical fingerprint:
- `UNIQUE(signature, transfer_fingerprint)` for normalized transfer rows.
- `transfer_index` is deterministic ordering metadata, not canonical uniqueness.

2. Candidate identity is event-scoped per wallet sync:
- `UNIQUE(wallet_sync_run_id, signature, transfer_index)` for candidate rows.

3. Upserts must be deterministic.
4. Reruns must not duplicate transfers, links, counterparties, or candidates.
5. Counters must be derived from persisted outcomes, not assumptions.

## Enforcement Hooks (Mandatory)

A change is non-compliant unless all pass:

1. DB constraints:
- Transactions must enforce canonical dedup via `UNIQUE(signature, transfer_fingerprint)`.
- Poisoning candidates must enforce `UNIQUE(wallet_sync_run_id, signature, transfer_index)`.

2. CI gates:
- Fixture suite must assert no candidate is emitted when any required gate is `UNKNOWN`.
- Fixture suite must assert `wallet_sync_run.incomplete_window = true` on unknown required gates.

3. Runtime guards:
- Never mark `baseline_complete = true` when baseline stopped due to timeout, tx/page cap, retry exhaustion, or cancellation.
- Every blocked candidate must persist an explicit `unknown_gate_reason`.

## Required Questions Before Implementation Starts

Implementation must stop and ask if any of these are unresolved:
1. Scan and baseline window bounds.
2. Baseline completeness behavior on truncation.
3. Dust threshold source for each asset.
4. Lookalike thresholds and recency limits.
5. Retry, timeout, and partial status semantics.
6. Candidate uniqueness and idempotency keys.
7. Fixture pass criteria against known poisoning corpus.

## Anti-Patterns (Project-Specific)

1. Using SPL token-account addresses as counterparties.
2. Emitting candidates when any gate is unknown.
3. Treating truncated baseline as complete.
4. Using global dust threshold across all assets.
5. Counting inbound-only counterparties as legit baseline.
6. Best-effort dropping unresolved transfers without reason codes.
7. Unbounded live Helius tests in routine CI.
8. Expanding to generic threat intelligence in Phase 0–1.
9. Introducing fragile logic without deterministic validation at boundaries.
10. Shipping changes that significantly degrade runtime performance without explicit approval.

## Final Response Contract (Mandatory)

For every response that includes repository file changes, the final response MUST include:
- `Branch: <branch-name>`
- `Commit: <commit-subject-line>`

If no repository files changed, the final response MUST include:
- `No branch/commit required (no file changes).`

## Pre-Final Checklist (Mandatory)

Before sending the final response:
1. Confirm all requested edits are completed (or explicitly call out blockers).
2. Confirm validation/test status is reported (or explicitly say not run).
3. Confirm the final response includes the Final Response Contract fields above.
