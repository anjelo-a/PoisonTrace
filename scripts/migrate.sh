#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "DATABASE_URL is required"
  exit 1
fi

if ! command -v psql >/dev/null 2>&1; then
  echo "psql is required but was not found in PATH"
  exit 1
fi

for file in migrations/*.sql; do
  echo "applying ${file}"
  psql "${DATABASE_URL}" -v ON_ERROR_STOP=1 -f "${file}"
done

seed_path="${DUST_THRESHOLDS_SEED_PATH:-data/seeds/asset_thresholds.seed.sql}"
if [[ -f "${seed_path}" ]]; then
  echo "applying ${seed_path}"
  psql "${DATABASE_URL}" -v ON_ERROR_STOP=1 -f "${seed_path}"
else
  echo "dust threshold seed not found: ${seed_path}"
fi

echo "migrations applied successfully"
