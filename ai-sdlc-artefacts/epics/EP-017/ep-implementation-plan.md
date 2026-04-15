# EP-017 — Implementation plan

**Pipeline:** Stage 8 ([pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md)).
**Previous:** [ep-scope.md](ep-scope.md) · [ep-requirements.md](ep-requirements.md) · [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-system-design.md](ep-system-design.md) · [ep-system-design-review.md](ep-system-design-review.md)
**Test strategy:** [strategy.md](../../strategy.md)

**AC ownership:** Every **AC-17.001**–**AC-17.018** MUST appear in at least one task verification line or in the validation row below. Mirror the [acceptance criteria index](ep-acceptance-criteria.md#acceptance-criteria-index) when marking tests done.

---

## Checkpoints

- [ ] **Checkpoint A:** After intent package, `go test ./internal/intent/...` passes.
- [ ] **Checkpoint B:** After config + wiring, `go test ./internal/config/... ./internal/core/...` passes.
- [ ] **Checkpoint C:** `make build && ./bin/validate EP-017 && make check` ([AC-17.018](ep-acceptance-criteria.md#ac-17-018)).

---

## Task list

- [ ] **1** — **`internal/intent`: Tier type and Classifier interface**
  - Define `Tier` (`simple`, `full`), `Result` struct (Tier, Stage, MessageLen), and `Classifier` interface with single `Classify(ctx, message) Result` method.
  - _Requirements:_ [REQ-17.001](ep-requirements.md#req-17-001)
  - _Acceptance Criteria:_ [AC-17.001](ep-acceptance-criteria.md#ac-17-001)
  - **Verification:** `go build ./internal/intent/...`

- [ ] **2** — **`internal/intent`: HeuristicClassifier**
  - Implement `HeuristicClassifier` with compiled regex patterns (`simplePatterns`, `fullPatterns`), `maxSimpleLen` threshold. `Classify(message)` returns `HeuristicResult{Tier, Confident}` per design logic: length check → simple patterns → full patterns → ambiguous.
  - _Requirements:_ [REQ-17.004](ep-requirements.md#req-17-004), [REQ-17.005](ep-requirements.md#req-17-005), [REQ-17.006](ep-requirements.md#req-17-006)
  - _Acceptance Criteria:_ [AC-17.004](ep-acceptance-criteria.md#ac-17-004), [AC-17.005](ep-acceptance-criteria.md#ac-17-005), [AC-17.006](ep-acceptance-criteria.md#ac-17-006), [AC-17.007](ep-acceptance-criteria.md#ac-17-007)
  - **Verification:** `go test ./internal/intent/...` — unit tests for greetings→simple, tool-intent→full, borderline→ambiguous, no I/O calls.

- [ ] **3** — **`internal/intent`: ModelClassifier**
  - Implement `ModelClassifier` with `llm.Provider`, model name, logger, timeout duration. `Classify(ctx, message)` applies context deadline, sends minimal classification prompt, parses response ("simple"/"full"/error). Logs token usage at INFO separately (`component=intent_classifier_model`).
  - _Requirements:_ [REQ-17.007](ep-requirements.md#req-17-007), [REQ-17.008](ep-requirements.md#req-17-008), [REQ-17.009](ep-requirements.md#req-17-009)
  - _Acceptance Criteria:_ [AC-17.008](ep-acceptance-criteria.md#ac-17-008), [AC-17.009](ep-acceptance-criteria.md#ac-17-009)
  - **Verification:** `go test ./internal/intent/...` — mock provider tests for "simple"/"full" responses, unparseable response, timeout.

- [ ] **4** — **`internal/intent`: CascadeClassifier**
  - Implement `CascadeClassifier` with optional heuristic + optional model. Cascade: heuristic confident→return; ambiguous+model→call model; model error→WARN log+default full; all nil→default full. Returns `Result` with deciding stage.
  - _Requirements:_ [REQ-17.010](ep-requirements.md#req-17-010), [REQ-17.011](ep-requirements.md#req-17-011)
  - _Acceptance Criteria:_ [AC-17.010](ep-acceptance-criteria.md#ac-17-010), [AC-17.011](ep-acceptance-criteria.md#ac-17-011)
  - **Verification:** `go test ./internal/intent/...` — cascade scenarios: heuristic-only, model-only, both, neither, model error.

> **Checkpoint A:** `go test ./internal/intent/...` — all unit tests pass.

- [ ] **5** — **Configuration: `IntentClassifierConfig` types and validation**
  - Add `IntentClassifierConfig`, `HeuristicConfig`, `ClassificationModelConfig` (with `Timeout` field) to `internal/config`. Add `IntentClassifier` field to `Config` struct. Validate at load: compile regexes (fail fast), check required fields when enabled, parse timeout duration. Extend `ResolvePaths` for `model_stage.api_key_path`.
  - _Requirements:_ [REQ-17.016](ep-requirements.md#req-17-016)
  - _Acceptance Criteria:_ [AC-17.014](ep-acceptance-criteria.md#ac-17-014), [AC-17.015](ep-acceptance-criteria.md#ac-17-015)
  - **Verification:** `go test ./internal/config/...` — valid config, invalid regex rejected, missing endpoint rejected, disabled config → nil.

- [ ] **6** — **Wiring: `cmd/pa` + `core.Run` + `conversationHandler`**
  - Build `intent.CascadeClassifier` in `cmd/pa` from config (instantiate classification provider via `llm.NewOpenAICompatible` when model stage enabled). Pass classifier into `core.Run` → `newRunConversationHandler`. Add `classifier` field to `conversationHandler`.
  - _Requirements:_ [REQ-17.008](ep-requirements.md#req-17-008), [REQ-17.016](ep-requirements.md#req-17-016)
  - _Acceptance Criteria:_ [AC-17.009](ep-acceptance-criteria.md#ac-17-009), [AC-17.014](ep-acceptance-criteria.md#ac-17-014)
  - **Verification:** `go build ./cmd/pa/...` passes; classifier is nil when config disabled.

- [ ] **7** — **`HandleMessage`: tier-based prompt assembly**
  - Insert classification call after `checkUserMessage`, before prompt construction. Gate `gatherRetrievedChunkTexts`, `selectSkillPackages`, `mergeSelectedToolIDs`, dynamic tail building, and `CompletionOptions.Tools` on `tier == TierFull`. Simple tier: `opts` is nil, no tools in request. Log classification result at INFO (tier, stage, message_len).
  - _Requirements:_ [REQ-17.002](ep-requirements.md#req-17-002), [REQ-17.003](ep-requirements.md#req-17-003), [REQ-17.012](ep-requirements.md#req-17-012), [REQ-17.013](ep-requirements.md#req-17-013), [REQ-17.014](ep-requirements.md#req-17-014), [REQ-17.015](ep-requirements.md#req-17-015), [REQ-17.017](ep-requirements.md#req-17-017)
  - _Acceptance Criteria:_ [AC-17.002](ep-acceptance-criteria.md#ac-17-002), [AC-17.003](ep-acceptance-criteria.md#ac-17-003), [AC-17.012](ep-acceptance-criteria.md#ac-17-012), [AC-17.013](ep-acceptance-criteria.md#ac-17-013), [AC-17.016](ep-acceptance-criteria.md#ac-17-016)
  - **Verification:** `go test ./internal/core/...` — integration tests: simple tier skips RAG/tools/tail (message count, no tools array); full tier byte-identical to pre-epic; classifier disabled → full path.

> **Checkpoint B:** `go test ./internal/config/... ./internal/core/... ./internal/intent/...` — all pass.

- [ ] **8** — **Observability: model-stage token logging exclusion from footer**
  - Verify that classification model usage is logged separately (from task 3) and that `usageTurnAcc` accumulation starts after classification (structural). Add integration test asserting footer shows only main-model tokens.
  - _Requirements:_ [REQ-17.018](ep-requirements.md#req-17-018)
  - _Acceptance Criteria:_ [AC-17.017](ep-acceptance-criteria.md#ac-17-017)
  - **Verification:** `go test ./internal/core/...` — test with mock classification + mock main provider; assert footer excludes classification tokens.

- [ ] **9** — **Documentation: update `docs/configuration.md`**
  - Add `intent_classifier` section to configuration docs with all fields, defaults, and examples.
  - _Requirements:_ [REQ-17.016](ep-requirements.md#req-17-016)
  - _Acceptance Criteria:_ —
  - **Verification:** Manual review; config section present.

- [ ] **10** — **AC coverage comments and final validation**
  - Add `// Covers AC-17.NNN` to every new test; ensure each AC has coverage. Run `make build && ./bin/validate EP-017 && make check`.
  - _Requirements:_ [REQ-17.019](ep-requirements.md#req-17-019), [REQ-17.020](ep-requirements.md#req-17-020)
  - _Acceptance Criteria:_ [AC-17.018](ep-acceptance-criteria.md#ac-17-018)
  - **Verification:** `make build && ./bin/validate EP-017 && make check` exits zero.

> **Checkpoint C:** `make build && ./bin/validate EP-017 && make check` passes.

---

## Dependencies

- Task **1** before **2**–**4** (types and interface).
- Tasks **2**–**4** before **6**–**7** (classifier implementations).
- Task **5** before **6** (config types for wiring).
- Task **6** before **7** (classifier injected into handler).
- Task **7** before **8** (observability tests need handler integration).
- **10** depends on all functional tasks.

---

## Notes

- **Stage 9** executes tasks **in numerical order** unless a dependency forces a wait; mark checkboxes in this file when each task completes.
- Do **not** commit without explicit user allowance ([AGENTS.md](../../../AGENTS.md)).
