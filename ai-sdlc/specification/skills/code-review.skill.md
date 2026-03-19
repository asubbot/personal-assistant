---
name: code-review.skill
description: >-
  Perform structured code review on a change set (PR, branch diff, or explicit paths).
  Use when the user asks for code review, PR review, security review of changes,
  or pre-merge quality check aligned with project rules (KISS, fail fast, AGENTS.md).
---

# Code review

**Pipeline:** Optional; not a numbered SDLC stage. Complements [09-audit.skill.md](09-audit.skill.md) (audit = plan/tests/coverage) and manual inspection before merge.  
**Output:** The review is **shown in chat** in full. A file under `ai-sdlc-artefacts/` is **not** created unless the user explicitly asks to save (e.g. "save", "write to file", "lgtm").

---

## 1. Context and goal

You are a senior reviewer. Your task is to review a **bounded change set** and report findings in **English**.

**Goal:** Give actionable, ordered feedback: correctness, safety, maintainability, and test gaps—aligned with **KISS**, **fail fast**, and [AGENTS.md](../../../AGENTS.md) (cooperation with the user; do not change source files unless the user allows).

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
3. **Apply the checklist (§3)** — Systematically walk through categories; note **severity** (Blocker / Major / Minor / Nit / Suggestion).
4. **Tests** — If the change is non-trivial and the repo has a standard check command, you **may** suggest the user run **`make check`**; run it **only if** the user asked for verification or Agent mode allows running commands. Record pass/fail in the review when you ran it.
5. **Output in chat** — Always output the **full** review using the structure in §4.
6. **Save only when requested** — Write a file (e.g. `ai-sdlc-artefacts/reviews/code-review-YYYY-MM-DD-<topic>.md`) **only** when the user explicitly asks to save. Use relative links if the review references epic artefacts.

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
2. **Summary** — 2–4 sentences: merge recommendation (approve / approve with nits / request changes) and why.
3. **Blockers** — Must fix before merge (empty if none).
4. **Findings** — Table or bullet list: **Severity** | Location | Issue | Recommendation.
5. **Test / verification** — What should be run (e.g. `make check`) and result if you ran it.
6. **Residual risks / follow-ups** — Optional; out-of-scope items for later.

---

## 5. Done when

- [ ] Scope is explicit or confirmed with the user.
- [ ] Full review delivered in chat using §4.
- [ ] Findings reference concrete locations where possible.
- [ ] No unsolicited edits or commits; English throughout.
- [ ] **If** the user asked to save: a markdown file was written under `ai-sdlc-artefacts/` (or path they specified) **after** that request.

---

## 6. Reference

**Project rules:** [AGENTS.md](../../../AGENTS.md)  
**Audit (status vs plan):** [09-audit.skill.md](09-audit.skill.md)  
**Pipeline:** [pipeline.spec.md](../pipeline.spec.md)
