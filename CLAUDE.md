# CLAUDE.md

Guidance for working in this repository.

## What this is

**AGK** is the official command-line developer toolchain for **AgenticGoKit**, a Go
framework for building multi-agent AI systems. AGK is *not* the framework itself —
it is the CLI that manages the lifecycle of agent projects built *with* the framework.

It is one part of a three-repo ecosystem:

| Part | Repo | Role |
|------|------|------|
| **Core framework** | `agenticgokit/agenticgokit` (sibling dir `../agenticgokit`) | The library agents are built with (`v1beta` builder API, workflows, memory, RAG, tools, observability). |
| **CLI tooling (this repo)** | `agenticgokit/agk` | Scaffold, evaluate, and trace agent projects. |
| **Template registry** | `agk-templates` | Remote templates that `agk init` can pull. |

Typical user flow: **design** with the framework → **scaffold** with `agk init` →
**test** with `agk eval` → **observe** with `agk trace`.

- Module: `github.com/agenticgokit/agk`, Go **1.24.1**
- Depends on the core framework: `github.com/agenticgokit/agenticgokit v0.5.5`
  (uses its `observability` package directly; generated projects import its `v1beta` API).
- Entry point: `main.go` → `cmd.Execute()` (Cobra root command `agk`).

## Product vision (five pillars)

The README frames AGK around a lifecycle. Two pillars are built, three are roadmap:

1. **Create** ✅ — `init` scaffolding + template registry
2. **Test** ✅ — `eval` semantic evaluation framework
3. **Observe** ✅ — `trace` observability (TUI, mermaid, export)
4. **Distribute** 🔜 — template `pack`/`push` (planned)
5. **Deploy** 🔜 — `agk deploy` to cloud/k8s/edge (planned)

When asked about "possibilities" or new features, the planned items are: multi-agent
templates, template distribution (`pack`/`push`), cloud deploy engine, interactive
init wizard (`agk init -i`), MCP server management, and RAG/knowledge-base management.

## Commands (all under `cmd/`)

| Command | File | What it does |
|---------|------|--------------|
| `agk init <name>` | `init.go` | Scaffold a project from a template (`--template`, `--llm`, `--output`, `--force`, `--list`). |
| `agk run [path]` | `run.go` | `go run .` with tracing on by default; prints a trace summary on exit. Flags: `--watch`, `--no-trace`, `--trace-level`. |
| `agk template list/add/remove` | `template.go` | Manage the local template cache (pull from GitHub/local/registry). |
| `agk eval <file.yaml>` | `eval.go` | Run YAML-defined eval tests against a running EvalServer over HTTP. |
| `agk trace [list/show/view/export/audit/mermaid]` | `trace.go` | Inspect traces stored in `.agk/runs/`. Bare `agk trace` launches the TUI explorer. |
| `agk version` | `version.go` | Build/version info (injected via ldflags). |

Global flags (in `cmd/root.go`): `--config`, `--verbose`, `--debug`, `--trace`,
`--trace-exporter` (console|otlp|file), `--trace-endpoint`, `--trace-sample`,
`--store-prompts`. Config is loaded by Viper from `$HOME/.agk.toml` with env prefix `AGK_`.

## Package layout

```
cmd/                 Cobra commands (root, init, template, eval, trace, version)
pkg/scaffold/        Project generation
  template.go            TemplateType, TemplateMetadata, TemplateGenerator interface
  template_registry.go   Built-in generators: Quickstart, Workflow + provider/model helpers
  external_generator.go  Renders cached registry templates (text/template + Sprig, ".tmpl" stripped)
  service.go             Higher-level Service wrapper
  templates/             Embedded built-in templates (quickstart/, workflow/) as *.tmpl
pkg/registry/        Template fetching/caching/resolution
  resolver.go            Resolves "github.com/...", "./local", "@version", or registry name
  fetcher.go             GitFetcher / LocalFetcher
  cache.go               CacheManager (local template cache)
  manifest.go            agk-template.toml schema (TemplateManifest/TemplateInfo/Variable)
  index.go               Fetches registry index.json (DefaultRegistryURL → agk-templates/registry)
internal/eval/       Evaluation framework
  parser.go              Parses + validates YAML test suites
  types.go               TestSuite/Target/Test/Expectation/SemanticConfig (canonical schema)
  runner.go              Executes suites against an HTTPTarget
  http_target.go         Talks to EvalServer: POST /invoke, GET /health
  matcher.go             MatcherFactory: exact/contains/regex/semantic
  embedding_matcher.go   Semantic strategy: embedding cosine similarity
  llm_judge_matcher.go   Semantic strategy: LLM-as-judge
  hybrid_matcher.go      Semantic strategy: hybrid (both)
  reporter.go            Output: console/json/junit/markdown
internal/audit/      Trace → reasoning analysis
  collector.go           Reads .agk/runs/<id> spans → TraceObject of typed events
  types.go               EventType (thought/tool_call/observation/llm_call/decision)
  mermaid.go             Mermaid flowchart generation
internal/tui/        Bubble Tea TUIs: trace_viewer, span_tree, styles
internal/config/     agk.toml generator (ProjectConfig → TOML)
internal/utils/      zerolog logging, filesystem, errors (has the only *_test.go files)
```

