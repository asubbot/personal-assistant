---
artefact: ep-system-design-review
epic_id: EP-041
status: draft
source_of_truth: true
gate: pass
latest_iteration: 1
open_counts:
  blocker: 0
  major: 0
  medium: 0
  minor: 0
next_action: proceed_to_stage_8
updated_at: 2026-05-31
---

# Architecture Review — EP-041 Full-tier prompt pipeline

**Reviewer:** AI Agent (delegated pipeline stage 7)

---

## Current Gate Summary

Gate: Pass
Latest iteration: 1
Last updated: 2026-05-31
Open counts: Blocker 0 | Major 0 | Medium 0 | Minor 0
Open findings: None (Nit/Suggestion items below do not block the gate)
Next action: Proceed to stage 8

---

## Review iteration 1

**Review date:** 2026-05-31
**Stage 7 iteration:** 1 of max 5
**Document reviewed:** [ep-system-design.md](ep-system-design.md)
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0
**Gate:** Pass (Blocker/Major/Medium/Minor all zero)

### Overall assessment

The system design correctly describes a structural full-tier assembly refactor: introduce `fullTierAssembler` in `handler_full_tier_pipeline.go`, execute five named steps in the order already implied by requirements and scope (skills → merge tools → dynamic cap → tail budget → completion options), route `buildTierFullMainPrompt` through a single `run()` entry, and preserve behaviour via existing tier/handler tests. Verification on branch `epic/EP-041-full-tier-pipeline` confirms the as-is call sequence in `handler_tier_main_prompt.go` matches the documented step mapping; step 1 currently lives in `buildTierFullMainPrompt` while steps 2–5 live in `mergeTailMergedToolsAndOptions`, which the design appropriately consolidates into the assembler. All eight REQs are addressed; scope guards (simple tier unchanged, no config change, parity) are explicit. The document is appropriately minimal for a mechanical refactor epic; optional stage 6 depth is noted as Nit/Suggestion only. Stage 8 may proceed.

**Verdict:** Pass gate

### Strengths

- **Accurate step order:** Design step list matches [ep-requirements.md](ep-requirements.md) flowchart and REQ-41.002; maps to existing helpers `selectSkillPackages`, `mergeSelectedToolIDs`, `mergedAfterDynamicToolCap`, `fitDynamicTailToBudget` / `buildDynamicTailString`, `completionOptionsMergedCatalogNative` (`internal/core/handler_tier_main_prompt.go:80-116`, `handler_tools.go`, `system_tail.go`).
- **Consolidation intent:** Entry `buildTierFullMainPrompt` → `newFullTierAssembler(...).run()` satisfies REQ-41.003 and scope goal of one readable pipeline; resolves today’s split between `buildTierFullMainPrompt` (skill select) and `mergeTailMergedToolsAndOptions` (tail path).
- **KISS scope:** New file only; tier dispatch stays in `handler_tier_main_prompt.go`; no package moves or algorithm changes — aligned with [ep-scope.md](ep-scope.md) out-of-scope guards.
- **Parity contract:** Design and scope require identical `tierMainLLMParams`, tool ids, and system tail for same inputs (REQ-41.004); existing `handler_tier_main_prompt_test.go` and EP-017/018 tier tests provide baseline.
- **Prerequisite satisfied:** Branch includes EP-040 dependency grouping merge (`0adc396`); pipeline code can use grouped `h.tools.*` / `h.llm.*` access consistently.
- **REQ coverage:** REQ-41.001–008 summarized in design § REQ traceability; matches requirement index.

### Findings

