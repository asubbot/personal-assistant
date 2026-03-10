# PersonalAssistant

Go application: Telegram bot, core orchestration, long-term memory (markdown), vector index, LLM providers, scheduler, tools, SSH access to nodes. Target: Synology DS220+ (Docker). See [docs/EP-104/REQUIREMENTS.md](docs/EP-104/REQUIREMENTS.md) and [docs/EP-104/](docs/EP-104/) for design and implementation plan.

**Requirements:** Go 1.26+ (CGO for vector store: SQLite + sqlite-vec). Config is a JSON file at `config/config.json` (path overridable via `-config` or `PA_CONFIG_PATH`). Secrets (tokens, API keys, SSH keys) are stored in files; config references them by path.

---

## Environment variables

| Variable         | Default      | Description |
|------------------|--------------|-------------|
| `PA_CONFIG_PATH` | `./config/config.json` | Path to the main config JSON file. Overridden by `-config` flag. |
| `PA_LOG_LEVEL`   | `info`       | Log level: `info` or `debug` (case-insensitive). At `debug`, the core logs full LLM request and response (including memory/vector context) in the handler; at `info`, only metadata (message count, response length, token usage). |

Defined in `.env` (see Setup). With [direnv](https://direnv.net/), `.envrc` loads `.env` into the shell.

---

## Setup (once per machine)

```bash
go mod tidy
cp .env.example .env
# Edit .env: set PA_CONFIG_PATH and PA_LOG_LEVEL if needed
```

With [direnv](https://direnv.net/): `direnv allow` so `.env` is loaded in the shell.

---

## Build and run

```bash
# Build
go build -o pa ./cmd/pa

# Run (uses PA_CONFIG_PATH from env or default ./config/config.json)
go run ./cmd/pa
go run ./cmd/pa -config=/path/to/config.json

# Debug LLM conversation (full request/response in logs)
PA_LOG_LEVEL=debug go run ./cmd/pa
```

### Verify node access

To check that SSH access to all configured nodes works (without starting the bot):

```bash
go run ./cmd/pa -config ./config/config.json -verify-nodes
# Optional: use another allowlisted command (e.g. "echo ok")
go run ./cmd/pa -config ./config/config.json -verify-nodes -verify-nodes-command "echo ok"
```

The command loads config and allowlist, connects to each node over SSH, runs one allowlisted command per node (default: `uptime`), and reports success or failure. Exit code 0 only when all nodes succeed. Ensure each node's allowlist file exists at the path set in config (`nodes.<id>.command_allowlist_path`) and contains the probe command (e.g. `uptime`).

---

## Development

```bash
make fmt    # Format code
make test   # Run all tests (unit + integration)
make test-integration  # Run only integration tests
make vet    # go vet
make lint   # golangci-lint (install separately)
make check  # fmt + vet + lint + test (coverage includes all tests)
```

Integration tests live in `tests/integration/` (build tag `integration`). They are included in `make test` and `make check`; coverage is collected from all tests. Use `make test-integration` to run only integration tests.

---

## Config

See [docs/EP-104/implementation-plan.md](docs/EP-104/implementation-plan.md) — section **Config file (JSON)** at the end of the file. Main config is JSON; paths to secrets (tokens, API keys, SSH keys) are set in config, not in env. Related files: Telegram users (JSON), command allowlist (text), scheduled tasks (JSON). Log level for application output is controlled by `PA_LOG_LEVEL` (see Environment variables above). Optional `telegram.notify_chat_id` is used as the default chat for the scheduler’s `notify` action when set.

**Paths in config:** All path fields in the config file (`telegram.token_path`, `telegram.users_path`, `paths.*`, `llm_providers[].api_key_path`, `embedding.api_key_path`, `paths.scheduled_tasks_path`, `nodes.<id>.auth.private_key_path`, `nodes.<id>.command_allowlist_path`, etc.) are interpreted **relative to the project root** — i.e. the process current working directory (CWD) at startup. Run the application from the project root (e.g. `go run ./cmd/pa` from the repo root), or use absolute paths in config.

**Scheduled tasks:** The file at `paths.scheduled_tasks_path` is a JSON array of tasks. Each task has a unique `name` (string), `schedule` (cron or `@every` interval), `action` (tool name or `notify`), and `params`. Duplicate or empty names cause a load error. Task names appear in logs when a task runs. Full format and examples: implementation plan, **Config file (JSON)** → Scheduled tasks file.

**Adding nodes and scheduled tasks without rebuild:** Add a new node in config (under `nodes`) or a new task in the scheduled tasks file (path in `paths.scheduled_tasks_path`); restart the application so the new config/tasks are loaded. No Docker image rebuild is required (AC-024).