## Built-in templates

Two are compiled in (see `pkg/scaffold/templates/`):

- **quickstart** (⭐, 2 files) — single `main.go` with a hardcoded agent via
  `v1beta.NewBuilder(...).WithLLM(...).Build()` + streaming.
- **workflow** (⭐⭐⭐, 3 files) — sequential multi-agent pipeline
  (researcher → summarizer → formatter) via `NewSequentialWorkflow` + step streaming.

Generated projects import `github.com/agenticgokit/agenticgokit/v1beta` and a provider
plugin `plugins/llm/<provider>`. Provider→default-model and provider→API-key-env mappings
live in `template_registry.go` (`getLLMModel`, `getAPIKeyEnv`). Supported `--llm` values:
`openai` (gpt-4o), `anthropic` (claude-sonnet-4), `ollama` (llama3.2), `azure`.

External/registry templates are rendered by `external_generator.go`, driven by an
`agk-template.toml` manifest (`pkg/registry/manifest.go`).

## Key conventions & filesystem layout

- **Traces** are written to `.agk/runs/<run-id>/` when `AGK_TRACE=true`:
  `trace.jsonl` (OTel spans), `events.jsonl`, `manifest.json`.
  Run IDs are `run-<unixnano>` (see `generateRunID` in `cmd/root.go`).
- **Eval reports** auto-save to `.agk/reports/eval-report-<timestamp>.md`.
- `AGK_TRACE_LEVEL` controls capture granularity: `minimal` | `standard` | `detailed`
  (use `detailed` to capture prompts/responses/tool args for `trace audit`).
- Observability is OpenTelemetry-based; the file exporter produces the JSONL that the
  `trace` and `audit` packages parse back.

## How `agk eval` actually works (important)

`agk eval` does **not** run the agent in-process. It is an HTTP client. The flow is:

1. The user's agent project runs a `v1beta.EvalServer` (see `../agenticgokit/v1beta/eval_server.go`),
   exposing `POST /invoke` and `GET /health`.
2. `agk eval tests.yaml` health-checks the target, then POSTs each test `input` to `/invoke`
   and matches the returned `output` against the expectation.

⚠️ **Doc vs. code mismatch:** the README's eval YAML example uses keys like `evalserver:`,
`workflow_name:`, and `expected_output:`. The **actual parser** (`internal/eval/types.go` +
`parser.go`) expects:

```yaml
name: "Suite name"            # required
target:                       # required
  type: http                  # only "http" is supported
  url: http://localhost:8787
semantic:                     # optional global config for "semantic" expectations
  strategy: llm-judge         # llm-judge | embedding | hybrid
  threshold: 0.7
  llm: { provider: ollama, model: llama3.2 }
tests:
  - name: "..."               # required
    input: "..."              # required
    expect:
      type: semantic          # exact | contains | regex | semantic
      value: "..."            # value | values | pattern depending on type
```

Treat `internal/eval/types.go` as the source of truth for the schema, not the README.

## Build, test, lint

```bash
make build            # go build -o agk main.go
make test             # go test -v -race ./...
make test-coverage    # coverage.txt + coverage.html
make lint             # golangci-lint run ./...
make fmt              # gofmt -s + goimports
make install          # go install with version ldflags
```

- `make install`/release inject `Version`/`GitCommit`/`BuildDate` into the `cmd` package
  via `-ldflags -X github.com/agenticgokit/agk/cmd.Version=...`.
- Linting is strict (`.golangci.yml`): includes `gosec`, `gocyclo` (min-complexity 35),
  `dupl`, `goconst`, `stylecheck`, `errcheck`, etc. Match existing style and keep new
  functions under the complexity threshold.
- `make test-integration` references `./test/integration/...`, which **does not exist yet** —
  there is no `test/` directory. Real test coverage today is only in `internal/utils/`.
- CI/release configured in `.github/workflows/` and `.goreleaser.yml`.

## Working with the sibling core framework

`../agenticgokit` is the framework this CLI is built around. Reach for it when you need to
understand:

- the `v1beta` builder/workflow/streaming API that generated templates use
  (`../agenticgokit/v1beta/builder.go`, `workflow.go`, `streaming.go`);
- the `EvalServer` contract that `agk eval` targets (`v1beta/eval_server.go`,
  `eval_types.go`, `eval_handlers.go`);
- the `observability` package this repo imports directly for tracer setup
  (`SetupTracer`, `WithRunID`, `WithLogger`, `GetTracer`).

Note the core framework is mid-migration: `v1beta` (formerly `vnext`) is the recommended
API; legacy `core`/`core/vnext` will be removed at v1.0. New scaffold templates should
target `v1beta`.

## Conventions for changes

- This is a Cobra CLI: each command is its own file in `cmd/`, registered via `init()` →
  `rootCmd.AddCommand(...)`. Follow that pattern for new commands.
- User-facing output uses `fatih/color`; structured logs use `rs/zerolog` (`cmd.GetLogger()`).
- Keep `internal/` for implementation detail and `pkg/` for reusable scaffold/registry
  logic (current split).
- Built-in templates are embedded; after editing files under
  `pkg/scaffold/templates/`, rebuild to pick them up.
</content>
</invoke>
