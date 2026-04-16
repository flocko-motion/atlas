#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
mkdir -p bin
cd ../server/go
go build -o ../../ranke-cli/bin/ranke-cli ./cmd/ranke-cli
