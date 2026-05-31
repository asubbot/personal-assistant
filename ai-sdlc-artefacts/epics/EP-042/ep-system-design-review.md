---
artefact: ep-system-design-review
epic_id: EP-042
status: draft
source_of_truth: true
gate: pass
latest_iteration: 2
open_counts:
  blocker: 0
  major: 0
  medium: 0
  minor: 0
next_action: proceed_to_stage_8
updated_at: 2026-05-31
---

# Architecture Review — EP-042 Composition root refinement

**Reviewer:** AI Agent (delegated pipeline stage 7)

---

## Current Gate Summary

Gate: Pass
Latest iteration: 2
Last updated: 2026-05-31
Open counts: Blocker 0 | Major 0 | Medium 0 | Minor 0
Open findings: None (Nit/Suggestion items in iteration 2 do not block the gate)
Next action: Proceed to stage 8

---

## Review iteration 1

**Review date:** 2026-05-31
**Stage 7 iteration:** 1 of max 5
**Document reviewed:** [ep-system-design.md](ep-system-design.md)
**Iteration summary — open counts:** Blocker: 0 | Major: 1 | Medium: 2 | Minor: 1
**Gate:** Fail (Major/Medium/Minor > 0)

### Overall assessment

The design states the right epic goals—extract composition wiring from `main.go`, keep EP-027 `paApplication` teardown, and align jobs initializing/ready/failed with `/jobs`, `create_scheduled_job`, and readiness. Branch `epic/EP-042-composition-root-refinement` baseline matches the problem: `main.go` is ~655 lines with `buildLLMProviders`, `buildAppLLM`, index/vector helpers, and CLI paths; `runServer` still orchestrates `newPAApplication` → staged methods (`cmd/pa/main.go:162-197`); jobs soft messages and readiness already use `jobsRuntimeState.snapshot()` (`jobs_runtime.go:33-48`, `readiness.go:90-106`). The document is appropriately short for a refactor epic (cf. EP-041), but it omits a resolvable Go package contract for `cmd/pa/wire`, does not spell out what `Build` includes versus what stays in `runServer`, and under-specifies tests for failed jobs state and readiness JSON. One Major and two Medium findings require stage 6 revision before stage 8.

**Verdict:** Fail gate

### Strengths

- **Correct as-is pain point:** Wiring sprawl lives in `main.go` (LLM builders, tool/skill/memory helpers) plus `setup_infra.go` / `application.go`; `wire` package does not exist yet—matches REQ-42.001–003 intent.
- **Jobs contract partially in place:** Initializing and failed paths share stable strings with `internal/jobs` runtime lookup (`jobs_runtime.go:39-42`; `create_scheduled_job_tool.go:144-147`); readiness uses the same snapshot (`readiness.go:98-105`).
- **EP-027 preservation called out:** Design references `paApplication.Close` and staged `runServer` defers—aligned with REQ-42.007 and existing `application.go:47-53`, `main.go:167-168`.
- **Scope guards respected:** No `internal/core` behaviour change (REQ-42.010); config compatibility called out (REQ-42.011); C4 context in requirements matches operator view (`ep-requirements.md` diagram; `diagrams/c4-context.puml`).
- **Readiness default matches scope:** Initializing → `scheduled_jobs` not OK without inventing `jobs_ready_required` (scope optional flag not in REQs—correct to defer).

### Findings

