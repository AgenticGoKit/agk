# AGK - AgenticGoKit Command Line Interface

A powerful, production-ready CLI for scaffolding, managing, and deploying agentic AI systems built with **AgenticGoKit**. Generate complete project structures in seconds with multiple template options, from simple quickstart projects to enterprise-grade multi-agent workflows.

![Version](https://img.shields.io/badge/version-0.1.0-blue)
![License](https://img.shields.io/badge/license-Apache%202.0-green)
![Status](https://img.shields.io/badge/status-Active%20Development-yellow)

## What is AGK?

AGK is a CLI tool that accelerates the development of agentic AI systems by providing:

- **📦 Project Scaffolding** - Generate complete, production-ready projects with a single command
- **🎯 Multiple Templates** - Choose from 5 progressive templates based on your project complexity
- **⚡ Smart Defaults** - Automatic LLM configuration detection for OpenAI, Anthropic, and Ollama
- **🔧 Developer-Friendly** - Clean, well-structured generated code with streaming enabled by default
- **🚀 Framework Integration** - Full integration with AgenticGoKit v0.5.1+ ecosystem

## Quick Start

### Installation

```bash
# Build from source
cd agk-new
go build -o agk main.go

# Verify installation
./agk version
```

### Create Your First Project

```bash
# List available templates
./agk init --list

# Generate a quickstart project
./agk init my-agent --template quickstart --llm openai

# Generate a single-agent with tools and memory
./agk init researcher --template single-agent --llm ollama

# Non-interactive with all options
./agk init my-project \
  --template single-agent \
  --llm anthropic \
  --description "AI Research Assistant" \
  --force
```

## Available Templates

### 1. **Quickstart** ⭐
Minimal setup - perfect for learning and experimentation.
- Simple hardcoded configuration
- Single agent setup
- Ideal for: Getting started, prototyping
- **Files:** 2 (main.go, go.mod)

**Use when:** You want the absolute simplest setup to understand how AgenticGoKit works.

```bash
./agk init my-project --template quickstart --llm openai
```

### 2. **Single-Agent** ⭐⭐
Single agent with tools and memory capabilities.
- Full agent configuration with tools support
- Memory management
- Environment-based configuration (.env)
- Streaming enabled by default
- **Files:** 5 (main.go, go.mod, .env, workflow/*, agk.toml)

**Use when:** Building a focused agent system with specific capabilities (research, analysis, content generation).

```bash
./agk init researcher --template single-agent --llm ollama
```

### 3. **Multi-Agent** ⭐⭐⭐
Multiple agents with workflow pipeline.
- Sequential workflow orchestration
- Multiple specialized agents
- Workflow factory pattern
- Advanced configuration
- **Files:** 8 (workflow pipeline with agent coordination)

**Use when:** You need multiple agents working together in a defined sequence (e.g., planning → execution → review).

```bash
./agk init workflow --template multi-agent --llm openai
```

### 4. **Config-Driven** ⭐⭐⭐⭐
Enterprise setup with TOML configuration.
- Factory pattern for agents
- TOML-based configuration management
- Shared memory across agents
- Advanced error handling
- **Files:** 12 (fully configured enterprise setup)

**Use when:** Building scalable systems that need centralized configuration and factory-based agent creation.

```bash
./agk init enterprise-system --template config-driven --llm anthropic
```

### 5. **Advanced** ⭐⭐⭐⭐⭐
Full-stack with server, frontend, and Docker.
- REST API server
- Web frontend integration
- Docker containerization
- WebSocket support for real-time updates
- Complete DevOps setup
- **Files:** 20+ (complete production system)

**Use when:** Building production systems with web interfaces, APIs, and containerization.

```bash
./agk init production-agent --template advanced --llm openai
```

## Supported LLM Providers

### OpenAI
```bash
./agk init my-project --template single-agent --llm openai
# Default model: gpt-4-turbo
```

### Anthropic
```bash
./agk init my-project --template single-agent --llm anthropic
# Default model: claude-3-sonnet-20240229
```

### Ollama (Local)
```bash
./agk init my-project --template single-agent --llm ollama
# Default model: llama3.2
```

## Features

### ✨ Smart Code Generation
- **Streaming by default** - All templates use streaming for real-time response feedback
- **System prompts included** - Customizable AI behavior prompts built-in
- **Error handling** - Production-ready error handling patterns
- **Clean code** - Idiomatic Go with proper error management

### 🛠️ Developer Experience
- **Clear examples** - Generated code includes helpful comments
- **Configuration templates** - .env and TOML configuration examples
- **Modular structure** - Workflow-based architecture for scalability
- **Version pinned** - AgenticGoKit v0.5.1 pinned for stability

### 📋 Project Structure
Each generated project includes:
```
my-project/
├── main.go                 # Entry point with streaming
├── go.mod                  # Go module configuration
├── .env                    # Environment variables template
├── agk.toml               # Project configuration
└── workflow/              # Workflow logic (if applicable)
    ├── workflow.go
    ├── agents.go
    └── factory.go
```

## Commands

### Initialize a Project
```bash
./agk init [project-name] [flags]
```

**Flags:**
- `--template, -t` - Template type (quickstart, single-agent, multi-agent, config-driven, advanced)
- `--llm` - LLM provider (openai, anthropic, ollama)
- `--description` - Project description
- `--output, -o` - Output directory (default: current directory)
- `--force, -f` - Overwrite existing files
- `--interactive, -i` - Enable interactive prompts (coming soon)

### List Templates
```bash
./agk init --list
```

Shows all available templates with complexity, features, and usage examples.

### Get Help
```bash
./agk init --help
./agk --help
```

## Development

### Building
```bash
# Build binary
go build -o agk main.go

# Build with version info
go build -ldflags "-X main.Version=0.1.0" -o agk main.go
```

### Testing
```bash
# Run tests
make test

# Run with coverage
make test-coverage
```

### Project Structure
```
agk-new/
├── cmd/                        # CLI commands
│   ├── init.go                # Init command implementation
│   └── root.go                # Root CLI setup
├── pkg/
│   └── scaffold/              # Project scaffolding
│       ├── template.go        # Template interfaces and types
│       ├── template_registry.go # Template generators
│       ├── template_loader.go  # Embedded template loading
│       └── templates/         # Embedded template files
│           ├── quickstart/
│           ├── single-agent/
│           ├── multi-agent/
│           ├── config-driven/
│           └── advanced/
├── main.go                     # Entry point
├── go.mod                      # Go dependencies
├── go.sum                      # Dependency checksums
├── LICENSE                     # Apache 2.0 License
└── README.md                   # This file
```

## Examples

### Create and Run a Quickstart Project
```bash
# Create project
./agk init hello-agent --template quickstart --llm openai

# Navigate to project
cd hello-agent

# Install dependencies
go mod tidy

# Run
go run main.go
```

### Create a Research Assistant
```bash
# Create project
./agk init researcher --template single-agent --llm anthropic --description "AI Research Assistant"

# Edit SystemPrompt in main.go to customize behavior
# Add tools in workflow/agents.go
# Run the project
cd researcher && go run main.go
```

## Dependencies

- **Go 1.21+** - Required for `embed` package
- **AgenticGoKit v0.5.1** - Core framework
- **Cobra** - CLI framework (automatically included)
- **Fatih/Color** - Terminal colors (automatically included)

## Requirements for Generated Projects

Each generated project requires:
- **Go 1.21 or higher**
- **API Keys** (depending on LLM provider):
  - OpenAI: `OPENAI_API_KEY` environment variable
  - Anthropic: `ANTHROPIC_API_KEY` environment variable
  - Ollama: Local instance running on port 11434 (no API key needed)

## Configuration

### Generated Project Config
Each project includes `agk.toml` for configuration:
```toml
[project]
name = "my-agent"
description = "AI Assistant"

[llm]
provider = "openai"
model = "gpt-4-turbo"
temperature = 0.7
max_tokens = 2000
```

### Environment Variables
Generated projects use `.env` for sensitive data:
```env
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
```

## Status & Roadmap

### ✅ Completed
- Template infrastructure and routing
- Embedded template system
- 2 production templates (Quickstart, Single-Agent)
- Multi-provider LLM detection
- Streaming-enabled generation
- System prompt support
- Template listing command

### 🚧 In Progress
- Template refinement and testing
- Multi-Agent template implementation
- Config-Driven template implementation
- Advanced template implementation

### 📅 Planned
- Interactive project creation (`--interactive` flag)
- Project upgrade commands
- Workflow testing utilities
- Trace visualization tools
- MCP server management
- Template customization options

## License

Apache License 2.0 - See [LICENSE](./LICENSE) file for details

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

## Support

- 📖 [AgenticGoKit Documentation](https://github.com/agenticgokit/agenticgokit)
- 💬 [GitHub Discussions](https://github.com/agenticgokit/agenticgokit/discussions)
- 🐛 [Report Issues](https://github.com/agenticgokit/agenticgokit/issues)

---

**Built with ❤️ for the AgenticGoKit community**
