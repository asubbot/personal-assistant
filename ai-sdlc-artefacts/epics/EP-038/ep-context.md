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

(To be derived in stage 4 from [ep-scope.md](ep-scope.md): file ownership map, parity tests, LOC/orchestration target, zero schema change, preserved `MessageHandler` API.)

## Acceptance Signals

(To be defined in stage 5: `make check`, existing handler_ep0xx regression suites, no prompt/tool-list diffs on golden paths.)

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
| Stage 4 ep-requirements | — |
| Stage 5 ep-acceptance-criteria | — |
| Stage 6 ep-system-design | — |

## Open Questions

None for stage 3 — HOTL: no config schema change; no tier-strategy framework unless stage 6 design proves a smaller alternative.

## Links

- [ep-scope.md](ep-scope.md)
- [strategy.md](../../strategy.md) — Refactoring 0.02, direction F
- [scope.md](../../scope.md)
- [EP-037 ep-scope](../EP-037/ep-scope.md) — deferred handler decomposition
- [EP-036 ep-scope](../EP-036/ep-scope.md) — two-tier intent prerequisite
- [EP-035 ep-scope](../EP-035/ep-scope.md) — package consolidation prerequisite
