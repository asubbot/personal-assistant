# PersonalAssistant

Go application: Telegram bot, core orchestration, long-term memory (markdown), vector index, LLM providers, scheduler, tools, SSH access to nodes. Target: Synology DS220+ (Docker). See [docs/EP-104/REQUIREMENTS.md](docs/EP-104/REQUIREMENTS.md) and [docs/EP-104/](docs/EP-104/) for design and implementation plan.

**Requirements:** Go 1.26+ (CGO for vector store: SQLite + sqlite-vec). Config is a JSON file at `config/config.json` (config directory overridable via `PA_CONFIG_DIR`). Secrets (tokens, API keys, SSH keys) are stored in files; config references them by path.

---

## Environment variables

| Variable         | Default      | Description |
|------------------|--------------|-------------|
| `PA_CONFIG_DIR`  | `./config`   | Directory containing the main config file; the file name is always `config.json`. This directory is the base for config-related paths (users, allowlist, scheduled_tasks). |
| `PA_DATA_DIR`    | `.`          | Base directory for data paths: `memory_dir`, `log_path`, `vector_index_path`, `llm_log_dir`. Relative path values in config are joined with this base. For local run with data in `./data/`, set `PA_DATA_DIR=./data` in `.env`. |
| `PA_SECRETS_DIR` | `.`          | Base directory for secret file paths: `token_path`, `users_path`, `api_key_path`, `private_key_path`. Relative path values in config are joined with this base. For local run with secrets in `.secrets/`, set `PA_SECRETS_DIR=.secrets` in `.env`. |
| `PA_LOG_LEVEL`   | `info`       | Log level: `info` or `debug` (case-insensitive). At `debug`, the core logs full LLM request and response (including memory/vector context) in the handler; at `info`, only metadata (message count, response length, token usage). |

