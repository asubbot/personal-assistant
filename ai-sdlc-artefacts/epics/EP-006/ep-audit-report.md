# EP-006 — Audit report

**Date and time of creation:** 2026-03-16 (audit after REQ-06.015 / REQ-06.016 and re-run of skills 4–9)

**Pipeline:** Stage 9 ([09-audit.skill.md](../../../ai-sdlc/specification/skills/09-audit.skill.md)).

**References:** [ep-implementation-plan.md](ep-implementation-plan.md) · [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-requirements.md](ep-requirements.md) · [ep-system-design.md](ep-system-design.md)

---

## Summary

**PASS:** `make check` completed successfully (fmt, vet, golangci-lint, `go test -tags=integration ./...`, module boundaries). EP-006 includes: config and chain vs transport, **typed** tool failures (`internal/core/toolfailure`, `noderunner` + `executeOneToolCall` wrapping), per-message escalation in the tool loop, **Hermes parse escalation** (`finishAfterFirstLLM`, `resolveHermesFollowUpCompletion`), structured logs with `failure_class` (`tool_execution`, `hermes_parse`). Artefacts [ep-requirements.md](ep-requirements.md) through [ep-implementation-plan.md](ep-implementation-plan.md) updated for REQ-06.015–016 and AC-06.012–013. Task 7 of the implementation plan remains **partial** (optional 3-provider integration test, README snippet).

---

## Implementation vs plan

