---
name: implementation-planning.skill
description: Produce implementation plan per story (stage 8); output st-implementation-plan.md. Use when planning tasks for a user story, e.g. "implementation plan for this story", "tasks for US-08", "break down story into tasks".
---

# Stage 8: Implementation planning

**Pipeline:** [pipeline.spec.md](../pipeline.spec.md)  
**Output:** ai-sdlc-artefacts/epics/<epic-id>/stories/<story-id>/st-implementation-plan.md

---

## Core Principles

Follow these principles for all implementation planning work:

1. **Never write until approved** — Do not create or overwrite st-implementation-plan.md until the user explicitly approves the draft (e.g. "lgtm", "save", "approve"). All edits go into the draft in chat; do not write to file until approval.
2. **Existing file is baseline** — If st-implementation-plan.md already exists for the story, treat it as the current baseline; propose changes as edits and overwrite only after user approval.
3. **Options when in doubt** — When multiple valid choices exist (e.g. task granularity, ordering, checkpoint placement), present options (e.g. A/B) and ask the user to choose before proceeding.
4. **References** — Links only to paths under `ai-sdlc-artefacts/`; every linked document must exist. Write in English.
5. **Practical and short** — Get to the point. Be practical above all. Be short and specific. Only include tasks that can be performed by a coding agent. Each task must have a verification criterion.
6. **Legacy** — Do not modify content under `legacy` folders; use as reference only.

---

## 1. Context and goal

You are the Tech Lead for this epic. Your role is to produce the implementation plan per story (stage 8).

**Goal:** Produce st-implementation-plan.md: ordered tasks with dependencies, verification per task, traceability to REQ and AC (from the story's st-scope), checkpoints, and optional parallel work indication.

**Inputs:** st-scope.md (ai-sdlc-artefacts/epics/<epic-id>/stories/<story-id>/; includes Traceability to AC/REQ table), ep-system-design.md (ai-sdlc-artefacts/epics/<epic-id>/), and test strategy (e.g. strategy.md under ai-sdlc-artefacts/). If st-scope is missing, ask the user to run stage 7 (User story planning) first.

**Questions to answer:** What are the discrete coding steps? In what order do we execute them? Where do we place checkpoints and how do we verify each step?

---

## 2. Implementation planning workflow

Follow this order:

1. **Check inputs** — Ensure st-scope.md exists for the story. If not, ask the user to run stage 7 first. Optionally use ep-system-design.md (epic) and test strategy (e.g. strategy.md) as reference.
2. **Check existing st-implementation-plan** — If st-implementation-plan.md exists for the story, treat it as the baseline; propose changes as edits.
3. **Draft in chat** — Draft the implementation plan in chat (task list, verification, checkpoints). Show it to the user and ask for clarification or changes. Do not write to file yet.
4. **Resolve choices** — When multiple valid options exist (e.g. task breakdown, ordering), present options (e.g. A/B) and ask the user to choose.
5. **Write after approval** — Create or update ai-sdlc-artefacts/epics/<epic-id>/stories/<story-id>/st-implementation-plan.md only when the user explicitly approves (e.g. "lgtm", "save", "approve").
6. **Legacy** — Do not modify content under `legacy` folders; use as reference only.

---

## 3. Output structure (st-implementation-plan.md)

Use these elements (or user-agreed equivalents):

- **Story reference** — Link to st-scope.md for this story (and optionally story ID, e.g. US-08).
- **Task list** — Numbered tasks with dependencies (e.g. "Task 2 depends on Task 1"). Clear objective per task; sub-bullets for details.
- **Verification per task** — For each task, state how to confirm the step is done (e.g. test passes, build succeeds, review done).
- **Checkpoints** — Explicit checkpoints (e.g. "Ensure all tests pass before proceeding"; "Ask the user if questions arise").
- **Traceability** — Each task that implements scope MUST reference REQ and AC where applicable. Use the REQ/AC from the story's st-scope (Traceability to AC/REQ table). Use "—" when a task has no direct link (e.g. checkpoints, tooling).

**Task format:**

- One clear objective per task.
- Sub-bullets for technical or procedural details.
- Traceability block: **Story:** &lt;story-id&gt;; **REQ:** REQ-XXX (link to ep-requirements if useful); **AC:** AC-XXX (link to ep-acceptance-criteria if useful). Use "—" when the task is a checkpoint or has no direct REQ/AC.

**Quality:** Tasks must be actionable by a coding agent. Each task has a verification criterion. Keep the plan short and specific.

**Example** (one task):

```markdown
### Task 1. Add LLM provider interface

- Define `LLMProvider` interface in package `internal/llm`.
- Add constructor that reads config and returns implementation.

**Verification:** `go build ./...` passes; new type visible in package.

**Traceability:** Story: US-08; REQ: REQ-008; AC: AC-015, AC-016.
```

---

## 4. Done when

Verify all before considering the stage complete:

- [ ] st-implementation-plan.md exists at ai-sdlc-artefacts/epics/<epic-id>/stories/<story-id>/st-implementation-plan.md
- [ ] Document contains story reference, task list (numbered, with dependencies), verification per task, and checkpoints
- [ ] Each task that implements scope has traceability to REQ/AC from the story's st-scope (or "—" for checkpoints)
- [ ] Every link in the document points to an existing path under `ai-sdlc-artefacts/` (no broken links)
- [ ] The user has explicitly approved the plan
