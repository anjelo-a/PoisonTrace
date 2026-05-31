# Azure Deployment Notes (Phase 0–1)

This project is local-first for development.
Azure is only used for deployment proof/demo in Phase 0–1.

## Suggested shape
- Azure Container Apps for scanner runtime
- Azure Database for PostgreSQL
- Azure Key Vault for secrets

## API deployment checklist (web console backend)
- Container command must run API mode:
  - `scanner serve-api --addr :8080`
- Container Apps ingress target port must be `8080`.
- Smoke checks (must return quickly):
  - `GET /healthz` -> `200 {"status":"ok"}`
  - `GET /api/overview` -> `200` (or a fast `4xx/5xx`, but never hang)

### Docker default
The Docker image in `ops/deploy/docker/Dockerfile` defaults to:
- `CMD ["serve-api", "--addr", ":8080"]`

If you need scanner batch mode for a job, override container command at deploy time.

## Required env vars
Use `ops/config.env.example` as baseline and configure all required bounds.

## Safety requirements
- Keep small concurrency defaults in cloud demo.
- Enforce run and wallet timeouts.
- Do not schedule unbounded runs.
