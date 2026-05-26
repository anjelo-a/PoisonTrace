# Free Plan Daemon

`scanner daemon` runs bounded scrape -> filter -> scan cycles continuously while staying under Helius Free limits.

Helius Free limits used by the daemon:

- RPC: 10 requests/second max. Daemon default: `--rpc-rps 5`.
- DAS & Enhanced APIs: 2 requests/second max. Daemon default: `--enhanced-rps 1`.
- Monthly credits: 1,000,000. Daemon default: `--daily-credit-budget 30000`.

Default command:

```sh
make daemon-free
```

Equivalent explicit command:

```sh
go run ./cmd/scanner daemon \
  --cycle-interval 24h \
  --daily-credit-budget 30000 \
  --rpc-rps 5 \
  --enhanced-rps 1 \
  --max-helius-retries 1 \
  --target-count 5
```

Operational behavior:

- Uses a Postgres advisory lock so only one daemon instance runs per database.
- Runs one cycle at a time; cycles never overlap.
- Estimates worst-case cycle credits before starting a cycle.
- Sleeps until the next UTC day if the daily credit budget is exhausted.
- Writes cycle wallet files under `artifacts/daemon/<cycle-id>/`.
- Keeps scanner caps bounded through daemon-specific defaults:
  - `--run-max-tx-pages 5`
  - `--run-max-tx-per-wallet 400`
  - `--run-max-concurrent-wallets 1`
  - `--max-candidates 30`
  - `--candidate-max-pages 3`
  - `--max-helius-retries 1`

For a one-cycle smoke run:

```sh
go run ./cmd/scanner daemon --once
```
