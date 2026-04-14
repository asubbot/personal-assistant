# EP-015 — Implementation plan

**Pipeline:** Stage 8.  
**Related:** [ep-scope.md](ep-scope.md), [ep-requirements.md](ep-requirements.md), [ep-acceptance-criteria.md](ep-acceptance-criteria.md), [ep-system-design.md](ep-system-design.md), [ep-system-design-review.md](ep-system-design-review.md), [strategy.md](../../strategy.md)

## Checkpoints

- After each task: `go test ./internal/core/... ./internal/telegram/...` for touched packages.
- Before completion: `make check` and `./bin/validate EP-015`.

## Tasks

- [x] **1** Per-turn usage accumulator and footer formatting (core)
  - Add a small type (for example `usageTurnAcc`) with `add(llm.Usage)` and `footerLine() string` implementing [REQ-15.001](ep-requirements.md#req-15-001), [REQ-15.002](ep-requirements.md#req-15-002), [REQ-15.006](ep-requirements.md#req-15-006).
  - _Requirements:_ [REQ-15.001](ep-requirements.md#req-15-001), [REQ-15.002](ep-requirements.md#req-15-002), [REQ-15.006](ep-requirements.md#req-15-006)
  - _Acceptance Criteria:_ [AC-15.001](ep-acceptance-criteria.md#ac-15-001)
  - **Verification:** Unit tests for `footerLine` with summed values; `go test ./internal/core/...` passes.

- [x] **2** Wire accumulator through all successful completions in one turn (core)
  - Extend `completeAt` (or equivalent) to accept optional accumulator; update `HandleMessage`, `finishAfterFirstLLM`, `runToolResultLoop`, and `resolveHermesFollowUpCompletion` call chains per [ep-system-design.md](ep-system-design.md).
  - On successful `Complete` only, add `result.Usage` to sums.
  - _Requirements:_ [REQ-15.001](ep-requirements.md#req-15-001), [REQ-15.002](ep-requirements.md#req-15-002), [REQ-15.003](ep-requirements.md#req-15-003)
  - _Acceptance Criteria:_ [AC-15.001](ep-acceptance-criteria.md#ac-15-001), [AC-15.002](ep-acceptance-criteria.md#ac-15-002)
  - **Verification:** Handler tests with mock provider returning usage on first and second completion; assert sums.

- [x] **3** Append footer to `HandleMessage` return value; keep session body clean (core)
  - When `footerLine()` non-empty and trimmed assistant body non-empty, return `body + "\n" + footer`.
  - Ensure `appendSessionIfEnabled` receives assistant text **without** the footer ([REQ-15.009](ep-requirements.md#req-15-009)).
  - _Requirements:_ [REQ-15.004](ep-requirements.md#req-15-004), [REQ-15.005](ep-requirements.md#req-15-005), [REQ-15.009](ep-requirements.md#req-15-009)
  - _Acceptance Criteria:_ [AC-15.001](ep-acceptance-criteria.md#ac-15-001), [AC-15.002](ep-acceptance-criteria.md#ac-15-002), [AC-15.005](ep-acceptance-criteria.md#ac-15-005)
  - **Verification:** Session snapshot test or handler test with session memory enabled.

- [x] **4** Telegram: split body, append footer to last chunk (telegram)
  - Implement suffix split helper (strict end pattern) and integrate into `sendLongOutboundText` per design.
  - Handle empty body + footer ([AC-15.004](ep-acceptance-criteria.md#ac-15-004)); handle overflow by extra final chunk if needed.
  - _Requirements:_ [REQ-15.007](ep-requirements.md#req-15-007), [REQ-15.008](ep-requirements.md#req-15-008)
  - _Acceptance Criteria:_ [AC-15.003](ep-acceptance-criteria.md#ac-15-003), [AC-15.004](ep-acceptance-criteria.md#ac-15-004), [AC-15.006](ep-acceptance-criteria.md#ac-15-006)
  - **Verification:** `internal/telegram` tests with mock outbound recording all `SendMessage` payloads.

- [x] **5** AC trace comments and validation
  - Add `// Covers AC-15.NNN` to new and updated tests per [VALIDATION.md](../../tools/validate/VALIDATION.md).
  - _Requirements:_ [REQ-15.010](ep-requirements.md#req-15-010), [REQ-15.012](ep-requirements.md#req-15-012)
  - _Acceptance Criteria:_ [AC-15.007](ep-acceptance-criteria.md#ac-15-007)
  - **Verification:** `make build && ./bin/validate EP-015` exit code 0.
