---
name: audit.skill
description: Produce epic or project-level audit reports (stage 11), write ep-audit-report.md or audit-report.md, and return compact chat summary.
---

# Stage 11: Audit

**Pipeline:** [pipeline.spec.md](../pipeline.spec.md)  
**Output:** The audit report is **written to file by default**: `ai-sdlc-artefacts/epics/<epic-id>/ep-audit-report.md` (epic) or `ai-sdlc-artefacts/audit-report.md` (project). In chat, output **summary only** (compact mode by default, full mode only on explicit request).

---

## 1. Context and goal

You are the QA and delivery lead. Your task is to produce an audit (status) report for the current branch (stage 11).

**Goal:** Produce an audit report (status of implementation vs plan, test results and coverage, quality gate, gaps or risks). Persist the full report to file by default; provide compact summary in chat. Plan = ep-implementation-plan.md; traceability to AC and implementation plan.

**Inputs:**

- Current branch (codebase).
- **Epic artefacts:** ai-sdlc-artefacts/epics/<epic-id>/ep-implementation-plan.md, ep-acceptance-criteria.md; optionally ep-requirements.md, ep-system-design.md, ep-manual-test-scenarios.md, **ep-code-review.md** (include **all** `## Review iteration N` sections when assessing the code-review gate per [pipeline.spec.md](../pipeline.spec.md) **§2.2**).
- **Prerequisite (epic delivery path):** Do not treat the epic as past the code-review gate until **§2.2** exit criteria are met (zero Blocker/Major/Medium/Minor) or the operator has recorded a decision after the iteration cap—see [10-code-review.skill.md](10-code-review.skill.md).
- **Test strategy:** e.g. ai-sdlc-artefacts/strategy.md or project-defined test/coverage commands.
- **Test and coverage outputs:** Run **`make check`** (or the project’s equivalent). This single command is sufficient, as it runs all defined checks (e.g. fmt, vet, lint, tests with coverage, module boundaries). Use its terminal output for pass/fail and for the **total** test coverage figure.
- **AC Coverage (DEFAULT-FIRST):** Before audit, run `./bin/validate EP-XXX --json` and use it as the primary source for AC↔test mapping. Fall back to manual code scan only when validator output is missing or inconsistent. See [VALIDATION.md](../../tools/validate/VALIDATION.md).

**Note:** The REQ/AC test coverage matrix is **generated inside the audit report**, not read from a separate file. The file ep-req-ac-test-coverage.md is not part of the pipeline; if it exists in the epic folder, do not use it as input—produce the matrix from ep-acceptance-criteria, ep-requirements, and codebase (e.g. `Covers AC-EE.NNN` comments).

**Epic ID:** Resolve from the branch name, from the path of existing epic artefacts, or ask the user if ambiguous.

**Project-level audit:** If the user requests an audit for the **whole project** (e.g. "audit the whole project", "project audit", "audit all epics"), do **not** resolve a single epic. Instead follow **§2a** and **§3a** to produce and save the project-level audit report (`audit-report.md`) with the epic summary table. Output only compact summary in chat unless user explicitly asks for full report.

**Questions to answer:** What is implemented vs planned? Do tests pass? What is the coverage? Are there lint or quality issues? What gaps or risks remain?

**Constraints:** Get right to the point. Be practical above all. Be short and specific. When multiple valid choices exist (e.g. report format, level of detail), present options and ask the user to choose. See [skills README](README.md) (Common behaviour).

**Rules:** Use English. Keep traceability to AC and implementation plan. References only to paths under `ai-sdlc-artefacts/`; every linked document must exist.

---

## 2. Audit workflow

Follow this order:

1. **Check inputs** — Ensure ep-implementation-plan.md and ep-acceptance-criteria.md exist for the epic. Resolve epic ID (branch, artefact path, or ask user). If inputs are missing, ask the user to run the missing stage(s) or provide the epic.
2. **Run tests and checks** — Execute **`make check`** (or the project’s equivalent single command that runs fmt, vet, lint, tests with coverage, and any boundary/static checks). Capture pass/fail and the **total** test coverage from the command output.
3. **Compare to plan** — For each task in ep-implementation-plan.md, determine status: done, pending, or blocked. Tie test results to acceptance criteria (AC-EE.NNN) where applicable.
4. **Write report to file** — Create or update `ai-sdlc-artefacts/epics/<epic-id>/ep-audit-report.md` by default.
5. **Output compact summary in chat** — Show only concise summary (quality gate, total coverage, key gaps/risks, report path). Output full report in chat only if user explicitly requests full mode.

---

## 2a. Project-level audit workflow

