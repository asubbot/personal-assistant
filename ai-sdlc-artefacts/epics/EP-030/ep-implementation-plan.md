# EP-030 — Implementation plan

**Purpose:** Ordered implementation for [ep-scope.md](ep-scope.md).  
**Inputs:** [ep-requirements.md](ep-requirements.md), [ep-acceptance-criteria.md](ep-acceptance-criteria.md), [ep-system-design.md](ep-system-design.md), [ep-system-design-review.md](ep-system-design-review.md) (iteration 2 gate pass).  
**Test strategy:** [strategy.md](../../strategy.md)

---

## Checkpoints

- After Task 1: `go test ./internal/config/...` passes.
- After Tasks 2–4: `go test ./internal/core/...` passes.
- Before final: `make check` and `make build && ./bin/validate EP-030` pass.

---

## Task list

- [ ] **1** — Config contract and strict removed keys  
  - Remove `TextBasedEnabled` from `ToolsConfig`; remove `SupportsJSONMode` from `LLMProvider`; require `default_response_format` == `text` only in `validateLLMProviderDefaults`.  
  - Add `rejectRemovedUnsupportedConfigKeys([]byte)` on raw JSON for `tools.text_based_enabled` and `llm_providers[].supports_json_mode`.  
  - Update all `internal/config/testdata` and `cmd/pa/main_test` fixtures.  
  - _Requirements:_ [REQ-30.004](ep-requirements.md#configuration), [REQ-30.005](ep-requirements.md#configuration), [REQ-30.006](ep-requirements.md#llm-defaults-and-request-shape), [REQ-30.013](ep-requirements.md#configuration)  
  - _Acceptance Criteria:_ [AC-30.004](ep-acceptance-criteria.md#ac-30-004), [AC-30.005](ep-acceptance-criteria.md#ac-30-005), [AC-30.006](ep-acceptance-criteria.md#ac-30-006)  
  - **Verification:** `go test ./internal/config/...`

- [ ] **2** — LLM client: text-only response format  
  - Remove `ForceJSONOutput` from `CompletionOptions`; drop `supportsJSONMode` from `OpenAICompatible`; simplify `resolveResponseFormat` per REQ-30.007.  
  - Update `internal/llm/openai_test.go` trace comments to AC-30.007.  
  - _Requirements:_ [REQ-30.007](ep-requirements.md#llm-defaults-and-request-shape)  
  - _Acceptance Criteria:_ [AC-30.007](ep-acceptance-criteria.md#ac-30-007)  
  - **Verification:** `go test ./internal/llm/...`

- [ ] **3** — Remove `internal/tooltext` and Hermes tail assembly  
  - Delete package `internal/tooltext`; remove Hermes blocks from `system_tail.go`, `handler_tier_main_prompt.go`, `systemprompt` wrappers if unused.  
  - Remove legacy catalog helpers and YAML fields for the removed text-tool path; native descriptions use `index_text` only.  
  - _Requirements:_ [REQ-30.001](ep-requirements.md#hermes-removal), [REQ-30.003](ep-requirements.md#hermes-removal), [REQ-30.011](ep-requirements.md#tool-catalog)  
  - _Acceptance Criteria:_ [AC-30.001](ep-acceptance-criteria.md#ac-30-001), [AC-30.003](ep-acceptance-criteria.md#ac-30-003), [AC-30.011](ep-acceptance-criteria.md#ac-30-011)  
  - **Verification:** `go test ./internal/toolcatalog/...` ; repo has no `internal/tooltext`

- [ ] **4** — Core handler: native tools only  
  - Remove `textBasedEnabled`, `textPath`, Hermes parse/follow-up, `hermes_parse` escalation class, `invoked_via=hermes`.  
  - Simplify `runToolResultLoop` / `finishAfterFirstLLM` / `appendToolRound` to native path only.  
  - Set `full_lite` dynamic tail merge to use the same dynamic-cap branch as `full` ([REQ-30.008](ep-requirements.md#dynamic-tool-capping)).  
  - _Requirements:_ [REQ-30.002](ep-requirements.md#hermes-removal), [REQ-30.008](ep-requirements.md#dynamic-tool-capping), [REQ-30.010](ep-requirements.md#observability), [REQ-30.016](ep-requirements.md#native-tool-contract)  
  - _Acceptance Criteria:_ [AC-30.002](ep-acceptance-criteria.md#ac-30-002), [AC-30.008](ep-acceptance-criteria.md#ac-30-008), [AC-30.010](ep-acceptance-criteria.md#ac-30-010)  
  - **Verification:** `go test ./internal/core/...`

- [ ] **5** — Startup WARN ([REQ-30.009](ep-requirements.md#operator-warning))  
  - Emit once from `cmd/pa` after catalog is available when baseline provider `supports_tools` is false and catalog has at least one tool.  
  - _Acceptance Criteria:_ [AC-30.009](ep-acceptance-criteria.md#ac-30-009)  
  - **Verification:** unit or integration test with `slog` capture

- [ ] **6** — Documentation and integration tests  
  - Update `docs/configuration.md`, `docs/README.md` if needed, architecture notes; remove Hermes references.  
  - Fix `tests/integration` and validate harness comments for EP-030.  
  - _Requirements:_ [REQ-30.012](ep-requirements.md#documentation)  
  - _Acceptance Criteria:_ [AC-30.012](ep-acceptance-criteria.md#ac-30-012)  
  - **Verification:** grep docs for Hermes / `text_based_enabled` / removed keys

- [ ] **7** — Final validation  
  - Ensure every `Test*` that remains documents `// Covers AC-30.NNN` where required for `./bin/validate EP-030`.  
  - _Acceptance Criteria:_ [AC-30.013](ep-acceptance-criteria.md#ac-30-013)  
  - **Verification:** `make check` and `./bin/validate EP-030`
