# Azure Deployment Notes (Phase 0–1)

This project is local-first for development.
Azure is only used for deployment proof/demo in Phase 0–1.

## Suggested shape
- Azure Container Apps for scanner runtime
- Azure Database for PostgreSQL
- Azure Key Vault for secrets

## Required env vars
Use `.env.example` as baseline and configure all required bounds.

## Safety requirements
- Keep small concurrency defaults in cloud demo.
- Enforce run and wallet timeouts.
- Do not schedule unbounded runs.
