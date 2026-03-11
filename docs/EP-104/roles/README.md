# Role prompts per pipeline stage

One prompt file per pipeline stage. Use the prompt from the file for the stage you are running. The [pipeline spec](../PIPELINE.SPEC.md) links to each file from the stage table.

| Stage | Name | Role | File |
|-------|------|------|------|
| 1 | Product requirements | Product Owner | [01-product-requirements.md](01-product-requirements.md) |
| 2 | Non-functional requirements | Tech Lead | [02-nfr.md](02-nfr.md) |
| 3 | Technical discovery | Tech Lead | [03-technical-discovery.md](03-technical-discovery.md) |
| 4 | System Design | Tech Lead | [04-system-design.md](04-system-design.md) |
| 5 | Delivery strategy | Product Owner | [05-delivery-strategy.md](05-delivery-strategy.md) |
| 6 | Test strategy | QA Lead | [06-test-strategy.md](06-test-strategy.md) |
| 7 | Epics | Product Owner | [07-epics.md](07-epics.md) |
| 8 | User stories | Analyst | [08-user-stories.md](08-user-stories.md) |
| 9 | Requirements refinement | Analyst | [09-requirements-refinement.md](09-requirements-refinement.md) |
| 10 | Acceptance criteria | QA Lead | [10-acceptance-criteria.md](10-acceptance-criteria.md) |
| 11 | Tasks decomposition | Tech Lead | [11-tasks-decomposition.md](11-tasks-decomposition.md) |
| 12 | Implementation plan | Tech Lead | [12-implementation-plan.md](12-implementation-plan.md) |
| 13 | Implementation | Developer | [13-implementation.md](13-implementation.md) |
| 14 | Quality gate | Developer | [14-quality-gate.md](14-quality-gate.md) |
| 15 | Test execution | QA Lead | [15-test-execution.md](15-test-execution.md) |
| 16 | Deployment | Tech Lead / DevOps | [16-deployment.md](16-deployment.md) |
| 17 | Acceptance verification | QA Lead | [17-acceptance-verification.md](17-acceptance-verification.md) |
| 18 | Closure / Retrospective | Product Owner / Tech Lead | [18-closure.md](18-closure.md) |

Each file contains a **Prompt for AI agent** section: use that content as the agent context when executing the stage.
