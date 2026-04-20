# SKILLS.md

## Purpose

This file is now a compact index for PoisonTrace discoverable skills.
Mission-critical policy and invariants remain in `AGENTS.md`; architecture contracts remain in `docs/architecture.md`.

## Discoverable Skills Map

1. `poisontrace-runtime-guards`
- Path: `.agents/skills/poisontrace-runtime-guards/SKILL.md`
- Use for bounded execution, retries/backoff, timeouts, concurrency, lock behavior, and partial/failed semantics.

2. `poisontrace-poisoning-gates`
- Path: `.agents/skills/poisontrace-poisoning-gates/SKILL.md`
- Use for normalization-to-owner mapping, directionality, unknown-gate blocking, baseline/newness semantics, and candidate gates.

3. `poisontrace-fixtures-and-idempotency`
- Path: `.agents/skills/poisontrace-fixtures-and-idempotency/SKILL.md`
- Use for uniqueness keys, deterministic upserts/reruns, transfer fingerprints, fixture coverage, and CI gate assertions.

## Layering Model

1. `AGENTS.md` defines always-on scope, invariants, and failure semantics.
2. `docs/architecture.md` defines system contracts and detection pipeline behavior.
3. Skills provide task-scoped implementation workflows and must not loosen those contracts.

## Notes

1. Default discovery is repo-scoped `.agents/skills`; no `.codex/config.toml` is required for basic skill discovery.
2. Optional `.codex/config.toml` is only for discovery tuning (for example, enable/disable skills or fallback instruction filenames).

## Delivery Output Requirement

For every response that includes repository file changes, the final response must include:
- `Branch: <branch-name>`
- `Commit: <commit-subject-line>`

If no repository files changed, the final response must include:
- `No branch/commit required (no file changes).`
