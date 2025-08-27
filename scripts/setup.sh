#!/bin/bash

# Go Agent Setup Script

set -e

echo "🚀 Go Agent Project Setup"
echo "========================"

# Check Go version
echo "Checking Go version..."
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21 or higher"
    echo "   Download: https://golang.org/dl/"
    exit 1
fi

GO_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//')
echo "✅ Go version: $GO_VERSION"

# Download dependencies
echo "📦 Installing dependencies..."
go mod tidy

# Create necessary directories
echo "📁 Creating directories..."
mkdir -p bin output records logs

# Build project
echo "🔨 Building project..."
go build -o bin/go-agent ./cmd/main.go

echo ""
echo "✅ Setup complete!"
echo ""
echo "Next steps:"
echo "  1. Set API key: export OPENAI_API_KEY='your-key'"
echo "  2. Run: ./bin/go-agent -i"
echo ""