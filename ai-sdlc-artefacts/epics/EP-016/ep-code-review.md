# Code review — EP-016

---

## Review iteration 1

**Review date:** 2026-04-14
**Stage 10 iteration:** 1 of max 5
**Scope:** Branch `epic/EP-016-memory-notes` vs `main` (product + tests): `cmd/pa/main.go`; `internal/core/` (handler.go, run.go, memory_vectors.go, vector_merge.go, telegram_context.go, integration_export.go, EP-016 tests); `internal/memory/store.go`, `internal/memory/ep016_notes_test.go`; `internal/tools/read_memory.go`, `internal/tools/write_memory.go`, `internal/tools/ep016_memory_tools_test.go`; `internal/config/` (config.go, load.go, runtime_skills.go, docs_ep016_test.go); `internal/telegram/adapter.go`; `internal/summarize/labels.go`; `internal/vector/sqlite/`; `tests/integration/memory_vector_test.go`. Cross-read only: `ai-sdlc-artefacts/epics/EP-016/ep-requirements.md`, `ep-system-design.md`.
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 4 | Nit: 2 | Suggestion: 1
**Gate:** Pass

### Summary

The EP-016 implementation matches the epic’s intent: append-only `notes.md` with RFC3339 UTC and optional `kind=`, native `write_memory` with `memory_dir` containment checks and clear errors when vector indexing fails after a successful disk append, extended `read_memory` with distinct `### Automatic summary` / `### Manual notes` sections and skip-empty-days behaviour, split sqlite-vec tables with merge order notes then summaries then turns, legacy summary reads filtered by `summary:*` id prefixes, Telegram `msg.Date` passed via context for event-aligned turn indexing, and delete-then-add upsert for turn deduplication. Tool invocation INFO logs use `redactLogString` on arguments and results like other tools. `make check` passed on the reviewed branch. Recommend **merge**; remaining items are polish and resilience trade-offs, not gate issues.

### Findings

| Severity | Location | Issue | Recommendation |
|----------|----------|-------|----------------|
| Minor | `internal/tools/read_memory.go` (`ReadMemoryTool.Description`) | Description still refers only to “day summaries”, not combined summary + notes output. | Update description to state that both automatic summaries and manual notes are returned for the requested calendar days. |
| Minor | `internal/core/handler.go` (`gatherSplitTableChunks`) | If `Notes.Search` returns an error, the function returns immediately with no chunks, so summary and turn retrieval are skipped as well. | Consider degraded mode (continue with summaries/turns) or document intentional fail-fast; add a test if the product decision is fixed. |
| Minor | `internal/core/handler.go` (`indexTurn`, stable id) | Design text describes a full SHA-256 hex digest; code uses a truncated slice (`sum[:12]` with `%x`). | Align code and design: either use full digest in the id or document the truncated form as the canonical id. |
| Minor | `internal/summarize/labels.go` (`VectorChunkLabel`) | Unknown id prefixes fall through to label `"turn"`, which may mis-tag anomalous rows. | Use a neutral default or explicit handling for unknown prefixes if such rows are possible. |
| Nit | `internal/tools/read_memory.go` (`underMemoryRoot`) | On `filepath.Abs` / `filepath.Rel` failure, returns `false`, surfacing as “outside memory_dir” rather than a distinct filesystem error. | Optionally return or log a clearer error path for Abs/Rel failures. |
| Nit | `cmd/pa/main.go` (`setup` / `openMemoryVectorBundle`) | All four memory vector tables are opened on every successful embedder setup. | Acceptable for EP-016; revisit only if startup or locking cost becomes measurable. |
| Suggestion | Epic design / tests | Optional integration test for “vector upsert fails after successful notes append” is called out in design as optional. | Add later if operators need automated assurance of the documented orphan case. |

### Test / verification

Ran `make check` on workspace branch `epic/EP-016-memory-notes` — **passed** (includes `go test -race -tags=integration ./...`).

### Residual risks

Four SQLite connections on one vector DB file may warrant monitoring under concurrent load (design already notes this). Operational recovery for a note present on disk but missing from `vec_notes` after a failed embed remains a manual / future reindex concern, as documented in the epic design.
