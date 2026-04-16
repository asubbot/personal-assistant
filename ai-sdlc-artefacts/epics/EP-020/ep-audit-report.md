# Audit report - EP-020 Natural-Language Scheduled Job Creation from Telegram

**Date and time of creation:** 2026-04-16 (UTC)  
**Pipeline:** [pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md) stage 11  
**Plan:** [ep-implementation-plan.md](ep-implementation-plan.md)  
**Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)  
**Requirements:** [ep-requirements.md](ep-requirements.md)  
**Code review gate:** [ep-code-review.md](ep-code-review.md) — iteration 3 **Pass**

## Summary

EP-020 implementation is complete against the approved implementation plan (all seven tasks done). Stage 10 gate is closed with no open Blocker/Major/Medium/Minor findings after iteration 3. Malformed explicit schedule-intent path is validated as LLM fallback escalation with tool-enforcement prompts. `make check` and `./bin/validate EP-020` both pass. Total project statement coverage from the same quality run is **73.8%**.

## Implementation vs plan

| Task | Status | Notes |
|------|--------|-------|
| 1. Deterministic NL parser and creation API in jobs manager | Done | `internal/jobs/manager.go`, `manager_test.go` |
| 2. Persist NL-created jobs with scheduler-compatible defaults | Done | `CreateScheduledJobFromSpec`, store integration in tests |
| 3. Native `create_scheduled_job` fallback tool | Done | `internal/jobs/create_scheduled_job_tool.go` |
| 4. Hybrid message routing (`jobsCommandHandler`) | Done | `cmd/pa/jobs_runtime.go`, `jobs_runtime_test.go` |
| 5. Timezone wiring from server config | Done | `cmd/pa/main.go` (`pa_timezone` into jobs runtime) |
| 6. E2E: strict + fallback create, manage, run-now, delivery | Done | `cmd/pa/ep020_e2e_test.go` |
| 7. Docs and AC validation | Done | Plan verification satisfied; `./bin/validate EP-020` 100% |

## Test results and coverage

| Command | Result |
|---------|--------|
| `make check` | Pass (fmt, vet, govulncheck, golangci-lint, `go test -race -tags=integration`, coverage, module boundaries) |
| `./bin/validate EP-020` | Pass — 9/9 AC traced (100% automated) |

**Total coverage:** `total: (statements) 73.8%`

## REQ/AC test coverage matrix

| AC | REQ | Unit | Integration | E2E | Manual | Link |
|----|-----|------|-------------|-----|--------|------|
| [AC-20.001](ep-acceptance-criteria.md#ac-20-001) | [REQ-20.001](ep-requirements.md#nl-request-intake), [REQ-20.002](ep-requirements.md#nl-request-intake) | ✓ | — | ✓ | — | `internal/jobs/manager_test.go`, `cmd/pa/ep020_e2e_test.go` |
| [AC-20.002](ep-acceptance-criteria.md#ac-20-002) | [REQ-20.003](ep-requirements.md#job-creation), [REQ-20.004](ep-requirements.md#job-creation), [REQ-20.005](ep-requirements.md#job-creation) | ✓ | — | — | — | `internal/jobs/manager_test.go` |
| [AC-20.003](ep-acceptance-criteria.md#ac-20-003) | [REQ-20.006](ep-requirements.md#user-feedback-safety-and-compatibility) | ✓ | — | — | — | `internal/jobs/manager_test.go` |
| [AC-20.004](ep-acceptance-criteria.md#ac-20-004) | [REQ-20.007](ep-requirements.md#user-feedback-safety-and-compatibility) | ✓ | ✓ | — | — | `internal/jobs/manager_test.go`, `cmd/pa/jobs_runtime_test.go` |
| [AC-20.005](ep-acceptance-criteria.md#ac-20-005) | [REQ-20.008](ep-requirements.md#user-feedback-safety-and-compatibility) | — | — | ✓ | — | `cmd/pa/ep020_e2e_test.go` |
| [AC-20.006](ep-acceptance-criteria.md#ac-20-006) | [REQ-20.009](ep-requirements.md#runtime-security-and-observability) | — | — | ✓ | — | `cmd/pa/ep020_e2e_test.go` |
| [AC-20.007](ep-acceptance-criteria.md#ac-20-007) | [REQ-20.010](ep-requirements.md#runtime-security-and-observability), [REQ-20.011](ep-requirements.md#runtime-security-and-observability) | ✓ | ✓ | — | — | `internal/telegram/adapter_test.go`, `cmd/pa/jobs_runtime_test.go` |
| [AC-20.008](ep-acceptance-criteria.md#ac-20-008) | [REQ-20.012](ep-requirements.md#runtime-security-and-observability) | ✓ | — | — | — | `internal/jobs/manager_test.go` |
| [AC-20.009](ep-acceptance-criteria.md#ac-20-009) | [REQ-20.013](ep-requirements.md#nl-request-intake), [REQ-20.006](ep-requirements.md#user-feedback-safety-and-compatibility) | ✓ | ✓ | — | — | `internal/jobs/manager_test.go`, `cmd/pa/jobs_runtime_test.go` |

**Notes:** Unit = package tests under `internal/jobs` or `internal/telegram`; Integration = `cmd/pa` handler wiring tests; E2E = `cmd/pa/ep020_e2e_test.go`. REQ-20.009 is traced under the “Runtime, Security, and Observability” section in [ep-requirements.md](ep-requirements.md).

## Quality gate

- Stage 10 code review gate: **Pass** ([ep-code-review.md](ep-code-review.md), iteration 3).
- `make check`: **Pass**.
- `./bin/validate EP-020`: **Pass**.

## Gaps, risks, recommendations

- **Gap:** None blocking for EP-020 scope.
- **Risk:** Fallback phrase coverage is regex-based; unsupported phrasing still falls through to normal chat unless it matches explicit schedule-intent heuristics.
- **Recommendation:** If user reports high rejection rates, extend documented phrase set or add metrics from `jobs audit` events in a follow-up epic.
