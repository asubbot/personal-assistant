---
artefact: ep-scope
epic_id: EP-035
status: draft
source_of_truth: true
updated_at: 2026-05-31
---

# Epic scope — EP-035 Consolidate small internal packages

| Field | Content |
|-------|---------|
| **ID** | EP-035 |
| **Status** | DONE |
| **Title** | Consolidate small internal packages |
| **Description** | Reduce tiny or empty `internal/` packages as part of Refactoring increment 0.02, without changing product behaviour, explicit JSON configuration, or EP-013/EP-029 security-sensitive prompt and logging contracts. |
| **First version date** | 2026-05-30 |

## Glossary

- **Package consolidation:** Removing stub packages, relocating misplaced tests, or merging tightly coupled libraries into one import path while preserving public API semantics needed by callers.
- **Trust policy:** The static `TrustPolicy` system-prefix string and wrap helpers from EP-013 (`internal/systemprompt` today).
- **Canonical block markers:** The exact `<<<PA_BEGIN_*>>>` / `<<<PA_END_*>>>` line constants and `TextContainsForbiddenMarkerLine` validation from EP-013 (`internal/promptmarkers` today).
- **Concurrent-write reliability test:** `TestConcurrentWrites_NoBusyErrors` (EP-022 / AC-22.010), currently in test-only `internal/reliability`.

## Scope (features/capabilities)

- **Remove `internal/logging`:** Delete the package (only `doc.go`, no production or test imports). LLM logging remains in `internal/llmlog` and `internal/logredact`.
- **Remove `internal/reliability`:** Delete the test-only package; relocate `concurrent_write_test.go` to `tests/integration` (or another existing test package that can import `internal/jobs`, `internal/vector/sqlite`, and `internal/sqlitepragma`). Preserve `-race` coverage for AC-22.010; update `docs/configuration.md` if it references the old path.
- **Merge `internal/promptmarkers` + `internal/systemprompt` → `internal/prompt`:** Single package holding marker constants, forbidden-line checks, `TrustPolicy`, and wrap helpers. Update all current importers: `internal/core` (`handler.go`, `system_tail.go`, tests), `internal/tools/write_memory.go`, `internal/runtimeskills`, `tests/integration/runtime_skills_handler_test.go`, `tests/integration/runtime_skills_config_test.go`, and merged package tests.
- **Security preservation (mandatory):** `TrustPolicy` text and every canonical marker line constant (`<<<PA_BEGIN_CONTEXT>>>`, etc.) MUST remain byte-identical; `TextContainsForbiddenMarkerLine` behaviour MUST remain equivalent. Existing EP-013 tests MUST pass or move unchanged in intent (marker collision, wrap layout, integration handler checks).
- **No `config.json` changes:** Do not add, remove, or alter configuration keys or validation for this epic.
- **Verification:** `make check` passes; no dead import paths to removed packages.

## Out of scope

- **`internal/lifecyclelog` (EP-029):** Deferred. It is ~38 LOC, imported only from `cmd/pa/jobs_runtime.go` and `internal/memoryjob`; folding it into either caller would couple unrelated subsystems or duplicate lifecycle attribute names (`lifecycle_event`, `subsystem`, `lifecycle_phase`, `duration_ms`). Keeping the package preserves the EP-029 structured lifecycle log contract without scope creep.
- Broader refactors (e.g. merging `llmlog`/`logredact`, reshaping `internal/core`, or other small packages from the architecture inventory).
- Behavioural changes to prompt assembly order, trust rules, redaction, or SQLite PRAGMA policy.

## Success criteria

- `internal/logging` and `internal/reliability` directories are gone; no Go import references remain.
- `internal/promptmarkers` and `internal/systemprompt` are gone; their behaviour lives under `internal/prompt` with byte-identical trust and marker constants (verified by existing or moved unit/integration tests).
- Relocated concurrent-write test still passes under `go test -race` at its new location.
- No edits to product `config.json` schema, examples, or load validation.
- `make check` passes.
- No functional regression in runtime skills loading, memory indexing marker rejection, or core system-message assembly.

## Traceability

- **Scope:** Supports evolving the **Core** without radical redesign by reducing package surface area ([scope.md](../../scope.md)).
- **Strategy:** Maps to **Refactoring 0.02** — remove extra architecture complexity ([strategy.md](../../strategy.md)).
- **Related epics:** Preserves EP-013 prompt/marker contracts; preserves EP-022 concurrent-write test intent; does not alter EP-029 lifecycle logging (deferred).
