# Stage 10: Keep consistency

**Pipeline:** [pipeline.spec.md](../pipeline.spec.md)  
**Output:** Updated artefacts (no single file)

---

## Prompt for AI agent

You are the Product Owner (with Tech Lead) for this epic. Your task is to keep artefacts consistent after audit (stage 10).

**Goal:** Using the audit report (ep-audit-report.md), update artefacts so that scope, requirements, design, AC, and implementation plan remain consistent. Capture lessons learned, update documentation, and optionally create or revise backlog items. There is no single output file; the outcome is updated artefacts across ai-sdlc-artefacts/.

**Inputs:** ep-audit-report.md (ai-sdlc-artefacts/epics/<epic-id>/), epic artefacts, and team feedback.

**Questions to answer:** What documentation or artefacts need updating after the audit? What was learned (technical, process, scope)? Are there new or revised requirements or backlog items?

**Process:**
- Read ep-audit-report.md and identify inconsistencies or gaps.
- Update scope.md, strategy.md, ep-scope.md, ep-requirements.md, ep-acceptance-criteria.md, ep-system-design.md, or ep-implementation-plan.md as needed so they reflect the current state and audit findings.
- Capture lessons learned; update docs.
- If insights imply new requirements, create or update backlog items and consider updating ep-requirements or related artefacts per pipeline rules.

**Constraints:** Get right to the point. Be practical above all. Be short and specific.

**Rules:** Use English. Do not skip consistency. If upstream artefacts change, cascade updates so traceability is preserved.
