# EP-006 — Implementation plan

**Pipeline:** Stage 7 ([07-implementation-planning.skill.md](../../../ai-sdlc/specification/skills/07-implementation-planning.skill.md)).

**Artefacts:** [ep-scope.md](ep-scope.md) · [ep-requirements.md](ep-requirements.md) · [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-system-design.md](ep-system-design.md) · [strategy.md](../../strategy.md)

**Goal:** Implement configurable baseline LLM provider, tool-driven escalation along the ordered provider list (bounded per user message), end-of-turn rollback to baseline, classification of tool failures, **centralized escalation-allowance mapping in `internal/escalationpolicy`** ([REQ-06.017](ep-requirements.md#nfr--security-testability-observability)), structured logging (no secrets), and tests.

---

## Checkpoints

- [x] After config + core wiring: `make check` passes.
- [x] After tests: every new test file references `Covers AC-06.NNN` where applicable.
- [ ] Before merge: user review of config shape and default (escalation off).

---

## Task list

- [x] **1. Config: `llm_escalation` and validation**
  - Add optional `llm_escalation` JSON object: `enabled` (bool), `max_per_user_message` (int, >= 0), `baseline_index` (int, 0-based into `llm_providers`).
  - When `enabled` is true: require `len(llm_providers) >= 2`, `baseline_index` in `[0, len-1)`, validate in `config.Load` / `validate`.
  - When section omitted or `enabled` false: no escalation chain in core (existing transport `FallbackProvider` only).
  - _Requirements:_ [REQ-06.002](ep-requirements.md#baseline-and-configuration)
  - _Acceptance Criteria:_ [AC-06.002](ep-acceptance-criteria.md#ac-06-002)
  - **Verification:** `go test ./internal/config/...` passes; invalid escalation config returns clear error at load.

- [x] **2. Build LLM chain vs transport in `cmd/pa`**
  - Extract building of individual `[]llm.Provider` and labels from current `newLLMProvider`.
  - If escalation enabled: pass non-nil `llmChain` + labels to `core.Run`, `llmTransport` nil.
  - If escalation disabled: pass `llmTransport` = `FallbackProvider`, `llmChain` nil (unchanged behaviour).
  - Scheduler / summarize / other entrypoints keep using `FallbackProvider` only.
  - _Requirements:_ [REQ-06.001](ep-requirements.md#baseline-and-configuration), [REQ-06.007](ep-requirements.md#escalation-policy-and-chain)
  - _Acceptance Criteria:_ [AC-06.001](ep-acceptance-criteria.md#ac-06-001), [AC-06.006](ep-acceptance-criteria.md#ac-06-006)
  - **Verification:** `go build -o /dev/null ./cmd/pa`; existing integration tests still compile and pass.

- [x] **3. Core `Run` + handler: dual mode (transport vs chain)**
  - Extend `core.Run` (and `newRunConversationHandler`) with `llmChain []llm.Provider` and `llmChainLabels []string` (parallel to existing `llm.Provider`).
  - When chain empty: use existing `conversationHandler.provider` (FallbackProvider).
  - When chain non-nil: store chain + labels + `cfg.LLMEscalation`; `completeAt(ctx, idx, messages, opts)` calls `chain[idx].Complete` and sets `result.Model` from label.
  - _Requirements:_ [REQ-06.001](ep-requirements.md#baseline-and-configuration), [REQ-06.006](ep-requirements.md#escalation-policy-and-chain), [REQ-06.014](ep-requirements.md#nfr--security-testability-observability)
  - _Acceptance Criteria:_ [AC-06.001](ep-acceptance-criteria.md#ac-06-001), [AC-06.005](ep-acceptance-criteria.md#ac-06-005), [AC-06.011](ep-acceptance-criteria.md#ac-06-011)
  - **Verification:** `go test ./internal/core/...`; `make check`.

- [x] **4. Typed tool failure package (`toolfailure`)**
  - `internal/core/toolfailure`: `Failure` type with `Escalate` bool, `NoEscalate(err)` / `MayEscalate(err)` wrappers, `Unwrap`; `QualifiesForEscalation` uses `errors.As` only (REQ-06.015); untyped errors do not qualify.
  - `internal/noderunner`: return typed failures for allowlist/connect/exec paths.
  - `executeOneToolCall`: wrap catalog/validate/substitute/cmdsafe/node-nil outcomes with `NoEscalate` or `MayEscalate` (unknown tool → no escalate). **Interim:** `wrapCatalogValidateError` in handler uses a substring bridge until **Task 8** moves mapping into `internal/escalationpolicy` and (follow-up) `toolcatalog` sentinels/kinds remove substring use ([REQ-06.015](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs)).
  - _Requirements:_ [REQ-06.003](ep-requirements.md#error-classification), [REQ-06.004](ep-requirements.md#error-classification), [REQ-06.005](ep-requirements.md#error-classification), [REQ-06.015](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs)
  - _Acceptance Criteria:_ [AC-06.003](ep-acceptance-criteria.md#ac-06-003), [AC-06.004](ep-acceptance-criteria.md#ac-06-004), [AC-06.012](ep-acceptance-criteria.md#ac-06-012)
  - **Verification:** unit tests in `toolfailure`; `make check`.

- [ ] **8. Package `internal/escalationpolicy` (central policy mapping)** — *not started; aligns with stakeholder choice*
  - Add `pa/internal/escalationpolicy` with `doc.go` describing scope: mapping **classified** tool-path causes to `toolfailure.NoEscalate` / `toolfailure.MayEscalate` ([REQ-06.004](ep-requirements.md#error-classification), [REQ-06.005](ep-requirements.md#error-classification), [REQ-06.017](ep-requirements.md#nfr--security-testability-observability)).
  - **Migrate** handler logic `wrapCatalogValidateError` into exported functions (e.g. `WrapCatalogValidateError(err error) error` or `ApplyCatalogValidatePolicy(err error) error`). Handler calls `escalationpolicy` from `executeOneToolCall`; remove or shrink inline policy in `handler.go`.
  - **Dependencies:** import only `pa/internal/core/toolfailure` initially; optionally `pa/internal/toolcatalog` after exported `ValidateError`/`Kind` or `errors.Is` sentinels ([REQ-06.015](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs)).
  - **Optional consolidation:** route `noderunner` through the same package for a single policy matrix (design decision; if deferred, document in package doc that node outcomes remain typed in `noderunner`).
  - **Tests:** table-driven unit tests in `internal/escalationpolicy`; **fail-closed** for unknown/unclassified input (no accidental `MayEscalate`); tests MUST NOT import `internal/core` handler or start Telegram/LLM ([REQ-06.017](ep-requirements.md#nfr--security-testability-observability)).
  - **Module boundaries:** confirm `scripts/check-module-boundaries.sh` after adding imports; `escalationpolicy` must not import `internal/core` (parent), `telegram`, or concrete LLM packages ([ep-system-design.md](ep-system-design.md)).
  - _Requirements:_ [REQ-06.017](ep-requirements.md#nfr--security-testability-observability), [REQ-06.004](ep-requirements.md#error-classification), [REQ-06.005](ep-requirements.md#error-classification)
  - _Acceptance Criteria:_ [AC-06.014](ep-acceptance-criteria.md#ac-06-014)
  - **Verification:** `go test ./internal/escalationpolicy/...`; `make check`.

- [x] **5. Per–user-message state: baseline, active index, escalation count**
  - At start of `HandleMessage`: `activeIdx = baseline_index`, `escalationUsed = 0` when escalation enabled.
  - Replace direct `provider.Complete` / `completeUserTurn` with `completeAt` using `activeIdx`.
  - In `runToolResultLoop` / `appendToolRound`: after executing tools in a round, if any tool returned an error and `QualifiesForEscalation` (typed) and escalation enabled and under cap, `maybeEscalate(..., "tool_execution")` then next `Complete` uses new index.
  - Hermes (REQ-06.016): after first `Complete`, if text-tool path and invalid Hermes parse, `maybeEscalate(..., "hermes_parse")` and retry `Complete` on next provider until success or no advance; follow-ups use `resolveHermesFollowUpCompletion`.
  - If cannot escalate further but rounds continue, keep using last provider until max tool rounds or success (deterministic stop per REQ-06.008).
  - _Requirements:_ [REQ-06.006](ep-requirements.md#escalation-policy-and-chain), [REQ-06.008](ep-requirements.md#exhaustion-and-stop), [REQ-06.009](ep-requirements.md#rollback-at-end-of-turn), [REQ-06.016](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs)
  - _Acceptance Criteria:_ [AC-06.005](ep-acceptance-criteria.md#ac-06-005), [AC-06.007](ep-acceptance-criteria.md#ac-06-007), [AC-06.008](ep-acceptance-criteria.md#ac-06-008), [AC-06.013](ep-acceptance-criteria.md#ac-06-013)
  - **Verification:** integration or handler test with mock providers; `make check`.

- [x] **6. Observability: escalation logs (no secrets)**
  - Log fields: `failure_class` (`tool_execution`, `hermes_parse`, …), `escalation` bool, `provider_index_before`, `provider_index_after`, optional `tried_providers` summary.
  - Use existing redaction for any user/tool content in same log line if needed.
  - _Requirements:_ [REQ-06.010](ep-requirements.md#observability), [REQ-06.011](ep-requirements.md#observability), [REQ-06.012](ep-requirements.md#nfr--security-testability-observability)
  - _Acceptance Criteria:_ [AC-06.009](ep-acceptance-criteria.md#ac-06-009)
  - **Verification:** grep tests or log capture in unit test; `make check`.

- [ ] **7. Integration tests and documentation** (partial: unit/run tests added; optional 3-provider E2E and README example deferred; may run after Task 8)
  - Add integration test: mock chain of 3 providers, qualifying failure on first → second provider receives next `Complete`; third user message starts from baseline (separate `HandleMessage` calls).
  - Optional: extend `config.json` example in repo docs/README with commented `llm_escalation` block.
  - _Requirements:_ [REQ-06.013](ep-requirements.md#nfr--security-testability-observability)
  - _Acceptance Criteria:_ [AC-06.010](ep-acceptance-criteria.md#ac-06-010)
  - **Verification:** `make check`; full test count unchanged or increased.

---

## Dependencies

- Task 2 depends on Task 1.
- Task 3 depends on Tasks 1–2.
- Task 4 can run in parallel with Task 1 (merge before Task 5).
- Task 5 depends on Tasks 3–4.
- Task 6 depends on Task 5.
- **Task 8** depends on Task 4 (`toolfailure` types stable); **should complete before or with** handler policy cleanup (replaces interim `wrapCatalogValidateError` in core).
- Task 7 depends on Tasks 5–6 (and benefits from Task 8 for policy coverage in tests).

---

## Traceability summary

| Theme | REQ (primary) | AC (primary) |
|-------|---------------|--------------|
| Config / baseline | 06.001, 06.002 | AC-06.001, AC-06.002 |
| Classification + typed errors | 06.003–06.005, 06.015 | AC-06.003, AC-06.004, AC-06.012 |
| Escalation / chain / Hermes | 06.006–06.008, 06.016 | AC-06.005–AC-06.007, AC-06.013 |
| Rollback end of turn | 06.009 | AC-06.008 |
| Observability | 06.010–06.012 | AC-06.009 |
| Tests / disabled mode | 06.013–06.014 | AC-06.010, AC-06.011 |
| Policy package / testability | 06.017 | AC-06.014 |
