# EP-025 — Acceptance criteria

## Introduction

Testable criteria for E2E separation and delivery-runner extraction, traceable to [ep-requirements.md](ep-requirements.md).

---

## Acceptance criteria index

| AC ID | REQ (trace) | Summary |
|-------|-------------|---------|
| [AC-25.001](#ac-25-001) | [REQ-25.001](ep-requirements.md#test-layout) | Job E2E scenarios live under `tests/e2e` with `e2e` tag |
| [AC-25.002](#ac-25-002) | [REQ-25.002](ep-requirements.md#test-layout) | Package builds without `e2e` tag |
| [AC-25.003](#ac-25-003) | [REQ-25.003](ep-requirements.md#make-targets) | Makefile defines `test-e2e` |
| [AC-25.004](#ac-25-004) | [REQ-25.004](ep-requirements.md#make-targets) | Makefile defines `coverage-e2e` |
| [AC-25.005](#ac-25-005) | [REQ-25.005](ep-requirements.md#ci) | CI summary mentions e2e layer |
| [AC-25.006](#ac-25-006) | [REQ-25.006](ep-requirements.md#coverage) | Default `coverage` target omits `e2e` tag |
| [AC-25.007](#ac-25-007) | [REQ-25.007](ep-requirements.md#refactor) | `DeliveryRunner` unit tests pass |
| [AC-25.008](#ac-25-008) | [REQ-25.008](ep-requirements.md#verification) | `make check` passes |

---

## Acceptance criteria

### AC-25.001

**Trace:** [REQ-25.001](ep-requirements.md#test-layout)

Given the repository is checked out  
When an engineer runs `go test -tags=integration,e2e ./tests/e2e/...`  
Then the former `cmd/pa` job digest and job management end-to-end tests execute successfully.

### AC-25.002

**Trace:** [REQ-25.002](ep-requirements.md#test-layout)

Given the E2E build tag is not enabled  
When `go test -tags=integration ./tests/e2e/...` runs  
Then the placeholder test package compiles and tests pass without executing e2e-tagged files.

### AC-25.003

**Trace:** [REQ-25.003](ep-requirements.md#make-targets)

Given the Makefile at the repository root  
When an automated test inspects the `test-e2e` recipe  
Then the recipe runs `go test` with `integration` and `e2e` tags against `./tests/e2e/...`.

### AC-25.004

**Trace:** [REQ-25.004](ep-requirements.md#make-targets)

Given the Makefile  
When an automated test inspects the `coverage-e2e` recipe  
Then the recipe writes `coverage-e2e.out`.

### AC-25.005

**Trace:** [REQ-25.005](ep-requirements.md#ci)

Given `.github/workflows/ci.yml`  
When an automated test reads the coverage summary step body  
Then the text references the e2e layer explicitly.

### AC-25.006

**Trace:** [REQ-25.006](ep-requirements.md#coverage)

Given the Makefile `coverage` target body  
When an automated test inspects the first `go test` line in that target  
Then the line does not include the substring `e2e`.

### AC-25.007

**Trace:** [REQ-25.007](ep-requirements.md#refactor)

Given `internal/jobs`  
When `go test ./internal/jobs -count=1` runs  
Then `DeliveryRunner` unit tests pass.

### AC-25.008

**Trace:** [REQ-25.008](ep-requirements.md#verification)

Given the epic branch  
When `make check` runs from the repository root  
Then the command exits with status zero.
