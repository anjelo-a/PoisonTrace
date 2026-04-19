# PoisonTrace Concepts And Principles (Phase 0-1)

This document explains the core ideas behind the current codebase and where each concept is implemented.

## 1) Mission And Detection Scope

PoisonTrace Phase 0-1 is a scanner-first Solana pipeline that detects **probable poisoning injections**.

It is intentionally scoped to:
- Solana only
- Helius Enhanced Transactions
- batch ingestion windows
- deterministic, idempotent reruns
- fail-closed candidate emission

It intentionally does **not** do victim attribution, campaign clustering, or generic cross-chain intelligence in this phase.

## 2) Core Principles

1. Fail-safe over best effort
- Unknown required gates block candidate emission.
- Incomplete windows are marked explicitly and persisted.

2. Idempotency over convenience
- Transfer dedup is canonical at event level (`signature + transfer_fingerprint`).
- Candidate dedup is per wallet sync (`wallet_sync_run_id + signature + transfer_index`).

3. No silent drops
- Unresolved normalization outcomes are stored with reason codes.
- Partial/truncated conditions persist reason metadata.

4. Bounded execution
- Run, wallet, pages, transactions, retries, and concurrency are all capped.

5. Owner-level Solana normalization
- SPL poisoning logic works on owner wallets (`fromUserAccount`/`toUserAccount`), not token-account shortcuts.

## 3) High-Level Pipeline Flow

Per wallet:
1. Build baseline + scan windows
2. Fetch enhanced transactions in bounded pages/retries
3. Normalize to deterministic transfer events
4. Map focal-wallet relation (`sender`/`receiver`)
5. Update counterparties from observations
6. Materialize poisoning candidates through strict gate evaluation
7. Persist progress/status/counters with explicit failure reasons

Main files:
- [`internal/pipeline/orchestrator.go`](../internal/pipeline/orchestrator.go)
- [`internal/pipeline/wallet_runner.go`](../internal/pipeline/wallet_runner.go)
- [`internal/pipeline/core.go`](../internal/pipeline/core.go)
- [`internal/pipeline/fetch.go`](../internal/pipeline/fetch.go)

## 4) Data Model And Persistence Contracts

Storage is defined as interfaces so pipeline code depends on contracts, not concrete DB details.

Primary contract file:
- [`internal/storage/repository.go`](../internal/storage/repository.go)

Postgres implementation:
- [`internal/storage/postgres_repository.go`](../internal/storage/postgres_repository.go)

Schema/migrations:
- [`migrations/0001_phase0_core.sql`](../migrations/0001_phase0_core.sql)
- [`migrations/0002_phase1_detection.sql`](../migrations/0002_phase1_detection.sql)
- [`migrations/0003_phase1_runtime_hardening.sql`](../migrations/0003_phase1_runtime_hardening.sql)

## 5) Solana Normalization Rules In Code

Native SOL:
- Normalize directly from wallet endpoints.

SPL fungible:
- Require owner-level endpoints from enhanced fields.
- Keep token-account addresses as trace fields only.

If owner resolution is missing or ambiguous:
- mark unresolved
- mark non-poisoning-eligible
- persist with reason code

File:
- [`internal/transactions/normalize.go`](../internal/transactions/normalize.go)

## 6) Candidate Gating (Fail-Closed)

Candidate emission requires all mandatory gates to pass.
If any required gate is unknown:
- candidate is blocked
- `incomplete_window = true`
- `unknown_gate_reason` is recorded

Files:
- [`internal/pipeline/candidate_materialize.go`](../internal/pipeline/candidate_materialize.go)
- [`internal/pipeline/detection.go`](../internal/pipeline/detection.go)

## 7) Runtime Guardrails

Guardrails include:
- bounded concurrency and timeouts
- bounded retries with backoff/jitter
- wallet lock ownership token
- finalize tails that still run after cancellation using bounded `WithoutCancel` contexts

Files:
- [`internal/pipeline/orchestrator.go`](../internal/pipeline/orchestrator.go)
- [`internal/pipeline/runtime_timing.go`](../internal/pipeline/runtime_timing.go)
- [`internal/pipeline/fetch.go`](../internal/pipeline/fetch.go)
- [`internal/storage/postgres_repository.go`](../internal/storage/postgres_repository.go)

## 8) OOP-Style And DSA Concepts Used

OOP-style (Go idioms):
- interfaces for abstraction/polymorphism
- dependency injection via constructor/options
- composition over inheritance

DSA/algorithmic patterns:
- maps as hash tables for O(1) lookups and dedup sets
- slices for deterministic ordering
- sorting for stable reason output/rule ordering
- bounded retry/backoff with jitter
- semaphore + goroutines + channels for bounded parallelism

## 9) How To Read The Codebase Quickly

Recommended order:
1. [`docs/architecture.md`](./architecture.md)
2. [`internal/storage/repository.go`](../internal/storage/repository.go)
3. [`internal/pipeline/orchestrator.go`](../internal/pipeline/orchestrator.go)
4. [`internal/pipeline/wallet_runner.go`](../internal/pipeline/wallet_runner.go)
5. [`internal/pipeline/core.go`](../internal/pipeline/core.go)
6. [`internal/transactions/normalize.go`](../internal/transactions/normalize.go)
7. [`internal/pipeline/candidate_materialize.go`](../internal/pipeline/candidate_materialize.go)
8. [`internal/pipeline/detection.go`](../internal/pipeline/detection.go)

Then run tests for behavior truth:
- `go test ./...`
- `go test ./internal/pipeline ./internal/storage ./internal/helius`
