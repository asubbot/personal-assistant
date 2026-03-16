# EP-001 PersonalAssistant MVP — Audit Report

**Date and time:** 2026-03-16 (UTC)  
**Purpose:** Stage 9 audit — implementation vs plan, tests, coverage, quality gate, gaps/risks.  
**Pipeline:** [ai-sdlc/specification/pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md)  
**Epic artefacts:** [ep-implementation-plan.md](ep-implementation-plan.md), [ep-acceptance-criteria.md](ep-acceptance-criteria.md), [ep-requirements.md](ep-requirements.md), [ep-manual-test-scenarios.md](ep-manual-test-scenarios.md)

---

## Summary

**Status: PASS.** All in-scope implementation plan tasks are done except the intentionally deferred task 10.2 (versioned state). `make check` passes (fmt, vet, golangci-lint, tests with integration tag, module boundaries). Total statement coverage **76.1%**. No open quality gate issues. Remaining gaps: part of AC covered only manually (AC-004, AC-032); no dedicated E2E; two-user SSH image uses Debian (Alpine was unstable on some hosts).

---

## Implementation vs plan

| Task | Status | Notes |
|------|--------|-------|
| 1 | Done | Go module, skeleton, config load — [ep-implementation-plan](ep-implementation-plan.md) §1 |
| 1.1 | Done | Config load and validation |
| 1.2 | Done | Unit tests for config validation |
| 2 | Done | Checkpoint §1 |
| 2.1 | Done | Per-node allowlist model |
| 2.2 | Done | Dedicated SSH user per node |
| 2.3 | Done | Unit tests allowlist + dedicated user |
| 3 | Done | Checkpoint §2 |
| 3.1–3.6 | Done | LLM provider, Telegram adapter, core, message validation, integration tests, DEBUG LLM logging |
| 4 | Done | Checkpoint §3 |
| 4.1–4.4 | Done | Memory store, vector interface + SQLite, wiring, tests |
| 5 | Done | Checkpoint §4 |
| 5.1–5.4 | Done | SSH client, core integration, integration tests, `-verify-nodes` CLI |
| 6 | Done | Checkpoint §5 |
| 6.1–6.5 | Done | Tool contract, scheduler, wiring, config add node/tool, tests |
| 7 | Done | Checkpoint §6 |
| 7.1–7.4 | Done | LLM logging, unavailable destination, redaction, unit tests |
| 8 | Done | Checkpoint §7 |
| 8.1, 8.1b, 8.2 | Done | Day/month/year summarization, CLI, tests |
| 9 | Done | Checkpoint §8 |
| 9.1–9.2 | Done | Dockerfile, docker-compose, container start verification |
| 10 | Done | Checkpoint §9 |
| 10.1 | Done | Module boundaries (script, no cycles/forbidden edges) |
| 10.2 | **Deferred** | Versioned state (REQ-016, AC-026, AC-027) — out of MVP scope |
| 11.1 | Done | LLM provider fallback (REQ-031) |
| 12.1 | Done | Secret leakage protection (unit + integration) |
| 13.1 | Done | Final checkpoint |

---

## Test results and coverage

- **Command run:** `make check` (go fmt, go vet, golangci-lint with `--build-tags=integration`, go test with coverage and module-boundary check).
- **Result:** **PASS** (exit code 0).
- **Total coverage:** **76.1%** of statements (`total: (statements) 76.1%` from `go tool cover -func=coverage.out`).
- **Per-package (selected):** cmd/pa 20.6%, internal/config 10.6%, internal/core 6.6%, internal/allowlist 2.9%, internal/llm 7.1%, internal/scheduler 6.1%, internal/ssh 1.7%, internal/summarize 13.1%, internal/telegram 4.0%, internal/tools 2.8%, internal/vector/sqlite 3.5%, tests/integration 22.2%. `internal/logging` and `internal/vector` have no test files (excluded from coverage).

Integration tests (tag `integration`) run with `make check` and include: Telegram flow (AC-001, AC-002, AC-016), SSH single-user and two-user (AC-006, AC-010), node runner allowlist (AC-007, AC-008), memory/vector (AC-013, AC-014), scheduler (AC-020, AC-021, AC-024), secret leakage (AC-028, AC-029, AC-030).

---

## REQ/AC test coverage matrix

