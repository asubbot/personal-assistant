---
artefact: ep-code-review
epic_id: EP-034
status: draft
source_of_truth: true
gate: pass
latest_iteration: 2
open_counts:
  blocker: 0
  major: 0
  medium: 0
  minor: 0
non_blocking_counts:
  nit: 1
  suggestion: 1
next_action: proceed_to_stage_11
updated_at: 2026-05-29
---

# Code review — EP-034 Remove tool-path LLM escalation

---

## Current Gate Summary

Gate: Pass
Latest iteration: 2
Last updated: 2026-05-29
Open counts: Blocker 0 | Major 0 | Medium 0 | Minor 0
Non-blocking counts: Nit 1 | Suggestion 1
Open findings: none
Next action: Proceed to stage 11

---

## Review iteration 1

**Review date:** 2026-05-29
**Stage 10 iteration:** 1 of max 5
**Scope:** All changes on branch `epic/EP-034-remove-tool-path-escalation` vs `main` (40 files, +148/−2121 lines). Includes uncommitted untracked files under `ai-sdlc-artefacts/epics/EP-034/`.
**Iteration summary — open counts:** Blocker: 0 | Major: 1 | Medium: 1 | Minor: 1 | Nit: 1 | Suggestion: 1
**Gate:** Fail

### Summary

The implementation is thorough and well-executed. All six tasks in the implementation plan are complete: escalation packages (`escalationpolicy`, `toolfailure`) deleted, config rejects `tools.llm_escalation`, `llmrouter` simplified to transport-only, EP-006 tests removed and replaced with EP-034 regression tests, and operator docs updated. `make check` passes cleanly (all tests, vet, lint, vuln, module-boundary checks). The core invariant — tool failures never advance provider index — is correctly enforced and tested.

One major finding blocks the gate: `docs/architecture-ru.md`, a new file added in this branch, still describes tool-path escalation and `baseline_index` as active product behaviour in approximately 10 locations, contradicting the removal.

### What was done well

- Clean deletion of `escalationpolicy` and `toolfailure` packages with no stale imports remaining.
- Config rejection via `rejectRemovedToolsConfigKeys` is a good refactoring that aligns with the existing `text_based_enabled` pattern.
- `completeViaRouter`, `runToolResultLoop`, and `appendToolRound` simplified correctly — `llmTurnState`, `maybeEscalate`, and qualifying-failure tracking all removed.
- Handler no longer carries `escalation *config.LLMEscalationConfig` field — correct per system design.
- Regression test `TestHandleMessage_toolFailure_doesNotAdvanceProvider` (AC-34.001) is well-structured: mocks two providers, triggers tool error, asserts p1 never called.
- Transport fallback test coverage (AC-34.004) is comprehensive across `llmrouter` and `provider_adapter` tests.
- `NewState` always returns `ActiveIndex: 0` — verified in `TestNewState_alwaysStartsAtZero` (AC-34.006).
- Traceability tests (AC-34.002, AC-34.003, AC-34.008–AC-34.012) scan the codebase for forbidden patterns — good regression safety net.
- Threat model and strategy updates are correct and proportionate.

### Findings

| ID | Severity | Location | Issue | Recommendation |
|----|----------|----------|-------|----------------|
| F-001 | **Major** | `docs/architecture-ru.md` lines 77, 264, 282, 288, 295, 297, 399, 478, 571, 573, 612, 627 | New file in this branch still describes tool-path escalation as active: "tool escalation" in router diagram, `baseline_index или 0`, `tools.llm_escalation.baseline_index` in routing table, "escalation на сильную модель" in summary. Contradicts AC-34.011 (docs must not present tool-path escalation as active). This is a new doc added in the same changeset, so it should reflect EP-034 state. | Update all ~10 references: remove escalation from mermaid diagrams, replace routing table rows with transport-fallback-only descriptions, update summary to reflect EP-034 removal. The file is in Russian but the technical terms and config paths must match the post-EP-034 reality. |
| F-002 | **Medium** | `internal/llmrouter/provider_adapter.go:25-27` | `SummarizeRouterConfig(cfg *config.Config)` accepts a `cfg` parameter that is explicitly discarded (`_ = cfg`). This is dead code that misleads callers into thinking config is used. | Either (a) remove the parameter and update the 2 call sites, or (b) document why the signature is kept (e.g. future use). Option (a) is KISS. |
| F-003 | **Minor** | `internal/llmrouter/router_test.go:47-48` | Duplicate line `// Covers AC-34.004` on `TestComplete_nonRetryable_stopsImmediately`. | Remove the duplicate comment line. |
| F-004 | **Nit** | `internal/llmrouter/router_test.go` | Several test functions have `// Covers AC-34.004` but test different aspects (non-retryable stop, nil state, out-of-range, max attempts, event payload, error text). The AC-34.004 traceability is technically correct since they all exercise `Complete` transport behaviour, but using the same AC for every test reduces signal. | Consider using more specific ACs where applicable (e.g. AC-34.006 for `TestNewState_alwaysStartsAtZero` — already done correctly there). Not blocking. |
| F-005 | **Suggestion** | `internal/core/ep034_traceability_test.go:193` | `TestEP034_makeCheckQualityGate` is an empty test body (no assertions). It serves as a traceability marker but runs no verification. | Either add a brief comment explaining it is intentionally a marker (AC-34.015 is verified externally by `make check`), or remove it if `TestEP034_validateCommandExitZero` already covers the quality gate chain. Not blocking. |

