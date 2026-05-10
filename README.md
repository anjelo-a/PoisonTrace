# PoisonTrace

Scanner-first Solana wallet poisoning injection detection pipeline.

## For internship reviewers
- Start here: [`REVIEWER.md`](REVIEWER.md)
- What this shows:
  - Backend engineering: bounded ingestion pipeline, strict normalization, fail-closed gates, deterministic persistence.
  - Frontend engineering: React/TypeScript dashboard for run visibility, candidate review, and operational workflows.
  - Engineering quality: idempotent reruns, fixture/corpus validation, and explicit handling of partial/unknown states.

## Phase 0–6 implementation status
- Scanner CLI with bounded wallet execution, timeout handling, and wallet-level failure isolation.
- Helius Enhanced Transaction ingestion for Solana baseline + scan windows.
- Owner-level normalization for native SOL and SPL fungible transfers, with unresolved/unsupported gating.
- Persisted poisoning-candidate materialization with strict fail-closed gate enforcement.
- Deterministic/idempotent persistence with fixture replay tests and CI policy checks.
- Phase 3 validation/tuning closeout with strict corpus checklist evidence in `docs/phase3_closeout.md`.
- Phase 4 execution closeout with reproducible bounded-run evidence in `docs/phase4_closeout.md`.
- Phase 6 operational hardening closeout in `docs/phase6_closeout.md`.

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

## Phase 4 execution kit
- Playbook: `docs/phase4_execution.md`
- Comprehensive plan: `docs/phase4_execution_plan.md`
- Profile matrix template: `docs/phase4_profile_matrix.template.md`
- Closeout template: `docs/phase4_closeout.template.md`
- Preflight command: `make phase4-preflight`
- Integrity check command: `make phase4-integrity RUN_ID=<ingestion_run_id>`
- Reproducibility check command: `make phase4-repro RUN_ID=<ingestion_run_id>`
