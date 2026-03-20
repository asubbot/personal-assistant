# EP-006 — Audit report

**Date and time of creation:** 2026-03-20 11:37 UTC (stage 9 epic audit; `make check` on same calendar date)

**Pipeline:** Stage 9 ([09-audit.skill.md](../../../ai-sdlc/specification/skills/09-audit.skill.md)).

**References:** [ep-implementation-plan.md](ep-implementation-plan.md) · [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-requirements.md](ep-requirements.md) · [ep-system-design.md](ep-system-design.md) · [ep-manual-tests.md](ep-manual-tests.md)

---

## Summary

**PASS:** `make check` succeeded (fmt, vet, golangci-lint, `go test -tags=integration ./...`, module boundaries). Escalation policy: `internal/escalationpolicy` covers catalog (`WrapCatalogValidateError` / `ValidateKind`) and noderunner (`WrapNodeOutcome`). Handler unit tests in [handler_ep006_audit_test.go](../../../internal/core/handler_ep006_audit_test.go) cover 3-provider order, max-escalation / last-provider stop, baseline per message (AC-06.001 + AC-06.008), escalation logs, escalation disabled with chain present, and Hermes parse escalation. Integration: [ep006_escalation_run_test.go](../../../tests/integration/ep006_escalation_run_test.go) including `TestEP006_Run_threeProviders_threeMessages_chainAndBaselineReset`. **README** documents `tools.llm_escalation`. **Manual:** [ep-manual-tests.md](ep-manual-tests.md) for operator sign-off in real environments.

---

## Implementation vs plan

| Task | Status | Notes |
|------|--------|-------|
| 1 Config `tools.llm_escalation` | Done | [load.go](../../../internal/config/load.go), [config.go](../../../internal/config/config.go) |
| 2 `cmd/pa` chain vs transport | Done | [main.go](../../../cmd/pa/main.go) |
| 3 `core.Run` + handler dual mode | Done | [run.go](../../../internal/core/run.go), [handler.go](../../../internal/core/handler.go) |
| 4 Typed `toolfailure` + paths | Done | [toolfailure/](../../../internal/core/toolfailure/); noderunner → [node.go](../../../internal/escalationpolicy/node.go); handler → `WrapCatalogValidateError` |
| 5 Per-message escalation + Hermes | Done | `maybeEscalate`, `resolveHermesFollowUpCompletion`, etc. |
| 6 Observability | Done | Structured `llm tool escalation` logs |
| 7 Integration tests + docs + manual | Done | [ep006_escalation_run_test.go](../../../tests/integration/ep006_escalation_run_test.go); [run_ep006_escalation_test.go](../../../internal/core/run_ep006_escalation_test.go); [ep-manual-tests.md](ep-manual-tests.md); README |
| 8 `internal/escalationpolicy` | Done | [catalog.go](../../../internal/escalationpolicy/catalog.go), [node.go](../../../internal/escalationpolicy/node.go), tests |
| 9 Catalog `ValidateKind` | Done | [toolcatalog/validate_error.go](../../../internal/toolcatalog/validate_error.go), [validate.go](../../../internal/toolcatalog/validate.go) |

---

## Test results and coverage

- **Command:** `make check`
- **Result:** PASS
- **Total statement coverage:** **78.3%** (from `go tool cover -func=coverage.out` total line)

---

## REQ/AC test coverage matrix

