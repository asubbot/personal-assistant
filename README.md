# PersonalAssistant

Go application: Telegram bot, core orchestration, long-term memory (markdown), vector index, LLM providers, scheduler, tools, SSH access to nodes. Target: Synology DS220+ (Docker). See [docs/EP-104/REQUIREMENTS.md](docs/EP-104/REQUIREMENTS.md) and [docs/EP-104/](docs/EP-104/) for design and implementation plan.

**Requirements:** Go 1.26+ (CGO for vector store: SQLite + sqlite-vec). Config is a JSON file (path via `-config` or `PA_CONFIG_PATH`). Secrets (tokens, API keys, SSH keys) are stored in files; config references them by path.

---

## Environment variables

| Variable         | Default      | Description |
|------------------|--------------|-------------|
| `PA_CONFIG_PATH` | `./config.json` | Path to the main config JSON file. Overridden by `-config` flag. |
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

# Run (uses PA_CONFIG_PATH from env or default ./config.json)
go run ./cmd/pa
go run ./cmd/pa -config=/path/to/config.json

# Debug LLM conversation (full request/response in logs)
PA_LOG_LEVEL=debug go run ./cmd/pa
```

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

See [docs/EP-104/implementation-plan.md](docs/EP-104/implementation-plan.md) — section **Config file (JSON)** at the end of the file. Main config is JSON; paths to secrets (tokens, API keys, SSH keys) are set in config, not in env. Related files: Telegram users (JSON), command allowlist (text), scheduled tasks (JSON). Log level for application output is controlled by `PA_LOG_LEVEL` (see Environment variables above).
