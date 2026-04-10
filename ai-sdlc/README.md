# ai-sdlc — Agentic SDLC process definition

This folder defines the **agent-driven development process**: pipeline stages and agent instructions (skills). It is the single source of truth for how epics are elaborated and delivered. Artefact structure is defined in each skill (e.g. "Output structure" or "Document sections").

- **Relation to artefacts:** Execution results (requirements, design, implementation plans, test strategy, etc.) are stored in **ai-sdlc-artefacts**, not here. This folder describes the *process*; artefacts live under `../ai-sdlc-artefacts/`.

- **Specification:**
  - [specification/pipeline.spec.md](specification/pipeline.spec.md) — pipeline stages, inputs/outputs, stage→skill mapping.
  - [specification/skills/](specification/skills/) — agent instructions (one skill per numbered stage 1–11; optional cross-cutting skills). Each skill defines the required structure of its output artefact(s).
