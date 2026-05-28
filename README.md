# PoisonTrace

PoisonTrace is a full-stack Solana wallet poisoning detection project: it ingests bounded Helius Enhanced Transaction windows for focal wallets, normalizes transfers to wallet-owner endpoints, and materializes only probable poisoning injection candidates when strict fail-closed gates pass. For an internship reviewer, the signal is the engineering shape: a Go/Postgres batch pipeline with bounded retries, wallet-level failure isolation, idempotent persistence, deterministic fixture validation, and a React/TypeScript dashboard for inspecting runs, candidates, transactions, exports, and operational state.

## What It Demonstrates

- **Bounded backend pipeline.** Runtime caps, timeouts, concurrency limits, retry limits, and partial-status persistence are configured in `internal/config/config.go`, enforced through `internal/pipeline/orchestrator.go`, `internal/pipeline/fetch.go`, and `internal/pipeline/wallet_runner.go`, and summarized in `docs/phase4_closeout.md`.
- **Fail-closed detection logic.** Candidate emission is blocked when required gates are unknown; unknown reasons and incomplete windows are persisted. See `AGENTS.md`, `internal/pipeline/candidate_materialize.go`, `internal/pipeline/detection.go`, and `scripts/ci_guardrails.sh`.
- **Idempotent, auditable persistence.** Transfer identity uses `(signature, transfer_fingerprint)` and candidate identity uses `(wallet_sync_run_id, signature, transfer_index)`. See `migrations/0001_phase0_core.sql`, `migrations/0002_phase1_detection.sql`, `internal/storage/postgres_repository.go`, and `docs/architecture.md`.
- **Full-stack implementation.** The backend is Go 1.22 with Postgres (`go.mod`, `cmd/scanner/main.go`, `internal/api/server.go`); the frontend is Vite + React + TypeScript with shared contracts in `packages/contracts/src/index.ts` and dashboard routes in `apps/web/src/app/routes.ts`.

## Architecture Overview

Data flow: wallet input enters the scanner as a file, explicit API request, daemon-sourced wallet set, or fixture case; the Go pipeline fetches Helius Enhanced Transactions for a historical baseline window and bounded scan window; transfers are normalized into deterministic owner-level events; counterparty history is updated; candidate materialization evaluates poisoning gates; and all run, wallet, transfer, counterparty, candidate, export, and operational-health state is persisted for review.

- **Phase 0:** Core schema and ingestion foundation for wallets, transactions, wallet relations, and run records (`migrations/0001_phase0_core.sql`).
- **Phase 1:** Poisoning-aware normalization, thresholds, candidate tables, runtime hardening, locks, and run counters (`migrations/0002_phase1_detection.sql` through `migrations/0004_phase1_run_counters.sql`).
- **Phase 2:** Strict detection engine for inbound zero/dust lookalike injections (`internal/pipeline/candidate_materialize.go`, `internal/pipeline/detection.go`, `docs/architecture.md`).
- **Phase 3:** Corpus validation and threshold tuning without expanding detection scope (`docs/phase3_closeout.md`, `artifacts/corpus_validation_report.json`).
- **Phase 4:** Bounded batch execution, reproducible exports, integrity checks, and stress/canary evidence (`docs/phase4_closeout.md`, `scripts/phase4_*`).
- **Phase 5:** Inspection and reporting surfaces through read APIs, reports, exports, and the frontend dashboard (`internal/api/server.go`, `internal/exports/dataset.go`, `apps/web/src/app/pages/app/`).
- **Phase 6:** Operational hardening: failure taxonomy, ops APIs, async export jobs, deterministic operational-health export artifacts, and recovery/idempotency tests (`docs/phase6_closeout.md`, `migrations/0006_phase6_backend_hardening.sql`, `migrations/0007_phase6_ops_observability_counters.sql`).

## Honest Metrics

