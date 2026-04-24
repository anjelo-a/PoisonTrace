-- Phase 4 run summary truncation metrics

ALTER TABLE ingestion_runs
  ADD COLUMN IF NOT EXISTS truncation_wallet_count INT NOT NULL DEFAULT 0;

ALTER TABLE ingestion_runs
  ADD COLUMN IF NOT EXISTS truncation_wallet_rate NUMERIC(12,8) NOT NULL DEFAULT 0;
