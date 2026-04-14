-- Seed thresholds for local development.
-- Values are placeholders and must be tuned from known poisoning corpus.

INSERT INTO asset_thresholds (asset_key, token_mint, dust_amount_raw_threshold, active_from)
VALUES
  ('SOL', NULL, 1000, NOW()),
  ('So11111111111111111111111111111111111111112', 'So11111111111111111111111111111111111111112', 1000, NOW())
ON CONFLICT DO NOTHING;