| AC | REQ | Unit | Integration | E2E | Manual | Link |
|----|-----|------|-------------|-----|--------|------|
| [AC-001](ep-acceptance-criteria.md#ac-001) | [REQ-001](ep-requirements.md#interface-and-deployment) | ✓ | ✓ | — | — | internal/core/handler_test.go, internal/telegram/adapter_test.go; tests/integration/telegram_flow_test.go |
| [AC-002](ep-acceptance-criteria.md#ac-002) | [REQ-001](ep-requirements.md#interface-and-deployment) | ✓ | ✓ | — | — | internal/core/handler_test.go, internal/telegram/adapter_test.go; tests/integration/telegram_flow_test.go |
| [AC-003](ep-acceptance-criteria.md#ac-003) | [REQ-001](ep-requirements.md#interface-and-deployment), [REQ-002](ep-requirements.md#interface-and-deployment) | ✓ | — | — | — | internal/core/run_test.go, internal/telegram/adapter_test.go |
| [AC-004](ep-acceptance-criteria.md#ac-004) | [REQ-002](ep-requirements.md#interface-and-deployment) | — | — | — | ✓ | [ep-manual-test-scenarios.md](ep-manual-test-scenarios.md#ac-004) |
| [AC-005](ep-acceptance-criteria.md#ac-005) | [REQ-003](ep-requirements.md#nodes-and-ssh), [REQ-024](ep-requirements.md#nodes-and-ssh) | ✓ | — | — | — | internal/config/config_test.go |
| [AC-006](ep-acceptance-criteria.md#ac-006) | [REQ-004](ep-requirements.md#nodes-and-ssh) | ✓ | ✓ | — | — | internal/ssh/client_test.go; tests/integration/ssh_client_test.go, ssh_node_test.go |
| [AC-007](ep-acceptance-criteria.md#ac-007) | [REQ-005](ep-requirements.md#nodes-and-ssh) | ✓ | ✓ | — | — | internal/allowlist/allowlist_test.go; tests/integration/ssh_node_test.go |
| [AC-008](ep-acceptance-criteria.md#ac-008) | [REQ-005](ep-requirements.md#nodes-and-ssh) | ✓ | ✓ | — | — | internal/allowlist/allowlist_test.go, internal/tools/run_on_node_test.go; tests/integration/ssh_node_test.go |
| [AC-009](ep-acceptance-criteria.md#ac-009) | [REQ-013](ep-requirements.md#nodes-and-ssh) | ✓ | ✓ | — | — | internal/ssh/client_test.go; tests/integration/ssh_node_test.go |
| [AC-010](ep-acceptance-criteria.md#ac-010) | [REQ-013](ep-requirements.md#nodes-and-ssh) | ✓ | ✓ | — | — | internal/noderunner/runner_test.go, internal/ssh/ssh_test.go; tests/integration/ssh_client_test.go |
| [AC-011](ep-acceptance-criteria.md#ac-011) | [REQ-006](ep-requirements.md#memory-and-indexing), [REQ-019](ep-requirements.md#memory-and-indexing) | ✓ | — | — | — | internal/summarize/summarize_test.go, cmd/pa/main_test.go |
| [AC-012](ep-acceptance-criteria.md#ac-012) | [REQ-006](ep-requirements.md#memory-and-indexing) | ✓ | — | — | — | internal/summarize/summarize_test.go, internal/memory (store) |
| [AC-013](ep-acceptance-criteria.md#ac-013) | [REQ-007](ep-requirements.md#memory-and-indexing) | ✓ | ✓ | — | — | internal/vector/sqlite/store_test.go, internal/core/handler_test.go; tests/integration/memory_vector_test.go |
| [AC-014](ep-acceptance-criteria.md#ac-014) | [REQ-007](ep-requirements.md#memory-and-indexing) | ✓ | ✓ | — | — | internal/vector/sqlite/store_test.go, internal/core/handler_test.go; tests/integration/memory_vector_test.go |
| [AC-015](ep-acceptance-criteria.md#ac-015) | [REQ-008](ep-requirements.md#llm-and-logging) | ✓ | — | — | — | internal/llm/openai_test.go, internal/llm/provider_test.go |
| [AC-016](ep-acceptance-criteria.md#ac-016) | [REQ-008](ep-requirements.md#llm-and-logging) | — | ✓ | — | — | tests/integration/telegram_flow_test.go |
| [AC-017](ep-acceptance-criteria.md#ac-017) | [REQ-014](ep-requirements.md#llm-and-logging) | ✓ | — | — | — | internal/llmlog/llmlog_test.go |
| [AC-018](ep-acceptance-criteria.md#ac-018) | [REQ-015](ep-requirements.md#llm-and-logging) | ✓ | — | — | — | internal/llmlog/llmlog_test.go, internal/core/handler_test.go |
| [AC-019](ep-acceptance-criteria.md#ac-019) | [REQ-015](ep-requirements.md#llm-and-logging) | ✓ | — | — | — | internal/llmlog/llmlog_test.go |
| [AC-020](ep-acceptance-criteria.md#ac-020) | [REQ-009](ep-requirements.md#scheduler-and-tools), [REQ-023](ep-requirements.md#scheduler-and-tools) | ✓ | ✓ | — | — | internal/telegram/adapter_test.go; tests/integration/scheduler_config_test.go |
| [AC-021](ep-acceptance-criteria.md#ac-021) | [REQ-009](ep-requirements.md#scheduler-and-tools) | ✓ | ✓ | — | — | internal/scheduler/scheduler_test.go; tests/integration/scheduler_config_test.go |
| [AC-022](ep-acceptance-criteria.md#ac-022) | [REQ-010](ep-requirements.md#scheduler-and-tools) | ✓ | — | — | — | internal/tools/registry_test.go, internal/tools/run_on_node_test.go |
| [AC-023](ep-acceptance-criteria.md#ac-023) | [REQ-010](ep-requirements.md#scheduler-and-tools) | ✓ | — | — | — | internal/tools/registry_test.go, internal/tools/run_on_node_test.go |
| [AC-024](ep-acceptance-criteria.md#ac-024) | [REQ-011](ep-requirements.md#extensibility-and-architecture) | ✓ | ✓ | — | — | internal/scheduler/loader_test.go; tests/integration/scheduler_config_test.go |
| [AC-025](ep-acceptance-criteria.md#ac-025) | [REQ-012](ep-requirements.md#extensibility-and-architecture) | — | — | — | ✓ | scripts/check-module-boundaries.sh (automated); strategy §2.3 manual |
| [AC-026](ep-acceptance-criteria.md#ac-026) | [REQ-016](ep-requirements.md#version-control-and-audit) | — | — | — | — | **Deferred** (post-MVP) |
| [AC-027](ep-acceptance-criteria.md#ac-027) | [REQ-016](ep-requirements.md#version-control-and-audit) | — | — | — | — | **Deferred** (post-MVP) |
| [AC-028](ep-acceptance-criteria.md#ac-028) | [REQ-017](ep-requirements.md#secret-protection-prompt-injection--exfiltration) | ✓ | ✓ | — | — | internal/core/run_test.go, handler_test.go; tests/integration/secret_leakage_test.go |
| [AC-029](ep-acceptance-criteria.md#ac-029) | [REQ-017](ep-requirements.md#secret-protection-prompt-injection--exfiltration) | — | ✓ | — | — | tests/integration/secret_leakage_test.go |
| [AC-030](ep-acceptance-criteria.md#ac-030) | [REQ-017](ep-requirements.md#secret-protection-prompt-injection--exfiltration) | — | ✓ | — | — | internal/llmlog/llmlog_test.go (redactor); tests/integration/secret_leakage_test.go |
| [AC-031](ep-acceptance-criteria.md#ac-031) | [REQ-021](ep-requirements.md#llm-and-logging) | ✓ | — | — | — | internal/core/handler_test.go |
| [AC-032](ep-acceptance-criteria.md#ac-032) | [REQ-022](ep-requirements.md#nodes-and-ssh) | — | — | — | ✓ | [ep-manual-test-scenarios.md](ep-manual-test-scenarios.md#ac-032) |
| [AC-033](ep-acceptance-criteria.md#ac-033) | [REQ-024](ep-requirements.md#nodes-and-ssh), [REQ-003](ep-requirements.md#nodes-and-ssh) | ✓ | — | — | — | internal/config/config_test.go, internal/telegram/adapter_test.go, internal/embedding, internal/llm/openai_test.go |
| [AC-034](ep-acceptance-criteria.md#ac-034) | [REQ-009](ep-requirements.md#scheduler-and-tools) | ✓ | — | — | — | internal/scheduler/loader_test.go |
| [AC-035](ep-acceptance-criteria.md#ac-035) | [REQ-010](ep-requirements.md#scheduler-and-tools) | ✓ | — | — | — | internal/tools/run_on_node_test.go |
| [AC-036](ep-acceptance-criteria.md#ac-036) | [REQ-025](ep-requirements.md#llm-and-logging) | ✓ | — | — | — | internal/llm/openai_test.go, internal/core/handler_test.go |
| [AC-037](ep-acceptance-criteria.md#ac-037) | [REQ-025](ep-requirements.md#llm-and-logging) | ✓ | — | — | — | internal/embedding/openai_test.go, internal/core/handler_test.go |
| [AC-038](ep-acceptance-criteria.md#ac-038) | [REQ-026](ep-requirements.md#secret-protection-prompt-injection--exfiltration), [REQ-027](ep-requirements.md#secret-protection-prompt-injection--exfiltration) | ✓ | — | — | — | internal/logredact/logredact_test.go, internal/core/handler_test.go |
| [AC-039](ep-acceptance-criteria.md#ac-039) | [REQ-027](ep-requirements.md#secret-protection-prompt-injection--exfiltration) | ✓ | — | — | — | internal/logredact/logredact_test.go |
| [AC-040](ep-acceptance-criteria.md#ac-040) | [REQ-028](ep-requirements.md#secret-protection-prompt-injection--exfiltration) | ✓ | — | — | — | internal/logredact/logredact_test.go |
| [AC-041](ep-acceptance-criteria.md#ac-041) | [REQ-029](ep-requirements.md#secret-protection-prompt-injection--exfiltration) | ✓ | — | — | — | internal/logredact/logredact_test.go, internal/config/config_test.go |
| [AC-042](ep-acceptance-criteria.md#ac-042) | [REQ-030](ep-requirements.md#configuration-paths-and-environment) | ✓ | — | — | — | cmd/pa/main_test.go, internal/config/resolve_test.go |
| [AC-043](ep-acceptance-criteria.md#ac-043) | [REQ-031](ep-requirements.md#llm-and-logging) | ✓ | — | — | — | internal/llm/fallback_test.go |
| [AC-044](ep-acceptance-criteria.md#ac-044) | [REQ-031](ep-requirements.md#llm-and-logging), [REQ-014](ep-requirements.md#llm-and-logging) | ✓ | — | — | — | internal/core/handler_test.go |

**Notes:** Unit = `*_test.go` in packages; Integration = `tests/integration/*_test.go` (build tag `integration`); E2E = none in repo; Manual = scenarios in [ep-manual-test-scenarios.md](ep-manual-test-scenarios.md). Run `make check` for all automated checks. Integration tests require Docker; the two-user SSH test uses a Debian image (see project README § Development).

---

## Quality gate

**Result: PASS.**  
`make check` runs: `go fmt ./...`, `go vet ./...`, `golangci-lint run --build-tags=integration ./...`, `go test -tags=integration -count=1 ./... -coverpkg=./... -coverprofile=coverage.out -covermode=atomic`, and module-boundary check. All passed; **0 issues** from the linter; module boundaries OK (no cycles, no forbidden edges).

---

## Gaps, risks, recommendations

**Gaps**

- **AC-004, AC-032:** Covered only by manual scenarios (DS220+ run and `-verify-nodes` with real SSH). No automated E2E for full container + Telegram + node verify.
- **AC-025:** Module boundaries are enforced by script; manual review is mentioned in strategy for "clear separation" narrative.

**Risks**

- **Two-user SSH image:** Uses Debian (bookworm-slim); Alpine was reverted due to auth failures on some Docker hosts (e.g. bind mount). If you standardise on Alpine later, two-user test may need a different setup or skip on incompatible hosts.
- **Integration tests depend on Docker:** SSH tests need Docker and free ports (2222, 2224). Cleanup of conflicting containers is in place; flakiness on busy or restricted CI possible.
- **Coverage:** 76.1% total; some packages (e.g. telegram Run, main) have low coverage; acceptable for MVP but leaves behaviour less guarded by tests.

**Recommendations**

1. Before release, run manual scenarios for [AC-004](ep-manual-test-scenarios.md#ac-004) and [AC-032](ep-manual-test-scenarios.md#ac-032) on target (or equivalent) environment.
2. Integration tests and the two-user SSH image (Docker, Debian) are documented in the project README (§ Development) and in `tests/integration/testdata/ssh/README.md`.
3. Optionally add a minimal E2E (e.g. container start + one Telegram message) in CI to protect AC-003/AC-004 on a single platform.
