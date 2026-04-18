# EP-026 — Implementation plan

**Purpose:** Execute [pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md) stage 9 from this ordered task list.

**Previous / related:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-requirements.md](ep-requirements.md) · [ep-system-design.md](ep-system-design.md) · [ep-system-design-review.md](ep-system-design-review.md) · [strategy.md](../../strategy.md)

**Checkpoints:** Run `make check` and `make build && ./bin/validate EP-026` before declaring the epic complete.

---

## Task list

- [x] **1** — Add `handler_tier_main_prompt.go`: `tierMainLLMParams`, `assembleTierMainLLMParams`, `buildTier{Simple,Full,FullLite}MainPrompt`, `mergeTailMergedToolsAndOptions`, and small helpers (`mergedAfterDynamicToolCap`, `includeHermesForMainTail`, `copyToolOriginMap`, `buildMainTurnMessagesPreTail`).
  - _Requirements:_ [REQ-26.001](ep-requirements.md#tier-builders), [REQ-26.002](ep-requirements.md#orchestrator), [REQ-26.005](ep-requirements.md#parity)
  - _Acceptance Criteria:_ [AC-26.001](ep-acceptance-criteria.md#ac-26-001), [AC-26.002](ep-acceptance-criteria.md#ac-26-002)
  - **Verification:** `go test -tags=integration -count=1 ./internal/core/... -run TierMainPrompt`

- [x] **2** — Refactor `HandleMessage` in `handler.go` to call `buildMainTurnMessagesPreTail` then `assembleTierMainLLMParams`; remove `//nolint:gocyclo` from `HandleMessage`.
  - _Requirements:_ [REQ-26.002](ep-requirements.md#orchestrator), [REQ-26.004](ep-requirements.md#lint)
  - _Acceptance Criteria:_ [AC-26.002](ep-acceptance-criteria.md#ac-26-002), [AC-26.004](ep-acceptance-criteria.md#ac-26-004)
  - **Verification:** `make lint`; `TestEP026_HandlerGoHasNoGocycloNolint`

- [x] **3** — Add `handler_tier_main_prompt_test.go` with tier builder and policy tests; bind `Covers AC-26.*` per repository convention.
  - _Requirements:_ [REQ-26.003](ep-requirements.md#tests)
  - _Acceptance Criteria:_ [AC-26.001](ep-acceptance-criteria.md#ac-26-001)–[AC-26.004](ep-acceptance-criteria.md#ac-26-004), supporting [AC-26.005](ep-acceptance-criteria.md#ac-26-005), [AC-26.006](ep-acceptance-criteria.md#ac-26-006)
  - **Verification:** `./bin/validate EP-026`

- [x] **4** — Checkpoint: `make check` and `make build && ./bin/validate EP-026`.
  - _Requirements:_ [REQ-26.006](ep-requirements.md#verification)
  - _Acceptance Criteria:_ [AC-26.005](ep-acceptance-criteria.md#ac-26-005), [AC-26.006](ep-acceptance-criteria.md#ac-26-006)
  - **Verification:** Exit code 0.

---

## Traceability note

Stages 10–11 follow this plan on the epic branch.
