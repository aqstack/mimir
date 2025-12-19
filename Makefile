.PHONY: build run test clean docker docker-run lint fmt help

# Variables
BINARY_NAME=kallm
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Default target
help:
	@echo "kallm - Kubernetes-native LLM Semantic Cache"
	@echo ""
	@echo "Usage:"
	@echo "  make build       Build the binary"
	@echo "  make run         Run locally"
	@echo "  make test        Run tests"
	@echo "  make lint        Run linter"
	@echo "  make fmt         Format code"
	@echo "  make docker      Build Docker image"
	@echo "  make docker-run  Run Docker container"
	@echo "  make clean       Clean build artifacts"

# Build
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/kallm

# Run locally
run: build
	./bin/$(BINARY_NAME)

# Run tests
test:
	go test -v -race -cover ./...

# Run linter
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Build Docker image
docker:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg DATE=$(DATE) \
		-t $(BINARY_NAME):$(VERSION) \
		-t $(BINARY_NAME):latest \
		.

# Run Docker container
docker-run:
	docker run -p 8080:8080 \
		-e OPENAI_API_KEY=$(OPENAI_API_KEY) \
		$(BINARY_NAME):latest

# Clean
clean:
	rm -rf bin/
	go clean
