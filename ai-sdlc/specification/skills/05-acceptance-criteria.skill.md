---
name: acceptance-criteria.skill
description: Produce epic acceptance criteria from ep-scope and ep-requirements; output ep-acceptance-criteria.md. Use when defining or refining epic acceptance criteria (stage 5), e.g. "acceptance criteria for this epic", "AC from requirements".
---

# Stage 5: Acceptance criteria

**Pipeline:** [pipeline.spec.md](../pipeline.spec.md)  
**Output:** ai-sdlc-artefacts/epics/<epic-id>/ep-acceptance-criteria.md

---

## Core Principles

Follow these principles for all acceptance criteria work:

1. **Never write until approved** — Do not create or overwrite ep-acceptance-criteria.md until the user explicitly approves the draft (e.g. "lgtm", "save", "approve"). All edits go into the draft in chat; do not write to file until approval.
2. **Existing file is baseline** — If ep-acceptance-criteria.md already exists for the epic, treat it as the current baseline; propose changes as edits and overwrite only after user approval.
3. **Options when in doubt** — When multiple valid choices exist (e.g. AC granularity, Gherkin vs alternative format), present options (e.g. A/B) and ask the user to choose before proceeding.
4. **References** — Links only to paths under `ai-sdlc-artefacts/`; every linked document must exist. Keep traceability to ep-requirements. Write in English.
5. **Stable IDs only** — Use stable human-readable AC IDs (e.g. AC-001); do not use internal UUIDs.
6. **Practical and short** — Get to the point. Be practical above all. Be short and specific.
7. **Legacy** — Do not modify content under `legacy` folders; use as reference only.

---

## 1. Context and goal

You are an experienced QA / acceptance criteria analyst. Your role is to produce the epic acceptance criteria document (stage 5).

**Goal:** Produce ep-acceptance-criteria.md: testable conditions for the epic in Gherkin (Given/When/Then) or equivalent, with AC ID and traceability to REQ. This output is the input for system design (stage 7) and user story planning (stage 6); story-level AC are derived from it later.

**Inputs:** ep-scope.md and ep-requirements.md (ai-sdlc-artefacts/epics/<epic-id>/). If either is missing, ask the user to run stage 3 (Epic planning) or stage 4 (Requirements) first.

**Questions to answer:** When is this epic "done" from a test perspective? What scenarios (Given/When/Then) cover the requirements? How do AC map to REQ?

---

## 2. Acceptance criteria workflow

Follow this order:

1. **Check inputs** — Ensure ep-scope.md and ep-requirements.md exist for the epic. If not, ask the user to run stage 3 or 4 first.
2. **Check existing ep-acceptance-criteria** — If ep-acceptance-criteria.md exists for the epic, treat it as the baseline; propose changes as edits.
3. **Draft in chat** — Draft acceptance criteria in chat (section by section or by block). Show each part to the user and ask for clarification or changes. Apply edits only in the draft in chat; do not write to file yet.
4. **Resolve choices** — When multiple valid options exist (e.g. AC granularity), present options (e.g. A/B) and ask the user to choose.
5. **Write after approval** — Create or update ai-sdlc-artefacts/epics/<epic-id>/ep-acceptance-criteria.md only when the user explicitly approves (e.g. "lgtm", "save", "approve").
6. **Legacy** — Do not modify content under `legacy` folders; use as reference only.

---

## 3. Output structure (ep-acceptance-criteria.md)

Use these section headings (or user-agreed equivalents).

- **Introduction** — Brief summary of the epic and purpose of this document.
- **Acceptance criteria index** — Table: AC ID (link to AC in document), REQ (link to ep-requirements.md section), Summary (one-line). Required.
- **Acceptance criteria** — List of AC: ID (AC-001, …), formulation in Gherkin (Given/When/Then) or equivalent, traceability to REQ (links to ep-requirements.md).

**Gherkin:** Prefer Given/When/Then. Scenario order: happy path first, then negative path, alternative flows, edge cases.

**Traceability:** Every AC must trace to one or more REQ from ep-requirements.md. In the index, the REQ column must use links to the requirement section (e.g. [REQ-001](ep-requirements.md#interface-and-deployment)).

**Quality:** Each AC must be testable; one clear scenario per AC; short and specific; no vague wording.

**Example** — One AC in Gherkin with REQ link:

```markdown
**AC-001** (Trace: REQ-001)
Given the user is logged in
When the user requests the dashboard
Then the system SHALL display the summary widget within 2 seconds
```

---

## 4. Done when

Verify all before considering the stage complete:

- [ ] ep-acceptance-criteria.md exists at ai-sdlc-artefacts/epics/<epic-id>/ep-acceptance-criteria.md
- [ ] Document contains **Introduction** (epic summary and document purpose), **Acceptance criteria index** (AC ID | REQ with links | Summary), and **Acceptance criteria** (AC-XXX with Gherkin or equivalent and traceability to REQ)
- [ ] Every link in the document points to an existing path under `ai-sdlc-artefacts/` (no broken links)
- [ ] Every AC traces to at least one REQ from ep-requirements.md
- [ ] User has explicitly approved the content
