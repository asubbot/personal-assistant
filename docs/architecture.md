# PersonalAssistant — architecture (composition root)

English overview of how the `pa` binary is wired. For a broader C4-style narrative (Russian), see [architecture-ru.md](architecture-ru.md).

## Composition root (`cmd/pa/wire`)

Server startup builds a single [`wire.Application`](../cmd/pa/wire/application.go) via [`wire.Build`](../cmd/pa/wire/build.go):

1. **`wire.BuildInfrastructure`** — Telegram adapter, memory store, embedding provider, vector stores, tool/skill indices, optional node runner.
2. **`Application.StartLLMProviders`** — conversation and summarization LLM handles.
3. **`Application.MaybeStartMemorySummarization`** — optional background worker.
4. **`Application.BuildToolRegistry`** — native tools (including scheduled-job creation when `paths.jobs_db_path` is set).
5. **`Application.BuildMessageHandler`** — core conversation handler.
6. **`cmd/pa` `wrapJobsHandler`** — `/jobs` command wrapper and async jobs DB initialization (stays in `main` to avoid import cycles with delivery types).

[`cmd/pa/main.go`](../cmd/pa/main.go) is the thin entry: flags, config load, logger setup, `wire.Build`, `runServer`, signal shutdown. CLI-only paths (`-summarize`, `-verify-nodes`, `-clear-context-on-start`) stay in `main`.

## Scheduled jobs runtime state (EP-042)

When `paths.jobs_db_path` is configured, [`wire.JobsRuntimeState`](../cmd/pa/wire/jobs_state.go) tracks an explicit phase:

| Phase | `/jobs` and `create_scheduled_job` | Readiness `scheduled_jobs` |
|-------|-----------------------------------|----------------------------|
| **Initializing** | Stable “initializing” soft message | Not OK, detail `initializing` |
| **Ready** | Manager commands run | OK |
| **Failed** | Stable “unavailable” soft message | Not OK, detail `initialization failed` |

Readiness evaluation lives in [`cmd/pa/wire/readiness.go`](../cmd/pa/wire/readiness.go) and must stay aligned with the phase enum.

## Adding a new subsystem — checklist

Use this when introducing a new long-lived dependency (adapter, store, index, background worker, tool, etc.):

1. **Constructor** — Add a focused constructor in `cmd/pa/wire/` (new file or extend [`infrastructure.go`](../cmd/pa/wire/infrastructure.go) / [`tools.go`](../cmd/pa/wire/tools.go)). Keep `internal/*` packages free of Telegram or process-global wiring.
2. **`BuildInfrastructure` or `Build`** — Open resources during infrastructure build or during `Application` method calls in the same order as existing subsystems (fail fast; call `Infrastructure.Close` on partial failure paths).
3. **`Application` fields** — Store handles on `wire.Application` only when `runServer` or readiness needs them after build.
4. **Teardown** — Release resources in `Infrastructure.Close` or a dedicated `Application` stop method; mirror defer order in `runServer` (same pattern as memory summarization worker).
5. **Tools / handler** — Register tools in `BuildToolRegistry`; pass dependencies into `BuildMessageHandler` via `core.BuildMessageHandler` options—do not construct subsystems inside `internal/core`.
6. **Readiness** — If operators probe subsystem health, add a check in `evalReadiness` with a stable check `name` and `detail` string.
7. **Tests** — Unit-test constructor failures and any new readiness or gating behaviour in `cmd/pa/*_test.go`.
8. **Config** — Document new keys in [`configuration.md`](configuration.md); add keys to example `config.json` with JSON `null` when optional (project config policy).

## Related docs

| Document | Topic |
|----------|--------|
| [operations.md](operations.md) | CLI flags, scheduler |
| [observability-http.md](observability-http.md) | Health and readiness HTTP |
| [configuration.md](configuration.md) | `config.json` |
