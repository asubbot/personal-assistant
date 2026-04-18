# EP-024 — Implementation plan

**Purpose:** Execute [pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md) stage 9 from this ordered task list.

**Previous / related:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-requirements.md](ep-requirements.md) · [ep-system-design.md](ep-system-design.md) · [ep-system-design-review.md](ep-system-design-review.md) (iteration 2 Pass) · [strategy.md](../../strategy.md)

**Checkpoints:** Run `make check` and `make build && ./bin/validate EP-024` before declaring the epic complete.

---

## Task list

- [x] **1** — Add `docs/llm-provider-roles-and-logging.md` describing the provider pool, main conversation routing (`tools.llm_escalation`, baseline index, transport fallback), summarization adapter (`SummarizeRouterConfig`), intent classifier model stage (separate from pool indices), three labelled configuration sketches, and `PA_ENV=development` guidance for diagnostic sessions.
  - _Requirements:_ [REQ-24.001](ep-requirements.md#operator-documentation), [REQ-24.002](ep-requirements.md#operator-documentation), [REQ-24.003](ep-requirements.md#operator-documentation), [REQ-24.004](ep-requirements.md#operator-documentation), [REQ-24.005](ep-requirements.md#operator-documentation), [REQ-24.009](ep-requirements.md#operator-documentation)
  - _Acceptance Criteria:_ [AC-24.001](ep-acceptance-criteria.md#ac-24-001), [AC-24.002](ep-acceptance-criteria.md#ac-24-002), [AC-24.003](ep-acceptance-criteria.md#ac-24-003), [AC-24.004](ep-acceptance-criteria.md#ac-24-004), [AC-24.005](ep-acceptance-criteria.md#ac-24-005), [AC-24.006](ep-acceptance-criteria.md#ac-24-006)
  - **Verification:** File exists; spot-check against `internal/llmrouter`, `cmd/pa` `buildAppLLM` / `SummarizeRouterConfig`, `buildIntentClassifier`.

- [x] **2** — Link the new page from `docs/configuration.md` (environment table and/or `llm_providers` overview) and add a short entry to `docs/README.md` if that index lists topical guides.
  - _Requirements:_ [REQ-24.001](ep-requirements.md#operator-documentation)
  - _Acceptance Criteria:_ [AC-24.001](ep-acceptance-criteria.md#ac-24-001)
  - **Verification:** Markdown links resolve in the repo.

- [x] **3** — Set `ENV PA_LOG_LEVEL=info` in the root `Dockerfile` runtime stage; add `PA_LOG_LEVEL=${PA_LOG_LEVEL:-info}` to the `pa` service `environment` in `docker-compose.yml`. Update `docs/docker.md` to state the image and Compose defaults explicitly.
  - _Requirements:_ [REQ-24.006](ep-requirements.md#docker-defaults), [REQ-24.007](ep-requirements.md#docker-defaults)
  - _Acceptance Criteria:_ [AC-24.007](ep-acceptance-criteria.md#ac-24-007), [AC-24.008](ep-acceptance-criteria.md#ac-24-008)
  - **Verification:** Grep `Dockerfile` / `docker-compose.yml`; automated tests from task 5.

- [x] **4** — In `cmd/pa`, implement `warnSensitiveLLMLogging(logger, level)` and call it once from `main` immediately after the root logger is constructed, matching [REQ-24.008](ep-requirements.md#startup-policy) and [AC-24.009](ep-acceptance-criteria.md#ac-24-009).
  - _Requirements:_ [REQ-24.008](ep-requirements.md#startup-policy)
  - _Acceptance Criteria:_ [AC-24.009](ep-acceptance-criteria.md#ac-24-009)
  - **Verification:** Unit tests in task 5.

- [x] **5** — Add `cmd/pa` tests: Dockerfile/compose content assertions; documentation phrase tests; `warnSensitiveLLMLogging` matrix (unset `PA_ENV`, non-development values, `development` / `DEVELOPMENT`, `info`). Use `// Covers AC-24.NNN` and `// Supporting AC-24.010` per repository convention.
  - _Requirements:_ —
  - _Acceptance Criteria:_ [AC-24.001](ep-acceptance-criteria.md#ac-24-001) through [AC-24.009](ep-acceptance-criteria.md#ac-24-009), supporting [AC-24.010](ep-acceptance-criteria.md#ac-24-010)
  - **Verification:** `go test ./cmd/pa -count=1 -run EP024` (or equivalent) passes.

- [x] **6** — Checkpoint: run `make check` and `make build && ./bin/validate EP-024`; fix any failures.
  - _Requirements:_ [REQ-24.010](ep-requirements.md#verification)
  - _Acceptance Criteria:_ [AC-24.010](ep-acceptance-criteria.md#ac-24-010)
  - **Verification:** Exit code 0 for both commands.

---

## Traceability note

Stages 10–11 (code review, audit) follow this plan on the epic branch; `./bin/validate EP-024` must report full AC coverage before the audit claims completion.
