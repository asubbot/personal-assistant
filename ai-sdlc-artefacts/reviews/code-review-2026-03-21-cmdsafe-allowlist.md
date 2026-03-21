# Code review: cmdsafe rune policy and allowlist hardening

Structured review per [code-review.skill.md](../../ai-sdlc/specification/skills/code-review.skill.md). Scope: uncommitted changes at review time (tracked files + new `internal/cmdsafe/runes*.go`).

---

## 1. Scope

- **Tracked modifications:** `ai-sdlc-artefacts/threat-model.md`, `config.examples/nas_allowlist.example`, `docs/configuration.md`, `docs/troubleshooting.md`, `internal/allowlist/*`, `internal/cmdsafe/shellmeta.go`, `internal/config/load.go` (single error-string tweak), `internal/core/adapter.go`, `internal/core/handler.go`, `internal/core/handler_test.go`, `internal/escalationpolicy/*`, `internal/noderunner/*`
- **New files:** `internal/cmdsafe/runes.go`, `internal/cmdsafe/runes_test.go`

Untracked local artefacts (e.g. `.config/`, `.data/`) were out of scope.

---

## 2. Summary

The change set tightens **remote command safety**: a **UTF-8 + allowlist-of-runes** gate (`RejectDisallowedRunes`) runs **before** shell-metacharacter checks and the node allowlist; allowlist files **reject bare `*`** and **multiple trailing `*`**. Escalation policy, docs, threat model, and tests are aligned. **`make check` passed** (fmt, vet, golangci-lint, race + integration tests).

**Recommendation:** **Approve with nits** — no blockers; call out **operational / compatibility** impact of the new rune policy and one small documentation/threat-model completeness nit.

---

## 3. Blockers

None.

---

## 4. Findings

| Severity | Location | Issue | Recommendation |
|----------|----------|--------|----------------|
| **Major** (compatibility / product) | `internal/cmdsafe/runes.go` (`allowedASCIIPunct`, `runeAllowed`) | Allowed punctuation is **narrow** (e.g. `~`, `*`, `?`, `` ` ``, quotes, `%`, glob chars are rejected). Legitimate remote paths or arguments using those characters will **fail** even if allowlisted. | Treat as intentional security posture; add a short **“migration / limitations”** note (e.g. in `docs/configuration.md` or release notes): operators must use paths/commands that fit the allowlist, or adjust templates. |
| **Minor** | `internal/allowlist/allowlist.go` — `match()` | For patterns ending in `*`, `prefix == "" \|\| strings.HasPrefix(...)` remains; after `validateAllowlistPattern`, empty prefix should not occur for loaded `*` patterns. | Optional: simplify `match` or add a one-line comment that `prefix == ""` is defensive only. |
| **Minor** | Allowlist semantics | `validateAllowlistPattern` only constrains **trailing** `*`; a pattern like `a*b*` (internal `*` + trailing `*`) is still loadable. Docs stress “wildcard **at end**” but do not spell out internal `*`. | If internal `*` is unintended, reject it; if allowed, document one example so operators are not surprised. |
| **Nit** | `ai-sdlc-artefacts/threat-model.md` | Remote-exec row cites `runner.go` for cmdsafe; **`internal/core/handler.go`** also applies `RejectDisallowedRunes` / `RejectShellMetacharacters` before `RunOnNode`. | Mention core path for completeness (trust-boundary narrative). |
| **Nit** | `internal/cmdsafe/runes_test.go` — `TestRejectDisallowedRunes_maxLength` | Loop uses `sb.Len()` (bytes) vs `utf8.RuneCountInString`; safe today because only ASCII `a` is written. | If the test ever uses multi-byte runes, use **rune count** explicitly to avoid a misleading test. |
| **Suggestion** | Duplication | `executeOneToolCall` and `Runner.RunOnNode` both run the same cmdsafe sequence. | Acceptable **defence in depth**; optional future refactor is a shared `ValidateRemoteCommand(string) error` if you want a single choke point. |

---

## 5. Test / verification

- Ran: **`make check`**
- **Result:** **PASS** (including `-race` and `-tags=integration`).

---

## 6. Residual risks / follow-ups

- **Breaking config:** Deployments that relied on a lone `*` allowlist line will **fail at startup** until fixed — mitigated by docs and example updates.
- **Error UX:** Some inputs now fail with **“disallowed character U+…“** instead of shell-meta wording (e.g. `;`) — behaviour is correct; operators may need to map hex to characters when reading logs.
