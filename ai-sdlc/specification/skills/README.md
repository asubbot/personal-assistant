# specification/skills — Agent instructions

Agent instructions for the SDLC pipeline. One skill file per pipeline stage (01–18). All paths in skills use **ai-sdlc-artefacts/epics/<epic-id>/** (e.g. ep-104).

---

## Stage → skill table

| Stage | Name | Skill file |
|-------|------|------------|
| 1 | Product requirements | [01-product-requirements.skill.md](01-product-requirements.skill.md) |
| 2 | Non-functional requirements | [02-nfr.skill.md](02-nfr.skill.md) |
| 3 | Technical discovery | [03-technical-discovery.skill.md](03-technical-discovery.skill.md) |
| 4 | System Design | [04-system-design.skill.md](04-system-design.skill.md) |
| 5 | Delivery strategy | [05-delivery-strategy.skill.md](05-delivery-strategy.skill.md) |
| 6 | Test strategy | [06-test-strategy.skill.md](06-test-strategy.skill.md) |
| 7 | Epics | [07-epics.skill.md](07-epics.skill.md) |
| 8 | User stories | [08-user-stories.skill.md](08-user-stories.skill.md) |
| 9 | Requirements refinement | [09-requirements-refinement.skill.md](09-requirements-refinement.skill.md) |
| 10 | Acceptance criteria | [10-acceptance-criteria.skill.md](10-acceptance-criteria.skill.md) |
| 11 | Tasks decomposition | [11-tasks-decomposition.skill.md](11-tasks-decomposition.skill.md) |
| 12 | Implementation plan | [12-implementation-plan.skill.md](12-implementation-plan.skill.md) |
| 13 | Implementation | [13-implementation.skill.md](13-implementation.skill.md) |
| 14 | Quality gate | [14-quality-gate.skill.md](14-quality-gate.skill.md) |
| 15 | Test execution | [15-test-execution.skill.md](15-test-execution.skill.md) |
| 16 | Deployment | [16-deployment.skill.md](16-deployment.skill.md) |
| 17 | Acceptance verification | [17-acceptance-verification.skill.md](17-acceptance-verification.skill.md) |
| 18 | Closure / Retrospective | [18-closure.skill.md](18-closure.skill.md) |

---

## Intent / trigger → skill (optional)

When a user request matches an intent below, use the corresponding skill so that "when user says X, use skill Y" is explicit.

| Intent / trigger | Skill |
|------------------|--------|
| Run code review / quality gate for epic | [14-quality-gate.skill.md](14-quality-gate.skill.md) |
| Audit test coverage / find gaps | [15-test-execution.skill.md](15-test-execution.skill.md) (coverage audit) or [14-quality-gate.skill.md](14-quality-gate.skill.md) (test audit workflow) |
| Plan or refine scope | [01-product-requirements.skill.md](01-product-requirements.skill.md), [07-epics.skill.md](07-epics.skill.md) |
| Write or update user stories | [08-user-stories.skill.md](08-user-stories.skill.md) |
| Write or update acceptance criteria | [10-acceptance-criteria.skill.md](10-acceptance-criteria.skill.md) |
| Execute implementation plan (code tasks) | [13-implementation.skill.md](13-implementation.skill.md) |

Pipeline spec: [../pipeline.spec.md](../pipeline.spec.md).
