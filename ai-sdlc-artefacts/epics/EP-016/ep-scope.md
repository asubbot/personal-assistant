# Epic scope — EP-016 Manual day notes, write_memory, and vector memory refinement

| Field | Content |
|-------|---------|
| **ID** | EP-016 |
| **Status** | DONE |
| **Title** | Manual day notes, write_memory, and vector memory refinement |
| **Description** | Introduce operator- and model-addressable **daily notes** on disk (`notes.md` per calendar day under `memory_dir`), a native **`write_memory`** tool that appends validated content to `notes.md` and indexes it for semantic retrieval, extend **`read_memory`** so date-range reads include both automatic **`summary.md`** and **`notes.md`**, and refine **vector memory**: separate **conversation-turn** embeddings from **rollup summary** embeddings, use **event-aligned calendar dates** in turn chunks (not “time of indexing”), and apply **deduplication** policy for turn indexing to limit redundant rows. |
| **First version date** | 2026-04-14 |

## Glossary

Terms from the project [scope.md](../../scope.md) glossary apply. Epic-specific terms:

| Term | Definition |
|------|------------|
| **Day summary** | Existing artefact: `memory_dir/YYYY/MM/DD/summary.md`, produced and **overwritten** by the automatic summarization pipeline (EP-002). Not written by `write_memory`. |
| **Day notes** | Operator- or model-authored markdown appendices for one calendar day: `memory_dir/YYYY/MM/DD/notes.md`. **Append-only** at the file level (new entries are appended; the file is never replaced by automatic summarization). Created on first `write_memory` (or equivalent) for that day. |
| **write_memory** | Native tool: validates arguments and calendar date in **pa_timezone**, appends a bounded text entry to the correct `notes.md`, then updates the **notes** vector index slice so retrieval can surface the entry without relying on grep. |
| **Turn chunk** | One vector document representing a single completed user message and the assistant’s **final** reply text for that handler invocation (current behaviour extended for date and id semantics per this epic). |
| **Summary chunk** | Vector document for `summary:day`, `summary:month`, or `summary:year` (stable id prefixes, upsert semantics) as today. |
| **Notes chunk** | Vector document whose text is derived from a **single** logical append to `notes.md` (or a row-level slice defined in design), with a stable id scheme and type label for retrieval (e.g. `[notes]` or equivalent) so the model can distinguish notes from turns and rollups. |
| **Event-aligned date** | The calendar date (in **pa_timezone**) associated with the **user message** (or the adapter-supplied message timestamp), used in the `Date:` line of a **turn** chunk—not necessarily the wall-clock instant when indexing runs. |

## Scope (features/capabilities)

### A — Day notes file (`notes.md`)

- **Path:** For a calendar day *D* in **pa_timezone**, the notes file path is **`memory_dir/YYYY/MM/DD/notes.md`** (same directory depth as `summary.md`, different filename).
- **Lifecycle:** The summarization job **must not** delete or overwrite `notes.md` when (re)writing `summary.md`. Day/month/year rollup LLM inputs continue to use day summaries as today unless a follow-up epic explicitly includes notes in rollups.
- **Format (minimum):** Append-only lines or blocks; each append **must** include a **UTC ISO-8601 timestamp** (or a format fixed in requirements) and optional **kind** (`fact`, `guideline`, `preference`, or free tag) so future filtering is possible. Exact template is for the requirements/design stage.
- **Size limits:** Configurable max bytes per append and/or per file; fail fast with a clear tool error when exceeded.

### B — Native `write_memory` tool

- **Arguments (conceptual):** At minimum: `text` (required), optional `date` (ISO `YYYY-MM-DD`, default “today” in **pa_timezone**), optional `kind` / tag enum.
- **Effects:** (1) Resolve target day and path under `memory_dir` only (path traversal rejected; same prefix checks as `read_memory`). (2) Append the formatted entry to `notes.md` (create parent dirs if needed). (3) Update vector index for **notes** per design (embed + add/replace row).
- **Security:** Same sensitivity class as memory and tools: no arbitrary paths, no secrets in examples, redaction-aware logging where tool I/O is logged.
- **Registration:** Native registry + allowlists / runtime skill docs updated so curated deployments can expose the tool where appropriate (exact EP-013 linkage in later artefacts).

### C — Extend `read_memory`

- For each day in the requested ISO `date` or `from`/`to` range, the tool returns content from **`summary.md`** when present **and** from **`notes.md`** when present, in a single per-day block with **unambiguous section headings** (exact strings in requirements). If neither file exists for a day, that day contributes nothing (same as today for missing summaries).
- Preserve existing limits: `max_span_days`, `max_output_bytes`, **pa_timezone** noon anchoring for iteration, and path-under-root checks.

### D — Vector store split (turns vs summaries)

