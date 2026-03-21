# EP-001 PersonalAssistant MVP — Audit Report

**Date and time:** 2026-03-16 (UTC)  
**Purpose:** Stage 9 audit — implementation vs plan, tests, coverage, quality gate, gaps/risks.  
**Pipeline:** [ai-sdlc/specification/pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md)  
**Epic artefacts:** [ep-implementation-plan.md](ep-implementation-plan.md), [ep-acceptance-criteria.md](ep-acceptance-criteria.md), [ep-requirements.md](ep-requirements.md), [ep-manual-test-scenarios.md](ep-manual-test-scenarios.md)

---

## Summary

**Status: PASS.** All in-scope implementation plan tasks are done except the intentionally deferred task 10.2 (versioned state). `make check` passes (fmt, vet, golangci-lint, tests with integration tag, module boundaries). Total statement coverage **76.1%**. No open quality gate issues. Remaining gaps: part of AC covered only manually (AC-01.004, AC-01.032); no dedicated E2E; two-user SSH image uses Debian (Alpine was unstable on some hosts).

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
| 10.2 | **Deferred** | Versioned state (REQ-01.016, AC-01.026, AC-01.027) — out of MVP scope |
| 11.1 | Done | LLM provider fallback (REQ-01.031) |
| 12.1 | Done | Secret leakage protection (unit + integration) |
| 13.1 | Done | Final checkpoint |

---

## Test results and coverage

- **Command run:** `make check` (go fmt, go vet, golangci-lint with `--build-tags=integration`, go test with coverage and module-boundary check).
- **Result:** **PASS** (exit code 0).
- **Total coverage:** **76.1%** of statements (`total: (statements) 76.1%` from `go tool cover -func=coverage.out`).
- **Per-package (selected):** cmd/pa 20.6%, internal/config 10.6%, internal/core 6.6%, internal/allowlist 2.9%, internal/llm 7.1%, internal/scheduler 6.1%, internal/ssh 1.7%, internal/summarize 13.1%, internal/telegram 4.0%, internal/tools 2.8%, internal/vector/sqlite 3.5%, tests/integration 22.2%. `internal/logging` and `internal/vector` have no test files (excluded from coverage).

Integration tests (tag `integration`) run with `make check` and include: Telegram flow (AC-01.001, AC-01.002, AC-01.016), SSH single-user and two-user (AC-01.006, AC-01.010), node runner allowlist (AC-01.007, AC-01.008), memory/vector (AC-01.013, AC-01.014), scheduler (AC-01.020, AC-01.021, AC-01.024), secret leakage (AC-01.028, AC-01.029, AC-01.030).

---

## REQ/AC test coverage matrix

