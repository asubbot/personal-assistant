# Audit report — EP-018 Tiered Prompt Cost Reduction

**Date and time of creation:** 2026-04-15 (UTC)  
**Pipeline:** [pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md) stage 11  
**Plan:** [ep-implementation-plan.md](ep-implementation-plan.md)  
**Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)  
**Code review gate:** [ep-code-review.md](ep-code-review.md) — iteration 1 **Pass** (§2.2)

## Summary

EP-018 implementation is **complete** against the implementation plan: three-way intent classification, `full_lite` prompt assembly, optional dynamic tool capping for `full` and `full_lite`, configuration validation, observability, tests, documentation, and quality gates. **`make check`** and **`./bin/validate EP-018`** both exited **0**. Total statement coverage from `make check`: **74.0%** (project-wide aggregate).

## Implementation vs plan

| Task | Status | Notes |
|------|--------|--------|
| 1 — intent `TierFullLite` + classification | Done | `internal/intent/` |
| 2 — config + validation | Done | `internal/config/load.go`, `ep018_dynamic_tools_test.go` |
| 3 — picker / cap | Done | `pickToolsForMainRequest` + `ApplyDynamicToolCap` in `internal/core/dynamic_tool_selection.go` |
| 4–6 — `HandleMessage` tiers + dynamic | Done | `internal/core/handler.go`, `run.go` |
| 7 — simple regression + docs | Done | `Supporting AC-18.002` on EP-017 test; `docs/configuration.md` tier table |
| 8 — observability | Done | `main llm prompt assembled` INFO |
| 9 — rune regression | Done | `handler_ep018_test.go` |
| 10 — cmd/pa wiring | Done | `buildIntentClassifier` passes `FullLitePatterns` |
| 11–12 — validate + CI | Done | `./bin/validate EP-018`, `make check` |

Checkpoints A–D: **all satisfied** (see implementation plan).

## Test results and coverage

| Command | Result |
|---------|--------|
| `make check` | Pass (fmt, vet, vuln, lint, `go test -race`, coverage, module boundaries) |
| `./bin/validate EP-018` | Pass — 21/21 AC traced (20 automated, 1 manual: AC-18.016) |

**Total coverage (from `make check` / `go tool cover -func`):** 74.0% of statements.

## REQ/AC test coverage matrix

