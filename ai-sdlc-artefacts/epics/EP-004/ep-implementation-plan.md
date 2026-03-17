# EP-004 Implementation plan — Structured tools and Tool-calling API

**Purpose:** Ordered implementation tasks for EP-004 with dependencies, verification per task, checkpoints per step, and traceability to requirements and acceptance criteria.

**Pipeline:** Stage 7 — Implementation planning.

**Previous/Related:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md), [ep-requirements.md](ep-requirements.md), [ep-system-design.md](ep-system-design.md).

---

## 1. Tool catalog and config

- [x] **1.1 Add tool catalog to config and define catalog format**
  - Add tool catalog path (or list of paths) to config; validate at load.
  - Define YAML schema for catalog: per-tool fields `id`, `short_description`, `template` (placeholders), `node_id`, `arguments` (name, type, required, allowed_values, pattern, min, max), optional `triggers`.
  - Parse catalog at startup; fail fast on parse or schema error.
  - _Requirements:_ [REQ-04.001](ep-requirements.md#tool-catalog-and-source-of-truth), [REQ-04.002](ep-requirements.md#tool-catalog-and-source-of-truth), [REQ-04.003](ep-requirements.md#tool-catalog-and-source-of-truth)
  - _Acceptance Criteria:_ [AC-04.001](ep-acceptance-criteria.md#ac-04-001), [AC-04.002](ep-acceptance-criteria.md#ac-04-002)
  - **Verification:** `go build ./...`; unit test for parse/validate; invalid catalog or path causes startup failure.
  - **Checkpoint:** Before proceeding: catalog is the single source of truth; no tools defined elsewhere; startup fails on invalid catalog.

---

## 2. Tool index (same vector DB, dedicated table)

- [x] **2.1 Add dedicated tool table in vector store**
  - In the same vector DB file as memory, add a dedicated table (e.g. `vec_tools`) for tool index entries.
  - API: add entries (tool id, embedding); search by query embedding, return top-k tool ids (and optionally scores). No separate DB instance.
  - _Requirements:_ [REQ-04.018](ep-requirements.md#tool-index-and-pre-selection)
  - _Acceptance Criteria:_ [AC-04.014](ep-acceptance-criteria.md#ac-04-014)
  - **Verification:** Unit or integration test: write tool entries, search by similarity, retrieve correct ids; same DB file contains both memory table and tool table.
  - **Checkpoint:** Before proceeding: vector store has dedicated tool table; search by embedding returns top-k tool ids from same DB as memory.

- [x] **2.2 Build tool index at startup (20 s or background + fallback)**
  - For each tool in parsed catalog: build text (id + short_description + optional triggers), obtain embedding (use **batched** embedding API where supported), store in `vec_tools`.
  - Index build MUST complete within 20 seconds from startup, or run in background with a defined fallback (e.g. default tool subset or "index not ready") until ready.
  - _Requirements:_ [REQ-04.018](ep-requirements.md#tool-index-and-pre-selection), [REQ-04.021](ep-requirements.md#tool-index-and-pre-selection)
  - _Acceptance Criteria:_ [AC-04.014](ep-acceptance-criteria.md#ac-04-014), [AC-04.017](ep-acceptance-criteria.md#ac-04-017)
  - **Verification:** Unit test with mock embedder; test or benchmark that index build for N tools (e.g. 100–1000) finishes within 20 s or background path and fallback behave correctly.
  - **Checkpoint:** Before proceeding: tool index build completes within 20 s or background + fallback is defined and working; no request uses tools until index is ready without fallback.

---

## 3. Tool pre-selection

- [x] **3.1 Implement tool pre-selection and fallback**
  - Input: user message (and optionally conversation context). Embed query, search `vec_tools`, return top-k tool ids.
  - If pre-selection returns no tools or fewer than configured minimum, apply fallback (e.g. default subset or all tools up to a cap) so the request still gets a non-empty, bounded tool list.
  - _Requirements:_ [REQ-04.019](ep-requirements.md#tool-index-and-pre-selection), [REQ-04.020](ep-requirements.md#tool-index-and-pre-selection)
  - _Acceptance Criteria:_ [AC-04.015](ep-acceptance-criteria.md#ac-04-015), [AC-04.016](ep-acceptance-criteria.md#ac-04-016)
  - **Verification:** Unit tests: search returns top-k; fallback when empty or below minimum yields bounded non-empty list.
  - **Checkpoint:** Before proceeding: pre-selection returns bounded tool ids; fallback guarantees non-empty list when selection is empty or too small.

---

## 4. Provider interface and one provider

- [x] **4.1 Extend LLM provider interface for tools and tool_calls**
  - Extend provider interface to accept optional tools payload in the request and to return `tool_calls` (id, name, arguments) and related metadata in the response.
  - _Requirements:_ [REQ-04.012](ep-requirements.md#provider-interface)
  - _Acceptance Criteria:_ [AC-04.009](ep-acceptance-criteria.md#ac-04-009)
  - **Verification:** Interface compiles; mock provider implements optional tools and returns tool_calls.
  - **Checkpoint:** Before proceeding: interface is extended; all call sites compile; mock provider can be used in tests.

- [x] **4.2 Implement tools + tool_calls for at least one provider**
  - Implement sending tools in the completion request and parsing `tool_calls` in the response for at least one provider (e.g. OpenAI-compatible or Ollama).
  - _Requirements:_ [REQ-04.004](ep-requirements.md#tool-calling-api), [REQ-04.005](ep-requirements.md#tool-calling-api)
  - _Acceptance Criteria:_ [AC-04.003](ep-acceptance-criteria.md#ac-04-003)
  - **Verification:** Integration test or manual run: request includes tools; response with tool_calls is parsed correctly.
  - **Checkpoint:** Before proceeding: at least one provider supports Tool-calling API end-to-end (request tools, receive tool_calls).

---

## 5. Core: tool list, tool_calls, validation, execution

- [x] **5.1 Core: wire pre-selection and tool list per request**
  - For each completion request that can trigger tools: run tool pre-selection, build tool list from catalog for selected ids only (id, short_description, parameters schema or example), pass to LLM provider.
  - _Requirements:_ [REQ-04.004](ep-requirements.md#tool-calling-api), [REQ-04.019](ep-requirements.md#tool-index-and-pre-selection)
  - _Acceptance Criteria:_ [AC-04.003](ep-acceptance-criteria.md#ac-04-003), [AC-04.015](ep-acceptance-criteria.md#ac-04-015)
  - **Verification:** Integration test with mock provider: request contains pre-selected tools in provider format.
  - **Checkpoint:** Before proceeding: each completion request that can trigger tools receives a bounded tool list built from pre-selection.

- [x] **5.2 Core: validate tool_calls and reject invalid/unknown**
  - On LLM response with tool_calls: look up tool id in catalog; validate arguments (types, allowed_values, pattern, min/max). If tool id unknown or validation fails: do not execute any command; return deterministic error.
  - _Requirements:_ [REQ-04.007](ep-requirements.md#validation-and-execution), [REQ-04.008](ep-requirements.md#validation-and-execution)
  - _Acceptance Criteria:_ [AC-04.005](ep-acceptance-criteria.md#ac-04-005), [AC-04.006](ep-acceptance-criteria.md#ac-04-006)
  - **Verification:** Unit tests for validation rules; integration test: unknown tool or invalid args → error, no command executed.
  - **Checkpoint:** Before proceeding: unknown tool or invalid arguments never trigger execution; deterministic error returned.

- [x] **5.3 Core: template substitution and execution via run_on_node**
  - For valid tool call: substitute validated arguments into tool template; execute resulting command via existing run_on_node path (allowlist and SSH unchanged).
  - _Requirements:_ [REQ-04.009](ep-requirements.md#validation-and-execution), [REQ-04.010](ep-requirements.md#validation-and-execution)
  - _Acceptance Criteria:_ [AC-04.007](ep-acceptance-criteria.md#ac-04-007)
  - **Verification:** Unit test for substitution; integration test: valid tool call runs via run_on_node and allowlist is enforced.
  - **Checkpoint:** Before proceeding: valid tool calls produce a single command and execute via run_on_node; allowlist and SSH model unchanged.

- [x] **5.4 Core: tool-result loop and errors to chat**
  - Return tool results (or errors) to provider; continue request–response–tool-result loop (append tool result, call provider again) or return final reply. Surface validation and execution errors to user in chat.
  - _Requirements:_ [REQ-04.006](ep-requirements.md#tool-calling-api), [REQ-04.011](ep-requirements.md#errors-to-chat)
  - _Acceptance Criteria:_ [AC-04.004](ep-acceptance-criteria.md#ac-04-004), [AC-04.008](ep-acceptance-criteria.md#ac-04-008)
  - **Verification:** Integration test: tool_calls → execution/results/errors → user sees reply or error in chat.
  - **Checkpoint:** Before proceeding: tool-result loop works; validation and execution errors are visible to the user in chat.

---

## 6. Observability and Sonos

- [ ] **6.1 Tool invocation logging**
  - Log tool id, arguments, and result or error for each tool invocation where the existing logging subsystem supports it.
  - _Requirements:_ [REQ-04.016](ep-requirements.md#nfr--security-testability-observability-consistency)
  - _Acceptance Criteria:_ [AC-04.013](ep-acceptance-criteria.md#ac-04-013)
  - **Verification:** Test or manual check that logs contain tool invocation data.
  - **Checkpoint:** Before proceeding: tool invocations (id, arguments, result/error) are traceable in logs.

- [ ] **6.2 Sonos tool in catalog**
  - Ensure at least one Sonos-related tool (e.g. volume or play) can be defined in the catalog, bound to a configured node, exposed to the LLM, and executed via the same validation and run_on_node path as other tools.
  - _Requirements:_ [REQ-04.013](ep-requirements.md#sonos-support), [REQ-04.017](ep-requirements.md#nfr--security-testability-observability-consistency)
  - _Acceptance Criteria:_ [AC-04.010](ep-acceptance-criteria.md#ac-04-010), [AC-04.011](ep-acceptance-criteria.md#ac-04-011)
  - **Verification:** Catalog example with Sonos tool; integration or E2E test that invokes it and verifies same path as other tools.
  - **Checkpoint:** Before proceeding: at least one Sonos tool is definable and executable; same catalog and execution path as other tools.

---

## 7. Tests and closure

- [ ] **7.1 Tests and regression**
  - Cover new and changed behaviour with unit and integration tests; ensure all existing tests pass.
  - _Requirements:_ [REQ-04.014](ep-requirements.md#nfr--security-testability-observability-consistency), [REQ-04.015](ep-requirements.md#nfr--security-testability-observability-consistency)
  - _Acceptance Criteria:_ [AC-04.011](ep-acceptance-criteria.md#ac-04-011), [AC-04.012](ep-acceptance-criteria.md#ac-04-012)
  - **Verification:** `make check` (or project equivalent) passes.
  - **Checkpoint:** Epic complete: all tests pass; no command is executed without known tool and schema-valid arguments; ask user if questions arise.

---

## Dependencies (summary)

- 1.1 → 2.1, 2.2, 3.1, 5.x
- 2.1 → 2.2
- 2.2 → 3.1
- 3.1 → 5.1
- 4.1 → 4.2, 5.1
- 4.2 → 5.1, 5.4
- 5.1 → 5.2 → 5.3 → 5.4
- 6.1, 6.2 can be done in parallel with 5.x once catalog and execution path exist
- 7.1 after all feature tasks
