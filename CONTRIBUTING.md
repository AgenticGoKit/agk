# Contributing to AGK

Thank you for your interest in contributing to AGK! This document provides guidelines and instructions for contributing.

## Code of Conduct

Please be respectful and constructive in all interactions. We're building something together.

## Getting Started

### Prerequisites

- Go 1.21 or later
- Git

### Setup

```bash
# Clone the repository
git clone https://github.com/agenticgokit/agk.git
cd agk

# Install dependencies
go mod download

# Build
go build ./...

# Run tests
go test ./...
```

## How to Contribute

### Reporting Issues

- Check existing issues before creating a new one
- Use a clear, descriptive title
- Include steps to reproduce, expected vs actual behavior
- Include Go version and OS

### Submitting Pull Requests

1. **Fork** the repository
2. **Create a branch**: `git checkout -b feature/your-feature`
3. **Make changes** following our coding standards
4. **Test**: `go test ./...`
5. **Lint**: `golangci-lint run`
6. **Commit** with clear messages
7. **Push** and create a Pull Request

### Commit Messages

Follow conventional commits:

```
type(scope): description

feat(trace): add hot reload support
fix(tui): resolve cursor navigation issue
docs(readme): update installation instructions
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

## Coding Standards

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Run `gofmt` on all code
- Run `golangci-lint run` before committing
- Keep functions focused and small
- Add comments for exported functions

### Testing

- Write tests for new features
- Maintain or improve code coverage
- Use table-driven tests where appropriate

```go
func TestExample(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"basic", "input", "expected"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test logic
        })
    }
}
```

### Documentation

- Update README.md for user-facing changes
- Add godoc comments to exported functions
- Include examples where helpful

## Project Structure

```
agk/
├── cmd/           # CLI commands (cobra)
├── internal/      # Private packages
│   ├── audit/     # Audit/evaluation types
│   └── tui/       # Terminal UI components
├── go.mod
└── README.md
```

## Development Workflow

```bash
# Build and install locally
go install ./...

# Run specific command
go run main.go trace show

# Run tests with coverage
go test -cover ./...

# Lint
golangci-lint run
```

## Questions?

- Open a [Discussion](https://github.com/agenticgokit/agk/discussions)
- Check existing [Issues](https://github.com/agenticgokit/agk/issues)

## License

By contributing, you agree that your contributions will be licensed under the Apache 2.0 License.
