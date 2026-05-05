#!/usr/bin/env bash
set -euo pipefail

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

base="./artifacts/phase4/repro_run_${run_id}"
a="$base/a"
b="$base/b"
mkdir -p "$a" "$b"

export_cmd() {
  local out_dir="$1"
  go run ./cmd/scanner export-dataset --out-dir "$out_dir" --run-id "$run_id"
}

echo "[repro] export A"
export_cmd "$a"
echo "[repro] export B"
export_cmd "$b"

files=(ingestion_runs.jsonl wallet_sync_runs.jsonl poisoning_candidates.jsonl manifest.json)
report="$base/repro_check.txt"
: > "$report"

echo "run_id=$run_id" | tee -a "$report" >/dev/null
echo "generated_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)" | tee -a "$report" >/dev/null

fail=0
for f in "${files[@]}"; do
  ha="$(shasum -a 256 "$a/$f" | awk '{print $1}')"
  hb="$(shasum -a 256 "$b/$f" | awk '{print $1}')"
  echo "$f:a=$ha" | tee -a "$report" >/dev/null
  echo "$f:b=$hb" | tee -a "$report" >/dev/null
  if [[ "$ha" != "$hb" ]]; then
    echo "$f: MISMATCH" | tee -a "$report" >/dev/null
    fail=1
  else
    echo "$f: MATCH" | tee -a "$report" >/dev/null
  fi
done

if [[ "$fail" -ne 0 ]]; then
  echo "repro_check=FAIL" | tee -a "$report" >/dev/null
  echo "repro mismatch detected. report: $report" >&2
  exit 1
fi

echo "repro_check=PASS" | tee -a "$report" >/dev/null
echo "repro report: $report"
