COVERAGE_THRESHOLD := 90

.PHONY: help test cover fmt fmt-check lint clean

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\\n", $$1, $$2}'

TEST_PKG := ./...

# Packages to exclude from coverage (examples, tests, generated code)
COVER_EXCLUDE := examples tests

test: ## Run tests
	go test -race $(TEST_PKG)

cover: ## Run tests with coverage (excludes examples)
	@# Get list of packages excluding examples and tests directories
	$(eval COVER_PKGS := $(shell go list ./... | grep -v -E '/(examples|tests)/'))
	go test -race -coverprofile=coverage.out -covermode=atomic $(COVER_PKGS)
	@go tool cover -func=coverage.out
	@coverage=$$(go tool cover -func=coverage.out | grep total | grep -oE '[0-9]+\.[0-9]+'); \
	echo "Coverage: $${coverage}%"; \
	awk -v cover="$${coverage}" -v thresh="$(COVERAGE_THRESHOLD)" 'BEGIN { if (cover < thresh) exit 1 }' || (echo "Coverage below threshold ($(COVERAGE_THRESHOLD)%)" && exit 1)

fmt: ## Format code
	gofmt -w .
	goimports -w .

fmt-check: ## Check formatting
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "gofmt check failed"; \
		gofmt -l .; \
		exit 1; \
	fi
	@if [ -n "$$(goimports -l .)" ]; then \
		echo "goimports check failed"; \
		goimports -l .; \
		exit 1; \
	fi

lint: ## Run linter
	@command -v golangci-lint >/dev/null || (echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run

clean: ## Clean generated files
	rm -f coverage.out
