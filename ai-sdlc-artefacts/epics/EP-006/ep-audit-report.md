# EP-006 — Audit report

**Date and time of creation:** 2026-03-19 (stage 9 after Task 8 + `make check`)

**Pipeline:** Stage 9 ([09-audit.skill.md](../../../ai-sdlc/specification/skills/09-audit.skill.md)).

**References:** [ep-implementation-plan.md](ep-implementation-plan.md) · [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-requirements.md](ep-requirements.md) · [ep-system-design.md](ep-system-design.md)

---

## Summary

**PASS:** `make check` succeeded (fmt, vet, golangci-lint, `go test -tags=integration ./...`, module boundaries). **Task 8** (`internal/escalationpolicy`) is **Done**: catalog validate errors are wrapped via `escalationpolicy.WrapCatalogValidateError` with fail-closed behaviour for unrecognized messages. **Task 7** remains **partial** (optional 3-provider integration test, README). Several AC still rely on implicit behaviour or manual verification (see matrix).

---

## Implementation vs plan

| Task | Status | Notes |
|------|--------|-------|
| 1 Config `llm_escalation` | Done | [load.go](../../../internal/config/load.go), [config.go](../../../internal/config/config.go) |
| 2 `cmd/pa` chain vs transport | Done | [main.go](../../../cmd/pa/main.go) |
| 3 `core.Run` + handler dual mode | Done | [run.go](../../../internal/core/run.go), [handler.go](../../../internal/core/handler.go) |
| 4 Typed `toolfailure` | Done | [toolfailure/](../../../internal/core/toolfailure/), [noderunner/runner.go](../../../internal/noderunner/runner.go) |
| 5 Per-message escalation + Hermes | Done | Handler `maybeEscalate`, `resolveHermesFollowUpCompletion`, etc. |
| 6 Observability | Done | Structured `llm tool escalation` logs |
| 7 Integration tests + docs | Partial | No dedicated 3-provider mock chain test; README optional |
| 8 `internal/escalationpolicy` | **Done** | [escalationpolicy/catalog.go](../../../internal/escalationpolicy/catalog.go), [catalog_test.go](../../../internal/escalationpolicy/catalog_test.go) |

---

## Test results and coverage

- **Command:** `make check`
- **Result:** PASS
- **Total statement coverage:** **76.0%** (from `go tool cover -func=coverage.out` total line)

---

## REQ/AC test coverage matrix

| AC | REQ (primary) | Unit | Integration | E2E | Manual | Link / notes |
|----|---------------|------|-------------|-----|--------|----------------|
| [AC-06.001](ep-acceptance-criteria.md#ac-06-001) | [REQ-06.001](ep-requirements.md#baseline-and-configuration) | ✓ | — | — | — | [run_test.go](../../../internal/core/run_test.go) `TestRun_llmChain_wiresHandler` — Covers AC-06.001 |
| [AC-06.002](ep-acceptance-criteria.md#ac-06-002) | [REQ-06.002](ep-requirements.md#baseline-and-configuration) | ✓ | — | — | — | [config_test.go](../../../internal/config/config_test.go) — Supporting |
| [AC-06.003](ep-acceptance-criteria.md#ac-06-003) | [REQ-06.003](ep-requirements.md#error-classification), [REQ-06.004](ep-requirements.md#error-classification) | — | — | — | — | Policy spread across `toolfailure`, `escalationpolicy`, handler; no single named test |
| [AC-06.004](ep-acceptance-criteria.md#ac-06-004) | [REQ-06.005](ep-requirements.md#error-classification) | ✓ | — | — | — | [failure_test.go](../../../internal/core/toolfailure/failure_test.go); [catalog_test.go](../../../internal/escalationpolicy/catalog_test.go) unknown tool |
| [AC-06.005](ep-acceptance-criteria.md#ac-06-005) | [REQ-06.006](ep-requirements.md#escalation-policy-and-chain) | ✓ | — | — | — | [run_test.go](../../../internal/core/run_test.go) chain/transport — Supporting; full escalation flow mock optional |
| [AC-06.006](ep-acceptance-criteria.md#ac-06-006) | [REQ-06.007](ep-requirements.md#escalation-policy-and-chain) | — | — | — | — | **Gap:** three-provider order test |
| [AC-06.007](ep-acceptance-criteria.md#ac-06-007) | [REQ-06.008](ep-requirements.md#exhaustion-and-stop) | — | — | — | — | Caps by code review; no dedicated test |
| [AC-06.008](ep-acceptance-criteria.md#ac-06-008) | [REQ-06.009](ep-requirements.md#rollback-at-end-of-turn) | — | — | — | — | **Gap:** two `HandleMessage` baseline assertion |
| [AC-06.009](ep-acceptance-criteria.md#ac-06-009) | [REQ-06.010](ep-requirements.md#observability)–[REQ-06.012](ep-requirements.md#nfr--security-testability-observability) | — | — | — | — | Fields implemented; no log-shape assertion |
| [AC-06.010](ep-acceptance-criteria.md#ac-06-010) | [REQ-06.013](ep-requirements.md#nfr--security-testability-observability) | ✓ | ✓ | — | — | Unit + integration suite; not every branch explicit |
| [AC-06.011](ep-acceptance-criteria.md#ac-06-011) | [REQ-06.014](ep-requirements.md#nfr--security-testability-observability) | — | — | — | — | Behaviour when chain nil; **Gap:** explicit `Covers AC-06.011` test |
| [AC-06.012](ep-acceptance-criteria.md#ac-06-012) | [REQ-06.015](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs) | ✓ | — | — | — | [failure_test.go](../../../internal/core/toolfailure/failure_test.go) |
| [AC-06.013](ep-acceptance-criteria.md#ac-06-013) | [REQ-06.016](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs) | — | — | — | — | **Gap:** mock Hermes + chain order |
| [AC-06.014](ep-acceptance-criteria.md#ac-06-014) | [REQ-06.017](ep-requirements.md#nfr--security-testability-observability) | ✓ | — | — | — | [catalog_test.go](../../../internal/escalationpolicy/catalog_test.go) — Covers AC-06.014 |

**Notes:** **Unit** = `go test ./internal/...`; **Integration** = `tests/integration` (build tag `integration`). Skill 08 requires every AC have automated or manual coverage—**gaps** above should be closed or deferred with user approval.

---

## Quality gate

- golangci-lint: **PASS**
- go vet: **PASS**
- Module boundaries: **PASS** (`escalationpolicy` → `core/toolfailure` only)

---

## Gaps, risks, recommendations

| Type | Item |
|------|------|
| **Gap** | AC-06.006 / AC-06.008 / AC-06.013: add focused tests per matrix. |
| **Gap** | AC-06.011: explicit test with escalation disabled and mock tool failure. |
| **Risk** | `catalogValidateErrorQualifies` uses substring markers; must stay in sync with `toolcatalog` messages or migrate to sentinels ([REQ-06.015](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs)). |
| **Recommendation** | Complete Task 7 (integration + README) when ready. |
