-- Phase 1 runtime hardening and lock coordination

ALTER TABLE wallet_sync_runs
  ADD COLUMN IF NOT EXISTS unknown_gate_reason TEXT;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'wallet_sync_runs_unknown_reason_when_incomplete'
  ) THEN
    ALTER TABLE wallet_sync_runs
      ADD CONSTRAINT wallet_sync_runs_unknown_reason_when_incomplete
      CHECK (
        incomplete_window = FALSE
        OR COALESCE(NULLIF(BTRIM(unknown_gate_reason), ''), '') <> ''
      );
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'poisoning_candidates_unknown_reason_when_incomplete'
  ) THEN
    ALTER TABLE poisoning_candidates
      ADD CONSTRAINT poisoning_candidates_unknown_reason_when_incomplete
      CHECK (
        incomplete_window = FALSE
        OR COALESCE(NULLIF(BTRIM(unknown_gate_reason), ''), '') <> ''
      );
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS wallet_locks (
  wallet_address TEXT PRIMARY KEY,
  acquired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  acquired_until TIMESTAMPTZ NOT NULL,
  holder_token TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (acquired_until > acquired_at)
);

CREATE INDEX IF NOT EXISTS idx_wallet_locks_acquired_until ON wallet_locks(acquired_until);
