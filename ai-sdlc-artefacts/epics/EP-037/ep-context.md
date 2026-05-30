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

- **REQ-37.001–004:** Required `tools.selection` (`tool_search_top_k`, `tool_min_count`, `tool_fallback_cap`, `enabled`, `max_tools_for_llm_request`) with same validation as legacy blocks.
- **REQ-37.005–008, 020:** Reject `tool_pre_selection` and `tools.dynamic_selection`; drop former from root keys; preserve explicit-JSON rules.
- **REQ-37.009–012:** Unchanged merge/cap behaviour and `min(top_k, tool_vector_top_k_cap)` for equivalent settings.
- **REQ-37.013–017:** Update all configs/docs; regression tests for rejection and parity.
- **REQ-37.022–024:** No `vector_search_tools` DRY, no EP-038 handler refactor, no new selection features.

## Acceptance Signals

- **Unit:** `tools.selection` five-field load; bounds; legacy-key rejection (`tool_pre_selection`, `tools.dynamic_selection`); unknown `tools` keys; merge/cap parity + `min(top_k, tool_vector_top_k_cap)`; repo example/testdata/integration configs load.
- **Manual:** operator docs + live `.config/config.json`; `make check`; `./bin/validate ears EP-037`; scope guards (no `vector_search_tools` DRY, no EP-038 refactor, no new selection features).
- **23 ACs** in [ep-acceptance-criteria.md](ep-acceptance-criteria.md) — all REQ-37.001–024 traced.

## Design Decisions

- `ToolsSelection` struct in `internal/config/config.go` on required `ToolsConfig.Selection` (five explicit JSON fields).
- Legacy keys rejected via `rejectRemovedUnsupportedConfigKeys` / `rejectRemovedToolsConfigKeys` (EP-034/036 raw-JSON pattern); `tool_pre_selection` dropped from `configRootJSONKeys`.
- `enabled` gates runtime cap only; when false, `max_tools_for_llm_request` may be 0 and is ignored at runtime.
- `runtime_skills.tool_vector_top_k_cap` unchanged; `vector_search_tools` DRY deferred.

## Interfaces / Contracts

- Config: `tools.selection` replaces `tool_pre_selection` + `tools.dynamic_selection`; `validateToolsSelection` + `validateToolsObjectKeys`.
- Runtime: handler reads `cfg.Tools.Selection`; `mergeSelectedToolIDs` / `mergedAfterDynamicToolCap` algorithms unchanged.

## Current Gate Summary

| Gate | Status |
|------|--------|
| Stage 3 ep-scope | draft |
| Stage 4 ep-requirements | draft |
| Stage 5 ep-acceptance-criteria | draft |
| Stage 6 ep-system-design | draft |

## Open Questions

None for stage 3 — HOTL default: defer `vector_search_tools` DRY; keep EP-038 for handler refactor.

## Links

- [ep-system-design.md](ep-system-design.md)
- [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- [ep-requirements.md](ep-requirements.md)
- [ep-scope.md](ep-scope.md)
- [scope.md](../../scope.md)
- [strategy.md](../../strategy.md) — Refactoring 0.02
