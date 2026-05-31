-- Phase 1 run counter persistence hardening

ALTER TABLE ingestion_runs
  ADD COLUMN IF NOT EXISTS poisoning_candidates_inserted INT NOT NULL DEFAULT 0;

ALTER TABLE wallet_sync_runs
  ADD COLUMN IF NOT EXISTS poisoning_candidates_inserted INT NOT NULL DEFAULT 0;
