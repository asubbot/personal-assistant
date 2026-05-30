---
artefact: ep-implementation-plan
epic_id: EP-036
status: draft
source_of_truth: true
updated_at: 2026-05-30
---

# EP-036 — Implementation plan

Pipeline stage 8 output for **EP-036**: simplify intent classification to a heuristic-only two-tier cascade (`simple`, `full`); remove the model stage and `full_lite` tier.

**Related artefacts**

- Scope: [ep-scope.md](ep-scope.md)
- Requirements: [ep-requirements.md](ep-requirements.md)
- Acceptance criteria: [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- System design: [ep-system-design.md](ep-system-design.md)
- Design review: [ep-system-design-review.md](ep-system-design-review.md) (gate **pass**, iteration 3)
- Test strategy: [strategy.md](../../strategy.md)

**Branch:** `epic/EP-036-simplify-intent-tiers` (implementation only; no git operations in this stage).

**AC trace convention:** Behavioural / rejection tests MUST include `// Covers AC-36.xxx` on the test function or table case. Structural or process gates marked **MANUAL ONLY** in [ep-acceptance-criteria.md](ep-acceptance-criteria.md) are satisfied by explicit plan notes (file absence, grep, doc read, `make check`, validate) — no unit test required for those ACs.

---

## Tasks

### Phase 1 — Config rejection (keep legacy structs until Phase 5)

- [ ] **1.1** Add `rejectRemovedIntentClassifierKeys(rawIC json.RawMessage) error` in `internal/config/load.go` and invoke it from `rejectRemovedUnsupportedConfigKeys` when root `intent_classifier` is present and not JSON `null` (EP-034 raw-map pattern: reject `model_stage` and `heuristic.full_lite_patterns` with explicit error messages).
  - _Requirements:_ [REQ-36.016](ep-requirements.md#req-36-016--reject-model_stage-config-key), [REQ-36.017](ep-requirements.md#req-36-017--reject-full_lite_patterns-config-key)
  - _Acceptance Criteria:_ [AC-36.013](ep-acceptance-criteria.md#ac-36-013), [AC-36.014](ep-acceptance-criteria.md#ac-36-014)
  - **Verification:** `go test ./internal/config/... -run RejectRemovedIntent` (or full package) passes after tests in 1.2–1.3 land.

- [ ] **1.2** Add rejection testdata fixtures under `internal/config/testdata/`: `intent_classifier_model_stage_rejected.json`, `intent_classifier_full_lite_patterns_rejected.json`. Extend `internal/config/intent_classifier_test.go` (or `config_test.go`) with `Load` tests that expect failure; add `// Covers AC-36.013` and `// Covers AC-36.014` on those tests.
  - _Requirements:_ [REQ-36.024](ep-requirements.md#req-36-024--reject-removed-keys-in-tests)
  - _Acceptance Criteria:_ [AC-36.013](ep-acceptance-criteria.md#ac-36-013), [AC-36.014](ep-acceptance-criteria.md#ac-36-014)
  - **Verification:** `go test ./internal/config/... -run 'ModelStage|FullLitePatterns|Reject'` passes; error strings name the removed keys.

- [ ] **1.3** Add positive fixture `internal/config/testdata/intent_classifier_enabled_heuristic_only.json` and a `Load` test with `// Covers AC-36.015` (enabled `heuristic` with `simple_patterns`, `full_patterns`, `max_simple_len` ≥ 1) plus invalid-regex / `max_simple_len` &lt; 1 negative cases in the same file or adjacent tests.
  - _Requirements:_ [REQ-36.018](ep-requirements.md#req-36-018--enabled-heuristic-schema), [REQ-36.019](ep-requirements.md#req-36-019--validate-heuristic-at-load)
  - _Acceptance Criteria:_ [AC-36.015](ep-acceptance-criteria.md#ac-36-015)
  - **Verification:** `go test ./internal/config/... -run IntentClassifier` passes.

- [ ] **1.4** Add automated positive-load coverage for `config.examples/config.example.json` and the new enabled-heuristic testdata (`// Covers AC-36.022`). Confirm root-key / `intent_classifier: null` tests still pass (`// Covers AC-36.016` where applicable).
  - _Requirements:_ [REQ-36.020](ep-requirements.md#req-36-020--keep-intent_classifier-root-key), [REQ-36.021](ep-requirements.md#req-36-021--null-intent_classifier-disables-classification), [REQ-36.022](ep-requirements.md#req-36-022--update-configs-and-operator-docs)
  - _Acceptance Criteria:_ [AC-36.016](ep-acceptance-criteria.md#ac-36-016), [AC-36.022](ep-acceptance-criteria.md#ac-36-022)
  - **Verification:** `go test ./internal/config/...` passes; `make check` still green (legacy structs still present).

- [ ] **1.5** **Checkpoint:** run `make check` after Phase 1; fix any compile issues before Phase 2.
  - _Requirements:_ [REQ-36.026](ep-requirements.md#req-36-026--make-check-passes)
  - _Acceptance Criteria:_ —
  - **Verification:** `make check` exit 0.

### Phase 2 — `internal/intent` (delete model stage; two tiers; heuristic-only cascade)

- [ ] **2.1** Delete `internal/intent/model.go` and `internal/intent/model_test.go`. **AC-36.008 MANUAL ONLY** — no unit test; plan satisfies AC via absent files + grep for `ModelClassifier` in product packages after Phase 7.
  - _Requirements:_ [REQ-36.008](ep-requirements.md#req-36-008--delete-model-stage-code), [REQ-36.009](ep-requirements.md#req-36-009--remove-modelclassifier-type)
  - _Acceptance Criteria:_ [AC-36.008](ep-acceptance-criteria.md#ac-36-008)
  - **Verification:** files absent; `go build ./internal/intent/...` succeeds once dependent edits in 2.2–2.5 complete.

- [ ] **2.2** `internal/intent/tier.go`: remove `TierFullLite`; keep only `TierSimple` and `TierFull`. Update `Result.Stage` comment to `heuristic` \| `default` and package doc (F-003).
  - _Requirements:_ [REQ-36.001](ep-requirements.md#req-36-001--two-complexity-tiers), [REQ-36.002](ep-requirements.md#req-36-002--remove-full_lite-tier), [REQ-36.011](ep-requirements.md#req-36-011--stage-values-heuristic-or-default)
  - _Acceptance Criteria:_ [AC-36.001](ep-acceptance-criteria.md#ac-36-001), [AC-36.002](ep-acceptance-criteria.md#ac-36-002)
  - **Verification:** unit test or compile-time constant check with `// Covers AC-36.001`; grep `TierFullLite` zero in `internal/intent`.

- [ ] **2.3** `internal/intent/heuristic.go`: remove `fullLitePatterns` field and evaluation step; change `NewHeuristicClassifier(simplePatterns, fullPatterns []string, maxSimpleLen int)`; update order comment to length → simple → full → ambiguous (F-003).
  - _Requirements:_ [REQ-36.004](ep-requirements.md#req-36-004--heuristic-evaluation-order), [REQ-36.005](ep-requirements.md#req-36-005--no-full_lite-patterns-in-heuristic)
  - _Acceptance Criteria:_ [AC-36.004](ep-acceptance-criteria.md#ac-36-004), [AC-36.005](ep-acceptance-criteria.md#ac-36-005)
  - **Verification:** `go test ./internal/intent/... -run Heuristic` with `// Covers AC-36.004` / `// Covers AC-36.005`; delete `TestHeuristic_FullLitePatterns`.

- [ ] **2.4** `internal/intent/cascade.go`: remove `CascadeClassifier.model` field and model branch; ambiguous heuristic → `TierFull`, `Stage: "default"` without LLM; confident → `Stage: "heuristic"`. `NewCascadeClassifier(heuristic *HeuristicClassifier, logger *slog.Logger)`.
  - _Requirements:_ [REQ-36.006](ep-requirements.md#req-36-006--ambiguous-defaults-to-full), [REQ-36.007](ep-requirements.md#req-36-007--confident-heuristic-stage-label), [REQ-36.011](ep-requirements.md#req-36-011--stage-values-heuristic-or-default), [REQ-36.023](ep-requirements.md#req-36-023--classification-and-config-load-tests)
  - _Acceptance Criteria:_ [AC-36.006](ep-acceptance-criteria.md#ac-36-006), [AC-36.007](ep-acceptance-criteria.md#ac-36-007)
  - **Verification:** `cascade_test.go` with `// Covers AC-36.006` / `// Covers AC-36.007`; delete model-stage tests (`TestCascade_AmbiguousToModel`, model-error cases, etc.).

- [ ] **2.5** Migrate `internal/intent/cascade_test.go`, `heuristic_test.go`, `observability_test.go`: all `NewHeuristicClassifier` → 3-arg; all `NewCascadeClassifier` → 2-arg; delete `TestModelClassifier_LogsUsageSeparately`; keep `TestCascadeClassifier_ResultContainsStageAndLen` with new signatures (F-005).
  - _Requirements:_ [REQ-36.025](ep-requirements.md#req-36-025--retire-obsolete-tier-tests)
  - _Acceptance Criteria:_ [AC-36.019](ep-acceptance-criteria.md#ac-36-019) (intent slice of inventory; **MANUAL ONLY** full inventory at Phase 7)
  - **Verification:** `go test ./internal/intent/...` passes; `make check` green.

### Phase 3 — `cmd/pa` wiring

- [ ] **3.1** `cmd/pa/main.go` — `buildIntentClassifier`: remove `ModelClassifier` / `llm.NewProvider` classification block, `model_stage` and `classifier_model` log attrs; wire `NewHeuristicClassifier(ic.Heuristic.SimplePatterns, ic.Heuristic.FullPatterns, ic.Heuristic.MaxSimpleLen)` and `NewCascadeClassifier(heuristic, logger)`. Drop unused imports from the removed block.
  - _Requirements:_ [REQ-36.010](ep-requirements.md#req-36-010--no-classification-llm-wiring)
  - _Acceptance Criteria:_ [AC-36.009](ep-acceptance-criteria.md#ac-36-009) (**MANUAL ONLY** — grep `cmd/pa` for classification LLM / `ModelClassifier`; behaviour of no extra LLM covered by AC-36.006 unit tests)
  - **Verification:** `go build -o /dev/null ./cmd/pa/...`; grep `ModelClassifier` / `model_stage` in `cmd/pa` → zero; `make check` green.

### Phase 4 — `internal/core` tier dispatch and tests

- [ ] **4.1** `internal/core/handler_tier_main_prompt.go`: remove `case intent.TierFullLite` and `buildTierFullLiteMainPrompt`; `assembleTierMainLLMParams` dispatches only `TierSimple` and `TierFull` (default → simple builder unchanged).
  - _Requirements:_ [REQ-36.012](ep-requirements.md#req-36-012--dispatch-simple-and-full-only), [REQ-36.013](ep-requirements.md#req-36-013--remove-full_lite-prompt-builder)
  - _Acceptance Criteria:_ [AC-36.010](ep-acceptance-criteria.md#ac-36-010)
  - **Verification:** package compiles; grep `buildTierFullLiteMainPrompt` / `TierFullLite` in `internal/core` → zero in production files.

- [ ] **4.2** `internal/core/handler_tier_main_prompt_test.go`: delete `TierFullLite` / `buildTierFullLiteMainPrompt` cases; retain `simple` and `full` dispatch tests with `// Covers AC-36.010`.
  - _Acceptance Criteria:_ [AC-36.010](ep-acceptance-criteria.md#ac-36-010)
  - **Verification:** `go test ./internal/core/... -run TierMainPrompt` passes.

- [ ] **4.3** `internal/core/handler_ep017_test.go`: migrate all `NewHeuristicClassifier` / `NewCascadeClassifier` calls to new signatures (no `full_lite` logic changes).
  - _Requirements:_ [REQ-36.025](ep-requirements.md#req-36-025--retire-obsolete-tier-tests)
  - _Acceptance Criteria:_ —
  - **Verification:** `go test ./internal/core/... -run EP017` passes.

- [ ] **4.4** `internal/core/handler_ep018_test.go`: delete `TestHandleMessage_FullLite_*` token-delta tests; migrate constructor signatures on retained tests; add integration-style test: pre-epic `full_lite` fixture messages → `full` assembly path with `// Covers AC-36.012`.
  - _Requirements:_ [REQ-36.014](ep-requirements.md#req-36-014--parity-for-simple-and-full-assembly), [REQ-36.015](ep-requirements.md#req-36-015--former-full_lite-uses-full-path)
  - _Acceptance Criteria:_ [AC-36.011](ep-acceptance-criteria.md#ac-36-011), [AC-36.012](ep-acceptance-criteria.md#ac-36-012)
  - **Verification:** `go test ./internal/core/... -run EP018` passes; simple/full baselines unchanged.

- [ ] **4.5** `internal/core/handler_ep018_coverage_test.go`: migrate `NewHeuristicClassifier` / `NewCascadeClassifier` at all constructor sites; delete or re-target `^LITESESS`, `^LITEHERM`, `^LITENOTOOL`, `^LITEDYN` cases to exercise the `full` path via `TierFull`; migrate `^FULLTOOLS` case. **Do not** rewrite `TestEP018_configurationDoc_containsTierMatrix` here (Phase 6 with docs).
  - _Requirements:_ [REQ-36.003](ep-requirements.md#req-36-003--one-tier-per-turn-when-enabled), [REQ-36.025](ep-requirements.md#req-36-025--retire-obsolete-tier-tests)
  - _Acceptance Criteria:_ [AC-36.003](ep-acceptance-criteria.md#ac-36-003) (`// Covers AC-36.003` on handler tier assignment test if not already covered)
  - **Verification:** `go test ./internal/core/... -run EP018` passes; `make check` green.

- [ ] **4.6** **Checkpoint:** `make check` after Phase 4.
  - _Acceptance Criteria:_ —
  - **Verification:** `make check` exit 0.

### Phase 5 — Config struct shrink and operator config files

- [ ] **5.1** `internal/config/config.go`: delete `ClassificationModelConfig`, `IntentClassifierConfig.ModelStage`, `HeuristicConfig.FullLitePatterns`. `internal/config/load.go`: delete `validateICModelStage` and `full_lite_patterns` validation loop. `internal/config/resolve.go`: remove `ModelStage.APIKeyPath` resolution block.
  - _Requirements:_ [REQ-36.016](ep-requirements.md#req-36-017--reject-full_lite_patterns-config-key) through [REQ-36.019](ep-requirements.md#req-36-019--validate-heuristic-at-load) (struct alignment with rejection from Phase 1)
  - _Acceptance Criteria:_ [AC-36.013](ep-acceptance-criteria.md#ac-36-013), [AC-36.014](ep-acceptance-criteria.md#ac-36-014), [AC-36.015](ep-acceptance-criteria.md#ac-36-015)
  - **Verification:** `go test ./internal/config/...` passes (rejection + positive tests from Phase 1 still pass).

- [ ] **5.2** `internal/config/intent_classifier_test.go`: delete all `TestValidateIntentClassifier_ModelStage*` tests; ensure no references to removed types remain.
  - _Requirements:_ [REQ-36.025](ep-requirements.md#req-36-025--retire-obsolete-tier-tests)
  - _Acceptance Criteria:_ [AC-36.019](ep-acceptance-criteria.md#ac-36-019) (config slice)
  - **Verification:** `go test ./internal/config/...` passes.

- [ ] **5.3** Update `.config/config.json`: remove `model_stage` object and `heuristic.full_lite_patterns`; merge former lite regexes into `full_patterns` per operator choice; keep `enabled: true` and heuristic-only shape. **AC-36.018 MANUAL ONLY** — no automated test loads this file; verify by starting the app or manual `Load` with full secret paths.
  - _Requirements:_ [REQ-36.022](ep-requirements.md#req-36-022--update-configs-and-operator-docs)
  - _Acceptance Criteria:_ [AC-36.018](ep-acceptance-criteria.md#ac-36-018)
  - **Verification:** manual — process starts / config validates without removed keys.

- [ ] **5.4** Confirm `config.examples/config.example.json` remains `intent_classifier: null` (explicit key); scan `internal/config/testdata/*.json` and `tests/integration/testdata/runtime_skills/minimal_ok/config.json` — no removed keys (already `null` per design). Update `internal/config/config.go` `ToolDynamicSelection` doc comment: drop `TierFullLite` reference.
  - _Requirements:_ [REQ-36.022](ep-requirements.md#req-36-022--update-configs-and-operator-docs)
  - _Acceptance Criteria:_ [AC-36.022](ep-acceptance-criteria.md#ac-36-022)
  - **Verification:** Phase 1.4 tests still pass; `make check` green.

### Phase 6 — Documentation and doc-content tests (atomic commit)

- [ ] **6.1** Update `docs/configuration.md`: two-tier (`simple`, `full`) heuristic-only cascade; remove `model_stage`, `full_lite_patterns`, and three-tier matrix prose.
  - _Requirements:_ [REQ-36.022](ep-requirements.md#req-36-022--update-configs-and-operator-docs)
  - _Acceptance Criteria:_ [AC-36.017](ep-acceptance-criteria.md#ac-36-017) (**MANUAL ONLY** — read docs for two-tier heuristic-only prose; no model-stage setup)
  - **Verification:** manual read + grep `full_lite` / `model_stage` absent from `docs/configuration.md`.

- [ ] **6.2** Update `docs/llm-provider-roles-and-logging.md`: remove intent classifier model-stage section and example that references `model_stage`.
  - _Requirements:_ [REQ-36.022](ep-requirements.md#req-36-022--update-configs-and-operator-docs)
  - _Acceptance Criteria:_ [AC-36.017](ep-acceptance-criteria.md#ac-36-017)
  - **Verification:** manual read; grep removed terms absent.

- [ ] **6.3** Update `docs/architecture-ru.md` stale lines (~162, 230, 260, 519): two-tier `simple` / `full` heuristic-only (F-004).
  - _Requirements:_ [REQ-36.022](ep-requirements.md#req-36-022--update-configs-and-operator-docs)
  - _Acceptance Criteria:_ [AC-36.017](ep-acceptance-criteria.md#ac-36-017)
  - **Verification:** grep `full_lite` absent from `docs/architecture-ru.md`.

- [ ] **6.4** Rewrite `TestEP018_configurationDoc_containsTierMatrix` in `internal/core/handler_ep018_coverage_test.go`: assert `### Intent tiers`, `simple`, `full`, `dynamic_selection`; **must not** assert `full_lite`.
  - _Requirements:_ [REQ-36.022](ep-requirements.md#req-36-022--update-configs-and-operator-docs)
  - _Acceptance Criteria:_ [AC-36.017](ep-acceptance-criteria.md#ac-36-017) (automated doc-content half)
  - **Verification:** `go test ./internal/core/... -run configurationDoc` passes in same commit as 6.1.

- [ ] **6.5** Rewrite `TestEP024_ProviderRolesDocContent` in `cmd/pa/ep024_operator_logging_test.go`: remove `checks` entries for `"model_stage"`, `"not selected by an index"`, and `"## Example: pool with intent classifier"`; keep remaining provider-role assertions (F-006).
  - _Requirements:_ [REQ-36.022](ep-requirements.md#req-36-022--update-configs-and-operator-docs)
  - _Acceptance Criteria:_ [AC-36.017](ep-acceptance-criteria.md#ac-36-017)
  - **Verification:** `go test ./cmd/pa/... -run EP024_ProviderRoles` passes in same commit as 6.2.

- [ ] **6.6** **Checkpoint:** `make check` after Phase 6 (docs + doc-content tests land together).
  - _Acceptance Criteria:_ —
  - **Verification:** `make check` exit 0.

### Phase 7 — Quality gates and manual AC closure

- [ ] **7.1** Run `make check` from repository root. **AC-36.020 MANUAL ONLY**.
  - _Requirements:_ [REQ-36.026](ep-requirements.md#req-36-026--make-check-passes)
  - _Acceptance Criteria:_ [AC-36.020](ep-acceptance-criteria.md#ac-36-020)
  - **Verification:** `make check` exit 0.

- [ ] **7.2** Build validator and run `./bin/validate ears EP-036`. **AC-36.021 MANUAL ONLY**.
  - _Requirements:_ [REQ-36.027](ep-requirements.md#req-36-027--epic-validation-passes)
  - _Acceptance Criteria:_ [AC-36.021](ep-acceptance-criteria.md#ac-36-021)
  - **Verification:** `./bin/validate ears EP-036` exit 0 (after `make build` if needed).

- [ ] **7.3** Residual-symbol grep across product code (`cmd/`, `internal/`): zero matches for `full_lite`, `model_stage`, `ModelClassifier`, `TierFullLite` in tier dispatch (optional hygiene: `internal/telegram/outbound_chunk_test.go`, `internal/llm/openai_test.go` literal samples — out of scope unless desired).
  - _Acceptance Criteria:_ [AC-36.002](ep-acceptance-criteria.md#ac-36-002), [AC-36.008](ep-acceptance-criteria.md#ac-36-008), [AC-36.009](ep-acceptance-criteria.md#ac-36-009)
  - **Verification:** `rg -l 'full_lite|model_stage|ModelClassifier|TierFullLite' cmd internal` → no production matches (or documented exceptions only).

- [ ] **7.4** **Manual AC checklist** (record in PR / epic notes): AC-36.008 (`model.go` absent, no `ModelClassifier`); AC-36.009 (`cmd/pa` grep); AC-36.017 (docs read); AC-36.018 (`.config/config.json` load); AC-36.019 (test inventory — no model-stage-only or `full_lite` token-delta-only tests remain).
  - _Acceptance Criteria:_ [AC-36.008](ep-acceptance-criteria.md#ac-36-008), [AC-36.009](ep-acceptance-criteria.md#ac-36-009), [AC-36.017](ep-acceptance-criteria.md#ac-36-017), [AC-36.018](ep-acceptance-criteria.md#ac-36-018), [AC-36.019](ep-acceptance-criteria.md#ac-36-019)
  - **Verification:** operator sign-off; all 21 ACs mapped to automated tests or MANUAL ONLY notes above.

---

## Dependencies and order

| Phase | Depends on |
|-------|------------|
| 1 | — (rejection can land while legacy config structs remain) |
| 2 | 1 checkpoint (recommended) |
| 3 | 2 |
| 4 | 2 (needs new `intent` constructors and no `TierFullLite`) |
| 5 | 1 (rejection already wired), 2–4 (call sites use shrunk API) |
| 6 | 5 recommended (config structs match docs); **must** ship doc-content tests (6.4, 6.5) in the **same commit** as their doc files (6.1–6.3) |
| 7 | 1–6 |

Phases 2–4 may be one commit if `make check` stays green. Phase 6 is a single atomic commit for docs + `TestEP018_configurationDoc_containsTierMatrix` + `TestEP024_ProviderRolesDocContent`.

---

## Checkpoints

- After **1.5**: rejection and positive config loads proven; tree still compiles with legacy struct fields.
- After **4.6**: intent + core + `cmd/pa` compile; no `TierFullLite` in product code; doc-content tests may still expect old doc strings until Phase 6.
- After **6.6**: operator docs and doc-content tests aligned.
- After **7.4**: epic ready for stage 10 (implementation); all ACs covered by tests or MANUAL ONLY verification.

---

## AC coverage summary

| AC | How the plan closes it |
|----|-------------------------|
| AC-36.001–007 | Phase 2 unit tests (`// Covers AC-36.00x`) |
| AC-36.008–009 | Phase 2.1, 3.1, 7.3–7.4 **MANUAL ONLY** |
| AC-36.010–012 | Phase 4 tests |
| AC-36.013–016, AC-36.022 | Phase 1, 5 automated config tests |
| AC-36.017 | Phase 6 docs + doc-content tests; 7.4 **MANUAL ONLY** read |
| AC-36.018 | Phase 5.3 **MANUAL ONLY** (live `.config/config.json`) |
| AC-36.019 | Phases 2.5, 4, 5.2, 7.4 **MANUAL ONLY** inventory |
| AC-36.020–021 | Phase 7.1–7.2 **MANUAL ONLY** |