| AC | REQ (primary) | Unit | Integration | E2E | Manual | Link / notes |
|----|---------------|------|-------------|-----|--------|----------------|
| [AC-06.001](ep-acceptance-criteria.md#ac-06-001) | [REQ-06.001](ep-requirements.md#baseline-and-configuration) | ✓ | ✓ | — | — | [handler_ep006_audit_test.go](../../../internal/core/handler_ep006_audit_test.go) `TestHandleMessage_escalation_eachMessageStartsFromBaseline`; [run_ep006_escalation_test.go](../../../internal/core/run_ep006_escalation_test.go) |
| [AC-06.002](ep-acceptance-criteria.md#ac-06-002) | [REQ-06.002](ep-requirements.md#baseline-and-configuration) | ✓ | — | — | — | [config_test.go](../../../internal/config/config_test.go) |
| [AC-06.003](ep-acceptance-criteria.md#ac-06-003) | [REQ-06.003](ep-requirements.md#error-classification), [REQ-06.004](ep-requirements.md#error-classification) | ✓ | — | — | — | [handler_ep006_audit_test.go](../../../internal/core/handler_ep006_audit_test.go); `toolfailure` / `escalationpolicy` |
| [AC-06.004](ep-acceptance-criteria.md#ac-06-004) | [REQ-06.005](ep-requirements.md#error-classification) | ✓ | — | — | — | [failure_test.go](../../../internal/core/toolfailure/failure_test.go); [catalog_test.go](../../../internal/escalationpolicy/catalog_test.go); handler catalog path tests |
| [AC-06.005](ep-acceptance-criteria.md#ac-06-005) | [REQ-06.006](ep-requirements.md#escalation-policy-and-chain) | ✓ | ✓ | — | — | [handler_ep006_audit_test.go](../../../internal/core/handler_ep006_audit_test.go); [ep006_escalation_run_test.go](../../../tests/integration/ep006_escalation_run_test.go) |
| [AC-06.006](ep-acceptance-criteria.md#ac-06-006) | [REQ-06.007](ep-requirements.md#escalation-policy-and-chain) | ✓ | ✓ | — | — | `TestHandleMessage_escalation_threeProviders_secondReceivesNextComplete`; integration EP-006 tests |
| [AC-06.007](ep-acceptance-criteria.md#ac-06-007) | [REQ-06.008](ep-requirements.md#exhaustion-and-stop) | ✓ | ✓ | — | — | `TestHandleMessage_escalation_maxZero_noAdvance`, `TestHandleMessage_escalation_atLastProvider_noFurtherAdvance`; integration where applicable |
| [AC-06.008](ep-acceptance-criteria.md#ac-06-008) | [REQ-06.009](ep-requirements.md#rollback-at-end-of-turn) | ✓ | ✓ | — | ✓ | `TestHandleMessage_escalation_eachMessageStartsFromBaseline`; `TestEP006_Run_twoMessages_resetsBaselineAfterEscalation`; `TestEP006_Run_threeProviders_threeMessages_chainAndBaselineReset`; [ep-manual-tests.md#mt-baseline-reset](ep-manual-tests.md#mt-baseline-reset) |
| [AC-06.009](ep-acceptance-criteria.md#ac-06-009) | [REQ-06.010](ep-requirements.md#observability)–[REQ-06.012](ep-requirements.md#nfr--security-testability-observability) | ✓ | — | — | ✓ | `TestHandleMessage_escalation_logContainsPolicyFields`; [ep-manual-tests.md#mt-no-secrets](ep-manual-tests.md#mt-no-secrets) for real secrets check |
| [AC-06.010](ep-acceptance-criteria.md#ac-06-010) | [REQ-06.013](ep-requirements.md#nfr--security-testability-observability) | ✓ | ✓ | — | ✓ | Unit + integration EP-006 suite; [ep-manual-tests.md#mt-operator](ep-manual-tests.md#mt-operator) |
| [AC-06.011](ep-acceptance-criteria.md#ac-06-011) | [REQ-06.014](ep-requirements.md#nfr--security-testability-observability) | ✓ | — | — | ✓ | `TestHandleMessage_escalationDisabled_qualifyingToolFailure_staysOnFirstProvider`; [ep-manual-tests.md#mt-esc-off](ep-manual-tests.md#mt-esc-off) |
| [AC-06.012](ep-acceptance-criteria.md#ac-06-012) | [REQ-06.015](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs) | ✓ | — | — | — | [failure_test.go](../../../internal/core/toolfailure/failure_test.go); [node_test.go](../../../internal/escalationpolicy/node_test.go); [runner_test.go](../../../internal/noderunner/runner_test.go) smoke |
| [AC-06.013](ep-acceptance-criteria.md#ac-06-013) | [REQ-06.016](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs) | ✓ | — | — | ✓ | `TestHandleMessage_textBasedHermes_parseFailure_escalatesToNextProvider`; [ep-manual-tests.md#mt-hermes](ep-manual-tests.md#mt-hermes) |
| [AC-06.014](ep-acceptance-criteria.md#ac-06-014) | [REQ-06.017](ep-requirements.md#nfr--security-testability-observability) | ✓ | — | — | — | [catalog_test.go](../../../internal/escalationpolicy/catalog_test.go); [node_test.go](../../../internal/escalationpolicy/node_test.go) |
| [AC-06.015](ep-acceptance-criteria.md#ac-06-015) | [REQ-06.018](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs) | ✓ | — | — | — | [validate_test.go](../../../internal/toolcatalog/validate_test.go); [catalog_test.go](../../../internal/escalationpolicy/catalog_test.go) |

**Notes:** **Unit** = `go test ./internal/...`; **Integration** = `tests/integration` (build tag `integration`). **Manual** = documented scenarios in [ep-manual-tests.md](ep-manual-tests.md) (not automated).

---

## Quality gate

- golangci-lint: **PASS** (0 issues)
- go vet: **PASS**
- Module boundaries: **PASS** (`escalationpolicy` → `core/toolfailure`, `toolcatalog`; no cycles)

---

## Gaps, risks, recommendations

| Type | Item |
|------|------|
| **Observation** | [REQ-06.011](ep-requirements.md#observability) (*tried_providers* summary): optional; handler tests note it may be absent — no functional gap. |
| **Recommendation** | Before production: run selected scenarios in [ep-manual-tests.md](ep-manual-tests.md) (escalation on/off, secrets in logs). |
| **Recommendation** | [ep-scope.md](ep-scope.md) remains **IN PROGRESS** until epic closure is decided after operator sign-off. |
