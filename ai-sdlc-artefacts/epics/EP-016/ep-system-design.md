# EP-016 — System design

**Pipeline:** Stage 6.  
**Inputs:** [ep-requirements.md](ep-requirements.md), [ep-acceptance-criteria.md](ep-acceptance-criteria.md)

## Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Components and interfaces](#components-and-interfaces)
- [Data models](#data-models)
- [Error handling](#error-handling)
- [Testing strategy](#testing-strategy)
- [Risks and trade-offs](#risks-and-trade-offs)
- [Requirement traceability](#requirement-traceability)

---

## Overview

EP-016 delivers **append-only `notes.md`**, native **`write_memory`**, **`read_memory`** output that combines summaries and notes with distinct headings, and a **three-way split** of memory vectors (`vec_notes`, `vec_summaries`, `vec_turns`) plus a **legacy-safe read path** over `vec_items` for summaries only (prefix filter). Turn indexing uses **event-aligned dates** from the Telegram message `date` when present, and a **documented fallback** to handler receive time in pa_timezone when absent. Turn deduplication uses **stable ids** and **upsert** (`Delete` then `Add`, mirroring summary upserts) so repeated indexing of the same canonical pair does not grow the turn table ([REQ-16.022](ep-requirements.md#req-16-022), [REQ-16.023](ep-requirements.md#req-16-023)).

---

## Architecture

<p align="center"><img src="diagrams/c4-container.png" alt="C4 C2 — Containers" /></p>

**Source:** [diagrams/c4-container.puml](diagrams/c4-container.puml). Regenerate: `plantuml -tpng diagrams/c4-container.puml` from this epic directory.

### Module boundaries

| Layer | Responsibility |
|-------|------------------|
| **internal/memory** | Path helpers and `AppendDayNote`, `ReadDayNotes` (or equivalent), size checks; reuse root and timezone from [memory.Store](ep-requirements.md#req-16-001) |
| **internal/tools** | `write_memory`, extended `read_memory`; parameter validation ([REQ-16.007](ep-requirements.md#req-16-007)–[REQ-16.014](ep-requirements.md#req-16-014)) |
| **internal/vector/sqlite** | Constants for `vec_notes`, `vec_summaries`, `vec_turns`; retain `vec_items` as legacy ([REQ-16.015](ep-requirements.md#req-16-015)–[REQ-16.019](ep-requirements.md#req-16-019)) |
| **internal/summarize** | Day/month/year upsert targets **only** `vec_summaries` ([REQ-16.015](ep-requirements.md#req-16-015)) |
| **internal/core** | `indexTurn` event date, dedup id, `gatherRetrievedChunkTexts` split queries and merge ([REQ-16.017](ep-requirements.md#req-16-017)–[REQ-16.023](ep-requirements.md#req-16-023)) |
| **internal/telegram** | Pass message `date` (unix) into `HandleMessage` / handler context for event alignment ([REQ-16.020](ep-requirements.md#req-16-020)) |
| **cmd/pa** | Open four stores from same `vector_index_path` file: summaries, turns, notes, optional legacy memory table handle for prefix search ([REQ-16.018](ep-requirements.md#req-16-018)) |
| **docs/configuration.md** | New byte limits for notes append; document `write_memory` in native tool section. This file is the operator-facing source that satisfies **REQ-16.027** together with any EP-013 runtime tool listing that references the same native tool ids ([REQ-16.027](ep-requirements.md#req-16-027)) |

---

## Components and interfaces

| Component | Responsibility | Requirements |
|-----------|----------------|--------------|
| **`memory.Store` extensions** | Append/read `notes.md` with byte counters | [REQ-16.001](ep-requirements.md#req-16-001)–[REQ-16.006](ep-requirements.md#req-16-006) |
| **`WriteMemoryTool`** | Schema: `text` (required), `date` optional ISO, `kind` optional enum; calls store then notes indexer | [REQ-16.007](ep-requirements.md#req-16-007)–[REQ-16.010](ep-requirements.md#req-16-010) |
| **`ReadMemoryTool`** | Per day: emit `## YYYY-MM-DD` then `### Automatic summary` block if summary non-empty, then `### Manual notes` if notes non-empty | [REQ-16.011](ep-requirements.md#req-16-011)–[REQ-16.014](ep-requirements.md#req-16-014) |
| **`vector.Store` instances** | `summaryStore`, `turnStore`, `noteStore` on dedicated tables; optional `legacyMemoryStore` = `vec_items` | [REQ-16.015](ep-requirements.md#req-16-015), [REQ-16.016](ep-requirements.md#req-16-016) |
| **`conversationHandler` bundle** | Holds embedder + store references; retrieval merges three searches | [REQ-16.017](ep-requirements.md#req-16-019) |
| **`summarize.Day` (and month/year)** | Replace calls that wrote to `vec_items` with writes to `vec_summaries` | [REQ-16.015](ep-requirements.md#req-16-018) |
| **`indexTurn`** | Build `Date:` from adapter unix → pa_timezone calendar; id `turn:` + `YYYY-MM-DD` + `:` + hex(sha256(canonical)); `Delete`+`Add` | [REQ-16.020](ep-requirements.md#req-16-023) |
| **Notes vector text** | One chunk per append: `Date: …\n[notes]\n` + body lines; id `notes:` + `YYYY-MM-DD` + `:` + append sequence or content hash (design: use monotonic append counter persisted in memory or hash of entry) | [REQ-16.010](ep-requirements.md#req-16-010) |

### Notes entry format (filesystem)

Each append is one block separated by a blank line from the previous block (if any):

1. Line 1: one UTC timestamp in **RFC3339** form (for example `2026-04-14T15:04:05Z`), matching [REQ-16.004](ep-requirements.md#req-16-004) and **AC-16.003** regex checks.
2. Line 2 (optional): `kind=<fact|guideline|preference|other>` when `kind` argument provided ([REQ-16.005](ep-requirements.md#req-16-005)).
3. Remaining lines: free text from `text` argument ([REQ-16.004](ep-requirements.md#req-16-004)).

### Canonicalised pair (turn dedup)

Before hashing: `strings.TrimSpace` on user and assistant UTF-8 text, normalize newlines to `\n`, collapse consecutive internal whitespace to single space (document exact function in code comments). Hash: SHA-256 over `user + "\n---\n" + assistant` UTF-8 bytes. Stable id: `turn:YYYY-MM-DD:<hex>` ([REQ-16.023](ep-requirements.md#req-16-023)).

### Event-aligned date

- **Primary:** `time.Unix(telegramDate, 0).In(paLoc)` → calendar `YYYY-MM-DD` ([REQ-16.020](ep-requirements.md#req-16-020)).
- **Fallback:** when unix timestamp is zero or not supplied, use `time.Now().In(paLoc)` at indexing call (same class of behaviour as pre-epic wall clock, but documented) ([REQ-16.021](ep-requirements.md#req-16-021)).

### Legacy `vec_items` read path

- **Summaries:** `Search` on `legacyMemoryStore` with **post-filter** (in Go) keeping only ids with prefixes `summary:day:`, `summary:month:`, `summary:year:`; cap `topK` after filter ([REQ-16.018](ep-requirements.md#req-16-018)).
- **Turns:** only `turnStore` (`vec_turns`); never merge raw legacy turn rows from `vec_items` ([REQ-16.019](ep-requirements.md#req-16-019)).

---

## Data models

| Entity | Location / id pattern | Notes |
|--------|----------------------|--------|
| Day notes file | `memory_dir/YYYY/MM/DD/notes.md` | Append-only |
| Summary file | `memory_dir/YYYY/MM/DD/summary.md` | Existing |
| Summary vector id | `summary:day:YYYY-MM-DD`, month/year variants | Unchanged semantics, new table |
| Turn vector id | `turn:YYYY-MM-DD:<sha>` | Upsert |
| Notes vector id | `notes:YYYY-MM-DD:<n>` or hash-based | Upsert per append |
| Config keys (example) | `memory_notes_max_append_bytes`, `memory_notes_max_file_bytes` under paths or memory section | Exact names in `docs/configuration.md` |

---

## Error handling

| Failure | Behaviour | REQ |
|---------|-----------|-----|
| Append exceeds limits | Return `fmt.Errorf` from tool with limit name | [REQ-16.006](ep-requirements.md#req-16-006) |
| Path escape | Same validation path as `read_memory` | [REQ-16.009](ep-requirements.md#req-16-009) |
| Vector upsert fails after successful file append | Return error to the tool caller stating vector update failed; the new text remains in `notes.md`; operator may run a future reindex job | [REQ-16.010](ep-requirements.md#req-16-010) |

**Order of operations for `write_memory`:** validate size → append to `notes.md` → embed and upsert notes vector. Tests cover the happy path; the orphan-on-index-failure case is documented only unless time permits an integration test.

---

## Testing strategy

- **Unit:** `memory` notes append/read; `read_memory` / `write_memory` tools; sqlite table isolation; `indexTurn` id and date; merge ordering with mocked stores ([REQ-16.025](ep-requirements.md#req-16-025)).
- **Integration:** `cmd/pa` or `core` handler tests with temp dirs and real sqlite vec tables where feasible.
- **Validation:** `./bin/validate EP-016` after registering AC ids in tests ([AC-16.020](ep-acceptance-criteria.md#ac-16-020)). The implementation plan SHALL mirror the AC index from [ep-acceptance-criteria.md](ep-acceptance-criteria.md) so no AC lacks an owner test or doc reference ([REQ-16.025](ep-requirements.md#req-16-025)).

---

## Risks and trade-offs

| Risk | Mitigation |
|------|------------|
| **Four sqlite vec opens** on one file | sqlite supports multiple connections; use same DSN; verify no lock contention in tests |
| **Legacy mixed `vec_items`** | Prefix filter only on summary path; turns never read legacy turns ([ep-scope.md](ep-scope.md) variant 1) |
| **File indexed but vector failed** | Document reindex path (future memoryjob); fail tool with clear message |
| **RFC3339 in notes** | Human-readable; grep-friendly |
| **Skip-if-exists alternative** for dedup | Out of scope for MVP; upsert chosen for parity with summary writes |

---

## Requirement traceability

| REQ | Design anchor |
|-----|----------------|
| REQ-16.001 | `notes.md` path layout in `memory.Store` |
| REQ-16.002 | Summarize job writes only `summary.md`; no touch of `notes.md` |
| REQ-16.003 | `Store.Location()` for path segments |
| REQ-16.004 | `timestamp=` RFC3339 line in append helper |
| REQ-16.005 | `kind=` optional second line |
| REQ-16.006 | Pre-append byte checks in `WriteMemoryTool` and store |
| REQ-16.007 | Native registry + allowlist in wiring |
| REQ-16.008 | `WriteMemoryTool.Run` |
| REQ-16.009 | `underMemoryRoot` shared checks |
| REQ-16.010 | Notes embed + `noteStore.Add` after append |
| REQ-16.011 | `ReadMemoryTool` summary subsection |
| REQ-16.012 | `### Manual notes` heading |
| REQ-16.013 | Skip empty day branch in loop |
| REQ-16.014 | Existing span and byte loops unchanged |
| REQ-16.015 | `vec_summaries` table + summarize writes |
| REQ-16.016 | `vec_notes` table + `write_memory` indexer |
| REQ-16.017 | `gatherRetrievedChunkTexts` merge |
| REQ-16.018 | Legacy summary search + prefix filter |
| REQ-16.019 | Turn search only `vec_turns` |
| REQ-16.020 | `indexTurn` uses Telegram `date` |
| REQ-16.021 | Fallback `time.Now` in pa_timezone |
| REQ-16.022 | Delete+Add upsert before add |
| REQ-16.023 | Id formula in `indexTurn` |
| REQ-16.024 | Same logging hooks as `read_memory` |
| REQ-16.025 | AC comments + validate |
| REQ-16.026 | CI local `make check` |
| REQ-16.027 | `docs/configuration.md` update |
