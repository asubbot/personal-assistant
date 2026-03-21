# Docker deployment

The image targets **linux/amd64** and **linux/arm64** (Intel/AMD servers, Synology x86_64, Apple Silicon, ARM64 NAS/RPi-class hosts). It is **Debian-based** because CGO and sqlite-vec require glibc (Alpine/musl is not used for the default build).

## Files

- **[Dockerfile](../Dockerfile)** — multi-stage build (`golang:1.26-bookworm` → `debian:bookworm-slim`), installs `cron`, runs **[scripts/entrypoint.sh](../scripts/entrypoint.sh)**.
- **[docker-compose.yml](../docker-compose.yml)** — base Compose: build context, mounts config and data, injects Docker secrets into `/run/secrets` (no fixed `platforms:`).
- **[docker-compose.arm64.yml](../docker-compose.arm64.yml)** — includes the base and sets **`build.platforms: [linux/arm64]`**.
- **[docker-compose.amd64.yml](../docker-compose.amd64.yml)** — includes the base and sets **`build.platforms: [linux/amd64]`**.

## Compose: base + single-arch overlays

Use an overlay when you want BuildKit to target **one** architecture explicitly (e.g. cross-build **amd64** on Apple Silicon, or document the same command on CI).

```bash
# ARM64 (Apple Silicon, ARM64 Linux hosts)
docker compose -f docker-compose.arm64.yml up -d --build

# AMD64 (Intel/AMD, typical Synology x86_64)
docker compose -f docker-compose.amd64.yml up -d --build
```

**Base file only** (`docker compose up -d --build` without `-f`): Compose does not set `platforms:`; the image is built for the **host** architecture (typical local quick start).

## Dockerfile and BuildKit (Intel + Apple Silicon)

BuildKit / **buildx** sets **`TARGETARCH`** / **`TARGETPLATFORM`** for the requested platform.

1. **`builder`** runs on **`$BUILDPLATFORM`**, runs **`go mod download`**, then compiles **`GOOS=linux`** with **`GOARCH=$TARGETARCH`** (`amd64` or `arm64`), using **cross-compilers** when the host arch differs from the target (`x86_64-linux-gnu-gcc`, `aarch64-linux-gnu-gcc`).
2. **Runtime** uses **`$TARGETPLATFORM`** so the base image matches the binary.

**buildx (direct, without Compose):**

```bash
# Load one variant into the local engine
docker buildx build --platform linux/arm64 --load -t pa:local-arm64 .
docker buildx build --platform linux/amd64 --load -t pa:local-amd64 .

# Optional: push a multi-arch manifest to a registry (one tag, both architectures)
docker buildx build --platform linux/amd64,linux/arm64 -t YOUR_REGISTRY/pa:latest --push .
```

Plain **`docker build`** (no buildx platform) defaults to **linux/amd64** for the final image when **`TARGETARCH`** is unset.

If you see **checksum mismatch** after a failed build, run **`docker builder prune`** and rebuild (stale BuildKit module cache).

## Compose layout

- **Config (read-only):** host `./config` → container `/etc/pa` with `PA_CONFIG_DIR=/etc/pa`.
- **Data:** named volume `pa_data` → `/data` with `PA_DATA_DIR=/data`.
- **Secrets:** Compose `secrets:` read from host `./.secrets/<file>` and are mounted under `/run/secrets` with **`PA_SECRETS_DIR=/run/secrets`**. Target filenames must match the **bare names** in `config.json` (e.g. `telegram_bot_token.txt`).

Example secrets block (see repository `docker-compose.yml` and overlays for the authoritative list):

- `telegram_bot_token.txt`
- `telegram_users.json`
- `openai_api_key.txt`
- `node_ssh_private_key` — SSH private key for a node when `private_key_path` is the bare name `node_ssh_private_key` (copy your key to `.secrets/node_ssh_private_key`; never commit it).

If you still have the old filename **`openclaw_synology`**, rename it to **`node_ssh_private_key`** on the host and set **`private_key_path`** in `config.json` to **`node_ssh_private_key`**.

Add more secret files and `secrets:` entries for any other relative `*_path` values resolved via `PA_SECRETS_DIR` (e.g. additional node keys).

## Environment variables in the container

Compose loads an optional **`.env`** file next to `docker-compose.yml` into the **`pa`** service (`env_file` with `required: false`). Copy **[.env.example](../.env.example)** to **`.env`** and set variables there (e.g. `PA_LOG_LEVEL=debug`). Keys listed under **`environment:`** in **`docker-compose.yml`** still **override** the same names from `.env` so container paths stay correct (`PA_CONFIG_DIR`, `PA_DATA_DIR`, `PA_SECRETS_DIR`).

| Variable | Typical value | Notes |
|----------|---------------|--------|
| `PA_CONFIG_DIR` | `/etc/pa` | Set in Compose (overrides `.env`). |
| `PA_DATA_DIR` | `/data` | Set in Compose (overrides `.env`). |
| `PA_SECRETS_DIR` | `/run/secrets` | Set in Compose (overrides `.env`). |
| `PA_LOG_LEVEL` | (unset → app default `info`) | From `.env` if set; use `debug` only on trusted hosts (full LLM bodies in logs). |
| `TZ` | e.g. `Europe/Lisbon` | **Optional but recommended** so in-container cron’s “yesterday” / “last month” / “last year” match `pa_timezone` in config. Default in entrypoint is `UTC`. |

## In-container summarization (cron)

[entrypoint.sh](../scripts/entrypoint.sh) writes `/etc/cron.d/pa-summarize` and starts `cron` in the background before launching `/pa`:

- **Day:** 00:15 daily — runs `/usr/local/bin/summarize.sh day`
- **Month:** 00:30 on the 1st — `summarize.sh month`
- **Year:** 00:45 on 1 January — `summarize.sh year`

[summarize.sh](../scripts/summarize.sh) computes the target date using **`TZ`**, then runs `/pa -summarize=<date>` with up to **3** attempts and **60s** pause between failures.

## Commands

```bash
# Pick the overlay that matches the image arch you need (see “Compose: base + single-arch overlays”).
docker compose -f docker-compose.arm64.yml up -d --build
docker compose -f docker-compose.arm64.yml logs -f pa
docker compose -f docker-compose.arm64.yml down
```

Use **`docker-compose.amd64.yml`** instead of **`docker-compose.arm64.yml`** when you need **linux/amd64**. For host-native build without a platform pin, you can use the base file only: `docker compose up -d --build`.

Ensure `config/config.json` exists and secret files are present under `.secrets/` as referenced by Compose.
