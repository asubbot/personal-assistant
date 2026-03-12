# Multi-stage build for PersonalAssistant (EP-104, REQ-002). Target: linux/amd64 (e.g. Synology DS220+).

# Build stage: CGO required for SQLite + sqlite-vec.
FROM golang:1.26-alpine AS builder
RUN apk add --no-cache build-base
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG CGO_ENABLED=1
ARG GOOS=linux
ARG GOARCH=amd64
RUN go build -ldflags="-s -w" -o /pa ./cmd/pa

# Runtime stage: minimal Alpine, single binary.
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
RUN adduser -D -g "" pa
USER pa
COPY --from=builder /pa /pa
# Config directory; config file is config.json inside it. Override via PA_CONFIG_DIR.
ENV PA_CONFIG_DIR=/etc/pa
ENTRYPOINT ["/pa"]
CMD []
