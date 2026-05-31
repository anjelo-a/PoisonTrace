-- Phase 1 detection artifacts

CREATE TABLE IF NOT EXISTS asset_thresholds (
  id BIGSERIAL PRIMARY KEY,
  asset_key TEXT NOT NULL,
  token_mint TEXT,
  dust_amount_raw_threshold NUMERIC(78,0) NOT NULL,
  active_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  active_to TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(asset_key, active_from)
);

CREATE TABLE IF NOT EXISTS poisoning_candidates (
  id BIGSERIAL PRIMARY KEY,
  wallet_sync_run_id BIGINT NOT NULL REFERENCES wallet_sync_runs(id),
  focal_wallet_id BIGINT NOT NULL REFERENCES wallets(id),
  signature TEXT NOT NULL,
  transfer_index INT NOT NULL,
  suspicious_counterparty TEXT NOT NULL,
  matched_legit_counterparty TEXT NOT NULL,
  token_mint TEXT,
  amount_raw NUMERIC(78,0) NOT NULL,
  block_time TIMESTAMPTZ NOT NULL,
  is_zero_value BOOLEAN NOT NULL,
  is_dust BOOLEAN NOT NULL,
  is_new_counterparty BOOLEAN NOT NULL,
  is_inbound BOOLEAN NOT NULL,
  legit_last_seen_at TIMESTAMPTZ NOT NULL,
  recency_days INT NOT NULL,
  repeat_injection_count INT NOT NULL,
  incomplete_window BOOLEAN NOT NULL DEFAULT FALSE,
  unknown_gate_reason TEXT,
  match_rule_version TEXT NOT NULL DEFAULT 'phase1-v1',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(wallet_sync_run_id, signature, transfer_index)
);

CREATE INDEX IF NOT EXISTS idx_poisoning_candidates_focal_wallet ON poisoning_candidates(focal_wallet_id);
CREATE INDEX IF NOT EXISTS idx_poisoning_candidates_counterparty ON poisoning_candidates(wallet_sync_run_id, suspicious_counterparty);
