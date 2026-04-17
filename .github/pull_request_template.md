## Summary

- What changed:
- Why:

## Phase 0-1 Invariant Checklist

- [ ] Candidate logic uses shared gate helpers (no duplicated gate rules across code paths).
- [ ] Runtime entry points validate required params and fail fast on invalid inputs.
- [ ] Boundary tests were added/updated for new numeric and URL inputs (`0`, negative, malformed, and large values as applicable).
- [ ] Unknown required gates remain fail-closed: candidate emission blocked and `incomplete_window`/reason preserved.
- [ ] Min-injection counting only includes inbound transfers that pass base eligibility gates before zero/dust qualification.

## Verification

- [ ] `go test ./...`
- [ ] Invariant guardrail tests (CI step) pass.
