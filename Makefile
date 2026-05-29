.PHONY: help fmt test test-race test-e2e test-integration vet vuln lint coverage coverage-e2e coverage-html check check-boundaries build validate

GOLANGCI_LINT_VERSION ?= v2.5.0

help:
	@echo "Available commands:"
	@echo ""
	@echo "Build:"
	@echo "  make build  - Build all binaries (pa + validate)"
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
	@echo "  make validate - Run AC coverage validator for all epics"
	@echo "  make validate EP-009 - Validate a single epic"
	@echo ""
	@echo "  make check  - Run fmt + vet + vuln + lint + test-race + test-e2e + coverage + check-boundaries"

# Build targets
build: bin/pa bin/validate
	@echo "✅ All binaries built successfully"

bin/pa: cmd/pa/main.go
	@mkdir -p bin
	go build -o ./bin/pa ./cmd/pa

bin/validate:
	@test -f ai-sdlc/tools/validate/go.mod || (echo "Missing ai-sdlc/: clone https://github.com/asubbot/ai-sdlc at pin in ai-sdlc.version (see README)" >&2; exit 1)
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

check: fmt vet vuln lint test-race test-e2e coverage check-boundaries

# Allow `make validate EP-XXX` without "No rule to make target EP-XXX".
%:
	@:
