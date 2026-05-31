---
artefact: ep-implementation-plan
epic_id: EP-040
status: draft
source_of_truth: true
updated_at: 2026-05-31
---

# EP-040 — Implementation plan

**Branch:** `epic/EP-040-handler-dependency-grouping`

## Tasks

- [x] **1.1** Add `handlerToolDeps`, `handlerMemoryDeps`, `handlerSessionDeps`, `handlerLLMDeps` and update `conversationHandler` in `handler.go`.
  - _REQ:_ REQ-40.001–004 | _AC:_ AC-40.001
  - **Verification:** `go build ./internal/core/...`

- [x] **1.2** Mechanical field access update across `handler*.go` (`h.tools.`, `h.memory.`, `h.session.`, `h.llm.`).
  - _REQ:_ REQ-40.005 | _AC:_ AC-40.002
  - **Verification:** `go build ./internal/core/...`

- [x] **1.3** Refactor `newRunConversationHandler` in `run.go` to construct groups.
  - _REQ:_ REQ-40.006 | _AC:_ AC-40.003

- [x] **1.4** Update test helpers / integration constructor; run full core tests.
  - _REQ:_ REQ-40.007–008 | _AC:_ AC-40.004, AC-40.005
  - **Verification:** `go test ./internal/core/... -count=1`

- [x] **1.5** `make check`; add `ep040_traceability_test.go` with `// Covers AC-40.001`.
  - _REQ:_ REQ-40.010 | _AC:_ AC-40.007
