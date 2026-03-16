---
name: audit.skill
description: Produce epic audit report (stage 9); output ep-audit-report.md. Use for "audit", "quality gate", "status report", "implementation vs plan".
---

# Stage 9: Audit

**Pipeline:** [pipeline.spec.md](../pipeline.spec.md)  
**Output:** The audit report is **always shown in chat** (full content). The file ai-sdlc-artefacts/epics/<epic-id>/ep-audit-report.md is **not** created unless the user explicitly asks to save (e.g. "save", "write to file", "approve").

---

## 1. Context and goal

You are the QA and delivery lead. Your task is to produce an audit (status) report for the current branch (stage 9).

**Goal:** Produce an audit report (status of implementation vs plan, test results and coverage, quality gate, gaps or risks). The report is **output in chat** in full; it is **not saved to a file** unless the user explicitly asks. Plan = ep-implementation-plan.md; traceability to AC and implementation plan.

**Inputs:**

- Current branch (codebase).
- **Epic artefacts:** ai-sdlc-artefacts/epics/<epic-id>/ep-implementation-plan.md, ep-acceptance-criteria.md; optionally ep-requirements.md, ep-system-design.md, ep-manual-test-scenarios.md.
- **Test strategy:** e.g. ai-sdlc-artefacts/strategy.md or project-defined test/coverage commands.
- **Test and coverage outputs:** Run **`make check`** (or the project’s equivalent). This single command is sufficient, as it runs all defined checks (e.g. fmt, vet, lint, tests with coverage, module boundaries). Use its terminal output for pass/fail and for the **total** test coverage figure.

**Note:** The REQ/AC test coverage matrix is **generated inside the audit report**, not read from a separate file. The file ep-req-ac-test-coverage.md is not part of the pipeline; if it exists in the epic folder, do not use it as input—produce the matrix from ep-acceptance-criteria, ep-requirements, and codebase (e.g. `Covers AC-xxx` comments).

**Epic ID:** Resolve from the branch name, from the path of existing epic artefacts, or ask the user if ambiguous.

**Questions to answer:** What is implemented vs planned? Do tests pass? What is the coverage? Are there lint or quality issues? What gaps or risks remain?

**Constraints:** Get right to the point. Be practical above all. Be short and specific. When multiple valid choices exist (e.g. report format, level of detail), present options and ask the user to choose. See [skills README](README.md) (Common behaviour).

**Rules:** Use English. Keep traceability to AC and implementation plan. References only to paths under `ai-sdlc-artefacts/`; every linked document must exist.

---

## 2. Audit workflow

Follow this order:

1. **Check inputs** — Ensure ep-implementation-plan.md and ep-acceptance-criteria.md exist for the epic. Resolve epic ID (branch, artefact path, or ask user). If inputs are missing, ask the user to run the missing stage(s) or provide the epic.
2. **Run tests and checks** — Execute **`make check`** (or the project’s equivalent single command that runs fmt, vet, lint, tests with coverage, and any boundary/static checks). Capture pass/fail and the **total** test coverage from the command output.
3. **Compare to plan** — For each task in ep-implementation-plan.md, determine status: done, pending, or blocked. Tie test results to acceptance criteria (AC-XXX) where applicable.
4. **Output in chat** — **Always** output the full audit report (all sections in §3) **in the chat message** so the user sees the complete content. Do **not** write to the file system at this step.
5. **Save to file only when requested** — Create or update ai-sdlc-artefacts/epics/<epic-id>/ep-audit-report.md **only** when the user explicitly asks to save (e.g. "save", "write to file", "lgtm", "approve"). If the user does not ask to save, the report exists only in chat.

---

## 3. Output structure (ep-audit-report.md)

Use these elements (or user-agreed equivalents):

- **Document header** — **Date and time of creation** (when the report is written; e.g. `2025-03-15 19:30 UTC`). Purpose, pipeline link, links to ep-implementation-plan.md and ep-acceptance-criteria.md (and optionally ep-requirements.md).
- **Summary** — Pass/fail or overall status in one short paragraph.
- **Implementation vs plan** — For each task (or task group) from ep-implementation-plan, state status: done / pending / blocked. Reference task identifiers (e.g. Task 1.1, 1.2). Link to ep-implementation-plan.md where helpful.
- **Test results and coverage** — Command run (e.g. `make check`), pass/fail, and **total test coverage** (the total line from the coverage output, e.g. `total: (statements) XX.X%`). Optionally include per-package breakdown. Reference which AC (AC-XXX) are covered by which tests where applicable.
- **REQ/AC test coverage matrix** — **Generate this inside the report** (do not reference a separate ep-req-ac-test-coverage.md; that file is not a pipeline artefact). Build a table that maps each AC to test levels and links:
  - Columns: AC | REQ | Unit | Integration | E2E | Manual | Link.
  - One row per AC from ep-acceptance-criteria.md. **AC and REQ columns must use markdown links** to the epic artefacts: e.g. `[AC-001](ep-acceptance-criteria.md#ac-001)`, `[REQ-001](ep-requirements.md#req-001)` (adjust anchors to match the document structure). REQ values come from ep-requirements.md; fill Unit/Integration/E2E/Manual (e.g. ✓ or —) and Link (test file paths or manual scenario links) by scanning the codebase for `Covers AC-xxx` / `Supporting AC-xxx` comments and any project test strategy (e.g. strategy.md, ep-manual-test-scenarios.md). Deferred AC (e.g. AC-026, AC-027) mark as deferred in Link.
  - Optionally add a short **Notes** subsection under the table (e.g. what Unit/Integration mean in this project, Make commands for test/coverage).
- **Quality gate** — Result of the checks run as part of `make check` (e.g. fmt, vet, lint, module boundaries): pass/fail or list of issues. If the project uses a single command like `make check`, state that it was run and its overall result.
- **Gaps, risks, recommendations** — **Gap:** something planned in the implementation plan or AC but not implemented or not verified. **Risk:** something that may cause problems later (e.g. tech debt, missing tests, instability). **Recommendations:** short actionable next steps.

**Traceability:** Implementation vs plan must reference task IDs from ep-implementation-plan. Test/AC mapping must reference AC-XXX from ep-acceptance-criteria.md. Every link must point to an existing path under `ai-sdlc-artefacts/`.

**Example** (Implementation vs plan snippet):

```markdown
| Task   | Status  | Notes |
|--------|--------|-------|
| 1.1    | Done   | Config load and validation — [ep-implementation-plan](ep-implementation-plan.md) |
| 1.2    | Pending| LLM client wiring |
```

---

## 4. Done when

Verify all before considering the stage complete:

- [ ] The **full report was shown in chat** (user can read it without opening a file).
- [ ] Report content (in chat) includes: header with **date and time**, Summary, Implementation vs plan, Test results and coverage, **REQ/AC test coverage matrix** (AC and REQ columns use markdown links to ep-acceptance-criteria.md and ep-requirements.md), Quality gate, Gaps/risks/recommendations.
- [ ] Implementation vs plan references task IDs from ep-implementation-plan and states status (done/pending/blocked).
- [ ] Traceability to AC and implementation plan is kept; all links point to existing paths under `ai-sdlc-artefacts/`.
- [ ] Each test has comment with AC and REQ for tracing; each AC has test; each REQ has one or more AC.
- [ ] **If** the user asked to save: ep-audit-report.md was written only after that explicit request.
