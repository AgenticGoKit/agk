# AGK Developer CLI

A comprehensive, production-ready developer CLI for the AgenticGoKit framework.

## Quick Start

```bash
# Build the binary
make build

# Run version
./agk version

# Run tests
make test
```

## Features

- **Project Scaffolding**: Generate new AgenticGoKit projects in seconds
- **Workflow Execution**: Run and test workflows from the command line
- **MCP Integration**: Manage and debug Model Context Protocol servers
- **Trace Visualization**: Debug and monitor agent execution
- **Memory Management**: Manage agent memory and knowledge bases

## Project Status

🚧 **Currently in Development** - Sprint 1: Foundation

This is a fresh implementation of AGK following the design specification in `DESIGN_SPEC.md`.

## Development

### Setup
```bash
# Install dependencies (already in go.mod)
go mod download

# Run tests
make test

# Run linter
make lint

# Build
make build
```

### Documentation
- [DESIGN_SPEC.md](./DESIGN_SPEC.md) - Complete architecture and design
- [IMPLEMENTATION_GUIDE.md](./IMPLEMENTATION_GUIDE.md) - Detailed implementation tasks
- [QUICKSTART.md](./QUICKSTART.md) - Developer quick start guide

### Architecture

```
agk-new/
├── cmd/                    # CLI commands
├── pkg/                    # Public packages
│   ├── scaffold/          # Project generation
│   ├── executor/          # Workflow execution
│   ├── mcp/               # MCP management
│   ├── trace/             # Trace visualization
│   └── ui/                # Terminal UI
├── internal/               # Private packages
│   ├── config/            # Config management
│   ├── v1beta/            # v1beta integration
│   ├── templates/         # Code generation
│   └── utils/             # Utilities
├── test/                  # Test fixtures
└── templates/             # Project templates
```

## License

Apache 2.0 - See LICENSE file for details

## Contributing

See CONTRIBUTING.md for guidelines
