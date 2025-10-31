.PHONY: all build test clean run-facilitator run-examples install deps

# Build all binaries
all: build

# Install dependencies
deps:
	go mod download
	go mod tidy

# Build all binaries
build: build-facilitator build-examples

# Build facilitator
build-facilitator:
	@echo "Building facilitator..."
	@mkdir -p bin
	go build -o bin/facilitator ./cmd/facilitator

# Build examples
build-examples:
	@echo "Building examples..."
	@mkdir -p bin
	go build -o bin/server-example ./examples/server
	go build -o bin/client-example ./examples/client
 
# Run facilitator
run-facilitator:
	go run cmd/facilitator/main.go

# Run server example
run-server-example:
	go run examples/server/main.go

# Run client example
run-client-example:
	go run examples/client/main.go

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Install tools
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Check for security vulnerabilities
audit:
	go list -json -m all | nancy sleuth

# Generate go.mod and go.sum
tidy:
	go mod tidy

# Docker build
docker-build:
	docker build -t x402-go:latest .

# Docker run
docker-run:
	docker run --env-file .env -p 8080:8080 x402-go:latest

# Help
help:
	@echo "Available targets:"
	@echo "  all              - Build all binaries (default)"
	@echo "  deps             - Download dependencies"
	@echo "  build            - Build all binaries"
	@echo "  build-facilitator - Build facilitator binary"
	@echo "  build-examples   - Build example binaries"
	@echo "  run-facilitator  - Run facilitator"
	@echo "  run-server-example - Run server example"
	@echo "  run-client-example - Run client example"
	@echo "  test             - Run tests"
	@echo "  test-coverage    - Run tests with coverage"
	@echo "  clean            - Clean build artifacts"
	@echo "  fmt              - Format code"
	@echo "  lint             - Lint code"
	@echo "  tidy             - Tidy go.mod"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-run       - Run Docker container"
