# specification/skills — Agent instructions

Agent instructions for the SDLC pipeline. **One numbered skill per pipeline stage (1–9)**, plus **optional cross-cutting** skills (code review, C4 C3 diagram, project comparison, threat model from code). Paths in skills use **ai-sdlc-artefacts/** (project root: `scope.md`, `strategy.md`, optional `analytics/`) and **ai-sdlc-artefacts/epics/<epic-id>/** for epic artefacts: ep-scope, ep-requirements, ep-acceptance-criteria, ep-system-design, ep-implementation-plan, ep-audit-report. Story-level paths are not used by the pipeline.

**Common behaviour:** The agent works in cooperation with the user. When several valid choices exist (output format, path, scope, or interpretation of the request), present them (e.g. A / B) and ask the user which they prefer. Do not proceed until the user has chosen. See [pipeline.spec.md](../pipeline.spec.md) (Human-in-the-loop).

---

## Stage → skill (pipeline 1–9)

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

**Execution order**

- **Full pipeline (stages 1–9):** 1 → 2 → 3 → … → 9 — see flowchart in [pipeline.spec.md](../pipeline.spec.md) §1.
- **Epic elaboration only:** 3 → 4 → 5 → 6 → 7 (Epic planning → Requirements → Acceptance criteria → System design → Implementation planning).
- **After plan approval:** 8 (task execution) → 9 (audit).

---

## Cross-cutting skills (not a numbered stage)

| Skill | Use when |
|-------|----------|
| [code-review.skill.md](code-review.skill.md) | PR / branch / pre-merge structured review |
| [ep-C4-component.skill.md](ep-C4-component.skill.md) | C4 **C3** Go component diagram for `ep-system-design.md` (optional; complements mandatory C2 container) |
| [project-comparison-report.skill.md](project-comparison-report.skill.md) | Compare an external repo with PersonalAssistant; analytics report under `ai-sdlc-artefacts/analytics/` |
| [user-documentation.skill.md](user-documentation.skill.md) | End-user / operator docs under `docs/` and root `README.md` (installation, config, Docker, operations) |
| [threat-model-report.skill.md](threat-model-report.skill.md) | Code-grounded threat model report (default `docs/threat-model.md` or `ai-sdlc-artefacts/analytics/...`) |

---

## Intent / trigger → skill

When a user request matches an intent below, use the corresponding skill.

| Intent / trigger | Skill |
|------------------|--------|
| Plan or refine **project** scope (`scope.md`) | [01-scope-analysis.skill.md](01-scope-analysis.skill.md) |
| **Epic** planning (`ep-scope.md` per epic) | [03-epic-planning.skill.md](03-epic-planning.skill.md) |
| Define delivery or test **strategy** (`strategy.md`) | [02-strategy-analysis.skill.md](02-strategy-analysis.skill.md) |
| Write or update **requirements** (EARS, `ep-requirements.md`) | [04-requirements.skill.md](04-requirements.skill.md) |
| Write or update **acceptance criteria** | [05-acceptance-criteria.skill.md](05-acceptance-criteria.skill.md) |
| **System design** / architecture / C2 container (`ep-system-design.md`) | [06-system-design.skill.md](06-system-design.skill.md) |
| **C4 C3** component diagram (Go packages) for epic system design | [ep-C4-component.skill.md](ep-C4-component.skill.md) |
| **Implementation plan** (tasks, ordering, verification) | [07-implementation-planning.skill.md](07-implementation-planning.skill.md) |
| **Execute** implementation plan (code tasks, commits) | [08-task-execution.skill.md](08-task-execution.skill.md) |
| **Audit** / quality gate / status report (epic or project) | [09-audit.skill.md](09-audit.skill.md) |
| **Code review** / PR review | [code-review.skill.md](code-review.skill.md) |
| **Analyse or compare** an external project with PA | [project-comparison-report.skill.md](project-comparison-report.skill.md) |
| **User docs** / operator guide / refresh **README** for deployers | [user-documentation.skill.md](user-documentation.skill.md) |
| **Threat model** / STRIDE / attack surface **from source code** | [threat-model-report.skill.md](threat-model-report.skill.md) |

---

## All skill files in this folder

Numbered: `01`–`09` (table above). Cross-cutting: `code-review.skill.md`, `ep-C4-component.skill.md`, `project-comparison-report.skill.md`, `user-documentation.skill.md`, `threat-model-report.skill.md`.

**Single source of stage I/O:** [pipeline.spec.md](../pipeline.spec.md) §2 (table of stages, inputs, outputs).
