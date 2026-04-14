# PoisonTrace Limitations (Phase 0–1)

## 1. Bounded History Can Produce False Negatives

PoisonTrace intentionally scans bounded windows.
If relevant history is outside baseline/scan bounds:
- legitimate history may be incomplete
- newness may be unknown
- candidates may be suppressed by fail-closed gates

This is expected behavior, not a bug.

## 2. Unresolved SPL Owner Endpoints

Some SPL transfers may not provide resolvable owner endpoints in Helius-enhanced fields.
When owner resolution fails:
- transfer is persisted as `unresolved_owner`
- it is marked non-poisoning-ready
- it is excluded from poisoning candidate logic

Result:
- potential missed signals rather than risky inferred mappings.

## 3. No Confirmed Victim Attribution

Phase 0–1 detects probable poisoning injections only.
It does not prove:
- user confusion
- incorrect fund routing
- realized victim loss

No victim attribution claims should be made from this phase.

## 4. No Campaign Clustering / Attacker Attribution

The system is focal-wallet scoped.
It does not perform:
- attacker clustering across wallets
- campaign graphing
- entity attribution

Those are explicitly deferred.

## 5. Dependency on Helius Data Shape and Quality

Pipeline correctness depends on:
- completeness of Helius Enhanced Transaction transfer fields
- consistency of owner endpoint exposure
- stable token standard classification

If source data is incomplete or inconsistent:
- unknown/unresolved rates increase
- detection recall may drop

## 6. Dust Threshold Limits

Dust classification is threshold-based, not economic-value-aware.
Constraints:
- thresholds must be configured per asset (`SOL` or token mint)
- missing threshold => `is_dust = unknown` (fail-closed for candidates)
- threshold quality directly affects precision/recall

Phase 0–1 does not include dynamic threshold learning.

## 7. Strict Fail-Closed Candidate Policy May Reduce Recall

Rule:
- if any required gate is unknown, candidate is not emitted.

Benefit:
- prevents weak or unsafe inferences.

Cost:
- suppresses some true positives under incomplete baseline or metadata gaps.

## 8. Two-Injection Gate Trades Recall for Precision

Phase 1 requires at least 2 qualifying inbound dust/zero events per focal+counterparty within scan window.
Effect:
- reduces one-off noise
- may miss single-injection poisoning attempts

## 9. Event-Ordering Dependency Residual Risk

Even with fingerprint-based dedup, provider-side field omissions (for instruction refs or transfer metadata) can reduce fingerprint strength in edge cases.
Mitigation in Phase 1:
- fallback to conservative unresolved status when canonical fingerprint fields are incomplete,
- suppress candidate emission on unknown required gates.
