# Go Agent Makefile

.PHONY: help build run test clean fmt example init web build-web

# Default target
help:
	@echo "Go Agent - Available Commands:"
	@echo ""
	@echo "  make init       - Initialize project"
	@echo "  make build      - Build the agent binary"
	@echo "  make run        - Run in interactive mode"
	@echo "  make web        - Run web server with SSE"
	@echo "  make build-web  - Build web server binary"
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

# Build web server
build-web:
	@echo "Building Web Server..."
	@mkdir -p bin
	@go build -o bin/go-agent-web ./cmd/web/main.go
	@echo "✓ Web server build complete: bin/go-agent-web"

# Run web server
web:
	@echo "Starting Web Server..."
	@echo "🚀 访问 http://localhost:8080 使用Web界面"
	@go run ./cmd/web/main.go

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/ output/ records/ logs/
	@echo "✓ Clean complete"
