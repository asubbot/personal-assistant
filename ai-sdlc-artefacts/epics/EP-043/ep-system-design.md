---
artefact: ep-system-design
epic_id: EP-043
updated_at: 2026-05-31
---

# EP-043 — Test suite organization — System design

## Overview

Test-only refactor ([ep-scope.md](ep-scope.md)): split `handler_test.go` (≤600 LOC), add `config_test_helpers.go`, merge traceability guards into `architecture_guards_test.go`.

## File split map

| New file | Tests moved (themes) |
|----------|---------------------|
| `handler_session_test.go` | session window, EP-014 |
| `handler_tools_test.go` | tool merge, execution, truncation |
| `handler_llm_test.go` | completion loop, router |
| `handler_memory_test.go` | retrieval, indexing |
| `handler_test_helpers.go` | shared mocks (extend `handler_test_deps_test.go` if needed) |

## Config helpers

`loadConfigFixture(t, name string) (*Config, string path)` reading `testdata/<name>.json`.

Migrate ≥10 inline JSON blocks from `config_test.go`.

## architecture_guards_test.go

Merge invariants from `ep034_traceability_test.go`, `ep038_traceability_test.go`, `ep040_traceability_test.go` (table-driven); delete merged files after.

## REQ traceability

REQ-43.001–010 via file moves + make check coverage floor.
