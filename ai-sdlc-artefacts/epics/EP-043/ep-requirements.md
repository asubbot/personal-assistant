---
artefact: ep-requirements
epic_id: EP-043
status: draft
source_of_truth: true
updated_at: 2026-05-31
---

# EP-043 — Test suite organization — Requirements (EARS / INCOSE)

> **10 requirements** · 7 FR · 3 NFR

## Introduction

Test-only refactor for maintainability ([ep-scope.md](ep-scope.md)). No product code behaviour changes except moving tests.

## C4 C1 — System Context

<p align="center"><img src="diagrams/c4-context.png" alt="C4 C1 — EP-043" /></p>

**Source:** [diagrams/c4-context.puml](diagrams/c4-context.puml)

## Requirement index

| Id | Type | Summary |
|----|------|---------|
| REQ-43.001 | FR | Split handler_test.go into domain files |
| REQ-43.002 | FR | handler_test.go ≤600 LOC after split |
| REQ-43.003 | FR | Shared handler test helpers file |
| REQ-43.004 | FR | config_test_helpers.go with loadFixture |
| REQ-43.005 | FR | Migrate ≥10 inline JSON blocks to testdata |
| REQ-43.006 | FR | architecture_guards_test.go consolidates traceability |
| REQ-43.007 | FR | Preserve Covers AC comments on moved tests |
| REQ-43.008 | NFR | Coverage drop ≤0.5% |
| REQ-43.009 | NFR | make check passes |
| REQ-43.010 | NFR | make validate passes |

## Requirements

<a id="req-43-001"></a>

### REQ-43.001 — Split handler tests

THE **repository** SHALL move test functions from `internal/core/handler_test.go` into domain-focused files (`handler_session_test.go`, `handler_tools_test.go`, `handler_llm_test.go`, `handler_memory_test.go`) without changing assertions.

<a id="req-43-002"></a>

### REQ-43.002 — handler_test.go size limit

THE **repository** SHALL reduce `handler_test.go` to at most **600** lines after the split (helpers-only or deleted empty shell).

<a id="req-43-003"></a>

### REQ-43.003 — Handler test helpers

THE **repository** SHALL centralize shared handler test constructors and mocks in `handler_test_helpers.go` when used by more than one handler test file.

<a id="req-43-004"></a>

### REQ-43.004 — Config fixture helper

THE **repository** SHALL provide `loadConfigFixture(t *testing.T, name string) *config.Config` (or equivalent) in `internal/config/config_test_helpers.go`.

<a id="req-43-005"></a>

### REQ-43.005 — Migrate inline JSON

THE **repository** SHALL migrate at least **ten** duplicated inline config JSON strings in `config_test.go` to `internal/config/testdata/` files loaded via the fixture helper.

<a id="req-43-006"></a>

### REQ-43.006 — Consolidated guards

THE **repository** SHALL merge epic-specific traceability test files in `internal/core` into **`architecture_guards_test.go`** using table-driven cases where feasible.

<a id="req-43-007"></a>

### REQ-43.007 — AC trace comments

WHEN tests move, THE **repository** SHALL preserve `// Covers AC-xx.xxx` comments on the corresponding test functions for validator traceability.

<a id="req-43-008"></a>

### REQ-43.008 — Coverage floor

THE **repository** total statement coverage reported by `make check` SHALL NOT decrease by more than **0.5** percentage points versus the pre-epic baseline on `main`.

<a id="req-43-009"></a>

### REQ-43.009 — make check

THE **repository** SHALL pass `make check`.

<a id="req-43-010"></a>

### REQ-43.010 — make validate

THE **repository** SHALL pass `make validate` with no new untraced in-scope ACs for EP-039–043 when those epics are registered.
