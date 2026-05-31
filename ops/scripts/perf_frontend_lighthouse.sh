#!/usr/bin/env bash
set -euo pipefail

PORT="${PERF_WEB_PORT:-4173}"
HOST="${PERF_WEB_HOST:-127.0.0.1}"
TARGET_URL="${PERF_WEB_URL:-http://${HOST}:${PORT}/}"
TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
OUT_DIR="${PERF_OUT_DIR:-artifacts/performance/${TIMESTAMP}/frontend}"
REPORT_PREFIX="${OUT_DIR}/lighthouse"
SUMMARY_OUT="${OUT_DIR}/summary.md"
CHROME_FLAGS="${PERF_CHROME_FLAGS:---headless=new --no-sandbox}"

mkdir -p "${OUT_DIR}"

echo "Building web app"
npm run web:build

cleanup() {
  if [[ -n "${PREVIEW_PID:-}" ]]; then
    kill "${PREVIEW_PID}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

echo "Starting Vite preview on ${HOST}:${PORT}"
npm exec --workspace @poisontrace/web -- vite preview --host "${HOST}" --port "${PORT}" > "${OUT_DIR}/vite-preview.log" 2>&1 &
PREVIEW_PID="$!"

for _ in $(seq 1 60); do
  if curl -fsS "${TARGET_URL}" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

if ! curl -fsS "${TARGET_URL}" >/dev/null 2>&1; then
  echo "Vite preview did not become ready at ${TARGET_URL}" >&2
  echo "Preview log: ${OUT_DIR}/vite-preview.log" >&2
  exit 1
fi

echo "Running Lighthouse against ${TARGET_URL}"
npx lighthouse "${TARGET_URL}" \
  --quiet \
  --chrome-flags="${CHROME_FLAGS}" \
  --output=json \
  --output=html \
  --output-path="${REPORT_PREFIX}"

JSON_OUT="$(find "${OUT_DIR}" -maxdepth 1 -type f -name 'lighthouse*.json' | head -n 1)"

if [[ -z "${JSON_OUT}" ]]; then
  echo "Lighthouse JSON report was not created in ${OUT_DIR}" >&2
  exit 1
fi

node --input-type=module - "${JSON_OUT}" "${SUMMARY_OUT}" "${TARGET_URL}" <<'NODE'
import fs from 'node:fs'

const [jsonPath, summaryPath, targetUrl] = process.argv.slice(2)
const report = JSON.parse(fs.readFileSync(jsonPath, 'utf8'))
const pct = (score) => typeof score === 'number' ? Math.round(score * 100) : 'n/a'
const auditValue = (id) => report.audits?.[id]?.displayValue ?? 'n/a'

const lines = [
  '# Frontend Lighthouse Performance',
  '',
  `- Target: \`${targetUrl}\``,
  `- Timestamp: ${new Date().toISOString()}`,
  '',
  '## Scores',
  '',
  `- Performance: ${pct(report.categories?.performance?.score)}`,
  `- Accessibility: ${pct(report.categories?.accessibility?.score)}`,
  `- Best practices: ${pct(report.categories?.['best-practices']?.score)}`,
  `- SEO: ${pct(report.categories?.seo?.score)}`,
  '',
  '## Core Metrics',
  '',
  `- First Contentful Paint: ${auditValue('first-contentful-paint')}`,
  `- Largest Contentful Paint: ${auditValue('largest-contentful-paint')}`,
  `- Total Blocking Time: ${auditValue('total-blocking-time')}`,
  `- Cumulative Layout Shift: ${auditValue('cumulative-layout-shift')}`,
  `- Speed Index: ${auditValue('speed-index')}`,
  '',
  `Raw JSON: \`${jsonPath}\``,
  '',
]

fs.writeFileSync(summaryPath, lines.join('\n'))
console.log(lines.join('\n'))
NODE
