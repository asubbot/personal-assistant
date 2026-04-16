# Operations

## Run the bot (foreground)

After configuration and secrets are in place:

```bash
./pa
```

With explicit env (example):

```bash
PA_DATA_DIR=./data PA_SECRETS_DIR=.secrets PA_LOG_LEVEL=info ./pa
```

The process runs until stopped (SIGINT/SIGTERM).

When **`nodes`** is non-empty, the binary runs a short SSH check per node (TCP, host key, key auth) immediately after startup setup. Failures are logged at **warn** severity; the bot still starts. For a stricter check that runs an allowlisted remote command and exits non-zero on failure, use **`-verify-nodes`**.

## CLI flags (`cmd/pa`)

All flags are defined in `cmd/pa/main.go` via the standard `flag` package.

| Flag | Purpose |
|------|---------|
| `-verify-nodes` | Load config, check allowlists, connect to each configured node over SSH, run **one** allowlisted command per node, then **exit** without starting the Telegram bot. Exit code **0** only if every node succeeds. |
| `-verify-nodes-command` | Command string for `-verify-nodes` (default `uptime`). Must appear in each node’s allowlist file. |
| `-summarize` | Run summarization for a scope and **exit**. Value format: **`YYYY-MM-DD`** (day), **`YYYY-MM`** (month), or **`YYYY`** (year). No default — flag must be non-empty. Also prunes old LLM log files per `paths.llm_log_retention_days`. |
| `-clear-context-on-start` | Before starting the bot, clear the **conversation turns** vector table (`vec_turns`) in the SQLite index. Does **not** delete summaries, notes, tool index data, or memory markdown files. |

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

### Token usage checks

To compare token usage before/after config or prompt changes, summarize one day of logs:

```bash
python3 - <<'PY'
import json, pathlib
p=pathlib.Path('.data/llm_logs/llm-2026-04-15.jsonl')
rows=[json.loads(l) for l in p.read_text().splitlines() if l.strip()]
pts=[r.get('usage',{}).get('prompt_tokens',0) for r in rows]
tts=[r.get('usage',{}).get('total_tokens',0) for r in rows]
def p95(v):
    if not v:
        return 0
    i=max(0, min(len(v)-1, int(round(0.95*(len(v)-1)))))
    return sorted(v)[i]
print('entries', len(rows))
print('prompt max', max(pts), 'p95', p95(pts), 'avg', round(sum(pts)/len(pts),1))
print('total  max', max(tts), 'p95', p95(tts), 'avg', round(sum(tts)/len(tts),1))
PY
```

## Scheduled jobs

- Runtime uses `paths.jobs_db_path` (`jobs.sqlite`) for persisted jobs.
- Legacy scheduler configuration fields are rejected at config load.
- Telegram management commands: `/jobs list`, `/jobs show <id>`, `/jobs pause <id>`, `/jobs resume <id>`, `/jobs run-now <id>`, `/jobs delete <id>`, `/jobs confirm-delete <id> <token>`.

### List responsiveness acceptance harness (profile-based)

Thresholds are defined outside runtime config in:

- `tests/integration/testdata/ep019/list_responsiveness_profiles.json`

Run the acceptance check for a selected profile:

```bash
PA_LIST_RESPONSIVENESS_PROFILE=baseline go test -tags=integration ./tests/integration/... -run TestEP019_ListResponsiveness_ProfileAcceptance
```

## LLM escalation (optional)

When `tools.llm_escalation.enabled` is true, startup logs include an `llm escalation` line with baseline index/model. When disabled, a single `llm model` line is logged for the first provider.