### Test / verification

- `make check` — **PASS** (exit 0). All packages pass including `pa/cmd/pa`, `pa/internal/core`, `pa/internal/llmrouter`, `pa/internal/config`, `pa/internal/noderunner`. Race detector enabled. golangci-lint 0 issues. govulncheck clean. Module boundaries OK.
- AC-34.001 coverage: `TestHandleMessage_toolFailure_doesNotAdvanceProvider` in `handler_ep034_regression_test.go`.
- AC-34.004 coverage: `TestComplete_retryableFirst_switchesToNext`, `TestProviderAdapter_retryableFallbackAndModelLabel`, and several related tests in `llmrouter`.
- AC-34.007 coverage: `TestLoad_ToolsLLMEscalation_rejected` with fixture `tools_llm_escalation_rejected.json`.
- No stale imports of `escalationpolicy` or `toolfailure` in product code (verified by traceability tests and grep).

### Residual risks / follow-ups

- `docs/architecture-ru.md` is new and untracked; the traceability test `TestEP034_operatorDocsNoActiveToolEscalation` only checks `configuration.md` and `llm-provider-roles-and-logging.md`, so it does not catch stale escalation references in additional doc files.
- If `SummarizeRouterConfig` signature is kept for API stability, consider adding a brief comment explaining why the parameter is retained.

---

## Review iteration 2

**Review date:** 2026-05-29
**Stage 10 iteration:** 2 of max 5
**Scope:** All changes on branch `epic/EP-034-remove-tool-path-escalation` vs `main` including uncommitted changes. Focused verification of F-001, F-002, F-003 fixes from iteration 1, plus full re-scan.
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0 | Nit: 1 | Suggestion: 1
**Gate:** Pass

### Summary

All three blocking findings from iteration 1 are resolved. `make check` passes (exit 0, all packages OK, golangci-lint 0 issues, govulncheck clean, module boundaries OK, race detector enabled). No new blocking findings identified. The implementation is approved for merge.

### Fix verification

| ID | Status | Verification |
|----|--------|--------------|
| F-001 | **Resolved** | `docs/architecture-ru.md` no longer contains `tool.?escalat`, `baseline_index`, `escalation на`, or `llm_escalation`. Grep returns zero matches. The file now reflects post-EP-034 transport-fallback-only reality. |
| F-002 | **Resolved** | `SummarizeRouterConfig()` now takes zero parameters (line 24 of `provider_adapter.go`). The dead `cfg` parameter and `_ = cfg` assignment are removed. Call site in `cmd/pa/main.go:66` updated accordingly. |
| F-003 | **Resolved** | Line 47 of `router_test.go` has a single `// Covers AC-34.004` comment before `TestComplete_nonRetryable_stopsImmediately`. No duplicate. |

### Full re-scan findings

| ID | Severity | Location | Issue | Recommendation |
|----|----------|----------|-------|----------------|
| F-004 | **Nit** | `internal/llmrouter/router_test.go` | Several test functions share `// Covers AC-34.004` but test distinct aspects (non-retryable stop, nil state, out-of-range, max attempts, event payload, error text). Correct but low signal. | Consider more specific AC references where applicable. Not blocking. |
| F-005 | **Suggestion** | `internal/core/ep034_traceability_test.go:193` | `TestEP034_makeCheckQualityGate` is an empty test body (marker only). | Add a brief comment explaining it is intentionally a marker for AC-34.015, or remove if `TestEP034_validateCommandExitZero` already covers the quality gate chain. Not blocking. |

### Test / verification

- `make check` — **PASS** (exit 0). All packages pass. Race detector enabled. golangci-lint 0 issues. govulncheck clean. Module boundaries OK. Coverage 75.8%.
- No stale `escalationpolicy` or `toolfailure` imports in product code.
- No active escalation references in `docs/` (remaining mentions correctly describe the feature as removed/rejected).
- `internal/cmdsafe/remote.go:17` mentions "escalation" in a comment about command validation (local→remote), unrelated to tool-path LLM escalation — no action needed.

### Residual risks / follow-ups

- F-004 (Nit) and F-005 (Suggestion) are non-blocking and may be addressed at the team's discretion in a future cleanup pass.