| Task | Status | Notes |
|------|--------|-------|
| 1 Config `llm_escalation` + validation | Done | `config.LLMEscalationConfig`, `validateLLMEscalation` in [load.go](../../../internal/config/load.go) |
| 2 `cmd/pa` chain vs transport | Done | `buildLLMProviders`, `newLLMForConversation`, `logLLMStartupInfo` |
| 3 `core.Run` + handler dual mode | Done | [run.go](../../../internal/core/run.go), [handler.go](../../../internal/core/handler.go) |
| 4 Typed tool failure package | Done | [toolfailure/failure.go](../../../internal/core/toolfailure/failure.go); [noderunner/runner.go](../../../internal/noderunner/runner.go); `wrapCatalogValidateError` in handler |
| 5 Per-message state + Hermes + tool loop | Done | `llmTurnState`, `maybeEscalate(..., failureClass)`, `resolveHermesFollowUpCompletion` |
| 6 Observability logs | Done | `slog.InfoContext` `"llm tool escalation"` with `failure_class` |
| 7 Integration tests + docs | Partial | Unit tests in `toolfailure`; optional multi-provider E2E / README deferred |
| 8 `internal/escalationpolicy` | **Not started** | Documented in [ep-implementation-plan.md](ep-implementation-plan.md); [REQ-06.017](ep-requirements.md#nfr--security-testability-observability), [AC-06.014](ep-acceptance-criteria.md#ac-06-014) |

---

## Test results and coverage

- **Command:** `make check`
- **Result:** PASS (0 lint issues)
- **Total statement coverage:** see latest `make check` output (≈ **75.9%** total line at last run)

---

## REQ/AC test coverage matrix

| AC | REQ (primary) | Unit | Integration | E2E | Manual | Link / notes |
|----|---------------|------|-------------|-----|--------|----------------|
| [AC-06.001](ep-acceptance-criteria.md#ac-06-001) | [REQ-06.001](ep-requirements.md#baseline-and-configuration) | — | — | — | — | Baseline index wiring; dedicated baseline≠0 test optional |
| [AC-06.002](ep-acceptance-criteria.md#ac-06-002) | [REQ-06.002](ep-requirements.md#baseline-and-configuration) | — | — | — | — | Validated at `Load`; optional invalid `llm_escalation` JSON fixture |
| [AC-06.003](ep-acceptance-criteria.md#ac-06-003) | [REQ-06.003](ep-requirements.md#error-classification), [REQ-06.004](ep-requirements.md#error-classification) | — | — | — | — | Policy encoded in typed `Failure` + Hermes path |
| [AC-06.004](ep-acceptance-criteria.md#ac-06-004) | [REQ-06.005](ep-requirements.md#error-classification) | ✓ | — | — | — | [failure_test.go](../../../internal/core/toolfailure/failure_test.go) `TestQualifiesForEscalation_policyErrors` |
| [AC-06.005](ep-acceptance-criteria.md#ac-06-005) | [REQ-06.006](ep-requirements.md#escalation-policy-and-chain) | — | — | — | — | Handler logic; mock-chain integration recommended |
| [AC-06.006](ep-acceptance-criteria.md#ac-06-006) | [REQ-06.007](ep-requirements.md#escalation-policy-and-chain) | — | — | — | — | Deferred: explicit 3-provider order test |
| [AC-06.007](ep-acceptance-criteria.md#ac-06-007) | [REQ-06.008](ep-requirements.md#exhaustion-and-stop) | — | — | — | — | Caps: `max_per_user_message`, tool rounds, chain end |
| [AC-06.008](ep-acceptance-criteria.md#ac-06-008) | [REQ-06.009](ep-requirements.md#rollback-at-end-of-turn) | — | — | — | — | **Gap:** automated second `HandleMessage` → baseline |
| [AC-06.009](ep-acceptance-criteria.md#ac-06-009) | [REQ-06.010](ep-requirements.md#observability)–[REQ-06.012](ep-requirements.md#nfr--security-testability-observability) | — | — | — | — | Fields implemented; no assertion on log line shape |
| [AC-06.010](ep-acceptance-criteria.md#ac-06-010) | [REQ-06.013](ep-requirements.md#nfr--security-testability-observability) | ✓ | — | — | — | [run_test.go](../../../internal/core/run_test.go), `toolfailure` tests |
| [AC-06.011](ep-acceptance-criteria.md#ac-06-011) | [REQ-06.014](ep-requirements.md#nfr--security-testability-observability) | — | — | — | — | Escalation off → `FallbackProvider` |
| [AC-06.012](ep-acceptance-criteria.md#ac-06-012) | [REQ-06.015](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs) | ✓ | — | — | — | [failure_test.go](../../../internal/core/toolfailure/failure_test.go) `TestQualifiesForEscalation_untypedFailsClosed` |
| [AC-06.013](ep-acceptance-criteria.md#ac-06-013) | [REQ-06.016](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs) | — | — | — | — | **Gap:** mock handler test for Hermes retry + `completeAt` order |

**Notes:** `wrapCatalogValidateError` still uses a substring check on **catalog** errors only to choose `NoEscalate` vs `MayEscalate` when building typed `Failure`; escalation **qualification** itself is via `errors.As` on `Failure` (REQ-06.015). To remove substring use entirely, add sentinels in `toolcatalog` in a follow-up.

---

## Quality gate

- golangci-lint: **PASS**
- go vet: **PASS**
- Module boundaries check: **PASS** (`noderunner` → `core/toolfailure` allowed)

---

## Gaps, risks, recommendations

| Type | Item |
|------|------|
| **Gap** | [AC-06.008](ep-acceptance-criteria.md#ac-06-008): two `HandleMessage` calls with escalation, assert second uses `baseline_index`. |
| **Gap** | [AC-06.013](ep-acceptance-criteria.md#ac-06-013): unit/integration test with mock LLM returning bad then valid Hermes. |
| **Gap** | [AC-06.005](ep-acceptance-criteria.md#ac-06-005) / [AC-06.006](ep-acceptance-criteria.md#ac-06-006): mock provider chain + qualifying typed failure. |
| **Follow-up** | Replace catalog substring bridge with exported sentinel errors from `toolcatalog`. |
| **Recommendation** | README / sample `llm_escalation` + Hermes behaviour note for operators. |
