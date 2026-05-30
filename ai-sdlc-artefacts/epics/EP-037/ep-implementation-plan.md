---
artefact: ep-implementation-plan
epic_id: EP-037
status: draft
source_of_truth: true
updated_at: 2026-05-30
---

# EP-037 — Implementation plan

Pipeline stage 8 output for **EP-037 Consolidate tool pre-selection configuration**.  
Purpose: ordered coding tasks to merge `tool_pre_selection` and `tools.dynamic_selection` into required `tools.selection` without changing runtime selection outcomes for equivalent settings.

**Related artefacts**

- Scope: [ep-scope.md](ep-scope.md)
- Requirements: [ep-requirements.md](ep-requirements.md)
- Acceptance criteria: [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- System design: [ep-system-design.md](ep-system-design.md)
- Design review: [ep-system-design-review.md](ep-system-design-review.md) (gate **pass**, iteration 2)

**Branch:** `epic/EP-037-consolidate-tool-selection` (no git operations from this stage).

**Out of scope (do not implement):** `tools.vector_search_tools` JSON DRY ([REQ-37.022](ep-requirements.md#req-37-022--defer-vector_search_tools-dry)); broad `internal/core` handler decomposition ([REQ-37.023](ep-requirements.md#req-37-023--limit-core-handler-changes)); new selection features ([REQ-37.024](ep-requirements.md#req-37-024--no-new-selection-features)).

---

## Tasks

### Phase 1 — `internal/config` schema and validation (keep legacy fields until Phase 4)

- [ ] **1.1** Add `ToolsSelection` struct and `ToolsConfig.Selection *ToolsSelection` in `internal/config/config.go`; keep `Config.ToolPreSelection`, `ToolsConfig.DynamicSelection`, and types `ToolPreSelection` / `ToolDynamicSelection` temporarily so existing readers and fixtures still compile.
  - Five explicit JSON fields: `tool_search_top_k`, `tool_min_count`, `tool_fallback_cap`, `enabled`, `max_tools_for_llm_request`.
  - _Requirements:_ [REQ-37.001](ep-requirements.md#req-37-001--require-toolsselection-block)
  - _Acceptance Criteria:_ [AC-37.001](ep-acceptance-criteria.md#ac-37-001)
  - **Verification:** `go build ./internal/config/...` succeeds.

- [ ] **1.2** Add `validateToolsSelectionBounds` (site A — pre-catalog, replaces `validateToolPreSelection` call in `validateMandatoryJSONSectionsCore`) and `validateToolsSelectionAlwaysIncludeFloor` (site B — post-`toolcatalog.Load`, replaces `validateToolDynamicSelection` at today’s ~line 86 site in `prepareConfig`). Wire both call sites in `internal/config/load.go`; **do not** merge floor check into site A (catalog nil → count 0 skips floor per design F-001).
  - Site A: required `tools.selection`; bounds on top-K / min / fallback; `enabled==true` ⇒ `max_tools_for_llm_request >= 1`; `enabled==false` ⇒ max may be 0.
  - Site B: `enabled==true` ⇒ `max_tools_for_llm_request >= countValidAlwaysIncludeTools(c)` (needs populated `c.ToolCatalog`).
  - Keep existing `validateToolPreSelection` / `validateToolDynamicSelection` calls until task **4.1** (avoid double-fail during transition).
  - _Requirements:_ [REQ-37.001](ep-requirements.md#req-37-001--require-toolsselection-block), [REQ-37.002](ep-requirements.md#req-37-002--validate-pre-selection-bounds), [REQ-37.003](ep-requirements.md#req-37-003--validate-dynamic-cap-when-enabled), [REQ-37.004](ep-requirements.md#req-37-004--allow-zero-max-when-disabled)
  - _Acceptance Criteria:_ [AC-37.001](ep-acceptance-criteria.md#ac-37-001), [AC-37.002](ep-acceptance-criteria.md#ac-37-002), [AC-37.003](ep-acceptance-criteria.md#ac-37-003), [AC-37.004](ep-acceptance-criteria.md#ac-37-004)
  - **Verification:** `go test ./internal/config/... -run 'ToolsSelection|tools_selection' -count=1` passes (new tests from task 1.3).

- [ ] **1.3** Add `internal/config/tools_selection_test.go` (port cases from `ep018_dynamic_tools_test.go`); add focused load tests with embedded/minimal JSON fixtures for required block and bounds. Mark each test with `// Covers AC-37.00x` per covered AC.
  - Suggested cases: missing `tools.selection`; out-of-range pre-selection ints; enabled + max &lt; 1; enabled + max &lt; always_include count (post-catalog fixture); disabled + max 0 succeeds.
  - Rename or add testdata: `tools_selection_missing.json`, `tools_selection_enabled_max_zero.json` (from `tools_dynamic_selection_enabled_max_zero.json` when migrated in task 5.1).
  - _Requirements:_ [REQ-37.001](ep-requirements.md#req-37-001--require-toolsselection-block)–[REQ-37.004](ep-requirements.md#req-37-004--allow-zero-max-when-disabled)
  - _Acceptance Criteria:_ [AC-37.001](ep-acceptance-criteria.md#ac-37-001)–[AC-37.004](ep-acceptance-criteria.md#ac-37-004)
  - **Verification:** `go test ./internal/config/... -run 'ToolsSelection|tools_selection|ToolsSelection' -count=1`; grep `// Covers AC-37.00[1-4]` in new test file.

**Checkpoint (Phase 1):** New selection validation tests pass; full `make check` may still fail until JSON migration (task 5.1) — expected.

---

### Phase 2 — Legacy key rejection and strict `tools` whitelist

- [ ] **2.1** Extend raw-JSON rejection (EP-034/EP-036 pattern) in `internal/config/load.go`: root `tool_pre_selection` in `rejectRemovedUnsupportedConfigKeys`; `tools.dynamic_selection` in `rejectRemovedToolsConfigKeys` (before whitelist). Add `validateToolsObjectKeys` with `allowedToolsKeys` = `always_include`, `selection`, `vector_search_tools`, `create_tool_secret_patterns`, `tool_output_artifacts` (parsed-but-ignored; no typed field in EP-037).
  - _Requirements:_ [REQ-37.005](ep-requirements.md#req-37-005--reject-tool_pre_selection-root-key), [REQ-37.006](ep-requirements.md#req-37-006--reject-toolsdynamic_selection-key), [REQ-37.008](ep-requirements.md#req-37-008--reject-unknown-tools-nested-keys)
  - _Acceptance Criteria:_ [AC-37.005](ep-acceptance-criteria.md#ac-37-005), [AC-37.006](ep-acceptance-criteria.md#ac-37-006), [AC-37.008](ep-acceptance-criteria.md#ac-37-008)
  - **Verification:** `go test ./internal/config/... -run 'ToolPreSelectionRejected|ToolsDynamicSelectionRejected|ToolsUnknownNestedKey' -count=1` passes.

- [ ] **2.2** Remove `"tool_pre_selection"` from `configRootJSONKeys` in `internal/config/root_keys.go`.
  - _Requirements:_ [REQ-37.007](ep-requirements.md#req-37-007--omit-tool_pre_selection-from-root-keys)
  - _Acceptance Criteria:_ [AC-37.007](ep-acceptance-criteria.md#ac-37-007)
  - **Verification:** `go test ./internal/config/... -run 'ConfigRootJSONKeys' -count=1` passes; test asserts `tool_pre_selection` not in list (`// Covers AC-37.007`).

- [ ] **2.3** Add rejection/negative testdata under `internal/config/testdata/`: `tool_pre_selection_rejected.json`, `tools_dynamic_selection_rejected.json`, `tools_unknown_nested_key.json` (e.g. `tools.legacy_selection_stub`). Unit tests with `// Covers AC-37.005`, `// Covers AC-37.006`, `// Covers AC-37.008`.
  - _Requirements:_ [REQ-37.016](ep-requirements.md#req-37-016--tests-reject-legacy-keys)
  - _Acceptance Criteria:_ [AC-37.005](ep-acceptance-criteria.md#ac-37-005), [AC-37.006](ep-acceptance-criteria.md#ac-37-006), [AC-37.008](ep-acceptance-criteria.md#ac-37-008)
  - **Verification:** `go test ./internal/config/... -run 'Rejected|UnknownNested' -count=1` passes.

**Checkpoint (Phase 2):** Legacy keys rejected at load; unknown `tools.*` keys fail; root key list updated.

---

### Phase 3 — `internal/core` wiring (read `tools.selection` only)

- [ ] **3.1** Wire `internal/core/run.go`: read `cfg.Tools.Selection` for `toolSearchTopK` / `toolMinCount` / `toolFallbackCap`; pass `toolsSelection: sel` into handler; remove use of `toolPreSelectionParams` / `cfg.ToolPreSelection` / `tc.DynamicSelection` at construction site (helper may remain until task 4.1).
  - _Requirements:_ [REQ-37.012](ep-requirements.md#req-37-012--wire-handler-from-toolsselection)
  - _Acceptance Criteria:_ [AC-37.012](ep-acceptance-criteria.md#ac-37-012)
  - **Verification:** `go build ./internal/core/...` succeeds.

- [ ] **3.2** Update `internal/core/handler.go` and `internal/core/handler_tier_main_prompt.go`: replace `toolsDynamic *config.ToolDynamicSelection` with `toolsSelection *config.ToolsSelection`; `mergedAfterDynamicToolCap` uses `toolsSelection == nil || !toolsSelection.Enabled` gate and `MaxToolsForLLMRequest` when enabled (algorithms unchanged).
  - _Requirements:_ [REQ-37.011](ep-requirements.md#req-37-011--preserve-ep-018-dynamic-cap), [REQ-37.012](ep-requirements.md#req-37-012--wire-handler-from-toolsselection), [REQ-37.023](ep-requirements.md#req-37-023--limit-core-handler-changes)
  - _Acceptance Criteria:_ [AC-37.011](ep-acceptance-criteria.md#ac-37-011), [AC-37.012](ep-acceptance-criteria.md#ac-37-012)
  - **Verification:** `go test ./internal/core/... -run 'Ep018|Dynamic|mergedAfter' -count=1` passes after task 3.4.

- [ ] **3.3** Update `internal/core/integration_export.go`: add `ToolsSelection *config.ToolsSelection` to `IntegrationConversationParams`; assign `toolsSelection: p.ToolsSelection` in `NewIntegrationConversationHandler` (additive — cap was not wired before; enables cap-parity tests).
  - _Requirements:_ [REQ-37.012](ep-requirements.md#req-37-012--wire-handler-from-toolsselection)
  - _Acceptance Criteria:_ [AC-37.011](ep-acceptance-criteria.md#ac-37-011), [AC-37.012](ep-acceptance-criteria.md#ac-37-012)
  - **Verification:** `go build ./internal/core/...` succeeds.

- [ ] **3.4** Update core tests: `run_test.go` (`Tools.Selection` on `&config.Config{}`), `handler_test.go`, `handler_ep018_coverage_test.go` (handler literals + doc needles `dynamic_selection` → `tools.selection` where applicable).
  - _Requirements:_ [REQ-37.012](ep-requirements.md#req-37-012--wire-handler-from-toolsselection)
  - _Acceptance Criteria:_ [AC-37.012](ep-acceptance-criteria.md#ac-37-012)
  - **Verification:** `go test ./internal/core/... -count=1` passes (parity tables may be added in task 6.1).

**Checkpoint (Phase 3):** Core compiles and unit tests pass with in-memory `ToolsSelection`; repository JSON may still use legacy keys until Phase 4.

---

### Phase 4 — Remove legacy config types and validators

- [ ] **4.1** Remove `ToolPreSelection`, `ToolDynamicSelection`, `Config.ToolPreSelection`, `ToolsConfig.DynamicSelection`; delete `validateToolPreSelection`, `validateToolDynamicSelection`, and `toolPreSelectionParams`; remove duplicate validation calls from `load.go` (only `validateToolsSelectionBounds` + `validateToolsSelectionAlwaysIncludeFloor` remain). Delete or fully supersede `ep018_dynamic_tools_test.go` if all cases live in `tools_selection_test.go`.
  - _Requirements:_ [REQ-37.001](ep-requirements.md#req-37-001--require-toolsselection-block)–[REQ-37.004](ep-requirements.md#req-37-004--allow-zero-max-when-disabled)
  - _Acceptance Criteria:_ [AC-37.012](ep-acceptance-criteria.md#ac-37-012)
  - **Verification:** `go test ./internal/config/... ./internal/core/... -count=1` passes; `grep -r 'ToolPreSelection\|ToolDynamicSelection\|validateToolPreSelection\|validateToolDynamicSelection\|toolPreSelectionParams' internal/ cmd/ tests/` returns no matches.

**Checkpoint (Phase 4):** Product Go has no legacy selection types; grep clean for symbols above.

---

### Phase 5 — Repository config migration (atomic per commit step; keep `make check` green)

Operator migration (1:1): `tool_pre_selection.*` → `tools.selection.*`; `tools.dynamic_selection.*` → `tools.selection.enabled` / `max_tools_for_llm_request`.

- [ ] **5.1** Migrate operator and example configs: `.config/config.json` (move `tool_pre_selection` + `tools.dynamic_selection` → `tools.selection`; keep `tools.tool_output_artifacts`), `config.examples/config.example.json`.
  - _Requirements:_ [REQ-37.013](ep-requirements.md#req-37-013--update-repository-configs), [REQ-37.020](ep-requirements.md#req-37-020--preserve-explicit-json-rules)
  - _Acceptance Criteria:_ [AC-37.013](ep-acceptance-criteria.md#ac-37-013), [AC-37.018](ep-acceptance-criteria.md#ac-37-018), [AC-37.023](ep-acceptance-criteria.md#ac-37-023) (operator config — **MANUAL ONLY** for live `.config/config.json` load at runtime; automated load covered by examples/testdata)
  - **Verification:** `go test ./internal/config/... -run 'Load.*example|config.example' -count=1` if present; else load example path in a one-off test or `go test ./internal/config/... -run TestLoad -count=1` after task 5.2.

- [ ] **5.2** Migrate **all 62** `internal/config/testdata/*.json` files that contain `tool_pre_selection` (exhaustive list in [ep-system-design.md](ep-system-design.md#test-inventory-grep-baseline)): move pre-selection fields and `dynamic_selection` (where present) into `tools.selection`; remove legacy keys. Update `tool_pre_selection_zero.json` → selection-focused naming if still needed for bounds tests.
  - Baseline before edit: `grep -rl 'tool_pre_selection' internal/config/testdata/ | wc -l` → **62**.
  - After edit: same grep → **0**; `grep -rl 'dynamic_selection' internal/config/testdata/` → **0** (except dedicated rejection fixtures from task 2.3).
  - _Requirements:_ [REQ-37.013](ep-requirements.md#req-37-013--update-repository-configs)
  - _Acceptance Criteria:_ [AC-37.013](ep-acceptance-criteria.md#ac-37-013)
  - **Verification:** `go test ./internal/config/... -run 'TestLoad_AllFixturesLoad|AllFixtures' -count=1` passes (`// Covers AC-37.013`).

- [ ] **5.3** Migrate `tests/integration/testdata/runtime_skills/minimal_ok/config.json` and `tests/integration/config_helpers.go` (`ensureCoreRunConfigRequiredSections`: set `cfg.Tools.Selection` instead of `cfg.ToolPreSelection`).
  - _Requirements:_ [REQ-37.013](ep-requirements.md#req-37-013--update-repository-configs)
  - _Acceptance Criteria:_ [AC-37.013](ep-acceptance-criteria.md#ac-37-013)
  - **Verification:** `go test ./tests/integration/... -run 'Config|RuntimeSkills' -count=1` passes (or full integration package if fast).

- [ ] **5.4** Update inline JSON strings in `internal/config/config_test.go`, `internal/config/intent_classifier_test.go`, `internal/config/vector_search_tools_test.go`, and `cmd/pa/main_test.go` to use `tools.selection` and no legacy keys.
  - _Requirements:_ [REQ-37.013](ep-requirements.md#req-37-013--update-repository-configs), [REQ-37.020](ep-requirements.md#req-37-020--preserve-explicit-json-rules)
  - _Acceptance Criteria:_ [AC-37.013](ep-acceptance-criteria.md#ac-37-013), [AC-37.018](ep-acceptance-criteria.md#ac-37-018)
  - **Verification:** `go test ./internal/config/... ./cmd/pa/... -count=1` passes.

**Checkpoint (Phase 5):** `grep -rl 'tool_pre_selection' internal/config/testdata/ config.examples/ tests/ .config/` returns only rejection fixtures (if any under testdata); `make check` should be green or one parity/docs task away.

---

### Phase 6 — Parity tests, documentation, quality gates

- [ ] **6.1** Add automated parity table tests in `internal/core` (and config load where useful):
  1. **Merge parity** — equivalent pre-selection values → identical `mergeSelectedToolIDs` output (`// Covers AC-37.010`).
  2. **Cap parity** — legacy dynamic cap settings mapped to `tools.selection` → identical post-cap ids on tier `full` (`// Covers AC-37.011`).
  3. **Runtime top-K cap** — `tool_search_top_k` &gt; `runtime_skills.tool_vector_top_k_cap` → effective top-K = min (`// Covers AC-37.009`).
  - Assert `tool_vector_top_k_cap` only under `runtime_skills` in struct inspection test (`// Covers AC-37.019`).
  - _Requirements:_ [REQ-37.009](ep-requirements.md#req-37-009--apply-runtime-skills-top-k-cap)–[REQ-37.011](ep-requirements.md#req-37-011--preserve-ep-018-dynamic-cap), [REQ-37.017](ep-requirements.md#req-37-017--tests-prove-equivalent-config-parity), [REQ-37.021](ep-requirements.md#req-37-021--keep-tool_vector_top_k_cap-location)
  - _Acceptance Criteria:_ [AC-37.009](ep-acceptance-criteria.md#ac-37-009), [AC-37.010](ep-acceptance-criteria.md#ac-37-010), [AC-37.011](ep-acceptance-criteria.md#ac-37-011), [AC-37.019](ep-acceptance-criteria.md#ac-37-019)
  - **Verification:** `go test ./internal/core/... -run 'Parity|Equivalent|TopK' -count=1` passes.

- [ ] **6.2** Update `docs/configuration.md`: document `tools.selection`; one-to-one migration table from `tool_pre_selection` and `tools.dynamic_selection`; intent-tier table references `tools.selection.enabled`; document `runtime_skills.tool_vector_top_k_cap` interaction with `tools.selection.tool_search_top_k`.
  - _Requirements:_ [REQ-37.014](ep-requirements.md#req-37-014--document-migration-in-configurationmd), [REQ-37.015](ep-requirements.md#req-37-015--document-tool_vector_top_k_cap-interaction)
  - _Acceptance Criteria:_ [AC-37.014](ep-acceptance-criteria.md#ac-37-014) (**MANUAL ONLY** — read doc for `tools.selection`, migration table, intent-tier `enabled`), [AC-37.015](ep-acceptance-criteria.md#ac-37-015) (**MANUAL ONLY** — read doc for `tool_vector_top_k_cap` under `runtime_skills`)
  - **Verification:** `grep -E 'tools\.selection|tool_pre_selection|dynamic_selection' docs/configuration.md` — `tools.selection` present; legacy keys only in migration/removal context.

- [ ] **6.3** Final quality gates and residual grep.
  - Run `make check` (**AC-37.016 MANUAL ONLY** — process gate, exit 0).
  - Run `make build` then `./bin/validate ears EP-037` and `./bin/validate EP-037` if epic validate target is registered (**AC-37.017 MANUAL ONLY** for validate CLI).
  - Repo-wide grep (zero residual):  
    `grep -rE 'tool_pre_selection|dynamic_selection|ToolPreSelection|ToolDynamicSelection' --include='*.go' --include='*.json' internal/ cmd/ tests/ config.examples/ docs/ .config/ 2>/dev/null || true`  
    Expect **no** matches except rejection test fixtures / migration prose in `docs/configuration.md`.
  - Scope guards (**MANUAL ONLY**): [AC-37.020](ep-acceptance-criteria.md#ac-37-020) no `vector_search_tools` schema DRY; [AC-37.021](ep-acceptance-criteria.md#ac-37-021) core diff wiring-only; [AC-37.022](ep-acceptance-criteria.md#ac-37-022) no new selection features; [AC-37.023](ep-acceptance-criteria.md#ac-37-023) operator `.config/config.json` loads at startup.
  - _Requirements:_ [REQ-37.016](ep-requirements.md#req-37-016--tests-reject-legacy-keys)–[REQ-37.019](ep-requirements.md#req-37-019--validate-ears-ep-037-passes), [REQ-37.018](ep-requirements.md#req-37-018--make-check-passes)
  - _Acceptance Criteria:_ [AC-37.005](ep-acceptance-criteria.md#ac-37-005)–[AC-37.007](ep-acceptance-criteria.md#ac-37-007), [AC-37.016](ep-acceptance-criteria.md#ac-37-016)–[AC-37.023](ep-acceptance-criteria.md#ac-37-023) (see MANUAL notes above)
  - **Verification:** `make check` exit 0; `./bin/validate ears EP-037` exit 0; `./bin/validate EP-037` exit 0 when available; grep command above clean.

---

## Dependencies and order

| Task | Depends on |
|------|------------|
| 1.2 | 1.1 |
| 1.3 | 1.2 |
| 2.1 | 1.1 |
| 2.2 | 2.1 |
| 2.3 | 2.1 |
| 3.1 | 1.1 |
| 3.2 | 3.1 |
| 3.3 | 3.1 |
| 3.4 | 3.2, 3.3 |
| 4.1 | 3.4 (readers migrated) |
| 5.1–5.4 | 1.2, 2.1 (validators + rejection active); **5.2–5.4** should land before or with **4.1** if `tools.selection` is required at load — prefer **5.2 immediately after 2.x** if `make check` must stay green through Phase 4 |
| 6.1 | 3.4, 4.1, 5.4 |
| 6.2 | 5.1 (operator example aligned) |
| 6.3 | 1.3, 2.3, 5.4, 6.1, 6.2 |

**Recommended green-build path:** 1.1 → 1.2 → 2.1 → 2.2 → 2.3 → **5.2 → 5.3 → 5.4 → 5.1** → 1.3 → 3.x → 4.1 → 6.1 → 6.2 → 6.3.

---

## Checkpoints

- **After 2.3:** Legacy keys fail load with explicit errors; whitelist accepts `tool_output_artifacts` without typed model.
- **After 5.2:** `grep -rl 'tool_pre_selection' internal/config/testdata/ | wc -l` is **0**; `TestLoad_AllFixturesLoad` green.
- **After 4.1:** No `ToolPreSelection` / `ToolDynamicSelection` symbols in product Go.
- **After 6.1:** Parity tests demonstrate equivalent config → identical merged ids, cap, and effective top-K.
- **Before stage 10:** `make check` green; `./bin/validate EP-037` / `ears EP-037` pass; MANUAL AC checklist signed off (docs read, operator config, scope guards).

---

## AC coverage map (automated vs manual)

| AC | Primary task(s) | Notes |
|----|-----------------|-------|
| AC-37.001–004 | 1.2, 1.3 | Unit; `// Covers AC-37.00x` |
| AC-37.005–008 | 2.1–2.3 | Unit + testdata |
| AC-37.007 | 2.2 | Unit |
| AC-37.009–011 | 6.1 | Automated parity tables |
| AC-37.012 | 3.1–3.4, 4.1 | Unit / compile |
| AC-37.013 | 5.1–5.4 | `TestLoad_AllFixturesLoad` |
| AC-37.014 | 6.2 | **MANUAL ONLY** |
| AC-37.015 | 6.2 | **MANUAL ONLY** |
| AC-37.016 | 6.3 | **MANUAL ONLY** (`make check`) |
| AC-37.017 | 6.3 | **MANUAL ONLY** (`./bin/validate ears EP-037`) |
| AC-37.018 | 5.4, 6.3 | Unit unknown root key + examples |
| AC-37.019 | 6.1 | Unit struct inspection |
| AC-37.020–022 | 6.3 | **MANUAL ONLY** (diff review) |
| AC-37.023 | 5.1, 6.3 | **MANUAL ONLY** (operator live config) |

---

## Files touched (reference)

Product Go: `internal/config/config.go`, `load.go`, `root_keys.go`, `tools_selection_test.go` (new), `config_test.go`, `intent_classifier_test.go`, `vector_search_tools_test.go`; `internal/core/run.go`, `handler.go`, `handler_tier_main_prompt.go`, `integration_export.go`, `run_test.go`, `handler_test.go`, `handler_ep018_coverage_test.go`; `cmd/pa/main_test.go`.

Configs: `.config/config.json`, `config.examples/config.example.json`, 62× `internal/config/testdata/*.json`, `tests/integration/testdata/runtime_skills/minimal_ok/config.json`, `tests/integration/config_helpers.go`.

Docs: `docs/configuration.md`.

**Do not edit:** `internal/config/vector_search_tools.go` (schema unchanged per REQ-37.022).
