---
name: code-review.skill
description: >-
  Perform structured code review on a change set (PR, branch diff, or explicit paths) — pipeline stage 10.
  Use when the user asks for code review, PR review, security review of changes,
  or pre-merge quality check aligned with project rules (KISS, fail fast; see repo AGENTS.md and [ai-sdlc README](../../README.md)).
---

# Stage 10: Code review

**Pipeline:** [pipeline.spec.md](../pipeline.spec.md). Complements [11-audit.skill.md](11-audit.skill.md) (audit = plan/tests/coverage). Runs after task execution (stage 9) and before audit (stage 11).  
**Output:** The review is **shown in chat** in full. A file is **not** created unless the user explicitly asks to save (e.g. "save", "write to file", "lgtm"). **Recommended epic path when saving:** `ai-sdlc-artefacts/epics/<epic-id>/ep-code-review.md`; other paths (e.g. `ai-sdlc-artefacts/reviews/...`) only if the user prefers. For the **§2.2** iteration loop, persist each iteration under **`## Review iteration N`** in that file when the user approves save (see **Code–review iteration** below).

## Code–review iteration ([pipeline.spec.md](../pipeline.spec.md) §2.2)

Stages **9** and **10** repeat until **zero** open findings in **Blocker**, **Major**, **Medium**, and **Minor**, or until the **operator decides** after the cap.

1. **Count iterations** — Each completed save of a **`## Review iteration N`** section in `ep-code-review.md` (or a full review recorded as iteration **N** per operator agreement) is one stage 10 iteration. **N** must not exceed **5** without an explicit operator decision recorded in chat or in the review file.
2. **Single file** — Use one `ep-code-review.md` per epic when persisting. For iteration **N**, add a **top-level** heading `## Review iteration N` with stable increasing **N**. **Retain** prior iteration sections.
3. **Exit loop** — After this iteration, if **Blocker**, **Major**, **Medium**, and **Minor** open counts are all **zero**, the iteration loop is **complete**; stage 11 may follow. **Nit** and **Suggestion** do not block.
4. **Cap** — If **N = 5** and any **Blocker**, **Major**, **Medium**, or **Minor** is still **> 0**, **stop** and require an **operator decision** before further stage 9/10 work or stage 11.
5. **Return to stage 9** — When Blocker/Major/Medium/Minor > 0 and **N < 5**, the orchestrator runs **stage 9** again to fix the codebase, then runs **stage 10** again (new **delegated** session per pipeline §3) on the updated change set.

## Mandatory delegation (pipeline stage 10)

When this skill is run as **pipeline stage 10**, execution MUST follow [pipeline.spec.md](../pipeline.spec.md) **§3**:

- **If you are the orchestrator** (you executed stage 9 / authored the change): **do not** perform the full structured review yourself in the same session. **Delegate** to a **subagent** or a **new chat** with fresh context; pass the agreed scope (PR URL, `base..head`, or file paths) and instruct the delegate to run this skill only.
- **If you are the delegated reviewer:** follow §1–§4 below; stay **readonly** on the repo unless the user explicitly asks for edits; output in chat first; save `ep-code-review.md` only when the user requests.

---

## 1. Context and goal

You are a senior reviewer. Your task is to review a **bounded change set** and report findings in **English**.

**Goal:** Give actionable, ordered feedback: correctness, safety, maintainability, and test gaps—aligned with **KISS**, **fail fast**, [AGENTS.md](../../../AGENTS.md) (workspace: user cooperation, permissions), and the SDLC expectations in [ai-sdlc README](../../README.md) / [pipeline.spec.md](../pipeline.spec.md).

**Inputs (resolve before reviewing):**

- **Scope** — One of: GitHub PR URL (describe findings from diff if only URL is given and tools allow), **base..head** branch pair, **list of file paths**, or “current uncommitted changes” if the user specifies.
- **Optional context** — Epic/requirements links under `ai-sdlc-artefacts/` if the user asks for review **against** EP-XXX; then cross-check behaviour against ep-requirements / ep-system-design for the touched areas only.

**Questions to answer:** Are there bugs, security issues, or API/contract breaks? Are errors handled explicitly? Are tests and observability adequate for the change? What would you change (suggestions only unless the user requests edits)?

**Constraints:** Be direct and specific (file + symbol or line region when possible). When scope is ambiguous (whole repo vs one PR), present options and ask the user to choose. Do not invent behaviour not visible in the diff/code.

**Rules:** All review text in **English**. Comments in code samples must be English. Do **not** commit or edit the repo unless the user explicitly asks.

---

## 2. Workflow

1. **Confirm scope** — If the user did not specify PR, branch, or paths, ask. If they want “full codebase review”, warn that it is broad; suggest narrowing to a PR or directory.
2. **Gather the diff** — Read changed files (and immediate callers/callees if needed for context). Prefer minimal context: only files relevant to the change.
3. **Apply the checklist (§3)** — Systematically walk through categories; note **severity** using the definitions below (**Blocker**, **Major**, **Medium**, **Minor** gate **§2.2**; **Nit** and **Suggestion** do not).
4. **Tests** — If the change is non-trivial and the repo has a standard check command, you **may** suggest the user run **`make check`**; run it **only if** the user asked for verification or Agent mode allows running commands. Record pass/fail in the review when you ran it.
5. **Output in chat** — Always output the **full** review using the structure in §4 (include **Iteration summary — open counts** for Blocker / Major / Medium / Minor / Nit / Suggestion when **§2.2** applies).
6. **Save only when requested** — Write a file **only** when the user explicitly asks to save. Prefer `ai-sdlc-artefacts/epics/<epic-id>/ep-code-review.md` for epic-scoped reviews; otherwise e.g. `ai-sdlc-artefacts/reviews/code-review-YYYY-MM-DD-<topic>.md` if the user prefers. For **§2.2**, append **`## Review iteration N`** to `ep-code-review.md` (preserve prior sections). Use relative links if the review references epic artefacts.

