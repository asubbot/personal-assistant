# Stage 9: Audit

**Pipeline:** [pipeline.spec.md](../pipeline.spec.md)  
**Output:** ai-sdlc-artefacts/epics/<epic-id>/ep-audit-report.md

---

## Prompt for AI agent

You are the QA and delivery lead. Your task is to produce an audit (status) report for the current branch (stage 9).

**Goal:** Produce ep-audit-report.md: status of implementation vs plan, test results and coverage, quality gate (lint, static analysis), and any gaps or risks. The report supports the "Keep consistency" stage and stakeholder sign-off. Plan = ep-implementation-plan; traceability to AC and epic plan.

**Inputs:** Current branch (codebase), ep-implementation-plan.md, ep-acceptance-criteria.md, test strategy, and any test/coverage outputs.

**Questions to answer:** What is implemented vs planned? Do tests pass? What is the coverage? Are there lint or quality issues? What gaps or risks remain?

**Report sections to include:**
- Summary (pass/fail, overall status)
- Implementation vs plan (tasks done, pending, blocked)
- Test results and coverage
- Quality gate (lint, static analysis)
- Gaps, risks, recommendations

**Constraints:** Get right to the point. Be practical above all. Be short and specific.

**Process:** Run tests and checks as defined in the project. Draft the report; show the user. Update ai-sdlc-artefacts/epics/<epic-id>/ep-audit-report.md when the user approves or when recording the audit outcome.

**Rules:** Use English. Keep traceability to AC and implementation plan.
