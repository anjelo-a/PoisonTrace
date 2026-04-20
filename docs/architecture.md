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

## 11. Roadmap Blueprint (Post Phase 0–1)

Phase framing:
- Phase 0–1 is complete foundation.
- Remaining roadmap phases are 2 through 7.

### Phase 2: Detection Engine

Goal:
- Finalize deterministic candidate materialization as the primary scanner output for bounded batch runs.

Inputs:
- Wallet batch, baseline/scan windows, and bounded runtime caps.
- Helius Enhanced Transactions payloads.
- Asset dust thresholds and lookalike recency configuration.
- Persisted global normalized transfers and wallet-scoped counterparties.

Outputs:
- Materialized `poisoning_candidates` rows for qualifying events.
- Wallet/run counters for gated, blocked, unresolved, and emitted outcomes.
- Explicit unknown/truncation reason metadata.

Components:
- Window planner and wallet runner orchestration.
- Fetch + normalize pipeline with owner-level SPL resolution.
- Candidate gate evaluator and materializer.
- Repository layer for idempotent upserts and run summaries.

Processing rules:
- Emit only when required gates pass, including min-2 qualifying inbound injections per `(focal_wallet, suspicious_counterparty)` in scan window.
- Treat unresolved owner, unsupported asset, and unknown dust as non-poisoning-ready.
- Compute relation context via owner endpoints and evaluate candidates only on `receiver` direction.
- Preserve canonical event identity with `UNIQUE(signature, transfer_fingerprint)`.

Failure rules:
- If any required gate is `UNKNOWN`, do not emit candidate, set `wallet_sync_run.incomplete_window = true`, and persist `unknown_gate_reason`.
- On timeout/cap/retry exhaustion, persist partial progress and truncation reason.
- Contain failure at wallet level; do not drop normalized rows silently.

Exit criteria:
- Candidate emission is deterministic and idempotent across reruns.
- Unknown-gate blocking and `incomplete_window` persistence are enforced in fixtures and CI.
- Run outputs are auditable from transfers to candidate rows with persisted reasons.

### Phase 3: Validation and Tuning

Goal:
- Calibrate detection quality and gate behavior without changing scanner-first bounded architecture.

Inputs:
- Phase 2 emitted/blocked candidate corpus.
- Fixture corpora for known poisoning patterns and edge cases.
- Gate-level counters, unknown reasons, and unresolved-owner distributions.

Outputs:
- Versioned detection parameter profiles (dust, recency, similarity, window bounds).
- Validation reports for precision/recall tradeoffs and blocked-candidate causes.
- Updated fixture suites and CI assertions.

Components:
- Offline validation harness over persisted run artifacts.
- Fixture management and replay tooling.
- Parameter configuration layer used by scanner runtime.

Processing rules:
- Tune only through explicit configuration and fixture-backed acceptance thresholds.
- Preserve hard invariants: fail-closed unknown gates, min-2 injections, owner-level SPL normalization, bounded execution.
- Compare new profiles against baseline using deterministic replays.

Failure rules:
- Reject parameter updates that regress invariant checks or idempotency behavior.
- Reject tuning outputs that require unbounded scans or live-credit-heavy routine CI.
- Persist validation failures with gate-level attribution.

Exit criteria:
- Candidate quality metrics improve or remain stable under fixed fixture set.
- CI passes for unknown-gate blocking, incomplete-window marking, and uniqueness constraints.
- Tuned configuration is documented and reproducible.

### Phase 4: Batch Scanner and Dataset Generation

Goal:
- Scale bounded batch execution and generate consistent datasets for downstream inspection and validation.

Inputs:
- Wallet batch definitions and schedule inputs.
- Tuned Phase 3 configuration profiles.
- Existing normalized transaction and counterparty state.

Outputs:
- Reproducible batch run snapshots with wallet/run-level status and counters.
- Exportable candidate and gate-result datasets keyed to run identifiers.
- Dataset manifests with configuration/version metadata.

Components:
- Batch scheduler/runner with concurrency caps.
- Dataset export job for candidate and gate outcomes.
- Storage indexing and query paths for large-run retrieval.

Processing rules:
- Enforce configured caps for wallets/pages/transactions/concurrency/timeouts/retries.
- Prefer incremental processing over recomputing unchanged history.
- Generate datasets from persisted outcomes, not transient in-memory assumptions.

