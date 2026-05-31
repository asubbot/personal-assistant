---
artefact: ep-system-design
epic_id: EP-041
status: draft
source_of_truth: true
updated_at: 2026-05-31
---

# EP-040 — Full-tier prompt pipeline — System design

## Overview

Introduce `fullTierAssembler` in `internal/core/handler_full_tier_pipeline.go` with explicit five-step pipeline for tier `full` ([ep-scope.md](ep-scope.md)).

## Pipeline steps (fixed order)

1. `stepSelectSkills` → `selectSkillPackages`
2. `stepMergeTools` → `mergeSelectedToolIDs`
3. `stepApplyDynamicCap` → `mergedAfterDynamicToolCap`
4. `stepFitTailBudget` → `fitDynamicTailToBudget` + `buildDynamicTailString`
5. `stepBuildCompletionOptions` → `completionOptionsMergedCatalogNative`

Entry: `buildTierFullMainPrompt` calls `newFullTierAssembler(h, ctx, ...).run()`.

`mergeTailMergedToolsAndOptions` logic moves into assembler; deprecated or thin wrapper.

## Files

| File | Action |
|------|--------|
| `handler_full_tier_pipeline.go` | New — type + steps |
| `handler_tier_main_prompt.go` | Delegate full tier to assembler |
| Tests | Parity unchanged |

## REQ traceability

REQ-41.001–008 covered by pipeline type, step order, parity tests, no config change.
