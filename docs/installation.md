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

## Agentic SDLC process clone (contributors only)

Required before **`make build`**, **`make check`**, or **`make validate`** (CI runs the same three gates). **Not** required for operator install below (`go build -o pa ./cmd/pa` / `./pa`).

The canonical process lives in [github.com/asubbot/ai-sdlc](https://github.com/asubbot/ai-sdlc). One-time setup from the product repository root (pin: last non-comment line in [`ai-sdlc.version`](../ai-sdlc.version), same extraction as [`.github/workflows/ci.yml`](../.github/workflows/ci.yml)):

```bash
git clone https://github.com/asubbot/ai-sdlc.git ai-sdlc
git -C ai-sdlc checkout "$(grep -v '^#' ai-sdlc.version | tail -1 | tr -d '[:space:]')"
```

Works for **tags and full commit SHAs** in [`ai-sdlc.version`](../ai-sdlc.version) (full clone + checkout; CI may use shallow `--branch` for tags only).

After a pin bump: `git -C ai-sdlc fetch && git -C ai-sdlc checkout <pin>`. Do **not** commit `ai-sdlc/` — it is listed in [`.gitignore`](../.gitignore).

**Pin enforcement:** `make check` runs [`scripts/verify-ai-sdlc-pin.sh`](../scripts/verify-ai-sdlc-pin.sh) first and fails fast if `ai-sdlc/` HEAD does not match the pin. Standalone `make build` and `make validate` do not run this check.

After checkout: process index at [`ai-sdlc/README.md`](../ai-sdlc/README.md); agents see [AGENTS.md](../AGENTS.md).

## Local environment file (optional)

```bash
cp .env.example .env
# Edit .env: PA_CONFIG_DIR, PA_DATA_DIR, PA_SECRETS_DIR, PA_LOG_LEVEL as needed
```

With [direnv](https://direnv.net/), you can use `.envrc` to load `.env` automatically (`direnv allow`).

## Build

```bash
make build
```

`make build` embeds the current git commit and UTC build time into `bin/pa`. Plain `go build` without `-ldflags` leaves `commit=unknown`.

## Configuration file

Copy the example and edit (never commit real secrets):

```bash
mkdir -p .config
cp config.examples/config.example.json .config/config.json
cp config.examples/known_hosts.example .config/known_hosts
cp config.examples/nas_allowlist.example .config/nas_allowlist
cp config.examples/tools.yaml .config/tools.yaml
```

The **`.config/`** directory is **gitignored**; committed templates live under **`config.examples/`**. Adjust paths and nodes; populate `known_hosts` (e.g. `ssh-keyscan`). See [configuration.md](configuration.md).

EP-019 note: scheduler persistence uses `paths.jobs_db_path` (default example value: `jobs.sqlite`, resolved under `PA_DATA_DIR`).

## First run

```bash
PA_DATA_DIR=./data PA_SECRETS_DIR=.secrets ./bin/pa
```

Or set the same variables in `.env` and run `./bin/pa` from a shell where they are exported.

Print build identity without starting the bot: `./bin/pa -version`.

## Contributors: full code quality gate (not installation)

Requires nested **`ai-sdlc/`** checkout at the pin in **`ai-sdlc.version`** (see [Agentic SDLC process clone](#agentic-sdlc-process-clone-contributors-only)).

Gate order (matches CI): clone process → **`make build`** → **`make check`** → **`make validate`**.

**`make check` does not install or deploy PersonalAssistant.** It runs the full automated verification of the **source tree**, in order: **ai-sdlc pin verify** ([`scripts/verify-ai-sdlc-pin.sh`](../scripts/verify-ai-sdlc-pin.sh)), `go fmt`, `go vet`, `golangci-lint` (with the integration build tag), **`go test -race -tags=integration ./...`** (race detector; slower than plain tests), **`go test` with coverage** across `./...`, and the **module-boundaries** script.

```bash
make check
make validate
```

For a **non-race** test run (e.g. faster local iteration), use `make test`. For integration tests only: `make test-integration`.

Integration tests require **Docker** and tools such as `ssh-keygen` / `ssh-keyscan` on `PATH`. See the root [README.md](../README.md#development).
