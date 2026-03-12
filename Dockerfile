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
    cron \
    gosu \
    libsqlite3-0 \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -s /bin/false pa

# Entrypoint: chown /data, write cron.d for summarization from env, start cron, exec app as user pa.
# Summarization auto-run is via in-container cron (day 0:15, month 1st 0:30, year Jan 1 0:45).
COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY --from=builder /pa /pa
COPY scripts/summarize.sh /usr/local/bin/summarize.sh
RUN chmod +x /usr/local/bin/summarize.sh
# Config directory; config file is config.json inside it. Override via PA_CONFIG_DIR.
ENV PA_CONFIG_DIR=/etc/pa
ENTRYPOINT ["/entrypoint.sh"]
CMD []
