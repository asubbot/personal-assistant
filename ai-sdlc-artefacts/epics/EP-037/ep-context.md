---
artefact: ep-context
epic_id: EP-037
status: draft
source_of_truth: false
updated_at: 2026-05-30
---

# Epic Context — EP-037 Consolidate tool pre-selection configuration

## Purpose

Reduce overlapping config for catalog tool vector pre-selection and EP-018 dynamic tool cap by merging them into `tools.selection`, without changing selection results for equivalent settings.

## Current Scope

Add required `tools.selection` (`tool_search_top_k`, `tool_min_count`, `tool_fallback_cap`, `enabled`, `max_tools_for_llm_request`). Remove `tool_pre_selection` and `tools.dynamic_selection`. Update config loader, minimal core wiring, examples, testdata, docs. Defer `vector_search_tools` JSON DRY.

## Key Requirements

Not yet written (stage 4).

## Acceptance Signals

Not yet written (stage 5).

## Design Decisions

- Target shape: single `tools.selection` under required `tools`.
- `runtime_skills.tool_vector_top_k_cap` unchanged; still caps effective top-K at runtime.
- `vector_search_tools` triple-schema repetition: out of scope (deferred).

## Interfaces / Contracts

- Config: `tools.selection` replaces `tool_pre_selection` + `tools.dynamic_selection`.
- Runtime: `mergeSelectedToolIDs` + `mergedAfterDynamicToolCap` behaviour unchanged.

## Current Gate Summary

| Gate | Status |
|------|--------|
| Stage 3 ep-scope | draft |

## Open Questions

None for stage 3 — HOTL default: defer `vector_search_tools` DRY; keep EP-038 for handler refactor.

## Links

- [ep-scope.md](ep-scope.md)
- [scope.md](../../scope.md)
- [strategy.md](../../strategy.md) — Refactoring 0.02
