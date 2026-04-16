# EP-021 — Audit report (stage 11)

**Date and time of creation:** 2026-04-16 (UTC)  
**Purpose:** Close EP-021 — scheduler routing without a Telegram-only gate; explicit `create_scheduled_job`; optional runtime skill template; extended test traceability for config, tool validation, and runtime defaults.  
**References:** [ep-implementation-plan.md](ep-implementation-plan.md), [ep-acceptance-criteria.md](ep-acceptance-criteria.md), [ep-requirements.md](ep-requirements.md)

## Summary

**PASS.** All implementation-plan tasks remain complete. `make check` passes. `./bin/validate EP-021` passes: **6/6** in-scope ACs traced to automated tests (**100%**); **AC-21.006** remains **DEFERRED / MANUAL** (static `systemStaticHead` diff). Project total statement coverage at this audit run: **73.8%** (from `make check` summary line).

## Implementation vs plan

| Task | Status |
|------|--------|
| 1. Config allowlist (`create_scheduled_job` when `jobs_db_path` set) | Done |
| 2. Optional example skill `config.examples/skills/scheduled-jobs/` | Done |
| 3. Native tool explicit params, `native_tool_explicit` | Done |
| 4. Manager: remove NL create / regex | Done |
| 5. Jobs wrapper: delegate only | Done |
| 6. Tests + AC traces ([REQ-21.012](ep-requirements.md#requirements)) | Done — includes `internal/config/runtime_skills_test.go`, extra `internal/jobs/manager_test.go` / `runtime_test.go` cases |
| 7. `docs/configuration.md` | Done |
| 8. `make check`, `./bin/validate EP-021` | Done |

## Test results and coverage

- **Command:** `make check` — **Pass** (fmt, vet, vuln, lint, race tests, integration, coverage, module boundaries).
- **Total statement coverage:** **73.8%** (`total: (statements)` from coverage summary).
- **AC validation:** `./bin/validate EP-021` — **Pass** (deferred AC-21.006 excluded from in-scope automated count per validator).

## REQ / AC test coverage matrix

Legend: **U** = unit (Go `_test.go` in product packages), **I** = integration (`tests/integration`), **E** = e2e-style (`cmd/pa` long-flow tests). **M** = manual only.

| AC | REQ (primary) | U | I | E | M | Evidence |
|----|-----------------|---|---|---|---|------------|
| [AC-21.001](ep-acceptance-criteria.md#ac-21001) | [REQ-21.001](ep-requirements.md#requirements) | ✓ | — | — | — | `cmd/pa/jobs_runtime_test.go` |
| [AC-21.002](ep-acceptance-criteria.md#ac-21002) | [REQ-21.002](ep-requirements.md#requirements) | ✓ | — | — | — | `cmd/pa/jobs_runtime_test.go` |
| [AC-21.003](ep-acceptance-criteria.md#ac-21003) | [REQ-21.002](ep-requirements.md#requirements), [REQ-21.003](ep-requirements.md#requirements) | ✓ | — | — | — | `cmd/pa/jobs_runtime_test.go` |
| [AC-21.004](ep-acceptance-criteria.md#ac-21004) | [REQ-21.004](ep-requirements.md#requirements), [REQ-21.005](ep-requirements.md#requirements), [REQ-21.009](ep-requirements.md#requirements) | ✓ | — | ✓ | — | `internal/jobs/manager_test.go`, `cmd/pa/ep020_e2e_test.go` |
| [AC-21.005](ep-acceptance-criteria.md#ac-21005) | [REQ-21.010](ep-requirements.md#requirements) | ✓ | — | — | — | `internal/jobs/manager_test.go` |
| [AC-21.006](ep-acceptance-criteria.md#ac-21006) | [REQ-21.006](ep-requirements.md#requirements) | — | — | — | ✓ | **DEFERRED** — operator diff `systemStaticHead` / static personality in `internal/core/handler.go` |
| [AC-21.007](ep-acceptance-criteria.md#ac-21007) | [REQ-21.007](ep-requirements.md#requirements), [REQ-21.008](ep-requirements.md#requirements) | ✓ | — | — | — | `internal/runtimeskills/load_test.go`, `internal/config/runtime_skills_test.go` |

**REQ-21.012** (test traceability): satisfied by `Covers AC-21.*` / `Trace: REQ-21.*` comments on the tests above and validator **EP-021** pass.

## Quality gate

- `make check`: **Pass**
- `./bin/validate EP-021`: **Pass** (in-scope ACs traced; one deferred manual AC)

## Gaps / risks / recommendations

- **Gap:** **AC-21.006** still pending a manual operator diff when convenient (confirm no unintended edits to static system head strings).
- **Risk:** Low — NL create now depends on the main handler calling `create_scheduled_job`; monitor first deployments for model refusal or ambiguous times (operational, not a code defect).
- **Recommendation:** Keep the example skill path documented for operators who want extra playbook context only when `runtime_skills` is enabled.

## Epic status

**DONE** (recorded in [ep-scope.md](ep-scope.md)).