When the user requested a **project-level** audit (token-optimized mode):

- **Epics with Status DONE or CANCELED:** do **not** re-review or recompute them; keep existing rows from current `ai-sdlc-artefacts/audit-report.md` unchanged.
- **Epics with Status NEW:** read minimal scope metadata only (ID/Title/Status) and keep coverage/report as `—` unless a report already exists.
- **Epics with Status IN_PROGRESS:** run the full epic-level audit workflow (§2) for each: check inputs, run `make check` (once for the project is enough if you have IN_PROGRESS epics), compare to plan, produce the full report, and **write** it to that epic’s ep-audit-report.md. Then use that report’s total coverage and date for the project-level table.

1. **Discover epics** — List all epic directories under ai-sdlc-artefacts/epics/ (e.g. EP-001, EP-002, EP-003, EP-004). For each, ep-scope.md must exist; skip or note epics that have no ep-scope.md.
2. **Load existing table first** — If `ai-sdlc-artefacts/audit-report.md` exists, use it as baseline for rows with Status DONE/CANCELED.
3. **Get current status per epic** — For each epic, read `ep-scope.md` and extract **ID**, **Title** (Name), **Status**.
4. **Run audit for IN_PROGRESS** — For each epic with Status **IN_PROGRESS**, run the epic-level audit (§2), produce full report per §3, and write it to `ai-sdlc-artefacts/epics/<epic-id>/ep-audit-report.md`.
5. **Keep DONE/CANCELED rows unchanged** — Do not re-open their artefacts for deep review; preserve existing table values unless user explicitly requests full refresh.
6. **Refresh NEW rows minimally** — Ensure EP/Name/Status are current; keep coverage/report links as-is unless a report exists.
7. **Build and write report** — Compose project-level report (§3a) and write `ai-sdlc-artefacts/audit-report.md`.
8. **Output compact summary in chat** — Show changed rows and high-level totals only.

---

## 3. Output structure (ep-audit-report.md)

Use these elements (or user-agreed equivalents):

