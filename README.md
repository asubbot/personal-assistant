# PersonalAssistant

[![CI](https://github.com/asubbot/personal-assistant/actions/workflows/ci.yml/badge.svg)](https://github.com/asubbot/personal-assistant/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/asubbot/personal-assistant/graph/badge.svg?token=0JE72IQFTW)](https://codecov.io/gh/asubbot/personal-assistant)
[![Go 1.26](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/doc/go1.26)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Go service: Telegram bot, conversation core, long-term memory (markdown), vector search, multiple LLM providers, optional **tool-path escalation** across providers, scheduler, tools, SSH access to configured nodes. Docker image: **linux/amd64** and **linux/arm64** (e.g. Synology x86_64, Apple Silicon / ARM NAS). Details: [docs/docker.md](docs/docker.md).

**Documentation:** operator guides live under **[docs/](docs/README.md)** (installation, configuration, Docker, operations, troubleshooting). This README is the short entry point.

**Design and process:** nested [ai-sdlc/](ai-sdlc/) clone (pin in [ai-sdlc.version](ai-sdlc.version); not in git — `git clone https://github.com/asubbot/ai-sdlc.git ai-sdlc` then checkout the pin) · epic artefacts [ai-sdlc-artefacts/epics/](ai-sdlc-artefacts/epics/) (e.g. EP-001 MVP, EP-004 tools, EP-006 escalation).

**Requirements:** **Go 1.26+** with **CGO** (SQLite + sqlite-vec). Main config: **`config.json`** inside **`PA_CONFIG_DIR`** (default `./.config`). Copy templates from **`config.examples/`** into **`.config/`** (gitignored). Secrets are **files** referenced from JSON — never commit real tokens or keys.

---

## Quick start

```bash
go mod tidy
cp .env.example .env
mkdir -p .config
cp config.examples/config.example.json .config/config.json
cp config.examples/known_hosts.example .config/known_hosts
cp config.examples/nas_allowlist.example .config/nas_allowlist
cp config.examples/tools.yaml .config/tools.yaml
# Edit .config/config.json; fill known_hosts (e.g. ssh-keyscan); place secrets under .secrets/ (or set PA_SECRETS_DIR)
```

```bash
go build -o pa ./cmd/pa
PA_DATA_DIR=./data PA_SECRETS_DIR=.secrets ./pa
```

More detail: [docs/installation.md](docs/installation.md), [docs/configuration.md](docs/configuration.md).

### Tool catalog durability (`create_tool`)

When the LLM uses the native **`create_tool`** tool, PersonalAssistant updates the YAML catalog at `paths.tool_catalog_path` using:

1. **Same-directory atomic replace** — a complete new file body is written to a temporary file in the same directory as the catalog, then renamed over the existing path so readers do not observe a partial YAML document mid-write.
2. **Explicit `Sync`** — the temporary file data is synced to stable storage before rename, and the parent directory is synced after rename so the directory entry for the catalog file is persisted.
3. **Post-write validation** — immediately after replace, the process re-loads the catalog with the same `toolcatalog.Load` entry point used at startup. If that validation fails, the previous catalog bytes are restored and the new tool is not applied to the in-memory catalog or tool vector index.

---

## Environment variables (summary)

| Variable | Default | Description |
|----------|---------|-------------|
| `PA_CONFIG_DIR` | `./.config` | Directory containing `config.json`; base for relative allowlist, `known_hosts`, tool catalog paths. |
| `PA_DATA_DIR` | `.` | Base for relative `memory_dir`, `log_path`, `vector_index_path`, `llm_log_dir`, `jobs_db_path`. |
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

**`make check`** does not install the app — it runs the full local **quality gate** on the repo: format, vet, **govulncheck** (known CVEs in module dependencies), lint, **tests with the race detector** (`-race`, integration tag), then a **coverage** pass, and module-boundary checks.

```bash
make check
```

**Coverage only** (summary table) or **HTML report** (opens in a browser):

```bash
make coverage
make coverage-html   # writes coverage.html (gitignored)
```

On **GitHub Actions** ([`.github/workflows/ci.yml`](.github/workflows/ci.yml)), each successful run adds a **Coverage** section to the job summary and uploads **`coverage-out`** (`coverage.out`) as a workflow artifact.

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
