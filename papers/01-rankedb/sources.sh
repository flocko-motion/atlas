#!/usr/bin/env bash
# sources.sh — generate sources.gen.md from sources.bib
#
# The rendered bibliography (sources.gen.md) provides anchors that in-paper
# citations can link to, e.g. [Talisman 2026](sources.gen.md#talisman2026provenance).
# Never edit sources.gen.md by hand — re-run this script after updating sources.bib.

set -euo pipefail
cd "$(dirname "$0")"

python3 - <<'PY'
import re
from pathlib import Path

bib_text = Path("sources.bib").read_text()
out_path = Path("sources.gen.md")

# Match @type{key, ...body... } with one level of brace nesting inside fields.
entry_re = re.compile(r"@(\w+)\s*\{\s*([^,\s]+)\s*,(.*?)\n\}", re.DOTALL)
field_re = re.compile(
    r"(\w+)\s*=\s*\{((?:[^{}]|\{[^{}]*\})*)\}", re.DOTALL
)

def parse_entries(text):
    for m in entry_re.finditer(text):
        bibtype = m.group(1).lower()
        key = m.group(2).strip()
        body = m.group(3)
        fields = {
            k.lower(): v.strip()
            for k, v in field_re.findall(body)
        }
        yield key, bibtype, fields

def strip_braces(s):
    s = s.strip()
    while s.startswith("{") and s.endswith("}"):
        s = s[1:-1].strip()
    return s

def short_author(raw):
    if not raw:
        return "Anonymous"
    first = raw.split(" and ")[0]
    first = strip_braces(first)
    if "," in first:
        return first.split(",", 1)[0].strip()
    parts = first.split()
    return parts[-1] if parts else first

def label(fields):
    year = fields.get("year", "n.d.").strip().strip("{}")
    return f"{short_author(fields.get('author', ''))} {year}"

def format_entry(bibtype, f):
    parts = []
    author = f.get("author", "")
    if author:
        parts.append(strip_braces(author) + ".")
    year = f.get("year", "")
    if year:
        parts.append(f"({strip_braces(year)}).")
    title = f.get("title", "")
    if title:
        parts.append(f"*{strip_braces(title)}*.")
    venue = (
        f.get("journal")
        or f.get("booktitle")
        or f.get("howpublished")
        or ""
    )
    if venue:
        parts.append(strip_braces(venue) + ".")
    doi = f.get("doi", "").strip()
    url = f.get("url", "").strip()
    eprint = f.get("eprint", "").strip()
    if doi:
        parts.append(f"[doi:{doi}](https://doi.org/{doi})")
    if url:
        parts.append(f"<{url}>")
    if eprint:
        prefix = f.get("archiveprefix", "arXiv")
        parts.append(f"{prefix}:{eprint}")
    note = f.get("note", "").strip()
    if note:
        parts.append(f"_{note}._")
    return " ".join(parts)

entries = list(parse_entries(bib_text))
entries.sort(key=lambda e: (short_author(e[2].get("author", "")).lower(),
                            e[2].get("year", "")))

lines = [
    "# Sources",
    "",
    "> Auto-generated from `sources.bib` by `sources.sh` — do not edit directly.",
    "",
    "In-paper citations should link here using the cite key as anchor,",
    "e.g. `[Talisman 2026](sources.gen.md#talisman2026provenance)`.",
    "Per-source reading notes live alongside this file in `sources/<citekey>.md`.",
    "",
]

for key, bibtype, fields in entries:
    # Use the citekey as the heading text so the renderer-generated
    # slug matches the anchor used in citations (#citekey).
    lines.append(f"### {key}")
    lines.append("")
    lines.append(
        f"**[{label(fields)}]** {format_entry(bibtype, fields)} "
        f"([notes](sources/{key}.md))"
    )
    lines.append("")

out_path.write_text("\n".join(lines))
print(f"wrote {out_path} ({len(entries)} entries)")
PY