| AC | REQ | Unit | Integration | E2E | Manual | Link |
|----|-----|------|---------------|-----|--------|------|
| [AC-18.001](ep-acceptance-criteria.md#ac-18-001) | [REQ-18.001](ep-requirements.md#req-18-001) | ✓ | — | — | — | `internal/core/handler_ep018_coverage_test.go` |
| [AC-18.002](ep-acceptance-criteria.md#ac-18-002) | [REQ-18.002](ep-requirements.md#req-18-002) | ✓ | — | — | — | `internal/core/handler_ep017_test.go` |
| [AC-18.003](ep-acceptance-criteria.md#ac-18-003) | [REQ-18.003](ep-requirements.md#req-18-003) | ✓ | — | — | — | `internal/core/handler_ep018_coverage_test.go` |
| [AC-18.004](ep-acceptance-criteria.md#ac-18-004) | [REQ-18.004](ep-requirements.md#req-18-004) | ✓ | — | — | — | `internal/core/handler_ep018_test.go` |
| [AC-18.005](ep-acceptance-criteria.md#ac-18-005) | [REQ-18.005](ep-requirements.md#req-18-005) | ✓ | — | — | — | `internal/core/handler_ep018_coverage_test.go` |
| [AC-18.006](ep-acceptance-criteria.md#ac-18-006) | [REQ-18.006](ep-requirements.md#req-18-006) | ✓ | — | — | — | `internal/core/handler_ep018_coverage_test.go` |
| [AC-18.007](ep-acceptance-criteria.md#ac-18-007) | [REQ-18.007](ep-requirements.md#req-18-007) | ✓ | — | — | — | `internal/core/handler_ep018_coverage_test.go` |
| [AC-18.008](ep-acceptance-criteria.md#ac-18-008) | [REQ-18.008](ep-requirements.md#req-18-008) | ✓ | — | — | — | `internal/core/handler_ep018_coverage_test.go` |
| [AC-18.009](ep-acceptance-criteria.md#ac-18-009) | [REQ-18.009](ep-requirements.md#req-18-009) | ✓ | — | — | — | `internal/intent/heuristic_test.go` |
| [AC-18.010](ep-acceptance-criteria.md#ac-18-010) | [REQ-18.010](ep-requirements.md#req-18-010) | ✓ | — | — | — | `internal/intent/model_test.go` |
| [AC-18.011](ep-acceptance-criteria.md#ac-18-011) | [REQ-18.011](ep-requirements.md#req-18-011) | ✓ | — | — | — | `internal/intent/cascade_test.go` |
| [AC-18.012](ep-acceptance-criteria.md#ac-18-012) | [REQ-18.012](ep-requirements.md#req-18-012) | ✓ | — | — | — | `internal/core/dynamic_tool_selection_test.go` |
| [AC-18.013](ep-acceptance-criteria.md#ac-18-013) | [REQ-18.013](ep-requirements.md#req-18-013) | ✓ | — | — | — | `internal/core/dynamic_tool_selection_test.go` |
| [AC-18.014](ep-acceptance-criteria.md#ac-18-014) | [REQ-18.014](ep-requirements.md#req-18-014) | ✓ | — | — | — | `internal/core/handler_ep018_coverage_test.go` |
| [AC-18.015](ep-acceptance-criteria.md#ac-18-015) | [REQ-18.015](ep-requirements.md#req-18-015) | ✓ | — | — | — | `internal/core/handler_ep018_coverage_test.go` |
| [AC-18.016](ep-acceptance-criteria.md#ac-18-016) | [REQ-18.016](ep-requirements.md#req-18-016) | — | — | — | ✓ | `internal/core/handler_ep018_coverage_test.go` (manual trace) |
| [AC-18.017](ep-acceptance-criteria.md#ac-18-017) | [REQ-18.017](ep-requirements.md#req-18-017) | ✓ | — | — | — | `internal/core/handler_ep018_coverage_test.go` |
| [AC-18.018](ep-acceptance-criteria.md#ac-18-018) | [REQ-18.018](ep-requirements.md#req-18-018) | ✓ | — | — | — | `internal/core/handler_ep018_test.go` |
| [AC-18.019](ep-acceptance-criteria.md#ac-18-019) | [REQ-18.019](ep-requirements.md#req-18-019) | ✓ | — | — | — | `internal/config/ep018_dynamic_tools_test.go`, `internal/core/dynamic_tool_selection_test.go` |
| [AC-18.020](ep-acceptance-criteria.md#ac-18-020) | [REQ-18.004](ep-requirements.md#req-18-004), [REQ-18.006](ep-requirements.md#req-18-006), [REQ-18.013](ep-requirements.md#req-18-013) | ✓ | — | — | — | `internal/core/handler_ep018_test.go` |
| [AC-18.021](ep-acceptance-criteria.md#ac-18-021) | [REQ-18.020](ep-requirements.md#req-18-020), [REQ-18.021](ep-requirements.md#req-18-021) | ✓ | — | — | — | `internal/core/handler_ep018_coverage_test.go` (`go run` validate) |

**Notes:** “Unit” here means package tests under `internal/` and `cmd/` per [strategy.md](../../strategy.md). AC-18.016 is explicitly traced as manual-only (pre-selection disabled + fallback list).

## Quality gate

`make check`: **pass**. Module boundary script: **pass**.

## Gaps, risks, recommendations

- **Gap:** None blocking; AC-18.016 lacks an automated integration test for tool index disabled — acceptable as manual trace per validation.
- **Risk:** Operators must configure `full_lite` / `full` patterns carefully to avoid mis-tiering; mitigated by EP-017-style cascade and default `full` on model failure.
- **Recommendation:** After merge, update [ep-scope.md](ep-scope.md) **Status** from NEW to DONE when the owner closes the epic.
