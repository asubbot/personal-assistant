# EP-025 — Implementation plan

**Purpose:** Execute [pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md) stage 9 from this ordered task list.

**Previous / related:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-requirements.md](ep-requirements.md) · [ep-system-design.md](ep-system-design.md) · [ep-system-design-review.md](ep-system-design-review.md) · [strategy.md](../../strategy.md)

**Checkpoints:** Run `make check` and `make build && ./bin/validate EP-025` before declaring the epic complete.

---

## Task list

- [x] **1** — Introduce `internal/jobs/delivery_runner.go` with `DeliveryRunner`, `JobChatSender`, `NewDeliveryRunner`, and failure classification moved from `cmd/pa`; wire `initJobsRuntimeAsync` to `jobs.NewDeliveryRunner`; update `cmd/pa/jobs_runtime_test.go` to use the new constructor.
  - _Requirements:_ [REQ-25.007](ep-requirements.md#refactor)
  - _Acceptance Criteria:_ [AC-25.007](ep-acceptance-criteria.md#ac-25-007), supporting [AC-25.008](ep-acceptance-criteria.md#ac-25-008)
  - **Verification:** `go test -tags=integration -count=1 ./internal/jobs/...`

- [x] **2** — Remove `cmd/pa` e2e test files; add `tests/e2e/jobs_e2e_test.go` (`//go:build e2e`) with merged job digest/management flows and local test doubles; add `tests/e2e/placeholder_test.go` (`//go:build !e2e`) so default builds include the package.
  - _Requirements:_ [REQ-25.001](ep-requirements.md#test-layout), [REQ-25.002](ep-requirements.md#test-layout)
  - _Acceptance Criteria:_ [AC-25.001](ep-acceptance-criteria.md#ac-25-001), [AC-25.002](ep-acceptance-criteria.md#ac-25-002)
  - **Verification:** `go test -tags=integration,e2e -count=1 ./tests/e2e/...` and `go test -tags=integration -count=1 ./tests/e2e/...`

- [x] **3** — Add Makefile targets `test-e2e` and `coverage-e2e`; extend `check` to run `test-e2e` after `test-race`; keep default `coverage` target free of the `e2e` substring on the primary `go test` line; pass `integration,e2e` to vet/vuln/lint where applicable.
  - _Requirements:_ [REQ-25.003](ep-requirements.md#make-targets), [REQ-25.004](ep-requirements.md#make-targets), [REQ-25.006](ep-requirements.md#coverage), [REQ-25.008](ep-requirements.md#verification)
  - _Acceptance Criteria:_ [AC-25.003](ep-acceptance-criteria.md#ac-25-003) through [AC-25.006](ep-acceptance-criteria.md#ac-25-006), [AC-25.008](ep-acceptance-criteria.md#ac-25-008)
  - **Verification:** Policy tests in task 4; `make test-e2e`

- [x] **4** — Add `tests/e2e/ep025_policy_test.go` (`//go:build !e2e`) asserting Makefile and `.github/workflows/ci.yml` content per AC-25.003–AC-25.006 (and supporting AC-25.008 where noted).
  - _Requirements:_ [REQ-25.003](ep-requirements.md#make-targets)–[REQ-25.006](ep-requirements.md#coverage)
  - _Acceptance Criteria:_ [AC-25.003](ep-acceptance-criteria.md#ac-25-003)–[AC-25.006](ep-acceptance-criteria.md#ac-25-006)
  - **Verification:** `go test -tags=integration -count=1 ./tests/e2e/... -run EP025`

- [x] **5** — Add `internal/jobs/delivery_runner_test.go` with `// Covers AC-25.007` (and Supporting AC-25.008 where applicable); ensure imports satisfy repository `gofumpt` / golangci rules.
  - _Requirements:_ [REQ-25.007](ep-requirements.md#refactor), [REQ-25.008](ep-requirements.md#verification)
  - _Acceptance Criteria:_ [AC-25.007](ep-acceptance-criteria.md#ac-25-007), [AC-25.008](ep-acceptance-criteria.md#ac-25-008)
  - **Verification:** `make lint`

- [x] **6** — Checkpoint: run `make check` and `make build && ./bin/validate EP-025`; fix any failures.
  - _Requirements:_ [REQ-25.008](ep-requirements.md#verification)
  - _Acceptance Criteria:_ [AC-25.008](ep-acceptance-criteria.md#ac-25-008)
  - **Verification:** Exit code 0 for both commands.

---

## Traceability note

Stages 10–11 follow this plan on the epic branch; `./bin/validate EP-025` must report full AC coverage before the audit claims completion.
