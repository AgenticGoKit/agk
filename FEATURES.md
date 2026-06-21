# AGK — Feature Ideas & Roadmap

Proposed improvements to the AGK CLI, aimed at the developer experience and the
AI-agent building experience. Each item is grounded in a concrete observation from the
codebase (file references included) so it's actionable, not aspirational.

Status legend: 🔴 not started · 🟡 partially built · 🟢 quick win

---

## Priority recommendation

In order of leverage:

1. **`agk run` / `agk dev`** — closes the scaffold→run→observe loop (nothing else has this leverage).
2. **Eval auto-serve + `expect.trace`** — turns the test pillar from "wire up two processes" into one command; the trace-assertion plumbing already exists.
3. **`agk doctor`** — kills the most common first-run failures, very cheap to build.
4. **Interactive `init` wizard** — already promised in the roadmap, TUI toolkit already imported.

Then fold in the correctness fixes (Section A) as each area is touched.

---

## A. Fix / finish what's already half-built

Low-risk, high-trust changes where the code already gestures at a feature but doesn't
deliver. These remove "the docs lied to me" friction.

### A1. Implement the `init -i` interactive flag 🟡
- **Evidence:** `initInteractive` is parsed in `cmd/init.go` and passed into
  `GenerateOptions.Interactive`, but no generator ever reads it.
- **Action:** Build a Bubble Tea wizard (deps `bubbletea`/`bubbles`/`lipgloss` already
  present). Flow: template → provider → model → features. Satisfies the "Interactive Init
  Wizard" roadmap item.

### A2. Wire up (or remove) `agk.toml` generation 🟡
- **Evidence:** `internal/config/generator.go` builds a full project config, and
  `cmd/init.go` help text promises "Project configuration (agk.toml)", but the built-in
  generators in `pkg/scaffold/template_registry.go` only write `main.go` + `go.mod`.
- **Action:** Either call the generator during `init` or drop the claim. A real project
  `agk.toml` (provider/model/memory defaults) would also let `run`/`eval` stop depending
  on env + flags.

### A3. `template remove` by name 🟢
- **Evidence:** Explicit `TODO` in `cmd/template.go` — only removes by exact source string.
- **Action:** Add name→source lookup in `registry.CacheManager`.

### A4. Fix eval doc/schema mismatch 🟢
- **Evidence:** README shows `evalserver:` / `expected_output:`; the parser
  (`internal/eval/types.go`, `parser.go`) expects `target:` / `expect:`.
- **Action:** Align README to the actual schema (or add a compatibility shim). Currently a
  confusing first-run failure.

### A5. Real cost estimation 🟢
- **Evidence:** `cmd/trace.go` hardcodes `estimatedCost := tokens * 0.00001`.
- **Action:** Per-model pricing table so `trace view` / `trace list` cost numbers are
  trustworthy.

### A6. Implement `expect.trace` validation 🟡 (see B2 — biggest unlock)
- **Evidence:** `TraceExpectation` (tool_calls, llm_calls, execution_path, min/max steps)
  is fully typed in `internal/eval/types.go` but there is a
  `// TODO: Validate trace expectations` at `internal/eval/runner.go:193`.
- **Action:** Validate against the captured trace (see B2).

---

## B. Net-new features that close loop gaps

### B1. `agk run` / `agk dev` — the missing center of the loop 🟢 ✅ SHIPPED
- **Gap:** The CLI *scaffolds* and *observes* but never *runs*. `printNextSteps` in
  `cmd/init.go` just tells the user to `go run main.go`.
- **Delivered (`cmd/run.go`):**
  - `agk run [path]` wraps `go run .`, auto-sets `AGK_TRACE=true` +
    `AGK_TRACE_EXPORTER=file`, and inherits stdio;
  - on exit, prints a compact trace summary (duration / spans / LLM calls / tokens / cost)
    + a `→ agk trace view <run-id>` hint;
  - `--watch` / `-w` re-runs on `.go` changes (debounced);
  - `--no-trace` and `--trace-level minimal|standard|detailed` flags.
- **Follow-ups:** a dedicated `agk dev` alias; making `agk trace` path-aware so summaries for
  `agk run <subdir>` link correctly from any CWD.

### B2. Eval: auto-serve + behavioral assertions 🔴/🟡 ⭐
- **Gap:** `agk eval` is an HTTP client (`internal/eval/http_target.go`) that assumes the
  user is *separately* running a `v1beta.EvalServer` in another terminal.
- **Proposal A — auto-serve:** `agk eval --serve ./...` (or `agk eval init`) builds/launches
  the user's eval server, runs tests, and tears it down. One command instead of two
  processes. `agk eval init` could also scaffold a starter `tests.yaml` + EvalServer wrapper.
- **Proposal B — behavioral assertions:** Implement `expect.trace` (types already exist) to
  assert "the `search` tool was called", "≤ 3 LLM calls", "path was research→summarize".
  Turns eval from output-matching into **behavioral** testing — what agent devs actually
  need. Validate against the existing `audit.TraceObject` event model
  (`internal/audit/types.go`).
- **Proposal C — more target types:** in-process / CLI target so simple agents don't need
  HTTP at all (parser currently rejects anything but `type: http`).

### B3. `agk doctor` — preflight diagnostics 🔴
- **Gap:** A large class of first-run failures is environmental (`OPENAI_API_KEY` unset,
  Ollama not running on `:11434`, model not pulled, registry unreachable). These surface as
  cryptic runtime errors deep in the agent.
- **Proposal:** `agk doctor` checks: Go version, provider API keys, Ollama reachability,
  registry index reachability, `.agk/` health. Cheap, big friction reducer.

### B4. More templates tied to framework strengths 🔴
- **Gap:** Only `quickstart` + `workflow` ship (`pkg/scaffold/templates/`). The core
  framework sells memory/RAG, MCP tools, multimodal, and parallel/DAG/loop workflows —
  **none of which have a template.**
- **Proposal:** Add self-contained templates: RAG agent, MCP-tool agent, parallel/DAG
  workflow, chat-REPL agent. Each is small and high-value.

### B5. `agk trace diff <a> <b>` 🔴
- **Gap:** The observe pillar is the most mature; improvements are incremental.
- **Proposal:** Run-to-run diff (latency / tokens / cost / execution path) — directly
  answers "did my prompt change help?". A `trace watch` live-tail is also close, since
  `internal/tui` `NewTraceViewerWithPath` already supports hot-reload.

### B6. MCP & RAG management 🔴 (longer-term, on-brand)
- **Gap:** The framework differentiates on "batteries-included" MCP + chromem vector store,
  but the CLI offers nothing here yet.
- **Proposal:** `agk mcp add/list` (register MCP servers + scaffold tool wiring) and
  `agk rag ingest <docs>` / `agk knowledge` (manage the embedded vector store). Bigger lifts,
  but strategically aligned with the framework's identity and the existing roadmap.

---

## Mapping to the five-pillar vision

| Pillar | Existing | Proposed additions |
|--------|----------|--------------------|
| **Create** | `init`, `template` | Interactive wizard (A1), `agk.toml` (A2), more templates (B4) |
| **Run** *(new)* | — | `agk run`/`dev` (B1), `agk doctor` (B3) |
| **Test** | `eval` | Auto-serve (B2-A), `expect.trace` (A6/B2-B), more targets (B2-C) |
| **Observe** | `trace` | Cost table (A5), `trace diff`/`watch` (B5) |
| **Distribute** | *(planned)* | template `pack`/`push` |
| **Deploy** | *(planned)* | `agk deploy` (cloud/k8s/edge), MCP/RAG mgmt (B6) |
</content>
