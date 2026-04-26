.PHONY: help build test test-coverage lint fmt vet run dev clean deps install-tools build-linux docker-build docker-run check

# Binary name
BINARY_NAME=pako-tts
BUILD_DIR=bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet
GOMOD=$(GOCMD) mod

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "; printf "Usage: make <target>\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?## / {printf "  %-16s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application binary into bin/
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

test: ## Run all tests with race detector
	$(GOTEST) -v -race ./...

test-coverage: ## Run tests and generate HTML coverage report
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

lint: ## Run golangci-lint
	golangci-lint run ./...

fmt: ## Format Go code
	$(GOFMT) ./...

vet: ## Run go vet
	$(GOVET) ./...

run: ## Run the application locally on HTTP_PORT=7009
	set -a; source ./.env; set +a; HTTP_PORT=7009 $(GOCMD) run ./cmd/server

dev: ## Run with hot reload (requires air)
	air

clean: ## Remove build artifacts and coverage files
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

deps: ## Download and tidy module dependencies
	$(GOMOD) download
	$(GOMOD) tidy

install-tools: ## Install development tools (golangci-lint, air)
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/air-verse/air@latest

build-linux: ## Cross-compile a linux/amd64 binary
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/server

docker-build: ## Build the Docker image (pako-tts:latest)
	docker build -t pako-tts:latest .

docker-run: ## Run the Docker image with .env on port 7009
	docker run -p 7009:8080 --env-file .env pako-tts:latest

check: fmt vet lint test ## Run fmt, vet, lint, and tests
