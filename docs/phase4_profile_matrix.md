# Phase 4 Profile Matrix (Locked)

Date: 2026-05-05
Owner: Codex + Anjelo
Corpus commit SHA: 1d4cf0c

## Locked Decisions

- Scan window bounds: `2026-04-28T00:00:00Z` to `2026-05-05T00:00:00Z`
- Baseline lookback days: `90`
- Dust threshold source/version: `db/seeds/asset_thresholds.seed.sql` (seeded on 2026-05-05)
- Lookalike config: `LOOKALIKE_RECENCY_DAYS=30`, `LOOKALIKE_PREFIX_MIN=4`, `LOOKALIKE_SUFFIX_MIN=4`, `LOOKALIKE_SINGLE_SIDE_MIN=6`, `MIN_INJECTION_COUNT=2`
- Retry/timeout semantics: `MAX_HELIUS_RETRIES`, wallet/run timeout caps from `.env`; partial/incomplete windows persisted with reasons
- Candidate uniqueness keys: `UNIQUE(wallet_sync_run_id, signature, transfer_index)`
- Transfer uniqueness keys: `UNIQUE(signature, transfer_fingerprint)`

## Profiles

| Profile | MAX_WALLETS_PER_RUN | MAX_TX_PAGES_PER_WALLET | MAX_TX_PER_WALLET | MAX_CONCURRENT_WALLETS | WALLET_SYNC_TIMEOUT_SECONDS | RUN_TIMEOUT_SECONDS | MAX_HELIUS_RETRIES | HELIUS_REQUEST_DELAY_MS |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| canary_low | 3 | 5 | 300 | 1 | 120 | 600 | 4 | 150 |
| standard_low | 10 | 12 | 900 | 2 | 180 | 900 | 4 | 150 |
| stress_low | 25 | 20 | 1500 | 3 | 240 | 1200 | 4 | 150 |

## Wallet Set Used

- `2AFrdmeRKiSk3vJPLnYZAbeKw5spzw7aLUHDvw7Ni2Wy`
- `2Ag5Be7xiS7B6pAD5hhCjkDCFz4teXSfSyUYCcb9qRcS`
- `2pTph6zKX5F6ZLhb7qdcxWKE9BCAB5jYPgQvfDXG21R8`

## Acceptance Gates

- Canary pass criteria: integrity check pass + repro check pass + no truncation-rate regression.
- Standard pass criteria: same as canary with higher caps and stable status distribution.
- Stress pass criteria: same as standard under higher concurrency and cap limits.
