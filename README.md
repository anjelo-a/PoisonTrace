# PoisonTrace

Scanner-first Solana wallet poisoning injection detection pipeline.

## Phase 0–1 implementation status
- Scanner CLI with bounded wallet execution, timeout handling, and wallet-level failure isolation.
- Helius Enhanced Transaction ingestion for Solana baseline + scan windows.
- Owner-level normalization for native SOL and SPL fungible transfers, with unresolved/unsupported gating.
- Persisted poisoning-candidate materialization with strict fail-closed gate enforcement.
- Deterministic/idempotent persistence with fixture replay tests and CI policy checks.

## Quick start
1. Copy `.env.example` to `.env` and set real values.
2. Run migrations: `source .env && make migrate`.
3. Build: `make build`
4. Test: `make test`

## Important
Implementation enforces fail-safe and idempotency constraints from `AGENTS.md` and project skills.
