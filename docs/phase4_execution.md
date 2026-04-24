# Phase 4 Execution Playbook

This playbook covers deterministic batch execution and reproducible dataset export for Phase 4.

## 1) Run a bounded batch scan

Example:

```bash
go run ./cmd/scanner run \
  --wallets ./wallets.seed.txt \
  --scan-start 2026-04-01T00:00:00Z \
  --scan-end 2026-04-08T00:00:00Z
```

Determinism and bounds:
- wallet scheduling order is canonicalized before fan-out.
- run-level notes are emitted in stable wallet-key order.
- run summary persists truncation metrics:
- `truncation_wallet_count`
- `truncation_wallet_rate`

## 2) Export canonical dataset artifacts

### Option A: Export a single ingestion run

```bash
go run ./cmd/scanner export-dataset \
  --out-dir ./artifacts/run_42 \
  --run-id 42
```

### Option B: Export by ingestion-run started_at window

```bash
go run ./cmd/scanner export-dataset \
  --out-dir ./artifacts/window_2026w17 \
  --started-at-from 2026-04-20T00:00:00Z \
  --started-at-to 2026-04-27T00:00:00Z
```

Outputs:
- `ingestion_runs.jsonl`
- `wallet_sync_runs.jsonl`
- `poisoning_candidates.jsonl`
- `manifest.json`

Manifest fields:
- `schema_version`
- `generated_at` (deterministic from selected run timestamps)
- `source_filters`
- per-file `row_count` and `sha256`

## 3) Reproducibility verification

Run the same export twice and compare hashes.

```bash
shasum -a 256 \
  ./artifacts/run_42/ingestion_runs.jsonl \
  ./artifacts/run_42/wallet_sync_runs.jsonl \
  ./artifacts/run_42/poisoning_candidates.jsonl \
  ./artifacts/run_42/manifest.json
```

Expected:
- byte-identical artifacts for same source filter and unchanged DB state.
- stable ordering by deterministic keys:
- candidates: wallet, block_time, signature, transfer_index
- wallet sync runs: wallet, scan_start_at, wallet_sync_run_id
- ingestion runs: ingestion_run_id
