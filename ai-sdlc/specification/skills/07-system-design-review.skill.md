---
name: system-design-review
description: Review system design documents for SDLC epics (stage 7). Use when reviewing ep-system-design.md files, checking architecture quality, requirement traceability, or when the user asks for architecture or system design review.
---

# Stage 7: System design review

**Pipeline:** [pipeline.spec.md](../pipeline.spec.md)  
**Output:** ai-sdlc-artefacts/epics/<epic-id>/ep-system-design-review.md (see § Report Template below).

This skill guides systematic review of system design documents (`ep-system-design.md`) within the SDLC pipeline.

## Mandatory delegation (pipeline stage 7)

When this skill is run as **pipeline stage 7** for an epic, execution MUST follow [pipeline.spec.md](../pipeline.spec.md) **§3**:

- **If you are the orchestrator** (you just helped author `ep-system-design.md` or earlier stages): **do not** perform this review yourself in the same session. **Delegate** to a **subagent** (Cursor Task / equivalent) or start a **new chat** with fresh context and a one-line brief: epic id, paths under `ai-sdlc-artefacts/epics/EP-XXX/`, instruction to run this skill end-to-end.
- **If you are the delegated reviewer**: treat inputs as read-only; produce the review for the user; write `ep-system-design-review.md` only after explicit user approval to save, per this skill.

## When to Use

- Reviewing `ai-sdlc-artefacts/epics/EP-XXX/ep-system-design.md`
- User asks for architecture or design review
- Before implementation planning (stage 8)

## Design–review iteration ([pipeline.spec.md](../pipeline.spec.md) §2.1)

Stages **6** and **7** repeat until **zero** open findings in **Blocker**, **Major**, **Medium**, and **Minor**, or until the **operator decides** after the cap.

1. **Count iterations** — Each completed save of a **`## Review iteration N`** section in `ep-system-design-review.md` is one stage 7 iteration. **N** must not exceed **5** without an explicit operator decision recorded in chat or in the review file (e.g. under the latest iteration).
2. **Single file** — Use one `ep-system-design-review.md` per epic. For iteration **N**, add (or replace only if the user agrees to discard a draft) a **top-level** heading `## Review iteration N` with a stable increasing **N**. **Retain** all prior `## Review iteration …` sections for history.
3. **Exit loop** — After this iteration’s findings are recorded, set **`Iteration summary — open counts`** for Blocker / Major / Medium / Minor. If all four are **zero**, the iteration loop is **complete**; stage 8 may follow.
4. **Cap** — If **N = 5** and any **Blocker**, **Major**, **Medium**, or **Minor** count is still **> 0**, **stop**: list remaining issues and require an **operator decision** before further stage 6/7 work or stage 8.
5. **Return to stage 6** — When Blocker/Major/Medium/Minor > 0 and **N < 5**, the orchestrator runs **stage 6** again to revise `ep-system-design.md`, then runs **stage 7** again (new **delegated** session per pipeline §3).

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

Categorize every finding into **one** severity (definitions align with pipeline **§2.1** exit counts):

| Severity | Criteria |
|----------|----------|
| **Blocker** | Missing requirement coverage, unacceptable security gap, data loss or integrity risk, design that cannot meet must-have REQ/AC |
| **Major** | Wrong or missing component/contract, traceability break, missing error-handling strategy for required flows, testability blocker |
| **Medium** | Unclear interfaces, incomplete non-critical specs, inconsistent structure, gaps that should be fixed before implementation |
| **Minor** | Documentation polish, optional consistency, low-risk improvements |

**Loop exit:** **Blocker = 0 AND Major = 0 AND Medium = 0 AND Minor = 0** (only open items in this iteration; resolved items from prior iterations do not need re-listing unless regressed).

### Step 6: Output Report

Generate or update **ep-system-design-review.md** in the same epic folder as `ep-system-design.md` (unless the user agrees another path). Add **`## Review iteration N`** per **Design–review iteration** above. Follow the template below. On first review, **N = 1**; on subsequent passes after stage 6 fixes, increment **N** (max **5** without operator decision).

---

## Report Template

Use **one** `ep-system-design-review.md` per epic. **First** iteration: create the file with the document title once. **Later** iterations: **append** a new top-level `## Review iteration N` block at the end; **do not remove** prior iteration sections.

```markdown
# Architecture Review — EP-XXX [optional title]

**Reviewer:** [AI Agent / Name]

---

## Review iteration N

**Review date:** YYYY-MM-DD
**Stage 7 iteration:** N of max 5
**Document reviewed:** [ep-system-design.md](ep-system-design.md)
**Iteration summary — open counts:** Blocker: X | Major: X | Medium: X | Minor: X
**Gate:** Pass (Blocker/Major/Medium/Minor all zero) | Fail (any Blocker/Major/Medium/Minor > 0) | Cap (N = 5 and Blocker/Major/Medium/Minor still > 0 — operator decision required)

### Overall assessment

[2–3 sentences for this iteration]

**Verdict:** [Pass gate / Fail gate / Cap — stop for operator]

### Strengths

- [Specific strength with reference]

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

### Architectural decisions (optional)

| Decision | Justification |
|----------|---------------|
| | |

### NFR coverage (optional)

| NFR | Coverage | Status |
|-----|----------|--------|
| | | OK / Needs work |

### Project rules compliance (optional)

| Rule | Compliance |
|------|------------|
| KISS | ✅ / ❌ / ⚠️ |
| Fail fast | ✅ / ❌ / ⚠️ |
| Security | ✅ / ❌ / ⚠️ |
| Testability | ✅ / ❌ / ⚠️ |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)
```

---

## Checklist

Before completing review:
- [ ] Iteration number **N** is set (1–5); prior iteration sections preserved when **N > 1**
- [ ] **Iteration summary — open counts** filled for Blocker / Major / Medium / Minor
- [ ] Gate reflects **§2.1** (Pass only if Blocker/Major/Medium/Minor all zero; Cap if **N = 5** and any of those > 0)
- [ ] All requirements have traceability entries for this iteration
- [ ] Blocker / Major / Medium / Minor issues have clear recommendations
- [ ] Report follows template structure under `## Review iteration N`
- [ ] Severity ratings are consistent with the definitions in Step 5
- [ ] Action items are specific and actionable
