.PHONY: help fmt test test-race test-integration vet vuln lint coverage coverage-html check check-boundaries build validate

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
	@echo "  make test-integration - Run only integration tests (requires Docker; two-user SSH uses Debian image)"
	@echo "  make vet    - Run go vet"
	@echo "  make vuln   - Run govulncheck on modules (known CVEs in dependencies)"
	@echo "  make lint   - Run golangci-lint (if installed)"
	@echo "  make coverage     - Print coverage summary (all tests)"
	@echo "  make coverage-html - Build HTML coverage report"
	@echo "  make check-boundaries - Verify module boundaries (no cycles, forbidden edges)"
	@echo ""
	@echo "Validation:"
	@echo "  make validate - Run AC coverage validator for all epics"
	@echo "  make validate EP-009 - Validate a single epic"
	@echo ""
	@echo "  make check  - Run fmt + vet + vuln + lint + test-race + coverage + check-boundaries"

# Build targets
build: bin/pa bin/validate
	@echo "✅ All binaries built successfully"

bin/pa: cmd/pa/main.go
	@mkdir -p bin
	go build -o ./bin/pa ./cmd/pa

bin/validate: ai-sdlc/tools/validate/main.go ai-sdlc/tools/validate/output.go ai-sdlc/tools/validate/ast_skip.go ai-sdlc/tools/validate/main_test.go
	@mkdir -p bin
	go build -o ./bin/validate ./ai-sdlc/tools/validate

validate: bin/validate
	@./bin/validate $(filter-out validate,$(MAKECMDGOALS))

fmt:
	go fmt ./...

test:
	go test -tags=integration ./...

test-race:
	go test -race -tags=integration ./...

test-integration:
	go test -tags=integration ./tests/integration/...

vet:
	go vet ./...

# Scans module dependencies for known vulnerabilities (not redundant with vet/lint).
# Uses go run so no separate install; https://go.dev/doc/security/vuln
vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest -tags=integration ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint is not installed."; \
		echo "Install: https://golangci-lint.run/welcome/install/"; \
		exit 1; \
	}
	golangci-lint run --build-tags=integration ./...

coverage:
	rm -f coverage.out
	go test -tags=integration -count=1 ./... -coverpkg=./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out

coverage-html:
	rm -f coverage.out
	go test -tags=integration -count=1 ./... -coverpkg=./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

check-boundaries:
	@./scripts/check-module-boundaries.sh

check: fmt vet vuln lint test-race coverage check-boundaries

# Allow `make validate EP-XXX` without "No rule to make target EP-XXX".
%:
	@:
