#!/usr/bin/env bash
set -euo pipefail

report_dir="${1:-./artifacts/phase4/preflight}"
mkdir -p "$report_dir"

required_env=(
  DATABASE_URL
  HELIUS_API_KEY
  HELIUS_BASE_URL
  MAX_WALLETS_PER_RUN
  MAX_TX_PAGES_PER_WALLET
  MAX_TX_PER_WALLET
  MAX_CONCURRENT_WALLETS
  WALLET_SYNC_TIMEOUT_SECONDS
  RUN_TIMEOUT_SECONDS
  MAX_HELIUS_RETRIES
  HELIUS_REQUEST_DELAY_MS
)

missing=0
for key in "${required_env[@]}"; do
  if [[ -z "${!key:-}" ]]; then
    echo "missing required env: $key" >&2
    missing=1
  fi
done
if [[ "$missing" -ne 0 ]]; then
  exit 1
fi

echo "[preflight] running corpus validation (strict miss reason)"
go run ./cmd/scanner validate-corpus \
  --fixtures-root data/fixtures \
  --strict-miss-reason \
  --report-out "$report_dir/corpus_validation_strict.json"

echo "[preflight] saving resolved execution bounds"
{
  echo "timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  for key in "${required_env[@]}"; do
    echo "$key=${!key}"
  done
} > "$report_dir/bounds.env.snapshot"

echo "[preflight] complete"
echo "- corpus report: $report_dir/corpus_validation_strict.json"
echo "- bounds snapshot: $report_dir/bounds.env.snapshot"