- Fixture corpus: `19` total cases, `19` passed, `0` failed, `7` expected in-scope, `7` detected in-scope, `12` expected out-of-scope, `0` out-of-scope with candidates, recall `1`, false-positive rate `0` (`artifacts/corpus_validation_report.json`; same strict summary in `artifacts/phase4/preflight/corpus_validation_strict.json`).
- Phase 3 validation: `cases=19 passed=19 failed=0 recall=1.000 false_positive_rate=0.000`; strict rerun SHA-256 `51916883035c83c0d5e93472093813a983d21336948889a46671ce2cc60703c7` (`docs/phase3_closeout.md`).
- Phase 4 sign-off runs `6`, `7`, and `8`: `3` wallets requested and `3` processed each run, `0` wallets failed, `15` transactions fetched each run, `0` poisoning candidates inserted each run, truncation wallet rate `0.0` each run (`docs/phase4_closeout.md`).
- Phase 4 idempotency evidence: transactions inserted were `50` in run `6`, then `0` in runs `7` and `8` under unchanged source filters (`docs/phase4_closeout.md`).
- Phase 4 normalization snapshot: `resolved=46`, `failed=4`, `dust_status true=4`, `dust_status unknown=46` (`docs/phase4_closeout.md`).
- Phase 4 integrity checks: runs `6`, `7`, and `8` each report `integrity_check=PASS`, `transfer_identity_duplicates=0`, `candidate_identity_duplicates=0`, and `candidates_total=0` (`artifacts/phase4/run_6/integrity_check.txt`, `artifacts/phase4/run_7/integrity_check.txt`, `artifacts/phase4/run_8/integrity_check.txt`).
- Phase 4 reproducibility checks: runs `6`, `7`, and `8` each report `repro_check=PASS` with matching hashes for `ingestion_runs.jsonl`, `wallet_sync_runs.jsonl`, `poisoning_candidates.jsonl`, and `manifest.json` (`artifacts/phase4/repro_run_6/repro_check.txt`, `artifacts/phase4/repro_run_7/repro_check.txt`, `artifacts/phase4/repro_run_8/repro_check.txt`).
- Phase 6 machine-readable checks: `go_test_all=pass`, `ci_guardrails=pass`, corpus validation `cases=19`, `passed=19`, `failed=0`, `recall=1.0`, `false_positive_rate=0.0` (`artifacts/phase6/phase6_checks_report.json`).
- Local API load caveat: autocannon against local `http://127.0.0.1:8091/healthz`, `3s`, `5` connections, reported `101344.00` average requests/sec, `304040` total requests, `0` errors, `0` timeouts (`artifacts/performance/20260528T050926Z/load/summary.md`). This is a local health endpoint check, not database-route throughput.
- Deployed backend load caveat: autocannon against `https://poisontrace-dev.livelymeadow-b0f616bc.southeastasia.azurecontainerapps.io`, `30s`, `10` connections, reported `189.70` average requests/sec, `5691` total requests, `0` errors, `0` timeouts (`artifacts/performance/20260528T051432Z/load/summary.md`). The summary does not include the route path.
- Frontend Lighthouse caveat: local Vite preview reported Performance `69`, Accessibility `91`, Best practices `96`, SEO `82`, LCP `12.1 s`, CLS `0.001` (`artifacts/performance/20260528T050816Z/frontend/summary.md`).
- Deployed frontend Lighthouse caveat: `https://poison-trace-web.vercel.app/` reported Performance `67`, Accessibility `91`, Best practices `100`, SEO `82`, LCP `9.3 s`, CLS `0.001` (`artifacts/performance/20260528T051326Z/frontend/summary.md`).

## Quick Start

Prerequisites:
- Go 1.22 (`go.mod`)
- Node/npm (`package.json`)
- Postgres and `psql` for migrations (`scripts/migrate.sh`)
- Helius API key for live ingestion (`.env.example`)

```sh
cp .env.example .env
npm install
make build
make test
npm run typecheck
```

For database-backed commands, load environment variables first:

```sh
set -a
source .env
set +a
make migrate
```

Validated Makefile targets:

```sh
make build
make test
make test-guardrails
make test-fixture-metadata
make validate-corpus
make run-fixture
make ts-install
make ts-check
make ts-fixtures
```

