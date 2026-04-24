#!/usr/bin/env bash
# Build all papers to PDF via Typst.
# Each subdirectory containing a main.typ is treated as a paper; the output PDF
# takes the directory name. Markdown files are notes, not papers, and are ignored.
#
# Requirements: typst (https://github.com/typst/typst)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

OUTPUT_DIR="pdf"
mkdir -p "$OUTPUT_DIR"

if ! command -v typst >/dev/null 2>&1; then
  echo "error: 'typst' not installed" >&2
  exit 1
fi

shopt -s nullglob
any=0
for paper_dir in */; do
  paper_dir="${paper_dir%/}"
  input="$paper_dir/main.typ"
  if [[ ! -f "$input" ]]; then
    continue
  fi
  any=1
  output="$OUTPUT_DIR/${paper_dir}.pdf"
  typst compile --root "$SCRIPT_DIR" "$input" "$output"
  echo "built: $output"
done

if [[ $any -eq 0 ]]; then
  echo "no main.typ files found under $SCRIPT_DIR" >&2
  exit 1
fi
