---
artefact: ep-acceptance-criteria
epic_id: EP-038
status: draft
source_of_truth: true
updated_at: 2026-05-31
---

# EP-038 — Refactor core conversation handler (god handler) — Acceptance criteria

## Introduction

Testable acceptance criteria for **EP-038**: decompose `internal/core/handler.go` into `handler_llm.go`, `handler_tools.go`, and `handler_memory.go` while keeping `handler.go` as slim orchestration, preserving runtime behaviour, explicit JSON configuration, and contracts from EP-013, EP-017/018, EP-034, EP-036, and EP-037. Criteria trace to [ep-requirements.md](ep-requirements.md) and [ep-scope.md](ep-scope.md). Test levels follow [strategy.md](../../strategy.md) §2 (Unit / Integration / E2E / Manual).

---

## Acceptance criteria index

| AC ID | REQ (trace) | Test level | Summary |
|-------|-------------|------------|---------|
| [AC-38.001](#ac-38-001) | [REQ-38.001](ep-requirements.md#prerequisites-and-landing-gate) | Manual | EP-035/036/037 merged before EP-038 lands |
| [AC-38.002](#ac-38-002) | [REQ-38.002](ep-requirements.md#slim-orchestration-entry-handlergo) | Integration | `HandleMessage` turn sequence unchanged |
| [AC-38.003](#ac-38-003) | [REQ-38.003](ep-requirements.md#slim-orchestration-entry-handlergo) | Manual | `handler.go` ≤~200 LOC orchestration |
| [AC-38.004](#ac-38-004) | [REQ-38.004](ep-requirements.md#slim-orchestration-entry-handlergo) | Unit | Turn constants remain in `handler.go` |
| [AC-38.005](#ac-38-005) | [REQ-38.005](ep-requirements.md#llm-completion-and-tool-loop-handler_llmgo) | Manual (grep) | LLM/tool-loop methods live in `handler_llm.go` |
| [AC-38.006](#ac-38-006) | [REQ-38.006](ep-requirements.md#llm-completion-and-tool-loop-handler_llmgo) | Integration | Router, round cap, roles, LLM logs preserved |
| [AC-38.007](#ac-38-007) | [REQ-38.007](ep-requirements.md#tool-offering-and-execution-handler_toolsgo) | Manual (grep) | Tool merge/execution methods in `handler_tools.go` |
| [AC-38.008](#ac-38-008) | [REQ-38.008](ep-requirements.md#tool-offering-and-execution-handler_toolsgo) | Manual (grep) | `runtime_tools` / `dynamic_tool_selection` / tier tail ownership unchanged |
| [AC-38.009](#ac-38-009) | [REQ-38.009](ep-requirements.md#memory-retrieval-and-turn-indexing-handler_memorygo) | Manual (grep) | Memory/index methods live in `handler_memory.go` |
| [AC-38.010](#ac-38-010) | [REQ-38.010](ep-requirements.md#memory-retrieval-and-turn-indexing-handler_memorygo) | Manual | `vector_merge` / `memory_vectors` / `system_tail` ownership unchanged |
| [AC-38.011](#ac-38-011) | [REQ-38.011](ep-requirements.md#tier-main-prompt-boundary) | Integration | Tier main-prompt dispatch retained in `handler_tier_main_prompt.go` |
| [AC-38.012](#ac-38-012) | [REQ-38.012](ep-requirements.md#tier-main-prompt-boundary) | Unit | No new tiers; `full_lite` absent |
| [AC-38.013](#ac-38-013) | [REQ-38.013](ep-requirements.md#tier-main-prompt-boundary) | Manual (grep) | `simple` / `full` direct switch; no strategy framework |
| [AC-38.014](#ac-38-014) | [REQ-38.014](ep-requirements.md#public-wiring-and-configuration) | Integration | `MessageHandler.HandleMessage` contract unchanged |
| [AC-38.015](#ac-38-015) | [REQ-38.015](ep-requirements.md#public-wiring-and-configuration) | Unit | `BuildMessageHandler` / `Run` surfaces unchanged |
| [AC-38.016](#ac-38-016) | [REQ-38.016](ep-requirements.md#public-wiring-and-configuration) | Integration | `NewIntegrationConversationHandler` / `IntegrationIndexTurn` unchanged |
| [AC-38.017](#ac-38-017) | [REQ-38.017](ep-requirements.md#public-wiring-and-configuration) | Unit | No `config.json` schema change |
| [AC-38.018](#ac-38-018) | [REQ-38.018](ep-requirements.md#verification-and-contracts) | Integration | Behaviour parity: tier, tools, prompts, routing, turn ids |
| [AC-38.019](#ac-38-019) | [REQ-38.019](ep-requirements.md#verification-and-contracts) | Integration | EP-013/034/036/037 contracts preserved |
| [AC-38.020](#ac-38-020) | [REQ-38.020](ep-requirements.md#verification-and-contracts) | Integration | Existing `handler_*` tests pass unchanged |
| [AC-38.021](#ac-38-021) | [REQ-38.021](ep-requirements.md#verification-and-contracts) | Manual (make check) | `make check` exits zero |
| [AC-38.022](#ac-38-022) | [REQ-38.022](ep-requirements.md#verification-and-contracts) | Manual (validate) | `./tools/validate/validate ears EP-038` passes |
| [AC-38.023](#ac-38-023) | [REQ-38.023](ep-requirements.md#verification-and-contracts) | Manual | No intentional product behaviour change in diff |
| [AC-38.024](#ac-38-024) | [REQ-38.024](ep-requirements.md#verification-and-contracts) | Manual (grep) | `conversationHandler` stays unexported |
| [AC-38.025](#ac-38-025) | [REQ-38.025](ep-requirements.md#tier-main-prompt-boundary) | Manual | Optional tier-file cleanup is naming/comments only |

---

## Scenarios

### AC-38.002 HandleMessage turn sequence (Trace: REQ-38.002)

Given a test `conversationHandler` with mocked LLM and dependencies  
When `HandleMessage` runs for a representative user turn  
Then the call order SHALL remain `checkUserMessage` → `buildMainTurnMessagesPreTail` → `assembleTierMainLLMParams` → `completeAt` → `finishAfterFirstLLM` (and usage footer when configured).

**Automated test hint:** `internal/core/handler_test.go`, `internal/core/handler_ep017_test.go` (`TestHandleMessage_*` integration-style cases).

### AC-38.006 LLM router and tool rounds (Trace: REQ-38.006)

Given two configured LLM providers and a tool failure during a tool round  
When the handler completes subsequent LLM calls for the same user message  
Then provider index SHALL NOT advance on tool failure  
And transport retryable errors SHALL still switch provider per router policy  
And tool-result rounds SHALL not exceed `maxToolRounds` (10).

**Automated test hint:** `internal/core/handler_ep034_regression_test.go`; `internal/llmrouter/router_test.go` for transport fallback.

### AC-38.011 Tier main-prompt dispatch (Trace: REQ-38.011)

Given enabled intent classification and tier `simple` or `full`  
When main-turn messages are assembled before the main LLM call  
Then `buildMainTurnMessagesPreTail`, `assembleTierMainLLMParams`, and tier builders in `handler_tier_main_prompt.go` SHALL produce the same prompt shapes as before EP-038 for each tier.

**Automated test hint:** `internal/core/handler_tier_main_prompt_test.go`; `internal/core/handler_ep017_test.go`; `internal/core/handler_ep018_test.go`.

### AC-38.018 Behaviour parity (Trace: REQ-38.018)

Given representative configurations and fixture tables from EP-017/018/037 parity tests  
When tier choice, tool merge, dynamic cap, main prompt assembly, and turn indexing run on the epic branch  
Then outcomes SHALL match pre-EP-038 baselines for the same fixtures.

**Automated test hint:** `internal/core/tools_selection_parity_test.go`; `internal/core/handler_ep017_test.go`; `internal/core/handler_ep018_test.go`; `internal/core/handler_ep018_coverage_test.go`.

### AC-38.019 Preserved epic contracts (Trace: REQ-38.019)

Given EP-038 implementation on the integration branch  
When running EP-034, EP-036, and EP-037 regression suites  
Then transport-only routing, two-tier dispatch, and `tools.selection` wiring contracts SHALL remain satisfied.

**Automated test hint:** `internal/core/handler_ep034_regression_test.go`; `internal/core/handler_ep036_test.go`; `internal/core/tools_selection_parity_test.go`; `internal/config/tools_selection_test.go`.

### AC-38.020 Existing handler tests (Trace: REQ-38.020)

Given EP-038 file moves without assertion changes (except import fixes)  
When `go test ./internal/core/...` runs  
Then all `handler_*` and integration handler tests SHALL pass.

**Automated test hint:** full `internal/core` package tests via `make check`.

---

## Acceptance criteria

<a id="ac-38-001"></a>

### AC-38.001

**Trace:** [REQ-38.001](ep-requirements.md#prerequisites-and-landing-gate)  
**Test level:** Manual  
**Status:** AC-38.001 MANUAL ONLY — verified by confirming EP-035, EP-036, and EP-037 are merged on the integration branch before EP-038 implementation lands (merge-base / branch history inspection).

Given EP-038 implementation is ready to land  
When reviewing the integration branch history or merge prerequisites  
Then EP-035 (package consolidation), EP-036 (two-tier intent), and EP-037 (`tools.selection`) SHALL already be merged  
So handler file splits do not conflict with concurrent core edits from those epics.

---

<a id="ac-38-002"></a>

### AC-38.002

**Trace:** [REQ-38.002](ep-requirements.md#slim-orchestration-entry-handlergo)  
**Test level:** Integration

Given a configured `conversationHandler` in tests  
When `HandleMessage` processes a user turn  
Then `conversationHandler` struct definition and `HandleMessage` SHALL remain in `internal/core/handler.go`  
And the turn pipeline SHALL follow `checkUserMessage` → `buildMainTurnMessagesPreTail` → `assembleTierMainLLMParams` → `completeAt` → `finishAfterFirstLLM` (plus optional usage footer).

**Automated test hint:** `internal/core/handler_test.go`, `internal/core/handler_ep017_test.go`.

---

<a id="ac-38-003"></a>

### AC-38.003

**Trace:** [REQ-38.003](ep-requirements.md#slim-orchestration-entry-handlergo)  
**Test level:** Manual  
**Status:** AC-38.003 MANUAL ONLY — verified by line-count inspection of `internal/core/handler.go` (orchestration and shared wiring only, approximately ≤200 LOC) after extraction completes.

Given EP-038 implementation is complete  
When inspecting `internal/core/handler.go`  
Then the file SHALL contain primarily orchestration (struct, `HandleMessage`, session-window helpers, shared constants)  
And implementation detail SHALL live mainly in `handler_llm.go`, `handler_tools.go`, `handler_memory.go`, and existing focused files.

---

<a id="ac-38-004"></a>

### AC-38.004

**Trace:** [REQ-38.004](ep-requirements.md#slim-orchestration-entry-handlergo)  
**Test level:** Unit

Given EP-038 implementation is complete  
When inspecting `internal/core/handler.go`  
Then turn-wide constants `maxToolRounds`, `logTruncateMaxLen`, and `maxToolResultPromptBytes` SHALL be defined in `handler.go` (or co-located with orchestration in that file)  
And `maxToolRounds` SHALL remain **10**.

**Automated test hint:** compile-time const usage in `handler_ep034_regression_test.go` and tool-loop tests; `make check`.

---

<a id="ac-38-005"></a>

### AC-38.005

**Trace:** [REQ-38.005](ep-requirements.md#llm-completion-and-tool-loop-handler_llmgo)  
**Test level:** Manual (grep)  
**Status:** AC-38.005 MANUAL ONLY — verified by grep or `go doc` that listed methods are defined on `conversationHandler` in `internal/core/handler_llm.go` and absent as implementations from `handler.go`.

Given EP-038 implementation is complete  
When inspecting `internal/core/handler_llm.go`  
Then it SHALL define (as methods on `conversationHandler`): `completeViaRouter`, `completeAt`, `onRouteEvent`, `finishAfterFirstLLM`, `runToolResultLoop`, `appendToolRound`, `truncateToolResultForPrompt`, `systemStaticHead`, `todayCalendarDateInPALocation`, `paLocationFromConfig`, `genRequestID`, `logLLMRequest`, `logMainLLMCompletion`, `logLLMResponse`, and `logMainLLMPromptAssembled`.

---

<a id="ac-38-006"></a>

### AC-38.006

**Trace:** [REQ-38.006](ep-requirements.md#llm-completion-and-tool-loop-handler_llmgo)  
**Test level:** Integration

Given EP-038 moves LLM completion code to `handler_llm.go`  
When exercising tool rounds, transport failures, and LLM logging in existing regression tests  
Then unified `llmrouter` usage, transport-only provider switching via `onRouteEvent`, `maxToolRounds` enforcement, tool-round message roles, and DEBUG/INFO LLM observability semantics SHALL match pre-EP-038 behaviour.

**Automated test hint:** `internal/core/handler_ep034_regression_test.go`; `internal/core/handler_test.go`; `make check`.

---

<a id="ac-38-007"></a>

### AC-38.007

**Trace:** [REQ-38.007](ep-requirements.md#tool-offering-and-execution-handler_toolsgo)  
**Test level:** Manual (grep)  
**Status:** AC-38.007 MANUAL ONLY — verified by grep that listed tool methods are defined in `internal/core/handler_tools.go`.

Given EP-038 implementation is complete  
When inspecting `internal/core/handler_tools.go`  
Then it SHALL define: `mergeSelectedToolIDs`, `selectSkillPackages`, `completionOptionsMergedCatalogNative`, `nativeToolDefs`, `executeOneToolCall`, `executeCatalogToolCall`, `parseToolArgumentsJSON`, and `remoteCommandFromRunOnNodeArgs`.

---

<a id="ac-38-008"></a>

### AC-38.008

**Trace:** [REQ-38.008](ep-requirements.md#tool-offering-and-execution-handler_toolsgo)  
**Test level:** Manual (grep)  
**Status:** AC-38.008 MANUAL ONLY — verified by grep that `mergeToolIDs` remains in `runtime_tools.go`, dynamic cap logic in `dynamic_tool_selection.go`, and tail merge helpers in `handler_tier_main_prompt.go`.

Given EP-038 implementation is complete  
When inspecting file ownership under `internal/core`  
Then `mergeToolIDs` SHALL remain in `runtime_tools.go`  
And dynamic pre-main cap logic SHALL remain in `dynamic_tool_selection.go`  
And `mergedAfterDynamicToolCap` / `mergeTailMergedToolsAndOptions` SHALL remain in `handler_tier_main_prompt.go`.

---

<a id="ac-38-009"></a>

### AC-38.009

**Trace:** [REQ-38.009](ep-requirements.md#memory-retrieval-and-turn-indexing-handler_memorygo)  
**Test level:** Manual (grep)  
**Status:** AC-38.009 MANUAL ONLY — verified by grep that listed memory/index methods are defined in `internal/core/handler_memory.go`.

Given EP-038 implementation is complete  
When inspecting `internal/core/handler_memory.go`  
Then it SHALL define: `gatherRetrievedChunkTexts`, `gatherSplitTableChunks`, `labeledChunksFromResults`, `retrievalChunkWithLabel`, `handleLLMSuccess`, `indexTurn`, `eventAlignedTurnDate`, `canonicalizeTurnPair`, and `canonicalizeTurnText`.

---

<a id="ac-38-010"></a>

### AC-38.010

**Trace:** [REQ-38.010](ep-requirements.md#memory-retrieval-and-turn-indexing-handler_memorygo)  
**Test level:** Manual  
**Status:** AC-38.010 MANUAL ONLY — verified by reviewing the EP-038 branch diff for `vector_merge.go`, `memory_vectors.go`, and `system_tail.go` (no responsibility relocation beyond import fixes).

Given EP-038 implementation is complete  
When reviewing `internal/core/vector_merge.go`, `memory_vectors.go`, and `system_tail.go`  
Then ownership and responsibilities of those files SHALL be unchanged by this epic.

---

<a id="ac-38-011"></a>

### AC-38.011

**Trace:** [REQ-38.011](ep-requirements.md#tier-main-prompt-boundary)  
**Test level:** Integration

Given tier main-prompt assembly after EP-038  
When inspecting `handler_tier_main_prompt.go` and running tier tests  
Then `buildMainTurnMessagesPreTail`, `assembleTierMainLLMParams`, `buildTierSimpleMainPrompt`, `buildTierFullMainPrompt`, and `mergeTailMergedToolsAndOptions` SHALL remain in that file with equivalent behaviour.

**Automated test hint:** `internal/core/handler_tier_main_prompt_test.go`; `internal/core/handler_ep017_test.go`; `internal/core/handler_ep018_test.go`.

---

<a id="ac-38-012"></a>

### AC-38.012

**Trace:** [REQ-38.012](ep-requirements.md#tier-main-prompt-boundary)  
**Test level:** Unit

Given EP-038 implementation is complete  
When searching production tier dispatch under `internal/core`  
Then no new intent tier values SHALL be introduced  
And `full_lite` / `TierFullLite` SHALL NOT be revived.

**Automated test hint:** `internal/core/handler_ep036_test.go`; `internal/intent` tier tests via `make check`.

---

<a id="ac-38-013"></a>

### AC-38.013

**Trace:** [REQ-38.013](ep-requirements.md#tier-main-prompt-boundary)  
**Test level:** Manual (grep)  
**Status:** AC-38.013 MANUAL ONLY — verified by reading `assembleTierMainLLMParams` (or equivalent) for a direct `simple` / `full` switch and absence of pluggable tier-strategy interfaces.

Given EP-038 implementation is complete  
When inspecting tier dispatch in `handler_tier_main_prompt.go`  
Then intent tiers `simple` and `full` SHALL be dispatched with a direct switch (or equivalent minimal dispatch)  
And SHALL NOT use a pluggable tier-strategy framework or registry.

---

<a id="ac-38-014"></a>

### AC-38.014

**Trace:** [REQ-38.014](ep-requirements.md#public-wiring-and-configuration)  
**Test level:** Integration

Given EP-038 implementation is complete  
When compiling packages that depend on `MessageHandler`  
Then `internal/core/adapter.go` SHALL still declare `HandleMessage(ctx, userID, sessionKey, text) (string, error)` unchanged.

**Automated test hint:** `go test ./internal/core/...` and `go test ./cmd/pa/...` via `make check`.

---

<a id="ac-38-015"></a>

### AC-38.015

**Trace:** [REQ-38.015](ep-requirements.md#public-wiring-and-configuration)  
**Test level:** Unit

Given EP-038 implementation is complete  
When inspecting `internal/core/run.go` and running `run_test.go`  
Then `BuildMessageHandler` and `Run` public signatures and behaviour SHALL be preserved (import or field wiring fixes only).

**Automated test hint:** `internal/core/run_test.go`; `make check`.

---

<a id="ac-38-016"></a>

### AC-38.016

**Trace:** [REQ-38.016](ep-requirements.md#public-wiring-and-configuration)  
**Test level:** Integration

Given EP-038 implementation is complete  
When integration tests construct `NewIntegrationConversationHandler` and call `IntegrationIndexTurn`  
Then those public surfaces in `integration_export.go` SHALL behave as before the refactor.

**Automated test hint:** `tests/integration/runtime_skills_handler_test.go`; `make check`.

---

<a id="ac-38-017"></a>

### AC-38.017

**Trace:** [REQ-38.017](ep-requirements.md#public-wiring-and-configuration)  
**Test level:** Unit

Given the EP-038 change set  
When config load runs for `config.examples/`, `internal/config/testdata/`, and integration fixtures  
Then no top-level `config.json` keys SHALL be added or removed  
And documented configuration semantics SHALL remain valid.

**Automated test hint:** `internal/config/tools_selection_test.go`; `internal/config/config_test.go`; `make check`.

---

<a id="ac-38-018"></a>

### AC-38.018

**Trace:** [REQ-38.018](ep-requirements.md#verification-and-contracts)  
**Test level:** Integration

Given representative configurations on the epic branch  
When running existing parity and handler regression tests  
Then intent tier choice, merged catalog tool ids, dynamic tool cap, main prompt message shapes, tool-loop round limits, transport-only LLM routing, and turn-index id format (`turn:<date>:<12-byte-hex-prefix>`) SHALL match pre-EP-038 baselines.

**Automated test hint:** `internal/core/tools_selection_parity_test.go`; `internal/core/handler_ep017_test.go`; `internal/core/handler_ep018_test.go`; `internal/core/handler_ep018_coverage_test.go`.

---

<a id="ac-38-019"></a>

### AC-38.019

**Trace:** [REQ-38.019](ep-requirements.md#verification-and-contracts)  
**Test level:** Integration

Given EP-038 implementation is complete  
When running contract regression tests from prior epics  
Then EP-013 marker/trust system tail assembly, EP-036 two-tier dispatch, EP-037 `tools.selection` handler wiring, and EP-034 prohibition of tool-path LLM provider escalation SHALL remain satisfied.

**Automated test hint:** `internal/core/handler_ep034_regression_test.go`; `internal/core/handler_ep036_test.go`; `internal/core/tools_selection_parity_test.go`; EP-013 tests under `internal/core` and `internal/prompt` via `make check`.

---

<a id="ac-38-020"></a>

### AC-38.020

**Trace:** [REQ-38.020](ep-requirements.md#verification-and-contracts)  
**Test level:** Integration

Given EP-038 moves methods across files  
When the test suite runs  
Then all existing `internal/core/handler_*` and integration handler tests SHALL pass without assertion changes except import or package-location fixes tied to moved unexported symbols.

**Automated test hint:** `go test ./internal/core/...`; `tests/integration/runtime_skills_handler_test.go`; `make check`.

---

<a id="ac-38-021"></a>

### AC-38.021

**Trace:** [REQ-38.021](ep-requirements.md#verification-and-contracts)  
**Test level:** Manual (make check)  
**Status:** AC-38.021 MANUAL ONLY — verified by running `make check` from the repository root (exit 0); this is a process gate, not a unit test.

Given EP-038 implementation is complete on the epic branch  
When `make check` runs from the repository root  
Then it SHALL exit with status zero.

---

<a id="ac-38-022"></a>

### AC-38.022

**Trace:** [REQ-38.022](ep-requirements.md#verification-and-contracts)  
**Test level:** Manual (validate)  
**Status:** AC-38.022 MANUAL ONLY — verified by running `./tools/validate/validate ears EP-038` from the repository root after `make build`; this is an artefact gate, not a product unit test.

Given `ep-requirements.md` for EP-038 on the epic branch  
When `./tools/validate/validate ears EP-038` runs from the repository root  
Then validation SHALL report no EARS format errors for the requirements artefact.

---

<a id="ac-38-023"></a>

### AC-38.023

**Trace:** [REQ-38.023](ep-requirements.md#verification-and-contracts)  
**Test level:** Manual  
**Status:** AC-38.023 MANUAL ONLY — verified by reviewing the EP-038 branch diff for absence of intentional changes to Telegram/session semantics, tool-selection algorithms, router policies, memory storage formats, or operator-visible configuration behaviour.

Given the EP-038 change set  
When reviewing product diffs outside test import fixes  
Then changes SHALL be structural (file moves and grouping) only  
And SHALL NOT intentionally alter product behaviour listed in REQ-38.023.

---

<a id="ac-38-024"></a>

### AC-38.024

**Trace:** [REQ-38.024](ep-requirements.md#verification-and-contracts)  
**Test level:** Manual (grep)  
**Status:** AC-38.024 MANUAL ONLY — verified by grep that `conversationHandler` is not exported and no new exported sub-handler types appear under `internal/core`.

Given EP-038 implementation is complete  
When searching `internal/core` for exported handler types introduced by this epic  
Then `conversationHandler` SHALL remain unexported  
And sub-handlers SHALL NOT be exported to other packages.

---

<a id="ac-38-025"></a>

### AC-38.025

**Trace:** [REQ-38.025](ep-requirements.md#tier-main-prompt-boundary)  
**Test level:** Manual  
**Status:** AC-38.025 MANUAL ONLY — verified by reviewing diffs to `handler_tier_main_prompt.go` for naming, comment, or visibility-only edits when cleanup is applied.

Given optional light cleanup is applied in `handler_tier_main_prompt.go`  
When reviewing that file’s diff  
Then changes SHALL be limited to naming, comments, or unexported helper visibility  
And SHALL NOT alter tier selection, prompt message shapes, or tool-list assembly order.

---
