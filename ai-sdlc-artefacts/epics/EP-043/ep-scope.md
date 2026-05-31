---
artefact: ep-scope
epic_id: EP-043
status: draft
source_of_truth: true
updated_at: 2026-05-31
git_branch: epic/EP-043-test-suite-organization
---

# Epic scope — EP-043 Test suite organization

| Field | Content |
|-------|---------|
| **ID** | EP-043 |
| **Status** | DONE |
| **Title** | Test suite organization |
| **Description** | Reduce maintainability cost of the test suite by splitting oversized handler tests, introducing shared config test fixtures, and consolidating epic traceability guard tests — without changing product behaviour or reducing meaningful coverage. |
| **First version date** | 2026-05-31 |
| **Git branch** | `epic/EP-043-test-suite-organization` |

## Glossary

- **Test move:** Relocate test functions to new files in the same `package core_test` or `package config_test` package; no assertion changes.
- **Fixture helper:** Shared `loadConfigFixture(t, name)` reading `internal/config/testdata/<name>.json`.
- **Traceability guard:** Tests that assert architectural invariants tied to epic ACs (e.g. EP-034 escalation removal, EP-038 handler file layout).

## Scope (features/capabilities)

- **Prerequisite gate:** Land after **EP-039–042** product refactors are merged so test splits do not conflict with in-flight handler/config edits.
- **Split `handler_test.go` (~1813 LOC):** Move test groups into focused files following existing precedent (`handler_ep017_test.go`, `handler_ep018_coverage_test.go`):
  - `handler_session_test.go` — session window, EP-014
  - `handler_tools_test.go` — tool merge, execution, truncation
  - `handler_llm_test.go` — completion loop, router events
  - `handler_memory_test.go` — retrieval, turn indexing
  - Keep shared test helpers in `handler_test.go` or new `handler_test_helpers.go` (target: main file ≤600 LOC).
- **Config test fixtures (REQ-43.004–006):** Add `internal/config/config_test_helpers.go` with `loadFixture(t, path)` and migrate duplicated inline JSON strings in `config_test.go` where a testdata file already exists or can be extracted (incremental: at least 10 duplicated blocks migrated).
- **Consolidate traceability guards:** Merge `ep034_traceability_test.go` and `ep038_traceability_test.go` into `architecture_guards_test.go` with a table-driven invariant list; preserve `// Covers AC-xx.xxx` comments on moved tests.
- **No coverage regression:** Total statement coverage SHALL NOT drop more than 0.5% versus pre-epic baseline; no deletion of behavioural tests.
- **Verification:** `make check` and `make validate` green.

## Out of scope / deferred

- New product features or test-only behaviour changes.
- Rewriting all inline JSON in `config_test.go` (incremental migration only).
- E2E test reorganization (EP-025 already addressed layout).
- Performance/load test suite.

## Success criteria

- `handler_test.go` ≤600 LOC; new files compile in same package with all tests passing.
- Config fixture helper used by migrated tests; fewer than 802 LOC in `config_test.go` OR ≥10 inline JSON blocks removed.
- Single `architecture_guards_test.go` replaces per-epic traceability files unless validator requires separate files.
- **`make check`** and **`make validate`** pass.

## Execution order

| Order | Epic | Branch |
|-------|------|--------|
| 5 | **EP-043 (this epic)** | `epic/EP-043-test-suite-organization` |

Run last in increment 0.02 continuation sequence so it absorbs test updates from EP-039–042.

## Traceability

- **Strategy:** Refactoring 0.02 test maintainability ([strategy.md](../../strategy.md) §2).
- **Related:** [EP-025](../EP-025/ep-scope.md) (test layout), [EP-038](../EP-038/ep-scope.md) (handler test split precedent).
