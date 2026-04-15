# EP-017 — Audit Report

**Date:** 2026-04-15
**Pipeline:** Stage 11 ([pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md))
**Epic:** [ep-scope.md](ep-scope.md) · [ep-implementation-plan.md](ep-implementation-plan.md) · [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-requirements.md](ep-requirements.md) · [ep-system-design.md](ep-system-design.md) · [ep-code-review.md](ep-code-review.md)

---

## Summary

**EP-017 Intent Classifier for Prompt Optimization** is fully implemented. All 10 tasks and 3 checkpoints from the implementation plan are done. All 18 acceptance criteria have automated test coverage (100%). `make check` passes (fmt, vet, vuln, lint 0 issues, tests -race, coverage 73.8%, module boundaries OK). Code review passed (iteration 2, gate pass, zero Blocker/Major/Medium/Minor). **Quality gate: PASS.**

---

## Implementation vs plan

| Task | Status | Notes |
|------|--------|-------|
| 1 — Tier type and Classifier interface | Done | `internal/intent/tier.go` |
| 2 — HeuristicClassifier | Done | `internal/intent/heuristic.go` |
| 3 — ModelClassifier | Done | `internal/intent/model.go` |
| 4 — CascadeClassifier | Done | `internal/intent/cascade.go` |
| Checkpoint A | Done | `go test ./internal/intent/...` — 22 pass |
| 5 — Config types and validation | Done | `internal/config/config.go`, `load.go`, `resolve.go` |
| 6 — Wiring cmd/pa + core.Run | Done | `cmd/pa/main.go`, `internal/core/run.go` |
| 7 — HandleMessage tier-based prompt assembly | Done | `internal/core/handler.go` |
| Checkpoint B | Done | config + core + intent — all pass |
| 8 — Observability token logging | Done | Verified: model-stage tokens logged separately, excluded from footer |
| 9 — Documentation update | Done | `docs/configuration.md` |
| 10 — AC coverage and final validation | Done | `./bin/validate EP-017` exit 0, `make check` exit 0 |
| Checkpoint C | Done | `make build && ./bin/validate EP-017 && make check` — all pass |

---

## Test results and coverage

**Command:** `make check`
**Result:** PASS (exit 0)

| Check | Result |
|-------|--------|
| `go fmt` | No changes |
| `go vet` | Clean |
| `govulncheck` | No vulnerabilities |
| `golangci-lint` | 0 issues |
| `go test -race -tags=integration` | All packages pass |
| `coverage` | **total: 73.8% of statements** |
| `module boundaries` | OK (no cycles, no forbidden edges) |

**AC coverage:** `./bin/validate EP-017` — 18/18 AC traced (100%), 18 automated (100%), 0 manual-only, 0 deferred.

---

## REQ/AC test coverage matrix

