# Code review: ValidateRemoteCommand, allowlist `*` rules, docs and audits

Structured review per [code-review.skill.md](../../ai-sdlc/specification/skills/code-review.skill.md). Scope: uncommitted changes at review time (second pass after refactor).

---

## 1. Scope

**Tracked modifications:**  
`ai-sdlc-artefacts/epics/EP-001/ep-audit-report.md`, `ai-sdlc-artefacts/epics/EP-004/ep-audit-report.md`, `ai-sdlc-artefacts/threat-model.md`, `config.examples/nas_allowlist.example`, `docs/configuration.md`, `docs/troubleshooting.md`, `internal/allowlist/*`, `internal/cmdsafe/shellmeta.go`, `internal/config/load.go`, `internal/core/adapter.go`, `internal/core/handler.go`, `internal/core/handler_test.go`, `internal/escalationpolicy/*`, `internal/noderunner/*`

**New / untracked:**  
`internal/cmdsafe/remote.go`, `internal/cmdsafe/remote_test.go`, `internal/cmdsafe/runes.go`, `internal/cmdsafe/runes_test.go`, `ai-sdlc-artefacts/reviews/` (directory)

**Excluded from code review (artefacts):** untracked `ai-sdlc.zip` at repo root (likely accidental bundle).

---

## 2. Summary

The refactor introduces a **single ordered gate** `cmdsafe.ValidateRemoteCommand` (runes/UTF-8/length, then REQ-04.031 shell substrings), with **`CommandValidationError` + `RejectKind`** so `noderunner` can still distinguish **shell-meta** vs **rune/charset** outcomes for escalation policy. Allowlist validation is **stricter**: `*` is allowed **only** as a **single terminal** wildcard (rejects `foo*bar`, bare `*`, `**`, etc.). Documentation, threat model, and epic audit tables are updated; tests cover the new API and allowlist rules.

**Recommendation:** **Approve with nits** — design is coherent, tests and CI are green; remaining notes are small (defensive branch, repo hygiene).

---

## 3. Blockers

None.

---

## 4. Findings

| Severity | Location | Issue | Recommendation |
|----------|----------|--------|----------------|
| **Major** (compatibility) | `internal/cmdsafe/runes.go` — `allowedASCIIPunct` | **Narrow** allowed punctuation; paths/args with `~`, quotes, glob chars, NBSP, etc. remain rejected even when allowlisted. | Intentional security posture; already better explained in `config.examples/nas_allowlist.example` and `docs/configuration.md`. Ensure release/operator notes mention it if you ship this. |
| **Minor** | `internal/noderunner/runner.go` — `RunOnNode` | On `ValidateRemoteCommand` failure, if `RejectKind` returns `ok == false`, the error is wrapped as **`NodeOutcomeDisallowedRunes`**. Today `ValidateRemoteCommand` only returns `*CommandValidationError`, so `ok` should always be true; the branch is **defensive / future-proofing** but slightly misleading if a future change returns a plain `error`. | Add a one-line comment (“unexpected error shape; treat as rune policy failure”), or use `panic`/`fmt.Errorf` in development-only builds — only if you want stricter invariants; otherwise comment is enough. |
| **Nit** | Repo root | Untracked **`ai-sdlc.zip`** | Do not commit; delete or add to `.gitignore` if it is a local backup pattern. |
| **Nit** | `internal/cmdsafe/runes_test.go` | `TestRejectDisallowedRunes_maxLength` still ties length to `strings.Builder.Len()` (bytes) while the limit is in **runes**; safe for ASCII-only loop. | If the test ever uses non-ASCII runes, count runes explicitly. |
| **Suggestion** | `internal/cmdsafe/remote.go` | `CommandValidationError.Unwrap` is untested in isolation (coverage report shows 0% for `Unwrap`); behaviour is standard. | Optional: tiny test that `errors.Is` / `Unwrap` reaches the inner error. |

**Positive (vs prior review):** `validateAllowlistPattern` now rejects **internal** `*` (e.g. `foo*bar`), with a dedicated test — this closes the earlier “ambiguous mid-line `*`” gap. `remote_test.go` documents that **semicolon** is classified as **runes** first (correct for current ordering). Threat model now names **`ValidateRemoteCommand`** and both **runner** and **handler** paths.

---

## 5. Test / verification

- Ran: **`make check`**
- **Result:** **PASS** (fmt, vet, golangci-lint, `-race`, `-tags=integration`, module boundaries).

---

## 6. Residual risks / follow-ups

- **Breaking allowlists:** Any pattern with `*` not exactly once at EOL, or bare `*`, fails at load — operators must update files (mitigated by examples and docs).
- **Double validation:** `executeOneToolCall` runs `ValidateRemoteCommand` before `RunOnNode`, which runs it again — acceptable **defence in depth**; small redundant CPU on the hot path.
- **Commit hygiene:** Decide whether `ai-sdlc-artefacts/reviews/*.md` and new `internal/cmdsafe/remote*.go` should be tracked in git (expected yes for product code; reviews optional per your process).
