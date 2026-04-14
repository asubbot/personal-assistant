# EP-016 — Implementation plan

**Pipeline:** [pipeline.spec.md](../../ai-sdlc/specification/pipeline.spec.md) stage 8  
**Related:** [ep-scope.md](ep-scope.md) · [ep-requirements.md](ep-requirements.md) · [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-system-design.md](ep-system-design.md) · [ep-system-design-review.md](ep-system-design-review.md) (iteration 3 **Pass**) · [strategy.md](../strategy.md)

## Goal

Deliver EP-016 in the Go codebase: `notes.md` I/O, **write_memory**, extended **read_memory**, three vector tables with split-query merge, event-aligned turn indexing with upsert ids, migration INFO log, config keys, tests, and `./bin/validate EP-016`.

## Checkpoints

1. After **Task 1–3**: `go test ./internal/memory/...` passes.  
2. After **Task 6–8**: integration test with temp sqlite + vec0 passes.  
3. Before merge: `make check` and `./bin/validate EP-016` exit 0; stage **10** code review exit per pipeline §2.2.

## Task list

- [ ] **1** — Config schema and validation  
  - Add limits: `write_memory.max_append_bytes`, `write_memory.max_notes_file_bytes`, optional `vector_search_top_k_{notes,summaries,turns}` with defaults derived from current `vector_search_top_k`.  
  - _Requirements:_ [REQ-16.024](ep-requirements.md#req-16-024)  
  - _Acceptance Criteria:_ [AC-16.017](ep-acceptance-criteria.md#ac-16-017)  
  - **Verification:** config load tests fail on invalid bounds; `go test ./internal/config/...`

- [ ] **2** — `internal/memory`: `notes.md` path helpers  
  - `pathForDayNotes`, `AppendDayNote`, `ReadDayNotes` (read full file), reuse `pa_timezone` calendar rules.  
  - _Requirements:_ [REQ-16.001](ep-requirements.md#req-16-001)  
  - _Acceptance Criteria:_ [AC-16.001](ep-acceptance-criteria.md#ac-16-001)  
  - **Verification:** unit tests for path layout and append idempotency of file creation

- [ ] **3** — `internal/vector/sqlite`: table constants  
  - Add `TableSummaries`, `TableTurns`, `TableNotes` (exact names `vec_summaries`, `vec_turns`, `vec_notes` per design).  
  - _Requirements:_ [REQ-16.010](ep-requirements.md#req-16-010)–[REQ-16.012](ep-requirements.md#req-16-012)  
  - _Acceptance Criteria:_ [AC-16.009](ep-acceptance-criteria.md#ac-16-009)  
  - **Verification:** `go test ./internal/vector/sqlite/...`

- [ ] **4** — Wire **vec_summaries** into summarize and memoryjob reindex  
  - Point `summarize.Day/Month/Year` and reindex helpers at summaries store; stop writing summary vectors into mixed `vec_items`.  
  - _Requirements:_ [REQ-16.010](ep-requirements.md#req-16-010), [REQ-16.021](ep-requirements.md#req-16-021)  
  - _Acceptance Criteria:_ [AC-16.009](ep-acceptance-criteria.md#ac-16-009)  
  - **Verification:** updated `internal/summarize` tests; `make check`

- [ ] **5** — Core: open three `Store` handles (or factory) on same `vector_index_path`  
  - `cmd/pa` `setup` and tests construct summaries/turns/notes stores; retain tools/skills stores unchanged.  
  - _Requirements:_ [REQ-16.021](ep-requirements.md#req-16-021)  
  - **Verification:** integration test opens DB and creates all vec0 tables

- [ ] **6** — `indexTurn`: **vec_turns**, event time parameter, SHA-256 id, upsert  
  - Extend `HandleMessage` / adapter contract per [ep-system-design.md](ep-system-design.md).  
  - _Requirements:_ [REQ-16.011](ep-requirements.md#req-16-011), [REQ-16.016](ep-requirements.md#req-16-016)–[REQ-16.019](ep-requirements.md#req-16-019)  
  - _Acceptance Criteria:_ [AC-16.012](ep-acceptance-criteria.md#ac-16-012), [AC-16.013](ep-acceptance-criteria.md#ac-16-013)  
  - **Verification:** handler unit tests with mock clock and mock adapter time

- [ ] **7** — Retrieval: triple `Search`, merge order, legacy `vec_items` summary-only filter  
  - Implement [ep-scope.md](ep-scope.md) variant 1; DEBUG log per [ep-system-design.md](ep-system-design.md) §Observability.  
  - _Requirements:_ [REQ-16.013](ep-requirements.md#req-16-013)–[REQ-16.015](ep-requirements.md#req-16-015), [REQ-16.025](ep-requirements.md#req-16-025)  
  - _Acceptance Criteria:_ [AC-16.010](ep-acceptance-criteria.md#ac-16-010), [AC-16.011](ep-acceptance-criteria.md#ac-16-011), [AC-16.015](ep-acceptance-criteria.md#ac-16-015)  
  - **Verification:** `internal/core` tests with stub vector stores returning fixed result order

- [ ] **8** — Native **write_memory** tool  
  - JSON schema, append format, truncate-on-vec-failure, `vec_notes` upsert.  
  - _Requirements:_ [REQ-16.003](ep-requirements.md#req-16-003)–[REQ-16.007](ep-requirements.md#req-16-007), [REQ-16.023](ep-requirements.md#req-16-023)  
  - _Acceptance Criteria:_ [AC-16.003](ep-acceptance-criteria.md#ac-16-003)–[AC-16.006](ep-acceptance-criteria.md#ac-16-006), [AC-16.016](ep-acceptance-criteria.md#ac-16-016)  
  - **Verification:** tool unit tests + integration with temp dir

- [ ] **9** — Extend **read_memory**  
  - Sections `## Summary` and `## Notes` per [AC-16.007](ep-acceptance-criteria.md#ac-16-007).  
  - _Requirements:_ [REQ-16.008](ep-requirements.md#req-16-008), [REQ-16.009](ep-requirements.md#req-16-009)  
  - _Acceptance Criteria:_ [AC-16.007](ep-acceptance-criteria.md#ac-16-007), [AC-16.008](ep-acceptance-criteria.md#ac-16-008)  
  - **Verification:** `internal/tools` tests

- [ ] **10** — Migration INFO log on startup  
  - _Requirements:_ [REQ-16.020](ep-requirements.md#req-16-020)  
  - _Acceptance Criteria:_ [AC-16.014](ep-acceptance-criteria.md#ac-16-014)  
  - **Verification:** log capture test or integration assert substring

- [ ] **11** — Runtime skills / native allowlist  
  - Register `write_memory` where policy allows (EP-013 alignment).  
  - **Verification:** `internal/runtimeskills` or config testdata updated

- [ ] **12** — `./bin/validate EP-016` and `make check`  
  - Add `Covers AC-16.NNN` comments across new tests.  
  - _Requirements:_ [REQ-16.026](ep-requirements.md#req-16-026)–[REQ-16.028](ep-requirements.md#req-16-028)  
  - _Acceptance Criteria:_ [AC-16.018](ep-acceptance-criteria.md#ac-16-018)  
  - **Verification:** `make build && ./bin/validate EP-016` and `make check`

- [ ] **13** — Pipeline **stage 10** code review (delegated) and fixes if any Blocker/Major/Medium  
  - —  
  - **Verification:** `ep-code-review.md` iteration with zero blocking severities

## Notes

- **Stages 9–10 (code + review)** were **not executed** in the session that produced this plan branch; implementation tasks **1–13** remain unchecked until development proceeds.  
- Optional CLI cleanup of legacy `vec_items` turn rows is **deferred** beyond task 10 unless scope tightens.
