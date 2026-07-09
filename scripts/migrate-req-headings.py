#!/usr/bin/env python3
"""Migrate ep-requirements.md REQ labels to canonical ### REQ-EE.NNN — Summary headings."""

from __future__ import annotations

import re
import sys
from pathlib import Path

REQ_INDEX_RE = re.compile(r"^\|\s*(REQ-\d{2,3}\.\d{3})\s*\|")
REQ_BOLD_PATTERN_RE = re.compile(r"^\*\*REQ-(\d{2,3}\.\d{3})\*\*\s*\(([^)]+)\)\s*$")
REQ_BOLD_DASH_RE = re.compile(r"^\*\*REQ-(\d{2,3}\.\d{3})\*\*\s*[—–-]\s*(.+?)\s*$")
REQ_ANCHOR_DASH_RE = re.compile(
    r'^<a id="[^"]+"></a>\*\*REQ-(\d{2,3}\.\d{3})\*\*\s*[—–-]\s*(.+?)\s*$'
)
REQ_ANCHOR_PATTERN_RE = re.compile(
    r'^<a id="[^"]+"></a>\*\*REQ-(\d{2,3}\.\d{3})\*\*\s*\(([^)]+)\)\s*$'
)
REQ_H4_RE = re.compile(r"^####\s+(REQ-\d{2,3}\.\d{3}\s*[—–-].*)$")
REQ_TYPO_RE = re.compile(r"REQ-(\d{2,3})-(\d{3})")


def parse_index_summaries(lines: list[str]) -> dict[str, str]:
    summaries: dict[str, str] = {}
    for line in lines:
        m = REQ_INDEX_RE.match(line.strip())
        if not m:
            continue
        parts = [p.strip() for p in line.strip().strip("|").split("|")]
        if len(parts) < 2:
            continue
        req_id = parts[0]
        summary = parts[-1]
        if req_id.startswith("REQ-"):
            summaries[req_id] = summary
    return summaries


def migrate_content(text: str) -> tuple[str, bool]:
    lines = text.splitlines()
    summaries = parse_index_summaries(lines)
    out: list[str] = []
    changed = False

    for line in lines:
        original = line
        line = REQ_TYPO_RE.sub(r"REQ-\1.\2", line)

        m = REQ_H4_RE.match(line)
        if m:
            line = f"### {m.group(1)}"
        elif (m := REQ_ANCHOR_DASH_RE.match(line.strip())):
            line = f"### REQ-{m.group(1)} — {m.group(2)}"
        elif (m := REQ_ANCHOR_PATTERN_RE.match(line.strip())):
            req_id = f"REQ-{m.group(1)}"
            summary = summaries.get(req_id, m.group(2))
            line = f"### {req_id} — {summary}"
        elif (m := REQ_BOLD_DASH_RE.match(line.strip())):
            line = f"### REQ-{m.group(1)} — {m.group(2)}"
        elif (m := REQ_BOLD_PATTERN_RE.match(line.strip())):
            req_id = f"REQ-{m.group(1)}"
            summary = summaries.get(req_id, m.group(2))
            line = f"### {req_id} — {summary}"

        if line != original:
            changed = True
        out.append(line)

    return "\n".join(out) + ("\n" if text.endswith("\n") else ""), changed


def main() -> int:
    root = Path(__file__).resolve().parents[1]
    epics_dir = root / "ai-sdlc-artefacts" / "epics"
    if not epics_dir.is_dir():
        print(f"epics dir not found: {epics_dir}", file=sys.stderr)
        return 1

    migrated = 0
    for epic_dir in sorted(epics_dir.glob("EP-*")):
        req_path = epic_dir / "ep-requirements.md"
        if not req_path.is_file():
            continue
        text = req_path.read_text(encoding="utf-8")
        new_text, changed = migrate_content(text)
        if changed:
            req_path.write_text(new_text, encoding="utf-8")
            print(f"migrated {epic_dir.name}")
            migrated += 1

    print(f"done: {migrated} files updated")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
