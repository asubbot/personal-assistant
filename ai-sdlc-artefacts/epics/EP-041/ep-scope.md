---
artefact: ep-scope
epic_id: EP-041
status: draft
source_of_truth: true
updated_at: 2026-05-31
git_branch: epic/EP-041-full-tier-pipeline
---

# Epic scope — EP-041 Full-tier prompt pipeline

| Field | Content |
|-------|---------|
| **ID** | EP-041 |
| **Status** | DONE |
| **Title** | Full-tier prompt pipeline |
| **Description** | Make the tier-`full` main-LLM assembly path explicit via a single pipeline type or ordered step function, so maintainers can read the sequence (skills → merge tools → dynamic cap → tail budget → completion options) in one place without tracing multiple files. Structural refactor; no behaviour change. |
| **First version date** | 2026-05-31 |
| **Git branch** | `epic/EP-041-full-tier-pipeline` |

## Glossary

- **Full-tier pipeline:** Ordered assembly of prompt tail components for `intent.TierFull` before the first main LLM completion.
- **Pipeline step:** One function or method boundary with a named responsibility (e.g. `stepMergeTools`, `stepApplyDynamicCap`).
- **tailFitState:** Existing struct in `system_tail.go` holding merged tools, sources, chunks, and skills during budget fitting.

## Scope (features/capabilities)

- **Prerequisite gate:** Land after **EP-040** (dependency grouping) when possible, so pipeline code accesses grouped deps consistently.
- **Introduce `fullTierAssembler` (or equivalent)** in `internal/core` — unexported type holding `*conversationHandler` and turn inputs (`ctx`, `userText`, `sysHead`, `chunks`, `messages`).
- **Explicit step sequence** matching current order:
  1. Select skill packages (`selectSkillPackages`)
  2. Merge catalog tool ids (`mergeSelectedToolIDs`)
  3. Apply dynamic tool cap (`mergedAfterDynamicToolCap` / `pickToolsForMainRequest`)
  4. Fit dynamic tail to rune budget (`fitDynamicTailToBudget` + `buildDynamicTailString`)
  5. Build completion options (`completionOptionsMergedCatalogNative`)
- **Replace implicit call chain** in `buildTierFullMainPrompt` / `mergeTailMergedToolsAndOptions` with pipeline entry that documents steps inline (comments or step methods on the assembler).
- **Preserve outputs:** `tierMainLLMParams` and final `messages[0].Content` identical for same inputs.
- **Keep tier dispatch** in `handler_tier_main_prompt.go`; simple tier unchanged.
- **Tests:** Existing tier and handler tests pass; add one focused test or comment traceability for pipeline order if gaps exist.

## Out of scope / deferred

- New tiers, tool algorithms, or dynamic-cap logic changes.
- Moving `system_tail.go` or `dynamic_tool_selection.go` to other packages.
- Simple-tier pipeline (only full tier is in scope).
- Config changes.

## Success criteria

- A maintainer can read one type/file section listing all full-tier assembly steps in execution order.
- **Behaviour parity:** EP-017/018/036/037 regression tests and handler tier tests unchanged in assertions.
- **`make check`** passes.

## Execution order

| Order | Epic | Branch |
|-------|------|--------|
| 1–2 | EP-039, EP-040 | (prerequisites) |
| 3 | **EP-041 (this epic)** | `epic/EP-041-full-tier-pipeline` |
| 4–5 | EP-042, EP-043 | |

## Traceability

- **Strategy:** Refactoring 0.02 ([strategy.md](../../strategy.md)).
- **Prerequisites:** [EP-038](../EP-038/ep-scope.md), [EP-040](../EP-040/ep-scope.md) (recommended).
- **Related:** [EP-026](../EP-026/ep-scope.md) (tier builder extraction precedent).
