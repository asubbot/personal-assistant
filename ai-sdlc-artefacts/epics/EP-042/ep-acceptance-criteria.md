---
artefact: ep-acceptance-criteria
epic_id: EP-042
status: draft
source_of_truth: true
updated_at: 2026-05-31
---

# EP-042 — Composition root refinement — Acceptance criteria

**Git branch:** `epic/EP-042-composition-root-refinement`

## Acceptance criteria index

| AC ID | REQ | Test level | Summary |
|-------|-----|------------|---------|
| [AC-42.001](#ac-42-001) | REQ-42.001, REQ-42.002, REQ-42.003, REQ-42.010 | Manual | wire.Build exists; main.go thin |
| [AC-42.002](#ac-42-002) | REQ-42.004, REQ-42.005, REQ-42.008 | Unit | Jobs states return deterministic messages |
| [AC-42.003](#ac-42-003) | REQ-42.006 | Unit | Readiness JSON reflects jobs state |
| [AC-42.004](#ac-42-004) | REQ-42.007 | Unit | Close releases resources |
| [AC-42.005](#ac-42-005) | REQ-42.009 | Manual | Subsystem insertion documented |
| [AC-42.006](#ac-42-006) | REQ-42.011, REQ-42.012 | Manual (make check) | make check passes |

## Acceptance criteria

<a id="ac-42-001"></a>

### AC-42.001

**Trace:** REQ-42.001, REQ-42.002, REQ-42.003  
**Test level:** Manual

Given the epic branch  
When inspecting `cmd/pa/main.go` and `cmd/pa/wire`  
Then startup wiring SHALL be invoked via `wire.Build` and `main.go` SHALL not contain subsystem constructor implementations.

**Status:** AC-42.001 MANUAL ONLY — verified by `TestEP027_MainRunServerDelegatesToApplication` and inspection of `cmd/pa/wire/`.

<a id="ac-42-002"></a>

### AC-42.002

**Trace:** REQ-42.004, REQ-42.005, REQ-42.008  
**Test level:** Unit

Given jobs runtime in initializing, ready, and failed states  
When `/jobs list` and create-job tool lookup run in `cmd/pa/jobs_runtime_test.go`  
Then responses SHALL match documented stable strings for each state.

<a id="ac-42-003"></a>

### AC-42.003

**Trace:** REQ-42.006  
**Test level:** Unit

Given jobs runtime initializing  
When readiness eval runs  
Then `scheduled_jobs` check SHALL be not OK with detail containing `initializing`.

<a id="ac-42-004"></a>

### AC-42.004

**Trace:** REQ-42.007  
**Test level:** Unit

Given `wire.Application`  
When teardown is required  
Then `Close()` SHALL release acquired resources.

<a id="ac-42-005"></a>

### AC-42.005

**Trace:** REQ-42.009  
**Test level:** Manual

Given `docs/architecture.md` on the epic branch  
When read by a maintainer  
Then subsystem insertion checklist SHALL be present.

**Status:** AC-42.005 MANUAL ONLY — verified by reading `docs/architecture.md` composition-root section.

<a id="ac-42-006"></a>

### AC-42.006

**Trace:** REQ-42.011, REQ-42.012  
**Test level:** Manual (make check)

When `make check` runs  
Then exit code SHALL be zero.

**Status:** AC-42.006 MANUAL ONLY — verified by running `make check` (exit 0).
