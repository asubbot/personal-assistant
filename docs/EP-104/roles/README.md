# Role prompts for pipeline stages

Agent prompts for each pipeline role. Use the prompt from the role file for the stage you are running. The [pipeline spec](../PIPELINE.SPEC.md) links to the exact section per stage.

| Role | Stages | File |
|------|--------|------|
| Product Owner | 1, 5, 7, 18 | [product-owner.md](product-owner.md) |
| Tech Lead | 2, 3, 4, 11, 12 | [tech-lead.md](tech-lead.md) |
| QA Lead | 6, 10, 15, 17 | [qa-lead.md](qa-lead.md) |
| Analyst | 8, 9 | [analyst.md](analyst.md) |
| Developer | 13, 14 | [developer.md](developer.md) |
| DevOps (Tech Lead / DevOps) | 16 | [devops.md](devops.md) |

Each file contains a **Prompt for AI agent** block per stage: copy the content of that block into the agent context when executing that stage.
