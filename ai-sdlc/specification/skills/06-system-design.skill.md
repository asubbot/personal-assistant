---
name: system-design.skill
description: Produce epic system design from ep-requirements and ep-acceptance-criteria; output ep-system-design.md. Use when defining or refining system design (stage 6), e.g. "system design for this epic", "architecture", "components and interfaces".
---

# Stage 6: System design

**Pipeline:** [pipeline.spec.md](../pipeline.spec.md)  
**Output:** ai-sdlc-artefacts/epics/<epic-id>/ep-system-design.md

---

## Core Principles

Follow these principles for all system design work:

1. **Never write until approved** — Do not create or overwrite ep-system-design.md until the user explicitly approves the draft (e.g. "lgtm", "save", "approve"). All edits go into the draft in chat; do not write to file until approval.
2. **Existing file is baseline** — If ep-system-design.md already exists for the epic, treat it as the current baseline; propose changes as edits and overwrite only after user approval.
3. **Options when in doubt** — When multiple valid choices exist (e.g. structure depth, module boundaries, references to research), present options (e.g. A/B) and ask the user to choose before proceeding.
4. **References** — Links only to paths under `ai-sdlc-artefacts/`; every linked document must exist. Keep traceability to ep-requirements. Write in English.
5. **Full REQ traceability** — Every requirement (REQ-XXX) from ep-requirements.md must be mentioned at least once in ep-system-design.md; the document must not omit any REQ.
6. **Practical and short** — Get to the point. Be practical above all. Be short and specific.
7. **Legacy** — Do not modify content under `legacy` folders; use as reference only (e.g. technical discovery or research).

---

## 1. Context and goal

You are the Tech Lead for this epic. Your role is to produce the epic system design document (stage 6).

**Goal:** Produce ep-system-design.md: components, interfaces, data models, and key design decisions. This output is the input for user story planning (stage 7) and implementation planning (stage 8); keep it traceable to ep-requirements and, where useful, to ep-acceptance-criteria. Optionally include technical discovery (options, comparison, recommendation, risks) and references to research under `legacy/` (read-only).

**Inputs:** ep-requirements.md and ep-acceptance-criteria.md (ai-sdlc-artefacts/epics/<epic-id>/). If either is missing, ask the user to run stage 4 (Requirements) or stage 5 (Acceptance criteria) first. Platform constraints and research or technical discovery (e.g. under epic `legacy/`) may be used as reference.

**Questions to answer:** How is the system structured? What are the main components and interfaces? What are the key design decisions and risks?

---

## 2. System design workflow

Follow this order:

1. **Check inputs** — Ensure ep-requirements.md and ep-acceptance-criteria.md exist for the epic. If not, ask the user to run stage 4 or 5 first.
2. **Check existing ep-system-design** — If ep-system-design.md exists for the epic, treat it as the baseline; propose changes as edits.
3. **Draft in chat** — Draft the design in chat (section by section or by block). Show each part to the user and ask for clarification or changes. Apply edits only in the draft in chat; do not write to file yet.
4. **Resolve choices** — When multiple valid options exist (e.g. structure depth, module boundaries), present options (e.g. A/B) and ask the user to choose.
5. **Write after approval** — Create or update ai-sdlc-artefacts/epics/<epic-id>/ep-system-design.md only when the user explicitly approves (e.g. "lgtm", "save", "approve").
6. **Legacy** — Do not modify content under `legacy` folders; use as reference only.

---

## 3. Output structure (ep-system-design.md)

Use these section headings (or user-agreed equivalents).

- **Overview** — Brief summary of the system and design scope, with traceability to key requirements. One short paragraph or bullet list.
- **Architecture** — High-level structure and boundaries. **Must include C4 Level 2 (Containers):** C4-PlantUML diagram: source in `diagrams/c4-container.puml`, PNG in `diagrams/c4-container.png`. In ep-system-design: centered image, then "Source:" with link to .puml and regeneration command (`plantuml -tpng diagrams/c4-container.puml` from epic directory). System context (C1) is in ep-requirements; C2 is drawn here. Optional subsection **Module boundaries**: layers/modules, dependency rules, and wiring responsibilities; optional diagram and verification step (script or checklist).
- **Components and interfaces** — Table or list: Component | Responsibility | Key interface/contract. Each component should trace to requirements where applicable.
- **Data models** — Main entities, schemas, and state transitions relevant to the design. Reference upstream artefacts under ai-sdlc-artefacts only.
- **Error handling** — How validation and runtime failure modes are handled; trace to requirements where applicable.
- **Testing strategy** (optional) — Short note on unit, integration, e2e, and deploy verification; can reference strategy.md under ai-sdlc-artefacts.
- **Optional:** Technical discovery, options comparison, recommendation, risks; references to research (e.g. legacy/technical-discovery) only for reading; no edits to legacy.

**Diagrams:** Use the epic's `diagrams/` folder (same as in ep-requirements); store `c4-container.puml` there and export PNG to `diagrams/c4-container.png` so the relative path in the document works.

**TOC:** Include a first-level table of contents (links to section anchors).

**Traceability:** In the body, link to ep-requirements.md sections (e.g. [REQ-001](ep-requirements.md#interface-and-deployment)). **Every REQ from ep-requirements.md must be referenced at least once** in the document (full requirement traceability). Every linked path must be under `ai-sdlc-artefacts/` and exist.

**Quality:** Be specific and testable where possible; avoid vague wording. Keep the document maintainable for stage 7 and 8.

---

## 4. Done when

Verify all before considering the stage complete:

- [ ] ep-system-design.md exists at ai-sdlc-artefacts/epics/<epic-id>/ep-system-design.md
- [ ] Document contains **Overview** (system summary and traceability to REQ), **Architecture** including **C4 C2** (source in `diagrams/c4-container.puml`, PNG in `diagrams/c4-container.png` embedded centered; Source line with regeneration command), **Components and interfaces**, and **Data models** (if applicable); **Error handling** and **Testing strategy** recommended where relevant
- [ ] Every link in the document points to an existing path under `ai-sdlc-artefacts/` (no broken links)
- [ ] Traceability to ep-requirements is maintained: **every REQ from ep-requirements.md is referenced at least once** in the document
- [ ] User has explicitly approved the content
