# Multi-stage build for PersonalAssistant (EP-104, REQ-002). Target: linux/amd64 (e.g. Synology DS220+).
# Debian-based: sqlite-vec C code requires glibc/Linux (u_int*_t etc.); Alpine/musl fails to compile.

# Build stage: CGO required for SQLite + sqlite-vec.
FROM golang:1.26-bookworm AS builder
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /build
COPY go.mod go.sum ./
# Reuse module cache across builds (BuildKit).
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG CGO_ENABLED=1
ARG GOOS=linux
ARG GOARCH=amd64
# BuildKit cache mounts: reuse Go module and build caches across builds (faster rebuilds).
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -o /pa ./cmd/pa

# Runtime stage: minimal Debian slim, binary + shared libs.
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    gosu \
    libsqlite3-0 \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -s /bin/false pa

# Entrypoint: run as root so we can chown the data volume; then exec app as user pa.
# Volume /data is often root-owned at first start; pa needs write access for memory_dir, logs, etc.
RUN printf '%s\n' \
    '#!/bin/sh' \
    'set -e' \
    'chown -R pa:pa /data 2>/dev/null || true' \
    'exec gosu pa /pa "$@"' \
    > /entrypoint.sh && chmod +x /entrypoint.sh

COPY --from=builder /pa /pa
# Config directory; config file is config.json inside it. Override via PA_CONFIG_DIR.
ENV PA_CONFIG_DIR=/etc/pa
ENTRYPOINT ["/entrypoint.sh"]
CMD []
