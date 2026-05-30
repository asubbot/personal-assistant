---
artefact: ep-system-design-review
epic_id: EP-036
status: draft
source_of_truth: true
gate: fail
latest_iteration: 2
open_counts:
  blocker: 0
  major: 0
  medium: 1
  minor: 1
next_action: return_to_stage_6
updated_at: 2026-05-30
---

# Architecture Review — EP-036 Simplify intent classification

**Reviewer:** AI Agent (fresh reviewer, did not author the design)

---

## Current Gate Summary

Gate: Fail
Latest iteration: 2
Last updated: 2026-05-30
Open counts: Blocker 0 | Major 0 | Medium 1 | Minor 1
Open findings:
- F-002 Medium (still open): The verification rationale is now accurate (manual operator-file check + `config.examples`/`testdata` unit coverage) but is **not reconciled with AC-36.018**, which still lists operator `.config/config.json` under **automated Unit tests**. Design declares that file manual-only, contradicting the AC instead of reconciling it.
- F-006 Minor (new): Design instructs editing `model_stage` / `classifier_model` **log-key expectations** in `cmd/pa/ep024_operator_logging_test.go`, but no such log-key assertion exists in that file (only a doc-content check). The production removal in `main.go` is correctly specified; the test-edit instruction (update 1) is spurious.

Resolved in iteration 2: F-001 (Major), F-003/F-004/F-005 (Minor) — verified against code.
Next action: Return to stage 6 (and amend AC-36.018 at stage 5 to mark the operator file Manual, or add an automated check)

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

---

## Review iteration 2

**Review date:** 2026-05-30
**Stage 7 iteration:** 2 of max 5
**Document reviewed:** [ep-system-design.md](ep-system-design.md)
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 1 | Minor: 1
**Gate:** Fail (Medium/Minor > 0)

### Overall assessment

The iteration-1 Major (F-001) and all three Minors (F-003/F-004/F-005) are genuinely resolved and verified against the actual code: the Testing strategy now carries an exhaustive, line-accurate test-file inventory (including `handler_ep018_coverage_test.go` and `handler_tier_main_prompt_test.go`, the `full_lite`-driven cases, and the two doc-content assertions that must be rewritten in the same commit as their docs), the `internal/intent` comment refresh is tabulated, `docs/architecture-ru.md` is added with the four exact stale lines, and "keep" tests are explicitly flagged for the new constructor signatures. F-002's rationale is now accurate (no automated test loads the operator file; manual verification), but it is **not reconciled with AC-36.018**, which still mandates automated Unit-level loading of operator `.config/config.json` — the design contradicts the AC rather than aligning it. One new Minor: the design instructs a test edit (`ep024_operator_logging_test.go` log-key expectations) that has no corresponding assertion in that file.

**Verdict:** Fail gate — return to stage 6 (with a stage-5 AC amendment for F-002).

### Strengths

- **F-001 fully resolved (verified).** The "Exhaustive test-file inventory" (design §Testing strategy) now names every site found by re-grepping `FullLite|full_lite|TierFullLite|ModelStage|ModelClassifier|NewHeuristicClassifier|NewCascadeClassifier`. Confirmed against code: `handler_ep018_coverage_test.go` old constructors (lines 83–87, 143–147, 197–201, 228–232, 269–273), the four `^LITE*` `full_lite` tests (session/ catalog-tools/ no-tools/ dynamic-cap), and the critical `TestEP018_configurationDoc_containsTierMatrix` (lines 64–77, asserts `"full_lite"`) flagged for rewrite; `handler_tier_main_prompt_test.go` `TierFullLite` (line 39) and `buildTierFullLiteMainPrompt` (line 73) flagged for deletion. Sequencing step 6 correctly couples both doc-content tests to their doc edits in one commit.
- **F-003/F-004/F-005 resolved (verified).** `internal/intent` comment table matches code exactly (`tier.go:1` package doc, `tier.go:18` `Result.Stage` comment, `heuristic.go:14` type doc, `heuristic.go:39` order comment); `docs/architecture-ru.md` lines 162/230/260/519 match the live doc; the "keep ⇒ migrate signatures" clarification matches `observability_test.go:40–41` (`NewHeuristicClassifier(..., nil, 40)` / `NewCascadeClassifier(h, nil, nil)`).
- **Previously-validated strengths still hold.** EP-034-style raw-JSON `rejectRemovedIntentClassifierKeys` (fail-fast before struct unmarshal; strict top-level validation untouched); complete config-file list (only `.config/config.json` carries the removed keys); complete production-symbol coverage (`tier.go`, `cascade.go`, `heuristic.go`, `model.go`, `handler_tier_main_prompt.go`, `cmd/pa/main.go:657,660` log attrs, `config.go`/`load.go`/`resolve.go`); safe full-fold of former `full_lite`/ambiguous turns into the richer `full` path.

