# Code review: EP-006 tool-call reliability branch

**Skill:** [10-code-review.skill.md](../../../ai-sdlc/specification/skills/10-code-review.skill.md)  
**Dates:** 2026-03-20 (initial + follow-ups); **repeat pass** same day (handler/router/noderunner, `.golangci.yml`, `logredact`, `cmd/pa` wiring).  
**Reviewer:** AI-assisted (Cursor), aligned with [AGENTS.md](../../../AGENTS.md) (KISS, fail fast).

---

## 1. Scope

- **Branch:** `EP-006-tool-call-reliability` vs **`main`** (~75 files in latest diff stat).
- **Primary themes:** EP-006 (typed tool failures, escalation policy, handler + `llmrouter`), removal of legacy `internal/llm/fallback` for the conversation path, config `tools.llm_escalation`, Hermes/text-tool path, integration tests; EP-007-style unified router for transport fallback + labels.
- **Follow-ups landed before repeat pass:** `validateLLMEscalation` requires `max_per_user_message >= 1` when enabled; `internal/noderunner` remote `stdout`/`stderr` on SSH exec failure (truncated, UTF-8-safe); DEBUG `ssh exec output` on success; **`SetLogRedactor` / `BuildLogRedactor`** from `cmd/pa` for **log** fragments only (errors returned to tools stay unredacted for diagnostics — see §4).
- **Tooling:** [`.golangci.yml`](../../../.golangci.yml) — added `revive`, `gocritic`, `unparam`, `forbidigo` with tuned `disabled-checks` / revive rule subset; small fixes in `logredact`, `allowlist` tests, `cmd/pa` (`stdout != ""`).

**Not in scope:** Line-by-line review of every artefact/diagram; full REQ-by-REQ audit of EP-006 (spot-check only).

---

## 2. Summary

The branch delivers a **coherent** implementation: explicit failure typing (`internal/core/toolfailure`, `toolcatalog.ValidateError`, `internal/escalationpolicy`), a single **`llmrouter`** for completion routing and events, and handler integration for native tools and Hermes follow-ups. Tests (unit + integration) are substantial.

**Repeat pass:** Unreachable `else if h.escalationEnabled()` after `h.router == nil` in `HandleMessage` was **removed**. Tool invocation **INFO** logs use **`redactLogString`** for arguments, results, and error text ([`appendToolRound`](../../../internal/core/handler.go)). Noderunner applies the same redactor to **log** attributes for remote stream fragments when configured.

**Merge recommendation: approve with nits** — no blockers. Remaining items are observability consistency (Hermes phase in routing events), optional transport policy for 429, and product stance on raw remote output inside **returned** tool errors (vs logs).

---

## 3. Blockers

*None.*

---

## 4. Findings

| Severity | Location | Issue | Recommendation |
|----------|----------|--------|----------------|
| **Minor** (addressed) | `internal/core/handler.go` — tool invocation INFO | Previously: full args/stdout/errors in logs. | **Addressed:** `redactLogString` on `arguments`, `result`, and `error` in [`appendToolRound`](../../../internal/core/handler.go). |
| **Minor** (addressed) | Config `max_per_user_message: 0` + `enabled: true` | Misconfiguration footgun. | **Addressed:** `validateLLMEscalation` requires `>= 1` when enabled. |
| **Minor** (addressed) | `internal/noderunner` — SSH exec failure | Missing stderr/context in errors/logs. | **Addressed:** [`finishRemoteExec`](../../../internal/noderunner/runner.go); tests in [`runner_test.go`](../../../internal/noderunner/runner_test.go). |
| **Minor** (addressed) | Dead branch in `HandleMessage` | `else if h.escalationEnabled()` unreachable when `router == nil`. | **Addressed:** branch removed; state only from `h.router.NewState()` when router non-nil. |
| **Nit** | `internal/core/handler.go` — `maybeEscalate` | `OnQualifyingFailure` always uses `llmrouter.PhaseToolFailure` even for `failureClass == "hermes_parse"` (`PhaseHermesParse` exists in [`types.go`](../../../internal/llmrouter/types.go)). | Pass correct `Phase` from `failureClass` (or separate param) for clearer routing telemetry. |
| **Nit** | `internal/llmrouter/provider_adapter.go` | `NewProviderAdapter` uses `Config{}` (no escalation). | Short comment that escalation is intentionally omitted for non-conversation callers (e.g. summarize). |
| **Nit** | [`.golangci.yml`](../../../.golangci.yml) | Several `gocritic` checks disabled (`builtinShadow`, `paramTypeCombine`, …) to limit noise. | Re-enable selectively over time or accept as review-dependent. |
| **Suggestion** | `internal/llmrouter/classifier.go` | Only 5xx + network/timeout → transport switch; 429 → `FailureClassOther`. | Extend if rate-limit failover is desired; else document as intentional. |
| **Suggestion** | `internal/noderunner/runner.go` — returned errors | `remoteStreamsSuffix` embeds raw truncated stdout/stderr in **returned** errors (by design for LLM/tool diagnostics); logs can be redacted via `SetLogRedactor`. | If policy forbids surfacing remote output to the model/user, redact or strip in the tool-result path separately from logs. |
| **Suggestion** | `internal/core/handler.go` — `runToolResultLoop` | On hitting round cap with pending `tool_calls`, UX may be unclear. | Optional deterministic “tool loop limit” user message. |
| **Suggestion** | Multi-provider tool loops | Same history continued after policy escalation. | Document limitation or extend tests if multi-vendor tool calling is a hard requirement. |
| **Nit** | Public API | `core.Run(..., []llm.Provider, []string, ...)` breaks external importers. | Note in release notes if the module is published. |

---

## 5. Test / verification

- **`make check`** (fmt, vet, golangci-lint including new linters, `go test -tags=integration ./...`, module boundaries) — **PASS** on repeat pass (0 lint issues).
- Integration: [ep006_escalation_run_test.go](../../../tests/integration/ep006_escalation_run_test.go).
- Unit: `handler_ep006_audit_test.go`, `llmrouter` tests, `noderunner` exec-error/stream/redaction tests.

---

## 6. Residual risks / follow-ups

- **Returned tool errors** may still carry sensitive remote fragments (truncated but not redacted); logs mitigated via redactor when wired from `main`.
- **Configuration migration:** operators copy `config.examples/*` into **`.config/`**; keep [README.md](../../../README.md) explicit.
- **Artefact drift:** Re-align [ep-audit-report.md](ep-audit-report.md) after further handler/router/noderunner edits.
- **Transport storms:** watch for `llmrouter: exceeded max attempts` in noisy provider environments.

---

## 7. Positive notes (non-exhaustive)

- **Fail fast:** Typed escalation qualification (`errors.As`), strict escalation config validation.
- **Observability:** Routing events with indices/labels; remote exec failures include structured `remote_stdout` / `remote_stderr`; optional log redaction for those fragments.
- **Test depth:** Large handler audit tests, router tests, EP-006 integration tests, noderunner stream/redaction tests.
- **Linting:** `revive` (incl. `unreachable-code`), `gocritic`, `unparam`, `forbidigo` integrated with pragmatic exclusions.

---

## Reference

- Epic: [scope](ep-scope.md), [requirements](ep-requirements.md), [acceptance criteria](ep-acceptance-criteria.md), [manual tests](ep-manual-tests.md).
- Project rules: [AGENTS.md](../../../AGENTS.md).
