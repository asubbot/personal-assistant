# ai-sdlc-artefacts — Process execution results

This folder stores the **outputs of the agentic SDLC process**: scope, strategy, requirements, design, implementation plans, user stories, acceptance criteria, audit reports, and other deliverables produced when running the pipeline defined in **ai-sdlc**.

- **Project-level:** scope.md, strategy.md in the **ai-sdlc-artefacts/** root (produced by the pipeline when running scope analysis and strategy analysis).

- **Hierarchy:** `epics/<epic-id>/` — one directory per epic (e.g. `epics/ep-104/`). Epic-level artefacts: ep-scope.md, ep-requirements.md, ep-system-design.md; optionally a `research/` subfolder. Existing files (01-02-requirements, 04-system-design, 05-delivery-strategy, 06-test-strategy, 08-user-stories, 11-12-implementation-plan, etc.) are kept; migration to new names is separate. **Story-level** artefacts live under `epics/<epic-id>/stories/<story-id>/`: st-scope.md, st-acceptance-criteria.md, st-implementation-plan.md, st-audit-report.md; one folder per user story ID. Existing story files (e.g. acceptance-criteria.md) are kept; migration is separate.

- **Active epic:** **ep-104** — see [epics/ep-104/](epics/ep-104/) for the current epic's deliverables. Content is created and updated according to [ai-sdlc/specification/pipeline.spec.md](../ai-sdlc/specification/pipeline.spec.md).

- **Paths:** Project-level artefacts use `ai-sdlc-artefacts/`; epic-level and story-level use `ai-sdlc-artefacts/epics/<epic-id>/` and `ai-sdlc-artefacts/epics/<epic-id>/stories/<story-id>/`.
