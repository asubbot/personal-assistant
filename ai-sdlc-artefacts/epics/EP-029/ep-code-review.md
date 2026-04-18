# EP-029 — Code review

## Review iteration 1

**Scope:** Working tree on `epic/EP-029-health-readiness-observability` — `cmd/pa` (observability HTTP, readiness, `main.go`, `jobs_runtime.go`, tests), `internal/config`, `internal/lifecyclelog`, `internal/memoryjob`, `internal/toolindex/build_log.go`, `docs/`, epic artefacts.

**Verification:** `make check` passed on the reviewed tree.

### Findings (resolved in iteration 2)

| Id | Severity | Summary |
|----|----------|---------|
| F-10.1 | Medium | Bind/listen failure only logged; bot continues — documented as intentional in `docs/observability-http.md` §Bind failures. |
| F-10.2 | Medium | AC-29.008 wording vs `go run` validate test — aligned `ep-acceptance-criteria.md`, `ep-requirements.md` REQ-29.008 with the automated test and CI entrypoint. |
| F-10.3 | Minor | `writeJSON` ignored JSON encode errors — fixed: log on encode failure; handler receives `logger`. |
| F-10.4 | Minor | Extra negative config tests — added empty `listen_address` and relative `health_path` fixtures + tests. |
| F-10.5 | Nit | Duplicate legacy + lifecycle logs — documented under §Duplicate log lines in operator doc. |
| F-10.6 | Nit | `duration_ms` when zero — call sites pass wall-clock duration from goroutine; acceptable. |

**Iteration summary — open counts:** Blocker: 0, Major: 1, Medium: 1, Minor: 2 *(Nit: 2 — not blocking per pipeline §2.2)*

---

## Review iteration 2

**Focus:** Closure on F-10.1–F-10.4 from iteration 1.

All four items verified resolved in the current tree (operator doc bind behaviour, REQ/AC alignment for validation, `writeJSON` logging, expanded config negative tests).

**Iteration summary — open counts:** Blocker: 0, Major: 0, Medium: 0, Minor: 0