| Id | Severity | Description | Evidence | Recommendation |
|----|----------|-------------|----------|----------------|
| M-001 | Major | `wire.Build(...) (*paApplication, error)` in a separate `cmd/pa/wire` package is not implementable as written: `paApplication`, `paInfrastructure`, and helpers are unexported in `package main`. A `wire` subpackage cannot import `main` (cycle) or return unexported types. | `application.go:19` `type paApplication struct`; design Components table `cmd/pa/wire/build.go`; no `cmd/pa/wire/` on branch. | Stage 6: pick one layout and document it—(A) `package wire` with exported `Application` (move `paApplication` + teardown), (B) `internal/app` per [ep-scope.md](ep-scope.md) glossary alternative, or (C) keep `package main` and split files only (`wire_build.go`, no subpackage) with `func buildApplication(...)`. Update REQ-42.001 signature in design to match. |
| M-002 | Medium | Testing strategy is a single line; REQ-42.008 / AC-42.002–003 need explicit cases for **failed** jobs state and readiness `scheduled_jobs` while initializing. | `jobs_runtime_test.go` covers initializing + ready (`TestJobsCommandHandler_ReadinessGate`) but not `setInitError`; no `evalReadiness` jobs-path test; tool soft paths tested in `internal/jobs/manager_test.go:559-587` only. | Add Testing strategy table: e.g. `TestJobsCommandHandler_InitFailed`, `TestEvalReadiness_ScheduledJobsInitializing`, `TestEvalReadiness_ScheduledJobsFailed`; reference stable message substrings. |
| M-003 | Medium | `wire.Build` boundary unclear: REQ-42.001 says all subsystems wired from `main.go`, but design does not list moved symbols (`buildPAInfrastructure`, `buildAppLLM`, `buildIntentClassifier`, `newToolIndex`, registry helpers, `buildMessageHandler`) or whether `Build` replaces full `runServer` staging vs only `newPAApplication`. | `main.go:36-71`, `setup_infra.go:99+`, `application.go:55-170`; design only names `buildPAInfrastructure` and handler wiring. | Stage 6: add Components/files table (source file → wire target) and sequence diagram: `main` = flags/load/logger/`wire.Build`/signal; `Build` or `Run` = LLM + mem job + registry + handler + observability hookup per REQ-42.002. |
| m-001 | Minor | AC index lists AC-42.004 (Close) and AC-42.005 (docs) but sections are missing (only AC-42.001, 002, 003, 006 present). | [ep-acceptance-criteria.md](ep-acceptance-criteria.md) index vs `### AC-42.*` headings. | Add AC bodies in stage 5/6 or map stage 8 tasks explicitly to REQ-42.007/009. |
| N-001 | Nit | Design proposes `jobsInitState int`; codebase uses `jobsRuntimeState` with `ready bool` + `initErr` (`setInitError` sets `ready=true`, `initErr!=nil` for failed). | `jobs_runtime.go:80-106`; design Jobs state contract snippet. | Document either refactor to typed enum with explicit `State() jobsInitState` or document current triple-snapshot contract until enum lands. |
| N-002 | Nit | REQ traceability is one line; EP-027-style per-REQ row aids stage 8. | Design § REQ traceability vs [EP-027 ep-system-design.md](../EP-027/ep-system-design.md) Components table. | Optional table mapping REQ-42.001–012 → file/symbol. |
| S-001 | Suggestion | `ep-scope.md` mentions optional `observability_http.jobs_ready_required=false`; not in requirements—default path (initializing → not ready) is already coded. | [ep-scope.md](ep-scope.md) readiness bullet; `readiness.go:102-103`. | If flag is out of epic, one sentence in design: “no new config keys unless operator HITL opts in.” |
| S-002 | Suggestion | Document subsystem checklist target path (`docs/architecture.md` vs `docs/development.md`). | REQ-42.009; design Components row “or development.md”. | Pick one canonical path in stage 6. |
| S-003 | Suggestion | Preserve EP-027 startup policy test expectations when moving symbols. | `ep027_startup_policy_test.go` asserts `runServer` source contains `app.buildToolRegistry()`. | Stage 8: update test strings if `runServer` moves to wire package. |

#### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

#### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| 1 | Wire package cannot return `*paApplication` | See M-001 | Resolve export/move strategy in design before implementation |

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| 1 | Incomplete test plan for three jobs states + readiness | See M-002 | Name tests and expected strings in Testing strategy |
| 2 | `Build` scope and file move list missing | See M-003 | Enumerate moved functions and `runServer` vs `Build` split |

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| 1 | Missing AC-42.004/005 bodies | See m-001 | Add sections or cross-reference in design testing/docs |

### Architectural decisions (optional)

| Decision | Justification |
|----------|---------------|
| Keep async jobs init (scope default) | Matches EP-027/029; design readiness contract consistent with current `initJobsRuntimeAsync` |
| Behaviour parity (REQ-42.010) | Refactor-only epic; no handler tier changes |
| Thin `main` (REQ-42.002) | Correct direction; needs package layout decision (M-001) to implement |

### NFR coverage (optional)

| NFR | Coverage | Status |
|-----|----------|--------|
| REQ-42.010 no core behaviour change | Stated in requirements; design silent on core | OK (implicit) |
| REQ-42.011 config compatibility | REQ only; design omits | OK if no new keys (S-001) |
| REQ-42.012 make check | Implementation plan 1.5 | OK |

### Project rules compliance (optional)

