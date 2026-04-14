-- Phase 0 core schema

CREATE TABLE IF NOT EXISTS wallets (
  id BIGSERIAL PRIMARY KEY,
  address TEXT NOT NULL UNIQUE,
  label TEXT,
  source TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_synced_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS transactions (
  id BIGSERIAL PRIMARY KEY,
  signature TEXT NOT NULL,
  transfer_index INT NOT NULL,
  transfer_fingerprint TEXT NOT NULL,
  slot BIGINT NOT NULL,
  block_time TIMESTAMPTZ NOT NULL,
  source_owner_address TEXT,
  destination_owner_address TEXT,
  source_token_account TEXT,
  destination_token_account TEXT,
  amount_raw NUMERIC(78,0) NOT NULL,
  token_mint TEXT,
  asset_type TEXT NOT NULL,
  asset_key TEXT NOT NULL,
  decimals INT,
  normalization_status TEXT NOT NULL,
  normalization_reason_code TEXT,
  poisoning_eligible BOOLEAN NOT NULL DEFAULT FALSE,
  dust_status TEXT NOT NULL DEFAULT 'unknown',
  is_success BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(signature, transfer_fingerprint)
);

CREATE TABLE IF NOT EXISTS wallet_transactions (
  id BIGSERIAL PRIMARY KEY,
  wallet_id BIGINT NOT NULL REFERENCES wallets(id),
  transaction_id BIGINT NOT NULL REFERENCES transactions(id),
  relation_type TEXT NOT NULL CHECK (relation_type IN ('sender', 'receiver')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(wallet_id, transaction_id, relation_type)
);

CREATE TABLE IF NOT EXISTS counterparties (
  id BIGSERIAL PRIMARY KEY,
  focal_wallet_id BIGINT NOT NULL REFERENCES wallets(id),
  counterparty_address TEXT NOT NULL,
  first_seen_at TIMESTAMPTZ NOT NULL,
  last_seen_at TIMESTAMPTZ NOT NULL,
  interaction_count BIGINT NOT NULL DEFAULT 0,
  first_inbound_at TIMESTAMPTZ,
  last_inbound_at TIMESTAMPTZ,
  inbound_count BIGINT NOT NULL DEFAULT 0,
  first_outbound_at TIMESTAMPTZ,
  last_outbound_at TIMESTAMPTZ,
  outbound_count BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(focal_wallet_id, counterparty_address)
);

CREATE TABLE IF NOT EXISTS ingestion_runs (
  id BIGSERIAL PRIMARY KEY,
  status TEXT NOT NULL,
  started_at TIMESTAMPTZ NOT NULL,
  completed_at TIMESTAMPTZ,
  wallets_requested INT NOT NULL DEFAULT 0,
  wallets_processed INT NOT NULL DEFAULT 0,
  wallets_failed INT NOT NULL DEFAULT 0,
  wallets_skipped INT NOT NULL DEFAULT 0,
  transactions_fetched INT NOT NULL DEFAULT 0,
  transactions_inserted INT NOT NULL DEFAULT 0,
  transactions_linked INT NOT NULL DEFAULT 0,
  transactions_failed_to_normalize INT NOT NULL DEFAULT 0,
  owner_unresolved_count INT NOT NULL DEFAULT 0,
  decimals_unresolved_count INT NOT NULL DEFAULT 0,
  counterparties_created INT NOT NULL DEFAULT 0,
  counterparties_updated INT NOT NULL DEFAULT 0,
  retry_exhausted_count INT NOT NULL DEFAULT 0,
  notes TEXT
);

CREATE TABLE IF NOT EXISTS wallet_sync_runs (
  id BIGSERIAL PRIMARY KEY,
  wallet_id BIGINT NOT NULL REFERENCES wallets(id),
  ingestion_run_id BIGINT NOT NULL REFERENCES ingestion_runs(id),
  status TEXT NOT NULL,
  started_at TIMESTAMPTZ NOT NULL,
  completed_at TIMESTAMPTZ,
  baseline_start_at TIMESTAMPTZ NOT NULL,
  baseline_end_at TIMESTAMPTZ NOT NULL,
  scan_start_at TIMESTAMPTZ NOT NULL,
  scan_end_at TIMESTAMPTZ NOT NULL,
  baseline_complete BOOLEAN NOT NULL DEFAULT FALSE,
  incomplete_window BOOLEAN NOT NULL DEFAULT FALSE,
  truncation_reason TEXT,
  transactions_fetched INT NOT NULL DEFAULT 0,
  transactions_inserted INT NOT NULL DEFAULT 0,
  transactions_linked INT NOT NULL DEFAULT 0,
  transactions_failed_to_normalize INT NOT NULL DEFAULT 0,
  counterparties_created INT NOT NULL DEFAULT 0,
  counterparties_updated INT NOT NULL DEFAULT 0,
  error_code TEXT,
  error_message TEXT,
  notes TEXT
);

CREATE INDEX IF NOT EXISTS idx_transactions_signature ON transactions(signature);
CREATE INDEX IF NOT EXISTS idx_transactions_block_time ON transactions(block_time);
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_wallet_id ON wallet_transactions(wallet_id);
CREATE INDEX IF NOT EXISTS idx_counterparties_focal_wallet_id ON counterparties(focal_wallet_id);
CREATE INDEX IF NOT EXISTS idx_wallet_sync_runs_wallet_id ON wallet_sync_runs(wallet_id);
