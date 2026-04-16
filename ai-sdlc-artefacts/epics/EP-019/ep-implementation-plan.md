# EP-019 — Implementation plan

**Purpose:** Implement Scheduled Agent Jobs with Telegram management commands, replace legacy `scheduled_tasks`, and deliver verified, testable behavior for EP-019.

**Pipeline:** Stage 8 (implementation planning)  
**Epic:** [ep-scope.md](ep-scope.md)

## Related artefacts

- Requirements: [ep-requirements.md](ep-requirements.md)
- Acceptance criteria: [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- System design: [ep-system-design.md](ep-system-design.md)
- System design review: [ep-system-design-review.md](ep-system-design-review.md)
- Project strategy: [../../strategy.md](../../strategy.md)

---

## Task list

- [x] **1. Add configuration contract for jobs DB and legacy rejection**
  - Introduce `paths.jobs_db_path` with path resolution rules consistent with existing `paths.*`.
  - Add config validation for required scheduling fields and explicit startup failure for legacy `scheduled_tasks` fields.
  - Update config examples to include new path and remove legacy scheduler references.
  - _Requirements:_ [REQ-19.019](ep-requirements.md#legacy-replacement-and-configuration), [REQ-19.020](ep-requirements.md#legacy-replacement-and-configuration)
  - _Acceptance Criteria:_ [AC-19.019](ep-acceptance-criteria.md#ac-19-019), [AC-19.020](ep-acceptance-criteria.md#ac-19-020)
  - **Verification:** unit tests for config load/validation pass; startup fails with explicit unsupported legacy fields; docs/examples compile logically with config schema.

- [x] **2. Implement SQLite Job Store schema and repository APIs** *(depends on Task 1)*
  - Add `jobs`, `job_runs`, `delete_challenges`, and `schema_migrations` tables in dedicated `jobs.sqlite`.
  - Implement CRUD and run-record methods with stable Job ID handling.
  - Ensure startup load of persisted jobs and durable updates for status/run metadata.
  - _Requirements:_ [REQ-19.001](ep-requirements.md#job-model-and-scheduling), [REQ-19.002](ep-requirements.md#job-model-and-scheduling), [REQ-19.004](ep-requirements.md#job-model-and-scheduling), [REQ-19.017](ep-requirements.md#telegram-job-management)
  - _Acceptance Criteria:_ [AC-19.001](ep-acceptance-criteria.md#ac-19-001), [AC-19.002](ep-acceptance-criteria.md#ac-19-002), [AC-19.004](ep-acceptance-criteria.md#ac-19-004), [AC-19.017](ep-acceptance-criteria.md#ac-19-017)
  - **Verification:** repository tests cover create/get/list/update/delete/run recording; migration initialization works on empty DB.

- [x] **3. Implement scheduler runtime with timezone, due evaluation, overlap, timeout** *(depends on Task 2)*
  - Build due-evaluation pipeline from cron + timezone.
  - Implement single-instance overlap skip policy and timeout policy enforcement.
  - Create run records for due and run-now triggers with deterministic outcomes.
  - _Requirements:_ [REQ-19.003](ep-requirements.md#job-model-and-scheduling), [REQ-19.005](ep-requirements.md#job-execution-and-delivery), [REQ-19.009](ep-requirements.md#job-execution-and-delivery), [REQ-19.010](ep-requirements.md#job-execution-and-delivery)
  - _Acceptance Criteria:_ [AC-19.003](ep-acceptance-criteria.md#ac-19-003), [AC-19.005](ep-acceptance-criteria.md#ac-19-005), [AC-19.009](ep-acceptance-criteria.md#ac-19-009), [AC-19.010](ep-acceptance-criteria.md#ac-19-010)
  - **Verification:** scheduler unit/integration tests validate due timing, overlap skip, timeout outcomes, and next-run updates.

- [x] **4. Implement job executor and Telegram delivery pipeline** *(depends on Task 3)*
  - Wire scheduled run execution to the standard agent-turn path.
  - Deliver success result messages and failure-class messages to configured Telegram targets.
  - Persist run outcome fields for diagnostics and management `show`.
  - _Requirements:_ [REQ-19.006](ep-requirements.md#job-execution-and-delivery), [REQ-19.007](ep-requirements.md#job-execution-and-delivery), [REQ-19.008](ep-requirements.md#job-execution-and-delivery)
  - _Acceptance Criteria:_ [AC-19.006](ep-acceptance-criteria.md#ac-19-006), [AC-19.007](ep-acceptance-criteria.md#ac-19-007), [AC-19.008](ep-acceptance-criteria.md#ac-19-008)
  - **Verification:** integration tests assert standard execution path invocation and Telegram success/failure delivery behavior.

- [x] **5. Implement Telegram management command service** *(depends on Task 2, Task 3)*
  - Add command handlers for `list`, `show`, `pause`, `resume`, `run-now`.
  - Ensure output includes required fields (id, schedule, timezone, status, next run, details).
  - Enforce deterministic responses for invalid/unknown IDs.
  - _Requirements:_ [REQ-19.011](ep-requirements.md#telegram-job-management), [REQ-19.012](ep-requirements.md#telegram-job-management), [REQ-19.013](ep-requirements.md#telegram-job-management), [REQ-19.014](ep-requirements.md#telegram-job-management), [REQ-19.015](ep-requirements.md#telegram-job-management)
  - _Acceptance Criteria:_ [AC-19.011](ep-acceptance-criteria.md#ac-19-011), [AC-19.012](ep-acceptance-criteria.md#ac-19-012), [AC-19.013](ep-acceptance-criteria.md#ac-19-013), [AC-19.014](ep-acceptance-criteria.md#ac-19-014), [AC-19.015](ep-acceptance-criteria.md#ac-19-015)
  - **Verification:** adapter/service integration tests for each command and state transition.

- [x] **6. Implement authz gate, delete confirmation flow, and audit events** *(depends on Task 5)*
  - Restrict management commands to authorized operators.
  - Implement two-step delete (`delete` challenge + `confirm-delete` token).
  - Add structured audit events for management operations and run lifecycle transitions.
  - _Requirements:_ [REQ-19.016](ep-requirements.md#telegram-job-management), [REQ-19.017](ep-requirements.md#telegram-job-management), [REQ-19.018](ep-requirements.md#telegram-job-management), [REQ-19.021](ep-requirements.md#non-functional-requirements)
  - _Acceptance Criteria:_ [AC-19.016](ep-acceptance-criteria.md#ac-19-016), [AC-19.017](ep-acceptance-criteria.md#ac-19-017), [AC-19.018](ep-acceptance-criteria.md#ac-19-018), [AC-19.021](ep-acceptance-criteria.md#ac-19-021)
  - **Verification:** tests for unauthorized rejection + audit, token mismatch/expiry handling, and successful confirm-delete.

- [x] **7. Add startup readiness gate and pre-ready command behavior** *(depends on Task 2, Task 5)*
  - Implement readiness contract: management commands return deterministic "scheduler initializing" before initial load completes.
  - Ensure no lifecycle mutation before readiness true.
  - _Requirements:_ [REQ-19.002](ep-requirements.md#job-model-and-scheduling)
  - _Acceptance Criteria:_ [AC-19.002](ep-acceptance-criteria.md#ac-19-002)
  - **Verification:** startup integration tests simulate early command arrival before and after readiness transition.

- [x] **8. Define profile-based responsiveness acceptance harness for `list`** *(depends on Task 5)*
  - Add profile-driven acceptance test configuration and test runner hook for `list` responsiveness checks.
  - Keep profile thresholds external to runtime config, per design contract.
  - _Requirements:_ [REQ-19.022](ep-requirements.md#non-functional-requirements)
  - _Acceptance Criteria:_ [AC-19.022](ep-acceptance-criteria.md#ac-19-022)
  - **Verification:** acceptance test passes for selected deployment profile and fails when threshold is violated.

- [x] **9. End-to-end scenario and regression sweep** *(depends on Tasks 1-8)*
  - Implement E2E scenario: scheduled digest delivery + operator `list` + delete confirmation lifecycle.
  - Remove legacy `scheduled_tasks` runtime path and legacy-oriented scheduler tests.
  - Update operator docs (`docs/configuration.md` and examples) to new model only.
  - _Requirements:_ [REQ-19.007](ep-requirements.md#job-execution-and-delivery), [REQ-19.011](ep-requirements.md#telegram-job-management), [REQ-19.016](ep-requirements.md#telegram-job-management), [REQ-19.019](ep-requirements.md#legacy-replacement-and-configuration), [REQ-19.020](ep-requirements.md#legacy-replacement-and-configuration)
  - _Acceptance Criteria:_ [AC-19.023](ep-acceptance-criteria.md#ac-19-023), [AC-19.019](ep-acceptance-criteria.md#ac-19-019), [AC-19.020](ep-acceptance-criteria.md#ac-19-020)
  - **Verification:** E2E passes; no legacy path remains in runtime wiring; docs/examples reflect only new schema.

---

## Checkpoints

- **Checkpoint A (after Task 3):** scheduler core behaviors verified before Telegram command layer expansion.
- **Checkpoint B (after Task 6):** security and destructive-flow behavior reviewed before E2E.
- **Checkpoint C (final):** run full quality gate (`make check`) and validate AC coverage for EP-019 (`./bin/validate EP-019` after coverage mapping is in place).
- **Checkpoint D (operator sync):** if implementation reveals schema or behavior gaps, update requirements/AC/design artefacts before merging.
