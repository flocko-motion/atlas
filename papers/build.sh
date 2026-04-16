#!/usr/bin/env bash
# Quick-and-dirty build: markdown -> PDF via pandoc + typst.
#
# Requirements:
#   brew install pandoc librsvg typst

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

PAPER_DIR="01-rankedb"
INPUT="$PAPER_DIR/rankedb.md"
OUTPUT_DIR="bin"

if [[ ! -f "$INPUT" ]]; then
  echo "error: input not found: $INPUT" >&2
  exit 1
fi

mkdir -p "$OUTPUT_DIR"

pandoc "$INPUT" \
  -o "$OUTPUT_DIR/rankedb.pdf" \
  --pdf-engine=typst \
  --resource-path="$PAPER_DIR" \
  --toc

echo "built: $OUTPUT_DIR/rankedb.pdf"
