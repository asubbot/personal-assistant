# Architecture Review - EP-020 Natural-Language Scheduled Job Creation from Telegram

**Reviewer:** delegated AI agent

---

## Review iteration 1

**Review date:** 2026-04-16  
**Stage 7 iteration:** 1 of max 5  
**Document reviewed:** [ep-system-design.md](ep-system-design.md)  
**Iteration summary - open counts:** Blocker: 0 | Major: 0 | Medium: 3 | Minor: 2  
**Gate:** Fail

### Overall assessment

Design covered the intended feature direction and reused EP-019 runtime correctly, but had structural completeness and contract specificity gaps that reduced implementation/testability confidence.

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
| 1 | Missing module boundaries table | `ep-system-design.md` | Add explicit module boundaries with dependency rules |
| 2 | Underspecified creation interfaces | NL create flow contracts | Define parser/validation/response contract fields and outcomes |
| 3 | Observability drift vs scope | scope expects parsed payload in audit | Include parsed schedule payload in observability design |

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| 1 | Testing strategy lacked AC mapping | Testing section | Add compact AC-to-test mapping table |
| 2 | Risks lacked concrete mitigations | Risks section | Add mitigations and trigger metrics |

### Traceability (this iteration)

- Scope: [ep-scope.md](ep-scope.md)
- Requirements: [ep-requirements.md](ep-requirements.md)
- Acceptance criteria: [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- Design: [ep-system-design.md](ep-system-design.md)

---

## Review iteration 2

**Review date:** 2026-04-16  
**Stage 7 iteration:** 2 of max 5  
**Document reviewed:** [ep-system-design.md](ep-system-design.md)  
**Iteration summary - open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0  
**Gate:** Pass

### Overall assessment

All iteration-1 findings were addressed. Design now has module boundaries, explicit NL-create contracts, observability details, and AC-oriented testing strategy. The stage-7 gate is satisfied.

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

### Traceability (this iteration)

- Scope: [ep-scope.md](ep-scope.md)
- Requirements: [ep-requirements.md](ep-requirements.md)
- Acceptance criteria: [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- Design: [ep-system-design.md](ep-system-design.md)