| AC | REQ | Unit | Integration | E2E | Manual | Link |
|----|-----|------|-------------|-----|--------|------|
| [AC-01.001](ep-acceptance-criteria.md#ac-01-001) | [REQ-01.001](ep-requirements.md#interface-and-deployment) | ✓ | ✓ | — | — | internal/core/handler_test.go, internal/telegram/adapter_test.go; tests/integration/telegram_flow_test.go |
| [AC-01.002](ep-acceptance-criteria.md#ac-01-002) | [REQ-01.001](ep-requirements.md#interface-and-deployment) | ✓ | ✓ | — | — | internal/core/handler_test.go, internal/telegram/adapter_test.go; tests/integration/telegram_flow_test.go |
| [AC-01.003](ep-acceptance-criteria.md#ac-01-003) | [REQ-01.001](ep-requirements.md#interface-and-deployment), [REQ-01.002](ep-requirements.md#interface-and-deployment) | ✓ | — | — | — | internal/core/run_test.go, internal/telegram/adapter_test.go |
| [AC-01.004](ep-acceptance-criteria.md#ac-01-004) | [REQ-01.002](ep-requirements.md#interface-and-deployment) | — | — | — | ✓ | [ep-manual-test-scenarios.md](ep-manual-test-scenarios.md#ac-01-004) |
| [AC-01.005](ep-acceptance-criteria.md#ac-01-005) | [REQ-01.003](ep-requirements.md#nodes-and-ssh), [REQ-01.024](ep-requirements.md#nodes-and-ssh) | ✓ | — | — | — | internal/config/config_test.go |
| [AC-01.006](ep-acceptance-criteria.md#ac-01-006) | [REQ-01.004](ep-requirements.md#nodes-and-ssh) | ✓ | ✓ | — | — | internal/ssh/client_test.go; tests/integration/ssh_client_test.go, ssh_node_test.go |
| [AC-01.007](ep-acceptance-criteria.md#ac-01-007) | [REQ-01.005](ep-requirements.md#nodes-and-ssh) | ✓ | ✓ | — | — | internal/allowlist/allowlist_test.go, internal/noderunner/runner_test.go; tests/integration/ssh_node_test.go |
| [AC-01.008](ep-acceptance-criteria.md#ac-01-008) | [REQ-01.005](ep-requirements.md#nodes-and-ssh) | ✓ | ✓ | — | — | internal/allowlist/allowlist_test.go, internal/noderunner/runner_test.go, internal/tools/run_on_node_test.go; tests/integration/ssh_node_test.go |
| [AC-01.009](ep-acceptance-criteria.md#ac-01-009) | [REQ-01.013](ep-requirements.md#nodes-and-ssh) | ✓ | ✓ | — | — | internal/ssh/client_test.go; tests/integration/ssh_node_test.go |
| [AC-01.010](ep-acceptance-criteria.md#ac-01-010) | [REQ-01.013](ep-requirements.md#nodes-and-ssh) | ✓ | ✓ | — | — | internal/noderunner/runner_test.go, internal/ssh/ssh_test.go; tests/integration/ssh_client_test.go |
| [AC-01.011](ep-acceptance-criteria.md#ac-01-011) | [REQ-01.006](ep-requirements.md#memory-and-indexing), [REQ-01.019](ep-requirements.md#memory-and-indexing) | ✓ | — | — | — | internal/summarize/summarize_test.go, cmd/pa/main_test.go |
| [AC-01.012](ep-acceptance-criteria.md#ac-01-012) | [REQ-01.006](ep-requirements.md#memory-and-indexing) | ✓ | — | — | — | internal/summarize/summarize_test.go, internal/memory (store) |
| [AC-01.013](ep-acceptance-criteria.md#ac-01-013) | [REQ-01.007](ep-requirements.md#memory-and-indexing) | ✓ | ✓ | — | — | internal/vector/sqlite/store_test.go, internal/core/handler_test.go; tests/integration/memory_vector_test.go |
| [AC-01.014](ep-acceptance-criteria.md#ac-01-014) | [REQ-01.007](ep-requirements.md#memory-and-indexing) | ✓ | ✓ | — | — | internal/vector/sqlite/store_test.go, internal/core/handler_test.go; tests/integration/memory_vector_test.go |
| [AC-01.015](ep-acceptance-criteria.md#ac-01-015) | [REQ-01.008](ep-requirements.md#llm-and-logging) | ✓ | — | — | — | internal/llm/openai_test.go, internal/llm/provider_test.go |
| [AC-01.016](ep-acceptance-criteria.md#ac-01-016) | [REQ-01.008](ep-requirements.md#llm-and-logging) | — | ✓ | — | — | tests/integration/telegram_flow_test.go |
| [AC-01.017](ep-acceptance-criteria.md#ac-01-017) | [REQ-01.014](ep-requirements.md#llm-and-logging) | ✓ | — | — | — | internal/llmlog/llmlog_test.go |
| [AC-01.018](ep-acceptance-criteria.md#ac-01-018) | [REQ-01.015](ep-requirements.md#llm-and-logging) | ✓ | — | — | — | internal/llmlog/llmlog_test.go, internal/core/handler_test.go |
| [AC-01.019](ep-acceptance-criteria.md#ac-01-019) | [REQ-01.015](ep-requirements.md#llm-and-logging) | ✓ | — | — | — | internal/llmlog/llmlog_test.go |
| [AC-01.020](ep-acceptance-criteria.md#ac-01-020) | [REQ-01.009](ep-requirements.md#scheduler-and-tools), [REQ-01.023](ep-requirements.md#scheduler-and-tools) | ✓ | ✓ | — | — | internal/telegram/adapter_test.go; tests/integration/scheduler_config_test.go |
| [AC-01.021](ep-acceptance-criteria.md#ac-01-021) | [REQ-01.009](ep-requirements.md#scheduler-and-tools) | ✓ | ✓ | — | — | internal/scheduler/scheduler_test.go; tests/integration/scheduler_config_test.go |
| [AC-01.022](ep-acceptance-criteria.md#ac-01-022) | [REQ-01.010](ep-requirements.md#scheduler-and-tools) | ✓ | — | — | — | internal/tools/registry_test.go, internal/tools/run_on_node_test.go |
| [AC-01.023](ep-acceptance-criteria.md#ac-01-023) | [REQ-01.010](ep-requirements.md#scheduler-and-tools) | ✓ | — | — | — | internal/tools/registry_test.go, internal/tools/run_on_node_test.go |
| [AC-01.024](ep-acceptance-criteria.md#ac-01-024) | [REQ-01.011](ep-requirements.md#extensibility-and-architecture) | ✓ | ✓ | — | — | internal/scheduler/loader_test.go; tests/integration/scheduler_config_test.go |
| [AC-01.025](ep-acceptance-criteria.md#ac-01-025) | [REQ-01.012](ep-requirements.md#extensibility-and-architecture) | — | — | — | ✓ | scripts/check-module-boundaries.sh (automated); strategy §2.3 manual |
| [AC-01.026](ep-acceptance-criteria.md#ac-01-026) | [REQ-01.016](ep-requirements.md#version-control-and-audit) | — | — | — | — | **Deferred** (post-MVP) |
| [AC-01.027](ep-acceptance-criteria.md#ac-01-027) | [REQ-01.016](ep-requirements.md#version-control-and-audit) | — | — | — | — | **Deferred** (post-MVP) |
| [AC-01.028](ep-acceptance-criteria.md#ac-01-028) | [REQ-01.017](ep-requirements.md#secret-protection-prompt-injection--exfiltration) | ✓ | ✓ | — | — | internal/core/run_test.go, handler_test.go; tests/integration/secret_leakage_test.go |
| [AC-01.029](ep-acceptance-criteria.md#ac-01-029) | [REQ-01.017](ep-requirements.md#secret-protection-prompt-injection--exfiltration) | — | ✓ | — | — | tests/integration/secret_leakage_test.go |
| [AC-01.030](ep-acceptance-criteria.md#ac-01-030) | [REQ-01.017](ep-requirements.md#secret-protection-prompt-injection--exfiltration) | — | ✓ | — | — | internal/llmlog/llmlog_test.go (redactor); tests/integration/secret_leakage_test.go |
| [AC-01.031](ep-acceptance-criteria.md#ac-01-031) | [REQ-01.021](ep-requirements.md#llm-and-logging) | ✓ | — | — | — | internal/core/handler_test.go |
| [AC-01.032](ep-acceptance-criteria.md#ac-01-032) | [REQ-01.022](ep-requirements.md#nodes-and-ssh) | — | — | — | ✓ | [ep-manual-test-scenarios.md](ep-manual-test-scenarios.md#ac-01-032) |
| [AC-01.033](ep-acceptance-criteria.md#ac-01-033) | [REQ-01.024](ep-requirements.md#nodes-and-ssh), [REQ-01.003](ep-requirements.md#nodes-and-ssh) | ✓ | — | — | — | internal/config/config_test.go, internal/telegram/adapter_test.go, internal/embedding, internal/llm/openai_test.go |
| [AC-01.034](ep-acceptance-criteria.md#ac-01-034) | [REQ-01.009](ep-requirements.md#scheduler-and-tools) | ✓ | — | — | — | internal/scheduler/loader_test.go |
| [AC-01.035](ep-acceptance-criteria.md#ac-01-035) | [REQ-01.010](ep-requirements.md#scheduler-and-tools) | ✓ | — | — | — | internal/tools/run_on_node_test.go |
| [AC-01.036](ep-acceptance-criteria.md#ac-01-036) | [REQ-01.025](ep-requirements.md#llm-and-logging) | ✓ | — | — | — | internal/llm/openai_test.go, internal/core/handler_test.go |
| [AC-01.037](ep-acceptance-criteria.md#ac-01-037) | [REQ-01.025](ep-requirements.md#llm-and-logging) | ✓ | — | — | — | internal/embedding/openai_test.go, internal/core/handler_test.go |
| [AC-01.038](ep-acceptance-criteria.md#ac-01-038) | [REQ-01.026](ep-requirements.md#secret-protection-prompt-injection--exfiltration), [REQ-01.027](ep-requirements.md#secret-protection-prompt-injection--exfiltration) | ✓ | — | — | — | internal/logredact/logredact_test.go, internal/core/handler_test.go |
| [AC-01.039](ep-acceptance-criteria.md#ac-01-039) | [REQ-01.027](ep-requirements.md#secret-protection-prompt-injection--exfiltration) | ✓ | — | — | — | internal/logredact/logredact_test.go |
| [AC-01.040](ep-acceptance-criteria.md#ac-01-040) | [REQ-01.028](ep-requirements.md#secret-protection-prompt-injection--exfiltration) | ✓ | — | — | — | internal/logredact/logredact_test.go |
| [AC-01.041](ep-acceptance-criteria.md#ac-01-041) | [REQ-01.029](ep-requirements.md#secret-protection-prompt-injection--exfiltration) | ✓ | — | — | — | internal/logredact/logredact_test.go, internal/config/config_test.go |
| [AC-01.042](ep-acceptance-criteria.md#ac-01-042) | [REQ-01.030](ep-requirements.md#configuration-paths-and-environment) | ✓ | — | — | — | cmd/pa/main_test.go, internal/config/resolve_test.go |
| **[AC-01.043](ep-acceptance-criteria.md#ac-01-043)** | **[REQ-01.031](ep-requirements.md#llm-and-logging)** | ✓ | — | — | — | internal/llm/openai_test.go; **LEGACY after EP-006 (transport/network fallback baseline only)** |
| **[AC-01.044](ep-acceptance-criteria.md#ac-01-044)** | **[REQ-01.031](ep-requirements.md#llm-and-logging)**, [REQ-01.014](ep-requirements.md#llm-and-logging) | ✓ | — | — | — | internal/core/handler_test.go; **LEGACY after EP-006 (not full tool-path escalation coverage)** |

**Notes:** Unit = `*_test.go` in packages; Integration = `tests/integration/*_test.go` (build tag `integration`); E2E = none in repo; Manual = scenarios in [ep-manual-test-scenarios.md](ep-manual-test-scenarios.md). Run `make check` for all automated checks. Integration tests require Docker; the two-user SSH test uses a Debian image (see project README § Development).

---

## Quality gate

**Result: PASS.**  
`make check` runs: `go fmt ./...`, `go vet ./...`, `golangci-lint run --build-tags=integration ./...`, `go test -tags=integration -count=1 ./... -coverpkg=./... -coverprofile=coverage.out -covermode=atomic`, and module-boundary check. All passed; **0 issues** from the linter; module boundaries OK (no cycles, no forbidden edges).

---

## Gaps, risks, recommendations

## EP-001 items impacted by EP-006

The following EP-001 traceability items are now **partially outdated** after EP-006 (tool-path reliability and model escalation):

- **REQ-01.031** fallback semantics are now **scope-limited**: they remain valid for transport/network fallback paths, but conversation tool-path recovery is governed by EP-006 policy (`tools.llm_escalation`, typed tool failures, bounded per-message escalation).
- **AC-01.043** and **AC-01.044** should be treated as **legacy baseline checks** for fallback transport behavior, not as full coverage of current conversation recovery behavior.
- **Tests cited for AC-01.043/AC-01.044** in this report (`internal/llm/fallback_test.go`, `internal/core/handler_test.go`) are therefore **not sufficient alone** to represent current post-EP-006 behavior; EP-006 tests (`internal/core/handler_ep006_audit_test.go`, `internal/escalationpolicy/catalog_test.go`) are now primary for escalation scenarios.

Recommendation: keep REQ/AC IDs in EP-001 unchanged for historical MVP traceability, but interpret them as baseline/legacy after EP-006 and rely on EP-006 audit matrix for escalation-related acceptance.

---

**Gaps**

- **AC-01.004, AC-01.032:** Covered only by manual scenarios (DS220+ run and `-verify-nodes` with real SSH). No automated E2E for full container + Telegram + node verify.
- **AC-01.025:** Module boundaries are enforced by script; manual review is mentioned in strategy for "clear separation" narrative.

**Risks**

- **Two-user SSH image:** Uses Debian (bookworm-slim); Alpine was reverted due to auth failures on some Docker hosts (e.g. bind mount). If you standardise on Alpine later, two-user test may need a different setup or skip on incompatible hosts.
- **Integration tests depend on Docker:** SSH tests need Docker and free ports (2222, 2224). Cleanup of conflicting containers is in place; flakiness on busy or restricted CI possible.
- **Coverage:** 76.1% total; some packages (e.g. telegram Run, main) have low coverage; acceptable for MVP but leaves behaviour less guarded by tests.

**Recommendations**

1. Before release, run manual scenarios for [AC-01.004](ep-manual-test-scenarios.md#ac-01-004) and [AC-01.032](ep-manual-test-scenarios.md#ac-01-032) on target (or equivalent) environment.
2. Integration tests and the two-user SSH image (Docker, Debian) are documented in the project README (§ Development) and in `tests/integration/testdata/ssh/README.md`.
3. Optionally add a minimal E2E (e.g. container start + one Telegram message) in CI to protect AC-01.003/AC-01.004 on a single platform.
