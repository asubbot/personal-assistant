# Installation

## Prerequisites

- **Go 1.26+** (see `go.mod` in the repository root).
- **CGO enabled** for the default build: the vector store uses SQLite with **sqlite-vec** (`github.com/mattn/go-sqlite3`). On Linux you typically need a C toolchain and SQLite development headers (e.g. `libsqlite3-dev` on Debian/Ubuntu).
- **Network**: outbound HTTPS for Telegram and configured LLM/embedding APIs (unless you use only local endpoints).

## Get the code

```bash
git clone <your-clone-url> PersonalAssistant
cd PersonalAssistant
go mod tidy
```

## Local environment file (optional)

```bash
cp .env.example .env
# Edit .env: PA_CONFIG_DIR, PA_DATA_DIR, PA_SECRETS_DIR, PA_LOG_LEVEL as needed
```

With [direnv](https://direnv.net/), you can use `.envrc` to load `.env` automatically (`direnv allow`).

## Build

```bash
go build -o pa ./cmd/pa
```

## Configuration file

Copy the example and edit (never commit real secrets):

```bash
cp config/config.example.json config/config.json
cp config/known_hosts.example config/known_hosts
cp config/nas_allowlist.example config/nas_allowlist
cp config/scheduled_tasks.example.json config/scheduled_tasks.json
```

`known_hosts`, `nas_allowlist`, and `scheduled_tasks.json` are **gitignored** so you can keep real host keys, allowlists, and schedules locally without scrubbing before push. Adjust paths and nodes; populate `known_hosts` (e.g. `ssh-keyscan`). See [configuration.md](configuration.md).

## First run

```bash
PA_CONFIG_DIR=./config PA_DATA_DIR=./data PA_SECRETS_DIR=.secrets ./pa
```

Or set the same variables in `.env` and run `./pa` from a shell where they are exported.

## Contributors: full code quality gate (not installation)

**`make check` does not install or deploy PersonalAssistant.** It runs the full automated verification of the **source tree**, in order: `go fmt`, `go vet`, `golangci-lint` (with the integration build tag), **`go test -race -tags=integration ./...`** (race detector; slower than plain tests), **`go test` with coverage** across `./...`, and the **module-boundaries** script.

```bash
make check
```

For a **non-race** test run (e.g. faster local iteration), use `make test`. For integration tests only: `make test-integration`.

Integration tests require **Docker** and tools such as `ssh-keygen` / `ssh-keyscan` on `PATH`. See the root [README.md](../README.md#development).
