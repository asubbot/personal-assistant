# EP-027 — Implementation plan

**Purpose:** Execute [pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md) stage 9 from this ordered task list.

**Previous / related:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-requirements.md](ep-requirements.md) · [ep-system-design.md](ep-system-design.md) · [ep-system-design-review.md](ep-system-design-review.md) · [strategy.md](../../strategy.md)

**Checkpoints:** Run `make check` and `make build && ./bin/validate EP-027` before declaring the epic complete.

---

## Task list

- [x] **1** — Add `setup_infra.go` with `paInfrastructure`, `buildPAInfrastructure`, and subsystem setup helpers; remove monolithic `setup` from `main.go`.
  - _Requirements:_ [REQ-27.001](ep-requirements.md#composition-root)
  - _Acceptance Criteria:_ [AC-27.001](ep-acceptance-criteria.md#ac-27-001)
  - **Verification:** `go test -tags=integration ./cmd/pa -run EP027 -count=1`

- [x] **2** — Add `application.go` with `paApplication`, `newPAApplication`, `Close`, `stopMemorySummarization`, and staged wiring methods; refactor `runServer` to delegate.
  - _Requirements:_ [REQ-27.002](ep-requirements.md#application-type), [REQ-27.003](ep-requirements.md#server-entry)
  - _Acceptance Criteria:_ [AC-27.002](ep-acceptance-criteria.md#ac-27-002), [AC-27.003](ep-acceptance-criteria.md#ac-27-003)
  - **Verification:** Read `main.go` `runServer`

- [x] **3** — Extend `CreateScheduledJobTool` with `NewCreateScheduledJobToolWithRuntimeLookup`; wire `cmd/pa` tool registry to snapshot-based lookup; add `jobs` unit tests.
  - _Requirements:_ [REQ-27.004](ep-requirements.md#jobs-hand-off)
  - _Acceptance Criteria:_ [AC-27.004](ep-acceptance-criteria.md#ac-27-004)
  - **Verification:** `go test ./internal/jobs -run CreateScheduledJobTool_RuntimeLookup -count=1`

- [x] **4** — Remove `//nolint:gocyclo` from deleted `setup`/`runServer` paths; add `cmd/pa` policy test for startup sources.
  - _Requirements:_ [REQ-27.005](ep-requirements.md#lint)
  - _Acceptance Criteria:_ [AC-27.005](ep-acceptance-criteria.md#ac-27-005)
  - **Verification:** `go test ./cmd/pa -run TestEP027_StartupSourcesHaveNoGocycloNolint -count=1`

- [x] **5** — Checkpoint: `make check` and `make build && ./bin/validate EP-027`.
  - _Requirements:_ [REQ-27.006](ep-requirements.md#verification)
  - _Acceptance Criteria:_ [AC-27.006](ep-acceptance-criteria.md#ac-27-006)
  - **Verification:** Exit code 0.

---

## Traceability note

Stages 10–11 reference this plan on the epic branch.
