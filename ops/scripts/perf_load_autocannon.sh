#!/usr/bin/env bash
set -euo pipefail

TARGET_URL="${PERF_LOAD_URL:-http://127.0.0.1:8080/healthz}"
DURATION_SECONDS="${PERF_LOAD_DURATION_SECONDS:-30}"
CONNECTIONS="${PERF_LOAD_CONNECTIONS:-10}"
PIPELINING="${PERF_LOAD_PIPELINING:-1}"
TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
OUT_DIR="${PERF_OUT_DIR:-artifacts/performance/${TIMESTAMP}/load}"
JSON_OUT="${OUT_DIR}/autocannon.json"
SUMMARY_OUT="${OUT_DIR}/summary.md"

mkdir -p "${OUT_DIR}"

echo "Running autocannon against ${TARGET_URL}"
echo "Writing ${JSON_OUT}"

npx autocannon \
  --json \
  --duration "${DURATION_SECONDS}" \
  --connections "${CONNECTIONS}" \
  --pipelining "${PIPELINING}" \
  "${TARGET_URL}" > "${JSON_OUT}"

node --input-type=module - "${JSON_OUT}" "${SUMMARY_OUT}" "${TARGET_URL}" "${DURATION_SECONDS}" "${CONNECTIONS}" "${PIPELINING}" <<'NODE'
import fs from 'node:fs'

const [jsonPath, summaryPath, targetUrl, durationSeconds, connections, pipelining] = process.argv.slice(2)
const report = JSON.parse(fs.readFileSync(jsonPath, 'utf8'))
const fmt = (value, digits = 2) => typeof value === 'number' ? value.toFixed(digits) : 'n/a'
const latency = report.latency ?? {}
const requests = report.requests ?? {}
const throughput = report.throughput ?? {}

const lines = [
  '# Autocannon Load Test',
  '',
  `- Target: \`${targetUrl}\``,
  `- Duration: ${durationSeconds}s`,
  `- Connections: ${connections}`,
  `- Pipelining: ${pipelining}`,
  `- Timestamp: ${new Date().toISOString()}`,
  '',
  '## Metrics',
  '',
  `- Requests/sec avg: ${fmt(requests.average)}`,
  `- Requests/sec p90: ${fmt(requests.p90)}`,
  `- Requests/sec p99: ${fmt(requests.p99)}`,
  `- Latency avg: ${fmt(latency.average)} ms`,
  `- Latency p90: ${fmt(latency.p90)} ms`,
  `- Latency p99: ${fmt(latency.p99)} ms`,
  `- Throughput avg: ${fmt(throughput.average)} bytes/sec`,
  `- Total requests: ${report.requests?.total ?? 'n/a'}`,
  `- Total errors: ${report.errors ?? 0}`,
  `- Total timeouts: ${report.timeouts ?? 0}`,
  '',
  `Raw JSON: \`${jsonPath}\``,
  '',
]

fs.writeFileSync(summaryPath, lines.join('\n'))
console.log(lines.join('\n'))
NODE