Live scanner/API commands verified from `cmd/scanner/main.go`:

```sh
go run ./cmd/scanner run --wallets data/seeds/wallets.example.txt --scan-start 2026-04-01T00:00:00Z --scan-end 2026-04-08T00:00:00Z
go run ./cmd/scanner replay-fixture --fixture baseline_truncated_newness_unknown
go run ./cmd/scanner validate-corpus --fixtures-root data/fixtures --report-out ./artifacts/corpus_validation_report.json
go run ./cmd/scanner serve-api --addr :8080
go run ./cmd/scanner daemon --once
```

Dataset export command shape:

```sh
go run ./cmd/scanner export-dataset --out-dir ./artifacts/export --run-id <run_id>
```

## Frontend Dashboard

The dashboard lives under `apps/web/` and uses Vite, React 18, React Router, React Query, MUI/Radix UI components, and shared contracts from `packages/contracts/`.

Routes are verified in `apps/web/src/app/routes.ts`:
- `/`
- `/methodology`
- `/app`
- `/app/candidates`
- `/app/transactions`
- `/app/runs`
- `/app/wallet-sync`
- `/app/counterparties`
- `/app/reports/wallets`
- `/app/exports`
- `/app/settings`

Run it locally:

```sh
go run ./cmd/scanner serve-api --addr :8080
npm run web:dev
```

Vite proxies `/api` and `/healthz` to `http://localhost:8080` in `apps/web/vite.config.ts`. Frontend validation commands:

```sh
npm run web:typecheck
npm run web:test
npm run web:build
```

## TypeScript Tooling

- Root TypeScript config: `tsconfig.json`
- Workspace package scripts: `package.json`
- Shared API contracts: `packages/contracts/src/index.ts`
- Fixture utility: `scripts/ts/check-fixtures.ts`

Commands:

```sh
npm run typecheck
npm run contracts:typecheck
npm run ts:fixtures
```

Performance harness commands:

```sh
npm run perf:load
npm run perf:web
```

Details and environment overrides are documented in `docs/performance_metrics.md`.

## Phase Documentation Index

- Architecture: `docs/architecture.md`
- Concepts and principles: `docs/concepts_and_principles.md`
- Limitations: `docs/limitations.md`
- Phase transition decisions: `docs/phase_transition_decisions.md`
- Phase 3 closeout: `docs/phase3_closeout.md`
- Phase 4 execution playbook: `docs/phase4_execution.md`
- Phase 4 closeout: `docs/phase4_closeout.md`
- Phase 6 execution plan: `docs/phase6_execution_plan.md`
- Phase 6 closeout: `docs/phase6_closeout.md`
- Free-plan daemon notes: `docs/free_plan_daemon.md`
- Azure deployment notes: `deploy/azure/README.md`
- Web design execution notes: `docs/web_design_execution.md`
- Performance metrics harness: `docs/performance_metrics.md`

<details>
<summary>Phase 4 Execution Kit</summary>

The Phase 4 kit is useful for reproducibility and release-style evidence, but most reviewers can start with `REVIEWER.md` and the closeout docs first.

```sh
make phase4-preflight
make phase4-integrity RUN_ID=<ingestion_run_id>
make phase4-repro RUN_ID=<ingestion_run_id>
```

Supporting files:
- `scripts/phase4_preflight.sh`
- `scripts/phase4_integrity_check.sh`
- `scripts/phase4_repro_check.sh`
- `docs/phase4_execution_plan.md`
- `docs/phase4_profile_matrix.md`
- `docs/phase4_profile_matrix.template.md`
- `docs/phase4_closeout.template.md`

</details>

## Reviewer Entry Point

Start with `REVIEWER.md` for a guided map of the backend, frontend, algorithms, data structures, tests, and API surfaces. Then read `docs/architecture.md` for the system model, `docs/phase3_closeout.md` and `docs/phase4_closeout.md` for validation evidence, and `docs/phase6_closeout.md` for operational-hardening evidence.