| AC | REQ | Unit | Link |
|----|-----|------|------|
| [AC-17.001](ep-acceptance-criteria.md#ac-17-001) | [REQ-17.001](ep-requirements.md#req-17-001) | ✓ | `intent/tier_test.go`, `intent/cascade_test.go` |
| [AC-17.002](ep-acceptance-criteria.md#ac-17-002) | [REQ-17.002](ep-requirements.md#req-17-002) | ✓ | `core/handler_ep017_test.go` |
| [AC-17.003](ep-acceptance-criteria.md#ac-17-003) | [REQ-17.003](ep-requirements.md#req-17-003) | ✓ | `core/handler_ep017_test.go` |
| [AC-17.004](ep-acceptance-criteria.md#ac-17-004) | [REQ-17.004](ep-requirements.md#req-17-004), [REQ-17.005](ep-requirements.md#req-17-005) | ✓ | `intent/heuristic_test.go` (2 tests) |
| [AC-17.005](ep-acceptance-criteria.md#ac-17-005) | [REQ-17.004](ep-requirements.md#req-17-004), [REQ-17.005](ep-requirements.md#req-17-005) | ✓ | `intent/heuristic_test.go` (2 tests) |
| [AC-17.006](ep-acceptance-criteria.md#ac-17-006) | [REQ-17.005](ep-requirements.md#req-17-005) | ✓ | `intent/heuristic_test.go` |
| [AC-17.007](ep-acceptance-criteria.md#ac-17-007) | [REQ-17.006](ep-requirements.md#req-17-006) | ✓ | `intent/heuristic_test.go` |
| [AC-17.008](ep-acceptance-criteria.md#ac-17-008) | [REQ-17.007](ep-requirements.md#req-17-007), [REQ-17.009](ep-requirements.md#req-17-009) | ✓ | `intent/model_test.go` (2 tests) |
| [AC-17.009](ep-acceptance-criteria.md#ac-17-009) | [REQ-17.008](ep-requirements.md#req-17-008) | ✓ | `config/intent_classifier_test.go`, `intent/model_test.go` (4 tests) |
| [AC-17.010](ep-acceptance-criteria.md#ac-17-010) | [REQ-17.010](ep-requirements.md#req-17-010) | ✓ | `intent/cascade_test.go` (4 tests) |
| [AC-17.011](ep-acceptance-criteria.md#ac-17-011) | [REQ-17.011](ep-requirements.md#req-17-011) | ✓ | `intent/cascade_test.go`, `intent/model_test.go` (5 tests) |
| [AC-17.012](ep-acceptance-criteria.md#ac-17-012) | [REQ-17.012](ep-requirements.md#req-17-012), [REQ-17.013](ep-requirements.md#req-17-013), [REQ-17.014](ep-requirements.md#req-17-014) | ✓ | `core/handler_ep017_test.go` |
| [AC-17.013](ep-acceptance-criteria.md#ac-17-013) | [REQ-17.015](ep-requirements.md#req-17-015) | ✓ | `core/handler_ep017_test.go` |
| [AC-17.014](ep-acceptance-criteria.md#ac-17-014) | [REQ-17.016](ep-requirements.md#req-17-016) | ✓ | `config/intent_classifier_test.go`, `core/handler_ep017_test.go` (3 tests) |
| [AC-17.015](ep-acceptance-criteria.md#ac-17-015) | [REQ-17.016](ep-requirements.md#req-17-016) | ✓ | `config/intent_classifier_test.go` (2 tests) |
| [AC-17.016](ep-acceptance-criteria.md#ac-17-016) | [REQ-17.017](ep-requirements.md#req-17-017) | ✓ | `intent/observability_test.go`, `intent/cascade_test.go`, `core/handler_ep017_test.go` (3 tests) |
| [AC-17.017](ep-acceptance-criteria.md#ac-17-017) | [REQ-17.018](ep-requirements.md#req-17-018) | ✓ | `intent/observability_test.go`, `core/handler_ep017_test.go` (2 tests) |
| [AC-17.018](ep-acceptance-criteria.md#ac-17-018) | [REQ-17.019](ep-requirements.md#req-17-019), [REQ-17.020](ep-requirements.md#req-17-020) | ✓ | `config/intent_classifier_test.go` (docs check) |

**Note:** All tests are white-box unit tests (`package core`, `package intent`, `package config`) with mock providers. No `tests/integration/` tests with build tag `integration` were added for EP-017.

---

## Quality gate

| Gate | Result |
|------|--------|
| Code review (Stage 10) | Pass — iteration 2, zero Blocker/Major/Medium/Minor |
| `make check` | Pass — 0 lint issues, all tests pass, 73.8% coverage |
| `./bin/validate EP-017` | Pass — 18/18 AC covered (100%) |
| **Overall** | **PASS** |

---

## Gaps, risks, recommendations

| Type | Item |
|------|------|
| **Risk** | Classification prompt injection (code review Suggestion #4) — crafted user message could influence cheap model response. Impact: low (wrong tier → defaults to full). Track for future security hardening. |
| **Risk** | `core.Run()` has 14 positional params (code review Suggestion #5). Future refactor to Options struct recommended. |
| **Risk** | No HTTP-level integration test for model stage (only mock provider). Low risk since `llm.Provider` abstracts HTTP. |
| **Recommendation** | After merge, add `intent_classifier` section to production `config.json` and verify e2e with real Telegram traffic. Start with heuristic-only (`model_stage.enabled: false`). |
| **Recommendation** | Monitor classification accuracy via INFO logs; tune patterns based on real message distribution. |
