.PHONY: build test test-guardrails test-fixture-metadata validate-corpus migrate run-fixture run

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

run-fixture:
	go run ./cmd/scanner replay-fixture --fixture baseline_truncated_newness_unknown
