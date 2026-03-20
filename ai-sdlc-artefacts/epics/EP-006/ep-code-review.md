# Code review: EP-006 tool-call reliability branch

**Skill:** [code-review.skill.md](../../../ai-sdlc/specification/skills/code-review.skill.md)  
**Date:** 2026-03-20  
**Reviewer:** AI-assisted (Cursor), aligned with [AGENTS.md](../../../AGENTS.md) (KISS, fail fast).

**Follow-up update:** Subsequent changes after the initial pass are reflected in §1, §2, §4, §5, and §7 (remote exec diagnostics in `noderunner`, config validation for escalation cap).

---

## 1. Scope

- **Branch:** `EP-006-tool-call-reliability` vs **`main`** (merge-base `bdc0cff9cea6a06089a4a05e43370bca72249e71`).
- **Approximate delta:** ~70 files, +4529 / −544 lines (product code, tests, config example, SDLC artefacts, diagrams, pipeline/docs), **plus** follow-up commits: `tools.llm_escalation.max_per_user_message >= 1` when `enabled`, and **`internal/noderunner`** remote `stdout`/`stderr` on SSH exec failure (truncated, UTF-8-safe; unified `remote_stdout` / `remote_stderr` log attrs; DEBUG `ssh exec output` on success).
- **Primary themes:** EP-006 (typed tool failures, escalation policy, handler/router integration, Hermes path), EP-007 (unified `llmrouter` for transport fallback + labels), removal of legacy `internal/llm/fallback` chain for conversation path, integration tests, epic documentation in this folder and related EP-001 updates.
- **Config / docs follow-up (landed):** `validateLLMEscalation` requires `max_per_user_message >= 1` when escalation is enabled; testdata + tests; README and epic manuals/plans updated (`internal/config/load.go`, `internal/config/config.go`, `internal/config/config_test.go`, `internal/config/testdata/tools_llm_escalation_max_zero.json`, `internal/core/handler_ep006_audit_test.go`, [ep-implementation-plan.md](ep-implementation-plan.md), [ep-manual-tests.md](ep-manual-tests.md), [EP-001 ep-implementation-plan.md](../EP-001/ep-implementation-plan.md)).

**Not in scope:** Line-by-line review of every artefact/diagram; full re-audit against every REQ line in EP-006 (spot-check only).

---

## 2. Summary

The branch delivers a **coherent** implementation of tool-path reliability and LLM escalation: explicit failure typing (`internal/core/toolfailure`, `internal/toolcatalog/ValidateError`, `internal/escalationpolicy`), a single **`llmrouter`** for completion routing and events, and **handler** integration covering native tools and Hermes/text-tool follow-ups. Tests (unit + integration) are substantial.

**Follow-up:** Remote commands that exit non-zero previously dropped node **stdout/stderr** from the returned error and from operator logs on the failure path; [`internal/noderunner/runner.go`](../../../internal/noderunner/runner.go) now appends labeled, truncated fragments to the exec error (so [`handler.appendToolRound`](../../../internal/core/handler.go) surfaces them to the model and JSONL), and logs `remote_stdout` / `remote_stderr` on `ssh exec` errors, with DEBUG `ssh exec output` using the same attribute names on success.

**Merge recommendation: approve with minor nits** — no blockers identified. Config validation removes the `max_per_user_message: 0` + `enabled: true` footgun (use `enabled: false` for “no policy escalation”).

---

## 3. Blockers

*None.*

---

## 4. Findings

| Severity | Location | Issue | Recommendation |
|----------|----------|--------|----------------|
| **Major** | — | — | — |
| **Minor** | `internal/core/handler.go` (`appendToolRound`, tool invocation logging) | At **INFO**, logs include **full tool arguments** and **stdout** on success (and **full `execErr.Error()`** on failure, which can now include truncated remote `stdout`/`stderr` from `noderunner`). May leak secrets or sensitive data. | Prefer truncation, DEBUG for full payload, or reuse redaction patterns where appropriate; align with logging/redaction requirements. |
| **Minor** | — | `tools.llm_escalation` with `enabled: true` and `max_per_user_message: 0` | **Addressed:** `validateLLMEscalation` requires `max_per_user_message >= 1` when enabled. |
| **Minor** | — | On SSH exec failure, node **stderr** (e.g. CLI error lines) was omitted from errors and from non-DEBUG logs, so LLM tool results showed only `Process exited with status 1`. | **Addressed:** [`finishRemoteExec`](../../../internal/noderunner/runner.go) in `noderunner`; tests in [`runner_test.go`](../../../internal/noderunner/runner_test.go). |
| **Nit** | `internal/llmrouter/provider_adapter.go` | `NewProviderAdapter` uses `Config{}` (no escalation config). Correct for summarize/auxiliary path but easy to misread. | Short comment that escalation policy is intentionally omitted for this adapter. |
| **Suggestion** | `internal/core/handler.go` (`runToolResultLoop`) | When exiting due to round cap with **pending** `tool_calls`, the user may see an odd final assistant message. | Optional: detect and return a deterministic “tool loop limit” style message if product wants clearer UX. |
| **Suggestion** | Multi-provider tool loops | After policy escalation, the **same message history** is continued on the next provider; usually OK but not guaranteed for all backends. | Document as known limitation or extend integration tests if multi-vendor tool calling is a hard requirement. |

---

## 5. Test / verification

- **`make check`** (fmt, vet, golangci-lint, `go test -tags=integration ./...`, module boundaries) — **PASS** on initial review date and after follow-up `noderunner` / config changes (re-run before release as usual).
- Integration coverage includes [ep006_escalation_run_test.go](../../../tests/integration/ep006_escalation_run_test.go) (escalation chain, baseline reset across messages).
- Unit coverage for remote streams: [`internal/noderunner/runner_test.go`](../../../internal/noderunner/runner_test.go) (`TestRunOnNode_execError_includesRemoteStderr`, stdout+stderr, truncation).

---

## 6. Residual risks / follow-ups

- **Configuration migration:** Removing committed `config/config.json` from the repo (if present on the branch) requires operators to use `config/config.example.json` (or equivalent) and local `config.json`; ensure onboarding docs stay explicit ([README.md](../../../README.md)).
- **Artefact drift:** Large epic/doc updates; periodic alignment of [ep-audit-report.md](ep-audit-report.md) with post-merge behaviour is advisable after any further handler/router/noderunner edits.
- **Remote output in errors/logs:** Truncation limits blast radius; optional future redaction for remote stderr in app logs if scripts echo secrets.

---

## 7. Positive notes (non-exhaustive)

- **Fail fast:** Typed errors for escalation qualification (`errors.As`), config validation for escalation (`baseline_index`, provider count, `max_per_user_message` when enabled).
- **Observability:** Structured routing events (`llm tool escalation`, transport switch) with indices/labels; **remote exec** failures include truncated **`remote_stdout` / `remote_stderr`** in structured logs and in the error string consumed by the handler / LLM tool messages.
- **Test depth:** `handler_ep006_audit_test.go`, `llmrouter` tests, EP-006 integration tests, `noderunner` exec-error stream tests.

---

## Reference

- Epic: [scope](ep-scope.md), [requirements](ep-requirements.md), [acceptance criteria](ep-acceptance-criteria.md), [manual tests](ep-manual-tests.md).
- Project rules: [AGENTS.md](../../../AGENTS.md).
