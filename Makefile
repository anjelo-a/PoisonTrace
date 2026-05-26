.PHONY: build test test-guardrails test-fixture-metadata validate-corpus migrate run-fixture run daemon-free ts-install ts-check ts-fixtures phase4-preflight phase4-integrity phase4-repro

build:
	go build ./cmd/scanner

test:
	go test ./...

test-guardrails:
	./scripts/ci_guardrails.sh

test-fixture-metadata:
	./scripts/ci_fixture_metadata_lint.sh

validate-corpus:
	mkdir -p artifacts
	go run ./cmd/scanner validate-corpus --fixtures-root data/fixtures --report-out ./artifacts/corpus_validation_report.json

migrate:
	./scripts/migrate.sh

run:
	go run ./cmd/scanner run --wallets data/seeds/wallets.example.txt --scan-start 2026-04-01T00:00:00Z --scan-end 2026-04-08T00:00:00Z

daemon-free:
	go run ./cmd/scanner daemon

run-fixture:
	go run ./cmd/scanner replay-fixture --fixture baseline_truncated_newness_unknown

ts-install:
	npm install

ts-check:
	npm run typecheck

ts-fixtures:
	npm run ts:fixtures

phase4-preflight:
	./scripts/phase4_preflight.sh

phase4-integrity:
	@test -n "$(RUN_ID)" || (echo "RUN_ID is required, example: make phase4-integrity RUN_ID=42" && exit 1)
	./scripts/phase4_integrity_check.sh --run-id $(RUN_ID)

phase4-repro:
	@test -n "$(RUN_ID)" || (echo "RUN_ID is required, example: make phase4-repro RUN_ID=42" && exit 1)
	./scripts/phase4_repro_check.sh --run-id $(RUN_ID)
