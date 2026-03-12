# EP-104: Current test coverage

**Purpose:** Reflect tests that exist in the codebase per AC; companion to test strategy (target coverage).  
**Pipeline:** [PIPELINE.SPEC.md](PIPELINE.SPEC.md)  
**Previous:** [06-test-strategy.md](06-test-strategy.md)  
**Next:** 16. Deployment (see [PIPELINE.SPEC.md](PIPELINE.SPEC.md)).  
**Related:** [10-acceptance-criteria.md](10-acceptance-criteria.md)

This table reflects **tests that exist in the codebase** at the time of the last update. The [test strategy](06-test-strategy.md) defines the *target* coverage per AC; this document shows which AC already have at least one corresponding test. **Update this file when adding or removing tests.**

Table rules: rows are ordered by **AC number ascending**. AC that have no tests yet are listed in the **last row** and that row is **bold**.

| AC | Unit | Integration | E2E | Manual | Notes |
|----|------|-------------|-----|--------|-------|
| [AC-001](10-acceptance-criteria.md#ac-001-us-01) | — | ✓ | — | — | `tests/integration/telegram_flow_test.go` |
| [AC-002](10-acceptance-criteria.md#ac-002-us-01) | ✓ | ✓ | — | — | Unit: `internal/core/handler_test.go`; integration: `tests/integration/telegram_flow_test.go` (empty, over max length) |
| [AC-003](10-acceptance-criteria.md#ac-003-us-02) | ✓ | — | — | — | Unit: `internal/core/run_test.go` (nil adapter/provider), `internal/telegram/adapter_test.go` (Run nil handler) |
| [AC-005](10-acceptance-criteria.md#ac-005-us-03) | ✓ | — | — | — | `internal/config/config_test.go` (invalid node config; extended: missing file, invalid JSON, users invalid/missing) |
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
| [AC-017](10-acceptance-criteria.md#ac-017-us-09) | ✓ | — | — | — | `internal/llmlog/llmlog_test.go` (TestLog_writesParseableJSONLWithRequiredFields: request/response recorded in parseable JSONL). |
| [AC-018](10-acceptance-criteria.md#ac-018-us-10) | ✓ | — | — | — | `internal/llmlog/llmlog_test.go` (TestLog_writesParseableJSONLWithRequiredFields: required fields, parseable format). |
| [AC-019](10-acceptance-criteria.md#ac-019-us-10) | ✓ | — | — | — | `internal/llmlog/llmlog_test.go` (TestNewWriter_rejectsPathThatIsFile, rejectsReadOnlyDirectory: destination unavailable → error). |
| [AC-020](10-acceptance-criteria.md#ac-020-us-11) | ✓ | ✓ | — | — | Unit: `internal/telegram/adapter_test.go` (notify_chat_id [REQ-023]: from config, fallback to first user, zero when none; SendMessage when bot nil); integration: `tests/integration/scheduler_config_test.go` (scheduler fires and runs tool) |
| [AC-021](10-acceptance-criteria.md#ac-021-us-11) | — | ✓ | — | — | `tests/integration/scheduler_config_test.go` (task with disallowed command not executed) |
| [AC-022](10-acceptance-criteria.md#ac-022-us-12) | ✓ | — | — | — | Unit: `internal/tools/run_on_node_test.go` (valid params, runner invoked), `registry_test.go` (Register/Get/contract, empty/duplicate name panics) |
| [AC-023](10-acceptance-criteria.md#ac-023-us-12) | ✓ | — | — | — | Unit: `internal/tools/run_on_node_test.go` (invalid params rejected), `registry_test.go` (ValidateParams) |
| [AC-024](10-acceptance-criteria.md#ac-024-us-13) | — | ✓ | — | — | `tests/integration/scheduler_config_test.go` (load different task file) |
| [AC-025](10-acceptance-criteria.md#ac-025-us-14) | — | — | — | ✓ | [06-manual-test-plan.md](06-manual-test-plan.md) (architecture review) |
| [AC-027](10-acceptance-criteria.md#ac-027-us-15) | — | — | — | ✓ | [06-manual-test-plan.md](06-manual-test-plan.md) (docs: tracked paths) |
| [AC-031](10-acceptance-criteria.md#ac-031-us-17) | ✓ | — | — | — | `internal/core/handler_test.go` (DEBUG vs INFO logging) |
| [AC-032](10-acceptance-criteria.md#ac-032-us-18) | — | — | — | ✓ | Manual only; scenario in [06-manual-test-plan.md](06-manual-test-plan.md). |
| [AC-033](10-acceptance-criteria.md#ac-033-us-19) | ✓ | — | — | — | Unit: `internal/config/config_test.go`, `internal/telegram/adapter_test.go` (token/users construction errors), `internal/llm/provider_test.go`, `internal/embedding/provider_test.go` (unsupported type, missing key) |
| [AC-034](10-acceptance-criteria.md#ac-034-us-11) | ✓ | — | — | — | `internal/scheduler/loader_test.go` (empty path, missing file, duplicate/empty name, invalid JSON) |
| [AC-035](10-acceptance-criteria.md#ac-035-us-12) | ✓ | — | — | — | `internal/tools/run_on_node_test.go` (nil runner, runner error) |
| [AC-036](10-acceptance-criteria.md#ac-036-us-08) | ✓ | — | — | — | `internal/llm/openai_test.go` (Complete error paths: empty choices, 4xx, invalid JSON, context canceled, unreachable). Supporting: `internal/core/handler_test.go` (memoryReadError still includes vector context; indexTurnError still returns reply). |
| [AC-037](10-acceptance-criteria.md#ac-037-us-07) | ✓ | — | — | — | `internal/embedding/openai_test.go` (Embed error paths: empty data, 4xx, invalid JSON, context canceled, unreachable). Supporting: `internal/core/handler_test.go` (indexTurnError still returns reply). |
| [AC-038](10-acceptance-criteria.md#ac-038-us-16) | ✓ | — | — | — | `internal/logredact/logredact_test.go` (Redact_builtInPatterns, BuiltInIDs: built-in patterns applied). |
| [AC-039](10-acceptance-criteria.md#ac-039-us-16) | ✓ | — | — | — | `internal/logredact/logredact_test.go` (ValidateConfig_emptyAdditional, ValidateConfig_valid without additional: only built-in used). |
| [AC-040](10-acceptance-criteria.md#ac-040-us-16) | ✓ | — | — | — | `internal/logredact/logredact_test.go` (Redact_additionalPatterns, Redact_builtInAndAdditional, ValidateConfig_valid: additional patterns applied, no duplicate built-in id). |
| [AC-041](10-acceptance-criteria.md#ac-041-us-16) | ✓ | — | — | — | Unit: `internal/config/config_test.go` (TestLoad_LogRedactionReservedID_ReturnsError, TestLoad_LogRedactionInvalidRegex_ReturnsError: refuse start, clear error). Unit: `internal/logredact/logredact_test.go` (ValidateConfig_reservedID, ValidateConfig_invalidRegex). |
| [AC-042](10-acceptance-criteria.md#ac-042-us-20) | ✓ | — | — | — | Unit: `cmd/pa/main_test.go` (config path from PA_CONFIG_DIR: set / unset or empty). `internal/config/resolve_test.go` (PA_DATA_DIR, PA_SECRETS_DIR: relative joined with base, absolute unchanged, empty unchanged, unset uses "."). |
| **[AC-026](10-acceptance-criteria.md#ac-026-us-15), [AC-028](10-acceptance-criteria.md#ac-028-us-16)–[AC-030](10-acceptance-criteria.md#ac-030-us-16)** | — | — | — | — | **No tests yet; feature or task not implemented (see [11-12-implementation-plan.md](11-12-implementation-plan.md)).** |
