# EP-006 — Audit report

**Date and time of creation:** 2026-03-16 (UTC) (stage 9 epic audit run via `make check`)

**Pipeline:** Stage 9 ([09-audit.skill.md](../../../ai-sdlc/specification/skills/09-audit.skill.md)).

**References:** [ep-implementation-plan.md](ep-implementation-plan.md) · [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-requirements.md](ep-requirements.md) · [ep-system-design.md](ep-system-design.md)

---

## Summary

**PASS:** `make check` succeeded (fmt, vet, golangci-lint, `go test -tags=integration ./...`, module boundaries). Escalation policy uses `ValidateKind`; **handler unit tests** in [handler_ep006_audit_test.go](../../../internal/core/handler_ep006_audit_test.go) cover 3-provider order, max-escalation / last-provider stop, baseline reset per message, escalation log shape, escalation disabled with chain present, and Hermes parse escalation. **README** documents `tools.llm_escalation`. **Optional:** dedicated `tests/integration` mock chain across two `HandleMessage` calls remains deferred.

---

## Implementation vs plan

| Task | Status | Notes |
|------|--------|-------|
| 1 Config `tools.llm_escalation` | Done | [load.go](../../../internal/config/load.go), [config.go](../../../internal/config/config.go) |
| 2 `cmd/pa` chain vs transport | Done | [main.go](../../../cmd/pa/main.go) |
| 3 `core.Run` + handler dual mode | Done | [run.go](../../../internal/core/run.go), [handler.go](../../../internal/core/handler.go) |
| 4 Typed `toolfailure` | Done | [toolfailure/](../../../internal/core/toolfailure/), [noderunner/runner.go](../../../internal/noderunner/runner.go) |
| 5 Per-message escalation + Hermes | Done | Handler `maybeEscalate`, `resolveHermesFollowUpCompletion`, etc. |
| 6 Observability | Done | Structured `llm tool escalation` logs |
| 7 Integration tests + docs | Done (partial) | Unit coverage in `handler_ep006_audit_test.go`; README; optional integration chain test still open |
| 8 `internal/escalationpolicy` | Done | [escalationpolicy/catalog.go](../../../internal/escalationpolicy/catalog.go), [catalog_test.go](../../../internal/escalationpolicy/catalog_test.go) |
| 9 Catalog `ValidateKind` | Done | [toolcatalog/validate_error.go](../../../internal/toolcatalog/validate_error.go), [validate.go](../../../internal/toolcatalog/validate.go) |

---

## Test results and coverage

- **Command:** `make check`
- **Result:** PASS
- **Total statement coverage:** **78.0%** (from `go tool cover -func=coverage.out` total line)

---

## REQ/AC test coverage matrix

