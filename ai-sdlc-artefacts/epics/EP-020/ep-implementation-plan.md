# EP-020 - Implementation Plan

**Purpose:** Implement hybrid natural-language scheduled job creation from Telegram chat (deterministic parser first, native-tool fallback second) on top of EP-019 scheduler/runtime.  
**Pipeline:** Stage 8  
**Related:** [ep-scope.md](ep-scope.md), [ep-requirements.md](ep-requirements.md), [ep-acceptance-criteria.md](ep-acceptance-criteria.md), [ep-system-design.md](ep-system-design.md), [ep-system-design-review.md](ep-system-design-review.md)

## Task list

- [x] **1. Implement deterministic NL create parser and creation API in jobs manager**
  - Add strict syntax matching for MVP phrase shape (`<instruction> and send it at HH:MM every day`).
  - Add deterministic success responses, malformed-path fallthrough for base LLM fallback, path marker, and creation audit outcomes.
  - _Requirements:_ [REQ-20.001](ep-requirements.md#nl-request-intake), [REQ-20.002](ep-requirements.md#nl-request-intake), [REQ-20.006](ep-requirements.md#user-feedback-safety-and-compatibility), [REQ-20.007](ep-requirements.md#user-feedback-safety-and-compatibility), [REQ-20.011](ep-requirements.md#runtime-security-and-observability), [REQ-20.012](ep-requirements.md#runtime-security-and-observability)
  - _Acceptance Criteria:_ [AC-20.001](ep-acceptance-criteria.md#ac-20-001), [AC-20.003](ep-acceptance-criteria.md#ac-20-003), [AC-20.004](ep-acceptance-criteria.md#ac-20-004), [AC-20.008](ep-acceptance-criteria.md#ac-20-008)
  - **Verification:** manager unit tests pass for success/malformed/non-match/audit.

- [x] **2. Persist NL-created jobs with scheduler-compatible defaults**
  - Use active status, default timezone (`pa_timezone`), and resolved delivery target chat.
  - Compute and persist next run timestamp at creation time.
  - _Requirements:_ [REQ-20.003](ep-requirements.md#job-creation), [REQ-20.004](ep-requirements.md#job-creation), [REQ-20.005](ep-requirements.md#job-creation)
  - _Acceptance Criteria:_ [AC-20.002](ep-acceptance-criteria.md#ac-20-002)
  - **Verification:** created job fields are asserted by unit tests.

- [x] **3. Implement native create tool fallback for explicit schedule-intent free-form messages**
  - Add `create_scheduled_job` native tool contract with strict input validation (instruction, HH:MM, timezone optional).
  - Reuse manager creation API for persistence to keep one write path.
  - _Requirements:_ [REQ-20.013](ep-requirements.md#nl-request-intake), [REQ-20.003](ep-requirements.md#job-creation), [REQ-20.006](ep-requirements.md#user-feedback-safety-and-compatibility), [REQ-20.011](ep-requirements.md#runtime-security-and-observability), [REQ-20.012](ep-requirements.md#runtime-security-and-observability)
  - _Acceptance Criteria:_ [AC-20.009](ep-acceptance-criteria.md#ac-20-009), [AC-20.003](ep-acceptance-criteria.md#ac-20-003), [AC-20.008](ep-acceptance-criteria.md#ac-20-008)
  - **Verification:** native tool tests pass for valid extract/create, invalid params, and deterministic error output.

- [x] **4. Extend message routing to invoke hybrid NL create flow safely**
  - In `jobsCommandHandler`, detect NL-create intent and route to manager while preserving existing `/jobs` command behavior.
  - Ensure deterministic parser is attempted before fallback tool path.
  - For malformed explicit schedule-intent non-matches, escalate to base LLM fallback with retry prompt that enforces `create_scheduled_job`.
  - Keep readiness/init-failed behavior consistent with scheduler gate.
  - _Requirements:_ [REQ-20.001](ep-requirements.md#nl-request-intake), [REQ-20.011](ep-requirements.md#runtime-security-and-observability)
  - _Acceptance Criteria:_ [AC-20.001](ep-acceptance-criteria.md#ac-20-001), [AC-20.007](ep-acceptance-criteria.md#ac-20-007)
  - **Verification:** routing tests pass for create/non-create/readiness cases.

- [x] **5. Wire timezone context from server config into creation manager**
  - Pass `pa_timezone` into jobs runtime initialization and manager defaults.
  - _Requirements:_ [REQ-20.004](ep-requirements.md#job-creation)
  - _Acceptance Criteria:_ [AC-20.002](ep-acceptance-criteria.md#ac-20-002)
  - **Verification:** integration tests confirm created job timezone equals configured timezone.

- [x] **6. Add end-to-end tests for deterministic and fallback creation -> management -> delivery**
  - Scenario A: strict NL template creates job, `/jobs list` shows it, run-now executes and delivers output.
  - Scenario B: explicit free-form schedule-intent creates job through fallback path, `/jobs show` returns expected metadata.
  - Include unauthorized and malformed paths in existing adapter/handler tests.
  - _Requirements:_ [REQ-20.008](ep-requirements.md#user-feedback-safety-and-compatibility), [REQ-20.009](ep-requirements.md#runtime-security-and-observability), [REQ-20.010](ep-requirements.md#runtime-security-and-observability), [REQ-20.013](ep-requirements.md#nl-request-intake)
  - _Acceptance Criteria:_ [AC-20.005](ep-acceptance-criteria.md#ac-20-005), [AC-20.006](ep-acceptance-criteria.md#ac-20-006), [AC-20.007](ep-acceptance-criteria.md#ac-20-007), [AC-20.009](ep-acceptance-criteria.md#ac-20-009)
  - **Verification:** EP-020 E2E/integration tests pass.

- [x] **7. Update operations/config docs and validate AC coverage**
  - Document strict syntax path, fallback path trigger rules, and validation behavior.
  - Run `make check` and `./bin/validate EP-020`.
  - _Requirements:_ [REQ-20.012](ep-requirements.md#runtime-security-and-observability)
  - _Acceptance Criteria:_ [AC-20.008](ep-acceptance-criteria.md#ac-20-008), [AC-20.009](ep-acceptance-criteria.md#ac-20-009)
  - **Verification:** checks pass and EP-020 AC coverage is 100%.

## Checkpoints

- **Checkpoint A (after Task 2):** deterministic parser + persistence behavior verified before fallback integration.
- **Checkpoint B (after Task 4):** hybrid routing behavior verified before timezone and E2E.
- **Checkpoint C (after Task 6):** full deterministic + fallback behavior verified through E2E before docs and final validation.
- **Checkpoint D (final):** `make check` and `./bin/validate EP-020` pass; ready for stage 10 review.
