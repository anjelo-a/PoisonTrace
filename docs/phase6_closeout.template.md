# Phase 6 Closeout Template

Date:
Owner:
Contract Version (failure taxonomy):

## Scope

- Profiles executed:
- Scan windows:
- Wallet sets:
- Forced-failure scenarios executed:

## Exit Criteria Results

1. Canonical failure classification coverage: `MET` / `NOT MET`
2. Bounded interruption finalization persistence: `MET` / `NOT MET`
3. Idempotent rerun stability after failure: `MET` / `NOT MET`
4. Deterministic operational-health exports: `MET` / `NOT MET`
5. Phase 0-1 invariants and fixture gates preserved: `MET` / `NOT MET`

## Evidence Artifacts

- Unit test reports:
- Integration test reports:
- Idempotency/recovery test reports:
- Phase 6 checks report (machine-readable):
- Ops API snapshots (`/api/ops/runs`, `/api/ops/wallet-sync`, `/api/ops/failures`):
- Dataset export directories (with operational health artifacts):
- Reproducibility hash reports:

## Reliability Metrics Summary

- run timeout rate:
- wallet timeout rate:
- cancel rate:
- retry exhausted count:
- lock conflict count:
- stale lock takeover count:
- incomplete window rate:
- uncategorized terminal failures:

## Canonical Failure-Class Distribution

- timeout:
- canceled:
- retry_exhausted:
- lock_conflict:
- persistence_error:
- normalization_error:
- upstream_error:
- unknown_required_gate:

## Open Risks / Follow-Ups

1.
2.
3.

## Final Verdict

- `PASS` / `BLOCKED`
- Sign-off notes:
