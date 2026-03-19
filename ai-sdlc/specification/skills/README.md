# specification/skills — Agent instructions

Agent instructions for the SDLC pipeline. One skill file per pipeline stage, plus optional cross-cutting skills (e.g. code review). Paths in skills use **ai-sdlc-artefacts/** (root for scope.md, strategy.md) and **ai-sdlc-artefacts/epics/<epic-id>/** for epic-level artefacts. Epic-level artefacts include ep-scope, ep-requirements, ep-acceptance-criteria, ep-system-design, ep-implementation-plan, ep-audit-report. Story-level paths are no longer used by the pipeline.

**Common behaviour:** The agent works in cooperation with the user. When several valid choices exist (output format, path, scope, or interpretation of the request), present them (e.g. A / B) and ask the user which they prefer. Do not proceed until the user has chosen. See [pipeline.spec.md](../pipeline.spec.md) (Human-in-the-loop).

---

## Stage → skill table

| Stage | Name | Skill file |
|-------|------|------------|
| 1 | Scope analysis | [01-scope-analysis.skill.md](01-scope-analysis.skill.md) |
| 2 | Strategy analysis | [02-strategy-analysis.skill.md](02-strategy-analysis.skill.md) |
| 3 | Epic planning | [03-epic-planning.skill.md](03-epic-planning.skill.md) |
| 4 | Requirements | [04-requirements.skill.md](04-requirements.skill.md) |
| 5 | Acceptance criteria | [05-acceptance-criteria.skill.md](05-acceptance-criteria.skill.md) |
| 6 | System design | [06-system-design.skill.md](06-system-design.skill.md) |
| 7 | Implementation planning | [07-implementation-planning.skill.md](07-implementation-planning.skill.md) |
| 8 | Task execution | [08-task-execution.skill.md](08-task-execution.skill.md) |
| 9 | Audit | [09-audit.skill.md](09-audit.skill.md) |
| 10 | Keep consistency | [10-keep-consistency.skill.md](10-keep-consistency.skill.md) |

**Cross-cutting (not a numbered stage):** [code-review.skill.md](code-review.skill.md) — structured PR/branch review.

**Execution order for stages 3–7:** 3 → 4 → 5 → 6 → 7 (Epic planning → Requirements → Acceptance criteria → System design → Implementation planning). See [pipeline.spec.md](../pipeline.spec.md).

---

## Intent / trigger → skill (optional)

When a user request matches an intent below, use the corresponding skill so that "when user says X, use skill Y" is explicit.

| Intent / trigger | Skill |
|------------------|--------|
| Plan or refine scope | [01-scope-analysis.skill.md](01-scope-analysis.skill.md), [03-epic-planning.skill.md](03-epic-planning.skill.md) |
| Define delivery or test strategy | [02-strategy-analysis.skill.md](02-strategy-analysis.skill.md) |
| Write or update acceptance criteria | [05-acceptance-criteria.skill.md](05-acceptance-criteria.skill.md) |
| Implementation plan for epic (tasks, ordering) | [07-implementation-planning.skill.md](07-implementation-planning.skill.md) |
| Execute implementation plan (code tasks) | [08-task-execution.skill.md](08-task-execution.skill.md) |
| Audit / quality gate / status report | [09-audit.skill.md](09-audit.skill.md) |
| Code review / PR review / pre-merge review | [code-review.skill.md](code-review.skill.md) |
| Keep artefacts consistent after audit | [10-keep-consistency.skill.md](10-keep-consistency.skill.md) |

Pipeline spec: [../pipeline.spec.md](../pipeline.spec.md).
