---
artefact: ep-implementation-plan
epic_id: EP-034
status: draft
source_of_truth: true
updated_at: 2026-05-29
---

# EP-034 — Implementation plan

Pipeline stage 8 output for EP-034.  
Purpose: remove EP-006 tool-path LLM escalation; keep transport fallback; simplify config, core, and docs.

**Related artefacts**

- Scope: [ep-scope.md](ep-scope.md)
- Requirements: [ep-requirements.md](ep-requirements.md)
- Acceptance criteria: [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- System design: [ep-system-design.md](ep-system-design.md)
- Design review: [ep-system-design-review.md](ep-system-design-review.md)

---

## Task list

- [x] 1. Remove config schema and examples for `tools.llm_escalation`
  - Delete `LLMEscalationConfig`, `ToolsLLMEscalation()`, `validateLLMEscalation` from `internal/config`.
  - Add `tools.llm_escalation` to `rejectRemovedUnsupportedConfigKeys` (same pattern as removed tool keys).
  - Remove escalation from `config.examples/` and config testdata; add rejection fixture.
  - Update `cmd/pa` wiring that passes escalation into router.
  - _Requirements:_ [REQ-34.007](ep-requirements.md#req-34-007--reject-toolsllm_escalation-config), [REQ-34.008](ep-requirements.md#req-34-008--update-example-configs)
  - _Acceptance Criteria:_ [AC-34.007](ep-acceptance-criteria.md#ac-34-007), [AC-34.008](ep-acceptance-criteria.md#ac-34-008)
  - **Verification:** `go test ./internal/config/...` passes; load of fixture with `llm_escalation` fails.

- [x] 2. Simplify `llmrouter` to transport fallback only
  - Remove `OnQualifyingFailure`, `DecideToolFailure`, `ActionEscalatePolicy`, `PhaseToolFailure`, `State.EscUsed`, escalation from `Config`.
  - `Complete` starts at index 0; local attempt loop for transport switch only.
  - Simplify `NewProviderAdapter` / `SummarizeRouterConfig` (no baseline_index).
  - Update `llmrouter` tests; remove escalation cases.
  - _Requirements:_ [REQ-34.004](ep-requirements.md#req-34-004--keep-transport-fallback), [REQ-34.005](ep-requirements.md#req-34-005--remove-router-tool-escalation-api), [REQ-34.006](ep-requirements.md#req-34-006--start-at-provider-index-0)
  - _Acceptance Criteria:_ [AC-34.004](ep-acceptance-criteria.md#ac-34-004), [AC-34.005](ep-acceptance-criteria.md#ac-34-005), [AC-34.006](ep-acceptance-criteria.md#ac-34-006)
  - **Verification:** `go test ./internal/llmrouter/...` passes.

- [x] 3. Delete escalation packages and simplify core handler
  - Delete `internal/escalationpolicy/` and `internal/core/toolfailure/`.
  - Remove `llmTurnState`, `maybeEscalate`, `escalationEnabled`, escalation field from `conversationHandler`.
  - Simplify `completeViaRouter`, `runToolResultLoop`, `executeCatalogToolCall`, noderunner error returns (plain errors).
  - Remove tool-escalation log events.
  - _Requirements:_ [REQ-34.001](ep-requirements.md#req-34-001--no-provider-change-on-tool-failure), [REQ-34.002](ep-requirements.md#req-34-002--remove-escalationpolicy-package), [REQ-34.003](ep-requirements.md#req-34-003--remove-toolfailure-package), [REQ-34.009](ep-requirements.md#req-34-009--plain-tool-errors), [REQ-34.010](ep-requirements.md#req-34-010--remove-tool-escalation-logs)
  - _Acceptance Criteria:_ [AC-34.001](ep-acceptance-criteria.md#ac-34-001), [AC-34.002](ep-acceptance-criteria.md#ac-34-002), [AC-34.003](ep-acceptance-criteria.md#ac-34-003), [AC-34.009](ep-acceptance-criteria.md#ac-34-009), [AC-34.010](ep-acceptance-criteria.md#ac-34-010)
  - **Verification:** `go test ./internal/core/... ./internal/noderunner/...` passes; no imports of deleted packages.

- [x] 4. Replace EP-006 escalation tests with EP-034 regression tests
  - Remove or rewrite: `handler_ep006_audit_test.go`, `run_ep006_escalation_test.go`, `tests/integration/ep006_escalation_run_test.go`, `escalationpolicy/*_test.go`, `toolfailure/*_test.go`.
  - Add tests with `// Covers AC-34.001` (no provider change on tool failure) and `// Covers AC-34.004` (transport fallback).
  - _Requirements:_ [REQ-34.013](ep-requirements.md#req-34-013--remove-ep-006-escalation-tests), [REQ-34.014](ep-requirements.md#req-34-014--add-no-escalation-regression-tests)
  - _Acceptance Criteria:_ [AC-34.013](ep-acceptance-criteria.md#ac-34-013), [AC-34.014](ep-acceptance-criteria.md#ac-34-014)
  - **Verification:** new tests pass; grep confirms no `escalationpolicy` / `toolfailure` in product code.

- [x] 5. Update operator docs and threat model
  - Update `docs/configuration.md`, `docs/llm-provider-roles-and-logging.md`, `docs/operations.md`, `docs/troubleshooting.md`.
  - Update `ai-sdlc-artefacts/threat-model.md` (remove escalation as active control).
  - Note EP-006 supersession in epic scope (already present).
  - _Requirements:_ [REQ-34.011](ep-requirements.md#req-34-011--update-operator-docs), [REQ-34.012](ep-requirements.md#req-34-012--document-ep-006-supersession)
  - _Acceptance Criteria:_ [AC-34.011](ep-acceptance-criteria.md#ac-34-011), [AC-34.012](ep-acceptance-criteria.md#ac-34-012)
  - **Verification:** manual grep — no `llm_escalation` / tool-path escalation in `docs/`.

- [x] 6. Quality gates
  - Run `make check`.
  - Run `./bin/validate EP-034`.
  - _Requirements:_ [REQ-34.015](ep-requirements.md#req-34-015--make-check-passes), [REQ-34.016](ep-requirements.md#req-34-016--validate-ep-034-passes)
  - _Acceptance Criteria:_ [AC-34.015](ep-acceptance-criteria.md#ac-34-015), [AC-34.016](ep-acceptance-criteria.md#ac-34-016)
  - **Verification:** both commands exit `0`.

---

## Dependencies and order

- Task 2 depends on Task 1 (config no longer supplies escalation).
- Task 3 depends on Tasks 1 and 2.
- Task 4 depends on Task 3.
- Task 5 can run in parallel with Task 4 after Task 3.
- Task 6 depends on Tasks 1–5.

---

## Checkpoints

- After Task 3: confirm transport fallback still works with two mock providers.
- After Task 4: confirm no remaining EP-006 escalation-only tests.
- Final checkpoint: `make check` and `./bin/validate EP-034` before stage 10.
