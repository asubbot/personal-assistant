---
artefact: ep-system-design-review
epic_id: EP-036
status: draft
source_of_truth: true
gate: fail
latest_iteration: 1
open_counts:
  blocker: 0
  major: 1
  medium: 1
  minor: 3
next_action: return_to_stage_6
updated_at: 2026-05-30
---

# Architecture Review — EP-036 Simplify intent classification

**Reviewer:** AI Agent (fresh reviewer, did not author the design)

---

## Current Gate Summary

Gate: Fail
Latest iteration: 1
Last updated: 2026-05-30
Open counts: Blocker 0 | Major 1 | Medium 1 | Minor 3
Open findings:
- F-001 Major: Core test-file inventory incomplete — `handler_ep018_coverage_test.go` (and unnamed `handler_tier_main_prompt_test.go`) reference removed symbols/old constructor signatures and a doc-content assertion, all of which break the build or fail after the doc update.
- F-002 Medium: No automated test loads operator `.config/config.json`; the step-1 "CI catches stale operator files" rationale is inaccurate and AC-36.018's Unit-level coverage of the live config is not achievable as designed.
- F-003 Minor: `intent.Result.Stage` doc comment ("heuristic", "model", or "default") and `intent` package doc comments (EP-017/EP-018) not flagged for update.
- F-004 Minor: `docs/architecture-ru.md` retains `full_lite` / three-tier references and is absent from the design's documentation-update list.
- F-005 Minor: Retained test `TestCascadeClassifier_ResultContainsStageAndLen` (and peers) call the old constructor signatures; design labels it "keep" without noting the required signature edit.
Next action: Return to stage 6

---

## Review iteration 1

**Review date:** 2026-05-30
**Stage 7 iteration:** 1 of max 5
**Document reviewed:** [ep-system-design.md](ep-system-design.md)
**Iteration summary — open counts:** Blocker: 0 | Major: 1 | Medium: 1 | Minor: 3
**Gate:** Fail (Major/Medium/Minor > 0)

### Overall assessment

The design is strong on the two highest-risk axes: config strictness (the EP-034-style raw-JSON `rejectRemovedIntentClassifierKeys` genuinely fails fast before struct unmarshal and does not weaken strict validation) and behavioural safety (former `full_lite`/ambiguous messages fold into the richer `full` path, `simple`/`full` assembly unchanged). Config-file completeness is accurate: `.config/config.json` is verifiably the only config file still carrying `model_stage` / `full_lite_patterns` (all `config.examples`, `testdata`, and integration fixtures already use `"intent_classifier": null`). The gate fails on one Major: the core test-file inventory is incomplete and would break `make check`, contradicting the design's own "compiles after each step" sequencing claim.

**Verdict:** Fail gate — return to stage 6.

### Strengths

- **Fail-fast config rejection is sound.** `rejectRemovedIntentClassifierKeys` mirrors the existing `rejectRemovedToolsConfigKeys` (`internal/config/load.go:111`) raw-`map[string]json.RawMessage` inspection invoked from `rejectRemovedUnsupportedConfigKeys`, correctly noting `encoding/json` would otherwise silently drop unknown fields after struct removal (REQ-36.016/017, AC-36.013/014). Strict top-level root-key validation is untouched.
- **Config-file list is complete and accurate.** Verified by searching all files including hidden dirs: only `.config/config.json` (lines 200–235) contains the removed keys; every fixture under `internal/config/testdata/`, `config.examples/config.example.json`, `tests/integration/testdata/.../config.json`, and the `cmd/pa/main_test.go` embedded config already use `"intent_classifier": null`. The three new testdata fixtures cover the rejection + positive load paths (AC-36.013/014/015/018).
- **Production symbol coverage is complete.** Every production site referencing removed symbols is accounted for: `tier.go` (`TierFullLite`), `cascade.go` (`model` field/branch, constructor), `heuristic.go` (`fullLitePatterns`, constructor, evaluation order), `model.go` (delete), `handler_tier_main_prompt.go` (`TierFullLite` case + `buildTierFullLiteMainPrompt` + tail comment), `config.go`/`load.go`/`resolve.go`, and `cmd/pa/main.go` `buildIntentClassifier` (lines 630–664, including `model_stage`/`classifier_model` log attrs).
- **Behavioural direction is safe.** Defaulting ambiguous + former-`full_lite` to `full` (richer prompt: skills + RAG + tail) is the conservative direction; `buildMainTurnMessagesPreTail` already gathers RAG when `tier == intent.TierFull`, so cascade `default → TierFull` preserves full-tier behaviour. The added-token risk is acknowledged in the Risks table.

### Issues and recommendations

#### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

#### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| F-001 | **Core test-file inventory incomplete — build break + doc-content test conflict.** The design's "Retire/rewrite" list (Testing strategy) names only `internal/core/handler_ep018_test.go`, but two other core test files reference removed symbols / old signatures and are unaddressed. (a) `internal/core/handler_ep018_coverage_test.go` is entirely unlisted: it constructs `intent.NewHeuristicClassifier(nil, nil, []string{...}, 100)` and `intent.NewCascadeClassifier(h, nil, nil)` (old 4-arg / 3-arg signatures → **compile error** after the signature changes in §Components), and contains `TestEP018_fullLite_*` tests driven by `TierFullLite`. (b) Critically, `TestEP018_configurationDoc_containsTierMatrix` (same file) asserts `docs/configuration.md` **contains** the substring `"full_lite"` — sequencing step 6 removes `full_lite` from that doc, so this test **fails** after the doc update. (c) `internal/core/handler_tier_main_prompt_test.go` is not explicitly named yet calls `h.assembleTierMainLLMParams(ctx, intent.TierFullLite, ...)` and `h.buildTierFullLiteMainPrompt(...)` → **compile error** after removal. This contradicts the design's claim that each sequencing step keeps `make check` green (REQ-36.026) and that step 4 "rewrite tier tests" is complete. | `internal/core/handler_ep018_coverage_test.go` (constructors at lines 83–87, 143–147, 197–201, 228–232, 269–273; doc assertion `TestEP018_configurationDoc_containsTierMatrix` lines 64–77; `TierFullLite`-driven tests 138–249, 263–309). `internal/core/handler_tier_main_prompt_test.go` (lines 39, 73). REQ-36.013/025/026, AC-36.010/019/020. | Extend the design's Testing strategy "Retire/rewrite" list to explicitly enumerate **both** files. State for `handler_ep018_coverage_test.go`: update all `NewHeuristicClassifier`/`NewCascadeClassifier` calls to the new signatures; delete or rewrite the `full_lite`-specific tests (session-parity, catalog-tools, dynamic-selection) to the `full` path; and **rewrite `TestEP018_configurationDoc_containsTierMatrix`** to assert the new two-tier doc content (must not assert `full_lite`). For `handler_tier_main_prompt_test.go`: delete the `TierFullLite`/`buildTierFullLiteMainPrompt` cases. Re-confirm the step-by-step "green after each step" sequencing accounts for these edits. |

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| F-002 | **Operator `.config/config.json` is not loaded by any automated test; sequencing rationale is inaccurate.** AC-36.018 lists the operator `.config/config.json` under **Unit** test level ("each representative config SHALL load successfully"), and sequencing step 1's rationale states "step 1 should precede stripping `.config` keys so CI catches stale operator files." No test loads `.config/config.json` (no Go test references that path; a real `Load` requires secrets/known_hosts/tool-catalog files that are not test fixtures). So after adding key rejection, a stale `.config/config.json` would **not** be caught by `make check` — it would fail only at runtime/process start. The design's automated-coverage assumption for the live config is unfounded. | Sequencing step 1 + Risks table ("CI catches stale operator files"); AC-36.018 (Unit). Verified: no `.go` test references `.config/config.json`. | Either (a) correct the design to state the live `.config/config.json` is verified **manually** (AC-36.018 unit coverage applies to `config.examples` + the new `testdata` fixtures only), or (b) add an explicit automated load/parse check for `.config/config.json` (e.g. a focused raw-JSON rejection test that does not require full `Load` side-effects). Remove/replace the inaccurate "CI catches stale operator files" rationale. |

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| F-003 | `intent.Result.Stage` doc comment still reads `// "heuristic", "model", or "default"` and the `intent` package/file doc comments reference the now-removed model stage ("two-stage intent classifier (EP-017, EP-018)", heuristic "Order: … → full_lite → ambiguous (EP-018)"). REQ-36.011 narrows `Stage` to `heuristic`/`default`; the design does not flag these comments for update. | `internal/intent/tier.go:1,18`; `internal/intent/heuristic.go:14,39`. REQ-36.011 alignment / doc hygiene. | Add a line to the §Components removals (or a small "doc/comment updates" note) to refresh the `Result.Stage` comment and package/order comments. |
| F-004 | `docs/architecture-ru.md` retains `full_lite` and three-tier references (`simple / full_lite / full`, `else tier = full_lite`, the `full_lite` tier description, the classification-flow list) but is not in the design's documentation-update list (nor in REQ-36.022, which enumerates only `configuration.md` and `llm-provider-roles-and-logging.md`). Leaving it produces stale architecture doc drift. | `docs/architecture-ru.md:162,230,260,519`. REQ-36.022 scope boundary. | Either add `docs/architecture-ru.md` to the design's docs-update list (preferred for consistency) or explicitly note it as accepted out-of-scope drift in the Risks table so the omission is deliberate. |
| F-005 | Retained intent tests use the old constructor signatures (`NewHeuristicClassifier(..., nil, ...)` 4-arg, `NewCascadeClassifier(h, nil, nil)` 3-arg). The design explicitly labels `TestCascadeClassifier_ResultContainsStageAndLen` as "keep" (§Components removed table) without noting it must be edited for the new 2-arg/3-arg→2-arg signatures; same applies to `cascade_test.go`/`heuristic_test.go` "fix intent tests". | `internal/intent/observability_test.go:40–41`; `cascade_test.go`, `heuristic_test.go`. | Clarify that "keep" means "retain intent but update to new signatures." Already implied by step 2 "fix intent tests"; a one-line note removes ambiguity. |

### Project rules compliance

| Rule | Compliance |
|------|------------|
| KISS | ✅ Removes a whole LLM stage + tier; net simplification. |
| Fail fast | ✅ Raw-JSON key rejection at load; heuristic regex/`max_simple_len` validated at load. |
| Security | ✅ No security posture change; removes an outbound classifier LLM call (fewer secrets/endpoints in play). |
| Testability | ⚠️ Test inventory incomplete (F-001); two core test files would break the build before fixes. |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)
