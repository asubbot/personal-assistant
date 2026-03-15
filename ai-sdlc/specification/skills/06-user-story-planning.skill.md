---
name: user-story-planning.skill
description: Produce story scope per user story (stage 6); output st-scope.md with traceability to AC/REQ. Use when planning user stories for an epic, e.g. "user stories for this epic", "slice stories", "story scope".
---

# Stage 6: User story planning

**Pipeline:** [pipeline.spec.md](../pipeline.spec.md)  
**Output:** ai-sdlc-artefacts/epics/<epic-id>/stories/<story-id>/st-scope.md

---

## Core Principles

Follow these principles for all user story planning work:

1. **Never write until approved** — Do not create or overwrite st-scope.md until the user explicitly approves the draft (e.g. "lgtm", "save", "approve"). All edits go into the draft in chat; do not write to file until approval.
2. **Existing files are baseline** — If st-scope.md already exists for a story, treat it as the current baseline; propose changes as edits and overwrite only after user approval.
3. **Options when in doubt** — When multiple valid choices exist (e.g. story boundaries, story ID format), present options (e.g. A/B) and ask the user to choose before proceeding.
4. **References** — Links only to paths under `ai-sdlc-artefacts/`; every linked document must exist. Keep traceability to ep-requirements. Write in English.
5. **Stable story IDs** — Use stable human-readable story IDs (e.g. ST-001); do not use internal UUIDs.
6. **Traceability in st-scope** — In st-scope.md do not duplicate the full text of acceptance criteria; provide traceability via the required **Traceability to AC/REQ** table (links only). The full criterion text stays in ep-acceptance-criteria.md.
7. **Practical and short** — Get to the point. Be practical above all. Be short and specific.
8. **Legacy** — Do not modify content under `legacy` folders; use as reference only.

---

## 1. Context and goal

You are an expert requirements analyst. Your role is to produce story scope per user story (stage 6).

**Goal:** For each user story, produce **st-scope.md** (story ID, title, formulation, Traceability to AC/REQ table, optionally traceability to ep-system-design). REQ traceability is covered by the REQ column in the table. Slice stories along design boundaries (stage 7 runs before this stage).

**Inputs:** ep-scope.md, ep-requirements.md, ep-acceptance-criteria.md, ep-system-design.md (ai-sdlc-artefacts/epics/<epic-id>/), and stakeholder input. If any of the four is missing, ask the user to run the corresponding prior stage first.

**Questions to answer:** Who wants what and why? What is the scope of each story? How do stories trace to requirements?

---

## 2. User story formulation

- **Structure** — Each user story is formulated as **"As a [user role], I want [capability], so that [benefit]."** (Formulation field in st-scope.md).
- **Style** — Keep it concise and specific; add context that clarifies scope or constraints when needed (in Dependencies, Notes, or a dedicated subsection).
- **Traceability in st-scope** — st-scope.md must include a **Traceability to AC/REQ** table (AC | REQ | Summary) with links to ep-acceptance-criteria.md and ep-requirements.md; do not duplicate the full criterion text in st-scope.
- **Creating a story** — Each user story is represented by st-scope.md under `stories/<story-id>/`. Write to file only after user approval.

---

## 3. User story planning workflow

Follow this order:

1. **Check inputs** — Ensure all four epic artefacts exist (ep-scope.md, ep-requirements.md, ep-acceptance-criteria.md, ep-system-design.md). If any is missing, ask the user to run the prior stage first.
2. **Identify or agree story set** — Agree on the list of stories (story ID + short title). Slice along ep-system-design boundaries (components, interfaces). When unsure, propose slicing options and ask the user to choose.
3. **Draft in chat** — For each story, draft st-scope (including Traceability to AC/REQ table). Do not write to file yet.
4. **Resolve choices** — When multiple valid options exist (e.g. story boundaries, IDs), present options (e.g. A/B) and ask the user to choose.
5. **Write after approval** — Create or update ai-sdlc-artefacts/epics/<epic-id>/stories/<story-id>/st-scope.md only when the user explicitly approves (e.g. "lgtm", "save", "approve").
6. **Legacy** — Do not modify content under `legacy` folders; use as reference only.

**Slicing:** One slice of value per story. Align story boundaries with ep-system-design (components, interfaces).

---

## 4. Output structure

### 4.1 st-scope.md

Use these elements (or user-agreed equivalents):

- **Story ID, title** — Stable story ID and short title.
- **Formulation** — Strictly **"As a [user role], I want [capability], so that [benefit]."** Keep it concise and specific; add context for scope or constraints in the following fields if needed.
- **Traceability to AC/REQ** — **Required.** Table with columns **AC | REQ | Summary**: each row links one epic AC (ep-acceptance-criteria.md) to its REQ (ep-requirements.md) and a one-line summary. This table provides REQ traceability (no separate Traceability to REQ section). Do not duplicate the full criterion text in st-scope; links only.
- **Traceability to design** (optional) — Component or boundary in ep-system-design.md when relevant.
- **Dependencies, notes** (optional).

**Traceability table format:**

```markdown
| AC | REQ | Summary |
|----|-----|---------|
| [AC-XXX](path/to/ep-acceptance-criteria.md#ac-xxx) | [REQ-YYY](path/to/ep-requirements.md#section) | One-line summary |
```

**Example** (excerpt for one story):

```markdown
## Traceability to AC/REQ

| AC | REQ | Summary |
|----|-----|---------|
| [AC-015](../../ep-acceptance-criteria.md#ac-015) | [REQ-008](../../ep-requirements.md#llm-and-logging) | Config specifies LLM provider → core uses it without code change |
| [AC-016](../../ep-acceptance-criteria.md#ac-016) | [REQ-008](../../ep-requirements.md#llm-and-logging) | Provider switch in config + restart → new provider used |
```

---

## 5. Done when

Verify all before considering the stage complete:

- [ ] For each planned story, the folder ai-sdlc-artefacts/epics/<epic-id>/stories/<story-id>/ exists with st-scope.md
- [ ] Each st-scope.md contains Story ID, title, Formulation ("As a… I want… so that…"), and **Traceability to AC/REQ** (table: AC | REQ | Summary); no full criterion text duplicated in st-scope
- [ ] Every link in st-scope points to an existing path under `ai-sdlc-artefacts/` (no broken links)
- [ ] The user has explicitly approved the artefacts
