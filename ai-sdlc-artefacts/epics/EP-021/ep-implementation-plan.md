# EP-021 — Implementation plan

## Goal

Deliver REQ-21.001–REQ-21.012 and AC-21.001–AC-21.007 by simplifying the jobs wrapper, rewriting `create_scheduled_job`, adding an **optional** example skill template and native allowlist entry, and updating tests. **Critical path:** wrapper + tool; skill is operator-facing only.

## Tasks (order)

1. **Config allowlist** — Extend `AllowedNativeToolIDs` to include `create_scheduled_job` when `paths.jobs_db_path` is non-empty ([REQ-21.008](ep-requirements.md#requirements)).
2. **Example skill (optional)** — Add `config.examples/skills/scheduled-jobs/SKILL.md` with frontmatter and playbook for operators using `runtime_skills` ([REQ-21.007](ep-requirements.md#requirements)); NL create does not depend on it.
3. **Native tool** — Replace `create_scheduled_job` implementation: explicit `instruction`, `hour`, `minute`, optional `timezone` / ids; `creation_path` `native_tool_explicit`; remove regex helpers and `ErrCreateScheduledJobNoMatch` ([REQ-21.004](ep-requirements.md#requirements) [REQ-21.005](ep-requirements.md#requirements) [REQ-21.010](ep-requirements.md#requirements)).
4. **Manager** — Remove `HandleNaturalLanguageCreate`, `createTool` field, and strict NL regex helpers ([REQ-21.003](ep-requirements.md#requirements)).
5. **Jobs wrapper** — `HandleMessage`: only `handleJobsCommand` then `base.HandleMessage` with `WithCreateContext`; delete fallback helpers ([REQ-21.002](ep-requirements.md#requirements) [REQ-21.003](ep-requirements.md#requirements)).
6. **Tests** — Update `internal/jobs/manager_test.go`, `cmd/pa/jobs_runtime_test.go`, `cmd/pa/ep020_e2e_test.go`; add runtimeskills test for skill validation ([REQ-21.012](ep-requirements.md#requirements)).
7. **Docs** — Optional one-line in `docs/configuration.md` for operators copying the example skill.
8. **Verification** — `make check` and `./bin/validate EP-021`.

## Non-goals

- Edits to `systemStaticHead` strings ([REQ-21.006](ep-requirements.md#requirements) — verified manually via AC-21.006 deferred).
