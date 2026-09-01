#!/usr/bin/env bash
# Hold docs/ to the Ranke Documentation Format, whose rule ids are cited below.
# Compiling proves nothing about the format: a bare term list and an item() both
# render, so without this the format is enforced by whoever reads the diff.
#
# Reads the construct groups out of the fetched shared/constructs.typ, so a group
# that changes upstream changes what this accepts without an edit here.
#
#   DOCS_DIR    the tree to check         (default docs)
#   PAPERS_DIR  where the fetch landed    (default docs/papers)
set -euo pipefail

docs="${DOCS_DIR:-docs}"
papers="${PAPERS_DIR:-docs/papers}"
constructs="$papers/shared/constructs.typ"
fail=0

say() { echo "  $*"; fail=1; }

[ -f "$constructs" ] || { echo "check-docs: no $constructs — run make docs-papers first" >&2; exit 1; }

# The names in one `#let <group> = ( "a", "b", )` list, one per line.
group() {
	awk -v g="$1" '
		$0 ~ "^#let " g " *= *\\(" { inside = 1; next }
		inside && /^\)/            { exit }
		inside                     { if (match($0, /"[^"]+"/)) print substr($0, RSTART + 1, RLENGTH - 2) }
	' "$constructs"
}

paper_only=$(group paper)

# G-ROOT: one root, applying the handbook show rule.
root="$docs/index.typ"
[ -f "$root" ] || say "G-ROOT: $root is missing"
grep -q '^#import "handbook.typ"' "$root" || say "G-ROOT: $root does not import handbook.typ"
grep -q '^#show: handbook' "$root" || say "G-ROOT: $root does not apply the handbook show rule"

for f in "$docs"/*.typ; do
	name=$(basename "$f")
	case "$name" in
		index.typ|vocabulary.typ|handbook.typ) continue ;;   # the root, and two supplied files
	esac

	# G-CHAPTER: opens with the vocabulary import and a labelled level-one heading.
	head -1 "$f" | grep -q '^#import "vocabulary.typ": \*$' \
		|| say "G-CHAPTER: $name does not open with #import \"vocabulary.typ\": *"
	grep -qE '^= .+ <[a-z]+:[a-z0-9-]+>$' "$f" \
		|| say "G-CHAPTER: $name has no level-one heading carrying a label"
	grep -q '^#include ' "$f" && say "G-CHAPTER: $name includes another chapter"

	# G-IMPORT: vocabulary.typ and nothing else.
	others=$(grep '^#import ' "$f" | grep -v '^#import "vocabulary.typ": \*$' || true)
	[ -z "$others" ] || say "G-IMPORT: $name imports more than vocabulary.typ: $others"
done

for f in "$docs"/*.typ; do
	name=$(basename "$f")
	case "$name" in index.typ|vocabulary.typ|handbook.typ) continue ;; esac

	# G-CONSTRUCTS: the paper group belongs to the papers.
	for c in $paper_only; do
		grep -qE "#$c[([]" "$f" && say "G-CONSTRUCTS: $name calls the paper construct $c"
	done

	# G-NOLAYOUT: the functions HTML export ignores or mangles.
	for c in align columns place pagebreak image; do
		grep -qE "#$c\(" "$f" && say "G-NOLAYOUT: $name calls $c()"
	done
	grep -qE '^#set (page|text|par)' "$f" && say "G-NOLAYOUT: $name sets document-level layout"
	grep -q '^#show' "$f" && say "G-NOLAYOUT: $name applies a show rule; the look belongs to the backend"
done

[ "$fail" -eq 0 ] || { echo "check-docs: the tree does not follow the documentation format" >&2; exit 1; }
chapters=$(ls "$docs"/*.typ | grep -vE '/(index|vocabulary|handbook)\.typ$' | wc -l | tr -d ' ')
echo "docs format OK (root + $chapters chapters in $docs)"
