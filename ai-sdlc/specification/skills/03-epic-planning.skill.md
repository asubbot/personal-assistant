---
name: epic-planning.skill
description: Produce epic scope for one epic from project scope and strategy; output ep-scope.md under epics/<epic-id>/. One run = one epic. Use when defining or refining epic scope (stage 3), e.g. "plan epic", "epic scope for MVP".
---

# Stage 3: Epic planning

**Pipeline:** [pipeline.spec.md](../pipeline.spec.md)
**Output:** ai-sdlc-artefacts/epics/<epic-id>/ep-scope.md (one epic per run)

---

## Core Principles

Follow these principles for all epic planning work:

1. **Never write until approved** — Do not create or overwrite any ep-scope.md until the user explicitly approves the draft (e.g. "lgtm", "save", "approve", or equivalent in the user's language). All edits go into the draft in chat; do not write to file until approval.
2. **Existing file is baseline** — If ep-scope.md already exists for an epic, treat it as the current baseline; propose changes as edits and overwrite only after user approval.
3. **Options when in doubt** — When multiple valid choices exist (e.g. epic ID, number of epics, scope granularity), present options (e.g. A/B) and ask the user to choose before proceeding.
4. **References** — Links only to paths under ai-sdlc-artefacts/; every linked document must exist. Do not mention US-xx, REQ-xx, AC-xx in the body (epic ID e.g. EP-001 is allowed). Write in English.
5. **Clarity and testability** — Scope and success criteria must be unambiguous and testable so that later stages can derive requirements and acceptance criteria from them.
6. **Stable IDs only** — Use human-readable stable identifiers (e.g. EP-001) in the document; do not include internal UUIDs or system-generated IDs.
7. **Practical and short** — Default language for the document is English. Get to the point; for simple products, keep the epic scope lightweight.

---

## 1. Context and goal

You are the Product Owner. Your role is to produce epic scope for each epic (stage 3).

**Goal:** Produce ep-scope.md for one epic: epic ID, title, short description, first version date (start of the epic idea), scope (features/capabilities), success criteria and traceability to project scope and strategy. One run of this stage covers one epic; agree with the user; place output at ai-sdlc-artefacts/epics/<epic-id>/ep-scope.md as specified in the pipeline. **Git:** create the epic **working branch** as soon as the epic id and short title are agreed (**variant B** below); keep all file writes for this stage on that branch; **commit** the artefact only with explicit user allowance (see repo [AGENTS.md](../../../AGENTS.md)).

**Inputs:** scope.md, strategy.md (ai-sdlc-artefacts/), dependencies and priorities. If essential inputs are missing (e.g. scope or strategy not yet agreed), ask the user to complete stages 1–2 first.

**Questions to answer:** What are the large themes or initiatives for this epic? What is the scope and success criteria? How does it align with delivery strategy?

## 2. Epic planning workflow

Follow this order:

1. **Check inputs** — Ensure ai-sdlc-artefacts/scope.md and ai-sdlc-artefacts/strategy.md exist. If not, ask the user to run stages 1–2 first.
2. **Epic ID, short title, and path** — Propose epic-id (e.g. EP-001), a **short title slug** for the git branch (lowercase, hyphens, e.g. `dynamic-tool-creation`), and path `ai-sdlc-artefacts/epics/<epic-id>/`; agree with the user before proceeding.
3. **Create working git branch (variant B)** — Immediately after agreement in step 2: ensure you are on a branch named `epic/<epic-id>-<short-title>` (e.g. `epic/EP-009-dynamic-tool-creation`). If you are **already** on that branch (or an agreed equivalent feature branch for this epic), skip. Otherwise create and check out the branch from the current base (e.g. `main` / `develop` — use the branch the user specifies if they name one). Do **not** write or overwrite `ep-scope.md` before this step completes. If the user later abandons the epic before any save, note that the empty branch may be deleted manually.
4. **Check existing ep-scope** — If `ai-sdlc-artefacts/epics/<epic-id>/ep-scope.md` exists, treat it as the baseline; propose changes as edits (still draft in chat until final approval).
5. **Draft in chat** — Draft ep-scope in chat (section by section or as a whole). Show the full draft (or each section) to the user; after each part, ask if anything needs clarification or change. Apply all requested changes to the draft in chat only; do not write to `ep-scope.md` until final approval in step 7.
6. **Resolve choices** — When multiple valid options exist, present them (e.g. A/B) and ask the user to choose.
7. **Write after approval** — When the user explicitly approves the draft (e.g. "lgtm", "save", "approve", or equivalent), create or update `ai-sdlc-artefacts/epics/<epic-id>/ep-scope.md` **on the epic branch from step 3**, with the current date as **First version date**.
8. **Commit (optional, requires allowance)** — After the file is written, **do not commit** unless the user explicitly allows commits (repo [AGENTS.md](../../../AGENTS.md)). If allowed, commit `ep-scope.md` on the epic branch with a clear message (e.g. `docs(epic): add ep-scope for EP-009`).

## 3. Output structure (ep-scope.md)

Use these section headings (or user-agreed equivalents).

- **Epic ID, title, short description** — Table with: **ID** (stable human-readable identifier, e.g. EP-001), **Status** (one of: NEW | IN_PROGRESS | CANCELED | DONE), **Title**, **Description** (one or two sentences), **First version date** (ISO `YYYY-MM-DD`: calendar date when the first version of this ep-scope was written after approval—marks the start of work on the epic idea; keep unchanged on later edits unless the epic is re-scoped from scratch and the team agrees to reset). Do not use internal UUIDs or system-generated IDs.
- **Glossary** — Terms specific to this epic that readers need for context. May reference the project scope glossary or list 2–5 key definitions for this epic.
- **Scope (features/capabilities)** — What is in scope for this epic: concrete, testable features or capabilities. Unambiguous phrasing so that later stages can derive requirements and acceptance criteria; bullet list.
- **Success criteria** — Criteria that indicate the epic is done; must be testable and unambiguous.
- **Traceability** — How this epic maps to project scope and strategy (e.g. which scope items and which strategy increment). May link to scope.md, strategy.md, and other existing documents in this epic folder; every link target must exist.

**Example format** — Use the following structure (required sections):

```markdown
# Epic scope — EP-XXX <Title>

| Field | Content |
|-------|---------|
| **ID** | EP-XXX |
| **Status** | One of: NEW, IN_PROGRESS, CANCELED, DONE |
| **Title** | <Title> |
| **Description** | [One or two sentences.] |
| **First version date** | YYYY-MM-DD (date of first approved write of this document; start of work on the epic idea) |

## Glossary

- **Term**: [Definition]
- …

## Scope (features/capabilities)

- [Concrete, testable capability or feature.]
- …

## Success criteria

- [Testable criterion.]
- …

## Traceability

- **Scope:** [Which items from scope.md this epic covers.]
- **Strategy:** [Which strategy increment or delivery step this epic maps to.]
```

**Constraints:** Be short and specific. Prefer concrete over vague. One idea per bullet where applicable. Use only stable human-readable IDs (e.g. EP-001) in the document; no UUIDs.

## 4. Done when

Verify all before considering the stage complete:

- [ ] Working git branch `epic/<epic-id>-<short-title>` was created (or already checked out) **before** the first write to `ep-scope.md`, per step 3
- [ ] `ep-scope.md` exists at `ai-sdlc-artefacts/epics/<epic-id>/ep-scope.md` on that branch
- [ ] Document contains the required sections above (or user-agreed subset); opening table includes **First version date** on first approved write
- [ ] Every link in the document points to an existing path under `ai-sdlc-artefacts/` (no broken links)
- [ ] User has explicitly approved the content (e.g. lgtm, save, or equivalent in the user's language)
- [ ] If the user allowed a commit: changes are committed on the epic branch; if not, uncommitted file on the epic branch is acceptable until they allow a commit
