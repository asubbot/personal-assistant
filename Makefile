.PHONY: help fmt test test-race test-integration vet lint coverage coverage-html check check-boundaries

help:
	@echo "Available commands:"
	@echo "  make fmt    - Format Go code"
	@echo "  make test   - Run all tests (unit + integration; integration tests require Docker)"
	@echo "  make test-race - Same as test but with -race (slower; no coverage)"
	@echo "  make test-integration - Run only integration tests (requires Docker; two-user SSH uses Debian image)"
	@echo "  make vet    - Run go vet"
	@echo "  make lint   - Run golangci-lint (if installed)"
	@echo "  make coverage     - Print coverage summary (all tests)"
	@echo "  make coverage-html - Build HTML coverage report"
	@echo "  make check-boundaries - Verify module boundaries (no cycles, forbidden edges)"
	@echo "  make check  - Run fmt + vet + lint + test-race + coverage + check-boundaries"

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

check: fmt vet lint test-race coverage check-boundaries
