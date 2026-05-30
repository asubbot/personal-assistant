---
artefact: ep-system-design-review
epic_id: EP-037
status: draft
source_of_truth: true
gate: fail
latest_iteration: 1
open_counts:
  blocker: 0
  major: 1
  medium: 3
  minor: 3
next_action: return_to_stage_6
updated_at: 2026-05-30
---

# Architecture Review — EP-037 Consolidate tool pre-selection configuration

**Reviewer:** AI Agent (fresh delegated reviewer, Stage 7 iteration 1)

---

## Current Gate Summary

Gate: Fail
Latest iteration: 1
Last updated: 2026-05-30
Open counts: Blocker 0 | Major 1 | Medium 3 | Minor 3
Open findings:
- F-001 Major: Merged `validateToolsSelection` call-site vs tool-catalog load ordering can drop the `always_include` floor check → behaviour drift, breaks AC-37.003 parity.
- F-002 Medium: `tools.tool_output_artifacts` present in operator `.config/config.json` is not in the proposed nested-key whitelist/struct; strict `validateToolsObjectKeys` (REQ-37.008) would reject it at load (AC-37.023). Design defers without a concrete decision.
- F-003 Medium: testdata inventory inaccurate — design states "52 files"; actual is 62 testdata fixtures contain `tool_pre_selection`; enumerated grep baseline omits ≥5 files.
- F-004 Medium: No "Risks and trade-offs" section (required by stage-7 structural checklist).
- F-005 Minor: `integration_export.go` contract names inaccurate (`NewConversationHandlerParams` vs actual `IntegrationConversationParams`/`NewIntegrationConversationHandler`; today the integration handler never wires the dynamic cap).
- F-006 Minor: Error-handling table link text "REQ-37.018" points to the REQ-37.020 anchor (should be REQ-37.020 / AC-37.018).
- F-007 Minor: `tests/integration/runtime_skills_handler_test.go` (7 field sites) is absent from the file/test inventory.
Next action: Return to stage 6

---

## Review iteration 1

**Review date:** 2026-05-30
**Stage 7 iteration:** 1 of max 5
**Document reviewed:** [ep-system-design.md](ep-system-design.md)
**Iteration summary — open counts:** Blocker: 0 | Major: 1 | Medium: 3 | Minor: 3
**Gate:** Fail (Major/Medium/Minor > 0)

### Overall assessment

The design is well-targeted and correctly preserves the runtime selection algorithms: `mergeSelectedToolIDs` top-K cap (`min(tool_search_top_k, runtime_skills.tool_vector_top_k_cap)`) and `mergedAfterDynamicToolCap`/`pickToolsForMainRequest` stay unchanged, with only field-source rewiring — matching the verified code. The legacy-key rejection approach (root `tool_pre_selection`, nested `tools.dynamic_selection`) is the correct EP-034/EP-036 raw-JSON pattern, and the `root_keys.go` change (drop `tool_pre_selection`, keep `tools`) is right. However, one Major load-ordering ambiguity could silently drop the `enabled`→`always_include` floor check and break parity, and three Medium issues (operator `tool_output_artifacts` vs the new strict nested whitelist, an undercounted testdata inventory, and a missing Risks section) should be resolved before stage 8.

**Verdict:** Fail gate

### Strengths

- Runtime parity intent is concrete and code-accurate: design's "Unchanged: `mergeSelectedToolIDs` top-K cap block; `pickToolsForMainRequest`" matches `internal/core/handler.go:318-321` and `handler_tier_main_prompt.go:88-93`.
- `enabled`↔`max_tools_for_llm_request` runtime semantics correctly mirror today's `mergedAfterDynamicToolCap` gate (`toolsDynamic == nil || !Enabled || len(merged)==0` → return merged): the numeric cap is correctly described as ignored at runtime when disabled.
- Struct-user inventory for Go sources is complete: every `.go` file referencing `ToolPreSelection`/`ToolDynamicSelection`/`.DynamicSelection` (`config.go`, `load.go`, `run.go`, `handler.go`, `ep018_dynamic_tools_test.go`, `run_test.go`, `handler_ep018_coverage_test.go`, `config_helpers.go`) is named in the design.
- Bounds are at least as strict as today: design reuses `maxToolSearchTopK=500`, `maxToolMinCount=500`, `maxToolFallbackCap=1000` (confirmed in `load.go:20-26`) with the same `>= 1` lower bounds.
- Full REQ→design traceability table covering all 24 requirements (REQ-37.001–37.024).

### Issues and recommendations

#### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

#### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| F-001 | Merging `validateToolPreSelection` + `validateToolDynamicSelection` into one `validateToolsSelection(c *Config)` wired into `validateTools` would run the `enabled`→`max >= countValidAlwaysIncludeTools(c)` floor check **before** the tool catalog is loaded, where `countValidAlwaysIncludeTools` returns 0 (catalog nil) and the floor check is skipped — a behavioural drift that lets previously-rejected configs load. | In `load.go`, `validateTools` is invoked from `validateMandatoryJSONSectionsCore` inside `validate(raw)` at `prepareConfig` line 55 — **before** `toolcatalog.Load` (lines 77-81). Today `validateToolDynamicSelection` is deliberately called at line 86 **after** catalog load because `countValidAlwaysIncludeTools` needs `c.ToolCatalog`. The design's Validation table maps "`tools.selection` present → `validateTools` → `validateToolsSelection`" and says "Replace … → single `validateToolsSelection(c *Config)`", placing the floor check at the early (pre-catalog) site. This breaks AC-37.003 / REQ-37.003 parity. | Specify the call ordering explicitly: keep the `enabled` always_include-floor portion at the post-catalog-load call site (line 86 today), or split into `validateToolsSelectionBounds` (early) + `validateToolsSelectionDynamicCap` (after catalog load). State in the design that the floor check must run after `ToolCatalog` is populated to preserve current semantics. |

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| F-002 | The operator `.config/config.json` contains a `tools.tool_output_artifacts` block (lines 42-55) that is **not** in `ToolsConfig` (no struct field, no Go reference) and is currently tolerated only because there is no strict nested-`tools` validation today. The proposed `validateToolsObjectKeys` whitelist (`always_include`, `selection`, `vector_search_tools`, `create_tool_secret_patterns`) excludes it, so REQ-37.008 strict rejection would fail-fast on the live operator config (AC-37.023). | `.config/config.json` is gitignored (so automated `make check`/testdata fixtures stay green and won't surface it), but startup load would break for the operator. The design acknowledges this in §"Unknown `tools` keys" but leaves it ambiguous ("either add to the whitelist and struct in a follow-up or document removal"). | Make a concrete, ordered decision in the design: either (a) add `tool_output_artifacts` to the whitelist + `ToolsConfig` (and validate it) within EP-037 sequencing, or (b) document its removal from the operator config as a required migration step, or (c) scope it to a named follow-up epic and explicitly exclude it from the EP-037 whitelist with operator guidance. Do not leave it as an open either/or. |
| F-003 | The testdata inventory undercounts: design says "**All** … (52 files per repo grep)" and the enumerated grep-baseline list, but `rg -l tool_pre_selection internal/config/testdata/` returns **62** files. At least 5 fixtures are missing from the enumerated list: `log_redaction_reserved_id.json`, `llm_default_max_tokens_zero.json`, `llm_default_temperature_zero.json`, `valid_with_tool_catalog.json`, `invalid_observability_http_relative_health_path.json` (the list also duplicates `log_redaction_invalid_regex.json`). | A wrong count and incomplete enumeration risk a partial migration → unconverted fixtures fail load (AC-37.013) or break the build. The "All … containing `tool_pre_selection`" catch-all and sequencing step 2 mitigate but the stated number/list are authoritative artefacts. | Correct the count to 62 and regenerate the enumerated list from a fresh grep (or replace the literal list with the grep command + count as source of truth), de-duplicate `log_redaction_invalid_regex.json`. |
| F-004 | No "Risks and trade-offs" section. The stage-7 structural checklist requires it; the doc has Overview, Architecture, Components, Data models, Error handling, Testing strategy, Implementation sequencing, Files-to-update, Test inventory, and Traceability, but no risks/trade-offs heading. | Stage-7 skill Step 2 structural check. | Add a short "Risks and trade-offs" section (e.g. atomic bulk-fixture migration risk vs `make check` greenness, the `tool_output_artifacts` operator-config break risk from F-002, the floor-check ordering risk from F-001). |

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| F-005 | The `integration_export.go` contract is described as "`NewConversationHandlerParams`"; the actual symbols are `IntegrationConversationParams` (params struct) and `NewIntegrationConversationHandler`. Also, that constructor does **not** currently wire any dynamic-cap field (no `toolsDynamic` assignment at lines 173-198), so cap-parity integration tests (AC-37.011) require adding the new field, not just renaming. | `internal/core/integration_export.go:114-199`. | Use the real type/function names and state that a `ToolsSelection *config.ToolsSelection` (or `DynamicEnabled`/`DynamicMax`) field must be **added** to `IntegrationConversationParams` and assigned into the handler. |
| F-006 | Error-handling table row "Unknown top-level key" link text reads "REQ-37.018" but the anchor target is `req-37-020`; the rule traces to REQ-37.020 / AC-37.018, and REQ-37.018 is "`make check` passes". | Design Error handling table. | Fix the link text to REQ-37.020 (and reference AC-37.018). |
| F-007 | `tests/integration/runtime_skills_handler_test.go` references the preserved `ToolSearchTopK`/`ToolMinCount`/`ToolFallbackCap` fields at 7 sites but is not listed in the file/test inventory. Harmless (fields are preserved, so it compiles unchanged), but the inventory claims to cover every match. | grep of the inventory needle. | Add the file to the inventory with a note "no change required (preserved fields)" so the grep baseline is complete. |

### Project rules compliance (optional)

| Rule | Compliance |
|------|------------|
| KISS | ✅ Single consolidated block; no new abstractions; algorithms untouched. |
| Fail fast | ⚠️ Strong legacy-key rejection, but F-001 (floor-check ordering) and F-002 (`tool_output_artifacts`) risk either skipping a check or fail-fast breaking the operator config. |
| Security | ✅ No change to security model, secret patterns, or tool contract. |
| Testability | ✅ Parity tables (merge/cap/top-K) and rejection tests are well specified; F-001 must be fixed for AC-37.003 to be testable as intended. |

### NFR coverage (optional)

| NFR | Coverage | Status |
|-----|----------|--------|
| REQ-37.020 explicit-JSON preserved | Root keys drop `tool_pre_selection`; every key still required once; nested whitelist added | Needs work (F-002) |
| REQ-37.021 keep `tool_vector_top_k_cap` location | Stays under `runtime_skills`; handler `min` unchanged | OK |
| REQ-37.022 no `vector_search_tools` DRY | Explicitly out of scope; `validateTools` block untouched | OK |
| REQ-37.023 limit core changes | Wiring-only diff in `run.go`/`handler.go`/`integration_export.go` | OK |
| REQ-37.024 no new selection features | No ranking/tier-cap changes | OK |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)
