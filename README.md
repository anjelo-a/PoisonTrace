# PoisonTrace

Scanner-first Solana wallet poisoning injection detection pipeline.

## Phase 0–3 implementation status
- Scanner CLI with bounded wallet execution, timeout handling, and wallet-level failure isolation.
- Helius Enhanced Transaction ingestion for Solana baseline + scan windows.
- Owner-level normalization for native SOL and SPL fungible transfers, with unresolved/unsupported gating.
- Persisted poisoning-candidate materialization with strict fail-closed gate enforcement.
- Deterministic/idempotent persistence with fixture replay tests and CI policy checks.
- Phase 3 validation/tuning closeout with strict corpus checklist evidence in `docs/phase3_closeout.md`.

## Quick start
1. Copy `.env.example` to `.env` and set real values.
2. Run migrations: `source .env && make migrate`.
3. Build: `make build`
4. Test: `make test`
5. Validate corpus: `go run ./cmd/scanner validate-corpus --fixtures-root data/fixtures --report-out /tmp/phase3_report.json`

## TypeScript tooling (project utilities)
- Install Node dependencies: `make ts-install`
- Type-check TS utilities: `make ts-check`
- Run fixture utility example: `make ts-fixtures`

TypeScript files live under `scripts/ts/` and are configured with strict type checking.

## Web dashboard
1. Start API: `go run ./cmd/scanner serve-api --addr :8080`
2. Start web app: `npm run web:dev`
3. Run web tests: `npm run web:test`

Phase coverage in current implementation:
1. Milestone 1: app shell + API contracts + overview/candidates wired.
2. Milestone 2: remaining pages wired with URL-driven pagination/filters where applicable.
3. Milestone 3: test hardening via API contract tests + frontend route/UI/format tests.

## Important
Implementation enforces fail-safe and idempotency constraints from `AGENTS.md` and project skills.
