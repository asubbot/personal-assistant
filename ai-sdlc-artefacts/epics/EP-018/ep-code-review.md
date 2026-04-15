# Code review — EP-018 Tiered Prompt Cost Reduction

---

## Review iteration 1

**Date:** 2026-04-15 (UTC)  
**Scope:** Branch `epic/EP-018-tiered-prompt-costs` — `cmd/pa/main.go`, `internal/config/*`, `internal/core/*` (handler, run, dynamic tool selection, EP-018 tests), `internal/intent/*`, `docs/configuration.md`.  
**Inputs:** [ep-requirements.md](ep-requirements.md), [ep-acceptance-criteria.md](ep-acceptance-criteria.md), [ep-system-design.md](ep-system-design.md).

### Iteration summary — open counts

**Blocker:** 0 | **Major:** 0 | **Medium:** 0 | **Minor:** 0 | **Nit:** 1 | **Suggestion:** 2

### Gate (§2.2)

**Pass** — zero open Blocker / Major / Medium / Minor.

### Summary

Implementation aligns with EP-018: `TierFullLite`, heuristic `full_lite_patterns` and model three-way classification, `tools.dynamic_selection` with load-time validation, `HandleMessage` branches (no RAG and no runtime skill tail for `full_lite`, session preserved), optional dynamic cap for `full` and `full_lite`, and INFO log `main llm prompt assembled`. Tests and `./bin/validate EP-018` provide traceability; AC-18.016 is explicitly manual. **Recommendation: approve** for merge from a stage-10 perspective.

### Blockers

*(none)*

### Findings

| Severity | Location | Issue | Recommendation |
|----------|----------|--------|------------------|
| Nit | `internal/intent/model_test.go` | Prior test name implied stricter “only” wording than assertions enforce. | Renamed to `TestModel_PromptContainsTierLabelsAndDelimitedUserMessage` in the same change set. |
| Suggestion | `internal/intent/cascade.go` | Model failure uses `Warn` without `context.Context`. | Consider `WarnContext` if trace correlation is needed. |
| Suggestion | `internal/core/dynamic_tool_selection.go` | `ApplyDynamicToolCap` with `max < 1` returns uncapped copy. | Document that EP-018 callers rely on config enforcing `max >= 1` when dynamic is enabled. |

### Test / verification

- `make check` — **pass** (exit 0).
- `./bin/validate EP-018` — **pass** (exit 0); 21/21 AC traced (20 automated, 1 manual for AC-18.016).

### Residual risks

- AC-18.016 fallback when vector pre-selection is disabled is manual-only; consider an integration test later.
- REQ-18.010 literal “only” vs small instructional template: acceptable per tests; strict doc alignment optional follow-up.
