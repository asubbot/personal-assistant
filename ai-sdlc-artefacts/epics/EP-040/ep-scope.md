---
artefact: ep-scope
epic_id: EP-040
status: draft
source_of_truth: true
updated_at: 2026-05-31
git_branch: epic/EP-040-handler-dependency-grouping
---

# Epic scope — EP-040 Handler dependency grouping

| Field | Content |
|-------|---------|
| **ID** | EP-040 |
| **Status** | DONE |
| **Title** | Handler dependency grouping |
| **Description** | Group the ~25 fields of `conversationHandler` into named sub-structs (tools, memory, session, LLM/observability) to reduce cognitive load and simplify construction in `run.go`, without changing runtime behaviour or public APIs. Structural refactor continuing increment 0.02 after EP-038. |
| **First version date** | 2026-05-31 |
| **Git branch** | `epic/EP-040-handler-dependency-grouping` |

## Glossary

- **Dependency grouping:** Embedding related handler fields in unexported sub-structs on `conversationHandler` (e.g. `toolDeps`, `memoryDeps`); methods access fields via the group (`h.tools.catalog`).
- **Structural refactor:** Field layout and constructor changes only; no intentional prompt, routing, or config semantics changes.
- **Preserved contracts:** Behaviour from EP-013, EP-017/018/036, EP-034, EP-037, and EP-038 file split.

## Scope (features/capabilities)

- **Prerequisite gate:** Land after **EP-038** (handler file split) is merged; may follow **EP-039** if that epic touches core wiring for config fields.
- **Define sub-structs in `handler.go` (or `handler_deps.go`):**
  - **`handlerToolDeps`:** catalog, indexes, native registry, skill packages, `toolsCfg`, `toolsSelection`, pre-selection numeric fields, `nodeRunner`, `runtimeSkillsCfg`.
  - **`handlerMemoryDeps`:** `memVec`, `embedder`, `memoryVectorTopK`, `paLoc`.
  - **`handlerSessionDeps`:** `sessionCfg`, `sessionStore`.
  - **`handlerLLMDeps`:** `router`, `llmLog`, `model`, `firstProviderSupportsTools`, `logRedactor`, `logger`, `classifier`, `maxMessageLength`, `maxDynamicSystemRunes`.
- **Update method receivers:** Replace flat `h.field` access with grouped access across `handler.go`, `handler_llm.go`, `handler_tools.go`, `handler_memory.go`, `handler_tier_main_prompt.go`, and tests. Prefer accessor-free direct group access (KISS).
- **Simplify `newRunConversationHandler` in `run.go`:** Construct sub-structs in one place; reduce repetitive field assignment.
- **Preserve public API:** `MessageHandler`, `BuildMessageHandler`, `Run`, `NewIntegrationConversationHandler` unchanged.
- **Tests:** All existing handler tests pass without assertion changes except field-path updates in test helpers if any construct handlers directly.

## Out of scope / deferred

- New interfaces, dependency-injection frameworks, or exported sub-handler types.
- Renaming `conversationHandler` or moving groups to separate packages.
- Full-tier pipeline extraction (EP-041).
- Config schema changes.
- Behaviour changes to tier selection, tool merge, or LLM routing.

## Success criteria

- `conversationHandler` field list in `handler.go` is replaced by ≤5 named sub-struct fields plus any unavoidable top-level fields (target: zero flat dependency fields remaining).
- `newRunConversationHandler` LOC reduced measurably (order-of-magnitude: fewer than half the individual field assignments).
- **Behaviour parity:** Existing unit and integration tests pass unchanged in assertions.
- **`make check`** passes.

## Execution order

| Order | Epic | Branch |
|-------|------|--------|
| 1 | EP-039 Config surface | `epic/EP-039-config-surface-simplification` |
| 2 | **EP-040 (this epic)** | `epic/EP-040-handler-dependency-grouping` |
| 3 | EP-041 Full-tier pipeline | `epic/EP-041-full-tier-pipeline` |
| 4 | EP-042 Composition root | `epic/EP-042-composition-root-refinement` |
| 5 | EP-043 Test suite | `epic/EP-043-test-suite-organization` |

## Traceability

- **Scope:** Core maintainability ([scope.md](../../scope.md)).
- **Strategy:** **Refactoring 0.02** ([strategy.md](../../strategy.md)).
- **Prerequisites:** [EP-038](../EP-038/ep-scope.md) (file split); recommended after [EP-039](../EP-039/ep-scope.md) if EP-039 wires new config into handler.
- **Deferred in EP-038:** Sub-struct grouping explicitly out of scope there.
