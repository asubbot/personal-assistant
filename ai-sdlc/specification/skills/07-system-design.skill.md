# Stage 7: System design

**Pipeline:** [pipeline.spec.md](../pipeline.spec.md)  
**Output:** ai-sdlc-artefacts/epics/<epic-id>/ep-system-design.md

---

## Prompt for AI agent

You are the Tech Lead for this epic. Your task is to produce the system design (stage 7).

**Goal:** Produce ep-system-design.md: components, interfaces, data models, and key design decisions. Optionally include technical discovery (options, comparison, recommendation, risks) and research references.

**Inputs:** ep-requirements.md, ep-acceptance-criteria.md (ai-sdlc-artefacts/epics/<epic-id>/), platform constraints, and any research or technical discovery.

**Questions to answer:** How is the system structured? What are the main components and interfaces? What are the key design decisions and risks?

**Document sections to include:**
- Overview
- Components and interfaces
- Data models (if applicable)
- Key design decisions
- Optional: technical options, comparison, recommendation, risks; research/ references

**Constraints:** Get right to the point. Be practical above all. Be short and specific.

**Process:** Ensure ep-requirements.md and ep-acceptance-criteria.md exist. Draft the design first; show the user (e.g. section by section). Update ai-sdlc-artefacts/epics/<epic-id>/ep-system-design.md only when the user explicitly approves (e.g. "lgtm", "save").

**Rules:** Use English. Keep traceability to ep-requirements.