Defined in `.env` (see Setup). With [direnv](https://direnv.net/), `.envrc` loads `.env` into the shell.

---

## Setup (once per machine)

```bash
go mod tidy
cp .env.example .env
# Edit .env: set PA_CONFIG_DIR, PA_SECRETS_DIR (e.g. .secrets for local), PA_LOG_LEVEL if needed
```

With [direnv](https://direnv.net/): `direnv allow` so `.env` is loaded in the shell.

---

## Build and run

```bash
# Build
go build -o pa ./cmd/pa

# Run (set env explicitly; data in ./data/, secrets in .secrets/)
PA_CONFIG_DIR=./config PA_DATA_DIR=./data PA_SECRETS_DIR=.secrets go run ./cmd/pa

# Or rely on .env / direnv, then:
go run ./cmd/pa

# Debug LLM conversation (full request/response in logs)
PA_CONFIG_DIR=./config PA_DATA_DIR=./data PA_SECRETS_DIR=.secrets PA_LOG_LEVEL=debug go run ./cmd/pa
```

### Verify node access

To check that SSH access to all configured nodes works (without starting the bot):

```bash
PA_CONFIG_DIR=./config PA_DATA_DIR=./data PA_SECRETS_DIR=.secrets go run ./cmd/pa -verify-nodes
# Optional: use another allowlisted command (e.g. "echo ok")
PA_CONFIG_DIR=./config PA_DATA_DIR=./data PA_SECRETS_DIR=.secrets go run ./cmd/pa -verify-nodes -verify-nodes-command "echo ok"
```

The command loads config and allowlist, connects to each node over SSH, runs one allowlisted command per node (default: `uptime`), and reports success or failure. Exit code 0 only when all nodes succeed. Ensure each node's allowlist file exists at the path set in config (`nodes.<id>.command_allowlist_path`) and contains the probe command (e.g. `uptime`).

### Docker deploy

Build and run in a container. The same `config/` directory is used; secrets are file-based (explicit, traceable).

**Environment in container** (set in `docker-compose.yml`):

| Variable          | Value in container | Purpose |
|-------------------|--------------------|---------|
| `PA_CONFIG_DIR`   | `/etc/pa`          | Config directory (volume `./config`). Base for `command_allowlist_path`, `scheduled_tasks_path`. |
| `PA_DATA_DIR`     | `/data`            | Data directory (volume `pa_data`). Base for `memory_dir`, `log_path`, `vector_index_path`, `llm_log_dir`. |
| `PA_SECRETS_DIR`  | `/run/secrets`     | Secrets directory (Docker secrets). Base for `token_path`, `users_path`, API keys, node private keys. |
| `PA_LOG_LEVEL`    | `info`             | Log level. |
| `TZ`              | (optional)         | Timezone for summarization cron (see below). Set to config's `pa_timezone` for correct \"yesterday\" / \"last month\" / \"last year\"; default UTC. |

**Summarization (cron):** The container runs cron in the background. Cron runs summarization on a schedule: **day** at 00:15, **month** on the 1st at 00:30, **year** on 1 January at 00:45. Set **TZ** in the container environment to match config's **pa_timezone** (e.g. `TZ=Europe/Moscow`) so that date ranges are correct; if unset, UTC is used. On transient failure (e.g. LLM timeout) the summarization script retries up to 2–3 times with a 1–2 minute pause before exiting.

Use a config with **bare secret filenames** (e.g. `token_path`: `"telegram_bot_token.txt"`, `users_path`: `"telegram_users.json"`, `api_key_path`: `"openai_api_key.txt"`) so that with `PA_SECRETS_DIR=.secrets` locally and `PA_SECRETS_DIR=/run/secrets` in Docker the same config works: secrets resolve to `.secrets/<name>` locally and `/run/secrets/<name>` in the container.

**Setup:**

1. **Config** — Ensure `config/config.json` exists with relative paths; secret paths are bare filenames (see `config/config.docker.example.json`).
2. **Secrets** — Create `.secrets/` and add one file per secret (no trailing newline for tokens). Same files are used for local run and for Docker (Compose mounts them into the container):

   | File in `.secrets/`     | Content          | Config field(s)        |
   |-------------------------|------------------|------------------------|
   | `telegram_bot_token.txt`| Bot token        | `telegram.token_path`  |
   | `telegram_users.json`   | JSON list of users | `telegram.users_path` |
   | `openai_api_key.txt`    | OpenAI API key   | `llm_providers[].api_key_path`, `embedding.api_key_path` |

   For nodes with SSH keys, add files (e.g. `pa_nas_ed25519`) and mount them in `docker-compose.yml` with the same target name as in config.

**Run:**

```bash
docker compose up -d --build
docker compose logs -f pa
docker compose down
```

---

## Development

```bash
make check  # fmt + vet + lint + test (coverage includes all tests)
```

Integration tests live in `tests/integration/` (build tag `integration`). They are included in `make test` and `make check`; coverage is collected from all tests. Use `make test-integration` to run only integration tests.

---

## Config

See [docs/EP-104/implementation-plan.md](docs/EP-104/implementation-plan.md) — section **Config file (JSON)** at the end of the file. Main config is JSON; paths to secrets (tokens, API keys, SSH keys) are set in config, not in env. Related files: Telegram users (JSON), command allowlist (text), scheduled tasks (JSON). Log level for application output is controlled by `PA_LOG_LEVEL` (see Environment variables above). Optional `telegram.notify_chat_id` is used as the default chat for the scheduler’s `notify` action when set.

**Paths in config:** Path values in the config file are either **absolute** (used as-is) or **relative** (joined with a base). Three bases are used: (1) config directory (`PA_CONFIG_DIR`) for `command_allowlist_path`, `scheduled_tasks_path`; (2) `PA_DATA_DIR` for `memory_dir`, `log_path`, `vector_index_path`, `llm_log_dir`; (3) `PA_SECRETS_DIR` for `token_path`, `users_path`, `api_key_path`, `private_key_path`. Default for `PA_DATA_DIR` and `PA_SECRETS_DIR` is `.` (current working directory), so relative paths behave as before when these env vars are unset. **paths.llm_log_retention_days** (required): number of days to keep LLM log files; files older than this are removed when summarization runs. Must be >= 1 (application refuses to start otherwise). Recommended: **7**.

**Scheduled tasks:** The file at `paths.scheduled_tasks_path` is a JSON array of tasks. Each task has a unique `name` (string), `schedule` (cron or `@every` interval), `action` (tool name or `notify`), and `params`. Duplicate or empty names cause a load error. Task names appear in logs when a task runs. Full format and examples: implementation plan, **Config file (JSON)** → Scheduled tasks file.

**Adding nodes and scheduled tasks without rebuild:** Add a new node in config (under `nodes`) or a new task in the scheduled tasks file (path in `paths.scheduled_tasks_path`); restart the application so the new config/tasks are loaded. No Docker image rebuild is required (AC-024).

---

## Log redaction (secrets)

All data written to the LLM request/response log and to application log output is redacted: matching secret-like values are replaced with `[REDACTED]` before writing.

**Built-in patterns** (always applied; cannot be disabled or overridden by config):

| What is redacted        | Pattern / format |
|-------------------------|------------------|
| OpenAI API keys         | `sk-` followed by 20+ alphanumeric characters |
| Telegram bot tokens     | Numeric bot ID, colon, 35-character token (e.g. `1234567890:AAH...`) |
| Bearer tokens           | Word `Bearer` (case-insensitive) followed by the token (e.g. JWT) |
| Paths to secret files   | File paths containing `token`, `secret`, `key`, `credential`, or `password` (e.g. `/run/secrets/openai_key`) |

**Additional patterns:** You can add more patterns in config under `log_redaction.additional_patterns`: each entry has `id`, `regex`, and `replacement`. Pattern `id` must not equal any built-in id (`api_key_openai`, `telegram_bot_token`, `bearer_token`, `generic_secret_path`); the regex must be valid or the app refuses to start. See the implementation plan, **Config file (JSON)** → `log_redaction`, for the full format and examples.
