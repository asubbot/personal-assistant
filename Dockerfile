# syntax=docker/dockerfile:1
# Multi-stage build. linux/amd64 and linux/arm64 (Debian/glibc for CGO + sqlite-vec).
# Builder runs on $BUILDPLATFORM (native go mod download / compile — avoids QEMU issues).
# go build uses TARGETARCH from buildx with cross-compilers when host != target.

FROM --platform=$BUILDPLATFORM golang:1.26-bookworm AS builder
ARG TARGETOS=linux
ARG TARGETARCH
ARG BUILDARCH
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    libsqlite3-dev \
    gcc-x86-64-linux-gnu \
    libc6-dev-amd64-cross \
    gcc-aarch64-linux-gnu \
    libc6-dev-arm64-cross \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN set -eux; \
  ARCH="${TARGETARCH:-amd64}"; \
  case "$ARCH" in \
    amd64) \
      if [ "$BUILDARCH" = "amd64" ]; then export CC=gcc CXX=g++; \
      else export CC=x86_64-linux-gnu-gcc CXX=x86_64-linux-gnu-g++; fi \
      ;; \
    arm64) \
      if [ "$BUILDARCH" = "arm64" ]; then export CC=gcc CXX=g++; \
      else export CC=aarch64-linux-gnu-gcc CXX=aarch64-linux-gnu-g++; fi \
      ;; \
    *) echo "unsupported TARGETARCH=$ARCH (use amd64 or arm64)" >&2; exit 1 ;; \
  esac; \
  CGO_ENABLED=1 GOOS="${TARGETOS:-linux}" GOARCH="$ARCH" \
  go build -ldflags="-s -w" -o /pa ./cmd/pa

ARG TARGETPLATFORM=linux/amd64
FROM --platform=$TARGETPLATFORM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    cron \
    gosu \
    libsqlite3-0 \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -s /bin/false pa

COPY --chmod=755 scripts/entrypoint.sh /entrypoint.sh
COPY --from=builder /pa /pa
COPY --chmod=755 scripts/summarize.sh /usr/local/bin/summarize.sh
ENV PA_CONFIG_DIR=/etc/pa
ENTRYPOINT ["/entrypoint.sh"]
CMD []
