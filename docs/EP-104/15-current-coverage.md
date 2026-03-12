# EP-104: Current test coverage

**Purpose:** Reflect tests that exist in the codebase per AC; companion to test strategy (target coverage).  
**Pipeline:** [PIPELINE.SPEC.md](PIPELINE.SPEC.md)  
**Previous:** [06-test-strategy.md](06-test-strategy.md)  
**Next:** 16. Deployment (see [PIPELINE.SPEC.md](PIPELINE.SPEC.md)).  
**Related:** [10-acceptance-criteria.md](10-acceptance-criteria.md)

This table reflects **tests that exist in the codebase** at the time of the last update. The [test strategy](06-test-strategy.md) defines the *target* coverage per AC; this document shows which AC already have at least one corresponding test. **Update this file when adding or removing tests.**

| AC | Unit | Integration | E2E | Manual | Notes |
|----|------|-------------|-----|--------|-------|
| [AC-001](10-acceptance-criteria.md#ac-001-us-01) | — | ✓ | — | — | `tests/integration/telegram_flow_test.go` |
| [AC-002](10-acceptance-criteria.md#ac-002-us-01) | ✓ | ✓ | — | — | Unit: `internal/core/handler_test.go`; integration: `tests/integration/telegram_flow_test.go` (empty, over max length) |
| [AC-005](10-acceptance-criteria.md#ac-005-us-03) | ✓ | — | — | — | `internal/config/config_test.go` |
| [AC-006](10-acceptance-criteria.md#ac-006-us-03) | ✓ | ✓ | — | — | Unit: `internal/ssh/client_test.go` (config-only); integration: `tests/integration/ssh_node_test.go` (uses config node) |
| [AC-007](10-acceptance-criteria.md#ac-007-us-04) | ✓ | ✓ | — | — | Unit: `internal/allowlist/allowlist_test.go`; integration: `tests/integration/ssh_node_test.go` (allowlist blocks disallowed) |
| [AC-008](10-acceptance-criteria.md#ac-008-us-04) | ✓ | ✓ | — | — | Unit: `internal/allowlist/allowlist_test.go`; integration: `tests/integration/ssh_node_test.go` (allowlist blocks disallowed) |
| [AC-009](10-acceptance-criteria.md#ac-009-us-05) | ✓ | ✓ | — | — | Unit: `internal/ssh/ssh_test.go`, `internal/ssh/client_test.go`; integration: `tests/integration/ssh_node_test.go` |
| [AC-010](10-acceptance-criteria.md#ac-010-us-05) | ✓ | ✓ | — | — | Unit: `internal/ssh/ssh_test.go` (multi-node user); integration: `tests/integration/ssh_node_test.go` |
| [AC-011](10-acceptance-criteria.md#ac-011-us-06) | ✓ | ✓ | — | — | Unit: `internal/memory/store_test.go`; integration: `tests/integration/memory_vector_test.go` (injects today memory) |
| [AC-012](10-acceptance-criteria.md#ac-012-us-06) | ✓ | ✓ | — | — | Unit: `internal/memory/store_test.go`; integration: `tests/integration/memory_vector_test.go` (injects today memory) |
| [AC-013](10-acceptance-criteria.md#ac-013-us-07) | ✓ | ✓ | — | ✓ | Unit: `internal/vector/sqlite/store_test.go`; integration: `tests/integration/memory_vector_test.go` (injects past context); manual: [06-manual-test-plan.md](06-manual-test-plan.md) |
| [AC-014](10-acceptance-criteria.md#ac-014-us-07) | ✓ | ✓ | — | ✓ | Unit: `internal/vector/sqlite/store_test.go`; integration: `tests/integration/memory_vector_test.go` (injects past context); manual: [06-manual-test-plan.md](06-manual-test-plan.md) |
| [AC-015](10-acceptance-criteria.md#ac-015-us-08) | ✓ | — | — | — | `internal/llm/provider_test.go` |
| [AC-016](10-acceptance-criteria.md#ac-016-us-08) | — | ✓ | — | — | `tests/integration/telegram_flow_test.go` (different provider per run) |
| [AC-020](10-acceptance-criteria.md#ac-020-us-11) | ✓ | ✓ | — | — | Unit: `internal/telegram/adapter_test.go` (notify_chat_id [REQ-023]: from config, fallback to first user, zero when none; SendMessage when bot nil); integration: `tests/integration/scheduler_config_test.go` (scheduler fires and runs tool) |
| [AC-021](10-acceptance-criteria.md#ac-021-us-11) | — | ✓ | — | — | `tests/integration/scheduler_config_test.go` (task with disallowed command not executed) |
| [AC-022](10-acceptance-criteria.md#ac-022-us-12) | ✓ | — | — | — | Unit: `internal/tools/run_on_node_test.go` (valid params, runner invoked), `registry_test.go` (Register/Get/contract) |
| [AC-023](10-acceptance-criteria.md#ac-023-us-12) | ✓ | — | — | — | Unit: `internal/tools/run_on_node_test.go` (invalid params rejected), `registry_test.go` (ValidateParams) |
| [AC-024](10-acceptance-criteria.md#ac-024-us-13) | — | ✓ | — | — | `tests/integration/scheduler_config_test.go` (load different task file) |
| [AC-025](10-acceptance-criteria.md#ac-025-us-14) | — | — | — | ✓ | [06-manual-test-plan.md](06-manual-test-plan.md) (architecture review) |
| [AC-027](10-acceptance-criteria.md#ac-027-us-15) | — | — | — | ✓ | [06-manual-test-plan.md](06-manual-test-plan.md) (docs: tracked paths) |
| [AC-031](10-acceptance-criteria.md#ac-031-us-17) | ✓ | — | — | — | `internal/core/handler_test.go` (DEBUG vs INFO logging) |
| [AC-032](10-acceptance-criteria.md#ac-032-us-18) | — | — | — | ✓ | Manual only; scenario in [06-manual-test-plan.md](06-manual-test-plan.md). |
| [AC-017](10-acceptance-criteria.md#ac-017-us-09), [AC-018](10-acceptance-criteria.md#ac-018-us-10), [AC-019](10-acceptance-criteria.md#ac-019-us-10), [AC-026](10-acceptance-criteria.md#ac-026-us-15), [AC-028](10-acceptance-criteria.md#ac-028-us-16)–[AC-030](10-acceptance-criteria.md#ac-030-us-16) | — | — | — | — | No tests yet; feature or task not implemented (see [11-12-implementation-plan.md](11-12-implementation-plan.md)). |