### Issues and recommendations

#### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

#### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None. F-001 resolved — test-file inventory is now exhaustive and line-accurate; verified against `handler_ep018_coverage_test.go` and `handler_tier_main_prompt_test.go`._

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| F-002 | **Verification rationale accurate but not reconciled with AC-36.018.** The design's F-002 note and Risks table now correctly state that **no** automated test loads operator `.config/config.json` (a real `config.Load` needs secrets/known_hosts/tool-catalog files), so the file is verified **manually**, and the inaccurate "CI catches stale operator files" rationale was removed from sequencing step 1. However, **AC-36.018 (Test level: Unit)** still reads: *"Given `config.examples/config.example.json`, operator `.config/config.json`, and updated files under `internal/config/testdata/` … When config load runs in automated tests, Then each representative config SHALL load successfully …"*. The design unilaterally reinterprets AC-36.018 in prose ("Unit coverage … is provided by `config.examples` + the three new `testdata` fixtures … operator `.config/config.json` … verified manually"), but the AC itself was not amended — so the design now **contradicts** the AC. As written, AC-36.018's automated-test expectation for the operator file would be provably unmet at acceptance. | Design §"Configuration files to update" F-002 note + Risks table; sequencing step 1/7. AC-36.018 (lines 314–323, ep-acceptance-criteria.md), Test level **Unit**, lists operator `.config/config.json` under "automated tests". | Reconcile, do not reinterpret. Either (a) **return to stage 5** to amend AC-36.018 so the operator `.config/config.json` is split to **Manual** test level (examples + testdata stay Unit), keeping AC and design consistent; **or** (b) add the focused automated raw-JSON / no-removed-keys check on `.config/config.json` (option (b) from iteration 1) so the design satisfies AC-36.018 as written. Pick one and make the AC and design agree explicitly. |

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| F-006 | **Spurious test-edit instruction — no matching assertion exists.** The design instructs (twice: §`cmd/pa` "Tests" line and Testing-strategy `cmd/pa` update (1)) to "remove expected log keys `model_stage` / `classifier_model` from intent-enabled fixture expectations" in `cmd/pa/ep024_operator_logging_test.go`. That file contains **no** log-key assertion — its only `model_stage` reference is the doc-content check in `TestEP024_ProviderRolesDocContent` (line 71), which is correctly handled by update (2). A repo-wide grep finds `model_stage`/`classifier_model` only in `cmd/pa/main.go:657,660` (production, correctly slated for removal) and that doc-content check. The instruction points the implementer at nothing and weakens the otherwise-exhaustive inventory's credibility. | `cmd/pa/ep024_operator_logging_test.go` (no log-key assertion; doc check at lines 65–78); `cmd/pa/main.go:657,660`. Design §`cmd/pa — buildIntentClassifier` "Tests" line + Testing-strategy `cmd/pa` bullet update (1). | Drop update (1) for `ep024_operator_logging_test.go` (there is no log-key fixture to edit); keep update (2) (doc-content checks) and the production log-attr removal in `main.go`. If a startup-log-key assertion is intended, point to the actual test/file that asserts it (none currently exists). |

### Project rules compliance

| Rule | Compliance |
|------|------------|
| KISS | ✅ Removes a whole LLM stage + tier; net simplification. |
| Fail fast | ✅ Raw-JSON key rejection at load; heuristic regex / `max_simple_len` validated at load. |
| Security | ✅ No posture change; removes an outbound classifier LLM call. |
| Testability | ⚠️ Test inventory now exhaustive (F-001 closed), but AC-36.018 automated-coverage claim for the operator file remains unreconciled (F-002). |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)
