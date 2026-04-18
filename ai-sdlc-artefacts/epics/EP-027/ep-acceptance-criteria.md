# EP-027 — Acceptance criteria

## Introduction

Testable criteria for the composition root and application lifecycle, traceable to [ep-requirements.md](ep-requirements.md).

---

## Acceptance criteria index

| AC ID | REQ (trace) | Summary |
|-------|-------------|---------|
| [AC-27.001](#ac-27-001) | [REQ-27.001](ep-requirements.md#composition) | `paInfrastructure` and `buildPAInfrastructure` exist |
| [AC-27.002](#ac-27-002) | [REQ-27.002](ep-requirements.md#application) | `paApplication` with `Close` / summarization stop |
| [AC-27.003](#ac-27-003) | [REQ-27.003](ep-requirements.md#server-entry) | `runServer` uses `paApplication` |
| [AC-27.004](#ac-27-004) | [REQ-27.004](ep-requirements.md#jobs-hand-off) | Runtime lookup yields soft strings |
| [AC-27.005](#ac-27-005) | [REQ-27.005](ep-requirements.md#lint) | Startup sources omit `gocyclo` nolint |
| [AC-27.006](#ac-27-006) | [REQ-27.006](ep-requirements.md#verification) | `make check` and validate EP-027 pass |

---

## Acceptance criteria

### AC-27.001

**Trace:** [REQ-27.001](ep-requirements.md#composition)

Given the `cmd/pa` source tree  
When a reviewer searches for `paInfrastructure` and `buildPAInfrastructure`  
Then both identifiers are defined in `setup_infra.go`.

### AC-27.002

**Trace:** [REQ-27.002](ep-requirements.md#application)

Given `application.go`  
When a reviewer inspects `paApplication`  
Then the type defines `Close` and `stopMemorySummarization` methods used from `runServer` defers.

### AC-27.003

**Trace:** [REQ-27.003](ep-requirements.md#server-entry)

Given `main.go`  
When a reviewer reads `runServer`  
Then the function calls `newPAApplication` and methods on the returned `*paApplication` instead of inlining the previous monolithic wiring.

### AC-27.004

**Trace:** [REQ-27.004](ep-requirements.md#jobs-hand-off)

Given package `jobs` tests  
When `go test ./internal/jobs -count=1 -run CreateScheduledJobTool_RuntimeLookup` runs  
Then tests pass and assert the soft reply strings for not-ready and init-failed snapshots.

### AC-27.005

**Trace:** [REQ-27.005](ep-requirements.md#lint)

Given `cmd/pa`  
When `go test ./cmd/pa -count=1 -run TestEP027_StartupSourcesHaveNoGocycloNolint` runs  
Then the test passes.

### AC-27.006

**Trace:** [REQ-27.006](ep-requirements.md#verification)

Given the epic branch  
When `make check` and `make build && ./bin/validate EP-027` run from the repository root  
Then both commands exit with status zero.
