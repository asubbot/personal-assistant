# Architecture Review — EP-013 Runtime skills and consolidated system prompt

**Review date:** 2026-04-09  
**Reviewer:** AI Agent  
**Document reviewed:** [ep-system-design.md](ep-system-design.md)

---

## 1. Overall Assessment

The system design is detailed and implementation-oriented: it names new packages (`internal/runtimeskills`, `internal/skillindex`, `internal/promptmarkers`, `internal/systemprompt`), maps each requirement to concrete anchors, includes C2 container diagram source, module boundaries, data models, error handling, and a testing strategy aligned with acceptance criteria. Before implementation planning, the **dynamic block order** in the merged system string should be reconciled with [ep-scope.md](ep-scope.md) and [REQ-13.016](ep-requirements.md#prompt-assembly) (and the implied ordering in [REQ-13.015](ep-requirements.md#prompt-assembly)), because the design’s stated assembly order may conflict with scope language that places retrieved context toward the tail after tool-related blocks. The design also omits an explicit **risks and trade-offs** section expected by the review checklist, and **REQ-13.012**’s tool-instruction budget half is not mirrored by a dedicated acceptance criterion.

**Verdict:** Needs clarification

---

## 2. Strengths

### 2.1 Traceability and structure

- Full **requirement traceability table** covering REQ-13.001 through REQ-13.020 with concrete design anchors ([ep-system-design.md](ep-system-design.md) §Requirement traceability).
- Clear **module boundaries** table tying `cmd/pa`, config, new internal packages, and handler responsibilities to requirements ([ep-system-design.md](ep-system-design.md) §Architecture — Module boundaries).
- **Components and interfaces** table maps config types, loaders, index build/search, system prompt helpers, and handler extensions to specific requirements ([ep-system-design.md](ep-system-design.md) §Components and interfaces).

### 2.2 Architecture and operability

- **C4 C2** container diagram with PlantUML source supports onboarding and boundary discussion ([ep-system-design.md](ep-system-design.md) §Architecture).
- **Fail-fast** posture is explicit for config, skill parse, marker collisions, `indexTurn`, and `vec_skills` build failure ([ep-system-design.md](ep-system-design.md) §Error handling).
- **Testing strategy** explicitly references acceptance criteria IDs ([ep-system-design.md](ep-system-design.md) §Testing strategy), aligning with [ep-acceptance-criteria.md](ep-acceptance-criteria.md).

### 2.3 Security and injection awareness

- Centralized **canonical markers** and `TextContainsForbiddenMarkerLine` reuse for skills and memory indexing addresses injection and delimiter collision goals ([ep-system-design.md](ep-system-design.md) §Components and interfaces — `internal/promptmarkers`).
- **Trust policy** and structured wrapping are assigned to `internal/systemprompt` ([ep-system-design.md](ep-system-design.md) §Components and interfaces).

---

## 3. Issues and Recommendations

### 3.1 Critical

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| C1 | **Merged-system block order may violate scope and REQ-13.016** | [ep-scope.md](ep-scope.md) states retrieved context is placed in `RETRIEVED_CONTEXT` “at the tail of system” alongside `RUNTIME_SKILLS`; [REQ-13.016](ep-requirements.md#prompt-assembly) requires retrieved context and runtime skills to appear after the trust policy and tool instruction blocks. The design’s `systemprompt` assembly order is: “trust + personality, then RETRIEVED_CONTEXT, then TOOL_INSTRUCTIONS, then HERMES, then RUNTIME_SKILLS” ([ep-system-design.md](ep-system-design.md) §Components and interfaces), which places retrieved context **before** tool instruction and Hermes blocks. | Resolve ordering in one place: update the design to match scope + REQ-13.016 (and consistent interpretation of REQ-13.015), **or** amend scope/requirements to match the chosen order. Do not start implementation until product text and design agree. |

### 3.2 Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| M1 | **Missing “Risks and trade-offs” section** | The [07-system-design-review.skill.md](../../../ai-sdlc/specification/skills/07-system-design-review.skill.md) structural checklist expects risks and trade-offs; [ep-system-design.md](ep-system-design.md) has no such section. | Add a short section (e.g. embedding cost at startup, marker maintenance, budget eviction impact on tool availability, Hermes vs native parity) with mitigations. |
| M2 | **REQ-13.012 tool-instruction budget not acceptance-tested** | [REQ-13.012](ep-requirements.md#selection-and-tool-union) has two clauses (skill rune budget; tool-instruction aggregate eviction). [AC-13.014](ep-acceptance-criteria.md#ac-13-014) covers only the skill side. | Add an acceptance criterion (or explicitly defer in implementation plan with test obligation) for `max_tool_instruction_runes_per_turn` and deterministic eviction of vector-only tools. |
| M3 | **Duplicate skill identity not traced in design** | [ep-scope.md](ep-scope.md) requires fail-fast on “duplicate skill identity rules” at startup; the numbered requirements in [ep-requirements.md](ep-requirements.md) do not isolate this as its own REQ, and [ep-system-design.md](ep-system-design.md) does not mention duplicate IDs or duplicate frontmatter names. | Confirm whether duplicate directory names or duplicate stable ids are in scope; if yes, add a requirement line + design anchor (e.g. in `LoadDir` or validation pass). |

### 3.3 Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| N1 | **E2E boundary wording** | Design notes integration tests may substitute for Telegram ([ep-system-design.md](ep-system-design.md) §Testing strategy); [ep-scope.md](ep-scope.md) success criteria allow “test double equivalent.” | In the implementation plan, name the single authoritative entry point (per REQ-13.020 / scope) to avoid ambiguity during `./bin/validate EP-013`. |
| N2 | **`Covers AC-13.*` reference** | Traceability cites “Test files … `Covers AC-13.*`” ([ep-system-design.md](ep-system-design.md) §Requirement traceability — REQ-13.020). | Ensure repository convention for AC tags is applied consistently when tests land (no change to design text required if already standard). |

---

## 4. Architectural Decisions

### 4.1 Justified trade-offs

| Decision | Justification |
|----------|---------------|
| **Dedicated `vec_skills` in the same SQLite file** | Matches REQ-13.008, reuses embedder/dimension assumptions, keeps deployment single-DB. |
| **New packages vs bloating handler** | Splits parse/validate (`runtimeskills`), index (`skillindex`), markers (`promptmarkers`), and prompt text (`systemprompt`) for testability and clear boundaries. |
| **`config.Config` derived slice for packages** | Keeps runtime structs out of serialized config while enabling startup validation and index build. |
| **Handler builds system once per user turn** | Directly implements REQ-13.017 and supports AC-13.012-style assertions. |

### 4.2 Potential improvements (post-MVP)

1. Hot-reload or admin-triggered skill reload (explicitly out of scope for this epic per [ep-scope.md](ep-scope.md)).
2. Optional loading of `references/` or skill `scripts/` behind separate epics and threat modeling.
3. Metrics for skill selection, budget evictions, and index rebuild duration for operations.

---

## 5. NFR Coverage

| NFR | Coverage | Status |
|-----|----------|--------|
| REQ-13.019 (actionable startup errors) | Config and parse error wrapping; tool/skill path identification ([ep-system-design.md](ep-system-design.md) §Error handling) | OK |
| REQ-13.020 (tests + E2E/integration) | Unit/integration/E2E notes and AC references ([ep-system-design.md](ep-system-design.md) §Testing strategy) | Needs work — tie-break E2E vs mock-only path in implementation plan; close AC gap for tool-instruction budget (see M2) |
| Security / injection (via FR markers + trust policy) | Markers module, `indexTurn` guard, trust prefix ([ep-system-design.md](ep-system-design.md) §Components and interfaces, §Error handling) | OK — pending resolved block order (C1) so policy placement stays correct |
| Observability | Not expanded beyond existing logging mention in REQ-13.018; design does not add new logging fields | ⚠️ Optional improvement — consider structured fields for skill selection and eviction in implementation plan |

---

## 6. Project Rules Compliance

| Rule | Compliance |
|------|------------|
| KISS | ✅ Focused MVP: SKILL.md only, no scripts/references, clear package split |
| Fail fast | ✅ Startup validation, no partial registry, index build failure exits |
| Security | ⚠️ Strong marker and memory guard story; **block order (C1)** must match trust/tail policy before sign-off |
| Testability | ✅ Interfaces (`SkillIndex`), mock embedder, handler tests; AC gap on tool budget (M2) |

---

## 7. Summary

**Needs clarification** with action items:

1. **Resolve merged-system block order** — Align [ep-system-design.md](ep-system-design.md) assembly sequence with [ep-scope.md](ep-scope.md), [REQ-13.015](ep-requirements.md#prompt-assembly), and [REQ-13.016](ep-requirements.md#prompt-assembly); update design or requirements so they are mutually consistent (addresses C1).
2. **Add risks and trade-offs** — Satisfy review checklist and capture operator-facing impacts (addresses M1).
3. **Close REQ-13.012 test coverage** — Add AC or explicit test plan entry for tool-instruction rune budget and vector-only eviction (addresses M2).
4. **Clarify duplicate skill identity** — Map scope bullet to requirements and design validation (addresses M3).

---

## Traceability

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)
