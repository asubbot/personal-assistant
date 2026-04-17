# ai-sdlc-artefacts — Process execution results

This folder stores the **outputs of the agentic SDLC process**: scope, strategy, requirements, design, implementation plans, user stories, acceptance criteria, audit reports, and other deliverables produced when running the pipeline defined in **ai-sdlc**.

- **Project-level:** scope.md, strategy.md in the **ai-sdlc-artefacts/** root (produced by the pipeline when running scope analysis and strategy analysis). **Security:** [threat-model.md](threat-model.md) — code-grounded threat model (see [threat-model-report.skill.md](../ai-sdlc/specification/skills/threat-model-report.skill.md)). **Architecture:** [pa-architecture-review.md](pa-architecture-review.md) — code-grounded architecture review with C4 diagrams and strengths/weaknesses/risks.

- **Hierarchy:** `epics/<epic-id>/` — one directory per epic (e.g. `epics/EP-104/`). Epic-level artefacts: ep-scope.md, ep-requirements.md, ep-system-design.md, implementation-plan.md, 10-acceptance-criteria.md (index), audit-report.md; optionally a `research/` subfolder. **Story-level** artefacts under `epics/<epic-id>/stories/<story-id>/`: st-scope.md, st-acceptance-criteria.md, st-implementation-plan.md, st-audit-report.md. Legacy placeholder files (01-02-requirements, 04-system-design, etc.) may remain; they point to docs/EP-104 for reference.

- **EP-104 migration:** Content for epic EP-104 has been migrated from **docs/EP-104** into this structure. Project-level [scope.md](scope.md) and [strategy.md](strategy.md) are in the root; epic-level and story-level artefacts are under [epics/EP-104/](epics/EP-104/). All internal links in migrated artefacts use paths under ai-sdlc-artefacts. Legacy source: docs/EP-104 (unchanged, for reference).

- **Active epic:** **EP-104** — see [epics/EP-104/](epics/EP-104/) for the current epic's deliverables. Content is created and updated according to [ai-sdlc/specification/pipeline.spec.md](../ai-sdlc/specification/pipeline.spec.md).

- **Paths:** Project-level artefacts use `ai-sdlc-artefacts/`; epic-level and story-level use `ai-sdlc-artefacts/epics/<epic-id>/` and `ai-sdlc-artefacts/epics/<epic-id>/stories/<story-id>/`.
