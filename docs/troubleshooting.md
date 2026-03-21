# Troubleshooting

Use **symptom → check → fix**. Prefer `./pa` with `PA_LOG_LEVEL=debug` only on trusted machines — logs can include full LLM payloads.

## Process exits immediately with “load config”

| Check | Action |
|-------|--------|
| `PA_CONFIG_DIR` wrong | Ensure the directory exists and contains **`config.json`**. |
| JSON invalid | Validate JSON; compare with [config/config.example.json](../config/config.example.json). |
| Validation error message | Read the error: escalation rules, missing paths, `llm_log_retention_days` &lt; 1, duplicate scheduled task names, tool catalog errors, etc. Fix config per [configuration.md](configuration.md). |

## Telegram / bot does not respond

| Check | Action |
|-------|--------|
| Token file | Path from `telegram.token_path` must exist under `PA_SECRETS_DIR` if relative. |
| Users file | `telegram.users_path` must exist and list allowed users (JSON). |
| Network | Host must reach `api.telegram.org`. |

## SSH / node / “run on node” failures

| Check | Action |
|-------|--------|
| Private key | `nodes.<id>.auth.private_key_path` resolved with `PA_SECRETS_DIR`; file readable. |
| Docker: key “no such file” | Use a **bare filename** in config (e.g. `node_ssh_private_key`) and mount the key under **`PA_SECRETS_DIR`** with the **same basename** (see [docker.md](docker.md) secrets table). Host paths like `/Users/.../.ssh/...` do not exist inside the container unless you mount them explicitly. |
| Allowlist | `command_allowlist_path` relative to `PA_CONFIG_DIR`; command exactly allowlisted (no shell metacharacters where forbidden). |
| `known_hosts` | `paths.ssh_known_hosts_path` relative to config dir; host key must match. |
| Probe | Run `./pa -verify-nodes` and optional `-verify-nodes-command` (must be allowlisted). |

## “No nodes in config” on verify-nodes

Expected if `nodes` is empty or omitted — exit success with an informational log.

## Vector / SQLite errors at startup

| Check | Action |
|-------|--------|
| CGO / OS | Use a supported build (Linux glibc for typical Docker; local dev needs CGO toolchain). |
| Permissions | `PA_DATA_DIR` and parent of `vector_index_path` writable. |
| Embedding dimensions | Must match store expectations; changing embedding model dimensions may require a new index path. |

## macOS: CGO deprecation warnings when building (sqlite-vec)

When you run `go build`, `go run ./cmd/pa`, or similar, Clang may warn from **`github.com/asg017/sqlite-vec-go-bindings/cgo`** that **`sqlite3_auto_extension`** and **`sqlite3_cancel_auto_extension`** are deprecated (Apple’s SDK `sqlite3.h`: process-global auto extensions are not supported the same way on Apple platforms). The default vector store loads sqlite-vec through those APIs.

| Check | Action |
|-------|--------|
| Build failed? | These messages are usually **warnings**, not errors — the binary often still links and runs. If the build fails, capture the **first error** line (not only warnings). |
| Noisy output | **Ignore** for local dev, or set `CGO_CFLAGS="-Wno-deprecated-declarations"` for that shell/session (**caveat:** hides all deprecation warnings in affected CGO compiles, not only SQLite). |
| Prefer Linux toolchain | Use **Linux** for builds or runtime (e.g. the project Docker image — [docker.md](docker.md)); avoids Apple’s deprecated declarations for this path. |
| Longer term | Watch **sqlite-vec** / **sqlite-vec-go-bindings** releases for macOS-related changes; the repo may also document non-default vector backends that avoid CGO (see [installation.md](installation.md) prerequisites). |

## Summarization or cron failures in Docker

| Check | Action |
|-------|--------|
| `TZ` | Set container `TZ` to align with `pa_timezone` in config (see [docker.md](docker.md)). |
| LLM / embedding | `-summarize` uses configured providers; API keys and network must work. |
| Logs | `docker compose logs pa` — cron runs `/pa -summarize=...` via `summarize.sh` with retries. |

## Escalation-related config rejected at startup

When `tools.llm_escalation.enabled` is true: require **≥ 2** `llm_providers`, valid `baseline_index`, and **`max_per_user_message` ≥ 1**. Disable escalation or fix values.
