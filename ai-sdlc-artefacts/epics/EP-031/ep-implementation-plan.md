# EP-031 — Implementation plan

Pipeline stage 8 output for EP-031.  
Purpose: implement native tool `search_vector_memory` and complete verification gates.

**Related artefacts**

- Scope: [ep-scope.md](ep-scope.md)
- Requirements: [ep-requirements.md](ep-requirements.md)
- Acceptance criteria: [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- System design: [ep-system-design.md](ep-system-design.md)
- Design review: [ep-system-design-review.md](ep-system-design-review.md)

---

## Task list

- [x] 1. Implement native tool `search_vector_memory` in `internal/tools`
  - Add tool type, parameter schema, validation, lane selection, embedding, and read-only vector search.
  - Implement deterministic ordering and output shaping with bounded output policy from design.
  - Return deterministic errors for invalid input and runtime failures.
  - _Requirements:_ [REQ-31.001](ep-requirements.md#tool-contract), [REQ-31.002](ep-requirements.md#tool-contract), [REQ-31.003](ep-requirements.md#retrieval-lanes), [REQ-31.004](ep-requirements.md#retrieval-lanes), [REQ-31.005](ep-requirements.md#limits-and-output-shaping), [REQ-31.006](ep-requirements.md#limits-and-output-shaping), [REQ-31.007](ep-requirements.md#safety-and-integration), [REQ-31.010](ep-requirements.md#retrieval-policy)
  - _Acceptance Criteria:_ [AC-31.001](ep-acceptance-criteria.md#ac-31-001), [AC-31.002](ep-acceptance-criteria.md#ac-31-002), [AC-31.003](ep-acceptance-criteria.md#ac-31-003), [AC-31.004](ep-acceptance-criteria.md#ac-31-004), [AC-31.005](ep-acceptance-criteria.md#ac-31-005), [AC-31.006](ep-acceptance-criteria.md#ac-31-006), [AC-31.007](ep-acceptance-criteria.md#ac-31-007), [AC-31.010](ep-acceptance-criteria.md#ac-31-010)
  - **Verification:** Unit tests for validation, lane routing, deterministic output ordering, and read-only behavior pass.

- [x] 2. Wire tool into runtime and config allowlist paths
  - Register tool in startup/native registry wiring with dependency checks.
  - Update allowed native tool IDs for runtime skills and related validation tests.
  - Ensure invocation logging follows redaction contract.
  - _Requirements:_ [REQ-31.001](ep-requirements.md#tool-contract), [REQ-31.008](ep-requirements.md#safety-and-integration), [REQ-31.009](ep-requirements.md#safety-and-integration)
  - _Acceptance Criteria:_ [AC-31.001](ep-acceptance-criteria.md#ac-31-001), [AC-31.008](ep-acceptance-criteria.md#ac-31-008), [AC-31.009](ep-acceptance-criteria.md#ac-31-009)
  - **Verification:** Startup/registry tests prove tool availability and runtime-skill validation accepts `search_vector_memory`.

- [x] 3. Add integration and E2E-oriented tests for conversational flow
  - Cover tool execution via conversation tool loop.
  - Cover retrieval while `conversation_context.memory_vector` lane top_k values are zero.
  - Add/adjust trace comments `Covers AC-31.NNN` for validation tooling.
  - _Requirements:_ [REQ-31.010](ep-requirements.md#retrieval-policy), [REQ-31.013](ep-requirements.md#verification)
  - _Acceptance Criteria:_ [AC-31.010](ep-acceptance-criteria.md#ac-31-010), [AC-31.013](ep-acceptance-criteria.md#ac-31-013)
  - **Verification:** Integration/E2E tests pass and demonstrate on-demand retrieval without auto-RAG lanes.

- [x] 4. Update documentation and skill examples
  - Document tool usage and bounded-output semantics in operator/dev docs where relevant.
  - Update sample runtime skill guidance to use `search_vector_memory` for semantic retrieval use-cases.
  - _Requirements:_ [REQ-31.008](ep-requirements.md#safety-and-integration), [REQ-31.009](ep-requirements.md#safety-and-integration)
  - _Acceptance Criteria:_ [AC-31.008](ep-acceptance-criteria.md#ac-31-008), [AC-31.009](ep-acceptance-criteria.md#ac-31-009)
  - **Verification:** Docs/examples reference correct tool id and align with implemented behavior.

- [x] 5. Run quality gates and AC coverage validation
  - Execute full checks after code and test updates.
  - Confirm no lint/test regressions and AC coverage mapping stays valid.
  - _Requirements:_ [REQ-31.011](ep-requirements.md#verification), [REQ-31.012](ep-requirements.md#verification)
  - _Acceptance Criteria:_ [AC-31.011](ep-acceptance-criteria.md#ac-31-011), [AC-31.012](ep-acceptance-criteria.md#ac-31-012)
  - **Verification:** `make check` exits 0 and `make validate` exits 0.

---

## Dependencies and order

- Task 2 depends on Task 1.
- Task 3 depends on Tasks 1 and 2.
- Task 4 can start after Task 2 and run in parallel with Task 3.
- Task 5 depends on Tasks 1–4.

---

## Checkpoints

- After Task 2: confirm tool id and runtime-skill allowlist are stable before expanding integration tests.
- After Task 3: confirm AC trace comments cover all implemented AC ids before final gate run.
- Final checkpoint: run full quality gates (`make check`, `make validate`) and present outputs before stage 10 review.
