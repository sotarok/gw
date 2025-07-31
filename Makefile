.PHONY: help build test clean lint fmt install release-dry release

BINARY_NAME=gw
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags "-s -w"

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME)

test: ## Run tests
	$(GO) test $(GOFLAGS) -race -coverprofile=coverage.txt -covermode=atomic ./...

test-verbose: ## Run tests with verbose output
	$(GO) test -v -race ./...

coverage: test ## Run tests and show coverage report
	$(GO) tool cover -html=coverage.txt -o coverage.html
	$(GO) tool cover -func=coverage.txt
	@echo "Coverage report generated at coverage.html"

coverage-report: ## Show coverage report in terminal
	@$(GO) tool cover -func=coverage.txt

clean: ## Clean build artifacts
	$(GO) clean
	rm -f $(BINARY_NAME)
	rm -f coverage.txt coverage.html

lint: ## Run linter
	@if ! which golangci-lint > /dev/null; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run

fmt: ## Format code
	$(GO) fmt ./...
	$(GO) mod tidy

install: build ## Install the binary to $GOPATH/bin
	$(GO) install

deps: ## Download dependencies
	$(GO) mod download
	$(GO) mod tidy

check: lint test ## Run all checks (lint + test)

# Cross compilation targets
build-all: ## Build for all platforms
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe

release-dry: ## Dry run of goreleaser
	@if ! which goreleaser > /dev/null; then \
		echo "Installing goreleaser..."; \
		go install github.com/goreleaser/goreleaser@latest; \
	fi
	goreleaser release --snapshot --clean --skip-publish

release: ## Create a new release (requires tag)
	@if ! which goreleaser > /dev/null; then \
		echo "Installing goreleaser..."; \
		go install github.com/goreleaser/goreleaser@latest; \
	fi
	goreleaser release --clean