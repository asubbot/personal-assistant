---
name: system-design-review
description: Review system design documents for SDLC epics (stage 7). Use when reviewing ep-system-design.md files, checking architecture quality, requirement traceability, or when the user asks for architecture or system design review.
---

# Stage 7: System design review

**Pipeline:** [pipeline.spec.md](../pipeline.spec.md)  
**Output:** ai-sdlc-artefacts/epics/<epic-id>/ep-system-design-review.md (see § Report Template below).

This skill guides systematic review of system design documents (`ep-system-design.md`) within the SDLC pipeline.

## When to Use

- Reviewing `ai-sdlc-artefacts/epics/EP-XXX/ep-system-design.md`
- User asks for architecture or design review
- Before implementation planning (stage 8)

## Review Workflow

### Step 1: Read Related Documents

Read in order:
1. `ep-scope.md` — understand feature scope and glossary
2. `ep-requirements.md` — verify all requirements are addressed
3. `ep-acceptance-criteria.md` — ensure testability alignment
4. `ep-system-design.md` — the document under review

### Step 2: Structural Check

Verify the design document contains:
- [ ] Overview with scope reference
- [ ] Architecture diagram (C4 C2 or equivalent)
- [ ] Module boundaries table
- [ ] Components and interfaces table
- [ ] Data models
- [ ] Error handling
- [ ] Testing strategy
- [ ] Risks and trade-offs
- [ ] Requirement traceability table

### Step 3: Requirement Traceability

For each requirement in `ep-requirements.md`:
- Verify explicit coverage in traceability table
- Confirm design component is identified
- Check acceptance criteria alignment

### Step 4: Architecture Quality

Assess:
- **KISS**: Is the solution as simple as possible?
- **Fail fast**: Are errors caught early with clear messages?
- **Security**: Are security controls adequate?
- **Testability**: Can components be tested in isolation?
- **Modularity**: Are boundaries clear and dependencies minimal?

### Step 5: Identify Issues

Categorize findings:

| Severity | Criteria |
|----------|----------|
| **Critical** | Missing requirement coverage, security gaps, data loss risk |
| **Medium** | Unclear contracts, missing error handling, incomplete specs |
| **Minor** | Documentation gaps, consistency issues, improvement suggestions |

### Step 6: Output Report

Generate **ep-system-design-review.md** in the same epic folder as `ep-system-design.md` (unless the user agrees another path). Follow the template below.

---

## Report Template

```markdown
# Architecture Review — EP-XXX [Title]

**Review date:** YYYY-MM-DD
**Reviewer:** [AI Agent / Name]
**Document reviewed:** [ep-system-design.md](ep-system-design.md)

---

## 1. Overall Assessment

[2-3 sentences summarizing quality and readiness]

**Verdict:** [Ready / Needs clarification / Not ready]

---

## 2. Strengths

### 2.1 [Category]
- [Specific strength with line reference]

---

## 3. Issues and Recommendations

### 3.1 Critical

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| C1 | [Description] | [Line X: quote] | [Action] |

### 3.2 Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

### 3.3 Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

---

## 4. Architectural Decisions

### 4.1 Justified Trade-offs

| Decision | Justification |
|----------|---------------|
| [Decision] | [Why it's acceptable] |

### 4.2 Potential Improvements (post-MVP)

1. [Improvement]

---

## 5. NFR Coverage

| NFR | Coverage | Status |
|-----|----------|--------|
| [REQ-XX.XXX] | [How addressed] | OK / Needs work |

---

## 6. Project Rules Compliance

| Rule | Compliance |
|------|------------|
| KISS | ✅ / ❌ / ⚠️ |
| Fail fast | ✅ / ❌ / ⚠️ |
| Security | ✅ / ❌ / ⚠️ |
| Testability | ✅ / ❌ / ⚠️ |

---

## 7. Summary

**[Ready status]** with action items:

1. **[Action]** — [Details]
2. **[Action]** — [Details]

---

## Traceability

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)
```

---

## Checklist

Before completing review:
- [ ] All requirements have traceability entries
- [ ] Critical issues have clear recommendations
- [ ] Report follows template structure
- [ ] Severity ratings are consistent
- [ ] Action items are specific and actionable
