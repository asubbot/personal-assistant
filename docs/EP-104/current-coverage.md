# EP-104: Current test coverage

**Epic:** EP-104  
**Related:** [test-strategy.md](test-strategy.md) (target strategy, §3); [acceptance-criteria.md](acceptance-criteria.md)

This table reflects **tests that exist in the codebase** at the time of the last update. The [test strategy](test-strategy.md) defines the *target* coverage per AC; this document shows which AC already have at least one corresponding test. **Update this file when adding or removing tests.**

| AC     | Unit | Integration | E2E | Manual | Notes |
|--------|------|-------------|-----|--------|-------|
| AC-001 | —    | ✓           | —   | —      | `tests/integration/telegram_flow_test.go` |
| AC-002 | ✓    | ✓           | —   | —      | Unit: `internal/core/handler_test.go`; integration: `tests/integration/telegram_flow_test.go` (empty, over max length) |
| AC-005 | ✓    | —           | —   | —      | `internal/config/config_test.go` |
| AC-006 | ✓    | ✓           | —   | —      | Unit: `internal/ssh/client_test.go` (config-only); integration: `tests/integration/ssh_node_test.go` (uses config node) |
| AC-007 | ✓    | ✓           | —   | —      | Unit: `internal/allowlist/allowlist_test.go`; integration: `tests/integration/ssh_node_test.go` (allowlist blocks disallowed) |
| AC-008 | ✓    | ✓           | —   | —      | Unit: `internal/allowlist/allowlist_test.go`; integration: `tests/integration/ssh_node_test.go` (allowlist blocks disallowed) |
| AC-009 | ✓    | ✓           | —   | —      | Unit: `internal/ssh/ssh_test.go`, `internal/ssh/client_test.go`; integration: `tests/integration/ssh_node_test.go` |
| AC-010 | ✓    | ✓           | —   | —      | Unit: `internal/ssh/ssh_test.go` (multi-node user); integration: `tests/integration/ssh_node_test.go` |
| AC-015 | ✓    | —           | —   | —      | `internal/llm/provider_test.go` |
| AC-016 | —    | ✓           | —   | —      | `tests/integration/telegram_flow_test.go` (different provider per run) |
| AC-031 | ✓    | —           | —   | —      | `internal/core/handler_test.go` (DEBUG vs INFO logging) |
| AC-011 | ✓    | ✓           | —   | —      | Unit: `internal/memory/store_test.go`; integration: `tests/integration/memory_vector_test.go` (injects today memory) |
| AC-012 | ✓    | ✓           | —   | —      | Unit: `internal/memory/store_test.go`; integration: `tests/integration/memory_vector_test.go` (injects today memory) |
| AC-013 | ✓    | ✓           | —   | ✓      | Unit: `internal/vector/sqlite/store_test.go`; integration: `tests/integration/memory_vector_test.go` (injects past context); manual: [manual-test-plan.md](manual-test-plan.md) |
| AC-014 | ✓    | ✓           | —   | ✓      | Unit: `internal/vector/sqlite/store_test.go`; integration: `tests/integration/memory_vector_test.go` (injects past context); manual: [manual-test-plan.md](manual-test-plan.md) |
| AC-025 | —    | —           | —   | ✓      | [manual-test-plan.md](manual-test-plan.md) (architecture review) |
| AC-027 | —    | —           | —   | ✓      | [manual-test-plan.md](manual-test-plan.md) (docs: tracked paths) |
| AC-020 | ✓    | ✓           | —   | —      | Unit: `internal/telegram/adapter_test.go` (notify_chat_id [REQ-023]: from config, fallback to first user, zero when none; SendMessage when bot nil); integration: `tests/integration/scheduler_config_test.go` (scheduler fires and runs tool) |
| AC-021 | —    | ✓           | —   | —      | `tests/integration/scheduler_config_test.go` (task with disallowed command not executed) |
| AC-022 | ✓    | —           | —   | —      | Unit: `internal/tools/run_on_node_test.go` (valid params, runner invoked), `registry_test.go` (Register/Get/contract) |
| AC-023 | ✓    | —           | —   | —      | Unit: `internal/tools/run_on_node_test.go` (invalid params rejected), `registry_test.go` (ValidateParams) |
| AC-024 | —    | ✓           | —   | —      | `tests/integration/scheduler_config_test.go` (load different task file) |
| AC-017, AC-018, AC-019, AC-026, AC-028–AC-030 | — | — | — | — | No tests yet; feature or task not implemented (see [implementation-plan.md](implementation-plan.md)). |
| AC-032 | — | —           | —   | ✓      | Manual only; scenario in [manual-test-plan.md](manual-test-plan.md). |
