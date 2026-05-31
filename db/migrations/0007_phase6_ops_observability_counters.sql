-- Phase 6 ops observability counters for guardrail confidence.

ALTER TABLE ingestion_runs
  ADD COLUMN IF NOT EXISTS unsupported_asset_count INT NOT NULL DEFAULT 0;

ALTER TABLE ingestion_runs
  ADD COLUMN IF NOT EXISTS unknown_gate_block_count INT NOT NULL DEFAULT 0;

ALTER TABLE ingestion_runs
  ADD COLUMN IF NOT EXISTS candidate_block_count INT NOT NULL DEFAULT 0;

ALTER TABLE ingestion_runs
  ADD COLUMN IF NOT EXISTS wallet_timeout_count INT NOT NULL DEFAULT 0;

ALTER TABLE ingestion_runs
  ADD COLUMN IF NOT EXISTS wallet_cap_hit_count INT NOT NULL DEFAULT 0;

ALTER TABLE wallet_sync_runs
  ADD COLUMN IF NOT EXISTS unsupported_asset_count INT NOT NULL DEFAULT 0;

ALTER TABLE wallet_sync_runs
  ADD COLUMN IF NOT EXISTS unknown_gate_block_count INT NOT NULL DEFAULT 0;

ALTER TABLE wallet_sync_runs
  ADD COLUMN IF NOT EXISTS candidate_block_count INT NOT NULL DEFAULT 0;
