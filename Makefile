.PHONY: help fmt test test-race test-e2e test-integration vet vuln lint coverage coverage-e2e coverage-html check check-boundaries verify-ai-sdlc-pin build bin/pa bin/validate run validate docker-build

GOLANGCI_LINT_VERSION ?= v2.5.0
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo unknown)
PA_LDFLAGS := -s -w \
	-X pa/internal/version.Commit=$(GIT_COMMIT) \
	-X pa/internal/version.BuildTime=$(BUILD_TIME)

help:
	@echo "Available commands:"
	@echo ""
	@echo "Build:"
	@echo "  make build        - Build all binaries (pa + validate)"
	@echo "  make run          - Run pa via go run with embedded git commit (dev)"
	@echo "  make docker-build - Build and start Docker image with git commit in binary"
	@echo ""
	@echo "Testing & Quality:"
	@echo "  make fmt    - Format Go code"
	@echo "  make test   - Run all tests (unit + integration; integration tests require Docker)"
	@echo "  make test-race - Same as test but with -race (slower; no coverage)"
	@echo "  make test-e2e - Run end-to-end tests under tests/e2e (build tag: e2e)"
	@echo "  make test-integration - Run only integration tests (requires Docker; two-user SSH uses Debian image)"
	@echo "  make vet    - Run go vet"
	@echo "  make vuln   - Run govulncheck on modules (known CVEs in dependencies)"
	@echo "  make lint   - Run pinned golangci-lint ($(GOLANGCI_LINT_VERSION))"
	@echo "  make coverage     - Print coverage summary (unit + integration; excludes e2e-tagged tests)"
	@echo "  make coverage-e2e - Print coverage summary for tests/e2e only (build tag: e2e)"
	@echo "  make coverage-html - Build HTML coverage report"
	@echo "  make check-boundaries - Verify module boundaries (no cycles, forbidden edges)"
	@echo ""
	@echo "Validation:"
	@echo "  make validate - Run AC + ears + req (all in-scope epics)"
	@echo "  make validate EP-009 - Run AC + ears + req for one epic"
	@echo ""
	@echo "  make check  - Verify ai-sdlc pin, then fmt + vet + vuln + lint + test-race + test-e2e + coverage + check-boundaries"

# Build targets.
# bin/pa and bin/validate are .PHONY: incrementality belongs to the Go build
# cache, which tracks every input. Make prerequisites cannot express the ldflags
# stamp below (git commit and build time), so a file target would report success
# while leaving a binary that claims the wrong commit.
build: bin/pa bin/validate
	@echo "✅ All binaries built successfully"

bin/pa:
	@mkdir -p bin
	go build -ldflags="$(PA_LDFLAGS)" -o ./bin/pa ./cmd/pa

run:
	go run -ldflags="$(PA_LDFLAGS)" ./cmd/pa

docker-build:
	GIT_COMMIT=$(GIT_COMMIT) BUILD_TIME=$(BUILD_TIME) docker compose up --build -d

bin/validate:
	@test -f ai-sdlc/tools/validate/go.mod || (echo "Missing ai-sdlc/: clone https://github.com/asubbot/ai-sdlc at pin in ai-sdlc.version (see docs/installation.md)" >&2; exit 1)
	@mkdir -p bin
	go build -C ai-sdlc/tools/validate -o $(CURDIR)/bin/validate .

validate: bin/validate
	@./bin/validate $(filter-out validate,$(MAKECMDGOALS))

fmt:
	go fmt ./...

test:
	go test -tags=integration ./...

test-race:
	go test -race -tags=integration ./...

test-e2e:
	go test -tags=integration,e2e -count=1 ./tests/e2e/...

test-integration:
	go test -tags=integration ./tests/integration/...

vet:
	go vet -tags=integration,e2e ./...

# Scans module dependencies for known vulnerabilities (not redundant with vet/lint).
# Uses go run so no separate install; https://go.dev/doc/security/vuln
vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest -tags=integration,e2e ./...

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run --build-tags=integration,e2e ./...

coverage:
	rm -f coverage.out
	go test -tags=integration -count=1 ./... -coverpkg=./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out

coverage-html:
	rm -f coverage.out
	go test -tags=integration -count=1 ./... -coverpkg=./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

coverage-e2e:
	rm -f coverage-e2e.out
	go test -tags=integration,e2e -count=1 ./tests/e2e/... -coverpkg=./tests/e2e/... -coverprofile=coverage-e2e.out -covermode=atomic
	go tool cover -func=coverage-e2e.out

check-boundaries:
	@./scripts/check-module-boundaries.sh

verify-ai-sdlc-pin:
	@./scripts/verify-ai-sdlc-pin.sh

check: verify-ai-sdlc-pin fmt vet vuln lint test-race test-e2e coverage check-boundaries

# Absorb the arguments of `make validate [subcommand] [EP-XXX|all]` so make does
# not treat them as targets. Listed explicitly rather than as a catch-all `%`,
# which also swallowed misspelled targets and exited 0.
EP-%:
	@:

ac req pipeline structure ears all:
	@:
