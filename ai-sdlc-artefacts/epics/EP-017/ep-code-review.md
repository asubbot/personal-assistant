# Code review — EP-017 Intent Classifier for Prompt Optimization

---

## Review iteration 1

**Review date:** 2026-04-15
**Stage 10 iteration:** 1 of max 5
**Scope:** Branch `epic/EP-017-intent-classifier`, all new files under `internal/intent/`, modified files in `internal/config/`, `internal/core/`, `cmd/pa/main.go`, `docs/configuration.md`, plus all new test files. Cross-checked against ep-requirements.md, ep-system-design.md, ep-acceptance-criteria.md, ep-implementation-plan.md, ep-scope.md.
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 2 | Nit: 1 | Suggestion: 2
**Gate:** Fail (Minor > 0)

### Summary

The EP-017 implementation is clean, well-structured, and closely follows the system design. The `internal/intent` package provides a well-separated `Classifier` interface with heuristic, model, and cascade implementations. Config validation is fail-fast, the `HandleMessage` gating on tier is surgical, and the test suite covers all 18 ACs. The two Minor findings relate to missing observability assertions in tests — the WARN log on model-stage failure (AC-17.011) and the handler-level INFO log fields (AC-17.016) are structurally present in code but never verified against captured logger output. Recommend approve with minor fixes.

### Findings

| # | Severity | Location | Issue | Recommendation |
|---|----------|----------|-------|----------------|
| 1 | **Minor** | `internal/intent/cascade_test.go` | AC-17.011 states "a WARN-level log entry SHALL be recorded with the error details". `TestCascade_ModelError_DefaultFull` creates the cascade with `logger: nil`, so the WARN path is never exercised. | Add a test with a captured logger buffer to verify the WARN is emitted with the error message. |
| 2 | **Minor** | `internal/core/handler_ep017_test.go:178` | `TestHandleMessage_ClassificationLogsInfo` (AC-17.016) uses `slog.Default()` and does not capture logger output to verify the INFO entry contains `tier`, `stage`, `message_len`. | Use a `bytes.Buffer`-backed logger and assert the captured output contains the required keys. |
| 3 | **Nit** | `cmd/pa/main.go:785` | `timeout, _ = time.ParseDuration(...)` silently discards the parse error. Config validation ensures this never fails, but inconsistent with fail-fast principle. | Add explicit error handling or a comment. |
| 4 | **Suggestion** | `internal/intent/model.go:44` | Classification prompt embeds user message via Sprintf — crafted input could influence cheap model response. Impact very low (worst case: wrong tier → defaults to full). | Future hardening: consider quoting/escaping the message. |
| 5 | **Suggestion** | `internal/core/run.go` | `core.Run()` now has 14 positional parameters. | Future cleanup: introduce an Options struct for optional dependencies. |

---

## Review iteration 2

**Review date:** 2026-04-15
**Stage 10 iteration:** 2 of max 5
**Scope:** Three files fixed after iteration 1 feedback — `internal/intent/cascade_test.go`, `internal/core/handler_ep017_test.go`, `cmd/pa/main.go`
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0 | Nit: 0 | Suggestion: 2
**Gate:** Pass

### Summary

All three iteration-1 findings (Minor #1, Minor #2, Nit #3) have been resolved correctly and introduce no new issues. The change set is approved. The two iteration-1 Suggestions (acknowledged as out-of-scope for EP-017) are carried forward as residual risks.

### Verification of fixes

- **Minor #1 resolved:** Added `TestCascade_ModelError_LogsWarn` with `bytes.Buffer`-backed logger; asserts WARN level and error string "connection refused".
- **Minor #2 resolved:** Updated `TestHandleMessage_ClassificationLogsInfo` with captured logger; asserts `tier=`, `stage=`, `message_len=` keys in output.
- **Nit #3 resolved:** Changed to explicit `parseErr` handling with `return nil, fmt.Errorf(...)` in `buildIntentClassifier`.

No new findings introduced by the fixes. `make check` passes.

### Residual risks / follow-ups

| # | Description |
|---|-------------|
| 1 | Classification prompt injection (Suggestion #4) — track for future security hardening. |
| 2 | `core.Run()` parameter sprawl (Suggestion #5) — consider Options struct in a future refactor. |
