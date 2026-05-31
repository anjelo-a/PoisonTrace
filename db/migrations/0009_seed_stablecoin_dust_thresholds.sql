-- Add common Solana stablecoin dust thresholds so zero-or-dust gates are known
-- for historical USDC/USDT transfers in bounded scanner runs.

INSERT INTO asset_thresholds (asset_key, token_mint, dust_amount_raw_threshold, active_from)
VALUES
  ('EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v', 'EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v', 1000, '1970-01-01T00:00:00Z'),
  ('Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB', 'Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB', 1000, '1970-01-01T00:00:00Z')
ON CONFLICT (asset_key, active_from) DO UPDATE
SET token_mint = EXCLUDED.token_mint,
    dust_amount_raw_threshold = EXCLUDED.dust_amount_raw_threshold,
    active_to = NULL;
