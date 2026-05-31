-- Phase 6 backend hardening: settings overrides + async export jobs

CREATE TABLE IF NOT EXISTS app_config_overrides (
  id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  max_wallets_per_run INT,
  max_tx_pages_per_wallet INT,
  max_tx_per_wallet INT,
  max_concurrent_wallets INT,
  wallet_sync_timeout_seconds INT,
  run_timeout_seconds INT,
  max_helius_retries INT,
  helius_request_delay_ms INT,
  baseline_lookback_days INT,
  scan_window_days INT,
  lookalike_recency_days INT,
  lookalike_prefix_min INT,
  lookalike_suffix_min INT,
  lookalike_single_side_min INT,
  min_injection_count INT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS export_jobs (
  id BIGSERIAL PRIMARY KEY,
  run_id BIGINT NOT NULL REFERENCES ingestion_runs(id),
  status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed')),
  out_dir TEXT NOT NULL,
  error_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_export_jobs_run_id_created_at ON export_jobs(run_id, created_at DESC);