Failure rules:
- Mark partial/incomplete runs explicitly on cap/time/retry boundaries.
- Skip candidate export for wallets with unresolved required-gate outcomes while preserving blocked evidence.
- Fail export jobs loudly on schema/config mismatch.

Exit criteria:
- Batch runs complete predictably within configured bounds.
- Dataset exports are reproducible for the same input set/configuration.
- No silent record loss between persisted state and exported datasets.

### Phase 5: Inspection and Reporting

Goal:
- Provide analyst-facing inspection and reporting outputs from materialized scanner results.

Inputs:
- Phase 4 datasets and run artifacts.
- Candidate rows, gate evidence, and unknown/truncation reasons.
- Counterparty history and matched-legit context fields.

Outputs:
- Structured reports for emitted, blocked, and unknown-gate outcomes.
- Wallet-level investigation bundles with traceable event provenance.
- Aggregated operational summaries by run, wallet, and reason code.

Components:
- Report query layer on top of persisted tables.
- Inspection views/templates for candidate lineage.
- Export pipeline for machine-readable and human-readable reporting.

Processing rules:
- Report only persisted facts; do not infer victim attribution or attacker identity.
- Maintain one-to-one traceability from report rows back to signatures and transfer fingerprints.
- Keep candidate definition consistent with Phase 2 emission contract.

Failure rules:
- Suppress derived assertions when underlying required signals are unknown.
- Mark stale or partial source runs as non-authoritative in reports.
- Reject report generation when input run metadata is incomplete.

Exit criteria:
- Reports are reproducible and auditable from storage records.
- Analysts can distinguish emitted candidates from blocked/unknown outcomes unambiguously.
- Reporting layer introduces no new detection logic divergence.

### Phase 6: Operational Hardening

Goal:
- Harden reliability, observability, and recoverability for sustained bounded scanner operations.

Inputs:
- Runtime telemetry, failure distributions, and throughput baselines.
- Historical retry/timeout/cap-hit metrics and lock-contention data.
- Incident and rollback history from prior phases.

Outputs:
- Operational runbooks and SLO-aligned alerts.
- Hardened retry/backoff/timeout/concurrency profiles.
- Recovery procedures for reruns, partial runs, and interrupted windows.

Components:
- Metrics and logging instrumentation across pipeline stages.
- Alerting and health-check layer for run/wallet failure states.
- Operational controls for safe restart and controlled backfill.

Processing rules:
- Keep execution bounded and deterministic under recovery scenarios.
- Prioritize fail-contained wallet isolation over maximizing per-run throughput.
- Use idempotent reruns as the default remediation path.

Failure rules:
- Trigger explicit degraded status on repeated cap/time/retry failures.
- Block unsafe operational modes that bypass invariant checks.
- Preserve forensic audit trails for failed and restarted runs.

Exit criteria:
- Operational failures are detected, classified, and recoverable without data corruption.
- Rerun and recovery workflows preserve idempotency and candidate integrity.
- Throughput and stability are predictable under configured limits.

### Phase 7: Optional Extensions Only

Goal:
- Add explicitly approved extensions without weakening Phase 0–6 contracts.

Inputs:
- Stable outputs and operational baselines from Phase 6.
- Explicitly approved extension scope and acceptance criteria.
- Security/performance impact assessments for each extension.

Outputs:
- Extension-specific artifacts isolated behind clear interfaces.
- Updated docs and fixtures proving no regression to core detection invariants.
- Go/no-go decision records per extension.

Components:
- Optional modules (for example, integration adapters or advanced analytics overlays) decoupled from core scanner.
- Feature flags or configuration gates controlling extension activation.
- Regression test packs focused on invariant preservation.

Processing rules:
- Keep core scanner pipeline and candidate contract authoritative.
- Do not backdoor unbounded, streaming-first, or generic cross-chain scope into core phases.
- Require fixture-backed validation before enabling any extension by default.

Failure rules:
- Disable extension paths that degrade bounded performance or violate fail-closed behavior.
- Reject extensions that require weakening owner-resolution or unknown-gate policies.
- Roll back extension state without mutating canonical detection history.

Exit criteria:
- Extensions remain optional and isolated.
- Core Phase 0–6 behavior remains unchanged and validated.
- Extension enablement is explicit, reversible, and documented.
