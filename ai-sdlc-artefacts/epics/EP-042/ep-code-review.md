---
artefact: ep-code-review
epic_id: EP-042
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
  suggestion: 2
next_action: proceed_to_stage_11
updated_at: 2026-05-31
---

# Code review — EP-042 Composition root refinement

---

## Current Gate Summary

Gate: Pass
Latest iteration: 1
Last updated: 2026-05-31
Open counts: Blocker 0 | Major 0 | Medium 0 | Minor 0
Non-blocking counts: Nit 1 | Suggestion 2
Open findings: none
Next action: Proceed to stage 11

---

## Review iteration 1

**Review date:** 2026-05-31
**Stage 10 iteration:** 1 of max 5
**Scope:** Branch `epic/EP-042-composition-root-refinement` vs `main` (22 files, +798/−523 lines). Commits: `docs(EP-042): stage 6+8 artefacts`, `feat(EP-042): extract cmd/pa/wire composition root`.
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0 | Nit: 1 | Suggestion: 2
**Gate:** Pass

### Summary

The implementation delivers the EP-042 plan cleanly: composition wiring moves into `cmd/pa/wire`, `main.go` delegates startup through `wire.Build`, jobs runtime uses an explicit three-phase enum aligned with readiness, new unit tests cover failed/initializing paths, and `docs/architecture.md` documents the subsystem insertion checklist. No changes under `internal/` — REQ-42.010 satisfied. **`make check` passes** (exit 0). **Approve** for merge.

### What was done well

- **`cmd/pa/wire` package** — Focused split (`build.go`, `application.go`, `infrastructure.go`, `tools.go`, `llm.go`, `intent.go`, `jobs_state.go`, `readiness.go`) with clear ownership; `Build` fail-fast on infrastructure errors.
- **Thin bootstrap** — `runServer` calls `wire.Build` then exported `Application` methods; EP-027 traceability test updated to assert `wire.Build` and `BuildToolRegistry`/`BuildMessageHandler` (AC-42.001).
- **Type aliases in `cmd/pa/types.go`** — `paApplication` / `paInfrastructure` aliases preserve EP-027/EP-029 test stability without duplicating types.
- **Jobs state contract** — `JobsRuntimePhase` enum with documented phases; `/jobs` handler and `SnapshotLegacy` return stable messages for initializing and failed; readiness `scheduled_jobs` check uses the same enum.
- **Tests** — `TestJobsCommandHandler_FailedStateMessage`, `TestJobsRuntimeState_SnapshotLegacy_Failed`, and `TestEvalReadiness_ScheduledJobsInitializing` cover AC-42.002 and AC-42.003; prior EP-019/EP-021 jobs tests retained.
- **Documentation** — New `docs/architecture.md` section with composition-root flow, jobs state table, and eight-step subsystem checklist (REQ-42.009); linked from `docs/README.md`.
- **No gocyclo nolint** — Startup sources scan includes `wire/*` paths per EP-027 policy.
- **Import-cycle boundary** — `wrapJobsHandler` and async jobs init remain in `cmd/pa` with explicit rationale in architecture doc.

### Findings

| ID | Severity | Location | Issue | Recommendation |
|----|----------|----------|-------|----------------|
| F-001 | **Nit** | `docs/architecture.md` step 6 | Checklist references `evalReadiness`; exported method is `EvalReadiness`. | Fix casing in the checklist for consistency with code. |
| F-002 | **Suggestion** | `cmd/pa/jobs_runtime_test.go` | AC-42.003 covers initializing readiness only; failed-phase readiness (`detail: initialization failed`) is implemented in `wire/readiness.go` but not unit-tested. | Add `TestEvalReadiness_ScheduledJobsFailed` mirroring the initializing test for stronger REQ-42.006 coverage. Not blocking. |
| F-003 | **Suggestion** | `cmd/pa/jobs_runtime_test.go` `TestJobsCommandHandler_ReadinessGate` | Initializing `/jobs` reply asserted with `strings.Contains(..., "initializing")`; failed-state test uses exact string match. | Align initializing test to exact `jobsMsgInitializing` constant for parity with failed test. Not blocking. |

### Plan alignment

| Plan step | Status | Notes |
|-----------|--------|-------|
| 1.1 Create `cmd/pa/wire/` with `Build` | Done | All wiring helpers moved |
| 1.2 Slim `main.go` | Done | Wiring out; CLI paths (`-summarize`, `-verify-nodes`) remain in main per design |
| 1.3 Jobs state type/docs + tests | Done | `JobsRuntimePhase`, tests for three states |
| 1.4 Subsystem insertion checklist | Done | `docs/architecture.md` |
| 1.5 `make check` + commit | Done | Commit present on branch |

**Deviations (justified):**

- System design proposed an exported `Application` **interface**; implementation uses an exported **`Application` struct**. Simpler for tests constructing partial apps; no cross-package unexported-type issue because struct is exported.
- Optional `observability_http.jobs_ready_required` flag from scope was not added; default initializing→not-ready behaviour matches REQ-42.006 without new config.

### REQ / AC traceability

| REQ / AC | Verdict |
|----------|---------|
| REQ-42.001, AC-42.001 | Pass — `wire.Build` exists; main delegates |
| REQ-42.002 | Pass — main is bootstrap + CLI utilities |
| REQ-42.003 | Pass — constructors in wire package |
| REQ-42.004, REQ-42.005, AC-42.002 | Pass — enum + stable messages + tests |
| REQ-42.006, AC-42.003 | Pass — readiness uses phase; initializing tested |
| REQ-42.007, AC-42.004 | Pass — `Application.Close` → `Infrastructure.Close`; EP-027 teardown preserved |
| REQ-42.008 | Pass — initializing, ready, failed covered for handler and tool |
| REQ-42.009, AC-42.005 | Pass — architecture checklist |
| REQ-42.010 | Pass — zero `internal/` diff |
| REQ-42.011 | Pass — no config schema changes |
| REQ-42.012, AC-42.006 | Pass — `make check` green |

### Test / verification

- `make check` — **PASS** (exit 0). All packages OK including `pa/cmd/pa`. Race detector enabled. golangci-lint 0 issues. govulncheck clean. Module boundaries OK. Coverage 56.7%.
- `pa/cmd/pa/wire` — no test files (tests live in `cmd/pa` via type aliases); acceptable for this refactor.

### Residual risks / follow-ups

- Jobs async init race (user tools before ready) unchanged from EP-027; documented in EP-027 review — operators should use readiness `/ready` for probe semantics.
- F-002 and F-003 are optional hardening; may be addressed in EP-043 test-suite work or a follow-up nit fix.
