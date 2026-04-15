# EP-018 — Implementation plan

**Pipeline:** Stage 8 ([pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md)).  
**Previous:** [ep-scope.md](ep-scope.md) · [ep-requirements.md](ep-requirements.md) · [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-system-design.md](ep-system-design.md) · [ep-system-design-review.md](ep-system-design-review.md)  
**Test strategy:** [strategy.md](../../strategy.md)

**AC ownership:** Every **AC-18.001**–**AC-18.021** MUST appear in at least one task verification line or in the validation row below.

---

## Checkpoints

- [x] **Checkpoint A:** After `internal/intent` changes, `go test ./internal/intent/...` passes.
- [x] **Checkpoint B:** After config types + validation, `go test ./internal/config/...` passes.
- [x] **Checkpoint C:** After core handler + picker wiring, `go test ./internal/core/...` passes for new EP-018 tests.
- [x] **Checkpoint D:** `make build && ./bin/validate EP-018 && make check` ([AC-18.021](ep-acceptance-criteria.md#ac-18-021)).

---

## Task list

- [x] **1** — **`internal/intent`: `TierFullLite` and three-way classification**
  - Add `TierFullLite` constant (`full_lite`); extend `ModelClassifier` to parse `full_lite` and to build the classification prompt body per design (preamble + three tier bullets + delimited user message only).
  - Extend `HeuristicClassifier` with configurable `fullLitePatterns` and ordering: after `full` patterns, match `full_lite` before ambiguous.
  - Update `CascadeClassifier` / `Result` if needed so deciding stage strings remain consistent for logs.
  - _Requirements:_ [REQ-18.009](ep-requirements.md#req-18-009), [REQ-18.010](ep-requirements.md#req-18-010), [REQ-18.011](ep-requirements.md#req-18-011)
  - _Acceptance Criteria:_ [AC-18.009](ep-acceptance-criteria.md#ac-18-009), [AC-18.010](ep-acceptance-criteria.md#ac-18-010), [AC-18.011](ep-acceptance-criteria.md#ac-18-011)
  - **Verification:** `go test ./internal/intent/...`

- [x] **2** — **`internal/config`: EP-018 structs and validation**
  - Add `full_lite_patterns` to heuristic config; add `tools.dynamic_selection` (or agreed key) with `enabled_for_full_lite`, `enabled_for_full`, `max_tools_for_llm_request`.
  - Fail-fast: compile regexes; require `max_tools_for_llm_request` ≥ 1 when any dynamic flag true; require `max_tools_for_llm_request` ≥ count of distinct valid `always_include` tool IDs in catalog when dynamic enabled; reject `enabled_for_full_lite` when text-based tools globally disabled (per design decision).
  - _Requirements:_ [REQ-18.019](ep-requirements.md#req-18-019)
  - _Acceptance Criteria:_ [AC-18.019](ep-acceptance-criteria.md#ac-18-019)
  - **Verification:** `go test ./internal/config/...`

> **Checkpoint A:** `go test ./internal/intent/...`  
> **Checkpoint B:** `go test ./internal/config/...`

- [x] **3** — **`internal/core`: `pickToolsForMainRequest` (name as implemented)**
  - Implement merge of `always_include` with ranked IDs preserving order rules from [ep-system-design.md](ep-system-design.md); enforce cap; handle pre-selection off via fallback list input ([REQ-18.016](ep-requirements.md#req-18-016)).
  - _Requirements:_ [REQ-18.012](ep-requirements.md#req-18-012), [REQ-18.013](ep-requirements.md#req-18-013), [REQ-18.014](ep-requirements.md#req-18-014), [REQ-18.016](ep-requirements.md#req-18-016)
  - _Acceptance Criteria:_ [AC-18.012](ep-acceptance-criteria.md#ac-18-012), [AC-18.013](ep-acceptance-criteria.md#ac-18-013), [AC-18.014](ep-acceptance-criteria.md#ac-18-014), [AC-18.016](ep-acceptance-criteria.md#ac-18-016)
  - **Verification:** `go test ./internal/core/...` (or dedicated package test file for picker)

- [x] **4** — **`HandleMessage`: `full_lite` prompt assembly**
  - Gate RAG (`gatherRetrievedChunkTexts`) off for `full_lite`; omit retrieved chunks from system message; omit runtime skills tail; keep session snapshot like `full`; Hermes only when final tool count non-zero ([REQ-18.004](ep-requirements.md#req-18-004)–[REQ-18.008](ep-requirements.md#req-18-008)).
  - _Requirements:_ [REQ-18.004](ep-requirements.md#req-18-004)–[REQ-18.008](ep-requirements.md#req-18-008)
  - _Acceptance Criteria:_ [AC-18.004](ep-acceptance-criteria.md#ac-18-004)–[AC-18.008](ep-acceptance-criteria.md#ac-18-008)
  - **Verification:** `go test ./internal/core/...`

- [x] **5** — **`HandleMessage`: `full` baseline and optional dynamic selection**
  - When `enabled_for_full` is false, preserve byte-identical / structurally identical EP-017 `full` path ([REQ-18.003](ep-requirements.md#req-18-003), [REQ-18.015](ep-requirements.md#req-18-015)).
  - When `enabled_for_full` is true, run picker after existing ranking ([REQ-18.012](ep-requirements.md#req-18-012)–[REQ-18.014](ep-requirements.md#req-18-014)).
  - _Requirements:_ [REQ-18.003](ep-requirements.md#req-18-003), [REQ-18.015](ep-requirements.md#req-18-015)
  - _Acceptance Criteria:_ [AC-18.003](ep-acceptance-criteria.md#ac-18-003), [AC-18.015](ep-acceptance-criteria.md#ac-18-015)
  - **Verification:** `go test ./internal/core/...`

- [x] **6** — **`HandleMessage`: `full_lite` + tools enabled uses dynamic picker**
  - Wire picker for `full_lite` when text-based tools enabled ([REQ-18.017](ep-requirements.md#req-18-017)).
  - _Requirements:_ [REQ-18.017](ep-requirements.md#req-18-017)
  - _Acceptance Criteria:_ [AC-18.017](ep-acceptance-criteria.md#ac-18-017)
  - **Verification:** `go test ./internal/core/...`

- [x] **7** — **`simple` tier regression and documentation matrix**
  - Confirm `simple` path unchanged vs EP-017 ([REQ-18.002](ep-requirements.md#req-18-002)); update tier matrix in `docs/configuration.md` ([REQ-18.001](ep-requirements.md#req-18-001)).
  - _Requirements:_ [REQ-18.001](ep-requirements.md#req-18-001), [REQ-18.002](ep-requirements.md#req-18-002)
  - _Acceptance Criteria:_ [AC-18.001](ep-acceptance-criteria.md#ac-18-001), [AC-18.002](ep-acceptance-criteria.md#ac-18-002)
  - **Verification:** doc review + targeted unit test if needed

- [x] **8** — **Observability**
  - INFO log after main prompt assembly: tier, main tool count, `dynamic_tool_selection` bool, classifier stage ([REQ-18.018](ep-requirements.md#req-18-018)).
  - _Requirements:_ [REQ-18.018](ep-requirements.md#req-18-018)
  - _Acceptance Criteria:_ [AC-18.018](ep-acceptance-criteria.md#ac-18-018)
  - **Verification:** `go test ./internal/core/...` with captured logger

- [x] **9** — **Token / rune reduction regression ([AC-18.020](ep-acceptance-criteria.md#ac-18-020))**
  - Automated test comparing `full` versus `full_lite` prompt rune counts on a fixture with ≥ 15% reduction constant.
  - _Requirements:_ [REQ-18.004](ep-requirements.md#req-18-004), [REQ-18.006](ep-requirements.md#req-18-006), [REQ-18.013](ep-requirements.md#req-18-013)
  - _Acceptance Criteria:_ [AC-18.020](ep-acceptance-criteria.md#ac-18-020)
  - **Verification:** `go test ./internal/core/...`

> **Checkpoint C:** `go test ./internal/core/...` passes including new EP-018 tests.

- [x] **10** — **`cmd/pa` wiring**
  - Pass updated config into classifier and handler construction paths.
  - _Requirements:_ [REQ-18.009](ep-requirements.md#req-18-009)–[REQ-18.011](ep-requirements.md#req-18-011) (wiring)
  - _Acceptance Criteria:_ —
  - **Verification:** `go build ./cmd/pa/...`

- [x] **11** — **AC coverage comments and validate tool**
  - Add `// Covers AC-18.NNN` to new tests; register epic in validate tool if required by repo convention; run `./bin/validate EP-018`.
  - _Requirements:_ [REQ-18.021](ep-requirements.md#req-18-021)
  - _Acceptance Criteria:_ [AC-18.021](ep-acceptance-criteria.md#ac-18-021)
  - **Verification:** `./bin/validate EP-018` exit 0

- [x] **12** — **Final CI**
  - _Requirements:_ [REQ-18.020](ep-requirements.md#req-18-020)
  - _Acceptance Criteria:_ [AC-18.021](ep-acceptance-criteria.md#ac-18-021)
  - **Verification:** `make check` exit 0

> **Checkpoint D:** `make build && ./bin/validate EP-018 && make check`

---

## Dependencies

- Task **1** before **4**–**6** (tier values in handler).
- Task **2** before **3**–**7**, **10** (config surface).
- Task **3** before **5**, **6** (picker used by branches).
- Tasks **4**–**6** depend on **1** and **3**.
- Task **11** depends on all functional tasks; **12** depends on **11**.

---

## Notes

- Stage **9** executes tasks in numerical order unless dependencies block; mark checkboxes when each task completes.
- Do **not** commit without explicit user allowance ([AGENTS.md](../../../AGENTS.md)).
- **Stage 9 (implementation)** is out of scope for this document until the owner starts execution on a branch.
