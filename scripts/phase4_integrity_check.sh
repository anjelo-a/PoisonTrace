#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "DATABASE_URL is required" >&2
  exit 1
fi

run_id=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --run-id)
      run_id="${2:-}"
      shift 2
      ;;
    *)
      echo "unknown arg: $1" >&2
      echo "usage: $0 --run-id <id>" >&2
      exit 2
      ;;
  esac
done

if [[ -z "$run_id" ]]; then
  echo "usage: $0 --run-id <id>" >&2
  exit 2
fi

out_dir="./artifacts/phase4/run_${run_id}"
mkdir -p "$out_dir"
report="$out_dir/integrity_check.txt"

psql_cmd=(psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -qAt)

check_count() {
  local label="$1"
  local sql="$2"
  local count
  count="$(${psql_cmd[@]} -c "$sql")"
  echo "$label=$count" | tee -a "$report" >/dev/null
  if [[ "$count" != "0" ]]; then
    echo "integrity check failed: $label expected 0, got $count" >&2
    exit 1
  fi
}

: > "$report"
echo "run_id=$run_id" | tee -a "$report" >/dev/null
echo "generated_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)" | tee -a "$report" >/dev/null

check_count "incomplete_wallet_sync_missing_reason" "
SELECT COUNT(*)
FROM wallet_sync_runs
WHERE ingestion_run_id = ${run_id}
  AND incomplete_window = TRUE
  AND COALESCE(NULLIF(BTRIM(unknown_gate_reason), ''), '') = '';
"

check_count "baseline_complete_with_truncation_or_timeout" "
SELECT COUNT(*)
FROM wallet_sync_runs
WHERE ingestion_run_id = ${run_id}
  AND baseline_complete = TRUE
  AND (
    status IN ('timeout', 'failed', 'canceled')
    OR COALESCE(NULLIF(BTRIM(truncation_reason), ''), '') LIKE '%baseline_truncation:%'
    OR COALESCE(NULLIF(BTRIM(unknown_gate_reason), ''), '') LIKE '%baseline_truncation:%'
  );
"

check_count "candidate_row_with_unknown_marker" "
SELECT COUNT(*)
FROM poisoning_candidates pc
JOIN wallet_sync_runs wsr ON wsr.id = pc.wallet_sync_run_id
WHERE wsr.ingestion_run_id = ${run_id}
  AND (
    pc.incomplete_window = TRUE
    OR COALESCE(NULLIF(BTRIM(pc.unknown_gate_reason), ''), '') <> ''
  );
"

check_count "transfer_identity_duplicates" "
SELECT COUNT(*)
FROM (
  SELECT t.signature, t.transfer_fingerprint, COUNT(*) AS c
  FROM transactions t
  JOIN wallet_transactions wt ON wt.transaction_id = t.id
  JOIN wallet_sync_runs wsr ON wsr.wallet_id = wt.wallet_id
  WHERE wsr.ingestion_run_id = ${run_id}
  GROUP BY t.signature, t.transfer_fingerprint
  HAVING COUNT(*) > 1
) d;
"

check_count "candidate_identity_duplicates" "
SELECT COUNT(*)
FROM (
  SELECT pc.wallet_sync_run_id, pc.signature, pc.transfer_index, COUNT(*) AS c
  FROM poisoning_candidates pc
  JOIN wallet_sync_runs wsr ON wsr.id = pc.wallet_sync_run_id
  WHERE wsr.ingestion_run_id = ${run_id}
  GROUP BY pc.wallet_sync_run_id, pc.signature, pc.transfer_index
  HAVING COUNT(*) > 1
) d;
"

${psql_cmd[@]} <<SQL | tee -a "$report" >/dev/null
SELECT
  'wallet_sync_status_breakdown=' || COALESCE(string_agg(status || ':' || cnt, ',' ORDER BY status), '')
FROM (
  SELECT status, COUNT(*) AS cnt
  FROM wallet_sync_runs
  WHERE ingestion_run_id = ${run_id}
  GROUP BY status
) s;

SELECT
  'wallets_total=' || COUNT(*)
FROM wallet_sync_runs
WHERE ingestion_run_id = ${run_id};

SELECT
  'wallets_incomplete=' || COUNT(*)
FROM wallet_sync_runs
WHERE ingestion_run_id = ${run_id}
  AND incomplete_window = TRUE;

SELECT
  'candidates_total=' || COUNT(*)
FROM poisoning_candidates pc
JOIN wallet_sync_runs wsr ON wsr.id = pc.wallet_sync_run_id
WHERE wsr.ingestion_run_id = ${run_id};
SQL

echo "integrity_check=PASS" | tee -a "$report" >/dev/null
echo "integrity report: $report"
