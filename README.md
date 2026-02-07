# AGK - AgenticGoKit CLI

> **The Unified Toolchain for Agentic AI Systems**

AGK is the official CLI for **AgenticGoKit**, designed to manage the entire lifecycle of intelligent agents. From scaffolding new projects to distributing templates and observing production workflows, AGK provides a standardized interface for building the next generation of AI software.

![Version](https://img.shields.io/badge/version-0.1.0-blue)
![License](https://img.shields.io/badge/license-Apache%202.0-green)
![Status](https://img.shields.io/badge/status-Active%20Development-yellow)

---

## Vision: The Complete Lifecycle

AGK aims to streamline the developer experience across five key pillars:

1.  **Create**: Scaffold powerful agents instantly using a rich registry of templates.
2.  **Test**: Validate workflows with semantic matching and automated evaluation.
3.  **Observe**: Gain deep observability into your agent's reasoning, prompts, and performance.
4.  **Distribute**: (Planned) Share your agent architectures and workflows with the community or your team.
5.  **Deploy**: (Planned) Seamlessly ship agents to cloud platforms, Kubernetes, or edge devices.

---

## Quick Start

### 1. Installation

```bash
# Build from source
cd agk
go build -o agk main.go
```

### 2. Create Your First Agent

```bash
# Initialize a new project with the quickstart template
./agk init my-agent --template quickstart --llm openai

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

## Templates & Registry

AGK features a powerful template system that lets you use both built-in and community-created templates. Explore the [Official Template Registry](https://github.com/agk-templates).

### Use a Template
```bash
./agk init enterprise-bot --template workflow --llm anthropic
```

### Manage Templates
Bring in templates from GitHub, local folders, or other sources.

```bash
# List all available templates (built-in + cached)
agk template list

# Add a template from a remote source
agk template add github.com/username/my-template

# Remove a cached template
agk template remove my-template
```

> **Want to build your own?** Check out the [Creating Templates Guide](docs/creating-templates.md).

### Built-in Templates

| Template | Best For | Description |
|----------|----------|-------------|
| **Quickstart** | Learning | Minimal setup. Single file. Hardcoded config. Perfect for understanding the basics. |
| **Workflow** | Pipelines | Multi-step workflow (e.g. Sequential, Parallel) structure. |

Run `agk init --list` to see all available templates including those from the registry.

**Example usage:**
```bash
./agk init enterprise-bot --template workflow --llm anthropic
```

---

## 🧪 Eval - Automated Testing

AGK provides a comprehensive **evaluation framework** for testing AI workflows with semantic matching, confidence scoring, and professional reports.

### Features
- **Semantic Matching**: Embedding similarity, LLM-as-judge, or hybrid strategies
- **Confidence Scoring**: Quantify how well outputs match expectations (0.0 - 1.0)
- **Professional Reports**: Auto-generated markdown with collapsible sections and visualizations
- **EvalServer Integration**: HTTP server mode for automated testing
- **Multiple Strategies**: Choose the right evaluation approach for your use case

### Quick Example

```yaml
# semantic-tests.yaml
name: "My Workflow Tests"
description: "Evaluate AI workflow outputs"

evalserver:
  url: "http://localhost:8787"
  workflow_name: "story"
  timeout: "180s"

semantic:
  strategy: "llm-judge"  # or "embedding" or "hybrid"
  threshold: 0.70
  llm:
    provider: "ollama"
    model: "llama3.2"

tests:
  - name: "Generate Report Test"
    input: "artificial intelligence"
    expected_output: |
      A comprehensive technical report with structured sections
```

```bash
# Run evaluations
agk eval semantic-tests.yaml --timeout 200

# View report
cat .agk/reports/eval-report-*.md
```

**Learn more**: See [Eval Documentation](docs/eval.md) for detailed guides on strategies, configuration, and best practices.

---

## 🔍 Trace - Observability

AGK includes a powerful **Trace system** to help you understand exactly what your agents are thinking.

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

**List & Show**
Quick access to trace summaries.
```bash
agk trace list
agk trace show <trace-id>
```

**Visual Flowchart (Mermaid)**
Generate a diagram of the agent's execution path.
```bash
agk trace mermaid > trace_flow.md
```

**Learn more**: See [Trace Documentation](docs/trace.md) for advanced usage and debugging workflows.

---

## 🛠️ Commands

| Command | Description |
|---------|-------------|
| `init` | Create a new project from a template. |
| `init --list` | Show details of all available templates. |
| `eval` | Run automated tests against workflows with semantic matching. |
| `trace list` | List all captured trace runs. |
| `trace show` | Display summary of a specific run. |
| `trace view` | Open the interactive TUI trace explorer. |
| `trace mermaid` | Generate Mermaid flowchart of trace execution. |

---

## Roadmap

### Completed
- **Template Registry System** (`list`, `add`, `remove`)
- **Smart Scaffolding** (Quickstart, Workflow bases)
- **Eval Framework** (Semantic matching, LLM-as-judge, professional reports)
- **Trace System** (Interactive TUI, Mermaid export, detailed spans)
- **Streaming Support** (Native across all templates)

### In Progress
- **Multi-Agent Templates**
- **Advanced Full-Stack Templates**

### Planned
- **Template Distribution** (`pack`, `push`)
- **Cloud Deployment Engine** (`agk deploy`)
- **Workflow Visualization** (Interactive graph editor)
- **Interactive Init Wizard** (`agk init -i`)
- **MCP Server Management**
- **RAG & Knowledge Base Management**

---

## 🤝 Contributing

We love contributions! Please read [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

## � License

Apache 2.0 - See [LICENSE](./LICENSE).

---
**Built with ❤️ for the AgenticGoKit community**
