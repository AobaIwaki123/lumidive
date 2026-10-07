#!/usr/bin/env bash
# ==============================================================================
# verify-all.sh - Strict local pre-push verification script for lumidive
# ==============================================================================

set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_DIR"

echo "========================================================"
echo "1. Checking Schema Drift (go generate & git diff)..."
echo "========================================================"
go generate ./...
if ! git diff --exit-code pkg/api/lumidive.gen.go; then
  echo "Error: Uncommitted generated code detected in pkg/api/lumidive.gen.go! Please commit generated files."
  exit 1
fi
echo "OK: Code generation is up to date."
echo ""

echo "========================================================"
echo "2. Running golangci-lint..."
echo "========================================================"
GOPATH_BIN="$(go env GOPATH)/bin"
if [[ -f "${GOPATH_BIN}/golangci-lint" ]]; then
  "${GOPATH_BIN}/golangci-lint" run ./...
else
  golangci-lint run ./...
fi
echo "OK: Linter passed with 0 issues."
echo ""

echo "========================================================"
echo "3. Running Unit & Integration Tests (-race)..."
echo "========================================================"
go test -race -v -cover ./...
echo "OK: All tests passed."
echo ""

echo "========================================================"
echo "4. Building all packages & binaries..."
echo "========================================================"
mkdir -p bin
go build -v -o bin/lumidive ./cmd/lumidive
echo "OK: Build successful."
echo ""

echo "========================================================"
echo "All local verification checks passed with 100% success!"
echo "========================================================"
