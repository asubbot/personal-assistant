---
artefact: ep-context
epic_id: EP-039
status: draft
source_of_truth: false
updated_at: 2026-05-31
git_branch: epic/EP-039-config-surface-simplification
---

# Epic Context — EP-039 Config surface simplification

## Purpose

Reduce config duplication and phantom keys: DRY `tools.vector_search_tools`, type and wire `tools.tool_output_artifacts`, DRY SQLite reliability blocks.

## Current Scope

Three config refactors in one epic; legacy shapes rejected at load; equivalent-config parity tests; docs and testdata migration.

## Key Requirements

- REQ-39.001–009: `vector_search_tools` defaults + per-tool overrides
- REQ-39.010–016: `tool_output_artifacts` typed, validated, wired to core
- REQ-39.017–022: `sqlite_store_defaults` + per-store overrides
- REQ-39.023–025: migration, legacy rejection, verification

## Acceptance Signals

- Operator config loads; truncation uses config bytes; vector tools resolve same settings for equivalent configs; `make check` green.

## Design Decisions

- Pending stage 6. HITL default: wire `tool_result_prompt_bytes` first; other artifact fields validated even if some retention paths remain future work.

## Interfaces / Contracts

- `config.Load`, `ToolsConfig`, `VectorSearchToolsConfig`, `SQLiteStoreReliabilityConfig`, `conversationHandler` truncation limits.

## Current Gate Summary

Stage 3–5 artefacts drafted; stages 6–11 not started.

## Open Questions

- Whether per-tool overrides allow partial field omission (default: yes, merge with defaults at resolve time).
- Whether `jobs_store_reliability` / `vector_store_reliability` remain required root keys with minimal bodies or become optional when defaults suffice (default: remain required with at least `foreign_keys`).

## Links

- [ep-scope.md](ep-scope.md)
- [ep-requirements.md](ep-requirements.md)
- [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- Branch: `epic/EP-039-config-surface-simplification`
