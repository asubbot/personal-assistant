# Operations

## Run the bot (foreground)

After configuration and secrets are in place:

```bash
./pa
```

With explicit env (example):

```bash
PA_CONFIG_DIR=./config PA_DATA_DIR=./data PA_SECRETS_DIR=.secrets PA_LOG_LEVEL=info ./pa
```

The process runs until stopped (SIGINT/SIGTERM).

## CLI flags (`cmd/pa`)

All flags are defined in `cmd/pa/main.go` via the standard `flag` package.

| Flag | Purpose |
|------|---------|
| `-verify-nodes` | Load config, check allowlists, connect to each configured node over SSH, run **one** allowlisted command per node, then **exit** without starting the Telegram bot. Exit code **0** only if every node succeeds. |
| `-verify-nodes-command` | Command string for `-verify-nodes` (default `uptime`). Must appear in each node’s allowlist file. |
| `-summarize` | Run summarization for a scope and **exit**. Value format: **`YYYY-MM-DD`** (day), **`YYYY-MM`** (month), or **`YYYY`** (year). No default — flag must be non-empty. Also prunes old LLM log files per `paths.llm_log_retention_days`. |
| `-clear-context-on-start` | Before starting the bot, clear the **conversation context** vector table (`vec_items`) in the SQLite index. Does **not** delete tool index data or memory markdown files. |

Examples:

```bash
./pa -verify-nodes
./pa -verify-nodes -verify-nodes-command "echo ok"
./pa -summarize=2026-03-19
```

## Application log

- Human-readable application log path is set by **`paths.log_path`** in config (resolved with `PA_DATA_DIR` if relative).
- Default severity is **`info`** unless `PA_LOG_LEVEL=debug`.

## LLM JSONL logs

- Directory: **`paths.llm_log_dir`** (resolved with `PA_DATA_DIR`).
- Daily files (naming convention implemented in `internal/llmlog`).
- Retention: **`paths.llm_log_retention_days`**; pruning runs when summarization is invoked (including via `-summarize`).

## Scheduler

If `paths.scheduled_tasks_path` points to a non-empty valid task list, the scheduler starts with the bot. Task names appear in logs when jobs run. The Telegram adapter can be used as notifier for `notify` actions when `telegram.notify_chat_id` is configured appropriately.

## LLM escalation (optional)

When `tools.llm_escalation.enabled` is true, startup logs include an `llm escalation` line with baseline index/model. When disabled, a single `llm model` line is logged for the first provider.