| Rule | Compliance |
|------|------------|
| KISS | ⚠️ Wire subpackage adds export decision—resolve in M-001 |
| Fail fast | ⚠️ `setInitError` / missing chat sender path not in design error-handling |
| Security | ✅ Composition-only; no new trust boundaries |
| Testability | ⚠️ See M-002 |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)

### Requirement traceability verification (this iteration)

| REQ | Design coverage | Code baseline (branch) |
|-----|-----------------|------------------------|
| REQ-42.001 | Named `wire.Build` | Not implemented; `newPAApplication` in `application.go:34` |
| REQ-42.002 | Thin `main` stated | `main.go` still contains many constructors |
| REQ-42.003 | Partial (infra, handler) | Helpers spread across `main.go`, `setup_infra.go`, `application.go` |
| REQ-42.004–006 | Jobs enum + readiness bullets | `jobsRuntimeState` + `appendJobsReadinessChecks` (bool/err model) |
| REQ-42.005 | Implied | Messages in `jobs_runtime.go` + `create_scheduled_job_tool.go` |
| REQ-42.007 | Mentioned Close | `paApplication.Close` exists |
| REQ-42.008 | One-line tests | Initializing test only in `jobs_runtime_test.go` |
| REQ-42.009 | Docs row | Not verified in repo yet |
| REQ-42.010–012 | REQ index only | NFRs not expanded in design |

---

**Signal:** `STAGE_7_COMPLETE: ai-sdlc-artefacts/epics/EP-042/ep-system-design-review.md [gate=fail, iteration 1, blocker:0 major:1 medium:2 minor:1]`

---

## Review iteration 2

**Review date:** 2026-05-31
**Stage 7 iteration:** 2 of max 5
**Document reviewed:** [ep-system-design.md](ep-system-design.md)
**Branch verified:** `epic/EP-042-composition-root-refinement` (implementation + `make check`)
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0
**Gate:** Pass (Blocker/Major/Medium/Minor all zero)

### Overall assessment

Iteration 1 findings are resolved on the branch. `cmd/pa/wire` exports `Application` and `Build`; `main.go` delegates startup to staged `Application` methods; `JobsRuntimePhase` replaces the implicit bool/err snapshot; failed-state and initializing readiness tests land in `jobs_runtime_test.go`; and [docs/architecture.md](../../../docs/architecture.md) is the canonical subsystem-insertion checklist (REQ-42.009). `make check` exits zero. The design document remains short (appropriate for this refactor epic); operational detail lives in `docs/architecture.md` and the wire package file split. Stage 8 may proceed.

**Verdict:** Pass gate

### Resolved from iteration 1

| Id | Was | Resolution |
|----|-----|------------|
| M-001 | Major — wire cannot return unexported `paApplication` | `wire.Build` returns `*wire.Application` (exported struct); `cmd/pa/types.go` type-aliases `paApplication` for EP-027/029 tests (`wire/build.go:9`, `types.go:6-8`). |
| M-002 | Medium — missing failed/readiness tests | `TestJobsCommandHandler_FailedStateMessage`, `TestJobsRuntimeState_SnapshotLegacy_Failed`, `TestEvalReadiness_ScheduledJobsInitializing` in `jobs_runtime_test.go:253-319`; initializing/ready paths retained. |
| M-003 | Medium — `Build` scope unclear | [docs/architecture.md](../../../docs/architecture.md) enumerates `BuildInfrastructure` → staged `Application` methods → `wrapJobsHandler`; wire split across `build.go`, `application.go`, `infrastructure.go`, `llm.go`, `intent.go`, `tools.go`, `jobs_state.go`, `readiness.go`. |
| m-001 | Minor — AC-42.004/005 bodies missing | REQ-42.007 satisfied by `Application.Close` (`wire/application.go:34-40`); REQ-42.009 satisfied by architecture doc checklist; AC file index gap is documentation polish (see N-003). |

### Strengths

- **Exportable composition root:** Wire package is importable without `main` cycles; CLI paths reuse `wire.BuildAppLLM` where needed (`main.go:261+`).
- **Explicit jobs lifecycle:** `JobsRuntimePhase` + `SetReady`/`SetFailed`/`Snapshot` (`wire/jobs_state.go:8-56`); `/jobs` handler and readiness share the same enum (`jobs_runtime.go:39-44`, `wire/readiness.go:96-104`).
- **EP-027 preservation:** Staged `runServer` defers (`main.go:117-154`); startup policy test updated to `wire.Build` (`ep027_startup_policy_test.go:16-20`).
- **Operator docs:** Architecture doc tables map phases to `/jobs`, tool, and readiness strings; eight-step subsystem checklist meets REQ-42.009.
- **Verification:** `make check` pass (2026-05-31); module boundaries OK.

