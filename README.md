# PoisonTrace

Scanner-first Solana wallet poisoning injection detection pipeline.

## Phase 0–1 scaffold included
- Go module and scanner CLI
- internal package layout for config, normalization, runs, counterparties, and pipeline
- PostgreSQL migration skeletons
- fixture and seed structure
- CI workflow with fixture policy checks

## Quick start
1. Copy `.env.example` to `.env` and set real values.
2. Run migrations: `source .env && make migrate`.
3. Build: `make build`
4. Test: `make test`

## Important
This scaffold encodes fail-safe and idempotency constraints from `AGENTS.md` and `SKILLS.md`.
