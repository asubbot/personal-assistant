---
artefact: ep-scope
epic_id: EP-037
status: draft
source_of_truth: true
updated_at: 2026-05-30
---

# Epic scope — EP-037 Consolidate tool pre-selection configuration

| Field | Content |
|-------|---------|
| **ID** | EP-037 |
| **Status** | NEW |
| **Title** | Consolidate tool pre-selection configuration |
| **Description** | Collapse overlapping configuration for catalog tool vector pre-selection and EP-018 per-request tool capping into one `tools.selection` block, remove the legacy top-level `tool_pre_selection` and `tools.dynamic_selection` keys, and keep runtime tool-selection outcomes unchanged for equivalent settings. Part of Refactoring increment 0.02 (architecture-simplification direction B). |
| **First version date** | 2026-05-30 |

## Glossary

- **Tool vector pre-selection:** Ranking catalog tools by semantic similarity to the user message (`toolindex.SelectToolIDs`), then merging with `always_include` and skill-linked tool ids (`mergeSelectedToolIDs` in core).
- **Dynamic tool cap:** After merge, optionally narrowing the tool id list sent to the main LLM (`pickToolsForMainRequest`, formerly `tools.dynamic_selection`).
- **`tools.selection`:** New required sub-object under `tools` holding pre-selection top-K / min / fallback and dynamic-cap settings in one place.
- **Config-shape refactor:** JSON schema and loader/handler wiring change only; selection algorithms and merge order stay as today.

## Scope (features/capabilities)

- Introduce **required** `tools.selection` with explicit fields (no implicit defaults):
  - `tool_search_top_k`, `tool_min_count`, `tool_fallback_cap` — same semantics and bounds as today’s top-level `tool_pre_selection` (≥ 1, existing upper caps).
  - `enabled` (bool) and `max_tools_for_llm_request` (int) — same semantics as today’s `tools.dynamic_selection` (when `enabled` is true: `max_tools_for_llm_request` ≥ 1 and ≥ distinct valid `always_include` count; when false, `max_tools_for_llm_request` may be 0).
- Remove top-level **`tool_pre_selection`** from the product root key list and config schema; reject it at load with a clear error.
- Remove **`tools.dynamic_selection`**; reject it at load.
- Update **`internal/config`** (`config.go`, `load.go`, `root_keys.go`), **`internal/core/run.go`** (and minimal handler field wiring in `handler.go` / `integration_export.go` only as needed to read the new struct — no broader handler refactor; that belongs to EP-038).
- Preserve runtime interaction: when `runtime_skills` is enabled, effective vector top-K remains `min(tool_search_top_k, runtime_skills.tool_vector_top_k_cap)` as today (`mergeSelectedToolIDs`).
- Update **`.config/config.json`**, all **`config.examples/`**, **`internal/config/testdata/`**, **`tests/integration/`** configs, and **`docs/configuration.md`** (including intent-tier table wording for the cap location).
- Add or adjust config-load and core tests proving **equivalent** configs (old field mapping documented for operators) yield the **same** merged tool id sets and dynamic-cap behaviour as before.
- Run **`make check`** green; epic validate target when registered.

## Out of scope / deferred

- **`tools.vector_search_tools` JSON DRY** (shared defaults + per-tool overrides): **deferred**. EP-032 already unified native vector-search tools under one block; the three per-tool objects differ mainly by `enabled` and operator tuning. A defaults/override schema adds validation and migration cost without changing selection behaviour; revisit only if operators ask for less repetition.
- Relocating **`runtime_skills.tool_vector_top_k_cap`** into `tools.selection` (stays under `runtime_skills`; document cross-field interaction only).
- Changing **`tools.always_include`**, **`tools.vector_search_tools`** runtime behaviour, tool-index algorithms, or intent-tier logic.
- **`internal/core` handler decomposition or large refactors** (planned EP-038).
- New selection features (tier-specific caps, new ranking signals).

## Success criteria

- Valid configs use **`tools.selection` only**; load fails on `tool_pre_selection`, `tools.dynamic_selection`, or unknown keys per explicit-JSON rules.
- **`configRootJSONKeys`** drops `tool_pre_selection`; every documented top-level key still appears exactly once in `.config/config.json` and examples.
- For representative configs, merged catalog tool ids and post-merge dynamic cap match pre-refactor behaviour (unit/integration coverage).
- Operator docs describe the single block and a one-to-one field migration from removed keys.
- **`make check`** passes.

## Traceability

- **Scope:** Reduces configuration surface around Core tool offering without changing the security model or tool contract ([scope.md](../../scope.md)).
- **Strategy:** **Refactoring 0.02** — remove extra architecture complexity; **direction B** (consolidate tool-selection config) ([strategy.md](../../strategy.md)).
- **Related:** EP-018 introduced dynamic cap; EP-032 unified native vector-search tools; EP-038 will refactor core handler structure separately.
