# Code review — EP-027 Composition root and application lifecycle

**Reviewer:** Pipeline stage 10

---

## Review iteration 1

**Review date:** 2026-04-17  
**Stage 10 iteration:** 1 of max 5  
**Scope:** `cmd/pa/main.go`, `cmd/pa/application.go`, `cmd/pa/setup_infra.go`, `cmd/pa/ep027_startup_policy_test.go`, `internal/jobs/create_scheduled_job_tool.go`, `internal/jobs/manager_test.go`

**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0 | Nit: 0 | Suggestion: 1  
**Gate:** Pass

### Summary

Composition splits infrastructure construction from application wiring; jobs create-tool uses runtime snapshot for user-soft messages. Startup sources carry no `gocyclo` nolint, matching [AGENTS.md](../../AGENTS.md). **Approve** for merge.

### Findings

| Severity | Location | Issue | Recommendation |
|----------|----------|-------|----------------|
| Suggestion | `paApplication.buildMessageHandler` | Jobs async init still races user tools; unchanged from prior design. | Document in ops runbook if operators need deterministic readiness probes beyond `/jobs`. |

### Test / verification

See [ep-audit-report.md](ep-audit-report.md).
