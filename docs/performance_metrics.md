# Performance Metrics

PoisonTrace keeps performance evidence under `artifacts/performance/<timestamp>/`.

## Load Test

Run an autocannon check against a local API endpoint:

```sh
npm run perf:load
```

Defaults:
- URL: `http://127.0.0.1:8080/healthz`
- Duration: `30s`
- Connections: `10`
- Pipelining: `1`

Override with environment variables:

```sh
PERF_LOAD_URL=http://127.0.0.1:8080/api/overview \
PERF_LOAD_DURATION_SECONDS=60 \
PERF_LOAD_CONNECTIONS=25 \
npm run perf:load
```

Outputs:
- `autocannon.json`
- `summary.md`

## Frontend Lighthouse

Run a production build, start Vite preview, and capture Lighthouse output:

```sh
npm run perf:web
```

Defaults:
- URL: `http://127.0.0.1:4173/`
- Port: `4173`

Override with environment variables:

```sh
PERF_WEB_PORT=4174 npm run perf:web
```

Outputs:
- Lighthouse JSON report
- Lighthouse HTML report
- `summary.md`
- `vite-preview.log`

## Full Local Pass

With the API already running on `127.0.0.1:8080`:

```sh
npm run perf
```
