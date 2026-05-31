---
artefact: ep-scope
epic_id: EP-042
status: draft
source_of_truth: true
updated_at: 2026-05-31
git_branch: epic/EP-042-composition-root-refinement
---

# Epic scope — EP-042 Composition root refinement

| Field | Content |
|-------|---------|
| **ID** | EP-042 |
| **Status** | NEW |
| **Title** | Composition root refinement |
| **Description** | Refine `cmd/pa` startup wiring after EP-027: extract a dedicated wire/build package, clarify scheduled-jobs initialization contract (ready vs initializing vs failed), and reduce `main.go` / `application.go` sprawl so new subsystems have an obvious insertion point. No product behaviour change beyond clearer jobs-not-ready responses and readiness alignment. |
| **First version date** | 2026-05-31 |
| **Git branch** | `epic/EP-042-composition-root-refinement` |

## Glossary

- **Composition root:** Where subsystems are constructed and connected (`cmd/pa`, EP-027 `paApplication`).
- **Wire package:** New `cmd/pa/wire` (or `internal/app`) holding `BuildApplication(cfg, logger) (*paApplication, error)` and subsystem constructors moved out of `main.go`.
- **Jobs init contract:** Explicit enum/state: `initializing`, `ready`, `failed` — surfaced to `/jobs` commands, create-job tool, and HTTP readiness (EP-029).

## Scope (features/capabilities)

- **Prerequisite gate:** Land after **EP-041** or in parallel with EP-043 if no core conflicts; must not overlap open edits on `cmd/pa` with other epics.
- **Extract wire/build module:** Move infrastructure and handler wiring from `main.go` into `cmd/pa/wire/` (or agreed path): `buildPAInfrastructure`, LLM provider build, intent classifier build, handler assembly, tool registry — `main.go` becomes thin entry (flags, config load, `wire.Build`, run, shutdown).
- **Document jobs lifecycle (HITL default: keep async init):** Keep background jobs DB init but make the contract explicit in one type (`jobsRuntimeState` + docs): tools and `/jobs` commands return deterministic messages for each state (already partially implemented; unify and test).
- **Readiness alignment:** `scheduled_jobs` check in readiness (EP-029) SHALL match the same state enum; `ready=false` while initializing unless operator opts into `observability_http.jobs_ready_required=false` (new optional flag only if needed — default: initializing → not ready).
- **Teardown:** Single `Close()` path remains; wire package documents acquisition order.
- **Tests:** Extend `jobs_runtime_test.go` and readiness tests for all three jobs states; no goroutine leaks.
- **Verification:** `make check` green.

## Out of scope / deferred

- Re-implementing EP-027 from scratch (this epic **refines** it).
- Blocking/synchronous jobs init (optional HITL — default keeps async).
- New subsystems, rate limiting, or microservice split.
- Changes to `internal/core` handler logic (EP-040/041 territory).

## Success criteria

- `main.go` LOC reduced materially (target: wiring helpers moved out; `main` reads as bootstrap only).
- New subsystem checklist documented: one constructor file + one line in wire build.
- Jobs state behaviour covered by unit tests for tool lookup, `/jobs list`, and readiness JSON.
- **`make check`** passes.

## Execution order

| Order | Epic | Branch |
|-------|------|--------|
| 4 | **EP-042 (this epic)** | `epic/EP-042-composition-root-refinement` |
| 5 | EP-043 Test suite | `epic/EP-043-test-suite-organization` |

## Traceability

- **Strategy:** Refactoring 0.02 ([strategy.md](../../strategy.md)).
- **Builds on:** [EP-027](../EP-027/ep-scope.md) (composition root — DONE), [EP-029](../EP-029/ep-scope.md) (readiness).
- **Architecture:** Addresses async jobs init weakness in [docs/architecture-ru.md](../../../docs/architecture-ru.md) without mandatory blocking startup.