| AC | REQ (primary) | Unit | Integration | E2E | Manual | Link / notes |
|----|---------------|------|-------------|-----|--------|----------------|
| [AC-06.001](ep-acceptance-criteria.md#ac-06-001) | [REQ-06.001](ep-requirements.md#baseline-and-configuration) | ✓ | — | — | — | [run_test.go](../../../internal/core/run_test.go) `TestRun_llmChain_wiresHandler` |
| [AC-06.002](ep-acceptance-criteria.md#ac-06-002) | [REQ-06.002](ep-requirements.md#baseline-and-configuration) | ✓ | — | — | — | [config_test.go](../../../internal/config/config_test.go) |
| [AC-06.003](ep-acceptance-criteria.md#ac-06-003) | [REQ-06.003](ep-requirements.md#error-classification), [REQ-06.004](ep-requirements.md#error-classification) | ✓ | — | — | — | [handler_ep006_audit_test.go](../../../internal/core/handler_ep006_audit_test.go) `TestEP006_classification_QualifiesForEscalation_table`; plus `toolfailure` / `escalationpolicy` |
| [AC-06.004](ep-acceptance-criteria.md#ac-06-004) | [REQ-06.005](ep-requirements.md#error-classification) | ✓ | — | — | — | [failure_test.go](../../../internal/core/toolfailure/failure_test.go); [catalog_test.go](../../../internal/escalationpolicy/catalog_test.go) |
| [AC-06.005](ep-acceptance-criteria.md#ac-06-005) | [REQ-06.006](ep-requirements.md#escalation-policy-and-chain) | ✓ | — | — | — | [run_test.go](../../../internal/core/run_test.go); [handler_ep006_audit_test.go](../../../internal/core/handler_ep006_audit_test.go) escalation tests |
| [AC-06.006](ep-acceptance-criteria.md#ac-06-006) | [REQ-06.007](ep-requirements.md#escalation-policy-and-chain) | ✓ | — | — | — | `TestHandleMessage_escalation_threeProviders_secondReceivesNextComplete` |
| [AC-06.007](ep-acceptance-criteria.md#ac-06-007) | [REQ-06.008](ep-requirements.md#exhaustion-and-stop) | ✓ | — | — | — | `TestHandleMessage_escalation_maxZero_noAdvance`, `TestHandleMessage_escalation_atLastProvider_noFurtherAdvance` |
| [AC-06.008](ep-acceptance-criteria.md#ac-06-008) | [REQ-06.009](ep-requirements.md#rollback-at-end-of-turn) | ✓ | — | — | — | `TestHandleMessage_escalation_eachMessageStartsFromBaseline` |
| [AC-06.009](ep-acceptance-criteria.md#ac-06-009) | [REQ-06.010](ep-requirements.md#observability)–[REQ-06.012](ep-requirements.md#nfr--security-testability-observability) | ✓ | — | — | — | `TestHandleMessage_escalation_logContainsPolicyFields` (structured fields); no-secrets spot-check via synthetic error text only |
| [AC-06.010](ep-acceptance-criteria.md#ac-06-010) | [REQ-06.013](ep-requirements.md#nfr--security-testability-observability) | ✓ | ✓ | — | — | Unit + integration suite; not every branch explicit |
| [AC-06.011](ep-acceptance-criteria.md#ac-06-011) | [REQ-06.014](ep-requirements.md#nfr--security-testability-observability) | ✓ | — | — | — | `TestHandleMessage_escalationDisabled_qualifyingToolFailure_staysOnFirstProvider` |
| [AC-06.012](ep-acceptance-criteria.md#ac-06-012) | [REQ-06.015](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs) | ✓ | — | — | — | [failure_test.go](../../../internal/core/toolfailure/failure_test.go) |
| [AC-06.013](ep-acceptance-criteria.md#ac-06-013) | [REQ-06.016](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs) | ✓ | — | — | — | `TestHandleMessage_textBasedHermes_parseFailure_escalatesToNextProvider` |
| [AC-06.014](ep-acceptance-criteria.md#ac-06-014) | [REQ-06.017](ep-requirements.md#nfr--security-testability-observability) | ✓ | — | — | — | [catalog_test.go](../../../internal/escalationpolicy/catalog_test.go) |
| [AC-06.015](ep-acceptance-criteria.md#ac-06-015) | [REQ-06.018](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs) | ✓ | — | — | — | [validate_test.go](../../../internal/toolcatalog/validate_test.go); [catalog_test.go](../../../internal/escalationpolicy/catalog_test.go) |

**Notes:** **Unit** = `go test ./internal/...`; **Integration** = `tests/integration` (tag `integration`).

---

## Quality gate

- golangci-lint: **PASS**
- go vet: **PASS**
- Module boundaries: **PASS** (`escalationpolicy` → `core/toolfailure`, `toolcatalog`; no cycles)

---

## Gaps, risks, recommendations

| Type | Item |
|------|------|
| **Gap (optional)** | End-to-end `tests/integration`: two `HandleMessage` calls with real config load + mock chain (cross-package). |
| **Recommendation** | Checkpoint: operator review of `tools.llm_escalation` defaults before merge. |
