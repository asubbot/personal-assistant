---
artefact: ep-system-design-review
epic_id: EP-034
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
updated_at: 2026-05-29
---

# Architecture Review — EP-034 Remove tool-path LLM escalation

**Reviewer:** AI Agent (delegated pipeline stage 7)

---

## Current Gate Summary

Gate: Pass
Latest iteration: 1
Last updated: 2026-05-29
Open counts: Blocker 0 | Major 0 | Medium 0 | Minor 0
Open findings: None
Next action: Proceed to stage 8

---

## Review iteration 1

**Review date:** 2026-05-29
**Stage 7 iteration:** 1 of max 5
**Document reviewed:** [ep-system-design.md](ep-system-design.md)
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0
**Gate:** Pass (Blocker/Major/Medium/Minor all zero)

### Overall assessment

The system design is appropriate for a removal/refactoring epic: it names concrete packages, APIs, tests, and config artefacts to delete, preserves transport fallback with a clear sequence diagram, and traces all 16 requirements. The design aligns with scope, acceptance criteria, and Refactoring 0.02 strategy without over-specifying replacement behaviour. No blockers or loop-exit findings; implementation can proceed to stage 8.

**Verdict:** Pass gate

### Strengths

- **Explicit removal contract:** The “Removed APIs” table and module-boundary deletions give implementers an unambiguous checklist ([ep-system-design.md](ep-system-design.md) § Components).
- **Behavioural clarity:** The mermaid sequence diagram separates tool errors (index unchanged) from transport fallback (next provider), directly supporting AC-34.001 and AC-34.004.
- **Full REQ coverage:** Traceability maps REQ-34.001 through REQ-34.016 to design sections; structural sections (overview, architecture, data models, error handling, testing, risks) are complete per stage 7 checklist.
- **Test migration plan:** Named EP-006 test files and testdata replacements reduce risk of orphaned escalation tests (AC-34.013, AC-34.014).
- **Honest trade-offs:** Risks table documents operator impact of dropping `baseline_index` and EP-006 supersession in docs/artefacts.

### Issues and recommendations

#### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

#### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

#### Nit

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| N-001 | Traceability label | REQ-34.008 row cites “Implementation plan” section, which is not in `ep-system-design.md` (stage 8 artefact). | In stage 8 plan, reference Testing strategy + `config.examples/`; optional one-line cross-link in design traceability row. |
| N-002 | Router state wording | `llmrouter.State` described as “**Removed** or reduced…” leaves two implementation options. | Pick one in implementation plan (prefer delete of escalation fields; ephemeral index only inside `Complete`). |

#### Suggestion

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| S-001 | Config rejection mechanism | Design mentions `validateTools` / unknown nested key; codebase already uses `rejectRemovedUnsupportedConfigKeys` for removed tool keys (e.g. `text_based_enabled`). | Stage 8: add `tools.llm_escalation` to `rejectRemovedUnsupportedConfigKeys` and remove `LLMEscalationConfig` / `validateLLMEscalation` per existing fail-fast pattern. |
| S-002 | Operator doc breadth | [ep-scope.md](ep-scope.md) lists `operations.md` and `troubleshooting.md`; design REQ-34.011 row names only `configuration.md` and `llm-provider-roles-and-logging.md` (matches AC-34.011). | Implementation plan should grep/update all four `docs/` files; AC could be extended later if full doc coverage must be machine-verified. |
| S-003 | Threat model | [ep-scope.md](ep-scope.md) calls out [threat-model.md](../../threat-model.md); risks table mentions it; no dedicated REQ row. | Include threat-model edit in implementation checklist (line ~106 still references `tools.llm_escalation` validation). |

### Architectural decisions (optional)

| Decision | Justification |
|----------|---------------|
| Delete `escalationpolicy` and `toolfailure` entirely | KISS; no partial deprecation for internal-only packages (REQ-34.002, REQ-34.003). |
| Always start at provider index 0 | Removes `baseline_index` mental model; acceptable HOTL trade-off documented in risks. |
| Transport fallback only in `llmrouter` | Preserves outage resilience without tool-driven provider hopping (REQ-34.004, REQ-34.005). |
| Reject legacy `tools.llm_escalation` at load | Fail-fast for operators with stale configs (REQ-34.007, AGENTS.md explicit-config rules). |

### NFR coverage (optional)

| NFR | Coverage | Status |
|-----|----------|--------|
| REQ-34.012 supersession | Overview + scope traceability | OK |
| REQ-34.013–34.014 tests | Testing strategy with file list | OK |
| REQ-34.015–34.016 quality gates | Testing strategy | OK |
| Security / observability | Plain errors; escalation logs removed; transport routing retained | OK |

### Project rules compliance (optional)

| Rule | Compliance |
|------|------------|
| KISS | ✅ Removal-first; no new subsystems |
| Fail fast | ✅ Config rejection; plain tool errors |
| Security | ✅ No weakening of allowlist/cmdsafe; escalation removal reduces complexity |
| Testability | ✅ Unit/integration replacements named |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md) — all 16 REQs mapped in design traceability table
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md) — scenarios align with sequence diagram and testing strategy
- **Scope:** [ep-scope.md](ep-scope.md) — in-scope removals and out-of-scope boundaries respected

### Structural checklist (stage 7)

| Section | Present |
|---------|---------|
| Overview with scope reference | ✅ |
| Architecture diagram (C4 container + mermaid flow) | ✅ |
| Module boundaries | ✅ |
| Components and interfaces | ✅ |
| Data models | ✅ |
| Error handling | ✅ |
| Testing strategy | ✅ |
| Risks and trade-offs | ✅ |
| Requirement traceability | ✅ |
