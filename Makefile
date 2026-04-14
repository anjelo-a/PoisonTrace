.PHONY: build test migrate run-fixture run

build:
	go build ./cmd/scanner

test:
	go test ./...

migrate:
	./scripts/migrate.sh

run:
	go run ./cmd/scanner run --wallets data/seeds/wallets.example.txt --scan-start 2026-04-01T00:00:00Z --scan-end 2026-04-08T00:00:00Z

run-fixture:
	go run ./cmd/scanner replay-fixture --fixture baseline_truncated_newness_unknown
