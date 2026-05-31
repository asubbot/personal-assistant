---
artefact: ep-scope
epic_id: EP-038
status: draft
source_of_truth: true
updated_at: 2026-05-31
---

# Epic scope — EP-038 Refactor core conversation handler (god handler)

| Field | Content |
|-------|---------|
| **ID** | EP-038 |
| **Status** | DONE |
| **Title** | Refactor core conversation handler (god handler) |
| **Description** | Decompose `internal/core/handler.go` (~663 LOC) into focused files and clearer tier/prompt boundaries while preserving all runtime behaviour, tests, and explicit JSON configuration. Part of Refactoring increment 0.02 (architecture-simplification direction F). |
| **First version date** | 2026-05-30 |

## Glossary

- **God handler:** `conversationHandler` in `internal/core/handler.go` that orchestrates intent tiering, RAG retrieval, tool pre-selection, dynamic tail, LLM completion, tool rounds, logging, and turn indexing in one file.
- **Structural refactor:** Move and group methods/types only; no intentional change to prompts, tool lists, routing, or config semantics.
- **Tier main prompt path:** Pre-main-LLM flow in `handler_tier_main_prompt.go`: classify tier → base messages → full-tier tail (skills, tools, dynamic tail budget) → `tierMainLLMParams`.
- **Preserved contracts:** Behaviour and observability established by EP-013 (trust/marker system tail), EP-017/018/036 (two-tier intent + full-tier assembly), EP-034 (no tool-path provider escalation; transport fallback only), and EP-037 (`tools.selection` merge/cap wiring).

## Scope (features/capabilities)

- **Prerequisite gate:** Land only after **EP-035** (package consolidation), **EP-036** (two-tier intent), and **EP-037** (`tools.selection`) are merged to the integration branch so handler splits do not fight concurrent core edits.
- **Slim orchestration entry (`handler.go`):** Keep `conversationHandler` struct definition, `HandleMessage` turn sequence (`checkUserMessage` → `buildMainTurnMessagesPreTail` → `assembleTierMainLLMParams` → `completeAt` → `finishAfterFirstLLM` → usage footer), session-window helpers, and shared turn constants (`maxToolRounds`, log truncation limits). Target: `handler.go` is primarily orchestration (~≤200 LOC), not implementation detail.
- **Extract LLM completion + tool loop (`handler_llm.go`, new):** Move from `handler.go`: `completeViaRouter`, `completeAt`, `onRouteEvent`, `finishAfterFirstLLM`, `runToolResultLoop`, `appendToolRound`, `truncateToolResultForPrompt`, `systemStaticHead`, `todayCalendarDateInPALocation`, `paLocationFromConfig`, `genRequestID`, and LLM observability (`logLLMRequest`, `logMainLLMCompletion`, `logLLMResponse`, `logMainLLMPromptAssembled`). No change to router usage, round cap, or message roles.
- **Extract tool offering + execution (`handler_tools.go`, new):** Move from `handler.go`: `mergeSelectedToolIDs`, `selectSkillPackages`, `completionOptionsMergedCatalogNative`, `nativeToolDefs`, `executeOneToolCall`, `executeCatalogToolCall`, `parseToolArgumentsJSON`, `remoteCommandFromRunOnNodeArgs`. Keep existing helpers in place: `runtime_tools.go` (`mergeToolIDs`), `dynamic_tool_selection.go` (`pickToolsForMainRequest`, cap), `handler_tier_main_prompt.go` (`mergedAfterDynamicToolCap`, tail merge).
- **Extract memory retrieval + turn indexing (`handler_memory.go`, new):** Move from `handler.go`: `gatherRetrievedChunkTexts`, `gatherSplitTableChunks`, `labeledChunksFromResults`, `retrievalChunkWithLabel`, `handleLLMSuccess` (llmLog + debug logs + turn index), `indexTurn`, `eventAlignedTurnDate`, `canonicalizeTurnPair`, `canonicalizeTurnText`. Keep `vector_merge.go`, `memory_vectors.go`, and `system_tail.go` ownership unchanged.
- **Tier/prompt boundary (`handler_tier_main_prompt.go`, existing):** Retain tier dispatch (`buildMainTurnMessagesPreTail`, `assembleTierMainLLMParams`, `buildTierSimpleMainPrompt`, `buildTierFullMainPrompt`, `mergeTailMergedToolsAndOptions`). Optional light cleanup only (naming/comments, unexported helpers)—**no** new tier values, no revival of `full_lite`, no third prompt path. Prefer a small `switch tier` dispatch over a heavy strategy framework (two tiers after EP-036).
- **Wiring unchanged externally:** `MessageHandler`, `BuildMessageHandler` / `Run` in `run.go`, and `NewIntegrationConversationHandler` in `integration_export.go` keep the same public surfaces; adjust imports/field wiring only if file moves require it. **No** `config.json` schema changes (prefer none; if unavoidable, document as explicit HITL exception—default is zero schema change).
- **Tests:** All existing `internal/core/handler_*` and integration handler tests pass without assertion changes except import/package-location fixes. No deletion of behavioural coverage; move test helpers only when tied to moved unexported symbols.
- **Verification:** `make check` green; epic validate target when registered.

## Out of scope / deferred

- **Product behaviour changes:** New tiers, tool-selection algorithms, router policies, memory formats, or Telegram/session semantics.
- **`tools.vector_search_tools` JSON DRY** and **`tools.tool_output_artifacts` typed validation** (deferred in EP-037; follow-up epic if needed).
- **Package moves outside `internal/core`** beyond minimal import updates in `cmd/pa` if a symbol moves packages (EP-035 already consolidated `internal/prompt`; do not reopen EP-035 scope).
- **New microservices, interfaces for every concern, or plugin registries** for tier/tool/LLM subsystems.
- **Async init / scheduler / SQLite concurrency** weaknesses called out in architecture docs.
- **Renaming `conversationHandler`** or exporting sub-handlers to other packages.

## Success criteria

- `handler.go` is reduced to orchestration + struct wiring (order-of-magnitude shrink from ~663 LOC; implementation lives in the new/existing focused files listed above).
- **Behaviour parity:** Representative configs and existing unit/integration tests show unchanged tier choice, merged tool ids, dynamic cap, prompt message shapes, tool-loop round limits, transport-only LLM routing, and turn-index id format.
- **Contracts:** EP-013 marker/trust tail assembly, EP-036 two-tier dispatch, EP-037 `tools.selection` fields, and EP-034 no tool-path escalation remain satisfied (existing EP-034/036/037 regression tests still pass).
- **Config:** No new or removed top-level keys; `.config/config.json` and examples load unchanged.
- **`make check`** passes.

## Traceability

- **Scope:** Improves maintainability of Core orchestration without changing the PersonalAssistant security model or tool contract ([scope.md](../../scope.md)).
- **Strategy:** **Refactoring 0.02** — remove extra architecture complexity; **direction F** (last in sequence E→C→B→F) ([strategy.md](../../strategy.md)).
- **Architecture:** Addresses the documented weakness “god handler” and the stated next step to split tier logic for maintainability ([pa-architecture-review.md](../../pa-architecture-review.md); product code remains source of truth).
- **Prerequisites:** [EP-035](../EP-035/ep-scope.md) (internal package consolidation), [EP-036](../EP-036/ep-scope.md) (two-tier intent), [EP-037](../EP-037/ep-scope.md) (tools.selection; explicitly deferred handler decomposition to this epic).
- **Related:** [EP-034](../EP-034/ep-scope.md) (escalation removal), [EP-026](../EP-026/ep-scope.md) (tier main prompt extraction precedent).
