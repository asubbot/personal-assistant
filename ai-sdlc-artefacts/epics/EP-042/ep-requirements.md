---
artefact: ep-requirements
epic_id: EP-042
status: draft
source_of_truth: true
updated_at: 2026-05-31
---

# EP-042 — Composition root refinement — Requirements (EARS / INCOSE)

> **12 requirements** · 9 FR · 3 NFR

## Introduction

Refines EP-027 composition root: wire package, jobs lifecycle clarity, readiness ([ep-scope.md](ep-scope.md)).

## C4 C1 — System Context

<p align="center"><img src="diagrams/c4-context.png" alt="C4 C1 — EP-042" /></p>

**Source:** [diagrams/c4-context.puml](diagrams/c4-context.puml)

## Requirement index

| Id | Type | Summary |
|----|------|---------|
| REQ-42.001 | FR | cmd/pa/wire package with Build function |
| REQ-42.002 | FR | main.go thin entry only |
| REQ-42.003 | FR | Subsystem constructors colocated in wire |
| REQ-42.004 | FR | Explicit jobs runtime state enum |
| REQ-42.005 | FR | Deterministic not-ready messages for tool and /jobs |
| REQ-42.006 | FR | Readiness scheduled_jobs matches runtime state |
| REQ-42.007 | FR | Preserve paApplication.Close teardown |
| REQ-42.008 | FR | Unit tests for three jobs states |
| REQ-42.009 | FR | Document new-subsystem insertion checklist |
| REQ-42.010 | NFR | No handler behaviour change |
| REQ-42.011 | NFR | No config breaking changes (optional flags only) |
| REQ-42.012 | NFR | make check passes |

## Requirements

<a id="req-42-001"></a>

#### REQ-42.001 — wire.Build

THE **PersonalAssistant** SHALL provide `wire.Build(cfg, configPath, logger) (*paApplication, error)` (or equivalent) in `cmd/pa/wire` that constructs the application and all subsystems currently wired from `main.go`.

<a id="req-42-002"></a>

#### REQ-42.002 — Thin main

THE **PersonalAssistant** `main.go` SHALL limit itself to flag parsing, config load, logger setup, `wire.Build`, run loop, and signal shutdown.

<a id="req-42-003"></a>

#### REQ-42.003 — Constructor colocation

THE **wire package** SHALL own or re-export subsystem build functions moved from `main.go` (LLM providers, intent classifier, infrastructure, handler, tool registry).

<a id="req-42-004"></a>

#### REQ-42.004 — Jobs state enum

THE **PersonalAssistant** SHALL represent scheduled-jobs runtime with an explicit three-state model: initializing, ready, failed.

<a id="req-42-005"></a>

#### REQ-42.005 — Deterministic jobs messages

WHILE jobs runtime is initializing or failed, THE **PersonalAssistant** SHALL return stable user-facing messages from the create-job tool and `/jobs` command handler without panics or generic internal errors.

<a id="req-42-006"></a>

#### REQ-42.006 — Readiness alignment

THE **readiness HTTP handler** SHALL report `scheduled_jobs` check consistent with the jobs state enum (not ready when initializing or failed).

<a id="req-42-007"></a>

#### REQ-42.007 — Teardown preserved

THE **paApplication.Close** method SHALL release all resources acquired in `wire.Build` without regression from EP-027.

<a id="req-42-008"></a>

#### REQ-42.008 — Jobs state tests

THE **repository** SHALL include unit tests covering initializing, ready, and failed jobs states for command handler and tool lookup paths.

<a id="req-42-009"></a>

#### REQ-42.009 — Insertion checklist

THE **repository** SHALL document in `docs/architecture.md` or operator dev notes how to add a new subsystem (constructor location + wire hook).

<a id="req-42-010"></a>

#### REQ-42.010 — No core behaviour change

THE **internal/core** conversation handler SHALL not change turn behaviour as part of this epic.

<a id="req-42-011"></a>

#### REQ-42.011 — Config compatibility

THE **Config loader** SHALL not require new mandatory root keys; optional observability flags MAY be added only with JSON null default in examples.

<a id="req-42-012"></a>

#### REQ-42.012 — make check

THE **repository** SHALL pass `make check`.
