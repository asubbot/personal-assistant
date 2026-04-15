# Architecture Review — EP-018 Tiered Prompt Cost Reduction

**Pipeline:** Stage 7 ([pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md))  
**Epic:** [ep-scope.md](ep-scope.md) · [ep-requirements.md](ep-requirements.md) · [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-system-design.md](ep-system-design.md)

---

## Review iteration 1

**Review date:** 2026-04-15  
**Stage 7 iteration:** 1 of max 5  
**Document reviewed:** [ep-system-design.md](ep-system-design.md) (revision before post-review design edits)

### Iteration summary — open counts

| Severity | Count |
|----------|------:|
| Blocker  | 0 |
| Major    | 1 |
| Medium   | 2 |
| Minor    | 1 |

**Gate:** Fail (Blocker / Major / Medium / Minor not all zero)

### Overall assessment

The design gives a coherent end-to-end picture (tiers, classifier extension, `HandleMessage` branches, dynamic picker, config, logging, tests) and the requirement traceability table lists every REQ through REQ-18.021. The main gap is that the model-stage **prompt body** contract required by REQ-18.010 / AC-18.010 is not spelled out in the design text (only the single-token output is), which is a material specification hole before implementation.

**Verdict:** Fail gate

### Structural checklist results

| Checklist item | Result |
|----------------|--------|
| Overview with scope reference | Pass |
| Architecture diagram (C4 C2 or equivalent) | Pass |
| Module boundaries table | Pass |
| Components and interfaces table | Partial — subsections instead of a single consolidated table |
| Data models | Pass |
| Error handling | Pass |
| Testing strategy | Pass |
| Risks and trade-offs | Pass (present in reviewed revision after orchestrator follow-up) |
| Requirement traceability table | Pass |

### Requirement traceability verification (REQ-18.001 – REQ-18.021)

Each requirement is referenced in `ep-system-design.md` (body links and/or traceability table).

### Architecture quality

| Dimension | Assessment |
|-----------|------------|
| **KISS** | Strong |
| **Fail-fast** | Strong for config |
| **Security** | Adequate for this epic |
| **Testability** | Strong |

### Findings and recommendations

#### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| — | _None_ | — | — |

#### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| 1 | Model-stage prompt body underspecified vs REQ-18.010 / AC-18.010 | REQ-18.010 requires the classification request prompt body to contain **only** the user message and three tier labels with brief descriptions. | Extend ModelClassifier section with explicit classification request shape. |

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| 1 | No consolidated components/interfaces table | Skill Step 2 expects a table. | Add consolidated component table. |
| 2 | `always_include` cardinality vs cap not resolved | Truncation after merge without policy when `always_include` exceeds cap. | Document load-time rule: `max_tools_for_llm_request` ≥ distinct valid `always_include` count. |

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| 1 | HTML entities in traceability table | `&gt;` in markdown source. | Use prose or plain `<` / `>`. |

---

### Orchestrator follow-up (post iteration 1)

The following updates were applied to [ep-system-design.md](ep-system-design.md) to address iteration 1 findings: **classification request shape** for [REQ-18.010](ep-requirements.md#req-18-010), a **consolidated component table**, **load-time cardinality** `max_tools_for_llm_request` ≥ distinct valid `always_include` tool count, and **traceability wording** cleanup.

**Pipeline [§3](../../../ai-sdlc/specification/pipeline.spec.md):** iteration 2 below was recorded after a **delegated** reviewer pass on the updated design.

---

## Review iteration 2

**Review date:** 2026-04-15  
**Stage 7 iteration:** 2 of max 5  
**Document reviewed:** [ep-system-design.md](ep-system-design.md)

### Iteration summary — open counts

| Severity | Count |
|----------|------:|
| Blocker  | 0 |
| Major    | 0 |
| Medium   | 0 |
| Minor    | 0 |

**Gate:** Pass

### Overall assessment

`ep-system-design.md` satisfies the stage 7 structural checklist (overview with scope tie-in, C4 container diagram, module boundaries, consolidated components/interfaces table, configuration data model, error handling, risks and trade-offs, testing strategy, and full REQ traceability through REQ-18.021). Iteration 1 items are addressed in the current design text; no new Blocker–Minor findings were identified in this delegated pass.

### Iteration 1 resolution

| Iteration 1 finding | Status | Where addressed in current `ep-system-design.md` |
|---------------------|--------|-----------------------------------------------------|
| **Major 1** — model classification prompt body underspecified | **Resolved** | **ModelClassifier** — explicit three-part body: preamble (single token), three tier bullets with brief descriptions, user text in delimited block; excludes Hermes/session/RAG; aligns with REQ-18.010 / AC-18.010. |
| **Medium** — consolidated component table; `always_include` versus cap | **Resolved** | **Consolidated component table** (single table with REQ ids); **Dynamic tool picker** — merge `always_include` then truncate to N; **Cardinality rule** under configuration validation — `max_tools_for_llm_request` ≥ distinct valid `always_include` count so cap does not drop forced tools. |
| **Minor** — HTML entities | **Resolved** | Traceability table uses prose (“non-zero” / “zero”) instead of HTML entities. |

### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| — | _None_ | — | — |

### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| — | _None_ | — | — |

### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| — | _None_ | — | — |

### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| — | _None_ | — | — |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)
