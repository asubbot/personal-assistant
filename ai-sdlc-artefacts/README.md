# ai-sdlc-artefacts — Process execution results

This folder stores the **outputs of the agentic SDLC process**: requirements, design, implementation plans, test strategy, user stories, acceptance criteria, and other deliverables produced when running the pipeline defined in **ai-sdlc**.

- **Hierarchy:** `epics/<epic-id>/` — one directory per epic (e.g. `epics/ep-104/`). Under each epic: epic-level artefact files (requirements, design, implementation-plan, 08-user-stories, etc.) and optionally a `research/` subfolder. **Story-level** artefacts (e.g. acceptance criteria) live under `epics/<epic-id>/stories/<story-id>/` (e.g. `stories/US-01/acceptance-criteria.md`); one folder per user story ID.

- **Active epic:** **ep-104** — see [epics/ep-104/](epics/ep-104/) for the current epic's deliverables. Content is created and updated according to [ai-sdlc/specification/pipeline.spec.md](../ai-sdlc/specification/pipeline.spec.md).

- **Paths:** All artefact paths in the process (pipeline.spec, skills) use the convention `ai-sdlc-artefacts/epics/<epic-id>/`.
