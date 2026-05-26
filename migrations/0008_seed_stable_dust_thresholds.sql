-- Ensure baseline dust thresholds are effective for historical scan windows.
-- The seed file is still the source for local refreshes; this migration repairs DBs
-- that already ran earlier migrations before seed loading was wired into migrate.sh.

INSERT INTO asset_thresholds (asset_key, token_mint, dust_amount_raw_threshold, active_from)
VALUES
  ('SOL', NULL, 1000, '1970-01-01T00:00:00Z'),
  ('So11111111111111111111111111111111111111112', 'So11111111111111111111111111111111111111112', 1000, '1970-01-01T00:00:00Z')
ON CONFLICT (asset_key, active_from) DO UPDATE
SET token_mint = EXCLUDED.token_mint,
    dust_amount_raw_threshold = EXCLUDED.dust_amount_raw_threshold,
    active_to = NULL;
