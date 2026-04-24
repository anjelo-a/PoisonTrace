#!/usr/bin/env bash

set -euo pipefail

go test ./internal/fixtures -run 'TestFixtureMetadataConsistency' -count=1
