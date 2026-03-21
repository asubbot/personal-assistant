# PersonalAssistant

Go service: Telegram bot, conversation core, long-term memory (markdown), vector search, multiple LLM providers, optional **tool-path escalation** across providers, scheduler, tools, SSH access to configured nodes. Docker image: **linux/amd64** and **linux/arm64** (e.g. Synology x86_64, Apple Silicon / ARM NAS). Details: [docs/docker.md](docs/docker.md).

**Documentation:** operator guides live under **[docs/](docs/README.md)** (installation, configuration, Docker, operations, troubleshooting). This README is the short entry point.

**Design and process:** [ai-sdlc/](ai-sdlc/) · epic artefacts [ai-sdlc-artefacts/epics/](ai-sdlc-artefacts/epics/) (e.g. EP-001 MVP, EP-004 tools, EP-006 escalation).

**Requirements:** **Go 1.26+** with **CGO** (SQLite + sqlite-vec). Main config: **`config.json`** inside **`PA_CONFIG_DIR`** (default `./config`). Secrets are **files** referenced from JSON — never commit real tokens or keys.

---

## Quick start

```bash
go mod tidy
cp .env.example .env
cp config/config.example.json config/config.json
cp config/known_hosts.example config/known_hosts
cp config/nas_allowlist.example config/nas_allowlist
cp config/scheduled_tasks.example.json config/scheduled_tasks.json
# Edit config.json; fill known_hosts (e.g. ssh-keyscan); place secrets under .secrets/ (or set PA_SECRETS_DIR)
```

```bash
go build -o pa ./cmd/pa
PA_CONFIG_DIR=./config PA_DATA_DIR=./data PA_SECRETS_DIR=.secrets ./pa
```

More detail: [docs/installation.md](docs/installation.md), [docs/configuration.md](docs/configuration.md).

---

## Environment variables (summary)

| Variable | Default | Description |
|----------|---------|-------------|
| `PA_CONFIG_DIR` | `./config` | Directory containing `config.json`; base for relative allowlist, scheduled tasks, `known_hosts`, tool catalog paths. |
| `PA_DATA_DIR` | `.` | Base for relative `memory_dir`, `log_path`, `vector_index_path`, `llm_log_dir`. |
| `PA_SECRETS_DIR` | `.` | Base for relative secret file paths (Telegram, API keys, SSH keys). |
| `PA_LOG_LEVEL` | `info` | `slog` level; **`debug`** logs full LLM request/response in the handler. |

Full behaviour: [docs/configuration.md](docs/configuration.md#environment-variables). Optional: load from `.env` / [direnv](https://direnv.net/).

---

## Common commands

```bash
# Node SSH check (does not start the bot)
./pa -verify-nodes
./pa -verify-nodes -verify-nodes-command "echo ok"

# Summarization (then exit)
./pa -summarize=2026-03-19

# Docker (pick arch: arm64 or amd64 — see docs/docker.md)
docker compose -f docker-compose.arm64.yml up -d --build
```

See [docs/operations.md](docs/operations.md) and [docs/docker.md](docs/docker.md).

---

## Development

**`make check`** does not install the app — it runs the full local **quality gate** on the repo: format, vet, lint, **tests with the race detector** (`-race`, integration tag), then a **coverage** pass, and module-boundary checks.

```bash
make check
```

Integration tests need **Docker** and SSH tooling on `PATH`; see [docs/installation.md](docs/installation.md#contributors-full-code-quality-gate-not-installation).

---

## Further reading

| Topic | Doc |
|-------|-----|
| Index of user guides | [docs/README.md](docs/README.md) |
| Docker secrets & cron | [docs/docker.md](docs/docker.md) |
| Config sections & validation | [docs/configuration.md](docs/configuration.md) |
| Problems | [docs/troubleshooting.md](docs/troubleshooting.md) |
| Threat model (security overview) | [ai-sdlc-artefacts/threat-model.md](ai-sdlc-artefacts/threat-model.md) |
