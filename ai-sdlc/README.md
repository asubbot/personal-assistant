# ai-sdlc — Agentic SDLC process definition

This folder defines the **agent-driven development process**: pipeline stages, agent instructions (skills), and artefact templates. It is the single source of truth for how epics are elaborated and delivered.

- **Relation to artefacts:** Execution results (requirements, design, implementation plans, test strategy, etc.) are stored in **ai-sdlc-artefacts**, not here. This folder describes the *process*; artefacts live under `../ai-sdlc-artefacts/epics/<epic-id>/`.

- **Specification:**
  - [specification/pipeline.spec.md](specification/pipeline.spec.md) — pipeline stages 1–18, inputs/outputs, stage→skill mapping.
  - [specification/skills/](specification/skills/) — agent instructions (one skill per stage; optional cross-cutting skills).

- **Templates:** [specification/templates/](specification/templates/) — artefact templates for consistent outputs.
