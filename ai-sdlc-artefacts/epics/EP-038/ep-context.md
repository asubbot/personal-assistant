---
artefact: ep-context
epic_id: EP-038
status: draft
source_of_truth: false
updated_at: 2026-05-31
---

# Epic Context — EP-038 Refactor core conversation handler (god handler)

## Purpose

Split the monolithic `conversationHandler` in `internal/core/handler.go` into maintainable files without changing product behaviour, config schema, or contracts from EP-013/017/018/034/037.

## Current Scope

Decompose ~663 LOC `handler.go` into: slim `handler.go` (orchestration), new `handler_llm.go` (router complete + tool loop + LLM logs), new `handler_tools.go` (selection + execution), new `handler_memory.go` (RAG chunks + turn index). Keep `handler_tier_main_prompt.go`, `system_tail.go`, `dynamic_tool_selection.go`, `runtime_tools.go` as-is in responsibility. No config changes; prerequisites EP-035/036/037 merged first.

## Key Requirements

- **25 REQs** (19 FR, 6 NFR): file ownership map (`handler.go` ≤~200 LOC; new `handler_llm|tools|memory.go`), behaviour parity, frozen `config.json`, unchanged `MessageHandler` / `Run` / integration surfaces, EP-034/036/037 contracts preserved, `make check` + `validate ears EP-038`.
- Land only after **EP-035/036/037** merged; no new tiers, no `full_lite`, no tier-strategy framework.
- Out of scope: product behaviour, config schema DRY follow-ups, `conversationHandler` rename/export.

## Acceptance Signals

- **25 ACs** (`AC-38.001`–`AC-38.025`) trace 1:1 to **25 REQs**; structural gates (file split, grep, LOC) are **MANUAL ONLY**; behaviour parity leans on existing suites.
- **Automated:** `handler_ep017_test.go`, `handler_ep018_test.go`, `handler_ep018_coverage_test.go`, `handler_ep034_regression_test.go`, `handler_ep036_test.go`, `tools_selection_parity_test.go`, `handler_tier_main_prompt_test.go`, `run_test.go`, integration handler tests; **`make check`** (AC-38.021).
- **Contracts:** EP-034 transport-only routing, EP-036 two-tier, EP-037 `tools.selection` — no assertion changes except import fixes (AC-38.019–020).
- **Gates:** `./tools/validate/validate ears EP-038` (AC-38.022); prerequisites EP-035/036/037 merged before land (AC-38.001).

## Design Decisions

- **File split over frameworks:** Prefer moving method groups to `handler_{llm,tools,memory}.go` over introducing tier-strategy interfaces (only `simple` / `full` after EP-036).
- **Leave prior extractions:** `system_tail.go`, `dynamic_tool_selection.go`, `runtime_tools.go`, `vector_merge.go` stay put.
- **Config frozen:** EP-037 deferred `tool_output_artifacts` typing and `vector_search_tools` DRY—out of scope here too.

## Interfaces / Contracts

- **Public:** `MessageHandler.HandleMessage`, `BuildMessageHandler`, `NewIntegrationConversationHandler` unchanged.
- **Internal:** `conversationHandler` remains unexported; method receivers move files but signatures stay equivalent.

## Current Gate Summary

| Gate | Status |
|------|--------|
| Stage 3 ep-scope | draft |
| Stage 4 ep-requirements | draft |
| Stage 5 ep-acceptance-criteria | draft (25 ACs) |
| Stage 6 ep-system-design | — |

## Open Questions

None for stage 4 — HOTL: no config schema change; no tier-strategy framework unless stage 6 design proves a smaller alternative.

## Links

- [ep-scope.md](ep-scope.md)
- [ep-requirements.md](ep-requirements.md)
- [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- [strategy.md](../../strategy.md) — Refactoring 0.02, direction F
- [scope.md](../../scope.md)
- [EP-037 ep-scope](../EP-037/ep-scope.md) — deferred handler decomposition
- [EP-036 ep-scope](../EP-036/ep-scope.md) — two-tier intent prerequisite
- [EP-035 ep-scope](../EP-035/ep-scope.md) — package consolidation prerequisite
