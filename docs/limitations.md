# PoisonTrace Limitations (Phase 0–1)

## 1. Bounded History Can Produce False Negatives

Constraint:
- PoisonTrace scans bounded baseline and scan windows.

Impact:
- history outside configured bounds can make newness and required gates `UNKNOWN`, which means no candidate emission and `incomplete_window = true` with a persisted reason.

## 2. Unresolved SPL Owner Endpoints

Constraint:
- Some SPL transfers do not expose resolvable owner endpoints in Helius Enhanced fields.

Impact:
- transfers are persisted as `unresolved_owner`, marked non-poisoning-ready, excluded from candidate logic, and can reduce recall.

## 3. No Confirmed Victim Attribution

Constraint:
- Phase 0–1 detects probable poisoning injection candidates only.

Impact:
- outputs do not prove user confusion, incorrect fund routing, or realized loss; victim attribution claims are out of scope.

## 4. No Campaign Clustering / Attacker Attribution

Constraint:
- The system is focal-wallet scoped.

Impact:
- attacker clustering across wallets, campaign graphing, and entity attribution are deferred.

## 5. Dependency on Helius Data Shape and Quality

Constraint:
- Pipeline correctness depends on complete and consistent Helius Enhanced transfer fields, owner endpoints, and token classification.

Impact:
- source data inconsistencies increase `UNKNOWN`/`unresolved` outcomes and lower detection recall.

## 6. Dust Threshold Limits

Constraint:
- Dust classification is threshold-based per asset (`SOL` or token mint), not dynamic or economic-value-aware.

Impact:
- missing thresholds produce `is_dust = UNKNOWN`, which blocks candidates and sets `incomplete_window = true`; threshold quality directly affects precision/recall.

## 7. Strict Fail-Closed Candidate Policy May Reduce Recall

Constraint:
- Required-gate policy is fail-closed.

Impact:
- when any required gate is `UNKNOWN`, candidates are blocked by design, improving safety while suppressing some true positives under incomplete baseline or metadata gaps.

## 8. Two-Injection Gate Trades Recall for Precision

Constraint:
- Phase 1 requires at least 2 qualifying inbound dust/zero events per focal+counterparty within the scan window.

Impact:
- one-off noise drops, but single-injection poisoning attempts can be missed.

## 9. Event-Ordering Dependency Residual Risk

Constraint:
- Provider-side omissions in instruction or transfer metadata can weaken fingerprint strength in edge cases.

Impact:
- conservative unresolved classification increases non-eligible events, and required-gate `UNKNOWN` outcomes block candidate emission.
