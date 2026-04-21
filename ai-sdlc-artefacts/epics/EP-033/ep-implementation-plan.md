# EP-033 — Implementation plan

Pipeline stage 8 output for EP-033.  
Purpose: deliver bounded retries for failed day summarization jobs in `memoryjob` (`catchup_day`, `summarize_yesterday`) with deterministic backoff and dedupe, without widening scope to month/year or date-range catch-up.

**Related artefacts**

- Scope: [ep-scope.md](ep-scope.md)
- Requirements: [ep-requirements.md](ep-requirements.md)
- Acceptance criteria: [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- System design: [ep-system-design.md](ep-system-design.md)
- Design review: [ep-system-design-review.md](ep-system-design-review.md)

---

## Task list

- [x] 1. Extend memoryjob queue model with retry metadata and deterministic scheduling gate
  - Add retry metadata for day jobs (`attempt`, `not_before`, retry policy, dedupe key) to queue item model.
  - Add deterministic scheduling gate so queue does not execute retry before `not_before`.
  - Keep existing queue priority behavior unchanged.
  - _Requirements:_ [REQ-33.004](ep-requirements.md#retry-scheduling-behavior), [REQ-33.008](ep-requirements.md#queue-semantics), [REQ-33.011](ep-requirements.md#verification)
  - _Acceptance Criteria:_ [AC-33.004](ep-acceptance-criteria.md#ac-33-004), [AC-33.008](ep-acceptance-criteria.md#ac-33-008), [AC-33.011](ep-acceptance-criteria.md#ac-33-011)
  - **Verification:** memoryjob unit tests confirm retry queue item timing gate and deterministic scheduling behavior.

- [x] 2. Implement bounded retry policy for day jobs and exhaustion behavior
  - Apply retry policy to `catchup_day` and `summarize_yesterday` failures only.
  - Stop retries at max attempts and emit terminal exhaustion path.
  - Keep month/year catch-up and rollups unchanged.
  - _Requirements:_ [REQ-33.001](ep-requirements.md#retry-policy-scope), [REQ-33.002](ep-requirements.md#retry-policy-scope), [REQ-33.003](ep-requirements.md#retry-policy-scope), [REQ-33.005](ep-requirements.md#retry-scheduling-behavior)
  - _Acceptance Criteria:_ [AC-33.001](ep-acceptance-criteria.md#ac-33-001), [AC-33.002](ep-acceptance-criteria.md#ac-33-002), [AC-33.003](ep-acceptance-criteria.md#ac-33-003), [AC-33.005](ep-acceptance-criteria.md#ac-33-005)
  - **Verification:** tests show successful retry for transient failures and no retry beyond exhaustion limit.

- [x] 3. Add dedupe and concurrency-safe enqueue semantics for day retry chains
  - Ensure one retry chain per day target key.
  - Guarantee atomic dedupe check + queue insert under runner mutex.
  - Preserve behavior when jobs are deferred due to active user turn.
  - _Requirements:_ [REQ-33.006](ep-requirements.md#retry-scheduling-behavior), [REQ-33.007](ep-requirements.md#queue-semantics)
  - _Acceptance Criteria:_ [AC-33.006](ep-acceptance-criteria.md#ac-33-006), [AC-33.007](ep-acceptance-criteria.md#ac-33-007)
  - **Verification:** tests cover duplicate enqueue prevention and deferred execution compatibility.

- [x] 4. Add retry observability and preserve existing day success path
  - Emit structured logs for retry scheduling and retry exhaustion.
  - Keep successful day summary write + vector index flow unchanged.
  - _Requirements:_ [REQ-33.009](ep-requirements.md#observability), [REQ-33.010](ep-requirements.md#existing-behavior-preservation)
  - _Acceptance Criteria:_ [AC-33.009](ep-acceptance-criteria.md#ac-33-009), [AC-33.010](ep-acceptance-criteria.md#ac-33-010)
  - **Verification:** tests/log assertions confirm retry fields are logged and baseline success-path tests still pass.

- [x] 5. Add AC trace tests and run quality gates
  - Add or update tests with explicit `Covers AC-33.xxx` comments.
  - Run `make check`.
  - Build validator and run `./bin/validate EP-033`.
  - _Requirements:_ [REQ-33.012](ep-requirements.md#verification), [REQ-33.013](ep-requirements.md#verification), [REQ-33.014](ep-requirements.md#verification)
  - _Acceptance Criteria:_ [AC-33.012](ep-acceptance-criteria.md#ac-33-012), [AC-33.013](ep-acceptance-criteria.md#ac-33-013), [AC-33.014](ep-acceptance-criteria.md#ac-33-014)
  - **Verification:** commands exit `0` and validator reports full AC coverage for EP-033.

---

## Dependencies and order

- Task 2 depends on Task 1.
- Task 3 depends on Tasks 1 and 2.
- Task 4 depends on Tasks 2 and 3.
- Task 5 depends on Tasks 1–4.

---

## Checkpoints

- After Task 2: verify retry scope remains day-only and month/year paths are unchanged.
- After Task 3: verify no duplicate retry chains for the same day target.
- Final checkpoint: run `make check` and `./bin/validate EP-033` before stage 10.
