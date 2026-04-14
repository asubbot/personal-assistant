# EP-016 — Implementation plan

**Pipeline:** Stage 8 ([pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md)).  
**Previous:** [ep-scope.md](ep-scope.md) · [ep-requirements.md](ep-requirements.md) · [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-system-design.md](ep-system-design.md) · [ep-system-design-review.md](ep-system-design-review.md)  
**Test strategy:** [strategy.md](../../strategy.md)

**AC ownership:** Every **AC-16.001**–**AC-16.021** MUST appear in at least one task verification line or in the validation row below. Mirror the [acceptance criteria index](ep-acceptance-criteria.md#acceptance-criteria-index) when marking tests done.

---

## Checkpoints

- [x] **Checkpoint A:** After vector and memory primitives, `go test ./internal/memory/... ./internal/vector/...` passes.
- [x] **Checkpoint B:** After core wiring, `go test ./internal/core/...` passes.
- [x] **Checkpoint C:** `make build && ./bin/validate EP-016 && make check` ([AC-16.020](ep-acceptance-criteria.md#ac-16-020) deferred to CI; [REQ-16.026](ep-requirements.md#req-16-026)).

---

## Task list

- [x] **1** — **Vector sqlite table names and constructors**  
  - Add `TableSummaries`, `TableTurns`, `TableNotes` constants; keep `TableMemory` (`vec_items`) as legacy.  
  - Extend tests in `internal/vector/sqlite` to open multiple tables on one DB path.  
  - _Requirements:_ [REQ-16.015](ep-requirements.md#req-16-015), [REQ-16.016](ep-requirements.md#req-16-016)  
  - _Acceptance Criteria:_ [AC-16.009](ep-acceptance-criteria.md#ac-16-009) (foundation)  
  - **Verification:** `go test ./internal/vector/sqlite/...`

- [x] **2** — **Configuration: notes byte limits**  
  - Add fields (names as in [ep-system-design.md](ep-system-design.md#data-models)) to config JSON; validate at load; document keys in `docs/configuration.md`.  
  - _Requirements:_ [REQ-16.006](ep-requirements.md#req-16-006)  
  - _Acceptance Criteria:_ [AC-16.004](ep-acceptance-criteria.md#ac-16-004)  
  - **Verification:** `go test ./internal/config/...`

- [x] **3** — **`memory.Store`: append and read `notes.md`**  
  - Implement append with RFC3339 first line, optional `kind=` line, blank-line separation; enforce per-append and file byte caps.  
  - Implement read helper for `read_memory`.  
  - _Requirements:_ [REQ-16.001](ep-requirements.md#req-16-001)–[REQ-16.006](ep-requirements.md#req-16-006)  
  - _Acceptance Criteria:_ [AC-16.001](ep-acceptance-criteria.md#ac-16-001), [AC-16.003](ep-acceptance-criteria.md#ac-16-003), [AC-16.017](ep-acceptance-criteria.md#ac-16-017)  
  - **Verification:** `go test ./internal/memory/...`

- [x] **4** — **`write_memory` native tool**  
  - New tool type; schema `text`, optional `date`, optional `kind`; call memory append then notes vector upsert ([ep-system-design.md](ep-system-design.md#error-handling)).  
  - Register in native registry and wiring; allowlist updates as needed.  
  - _Requirements:_ [REQ-16.007](ep-requirements.md#req-16-007)–[REQ-16.010](ep-requirements.md#req-16-010)  
  - _Acceptance Criteria:_ [AC-16.003](ep-acceptance-criteria.md#ac-16-003), [AC-16.005](ep-acceptance-criteria.md#ac-16-005), [AC-16.016](ep-acceptance-criteria.md#ac-16-016), [AC-16.018](ep-acceptance-criteria.md#ac-16-018)  
  - **Verification:** `go test ./internal/tools/...`

- [x] **5** — **Extend `read_memory` for notes + headings**  
  - Per-day `## YYYY-MM-DD`, `### Automatic summary`, `### Manual notes`; skip empty days.  
  - _Requirements:_ [REQ-16.011](ep-requirements.md#req-16-011)–[REQ-16.014](ep-requirements.md#req-16-014)  
  - _Acceptance Criteria:_ [AC-16.006](ep-acceptance-criteria.md#ac-16-006)–[AC-16.008](ep-acceptance-criteria.md#ac-16-008)  
  - **Verification:** `go test ./internal/tools/...`

- [x] **6** — **`cmd/pa`: open summary, turn, note, and legacy memory stores**  
  - From `vector_index_path`, construct four `vector.Store` handles where applicable; thread into summarize job and core handler dependencies.  
  - _Requirements:_ [REQ-16.015](ep-requirements.md#req-16-018)  
  - _Acceptance Criteria:_ [AC-16.009](ep-acceptance-criteria.md#ac-16-009)  
  - **Verification:** `go test ./cmd/pa/...` (or targeted integration test)

- [x] **7** — **Summarize pipeline: write rollup vectors only to `vec_summaries`**  
  - Change day/month/year upsert paths off `vec_items` to `vec_summaries`.  
  - _Requirements:_ [REQ-16.015](ep-requirements.md#req-16-015), [REQ-16.018](ep-requirements.md#req-16-018)  
  - _Acceptance Criteria:_ [AC-16.009](ep-acceptance-criteria.md#ac-16-009), [AC-16.011](ep-acceptance-criteria.md#ac-16-011)  
  - **Verification:** `go test ./internal/summarize/... ./internal/memoryjob/...`

- [x] **8** — **Core retrieval: split search + merge order**  
  - Implement `gatherRetrievedChunkTexts` (or successor) to query `vec_notes`, `vec_summaries` + filtered legacy `vec_items`, `vec_turns`; merge notes → summary → turn; dedupe ids ([REQ-16.019](ep-requirements.md#req-16-019)).  
  - _Requirements:_ [REQ-16.017](ep-requirements.md#req-16-019)  
  - _Acceptance Criteria:_ [AC-16.010](ep-acceptance-criteria.md#ac-16-010), [AC-16.012](ep-acceptance-criteria.md#ac-16-012)  
  - **Verification:** `go test ./internal/core/...`

- [x] **9** — **`indexTurn`: event-aligned date + stable id + upsert**  
  - Extend `MessageHandler` / handler to accept optional inbound message unix time from Telegram adapter; implement canonicalisation + SHA id + delete-before-add.  
  - _Requirements:_ [REQ-16.020](ep-requirements.md#req-16-023)  
  - _Acceptance Criteria:_ [AC-16.013](ep-acceptance-criteria.md#ac-16-013)–[AC-16.015](ep-acceptance-criteria.md#ac-16-015)  
  - **Verification:** `go test ./internal/core/... ./internal/telegram/...`

- [x] **10** — **Logging / redaction for `write_memory`**  
  - Ensure tool invocation logging uses same redaction path as `read_memory`.  
  - _Requirements:_ [REQ-16.024](ep-requirements.md#req-16-024)  
  - _Acceptance Criteria:_ [AC-16.019](ep-acceptance-criteria.md#ac-16-019)  
  - **Verification:** `go test ./internal/core/...` or `./internal/tools/...`

- [x] **11** — **Summarize job preserves `notes.md`**  
  - Confirm no code path deletes or truncates `notes.md`; add regression test if missing.  
  - _Requirements:_ [REQ-16.002](ep-requirements.md#req-16-002)  
  - _Acceptance Criteria:_ [AC-16.002](ep-acceptance-criteria.md#ac-16-002)  
  - **Verification:** `go test ./internal/summarize/...` or integration

- [x] **12** — **Documentation: REQ-16.027**  
  - Update `docs/configuration.md` (and EP-013 runtime doc pointer if separate) so `write_memory` appears next to `read_memory` where the curated profile lists memory tools.  
  - _Requirements:_ [REQ-16.027](ep-requirements.md#req-16-027)  
  - _Acceptance Criteria:_ [AC-16.021](ep-acceptance-criteria.md#ac-16-021)  
  - **Verification:** Manual review checklist or lightweight test that files contain both tool names in the same section

- [x] **13** — **AC↔test comments and validation**  
  - Add `// Covers AC-16.NNN` to every new test; ensure each AC has coverage.  
  - _Requirements:_ [REQ-16.025](ep-requirements.md#req-16-025), [REQ-16.026](ep-requirements.md#req-16-026)  
  - _Acceptance Criteria:_ [AC-16.020](ep-acceptance-criteria.md#ac-16-020)  
  - **Verification:** `make build && ./bin/validate EP-016 && make check`

---

## Dependencies

- Task **2** before **3**–**5** (limits in tools).  
- Task **1** before **6**–**8**.  
- Task **3** before **4**–**5**.  
- Task **6** before **7**–**10**.  
- Task **9** depends on **6** (turn store wiring).  
- **13** depends on all functional tasks.

---

## Notes

- **Stage 9** executes tasks **in numerical order** unless a dependency forces a wait; mark checkboxes in this file when each task completes (with user approval per [09-task-execution.skill.md](../../../ai-sdlc/specification/skills/09-task-execution.skill.md) for checkbox flips if the team treats the plan as living).  
- Do **not** commit without explicit user allowance ([AGENTS.md](../../../AGENTS.md)).