---

## 2a. Severity definitions (align with pipeline §2.2)

| Severity | Meaning |
|----------|---------|
| **Blocker** | Must not merge: correctness bug, security flaw, broken contract, data loss, or CI-breaking defect. |
| **Major** | Should block merge until fixed: significant gap (missing tests for new critical path, serious API misuse, missing error handling for required flow). |
| **Medium** | Should fix before merge if time allows, or track immediately after: maintainability, incomplete edge cases, doc/code drift that misleads. |
| **Minor** | Polish, low-risk cleanups; blocks **§2.2** exit until resolved or operator decision at cap. |
| **Nit** | Style-only or preference; does not block exit. |
| **Suggestion** | Optional improvement; does not block exit. |

**§2.2 gate:** Exit the 9↔10 loop only when open **Blocker** = 0, **Major** = 0, **Medium** = 0, and **Minor** = 0 for the current iteration.

---

## 3. Review checklist

Use as a guide; omit categories that do not apply.

| Area | Look for |
|------|----------|
| **Correctness** | Logic errors, off-by-one, races, wrong conditions, ignored errors, nil/empty handling. |
| **Fail fast** | Invalid state detected early; no silent swallow of errors without documented reason. |
| **Security** | Injection (shell, SQL if any), path traversal, secrets in logs/config, trust boundaries (SSH, tokens, user input to tools). |
| **Concurrency** | Goroutine leaks, shared mutable state without synchronization, context cancellation. |
| **API / compatibility** | Breaking changes to exported symbols or config without migration notes. |
| **Performance** | Hot-path allocations, unbounded buffers, N+1 I/O; only if relevant to the change. |
| **Tests** | New behaviour untested; edge cases; table-driven coverage; integration tests for I/O boundaries. |
| **Observability** | Useful logs at appropriate level; no sensitive data in logs. |
| **Style & KISS** | Idiomatic Go (if Go project), small functions, clear naming, no unnecessary abstraction. |
| **Docs / artefacts** — If the user linked an epic, flag mismatches between code and stated REQ/AC/design. |

---

## 4. Output structure (chat)

Use this layout (or user-agreed equivalent):

1. **Scope** — What was reviewed (PR, commits, files).
2. **Iteration summary — open counts** — (when **§2.2** applies) Blocker: *n* \| Major: *n* \| Medium: *n* \| Minor: *n* \| Nit: *n* \| Suggestion: *n*.
3. **Gate (§2.2)** — Pass \| Fail \| Cap (iteration 5 and Blocker/Major/Medium/Minor still > 0 — operator decision required).
4. **Summary** — 2–4 sentences: merge recommendation (approve / approve with nits / request changes) and why.
5. **Blockers** — Must fix before merge (empty if none).
6. **Findings** — Table or bullet list: **Severity** | Location | Issue | Recommendation.
7. **Test / verification** — What should be run (e.g. `make check`) and result if you ran it.
8. **Residual risks / follow-ups** — Optional; out-of-scope items for later.

### Saved file layout (`ep-code-review.md`, when user approves save)

On **first** save for an epic, create the file with a document title once. On **later** iterations, **append** `## Review iteration N` at the end; keep prior sections.

```markdown
# Code review — EP-XXX [optional title]

---

## Review iteration N

**Review date:** YYYY-MM-DD
**Stage 10 iteration:** N of max 5
**Scope:** [PR / branch / paths]
**Iteration summary — open counts:** Blocker: X | Major: X | Medium: X | Minor: X | Nit: X | Suggestion: X
**Gate:** Pass | Fail | Cap — operator decision required

### Summary
...

### Findings
(Severity | Location | Issue | Recommendation)
```

---

## 5. Done when

- [ ] Scope is explicit or confirmed with the user.
- [ ] Full review delivered in chat using §4.
- [ ] **If §2.2:** iteration number **N**, open counts, and Gate filled; prior `## Review iteration …` sections preserved when **N > 1** and saving to file.
- [ ] Findings reference concrete locations where possible.
- [ ] No unsolicited edits or commits; English throughout.
- [ ] **If** the user asked to save: a markdown file was written under `ai-sdlc-artefacts/` (or path they specified) **after** that request, with **`## Review iteration N`** when **§2.2** applies.

---

## 6. Reference

**Workspace rules:** [AGENTS.md](../../../AGENTS.md)  
**SDLC entry (pipeline + agent behaviour):** [ai-sdlc README](../../README.md)  
**Audit (status vs plan):** [11-audit.skill.md](11-audit.skill.md)  
**Pipeline:** [pipeline.spec.md](../pipeline.spec.md)
