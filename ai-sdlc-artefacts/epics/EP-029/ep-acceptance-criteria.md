# EP-029 — Acceptance criteria

## Introduction

Testable criteria for health, readiness, lifecycle logging, and operator documentation, traceable to [ep-requirements.md](ep-requirements.md).

---

## Acceptance criteria index

| AC ID | REQ (trace) | Summary |
|-------|---------------|---------|
| [AC-29.001](#ac-29-001) | [REQ-29.001](ep-requirements.md#configuration) | Explicit `observability_http` validates and loads |
| [AC-29.002](#ac-29-002) | [REQ-29.002](ep-requirements.md#health) | Health endpoint returns liveness JSON |
| [AC-29.003](#ac-29-003) | [REQ-29.003](ep-requirements.md#readiness) | Readiness HTTP status reflects aggregate state |
| [AC-29.004](#ac-29-004) | [REQ-29.004](ep-requirements.md#readiness-checks) | Readiness composes subsystem checks |
| [AC-29.005](#ac-29-005) | [REQ-29.005](ep-requirements.md#lifecycle-logs) | Lifecycle `slog` attributes are stable |
| [AC-29.006](#ac-29-006) | [REQ-29.006](ep-requirements.md#no-implicit-listener) | Config without `observability_http` leaves field nil |
| [AC-29.007](#ac-29-007) | [REQ-29.007](ep-requirements.md#operator-documentation) | Operator observability doc exists |
| [AC-29.008](#ac-29-008) | [REQ-29.008](ep-requirements.md#verification) | `make check` and validate EP-029 pass |

---

## Acceptance criteria

### AC-29.001

**Trace:** [REQ-29.001](ep-requirements.md#configuration)

Given `internal/config/testdata/valid_observability_http.json`  
When `config.Load` runs  
Then the load succeeds and `cfg.ObservabilityHTTP` is non-nil with the expected `listen_address`, paths, and `probe_llm`.

Given `internal/config/testdata/invalid_observability_http_same_paths.json`  
When `config.Load` runs  
Then the load fails with an error indicating health and readiness paths must differ.

### AC-29.002

**Trace:** [REQ-29.002](ep-requirements.md#health)

Given `cmd/pa` tests  
When `TestObservabilityHTTPHandler_HealthAndReadiness` runs  
Then a `GET` against the configured health path returns HTTP 200 and JSON containing `"status":"alive"`.

### AC-29.003

**Trace:** [REQ-29.003](ep-requirements.md#readiness)

Given `cmd/pa` tests  
When `TestObservabilityHTTPHandler_HealthAndReadiness` runs  
Then a `GET` against the configured readiness path returns HTTP 200 when all checks pass.

Given `cmd/pa` tests  
When `TestEvalReadiness_ToolIndexNotReady` runs  
Then `evalReadiness` reports `ready` false while the tool index is not ready.

### AC-29.004

**Trace:** [REQ-29.004](ep-requirements.md#readiness-checks)

Given `cmd/pa` tests  
When `TestEvalReadiness_AllOKWithoutJobsOrMemoryWorker` runs  
Then `evalReadiness` returns `ready` true when LLM providers are loaded, vector bundle is complete, and the tool index is ready (with jobs and memory summarization not required in that fixture).

### AC-29.005

**Trace:** [REQ-29.005](ep-requirements.md#lifecycle-logs)

Given `internal/lifecyclelog` tests  
When `TestInfo_IncludesLifecycleFields` runs  
Then log output includes `lifecycle_event`, `subsystem`, `lifecycle_phase`, and `duration_ms`.

Given `internal/toolindex` tests  
When `TestLogBuildOutcome_success_infoWithTools` runs  
Then the success record uses message `lifecycle` and includes `subsystem=tool_index` and `tool_count`.

### AC-29.006

**Trace:** [REQ-29.006](ep-requirements.md#no-implicit-listener)

Given `internal/config/testdata/valid_no_users.json`  
When `config.Load` runs  
Then `cfg.ObservabilityHTTP` is nil.

### AC-29.007

**Trace:** [REQ-29.007](ep-requirements.md#operator-documentation)

Given the repository checkout  
When a reviewer opens `docs/observability-http.md`  
Then the document describes `observability_http`, example Docker `HEALTHCHECK` using the health path, readiness usage, and lifecycle log field names.

### AC-29.008

**Trace:** [REQ-29.008](ep-requirements.md#verification)

Given `cmd/pa` tests  
When `TestEP029_validateCommandExitZero` runs (as part of `go test ./cmd/pa` inside `make check`)  
Then it executes the repository validate entrypoint for **EP-029** with exit status zero (same logic as `go run ./ai-sdlc/tools/validate EP-029`).

Supporting verification for operators and release tags: from the repository root, `make check` exits zero, and `make build && ./bin/validate EP-029` exits zero when the `bin/validate` helper is built.
