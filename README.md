# AGK - AgenticGoKit CLI

> **Build production-ready agentic AI systems in seconds.**

AGK is the official CLI for **AgenticGoKit**, designed to accelerate your development workflow. From scaffolding simple agents to deploying enterprise-grade multi-agent swarms, AGK handles the boilerplate so you can focus on intelligence.

![Version](https://img.shields.io/badge/version-0.1.0-blue)
![License](https://img.shields.io/badge/license-Apache%202.0-green)
![Status](https://img.shields.io/badge/status-Active%20Development-yellow)

---

## 🚀 Why AGK?

- **⚡ Instant Scaffolding**: Generate complete, compilable projects with one command.
- **🧠 Smart Defaults**: Auto-detects configuration for OpenAI, Anthropic, and Ollama.
- **� Trace Auditor**: Built-in observability to debug agent thoughts and prompts.
- **📦 Production Ready**: Docker support, idiomatic Go code, and established patterns.
- **🌊 Streaming Native**: All templates support real-time token streaming out of the box.

---

## 🏁 Quick Start

### 1. Installation

```bash
# Build from source
cd agk-new
go build -o agk main.go
```

### 2. Create Your First Agent

```bash
# Initialize a new project with the single-agent template
./agk init my-agent --template single-agent --llm openai

# Navigate to the project
cd my-agent

# Install dependencies
go mod tidy
```

### 3. Run It

```bash
# Set your API key
export OPENAI_API_KEY=sk-...

# Run the agent
go run main.go
```

---

## 📦 Templates

Choose the right foundation for your project:

| Template | Complexity | Best For | Description |
|----------|------------|----------|-------------|
| **Quickstart** | ⭐ | Learning | Minimal setup. Single file. Hardcoded config. Perfect for understanding the basics. |
| **Single-Agent** | ⭐⭐ | Prototypes | Adds tools, memory, and environment config. The "standard" starting point. |
| **Multi-Agent** | ⭐⭐⭐ | Workflows | Sequential pipeline of specialized agents (e.g., Researcher → Writer). |
| **Config-Driven** | ⭐⭐⭐⭐ | Enterprise | Factory patterns, TOML config, shared memory. Built for scale. |
| **Advanced** | ⭐⭐⭐⭐⭐ | Production | Full-stack: REST API, WebSocket, Docker, Frontend integration. |

**Example usage:**
```bash
./agk init enterprise-bot --template config-driven --llm anthropic
```

---

## 🔍 Trace Auditor

AGK includes a powerful **Trace Auditor** to help you understand exactly what your agents are thinking.

### 1. Capture Traces
Control data granularity with `AGK_TRACE_LEVEL`:

| Level | Data Captured | Use Case |
|-------|---------------|----------|
| `minimal` | Timing, status | Production monitoring |
| `standard` | + Tokens, latency | General debugging |
| `detailed` | + Prompts, responses, tool args | **Deep evaluation & auditing** |

```bash
# Enable detailed tracing to see prompts and thoughts
$env:AGK_TRACE="true"
$env:AGK_TRACE_LEVEL="detailed"
go run main.go
```

### 2. Analyze Traces

**Interactive Viewer (TUI)**
Browse traces, explore spans, and view content details.
```bash
agk trace view
# Tip: Press 'd' on a span to see the full Prompt & Response content!
```

**Audit Report (JSON)**
Export structured data for automated evaluation pipelines.
```bash
agk trace audit > evaluation_dataset.json
```

**Visual Flowchart (Mermaid)**
Generate a diagram of the agent's execution path.
```bash
agk trace mermaid > trace_flow.md
```

---

## 🛠️ Commands

| Command | Description |
|---------|-------------|
| `init` | Create a new project from a template. |
| `init --list` | Show details of all available templates. |
| `trace list` | List all captured trace runs. |
| `trace show` | Display summary of a specific run. |
| `trace view` | Open the interactive TUI trace explorer. |
| `trace audit` | Analyze a trace for reasoning quality. |
| `trace export` | Export trace data (OTEL, Jaeger, JSON). |

---

## 🗺️ Roadmap

### ✅ Completed
- Template system (Quickstart, Single-Agent)
- Smart LLM Provider detection
- Streaming support
- **Trace Auditor** (Audit & Mermaid commands)
- **Interactive Trace Viewer** (with content inspection)

### 🚧 In Progress
- Multi-Agent & Enterprise templates
- Advanced full-stack template

### 📅 Planned
- Interactive init wizard (`agk init -i`)
- MCP Server management
- Project upgrade tools

---

## 🤝 Contributing

We love contributions! Please read [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

## � License

Apache 2.0 - See [LICENSE](./LICENSE).

---
**Built with ❤️ for the AgenticGoKit community**
