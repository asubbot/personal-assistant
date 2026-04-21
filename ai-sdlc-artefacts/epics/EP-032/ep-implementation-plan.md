# EP-032 — Implementation plan

Pipeline stage 8 output for EP-032.  
Purpose: deliver specialized vector knowledge tools and unified config block `tools.vector_search_tools`.

**Related artefacts**

- Scope: [ep-scope.md](ep-scope.md)
- Requirements: [ep-requirements.md](ep-requirements.md)
- Acceptance criteria: [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- System design: [ep-system-design.md](ep-system-design.md)
- Design review: [ep-system-design-review.md](ep-system-design-review.md)

---

## Task list

- [x] 1. Add unified config block for all vector-search tools
  - Extend `tools` config schema with `vector_search_tools` and per-tool limits/switches.
  - Add fail-fast validation for invalid numeric bounds and inconsistent values.
  - Ensure `search_vector_memory` runtime settings are sourced from this block.
  - _Requirements:_ [REQ-32.004](ep-requirements.md#unified-config-block), [REQ-32.005](ep-requirements.md#unified-config-block), [REQ-32.006](ep-requirements.md#unified-config-block)
  - _Acceptance Criteria:_ [AC-32.004](ep-acceptance-criteria.md#ac-32-004), [AC-32.005](ep-acceptance-criteria.md#ac-32-005), [AC-32.006](ep-acceptance-criteria.md#ac-32-006)
  - **Verification:** config unit tests pass for valid and invalid block values.

- [x] 2. Implement `search_vector_tool` and `search_vector_skill` native tools
  - Add new read-only specialized tools with `query` and optional `top_k`.
  - Apply deterministic ordering and bounded output with source identifiers.
  - Keep strict validation behavior and no write side effects.
  - _Requirements:_ [REQ-32.001](ep-requirements.md#tool-contract), [REQ-32.002](ep-requirements.md#tool-contract), [REQ-32.009](ep-requirements.md#retrieval-behavior), [REQ-32.010](ep-requirements.md#retrieval-behavior), [REQ-32.011](ep-requirements.md#retrieval-behavior), [REQ-32.012](ep-requirements.md#safety-and-observability)
  - _Acceptance Criteria:_ [AC-32.001](ep-acceptance-criteria.md#ac-32-001), [AC-32.002](ep-acceptance-criteria.md#ac-32-002), [AC-32.009](ep-acceptance-criteria.md#ac-32-009), [AC-32.010](ep-acceptance-criteria.md#ac-32-010), [AC-32.011](ep-acceptance-criteria.md#ac-32-011), [AC-32.012](ep-acceptance-criteria.md#ac-32-012)
  - **Verification:** specialized tool unit tests pass for validation, ordering, and bounded output.

- [x] 3. Wire runtime registration and allowlist integration
  - Register specialized tools in startup wiring when dependencies are available.
  - Keep `search_vector_memory` domain unchanged and preserve existing memory lanes.
  - Update runtime skill allowed native IDs to include both new tool ids.
  - _Requirements:_ [REQ-32.003](ep-requirements.md#tool-contract), [REQ-32.007](ep-requirements.md#unified-config-block), [REQ-32.008](ep-requirements.md#unified-config-block), [REQ-32.014](ep-requirements.md#runtime-skills-integration)
  - _Acceptance Criteria:_ [AC-32.003](ep-acceptance-criteria.md#ac-32-003), [AC-32.007](ep-acceptance-criteria.md#ac-32-007), [AC-32.008](ep-acceptance-criteria.md#ac-32-008), [AC-32.014](ep-acceptance-criteria.md#ac-32-014)
  - **Verification:** startup/registry tests and runtime-skill validation tests pass.

- [x] 4. Add integration and E2E-oriented conversation tests with logging checks
  - Cover tool-calling loop for `search_vector_tool` and `search_vector_skill`.
  - Verify invocation logs include metadata and honor redaction policy.
  - Ensure AC trace comments are added in tests.
  - _Requirements:_ [REQ-32.013](ep-requirements.md#safety-and-observability), [REQ-32.017](ep-requirements.md#verification)
  - _Acceptance Criteria:_ [AC-32.013](ep-acceptance-criteria.md#ac-32-013), [AC-32.017](ep-acceptance-criteria.md#ac-32-017)
  - **Verification:** integration tests pass and validate grounded same-turn final answers.

- [x] 5. Update configuration/docs examples and run quality gates
  - Update `.config/config.json` and docs/examples for unified config block and new tools.
  - Run `make check` and `./bin/validate EP-032`.
  - _Requirements:_ [REQ-32.015](ep-requirements.md#verification), [REQ-32.016](ep-requirements.md#verification)
  - _Acceptance Criteria:_ [AC-32.015](ep-acceptance-criteria.md#ac-32-015), [AC-32.016](ep-acceptance-criteria.md#ac-32-016)
  - **Verification:** both commands exit `0`.

---

## Dependencies and order

- Task 2 depends on Task 1.
- Task 3 depends on Tasks 1 and 2.
- Task 4 depends on Task 3.
- Task 5 depends on Tasks 1–4.

---

## Checkpoints

- After Task 1: validate config load errors are deterministic and field-specific.
- After Task 3: verify no regression in `search_vector_memory` lane behavior.
- Final checkpoint: run full quality gates (`make check`, `./bin/validate EP-032`) before stage 10.
