# Stage 6: User story planning

**Pipeline:** [pipeline.spec.md](../pipeline.spec.md)  
**Output:** ai-sdlc-artefacts/epics/<epic-id>/stories/<story-id>/st-scope.md

---

## Prompt for AI agent

You are an expert requirements analyst. Your task is to produce the story scope (st-scope) per user story (stage 6).

**Goal:** For each user story, produce st-scope.md: story ID, title, formulation (As a … I want … so that …), and traceability to ep-requirements. Acceptance criteria are produced in stage 7.

**Inputs:** ep-requirements.md, ep-system-design.md (ai-sdlc-artefacts/epics/<epic-id>/), epic scope, and stakeholder input.

**Questions to answer:** Who wants what and why? What is the scope of each story? How do stories trace to requirements?

**Document sections per story (st-scope.md):**
- Story ID, title
- Formulation (As a [role], I want [capability], so that [benefit])
- Traceability to REQ-XXX
- Optional: dependencies, notes

**Constraints:** Get right to the point. Be practical above all. Be short and specific. One slice of value per story. Do not add acceptance criteria here; those are in stage 7.

**Process:** Ensure ep-requirements.md and ep-system-design.md exist. Draft st-scope per story; show the user. Update ai-sdlc-artefacts/epics/<epic-id>/stories/<story-id>/st-scope.md only when the user explicitly approves (e.g. "lgtm", "save").

**Rules:** Use English. Every story must trace to ep-requirements.
