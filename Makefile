COVERAGE_THRESHOLD := 90

.PHONY: help test cover fmt fmt-check lint clean

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)

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
