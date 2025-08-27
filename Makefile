# Go Agent Makefile

.PHONY: help build run test clean fmt example init

# Default target
help:
	@echo "Go Agent - Available Commands:"
	@echo ""
	@echo "  make init       - Initialize project"
	@echo "  make build      - Build the agent binary"
	@echo "  make run        - Run in interactive mode"
	@echo "  make example    - Run example"
	@echo "  make test       - Run tests"
	@echo "  make fmt        - Format code"
	@echo "  make clean      - Clean build artifacts"
	@echo ""

# Initialize project
init:
	@echo "Initializing project..."
	@go mod tidy
	@mkdir -p bin output records logs
	@echo "✓ Project initialized"

# Build project
build:
	@echo "Building Go Agent..."
	@mkdir -p bin
	@go build -o bin/go-agent ./cmd/main.go
	@echo "✓ Build complete: bin/go-agent"

# Run interactive mode
run: build
	@echo "Starting Go Agent..."
	@./bin/go-agent -i

# Run example
example:
	@echo "Running example..."
	@go run ./examples/simple_example.go

# Run tests
test:
	@echo "Running tests..."
	@go test ./pkg/... -v

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/ output/ records/ logs/
	@echo "✓ Clean complete"
