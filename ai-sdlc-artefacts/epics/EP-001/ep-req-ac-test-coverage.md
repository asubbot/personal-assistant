# EP-001 REQ/AC traceability and test coverage

**Purpose:** Trace requirements (REQ) to acceptance criteria (AC) and map each AC to test levels (unit, integration, E2E, manual) with links to test code or verification steps.

**Sources:** [ep-requirements.md](ep-requirements.md), [ep-acceptance-criteria.md](ep-acceptance-criteria.md), [strategy.md](../../strategy.md) §2 (test strategy), codebase `Covers AC-xxx` comments. **Manual test scenarios:** [ep-manual-test-scenarios.md](ep-manual-test-scenarios.md).

---

## Test coverage matrix

| AC | REQ | Unit | Integration | E2E | Manual | Link |
|----|-----|------|-------------|-----|--------|------|
| [AC-001](ep-acceptance-criteria.md#ac-001) | REQ-001 | ✓ | ✓ | — | — | Unit: `internal/core/handler_test.go`, `internal/telegram/adapter_test.go`. Integration: `tests/integration/telegram_flow_test.go`. |
| [AC-002](ep-acceptance-criteria.md#ac-002) | REQ-001 | ✓ | ✓ | — | — | Unit: `internal/core/handler_test.go`, `internal/telegram/adapter_test.go`. Integration: `tests/integration/telegram_flow_test.go`. |
| [AC-003](ep-acceptance-criteria.md#ac-003) | REQ-001, REQ-002 | ✓ | — | — | — | Unit: `internal/core/run_test.go`, `internal/telegram/adapter_test.go`. |
| [AC-004](ep-acceptance-criteria.md#ac-004) | REQ-002 | — | — | — | ✓ | [ep-manual-test-scenarios.md#ac-004](ep-manual-test-scenarios.md#ac-004). |
| [AC-005](ep-acceptance-criteria.md#ac-005) | REQ-003, REQ-024 | ✓ | — | — | — | Unit: `internal/config/config_test.go`, `internal/llm/provider_test.go`, etc. |
| [AC-006](ep-acceptance-criteria.md#ac-006) | REQ-004 | ✓ | ✓ | — | — | Unit: `internal/ssh/client_test.go`. Integration: `tests/integration/ssh_node_test.go`. |
| [AC-007](ep-acceptance-criteria.md#ac-007) | REQ-005 | ✓ | ✓ | — | — | Unit: `internal/allowlist/allowlist_test.go`, `internal/noderunner/runner_test.go`. Integration: `tests/integration/ssh_node_test.go`. |
| [AC-008](ep-acceptance-criteria.md#ac-008) | REQ-005 | ✓ | ✓ | — | — | Unit: `internal/allowlist/allowlist_test.go`, `internal/noderunner/runner_test.go`, `internal/tools/run_on_node_test.go`. Integration: `tests/integration/ssh_node_test.go`. |
| [AC-009](ep-acceptance-criteria.md#ac-009) | REQ-013 | ✓ | ✓ | — | — | Unit: `internal/ssh/ssh_test.go`. Integration: `tests/integration/ssh_node_test.go`. |
| [AC-010](ep-acceptance-criteria.md#ac-010) | REQ-013 | ✓ | — | — | — | Unit: `internal/ssh/ssh_test.go`. |
| [AC-011](ep-acceptance-criteria.md#ac-011) | REQ-006, REQ-019 | ✓ | — | — | — | Unit: `internal/memory/store_test.go`, `internal/summarize/summarize_test.go`, `cmd/pa/main_test.go`, `internal/llmlog/llmlog_test.go`. |
| [AC-012](ep-acceptance-criteria.md#ac-012) | REQ-006 | ✓ | — | — | — | Unit: `internal/memory/store_test.go`, `internal/summarize/summarize_test.go`, `cmd/pa/main_test.go`. |
| [AC-013](ep-acceptance-criteria.md#ac-013) | REQ-007 | ✓ | ✓ | — | — | Unit: `internal/vector/sqlite/store_test.go`. Integration: `tests/integration/memory_vector_test.go`. |
| [AC-014](ep-acceptance-criteria.md#ac-014) | REQ-007 | ✓ | ✓ | — | — | Unit: `internal/vector/sqlite/store_test.go`. Integration: `tests/integration/memory_vector_test.go`. |
| [AC-015](ep-acceptance-criteria.md#ac-015) | REQ-008 | ✓ | — | — | — | Unit: `internal/llm/provider_test.go`, `internal/llm/openai_test.go`. |
| [AC-016](ep-acceptance-criteria.md#ac-016) | REQ-008 | — | ✓ | — | — | Integration: `tests/integration/telegram_flow_test.go` (different provider per run). |
| [AC-017](ep-acceptance-criteria.md#ac-017) | REQ-014 | ✓ | — | — | — | Unit: `internal/llmlog/llmlog_test.go`. |
| [AC-018](ep-acceptance-criteria.md#ac-018) | REQ-015 | ✓ | — | — | — | Unit: `internal/llmlog/llmlog_test.go`. |
| [AC-019](ep-acceptance-criteria.md#ac-019) | REQ-015 | ✓ | — | — | — | Unit: `internal/llmlog/llmlog_test.go` (unwritable/read-only dir). |
| [AC-020](ep-acceptance-criteria.md#ac-020) | REQ-009, REQ-023 | ✓ | ✓ | — | — | Unit: `internal/telegram/adapter_test.go`, `internal/scheduler/scheduler_test.go`. Integration: `tests/integration/scheduler_config_test.go`. |
| [AC-021](ep-acceptance-criteria.md#ac-021) | REQ-009 | ✓ | ✓ | — | — | Unit: `internal/scheduler/scheduler_test.go`. Integration: `tests/integration/scheduler_config_test.go`. |
| [AC-022](ep-acceptance-criteria.md#ac-022) | REQ-010 | ✓ | — | — | — | Unit: `internal/tools/registry_test.go`, `internal/tools/run_on_node_test.go`. |
| [AC-023](ep-acceptance-criteria.md#ac-023) | REQ-010 | ✓ | — | — | — | Unit: `internal/tools/registry_test.go`, `internal/tools/run_on_node_test.go`. |
| [AC-024](ep-acceptance-criteria.md#ac-024) | REQ-011 | ✓ | ✓ | — | — | Unit: `internal/scheduler/loader_test.go`. Integration: `tests/integration/scheduler_config_test.go`. |
| [AC-025](ep-acceptance-criteria.md#ac-025) | REQ-012 | — | — | — | ✓ | Automated: `make check` / `make check-boundaries` ([scripts/check-module-boundaries.sh](../../../scripts/check-module-boundaries.sh)). Design: [ep-system-design.md §2.1](ep-system-design.md#21-module-boundaries-req-012-ac-025). |
| [AC-026](ep-acceptance-criteria.md#ac-026) | REQ-016 | — | — | — | — | **Deferred (post-MVP).** Not in scope for EP-001 validation. |
| [AC-027](ep-acceptance-criteria.md#ac-027) | REQ-016 | — | — | — | — | **Deferred (post-MVP).** Not in scope for EP-001 validation. |
| [AC-028](ep-acceptance-criteria.md#ac-028) | REQ-017 | — | ✓ | — | — | Integration: `tests/integration/secret_leakage_test.go`. |
| [AC-029](ep-acceptance-criteria.md#ac-029) | REQ-017 | — | ✓ | — | — | Integration: `tests/integration/secret_leakage_test.go`. |
| [AC-030](ep-acceptance-criteria.md#ac-030) | REQ-017 | ✓ | ✓ | — | — | Unit: `internal/llmlog/llmlog_test.go` (redactor). Integration: `tests/integration/secret_leakage_test.go`. |
| [AC-031](ep-acceptance-criteria.md#ac-031) | REQ-021 | ✓ | — | — | — | Unit: `internal/core/handler_test.go`. |
| [AC-032](ep-acceptance-criteria.md#ac-032) | REQ-022 | — | — | — | ✓ | [ep-manual-test-scenarios.md#ac-032](ep-manual-test-scenarios.md#ac-032). |
| [AC-033](ep-acceptance-criteria.md#ac-033) | REQ-024, REQ-003 | ✓ | — | — | — | Unit: `internal/config/config_test.go`, `internal/telegram/adapter_test.go`, `internal/llm/provider_test.go`, `internal/embedding/provider_test.go`, `internal/llm/openai_test.go`, `internal/embedding/openai_test.go`. |
| [AC-034](ep-acceptance-criteria.md#ac-034) | REQ-009 | ✓ | — | — | — | Unit: `internal/scheduler/loader_test.go`. |
| [AC-035](ep-acceptance-criteria.md#ac-035) | REQ-010 | ✓ | — | — | — | Unit: `internal/tools/run_on_node_test.go`. |
| [AC-036](ep-acceptance-criteria.md#ac-036) | REQ-025 | ✓ | — | — | — | Unit: `internal/llm/openai_test.go`, `internal/core/handler_test.go`. |
| [AC-037](ep-acceptance-criteria.md#ac-037) | REQ-025 | ✓ | — | — | — | Unit: `internal/embedding/openai_test.go`, `internal/core/handler_test.go`. |
| [AC-038](ep-acceptance-criteria.md#ac-038) | REQ-026, REQ-027 | ✓ | — | — | — | Unit: `internal/logredact/logredact_test.go`. |
| [AC-039](ep-acceptance-criteria.md#ac-039) | REQ-027 | ✓ | — | — | — | Unit: `internal/logredact/logredact_test.go`. |
| [AC-040](ep-acceptance-criteria.md#ac-040) | REQ-028 | ✓ | — | — | — | Unit: `internal/logredact/logredact_test.go`. |
| [AC-041](ep-acceptance-criteria.md#ac-041) | REQ-029 | ✓ | — | — | — | Unit: `internal/config/config_test.go`, `internal/logredact/logredact_test.go`. |
| [AC-042](ep-acceptance-criteria.md#ac-042) | REQ-030 | ✓ | — | — | — | Unit: `internal/config/resolve_test.go`, `cmd/pa/main_test.go`. |
| [AC-043](ep-acceptance-criteria.md#ac-043) | REQ-031 | — | — | — | — | Not implemented yet. |

---

## Notes

- **Unit:** Tests in `internal/*_test.go` and `cmd/pa/main_test.go`. Included in `make test`.
- **Integration:** Tests in `tests/integration/*_test.go` (build tag `integration`). Included in `make test`; integration only: `make test-integration`.
- **E2E:** Full flow Telegram → core → LLM → reply; strategy §2.2. Covered by integration telegram flow test with mocks; real E2E with live Telegram/LLM is optional/manual.
- **Manual:** See [ep-manual-test-scenarios.md](ep-manual-test-scenarios.md). Strategy §2.3 — node/CLI verification, docs/architecture review.
- **Deferred:** AC-026 and AC-027 are out of MVP scope; no test coverage required for EP-001.

**Make commands (testing):**

| Command | Description |
|---------|-------------|
| `make test` | Run all tests (unit + integration). |
| `make test-integration` | Run only integration tests. |
| `make coverage` | Run all tests and print coverage summary. |
| `make coverage-html` | Run all tests and generate HTML coverage report (`coverage.html`). |
| `make check-boundaries` | Verify module boundaries (AC-025); no cycles, no forbidden edges. |
| `make check` | Format, vet, lint, coverage, check-boundaries (full quality gate). |