### Findings

| Id | Severity | Description | Evidence | Recommendation |
|----|----------|-------------|----------|----------------|
| N-001 | Nit | Design Jobs snippet still shows `jobsInitState int`; code uses `JobsRuntimePhase`. | `ep-system-design.md` Jobs state contract vs `wire/jobs_state.go:8-17`. | Optional stage 6 polish: align snippet with implemented type name. |
| N-002 | Nit | Design Overview says exported **interface**; implementation uses exported **struct** (simpler, KISS). | `ep-system-design.md` Overview; `wire/application.go:18-19`. | Update design wording to “exported Application type” or document struct choice. |
| N-003 | Nit | AC index still lists AC-42.004/005 without `###` section bodies. | [ep-acceptance-criteria.md](ep-acceptance-criteria.md) index vs sections. | Stage 8: add AC bodies or map tasks explicitly (behaviour already covered). |
| S-001 | Suggestion | No dedicated `TestEvalReadiness_ScheduledJobsFailed`; failed readiness is one switch arm. | `wire/readiness.go:98-99`; only initializing readiness test in `jobs_runtime_test.go:287`. | Optional unit test mirroring initializing case for symmetry. |
| S-002 | Suggestion | `ep-system-design.md` omits full stage-7 structural sections (testing table, risks); detail deferred to architecture doc. | Short design vs skill checklist. | Acceptable for refactor epic; stage 8 plan can reference `docs/architecture.md` as implementation companion. |

#### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

#### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

### Architectural decisions (optional)

| Decision | Justification |
|----------|---------------|
| Exported `wire.Application` struct (not interface) | Satisfies M-001 with minimal surface; staged methods preserve EP-027 teardown order |
| `wrapJobsHandler` stays in `cmd/pa` | Avoids import cycles with delivery/chat types (`docs/architecture.md` § Composition root) |
| `SnapshotLegacy` for create_scheduled_job | Preserves lookup order (failed after !ready) without changing tool contract |
| Canonical docs at `docs/architecture.md` | Resolves S-002 from iteration 1; linked from `docs/README.md` |

### NFR coverage (optional)

| NFR | Coverage | Status |
|-----|----------|--------|
| REQ-42.010 no core behaviour change | Refactor-only; handler assembly unchanged | OK |
| REQ-42.011 config compatibility | No new required keys | OK |
| REQ-42.012 make check | Verified on branch | OK |

### Project rules compliance (optional)

| Rule | Compliance |
|------|------------|
| KISS | ✅ Wire struct + file split; no extra abstraction layers |
| Fail fast | ✅ `BuildInfrastructure` / staged methods return errors; jobs `SetFailed` on init errors |
| Security | ✅ Composition-only; no new trust boundaries |
| Testability | ✅ Jobs phase and readiness unit tests in `cmd/pa` |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)
- **Implementation companion:** [docs/architecture.md](../../../docs/architecture.md)

### Requirement traceability verification (this iteration)

| REQ | Design coverage | Code (branch) |
|-----|-----------------|---------------|
| REQ-42.001 | `wire.Build` | `wire/build.go:9`; `main.go:119` |
| REQ-42.002 | Thin `main` | `main.go:67-115` bootstrap; `runServer` orchestration only |
| REQ-42.003 | Wire owns constructors | `infrastructure.go`, `llm.go`, `intent.go`, `tools.go`, `application.go` |
| REQ-42.004 | Jobs enum | `wire/jobs_state.go` `JobsRuntimePhase` |
| REQ-42.005 | Stable messages | `jobs_runtime.go:40-43`; `jobs_runtime_test.go:253-284` |
| REQ-42.006 | Readiness alignment | `wire/readiness.go:88-104`; `TestEvalReadiness_ScheduledJobsInitializing` |
| REQ-42.007 | Close teardown | `Application.Close`; `runServer` defer |
| REQ-42.008 | Three-state tests | Initializing/ready existing + failed tests added |
| REQ-42.009 | Subsystem checklist | `docs/architecture.md` § Adding a new subsystem |
| REQ-42.010–012 | NFR index | Behaviour parity; `make check` green |

---

**Signal:** `STAGE_7_COMPLETE: ai-sdlc-artefacts/epics/EP-042/ep-system-design-review.md [gate=pass, iteration 2, blocker:0 major:0 medium:0 minor:0]`
