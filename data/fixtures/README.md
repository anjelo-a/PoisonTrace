# Fixture Specification for PoisonTrace Phase 0–1

## Why Fixtures Are Required

Routine tests must not burn live Helius credits.
Fixtures provide:
- deterministic behavior
- stable regression coverage
- low-cost CI execution
- explicit validation of edge cases

Live Helius sanity checks are manual/sparse only.

## Directory Layout

Each fixture case lives under:

`data/fixtures/<case_id>/`

Required files:
1. `meta.json`
2. `raw/helius_page_001.json` (and additional pages as needed)
3. `expected/normalized_transfers.json`
4. `expected/wallet_transactions.json`
5. `expected/counterparties.json`
6. `expected/poisoning_candidates.json`
7. `expected/wallet_sync_run.json`
8. `expected/ingestion_run_delta.json`

Optional:
- `notes.md`
- `raw/helius_page_00N.json`

## `meta.json` Format

```json
{
  "case_id": "poisoning_repeat_inbound_sol",
  "description": "Two inbound zero-value lookalike injections in scan window",
  "focal_wallet": "Focal1111111111111111111111111111111111111",
  "baseline_start": "2026-01-01T00:00:00Z",
  "baseline_end": "2026-03-31T00:00:00Z",
  "scan_start": "2026-03-31T00:00:00Z",
  "scan_end": "2026-04-07T00:00:00Z",
  "expected_in_scope": true,
  "expected_miss_reason": null
}
```

## Raw Helius Page Shape (Consumed Fields)

Raw files mirror Helius Enhanced Transactions response shape enough for parser realism.

Minimal consumed fields per transaction:
- `signature`
- `slot`
- `timestamp` (or equivalent block time)
- `transactionError`
- `nativeTransfers[]`:
  - `fromUserAccount`
  - `toUserAccount`
  - `amount`
- `tokenTransfers[]`:
  - `fromUserAccount`
  - `toUserAccount`
  - `fromTokenAccount`
  - `toTokenAccount`
  - `mint`
  - `tokenAmount.amount`
  - `tokenAmount.decimals`
  - `tokenStandard` (if provided)
  - instruction references if available (instruction index / inner index)

Example:

```json
{
  "transactions": [
    {
      "signature": "5Nf...abc",
      "slot": 345678901,
      "timestamp": 1770001000,
      "transactionError": null,
      "nativeTransfers": [
        {
          "fromUserAccount": "Atk111...",
          "toUserAccount": "Focal111...",
          "amount": 0
        }
      ],
      "tokenTransfers": []
    }
  ],
  "before": "cursor_2"
}
```

## Expected Normalized Output Format

`expected/normalized_transfers.json` is an ordered array of normalized transfer events:

```json
[
  {
    "signature": "5Nf...abc",
    "transfer_index": 0,
    "transfer_fingerprint": "sha256:...",
    "slot": 345678901,
    "block_time": "2026-02-01T10:03:20Z",
    "source_owner_address": "Atk111...",
    "destination_owner_address": "Focal111...",
    "source_token_account": null,
    "destination_token_account": null,
    "asset_type": "native_sol",
    "asset_key": "SOL",
    "token_mint": null,
    "amount_raw": "0",
    "decimals": 9,
    "normalization_status": "resolved",
    "poisoning_eligible": true,
    "dust_status": "true",
    "is_success": true
  }
]
```

Allowed `normalization_status` values:
- `resolved`
- `unresolved_owner`
- `unsupported_asset`
- `failed`

## Expected Candidate Output Format

`expected/poisoning_candidates.json` contains only emitted candidates.

```json
[
  {
    "wallet_sync_run_id": "wsr_001",
    "focal_wallet": "Focal111...",
    "signature": "5Nf...abc",
    "transfer_index": 0,
    "suspicious_counterparty": "Atk111...",
    "matched_legit_counterparty": "Legit111...",
    "lookalike_prefix_match_len": 6,
    "lookalike_suffix_match_len": 4,
    "recency_days": 12,
    "is_new_counterparty": true,
    "baseline_complete": true,
    "injection_count_in_scan_window": 2,
    "incomplete_window": false
  }
]
```

Uniqueness invariant:
- `(wallet_sync_run_id, signature, transfer_index)` must be unique.

## Unknown Gate Behavior in Fixtures

When any required gate is unknown:
- candidate must be absent from `poisoning_candidates.json`
- `expected/wallet_sync_run.json` must set:
  - `"incomplete_window": true`
- include reason in notes or expected run fields (for example `unknown_gate_reason`).

## Window Semantics

Window semantics are mandatory:
- baseline: `[baseline_start, baseline_end)`
- scan: `[scan_start, scan_end)`
- no event may belong to both windows.

## Required Canonical Fixture Cases

1. `baseline_truncated_newness_unknown`
2. `spl_unresolved_owner_non_poisoning_ready`
3. `missing_threshold_dust_unknown`
4. `same_signature_multiple_wallets`
5. `repeat_inbound_two_injections_pass`
6. `single_injection_fail_min_count`
7. `lookalike_prefix_only_pass`
8. `lookalike_suffix_only_pass`
9. `legit_baseline_outbound_non_dust_only`
10. `rate_limited_then_retry_success`
11. `wallet_timeout_partial`
12. `max_tx_cap_truncated`
13. `duplicate_event_across_pages`
14. `out_of_order_events_same_signature`
15. `scan_boundary_exact_timestamp`
16. `partial_owner_present`
17. `self_transfer_owner_level`
18. `two_injection_gate_with_unknown_second`
19. `multi_legit_match_tiebreak`

## How Fixtures Are Used in Tests

Test flow:
1. Load fixture `meta.json`.
2. Feed raw pages into normalizer/ingestor.
3. Persist into test DB.
4. Run counterparty and detection pipeline.
5. Compare persisted outputs against all `expected/*.json` files.
6. Assert run counters and status transitions.
7. Assert candidate emission suppression when unknown gates exist.

## Known Case Validation Integration

Known poisoning corpus should be represented as fixture cases with deterministic signatures.
For each case:
- if `expected_in_scope = true`, candidate must exist.
- if false, candidate must be absent and `expected_miss_reason` must match.