| Id | Severity | Description | Evidence | Recommendation |
|----|----------|-------------|----------|----------------|
| N-001 | Nit | Design document title says **EP-040** instead of **EP-041**. | `ep-system-design.md` line 9 vs `epic_id: EP-041` front matter | Fix title in stage 6 polish or stage 8 checklist (cosmetic). |
| N-002 | Nit | Stage 6 recommends **Testing strategy** and **Risks** sections; design only mentions “Tests \| Parity unchanged” in Files table. | `ep-system-design.md`; compare EP-040 design § Testing strategy / Risks | Optional one-liners: parity via existing tests; risk = missed step wiring caught by tests/compiler. |
| N-003 | Nit | REQ traceability is a single summary line; no per-REQ table. | `ep-system-design.md` § REQ traceability vs EP-040 table format | Expand to REQ table in polish pass if validating structure; not required for this epic size. |
| N-004 | Nit | C4 context diagram lives in requirements, not embedded in design. | `ep-requirements.md` C4 C1; `diagrams/c4-context.puml` | Acceptable — requirements artefact carries diagram; optional cross-link from design Overview. |
| S-001 | Suggestion | `ep-acceptance-criteria.md` index lists AC-41.002 and AC-41.004 but omits their `### AC-…` bodies. | AC index vs sections in `ep-acceptance-criteria.md` | Stage 8 maps verification to indexed ACs; optional stage 5 fix for missing bodies. |
| S-002 | Suggestion | `mergeTailMergedToolsAndOptions` fate slightly open (“deprecated or thin wrapper”). | `ep-system-design.md` Files table; scope says replace implicit chain | Stage 8 task 1.2: prefer thin delegate calling assembler during transition, or remove once `run()` owns all steps. |
| S-003 | Suggestion | `ep041_traceability_test.go` appears in implementation plan but not design Testing strategy. | `ep-implementation-plan.md` task 1.4 | Add during implementation if desired; documents step order for AC-41.002 / REQ-41.006. |
| S-004 | Suggestion | `buildDynamicTailString` is post-step-4 assembly on `messages[0].Content`; design lists it under step 4 only. | `handler_tier_main_prompt.go:115` | Stage 8: keep tail string build in step 4 method or document as step 4 output side effect; parity tests guard behaviour. |

#### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

#### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

### Architectural decisions (optional)

| Decision | Justification |
|----------|---------------|
| Unexported `fullTierAssembler` in `internal/core` | REQ-41.001; no new public API |
| Five explicit step methods on assembler | REQ-41.002, REQ-41.006; readability per scope |
| Single `run()` entry from `buildTierFullMainPrompt` | REQ-41.003; replaces ad-hoc two-function chain |
| Reuse existing helper functions unchanged | Behaviour parity (REQ-41.004); KISS refactor |
| Keep tier dispatch in `handler_tier_main_prompt.go` | REQ-41.005; simple tier path untouched |
| No config changes | REQ-41.007; structural refactor only |

### NFR coverage (optional)

| NFR | Coverage | Status |
|-----|----------|--------|
| REQ-41.007 no config schema change | Overview + scope out-of-scope | OK |
| REQ-41.008 `make check` | Files table + scope success criteria | OK (post-implementation) |
| REQ-41.004 parity | Design + existing tier tests | OK |
| Security | No new surfaces; same LLM/tool paths | OK |

### Project rules compliance (optional)

| Rule | Compliance |
|------|------------|
| KISS | ✅ Pipeline type only; no new subsystems or algorithms |
| Fail fast | ✅ Existing error paths (`selectSkillPackages`, `mergeSelectedToolIDs`, `completionOptionsMergedCatalogNative`) unchanged |
| Security | ✅ No config or redaction changes |
| Testability | ✅ Step methods testable via handler fixtures; parity tests |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md) — REQ-41.001–008
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md) — see S-001 for partial AC bodies
- **Scope:** [ep-scope.md](ep-scope.md)

### Requirement traceability verification (this iteration)

| REQ | Design coverage | Branch baseline aligned |
|-----|-----------------|-------------------------|
| REQ-41.001 | `fullTierAssembler` type in new file | Not yet implemented; target file absent (expected pre-stage 8) |
| REQ-41.002 | Five named steps in fixed order | Logic order matches in `handler_tier_main_prompt.go:80-116` |
| REQ-41.003 | `buildTierFullMainPrompt` → assembler `run()` | Today: direct call to `mergeTailMergedToolsAndOptions` after skill select |
| REQ-41.004 | Parity via unchanged helpers + tests | Helpers in `handler_tools.go`, `system_tail.go`; tier tests present |
| REQ-41.005 | Tier dispatch unchanged | `assembleTierMainLLMParams` switch unchanged (`handler_tier_main_prompt.go:67-73`) |
| REQ-41.006 | Step names in pipeline source | Planned in `handler_full_tier_pipeline.go` |
| REQ-41.007 | No config changes | No config edits in epic scope |
| REQ-41.008 | `make check` | Design defers to implementation verification |

---

**Signal:** `STAGE_7_COMPLETE: ai-sdlc-artefacts/epics/EP-041/ep-system-design-review.md [gate=pass, iteration 1, blocker:0 major:0 medium:0 minor:0]`
