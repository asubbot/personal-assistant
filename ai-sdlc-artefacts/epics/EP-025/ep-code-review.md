# Code review — EP-025 Test layout cleanup: E2E separation

**Reviewer:** Delegated agent (pipeline stage 10)

---

## Review iteration 1

**Review date:** 2026-04-17  
**Stage 10 iteration:** 1 of max 5  
**Scope:** `internal/jobs/delivery_runner.go`, `internal/jobs/delivery_runner_test.go`; `cmd/pa/jobs_runtime.go` (scheduled-job init and wiring), `cmd/pa/jobs_runtime_test.go`; `tests/e2e/*.go`; `Makefile` (`test-e2e`, `coverage-e2e`, `coverage`, `check`, `vet`, `vuln`, `lint`); `.github/workflows/ci.yml` (coverage summary step). Cross-checked [ep-requirements.md](ep-requirements.md) (REQ-25.001–REQ-25.008).

**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0 | Nit: 3 | Suggestion: 5  
**Gate:** Pass

### Summary

The change set matches EP-025: job flows live under `tests/e2e` with `//go:build e2e`, a non-e2e package surface remains via `!e2e` files, `DeliveryRunner` is implemented in `internal/jobs` and wired from `cmd/pa`, Makefile targets and default `coverage` (integration-only tags) align with the requirements, and CI documents the split between `coverage.out` and e2e execution. **Approve** for merge from a stage-10 perspective; remaining notes are nits and optional follow-ups only. Stage 11 recorded `make check` (exit 0) and `./bin/validate EP-025` (8/8 ACs) in [ep-audit-report.md](ep-audit-report.md).

### Findings

| Severity | Location | Issue | Recommendation |
|----------|----------|-------|------------------|
| Nit | `tests/e2e/ep025_policy_test.go` — `TestCIWorkflowMentionsE2ELayer` | Assertion is only that `ci.yml` contains the substring `e2e` (case-insensitive); it does not assert the step summary documents the coverage split as in REQ-25.005. | Optionally tighten the check (e.g. require phrases such as `coverage-e2e` or “separately” / “default” in the summary block) if the test should track the full AC wording. |
| Nit | `tests/e2e/jobs_e2e_test.go` — `mustListJobID` | `return ""` after `t.Fatalf` is redundant for humans but satisfies `go vet` on all return paths. | Keep comment explaining vet; optional refactor to a helper that returns `(string, bool)` if linters later flag it. |
| Nit | `Makefile` — `help` target | `make test` is described as “all tests” but e2e is exercised via `test-e2e` / `check`, which can confuse newcomers. | Optionally clarify in help text that default `test` / `test-race` do not include the `e2e` tag. |
| Suggestion | `cmd/pa/jobs_runtime_test.go` vs `internal/jobs/delivery_runner_test.go` | Overlapping coverage for `NewDeliveryRunner` behaviour in two packages. | Consider concentrating behaviour tests in `internal/jobs` unless `package main` coupling is intentional. |
| Suggestion | `internal/jobs/delivery_runner.go` — `classifyJobFailure` | `context.DeadlineExceeded` is handled without a dedicated unit test. | Optional table-driven test for the timeout branch. |
| Suggestion | `internal/jobs/delivery_runner.go` — success path when `Sender != nil` | If `SendMessageToChat` fails after a successful handler run, `Run` still returns `nil`. | If “delivered to chat” matters for ops metrics, consider propagating send failures or document best-effort notify explicitly. |
| Suggestion | `Makefile` — `check` / `test-race` vs `test-e2e` | `test-race` uses `-tags=integration` only; e2e runs in a separate `go test` without `-race`. | Optional `go test -race` for `./tests/e2e/...` or a Makefile comment explaining the split. |
| Suggestion | `Makefile` — `coverage-e2e` | `-coverpkg=./tests/e2e/...` measures only the test package. | Optionally widen `-coverpkg` if e2e runs should reflect production code coverage. |

### Test / verification

Subagent spot-checks (exit 0): `go test` for `./internal/jobs/...`, `./cmd/pa/...`, `./tests/e2e/...` with `integration` and with `integration,e2e`.

Epic gate: `make check` and `./bin/validate EP-025` (see [ep-audit-report.md](ep-audit-report.md)).

### Residual risks / follow-ups

E2E tests remain in-process against `internal/jobs` (SQLite + runtime + manager) rather than a subprocess/binary harness; that matches the relocated tests but is not full black-box system testing—acceptable if epic intent is “tagged long job flows,” not “only `bin/pa`.”
