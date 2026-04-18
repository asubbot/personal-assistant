# Code review — EP-026 Core refactor: tier builders in conversation handler

**Reviewer:** Pipeline stage 10 (orchestrated review)

---

## Review iteration 1

**Review date:** 2026-04-17  
**Stage 10 iteration:** 1 of max 5  
**Scope:** `internal/core/handler.go` (`HandleMessage`), `internal/core/handler_tier_main_prompt.go`, `internal/core/handler_tier_main_prompt_test.go`

**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0 | Nit: 1 | Suggestion: 1  
**Gate:** Pass

### Summary

Tier-specific prompt assembly is extracted without changing the dynamic-cap predicate split between full and full_lite (`mergedAfterDynamicToolCap`). `HandleMessage` delegates pre-tail construction and tier assembly; `gocyclo` nolint is removed from `HandleMessage`. Tests cover tier dispatch, explicit builders, and absence of the old nolint marker. **Approve** for merge from a stage-10 perspective.

### Findings

| Severity | Location | Issue | Recommendation |
|----------|----------|-------|------------------|
| Nit | `buildMainTurnMessagesPreTail` | `sysHead` is re-read from `messages[0].Content` in `HandleMessage` after pre-tail; correct but slightly indirect. | Optional: return `sysHead` as a separate value from `buildMainTurnMessagesPreTail` to avoid coupling to slice layout. |
| Suggestion | `copyToolOriginMap` | Package-level helper; fine for complexity, but could be a tiny private method if you prefer all state on `conversationHandler`. | Keep as-is unless style guide prefers methods only. |

### Test / verification

`make check` and `./bin/validate EP-026` recorded in [ep-audit-report.md](ep-audit-report.md).

### Residual risks

Regression relies on existing `internal/core` tests plus new tier tests; no cross-package API change.
