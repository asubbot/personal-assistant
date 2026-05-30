---
artefact: ep-implementation-plan
epic_id: EP-038
status: draft
source_of_truth: true
updated_at: 2026-05-31
---

# EP-038 — Implementation plan

Pipeline stage 8 output for **EP-038 Refactor core conversation handler (god handler)**.  
Purpose: cut/paste `internal/core/handler.go` (~663 LOC) into `handler_memory.go`, `handler_tools.go`, and `handler_llm.go` while keeping slim orchestration in `handler.go` — **no logic edits**, no config schema change, behaviour parity via existing suites.

**Related artefacts**

- Scope: [ep-scope.md](ep-scope.md)
- Requirements: [ep-requirements.md](ep-requirements.md)
- Acceptance criteria: [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- System design: [ep-system-design.md](ep-system-design.md)
- Design review: [ep-system-design-review.md](ep-system-design-review.md) (gate **pass**, iteration 2)
- Test strategy: [strategy.md](../../strategy.md)

**Branch:** `epic/EP-038-refactor-core-handler` (implementation only; no git commit from this stage).

**Out of scope (do not implement):** product behaviour changes ([REQ-38.023](ep-requirements.md#req-38-023--no-product-behaviour-changes)); `config.json` schema edits ([REQ-38.017](ep-requirements.md#req-38-017--no-configjson-schema-change)); tier-strategy framework or new tiers ([REQ-38.012](ep-requirements.md#req-38-012--no-new-tier-values-or-full_lite-revival), [REQ-38.013](ep-requirements.md#req-38-013--use-simple-tier-switch-no-strategy-framework)); relocating `runtime_tools.go`, `dynamic_tool_selection.go`, `system_tail.go`, `vector_merge.go`, or `memory_vectors.go` responsibilities ([REQ-38.008](ep-requirements.md#req-38-008--leave-runtime_tools-and-dynamic_tool_selection-in-place), [REQ-38.010](ep-requirements.md#req-38-010--leave-vector_merge-memory_vectors-system_tail-ownership)).

**AC trace convention:** EP-038 adds **no new product tests** unless a move breaks compile-only helpers ([ep-system-design.md](ep-system-design.md#testing-strategy)). Existing suites cover behaviour parity. If a new test is required (e.g. import/location fix for an unexported helper), mark it `// Covers AC-38.xxx`. Structural and process gates marked **MANUAL ONLY** in [ep-acceptance-criteria.md](ep-acceptance-criteria.md) are satisfied by grep, LOC, diff review, `make check`, and validate — no unit test required for those ACs.

**Move rule:** Same `package core`; cut/paste only; trim per-file imports after each extraction ([ep-system-design-review.md](ep-system-design-review.md) S-001). Package-level helpers (`genRequestID`, `parseToolArgumentsJSON`, etc.) move by symbol name — MANUAL grep checks definition **in target file**, not receiver syntax ([ep-system-design.md#manual-grep-guidance](ep-system-design.md#manual-grep-guidance)).

---

## Tasks

### Phase 0 — Prerequisite gate

- [ ] **0.1** Confirm EP-035, EP-036, and EP-037 are merged on the integration branch before landing EP-038 (merge-base / branch history).
  - _Requirements:_ [REQ-38.001](ep-requirements.md#req-38-001--land-after-ep-035-ep-036-ep-037-merged)
  - _Acceptance Criteria:_ [AC-38.001](ep-acceptance-criteria.md#ac-38-001) (**MANUAL ONLY**)
  - **Verification:** `git log --oneline --grep='EP-035\|EP-036\|EP-037'` or merge-base inspection shows prerequisite epics merged; branch `epic/EP-038-refactor-core-handler` builds on their commits.

---

### Phase 1 — Extract `handler_memory.go`

- [ ] **1.1** Create `internal/core/handler_memory.go`. Move from `handler.go` (no logic edits):
  - **Receiver methods:** `gatherRetrievedChunkTexts`, `gatherSplitTableChunks`, `labeledChunksFromResults`, `handleLLMSuccess`, `indexTurn`
  - **Package-level helpers:** `retrievalChunkWithLabel`, `eventAlignedTurnDate`, `canonicalizeTurnPair`, `canonicalizeTurnText`
  - Trim imports to memory/vector/crypto paths only; remove moved symbols from `handler.go`.
  - _Requirements:_ [REQ-38.009](ep-requirements.md#req-38-009--extract-rag-chunk-and-turn-index-methods)
  - _Acceptance Criteria:_ [AC-38.009](ep-acceptance-criteria.md#ac-38-009) (**MANUAL ONLY** — grep after Phase 7)
  - **Verification:** `go test ./internal/core/... -count=1` passes; `rg '^func (gatherRetrievedChunkTexts|indexTurn|canonicalizeTurnPair)\b' internal/core/handler_memory.go` matches; same symbols absent from `handler.go`.

- [ ] **1.2** Confirm `vector_merge.go`, `memory_vectors.go`, and `system_tail.go` diffs are import-only (no responsibility relocation).
  - _Requirements:_ [REQ-38.010](ep-requirements.md#req-38-010--leave-vector_merge-memory_vectors-system_tail-ownership)
  - _Acceptance Criteria:_ [AC-38.010](ep-acceptance-criteria.md#ac-38-010) (**MANUAL ONLY** — diff review)
  - **Verification:** `git diff internal/core/vector_merge.go internal/core/memory_vectors.go internal/core/system_tail.go` shows no logic changes (empty or import-only).

**Checkpoint (Phase 1):** `go test ./internal/core/... -count=1` green; memory symbols compile in `handler_memory.go`.

---

### Phase 2 — Extract `handler_tools.go`

- [ ] **2.1** Create `internal/core/handler_tools.go`. Move from `handler.go` (no logic edits):
  - **Receiver methods:** `mergeSelectedToolIDs`, `selectSkillPackages`, `completionOptionsMergedCatalogNative`, `nativeToolDefs`, `executeOneToolCall`, `executeCatalogToolCall`
  - **Package-level helpers:** `parseToolArgumentsJSON`, `remoteCommandFromRunOnNodeArgs`
  - Trim imports; remove moved symbols from `handler.go`.
  - _Requirements:_ [REQ-38.007](ep-requirements.md#req-38-007--extract-tool-merge-selection-and-execution)
  - _Acceptance Criteria:_ [AC-38.007](ep-acceptance-criteria.md#ac-38-007) (**MANUAL ONLY** — grep after Phase 7)
  - **Verification:** `go test ./internal/core/... -count=1` passes; `rg '^func (mergeSelectedToolIDs|executeOneToolCall|parseToolArgumentsJSON)\b' internal/core/handler_tools.go` matches.

- [ ] **2.2** Verify unchanged tool-module ownership (no moves into `handler_tools.go`).
  - _Requirements:_ [REQ-38.008](ep-requirements.md#req-38-008--leave-runtime_tools-and-dynamic_tool_selection-in-place)
  - _Acceptance Criteria:_ [AC-38.008](ep-acceptance-criteria.md#ac-38-008) (**MANUAL ONLY** — grep)
  - **Verification:** `rg '^func mergeToolIDs\b' internal/core/runtime_tools.go`; `rg '^func pickToolsForMainRequest\b' internal/core/dynamic_tool_selection.go`; `rg 'mergedAfterDynamicToolCap|mergeTailMergedToolsAndOptions' internal/core/handler_tier_main_prompt.go` — all present; no duplicate definitions in `handler_tools.go`.

**Checkpoint (Phase 2):** `go test ./internal/core/... -count=1` green; tier file still calls moved receivers in same package.

---

### Phase 3 — Extract `handler_llm.go`

- [ ] **3.1** Create `internal/core/handler_llm.go`. Move from `handler.go` (no logic edits):
  - **Receiver methods:** `completeViaRouter`, `completeAt`, `onRouteEvent`, `finishAfterFirstLLM`, `runToolResultLoop`, `appendToolRound`, `systemStaticHead`, `logLLMRequest`, `logMainLLMCompletion`, `logLLMResponse`, `logMainLLMPromptAssembled`
  - **Package-level / handler-param helpers:** `genRequestID`, `truncateToolResultForPrompt`, `todayCalendarDateInPALocation`, `paLocationFromConfig`
  - `paLocationFromConfig` remains callable from `run.go` (`BuildMessageHandler`) in same package.
  - Trim imports; remove moved symbols from `handler.go`.
  - _Requirements:_ [REQ-38.005](ep-requirements.md#req-38-005--extract-llm-completion-and-tool-loop-methods)
  - _Acceptance Criteria:_ [AC-38.005](ep-acceptance-criteria.md#ac-38-005) (**MANUAL ONLY** — grep after Phase 7)
  - **Verification:** `go test ./internal/core/... -count=1` passes; `rg '^func (genRequestID|completeAt|runToolResultLoop|paLocationFromConfig)\b' internal/core/handler_llm.go` matches; `rg '^func (genRequestID|completeAt)\b' internal/core/handler.go` → no matches.

- [ ] **3.2** Run EP-034 LLM routing regression slice (behaviour unchanged after move).
  - _Requirements:_ [REQ-38.006](ep-requirements.md#req-38-006--preserve-router-usage-round-cap-message-roles), [REQ-38.019](ep-requirements.md#req-38-019--preserve-ep-013034036037-contracts)
  - _Acceptance Criteria:_ [AC-38.006](ep-acceptance-criteria.md#ac-38-006), [AC-38.019](ep-acceptance-criteria.md#ac-38-019)
  - **Verification:** `go test ./internal/core/... -run 'EP034|ToolResult|Route' -count=1` passes (existing tests; no new tests required).

**Checkpoint (Phase 3):** `go test ./internal/core/... -count=1` green; `handler_tier_main_prompt.go` still invokes `systemStaticHead` / completion path via same-package receivers.

---

### Phase 4 — Slim `handler.go` (orchestration only)

- [ ] **4.1** Leave in `handler.go` only: `conversationHandler` struct, turn constants (`maxToolRounds`, `logTruncateMaxLen`, `maxToolResultPromptBytes`, `defaultMaxDynamicSystemRunes`), `redactLogString`, `checkUserMessage`, `sessionMemoryEnabled`, `appendSessionIfEnabled`, `HandleMessage`, and usage-footer wiring. Remove any leftover moved implementations or unused imports.
  - _Requirements:_ [REQ-38.002](ep-requirements.md#req-38-002--keep-struct-and-handlemessage-turn-sequence), [REQ-38.003](ep-requirements.md#req-38-003--reduce-handlergo-to-orchestration-200-loc), [REQ-38.004](ep-requirements.md#req-38-004--retain-shared-turn-constants-in-handlergo)
  - _Acceptance Criteria:_ [AC-38.002](ep-acceptance-criteria.md#ac-38-002), [AC-38.003](ep-acceptance-criteria.md#ac-38-003) (**MANUAL ONLY** for LOC), [AC-38.004](ep-acceptance-criteria.md#ac-38-004)
  - **Verification:** `wc -l internal/core/handler.go` ≤ ~200; `rg '^func (HandleMessage|checkUserMessage)\b' internal/core/handler.go`; `rg 'maxToolRounds\s*=' internal/core/handler.go`; `go test ./internal/core/... -run 'HandleMessage|TestHandleMessage' -count=1` passes.

- [ ] **4.2** Confirm public wiring unchanged (`adapter.go`, `run.go`, `integration_export.go`).
  - _Requirements:_ [REQ-38.014](ep-requirements.md#req-38-014--preserve-messagehandlerhandlemessage-signature), [REQ-38.015](ep-requirements.md#req-38-015--preserve-buildmessagehandler-and-run-surfaces), [REQ-38.016](ep-requirements.md#req-38-016--preserve-newintegrationconversationhandler), [REQ-38.024](ep-requirements.md#req-38-024--do-not-rename-or-export-conversationhandler)
  - _Acceptance Criteria:_ [AC-38.014](ep-acceptance-criteria.md#ac-38-014), [AC-38.015](ep-acceptance-criteria.md#ac-38-015), [AC-38.016](ep-acceptance-criteria.md#ac-38-016), [AC-38.024](ep-acceptance-criteria.md#ac-38-024) (**MANUAL ONLY** for grep on unexported type)
  - **Verification:** `go test ./internal/core/... ./tests/integration/... -run 'Run|Integration|RuntimeSkills' -count=1` passes; `rg '^type conversationHandler\b' internal/core/handler.go`; `rg '^func New.*Handler' internal/core/` — no exported `conversationHandler`.

**Checkpoint (Phase 4):** `make check` — first full-repo gate after structural work complete.

---

### Phase 5 — Tier boundary guard (no moves)

- [ ] **5.1** Verify `handler_tier_main_prompt.go` retains tier dispatch; optional naming/comment-only cleanup per [REQ-38.025](ep-requirements.md#req-38-025--optional-namingcomments-cleanup-only-in-tier-prompt-file).
  - _Requirements:_ [REQ-38.011](ep-requirements.md#req-38-011--retain-tier-main-prompt-dispatch-in-handler_tier_main_promptgo), [REQ-38.012](ep-requirements.md#req-38-012--no-new-tier-values-or-full_lite-revival), [REQ-38.013](ep-requirements.md#req-38-013--use-simple-tier-switch-no-strategy-framework), [REQ-38.025](ep-requirements.md#req-38-025--optional-namingcomments-cleanup-only-in-tier-prompt-file)
  - _Acceptance Criteria:_ [AC-38.011](ep-acceptance-criteria.md#ac-38-011), [AC-38.012](ep-acceptance-criteria.md#ac-38-012), [AC-38.013](ep-acceptance-criteria.md#ac-38-013) (**MANUAL ONLY** for switch grep), [AC-38.025](ep-acceptance-criteria.md#ac-38-025) (**MANUAL ONLY** if cleanup applied)
  - **Verification:** `go test ./internal/core/... -run 'TierMainPrompt|EP017|EP018' -count=1` passes; `rg 'TierSimple|TierFull' internal/core/handler_tier_main_prompt.go`; `rg 'TierFullLite|TierStrategy' internal/core/` → zero in production files.

---

### Phase 6 — Test and config parity (existing suites only)

- [ ] **6.1** Fix test/import references only if file moves break compile (no assertion changes per [REQ-38.020](ep-requirements.md#req-38-020--existing-handler-tests-pass-without-assertion-changes)). Do **not** add behavioural tests unless required for compile-only helpers — if added, use `// Covers AC-38.xxx`.
  - _Requirements:_ [REQ-38.020](ep-requirements.md#req-38-020--existing-handler-tests-pass-without-assertion-changes)
  - _Acceptance Criteria:_ [AC-38.020](ep-acceptance-criteria.md#ac-38-020)
  - **Verification:** `go test ./internal/core/... -count=1` passes; `git diff internal/core/*_test.go` shows import/location fixes only (no assertion edits).

- [ ] **6.2** Run behaviour-parity and contract regression suites (unchanged assertions).
  - _Requirements:_ [REQ-38.017](ep-requirements.md#req-38-017--no-configjson-schema-change), [REQ-38.018](ep-requirements.md#req-38-018--behaviour-parity-on-tier-tools-prompts-routing), [REQ-38.019](ep-requirements.md#req-38-019--preserve-ep-013034036037-contracts)
  - _Acceptance Criteria:_ [AC-38.017](ep-acceptance-criteria.md#ac-38-017), [AC-38.018](ep-acceptance-criteria.md#ac-38-018), [AC-38.019](ep-acceptance-criteria.md#ac-38-019)
  - **Verification:** `go test ./internal/core/... -count=1`; `go test ./internal/config/... -count=1`; key files: `handler_ep034_regression_test.go`, `handler_ep036_test.go`, `tools_selection_parity_test.go`, `handler_ep017_test.go`, `handler_ep018_test.go`, `handler_ep018_coverage_test.go`, `tests/integration/runtime_skills_handler_test.go`.

**Checkpoint (Phase 6):** `make check` green before final gates.

---

### Phase 7 — Manual structural verification and quality gates

- [ ] **7.1** Run manual grep / LOC checklist from [ep-system-design.md#manual-grep-guidance](ep-system-design.md#manual-grep-guidance).
  - _Acceptance Criteria:_ [AC-38.003](ep-acceptance-criteria.md#ac-38-003), [AC-38.005](ep-acceptance-criteria.md#ac-38-005), [AC-38.007](ep-acceptance-criteria.md#ac-38-007), [AC-38.008](ep-acceptance-criteria.md#ac-38-008), [AC-38.009](ep-acceptance-criteria.md#ac-38-009), [AC-38.013](ep-acceptance-criteria.md#ac-38-013), [AC-38.024](ep-acceptance-criteria.md#ac-38-024) (**MANUAL ONLY**)
  - **Verification:** Commands in design Manual grep guidance table all succeed; turn-index format note: first 12 **bytes** of SHA-256 digest (`sum[:12]` with `%x` → 24 hex digits) per code comment in `indexTurn`.

- [ ] **7.2** Review branch diff for structural-only changes (no intentional behaviour edits).
  - _Requirements:_ [REQ-38.023](ep-requirements.md#req-38-023--no-product-behaviour-changes)
  - _Acceptance Criteria:_ [AC-38.023](ep-acceptance-criteria.md#ac-38-023) (**MANUAL ONLY**)
  - **Verification:** `git diff main...HEAD -- internal/core/` — moves/grouping only; no router, tool-selection, or config-semantics edits outside import wiring.

- [ ] **7.3** Final quality gates.
  - _Requirements:_ [REQ-38.021](ep-requirements.md#req-38-021--make-check-passes), [REQ-38.022](ep-requirements.md#req-38-022--validate-ears-ep-038-passes)
  - _Acceptance Criteria:_ [AC-38.021](ep-acceptance-criteria.md#ac-38-021) (**MANUAL ONLY**), [AC-38.022](ep-acceptance-criteria.md#ac-38-022) (**MANUAL ONLY**)
  - **Verification:** `make check` exit 0; `make build` then `./bin/validate ears EP-038` (25 reqs, 0 errors) and `./bin/validate req EP-038` (25/25) exit 0.

---

## Dependencies and order

| Task | Depends on |
|------|------------|
| 1.1 | 0.1 (informational; branch already has prerequisites) |
| 1.2 | 1.1 |
| 2.1 | 1.1 |
| 2.2 | 2.1 |
| 3.1 | 2.1 |
| 3.2 | 3.1 |
| 4.1 | 3.1 |
| 4.2 | 4.1 |
| 5.1 | 4.1 |
| 6.1 | 4.1 |
| 6.2 | 6.1, 5.1 |
| 7.1 | 4.1, 5.1 |
| 7.2 | 6.2 |
| 7.3 | 6.2, 7.1, 7.2 |

**Recommended path:** 0.1 → 1.1 → 1.2 → 2.1 → 2.2 → 3.1 → 3.2 → 4.1 → 4.2 → **`make check`** → 5.1 → 6.1 → 6.2 → **`make check`** → 7.1 → 7.2 → 7.3.

---

## Checkpoints

- **After Phase 1:** `go test ./internal/core/... -count=1` — memory extraction compiles; tier file can still call `gatherRetrievedChunkTexts`.
- **After Phase 2:** `go test ./internal/core/... -count=1` — tool extraction compiles; `handler_tier_main_prompt.go` unchanged in responsibility.
- **After Phase 3:** `go test ./internal/core/... -count=1` — LLM/tool-loop path compiles; EP-034 regression slice green.
- **After Phase 4:** **`make check`** — first full-repo gate; `handler.go` ≤ ~200 LOC.
- **After Phase 6:** **`make check`** — all handler and integration tests green before manual checklist.
- **Before stage 10:** Phase 7 manual grep + diff review + `make check` + `./bin/validate ears EP-038` + `./bin/validate req EP-038`.

---

## AC coverage map (automated vs manual)

| AC | Primary task(s) | Notes |
|----|-----------------|-------|
| AC-38.001 | 0.1 | **MANUAL ONLY** — prerequisite merge |
| AC-38.002 | 4.1 | Integration — existing `handler_*` tests |
| AC-38.003 | 4.1, 7.1 | **MANUAL ONLY** — `wc -l handler.go` |
| AC-38.004 | 4.1 | Unit — const in `handler.go`; existing tool-loop tests |
| AC-38.005 | 3.1, 7.1 | **MANUAL ONLY** — grep symbol ownership |
| AC-38.006 | 3.2 | Integration — `handler_ep034_regression_test.go` |
| AC-38.007 | 2.1, 7.1 | **MANUAL ONLY** — grep |
| AC-38.008 | 2.2, 7.1 | **MANUAL ONLY** — grep unchanged modules |
| AC-38.009 | 1.1, 7.1 | **MANUAL ONLY** — grep |
| AC-38.010 | 1.2 | **MANUAL ONLY** — diff review |
| AC-38.011 | 5.1 | Integration — tier prompt tests |
| AC-38.012 | 5.1 | Unit — `handler_ep036_test.go` |
| AC-38.013 | 5.1, 7.1 | **MANUAL ONLY** — switch grep |
| AC-38.014 | 4.2 | Integration — compile + core tests |
| AC-38.015 | 4.2 | Unit — `run_test.go` |
| AC-38.016 | 4.2, 6.2 | Integration — `runtime_skills_handler_test.go` |
| AC-38.017 | 6.2 | Unit — config load tests |
| AC-38.018 | 6.2 | Integration — parity / EP-017/018 suites |
| AC-38.019 | 3.2, 6.2 | Integration — EP-034/036/037 regressions |
| AC-38.020 | 6.1 | Integration — full `internal/core` tests |
| AC-38.021 | 4.2, 6.2, 7.3 | **MANUAL ONLY** — `make check` |
| AC-38.022 | 7.3 | **MANUAL ONLY** — `./bin/validate ears EP-038` |
| AC-38.023 | 7.2 | **MANUAL ONLY** — diff review |
| AC-38.024 | 4.2, 7.1 | **MANUAL ONLY** — grep unexported type |
| AC-38.025 | 5.1 | **MANUAL ONLY** — optional tier-file diff |

**New tests:** None required by design. If task 6.1 adds a compile-only helper test, annotate `// Covers AC-38.020` (or the specific AC exercised).

---

## Files touched (reference)

**New:** `internal/core/handler_memory.go`, `handler_tools.go`, `handler_llm.go`.

**Modified:** `internal/core/handler.go` (slim orchestration); possibly import-only touches in `run.go`, `handler_tier_main_prompt.go`, `integration_export.go`; test files only for import/location fixes.

**Do not edit (responsibility):** `runtime_tools.go`, `dynamic_tool_selection.go`, `system_tail.go`, `vector_merge.go`, `memory_vectors.go`; `internal/config/*`; `.config/config.json`; `docs/*`.
