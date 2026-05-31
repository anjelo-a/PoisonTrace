-- Seed thresholds for local development and bounded smoke runs.
-- Values are conservative placeholders and must be tuned from known poisoning corpus.
-- Use a stable historical active_from so historical scans do not classify SOL dust as unknown.

INSERT INTO asset_thresholds (asset_key, token_mint, dust_amount_raw_threshold, active_from)
VALUES
  ('SOL', NULL, 1000, '1970-01-01T00:00:00Z'),
  ('So11111111111111111111111111111111111111112', 'So11111111111111111111111111111111111111112', 1000, '1970-01-01T00:00:00Z'),
  ('EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v', 'EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v', 1000, '1970-01-01T00:00:00Z'),
  ('Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB', 'Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB', 1000, '1970-01-01T00:00:00Z')
ON CONFLICT (asset_key, active_from) DO UPDATE
SET token_mint = EXCLUDED.token_mint,
    dust_amount_raw_threshold = EXCLUDED.dust_amount_raw_threshold,
    active_to = NULL;
