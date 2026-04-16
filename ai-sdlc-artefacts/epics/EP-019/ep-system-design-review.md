# Architecture Review — EP-019 Scheduled Agent Jobs and Legacy Scheduler Replacement

**Reviewer:** Delegated AI Reviewer (Stage 7)

---

## Review iteration 1

**Review date:** 2026-04-16  
**Stage 7 iteration:** 1 of max 5  
**Document reviewed:** [ep-system-design.md](ep-system-design.md)  
**Iteration summary — open counts:** Blocker: 0 | Major: 1 | Medium: 2 | Minor: 1  
**Gate:** Fail (any Blocker/Major/Medium/Minor > 0)

### Overall assessment

The design is coherent and practical for EP-019 goals: it introduces a dedicated scheduling model, explicit operator command flow, and clear data-store separation with `jobs.sqlite`. The main architecture and error/test sections are present and usable for implementation planning.  
However, traceability depth and several structural/contract details are incomplete for Stage 7 gate criteria.

**Verdict:** Fail gate

### Strengths

- Clear component boundaries for scheduler, management commands, execution, and delivery.
- Strong operational decision to separate job data from vector index storage.
- Good baseline test strategy across unit/integration/E2E levels.

### Issues and recommendations

#### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| — | None | — | — |

#### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| 1 | Requirement traceability table lacks full verification depth | Current table maps REQ to short design phrases only | Expand traceability to include design component(s), interface or flow reference, and explicit AC mapping per REQ |

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| 1 | Missing `Risks and trade-offs` section | Structural checklist for stage 7 requires this section | Add dedicated section with key risks, trade-offs, mitigations, and residual risk notes |
| 2 | Startup readiness gate for management commands is implicit | REQ-19.002 requires loading jobs before accepting management commands | Define explicit readiness contract and deterministic response for pre-ready command attempts |

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| 1 | REQ-19.022 profile-based responsiveness contract is under-specified | Testing strategy mentions profile validation without threshold-source details | Add short subsection defining profile source, metrics, and acceptance binding |

### Architectural decisions (optional)

| Decision | Justification |
|----------|---------------|
| Dedicated SQLite file for scheduling data | Isolates scheduler state from vector storage and reduces cross-domain coupling |
| Two-step delete with confirmation token | Reduces accidental destructive operations |

### Project rules compliance (optional)

| Rule | Compliance |
|------|------------|
| KISS | ✅ |
| Fail fast | ⚠️ |
| Security | ⚠️ |
| Testability | ✅ |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)

---

## Review iteration 2

**Review date:** 2026-04-16  
**Stage 7 iteration:** 2 of max 5  
**Document reviewed:** [ep-system-design.md](ep-system-design.md)  
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0  
**Gate:** Pass (Blocker/Major/Medium/Minor all zero)

### Overall assessment

The iteration-2 design satisfies the stage-7 structural checklist and now includes all required sections, including `Risks and trade-offs`, explicit startup readiness behavior, and a concrete REQ-19.022 responsiveness contract.
Requirement traceability is complete for REQ-19.001 through REQ-19.022 with explicit design components, interface/flow references, and AC alignment per requirement.
Architecture quality is acceptable for EP-019 scope: boundaries are clear, fail-fast behavior is defined for startup/config/authorization paths, security controls required by scope are present, and testability is covered across unit/integration/E2E/profile levels.

**Verdict:** Pass gate

### Strengths

- Structural completeness is full against stage-7 checklist (`Overview`, architecture diagram, module boundaries, components/interfaces, data models, error handling, testing strategy, risks/trade-offs, traceability table).
- Traceability table has full depth for each REQ with component + contract/flow + AC linkage.
- Startup readiness gate is explicit and deterministic (`scheduler initializing`) and directly supports REQ-19.002.
- REQ-19.022 is concretely specified via profile-threshold contract and mapped to AC-19.022.
- Architecture remains practical while preserving isolation of scheduling data in `jobs.sqlite`.

### Issues and recommendations

#### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| — | None | — | — |

#### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| — | None | — | — |

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| — | None | — | — |

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| — | None | — | — |

### Project rules compliance (optional)

| Rule | Compliance |
|------|------------|
| KISS | ✅ |
| Fail fast | ✅ |
| Security | ✅ |
| Testability | ✅ |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)
