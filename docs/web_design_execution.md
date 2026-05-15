# Web Design Execution (ZIP-Exact)

## Scope
Core-first execution with ZIP-exact layout fidelity:
1. Overview
2. Candidates
3. Transactions
4. Runs
5. Wallet Sync

Secondary parity pass:
1. Counterparties
2. Exports
3. Settings
4. Landing
5. Methodology

## Parity Checklist
For each screen, verify:
1. Layout grid and section order match ZIP wireframe.
2. Spacing rhythm and typography scale match ZIP.
3. Table structure (header, rows, sticky behavior) matches ZIP.
4. Navigation shell parity on desktop and mobile.
5. Detail drawer behavior (where present) matches ZIP.
6. Loading/empty/error states are explicit and non-misleading.
7. Unknown/partial semantics are shown, never hidden.

## Contract and Data Rules
1. URL query params are the source of truth for filters/pagination.
2. API payload names remain contract-safe (`runId`, `walletSyncRunId`).
3. Candidate endpoints only return emitted probable candidates.
4. Unknown required-gate rows must not be silently normalized.

## Deployment Profile (API + Web)
1. API command: `go run ./cmd/scanner serve-api --addr :8080`
2. API server timeouts:
   - `ReadHeaderTimeout: 5s`
   - `ReadTimeout: 15s`
   - `WriteTimeout: 30s`
   - `IdleTimeout: 60s`
3. API graceful shutdown on `SIGINT/SIGTERM`.
4. Web dev proxy routes `/api` and `/healthz` to `http://localhost:8080`.

## Validation Gates
1. `go test ./...`
2. `npm run contracts:typecheck`
3. `npm run web:typecheck`
4. `npm run web:test`
5. `npm run web:build`
