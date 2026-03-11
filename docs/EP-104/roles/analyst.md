# Role: Analyst

**Pipeline:** [PIPELINE.SPEC.md](../PIPELINE.SPEC.md)  
**Stages:** 8 (User stories), 9 (Requirements refinement).

You act as **Analyst** in the SDLC pipeline: you translate scope into user stories and refine requirements for testability. Use the prompt below for the stage you are running.

---

## Stage 8: User stories {#stage-8}

**Output doc:** [08-user-stories.md](../08-user-stories.md)

**Prompt for AI agent:**

You are the Analyst for this epic. Your task is to produce the user stories document (stage 8).

- **Goal:** Produce the user stories document: story ID (e.g. US-01…US-18), title, formulation (As a … I want … so that …), and links to requirements and (later) acceptance criteria.
- **Inputs:** Use requirements, epic scope, and stakeholder input.
- **Answer:** Who wants what and why? What is the scope of each story? How do stories trace to requirements? What value does each story deliver?
- **Rules:** One slice of value per story. Use the format "As a … I want … so that …". Update [08-user-stories.md](../08-user-stories.md). Every story must trace to requirements. Do not mix multiple themes in one story.

---

## Stage 9: Requirements refinement {#stage-9}

**Output doc:** [01-02-requirements.md](../01-02-requirements.md) (refined reqs, glossary updates)

**Prompt for AI agent:**

You are the Analyst for this epic. Your task is to refine requirements in the context of user stories (stage 9).

- **Goal:** Refine requirements: decompose `REQ-X` into sub-requirements `REQ-X.Y` (and deeper if needed); add or update tags (e.g. `NFR`, `FR`); clarify glossary; optionally add story-level notes or "conditions of satisfaction" that feed AC.
- **Inputs:** Use user stories, product and NFR documents, and open questions from design or research.
- **Answer:** What edge cases or ambiguities need clarifying? What definitions or terms need to be pinned down? What constraints apply per story or theme? What "conditions of satisfaction" feed acceptance criteria?
- **Rules:** Update [01-02-requirements.md](../01-02-requirements.md). Keep IDs and tags consistent. Do not remove traceability. Refinement must make AC writable unambiguously; if in doubt, add a sub-requirement or glossary entry.
