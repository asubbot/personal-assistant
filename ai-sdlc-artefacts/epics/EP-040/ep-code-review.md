---
artefact: ep-code-review
epic_id: EP-040
status: draft
source_of_truth: true
gate: pass
latest_iteration: 1
open_counts:
  blocker: 0
  major: 0
  medium: 0
  minor: 0
non_blocking_counts:
  nit: 1
  suggestion: 1
next_action: proceed_to_stage_11
updated_at: 2026-05-31
---

# Code review — EP-040 Handler dependency grouping

---

## Current Gate Summary

Gate: Pass
Latest iteration: 1
Last updated: 2026-05-31
Open counts: Blocker 0 | Major 0 | Medium 0 | Minor 0
Non-blocking counts: Nit 1 | Suggestion 1
Open findings: (none)
Next action: Proceed to stage 11

---

## Review iteration 1

**Review date:** 2026-05-31
**Stage 10 iteration:** 1 of max 5
**Scope:** Branch `epic/EP-040-handler-dependency-grouping` vs `main` (30 files, +621/−343 lines). Product scope: structural grouping of `conversationHandler` dependencies into four unexported sub-structs; mechanical field-access migration; constructor and test-helper updates. No runtime behaviour, config, or public API changes intended.
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0 | Nit: 1 | Suggestion: 1
**Gate:** Pass

### Summary

The EP-040 implementation is a clean, mechanical structural refactor that fully aligns with the approved system design, scope, and implementation plan. All five plan tasks are complete: four dependency sub-structs defined on `conversationHandler`, grouped access applied across handler files and related helpers, both production constructors (`newRunConversationHandler`, `NewIntegrationConversationHandler`) build four struct literals, and tests updated via a centralized `testHandlerDeps` helper. `make check` passes cleanly. No config or public API changes detected. Recommend merge to stage 11.

### What was done well

- **Struct layout:** `handlerToolDeps`, `handlerMemoryDeps`, `handlerSessionDeps`, and `handlerLLMDeps` in `handler.go` contain all fields specified in REQ-40.001–004; `toolResultPromptBytes` correctly remains top-level per EP-039/design.
- **Mechanical migration:** Flat `h.field` access eliminated from `handler*.go`, `dynamic_tool_selection.go`, and `system_tail.go`; grep confirms only stale comment references to old flat names remain (non-blocking).
- **Constructors:** `newRunConversationHandler` and `NewIntegrationConversationHandler` assign one struct literal per group; post-construction `h.tools.catalog = cfg.ToolCatalog` preserves prior ordering semantics.
- **Test ergonomics:** `handler_test_deps_test.go` introduces `testHandlerDeps` with a `.handler()` builder — reduces duplication across ~160 lines of test updates and mirrors production grouping.
- **Traceability:** `ep040_traceability_test.go` asserts exactly five fields on `conversationHandler` with correct types (`// Covers AC-40.001`).
- **Scope guards:** No `config.json` / `internal/config` changes in the diff; `BuildMessageHandler`, `Run`, and integration export signatures unchanged.

### Findings

| ID | Severity | Location | Issue | Recommendation |
|----|----------|----------|-------|----------------|
| F-001 | **Nit** | `internal/core/handler.go:21`, `internal/core/handler_test.go:441,458` | Comments still reference pre-EP-040 flat paths (`handler.maxDynamicSystemRunes`, `h.model`) while code uses grouped access (`h.llm.maxDynamicSystemRunes`, `h.llm.model`). | Optional: update comments to grouped paths for consistency with AC-40.002 intent. |
| F-002 | **Suggestion** | AC-40.002; design Testing strategy | AC-40.002 (no flat field access outside group definitions) is verified manually via grep in this review; no automated regression guard. Design marks traceability test as optional. | Optional follow-up: add a lightweight source-scan test (similar to EP-034 traceability) forbidding `h.(catalog\|router\|memVec\|…)` patterns in `handler*.go`. Not required for merge. |

### Plan alignment

| Area | Plan / REQ | Status |
|------|------------|--------|
| Task 1.1 — Define four sub-structs | Tasks 1.1, REQ-40.001–004 | ✅ Complete |
| Task 1.2 — Mechanical field access | Task 1.2, REQ-40.005, AC-40.002 | ✅ Complete |
| Task 1.3 — Refactor `newRunConversationHandler` | Task 1.3, REQ-40.006, AC-40.003 | ✅ Complete |
| Task 1.4 — Test helpers + core tests | Task 1.4, REQ-40.007–008, AC-40.004–005 | ✅ Complete |
| Task 1.5 — make check + traceability test | Task 1.5, REQ-40.010, AC-40.007 | ✅ Complete |
| Scope: no config schema change | REQ-40.009, AC-40.006 | ✅ Verified (no config files in diff) |
| Scope: public API unchanged | REQ-40.007, AC-40.004 | ✅ Verified |

### Test / verification

- `make check` — **PASS** (exit 0, 2026-05-31). fmt, vet, vuln, golangci-lint (0 issues), race-enabled unit tests (including `pa/internal/core`), integration tests, module boundaries all green.
- Traceability: `TestEP040_conversationHandlerGroupedDeps` — struct layout matches design (5 fields: four groups + `toolResultPromptBytes`).
- Test parity: `handler_test.go` and epic-specific handler tests show struct-literal / helper-path updates only; no changed assertion strings, tool ids, or message shapes observed in diff review.
- Constructor coverage: exactly three direct `&conversationHandler{…}` sites — `run.go`, `integration_export.go`, `handler_test_deps_test.go`; all use grouped literals.

### Residual risks / follow-ups

- Structural refactor risk (missed field reference) mitigated by compiler, full test suite, and green `make check`; no residual blocking risk identified.
- Non-blocking: F-001 (comment drift), F-002 (optional flat-access guard test).