- **Document header** — **Date and time of creation**: use the **current date** (the date of the user's audit request). Take it from context (e.g. user_info "Today's date"); format e.g. `YYYY-MM-DD (UTC)` or `YYYY-MM-DD HH:MM UTC`. Purpose, pipeline link, links to ep-implementation-plan.md and ep-acceptance-criteria.md (and optionally ep-requirements.md).
- **Summary** — Pass/fail or overall status in one short paragraph.
- **Implementation vs plan** — For each task (or task group) from ep-implementation-plan, state status: done / pending / blocked. Reference task identifiers (e.g. Task 1.1, 1.2). Link to ep-implementation-plan.md where helpful.
- **Test results and coverage** — Command run (e.g. `make check`), pass/fail, and **total test coverage** (the total line from the coverage output, e.g. `total: (statements) XX.X%`). Optionally include per-package breakdown. Reference which AC (AC-EE.NNN) are covered by which tests where applicable.
- **REQ/AC test coverage matrix** — **Generate this inside the report** (do not reference a separate ep-req-ac-test-coverage.md; that file is not a pipeline artefact). Use `./bin/validate EP-XXX --json` as primary source. Build a table that maps each AC to test levels and links:
  - Columns: AC | REQ | Unit | Integration | E2E | Manual | Link.
  - One row per AC from ep-acceptance-criteria.md. **AC and REQ columns must use markdown links** to the epic artefacts: e.g. `[AC-01.001](ep-acceptance-criteria.md#ac-01-001)`, `[REQ-01.001](ep-requirements.md#...)` (adjust anchors to match the document structure; AC anchors use hyphen form e.g. ac-01-001). REQ/AC IDs use the **REQ-EE.NNN** and **AC-EE.NNN** format (epic EE, number NNN). REQ values come from ep-requirements.md; fill Unit/Integration/E2E/Manual (e.g. ✓ or —) and Link (test file paths or manual scenario links) by scanning the codebase for `Covers AC-EE.NNN` / `Supporting AC-EE.NNN` comments and any project test strategy (e.g. strategy.md, ep-manual-test-scenarios.md). Deferred AC mark as deferred in Link.
  - Optionally add a short **Notes** subsection under the table (e.g. what Unit/Integration mean in this project, Make commands for test/coverage).
- **Quality gate** — Result of the checks run as part of `make check` (e.g. fmt, vet, lint, module boundaries): pass/fail or list of issues. If the project uses a single command like `make check`, state that it was run and its overall result.
- **Gaps, risks, recommendations** — **Gap:** something planned in the implementation plan or AC but not implemented or not verified. **Risk:** something that may cause problems later (e.g. tech debt, missing tests, instability). **Recommendations:** short actionable next steps.

**Traceability:** Implementation vs plan must reference task IDs from ep-implementation-plan. Test/AC mapping must reference AC-EE.NNN from ep-acceptance-criteria.md. Every link must point to an existing path under `ai-sdlc-artefacts/`.

**Example** (Implementation vs plan snippet):

```markdown
| Task   | Status  | Notes |
|--------|--------|-------|
| 1.1    | Done   | Config load and validation — [ep-implementation-plan](ep-implementation-plan.md) |
| 1.2    | Pending| LLM client wiring |
```

---

## 3a. Project-level output structure (audit-report.md)

When producing the **project-level** audit report, use this structure. File path: **ai-sdlc-artefacts/audit-report.md**.

- **Document header** — **Date and time of creation**: use the **current date** (the date of the user's audit request). Take it from context (e.g. user_info "Today's date"); format e.g. `YYYY-MM-DD (UTC)` or `YYYY-MM-DD HH:MM UTC`. Purpose: project-level audit summary. Link to [scope.md](scope.md) and [strategy.md](strategy.md). Do **not** include a line about `make check` or project coverage in the project-level report.
- **Mode note (token optimization)** — DONE/CANCELED rows are preserved from previous project audit unless the user explicitly requests full recomputation.
- **Epic summary table** — A single table with exactly these columns:

| Column | Content |
|--------|--------|
| **EP** | Markdown link to the epic scope: `[EP-XXX](epics/EP-XXX/ep-scope.md)`. |
| **Name** | Epic title (from ep-scope.md, "Title" field). |
| **Status** | Epic status (from ep-scope.md, "Status" field: NEW, IN_PROGRESS, CANCELED, DONE). |
| **Test coverage** | Total statement coverage from that epic's ep-audit-report.md if present (e.g. `76.1%`); otherwise `—`. |
| **ep_audit-report** | If the file exists: markdown link to ep-audit-report plus the **date of that report** (from its "Date and time" header); e.g. `[ep-audit-report (YYYY-MM-DD)](epics/EP-XXX/ep-audit-report.md)` or link and date. Otherwise `—`. |

**Example:**

```markdown
| EP | Name | Status | Test coverage | ep_audit-report |
|----|------|--------|----------------|-----------------|
| [EP-001](epics/EP-001/ep-scope.md) | PersonalAssistant MVP | DONE | 76.1% | [ep-audit-report (2026-03-16)](epics/EP-001/ep-audit-report.md) |
| [EP-002](epics/EP-002/ep-scope.md) | Automatic memory summarization | NEW | — | — |
| [EP-003](epics/EP-003/ep-scope.md) | Agent security hardening | NEW | — | — |
| [EP-004](epics/EP-004/ep-scope.md) | Structured tools and Tool-calling API | NEW | — | — |
```

All links must be relative to ai-sdlc-artefacts/ and point to existing paths. Do not add rows for epics that have no ep-scope.md unless you explicitly note them as missing.

---

## 4. Done when

**Epic-level audit:** Verify all before considering the stage complete:

- [ ] Full report was written to `ep-audit-report.md`.
- [ ] Chat output contains compact summary: quality gate, total coverage, key gaps/risks, report path.
- [ ] Report content (in file) includes: header with **date and time**, Summary, Implementation vs plan, Test results and coverage, **REQ/AC test coverage matrix** (AC and REQ columns use markdown links to ep-acceptance-criteria.md and ep-requirements.md), Quality gate, Gaps/risks/recommendations.
- [ ] Implementation vs plan references task IDs from ep-implementation-plan and states status (done/pending/blocked).
- [ ] Traceability to AC and implementation plan is kept; all links point to existing paths under `ai-sdlc-artefacts/`.
- [ ] Each test has comment with AC and REQ for tracing; each AC has test; each REQ has one or more AC.
- [ ] File write happened by default (unless user explicitly requested dry-run/no-write).

**Project-level audit:** Verify all before considering the stage complete:

- [ ] Project report was written to `ai-sdlc-artefacts/audit-report.md`.
- [ ] Chat output contains compact summary and changed-row highlights.
- [ ] Draft includes: header with **date and time**, and the **epic summary table** with columns: EP (link), Name, Status, Total coverage, ep_audit-report (link + report date when present, otherwise —).
- [ ] Every row corresponds to an epic that has ep-scope.md under ai-sdlc-artefacts/epics/; EP and ep_audit-report links point to existing paths.
- [ ] DONE/CANCELED rows were not re-reviewed or recomputed; existing values were preserved unless user explicitly requested full refresh.
