---
artefact: ep-scope
epic_id: EP-039
status: draft
source_of_truth: true
updated_at: 2026-05-31
git_branch: epic/EP-039-config-surface-simplification
---

# Epic scope — EP-039 Config surface simplification

| Field | Content |
|-------|---------|
| **ID** | EP-039 |
| **Status** | NEW |
| **Title** | Config surface simplification |
| **Description** | Reduce operator and repository config duplication by introducing DRY schemas for `tools.vector_search_tools` and SQLite store reliability blocks, and by typing and wiring `tools.tool_output_artifacts` so phantom JSON keys are validated and consumed at runtime. Part of Refactoring increment 0.02 (post EP-037/038). |
| **First version date** | 2026-05-31 |
| **Git branch** | `epic/EP-039-config-surface-simplification` |

## Glossary

- **DRY defaults + overrides:** A shared `defaults` object holds common field values; per-tool or per-store objects list only fields that differ from defaults.
- **Phantom config:** JSON present in operator config and accepted by the whitelist but not mapped to a Go struct or runtime behaviour (today: `tools.tool_output_artifacts`).
- **Equivalent configuration:** Operator settings that produce the same resolved runtime values before and after schema migration (field-for-field mapping documented in operator docs).
- **Explicit JSON configuration:** Every documented key appears exactly once at its level; unknown keys fail load; optional product blocks use JSON `null` at the root.

## Scope (features/capabilities)

- **Prerequisite gate:** Land only after **EP-037** (`tools.selection`) and **EP-038** (handler file split) are merged to the integration branch.
- **`tools.vector_search_tools` DRY (REQ-39.001–009):** Replace three repeated five-field tool objects with a required `defaults` sub-object plus three per-tool override objects (`search_vector_memory`, `search_vector_tool`, `search_vector_skill`) that may specify only `enabled` and optional field overrides. Reject the legacy flat repeated shape at load. Preserve resolved settings and native tool behaviour for equivalent operator configs.
- **`tools.tool_output_artifacts` typed + wired (REQ-39.010–016):** Add `ToolOutputArtifactsConfig` to `ToolsConfig`, validate all documented fields at load, and wire `tool_result_prompt_bytes` (and other fields where runtime already has hardcoded equivalents) into `internal/core` tool-result truncation and artifact paths. Remove parsed-but-ignored behaviour.
- **SQLite reliability DRY (REQ-39.017–022):** Introduce a required root-level `sqlite_store_defaults` block with shared PRAGMA fields (`journal_mode`, `busy_timeout`, `synchronous`). Reduce `vector_store_reliability` and `jobs_store_reliability` to per-store overrides (at minimum `foreign_keys`; allow full override when operator needs divergence). Reject duplicate-only blocks at load. Preserve effective PRAGMA policy for equivalent configs.
- **Migration:** Update `config.examples/`, `internal/config/testdata/`, integration configs, operator docs (`docs/configuration.md`), and provide a documented operator migration table from pre-EP-039 shapes.
- **Legacy rejection:** Fail-fast on pre-EP-039 `vector_search_tools` shape and on removed redundant reliability fields when defaults cover them (explicit error messages naming the unsupported key path).
- **Verification:** `make check` green; epic validate target when registered.

## Out of scope / deferred

- Changes to tool-selection algorithms, intent tiers, handler file layout, or LLM routing.
- New tool-output artifact features (retention sweeper behaviour beyond wiring existing config fields).
- Relocating `runtime_skills.tool_vector_top_k_cap`.
- Broad `internal/config/load.go` refactors unrelated to the three config areas above.
- Performance, load, or multi-tenant scaling work.

## Success criteria

- Operator `.config/config.json` and all repository example/test configs load under the new schemas without behaviour regression for equivalent settings.
- `tools.vector_search_tools` JSON no longer repeats identical five-field blocks when all three tools share defaults.
- `tools.tool_output_artifacts` is validated at load and at least `tool_result_prompt_bytes` drives runtime truncation (replacing the hardcoded `maxToolResultPromptBytes` constant where they differ).
- SQLite reliability configs share defaults; per-store blocks differ only where semantically required (`foreign_keys` and optional overrides).
- Legacy config shapes fail load with explicit, actionable errors.
- **`make check`** passes.

## Execution order (increment 0.02 continuation)

| Order | Epic | Branch |
|-------|------|--------|
| 1 | EP-039 (this epic) | `epic/EP-039-config-surface-simplification` |
| 2 | EP-040 Handler dependency grouping | `epic/EP-040-handler-dependency-grouping` |
| 3 | EP-041 Full-tier prompt pipeline | `epic/EP-041-full-tier-pipeline` |
| 4 | EP-042 Composition root refinement | `epic/EP-042-composition-root-refinement` |
| 5 | EP-043 Test suite organization | `epic/EP-043-test-suite-organization` |

Each epic merges to the integration branch (real merge, no fast-forward) before the next epic branch is cut.

## Traceability

- **Scope:** Maintainability and explicit configuration ([scope.md](../../scope.md)).
- **Strategy:** **Refactoring 0.02** — remove extra architecture complexity ([strategy.md](../../strategy.md)).
- **Prerequisites:** [EP-037](../EP-037/ep-scope.md) (deferred `vector_search_tools` DRY and `tool_output_artifacts` typing), [EP-038](../EP-038/ep-scope.md) (handler split complete).
- **Architecture:** Addresses config duplication and phantom keys noted in [pa-architecture-review.md](../../pa-architecture-review.md) and post-EP-037 follow-ups.