- **Problem addressed:** Today, automatic **day/month/year summaries** and **per-turn** conversation embeddings share one **`vec_items`** (or equivalent) namespace, which complicates prioritisation, TTL, and growth control.
- **Requirement:** **Rollup summary** vectors (day/month/year) **must** live in a **separate** sqlite-vec virtual table (or equivalently isolated store) from **turn** vectors. Existing table naming in code (`vec_items`, etc.) is an implementation detail; the epic requires **logical separation** and **documented merge order** when assembling “Relevant past context”.
- **Merge order (default policy):** When building retrieved chunks for the LLM, **notes** matches (if any) **must** rank **before** **summary:*** chunks, which **must** rank **before** **turn** chunks, unless design documents a configurable order. Within each class, keep similarity order from search.
- **Migration:** One-time or operator-triggered strategy (clear turns table, reindex from logs optional) is **in scope** at epic level as a decision for design; silent data loss without operator awareness is **out of scope**. A **supported rollout** is: **legacy vectors remain on disk**, **new** writes use the **new** tables only, while retrieval follows **variant 1 (split queries)** below so behaviour stays correct without an immediate bulk delete.

#### Variant 1 — Split queries (preferred rollout for mixed legacy)

During and after rollout, **semantic retrieval must not rely on a single undifferentiated vector search** over a table that still contains **legacy turn rows** alongside summaries.

- **Notes:** Run **vector search only** against the **notes** table (e.g. `vec_notes`). No notes rows live in legacy mixed storage.
- **Rollup summaries (day / month / year):** Run **vector search only** against the **summary** table, **or** against legacy storage **restricted** to rows whose **id** matches the stable **`summary:*`** prefixes (so legacy **turn** rows in the same physical table, if any, are **never** candidates).
- **Turns:** Run **vector search only** against the **turn** table (e.g. `vec_turns`). **Legacy turn rows** that remain in an old shared table **must not** be returned by the summary-only query and **must not** be merged again from a second path (avoids duplicates and “stale + new” turn pairs for the same exchange).
- **Merge:** Combine the three result lists per **merge order** (notes → summaries → turns), each list ordered by **similarity within that search**, then apply the existing **dynamic system tail** rune budget.

This variant allows **old rows to stay on disk** until an operator opts into cleanup or reindex, without polluting RAG, provided the **read path** implements the split as above.

### E — Event-aligned `Date:` for turn chunks

- Turn chunk text **must** use **event-aligned date** for the `Date: YYYY-MM-DD` line (see glossary). The core **must** obtain a timestamp from the inbound message path (e.g. Telegram message `date`) or a defined fallback documented when the adapter does not supply one.
- **Tests:** Cover adapter present vs missing clock skew edge cases as far as unit/integration harness allows.

### F — Deduplication for turn indexing

- **Goal:** Avoid unbounded duplicate vectors when the same user+assistant text (or normalised form) is indexed repeatedly.
- **Minimum acceptable policy:** Stable **id** derived from **event-aligned date** plus a **hash** of a **canonicalised** `(user_text, assistant_text)` pair; **upsert** (`delete` by id then `add`, mirroring summary behaviour) **or** skip-if-exists per documented semantics.
- **Optional stretch (design may defer):** Near-duplicate detection via embedding distance; if deferred, record as follow-up epic.

## Success criteria

- Appending via **`write_memory`** creates or updates only **`notes.md`** for the target day; **`summary.md`** content for that day is unchanged by the tool.
- **`read_memory`** for a range returns **both** summary and notes sections when both files exist; output stays within configured byte limits.
- Vector retrieval uses **separate** storage for **turns** vs **rollup summaries**, and includes **notes** vectors in the merged result with the agreed **priority order**.
- Turn chunks exposed to the model carry **`Date:`** consistent with **pa_timezone** calendar rules and the **event-aligned** definition.
- Re-indexing the **same** turn content under the dedup policy does **not** grow the turn table without bound (verified by test or by row-count assertion in integration).
- **`make check`** passes for the delivered implementation.

## Out of scope / deferred

- Changing automatic summarization prompts to ingest **`notes.md`** into day/month/year rollup inputs (possible follow-up epic).
- Multi-user isolation of `notes.md` per Telegram user (unless the product already defines per-user memory layout—default remains single shared `memory_dir` per EP-002).
- UI outside Telegram for editing notes.
- Cross-device real-time sync of `notes.md` beyond normal filesystem semantics of the deployment.
- Replacing **`read_memory`** with natural-language-only date resolution in core (remains skill-driven per EP-002 / EP-013).

## Traceability

- **Scope:** Extends [scope.md](../../scope.md) **Long-term memory** (markdown calendar layout + vector index) with an explicit **manual** write path and clearer retrieval composition.
- **Strategy:** Aligns with [strategy.md](../../strategy.md): incremental, testable slices (tool + file layout first, then vector split and merge policy, then turn date/dedup).
