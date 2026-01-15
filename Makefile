.PHONY: build test lint clean install help fmt test-coverage test-integration

# Default target
help:
	@echo "AGK Developer CLI - Build Commands"
	@echo ""
	@echo "Usage:"
	@echo "  make build              Build the binary"
	@echo "  make test               Run unit tests"
	@echo "  make test-coverage      Run tests with coverage report"
	@echo "  make test-integration   Run integration tests"
	@echo "  make lint               Run linter"
	@echo "  make fmt                Format code"
	@echo "  make clean              Clean build artifacts"
	@echo "  make install            Install binary"
	@echo "  make help               Show this help message"

# Build binary
build:
	@echo "Building AGK..."
	@go build -o agk main.go
	@echo "✓ Build complete: ./agk"

# Run tests
test:
	@echo "Running tests..."
	@go test -v -race ./...
	@echo "✓ Tests passed"

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.txt -covermode=atomic ./...
	@go tool cover -html=coverage.txt -o coverage.html
	@echo "✓ Coverage report: coverage.html"

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	@go test -v -tags=integration ./test/integration/...
	@echo "✓ Integration tests passed"

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run ./...
	@echo "✓ Linting passed"

# Format code
fmt:
	@echo "Formatting code..."
	@gofmt -s -w .
	@goimports -w .
	@echo "✓ Code formatted"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f agk agk.exe coverage.txt coverage.html
	@rm -rf dist/ build/
	@go clean
	@echo "✓ Clean complete"

# Install binary
install:
	@echo "Installing AGK..."
	@go install -ldflags "-X github.com/agenticgokit/agk/cmd.Version=$(shell git describe --tags --always) -X github.com/agenticgokit/agk/cmd.GitCommit=$(shell git rev-parse HEAD) -X github.com/agenticgokit/agk/cmd.BuildDate=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')"
	@echo "✓ Installation complete"

# Development build with all flags
dev-build:
	@echo "Building development version..."
	@go build -o agk-dev main.go
	@echo "✓ Dev build complete: ./agk-dev"

# Run the binary
run: build
	@./agk version

# Development mode - watch for changes
watch:
	@echo "Watching for changes..."
	@go run main.go version

# Generate mocks (if using mockgen)
mocks:
	@echo "Generating mocks..."
	@go generate ./...
	@echo "✓ Mocks generated"
