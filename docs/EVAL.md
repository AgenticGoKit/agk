# AGK Eval - Automated Workflow Testing

The `agk eval` command provides comprehensive automated testing for AI workflows using semantic matching, confidence scoring, and professional reporting.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Test Configuration](#test-configuration)
- [Semantic Matching Strategies](#semantic-matching-strategies)
- [EvalServer Integration](#evalserver-integration)
- [Reports](#reports)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

---

## Overview

The eval framework enables you to:
- **Validate workflow outputs** using semantic understanding (not exact string matching)
- **Score confidence** on a 0.0-1.0 scale for each test
- **Generate professional reports** with visualizations and detailed analysis
- **Integrate with CI/CD** for automated quality gates
- **Debug failures** using trace integration

### Architecture

```
┌─────────────────┐
│  Test Suite     │
│  (YAML)         │
└────────┬────────┘
         │
         ▼
┌─────────────────┐      ┌──────────────────┐
│  AGK Eval       │─────▶│  EvalServer      │
│  Command        │      │  (HTTP Server)   │
└────────┬────────┘      └──────────────────┘
         │                        │
         │                        ▼
         │               ┌──────────────────┐
         │               │  Your Workflow   │
         │               └──────────────────┘
         │
         ▼
┌─────────────────┐      ┌──────────────────┐
│  Semantic       │─────▶│  Embedding or    │
│  Matcher        │      │  LLM Judge       │
└────────┬────────┘      └──────────────────┘
         │
         ▼
┌─────────────────┐
│  Report         │
│  Generator      │
└─────────────────┘
```

---

## Quick Start

### 1. Create Your Workflow

First, ensure your workflow supports EvalServer mode:

```go
// main.go
package main

import (
    "context"
    "os"
    agk "github.com/agenticgokit/agenticgokit/v1beta"
)

func main() {
    if os.Getenv("AGK_EVAL_MODE") == "true" {
        runEvalServer()
        return
    }
    runNormal()
}

func runEvalServer() {
    ctx := context.Background()
    
    // Load your workflow
    workflow, _ := agk.LoadWorkflowFromTOML("config.toml")
    workflow.Initialize(ctx)
    defer workflow.Shutdown(ctx)
    
    // Start EvalServer
    server := agk.NewEvalServer(
        agk.WithEvalWorkflow("myworkflow", workflow),
        agk.WithEvalPort(8787),
    )
    
    server.ListenAndServe()
}

func runNormal() {
    // Your normal workflow execution
}
```

### 2. Create Test Configuration

```yaml
# tests.yaml
name: "My Workflow Tests"
description: "Semantic evaluation of AI outputs"

evalserver:
  url: "http://localhost:8787"
  workflow_name: "myworkflow"
  timeout: "180s"

semantic:
  strategy: "llm-judge"
  threshold: 0.70
  llm:
    provider: "ollama"
    model: "llama3.2"
    temperature: 0.0
    max_tokens: 2000

tests:
  - name: "Test Case 1"
    input: "Your input here"
    expected_output: |
      Description of what you expect the output to contain,
      not an exact string match
```

### 3. Run Tests

```bash
# Terminal 1: Start your workflow in EvalServer mode
AGK_EVAL_MODE=true ./myworkflow

# Terminal 2: Run tests
agk eval tests.yaml --timeout 200

# View report
cat .agk/reports/eval-report-*.md
```

---

## Test Configuration

### Full YAML Specification

```yaml
# Test suite metadata
name: "Suite Name"
description: "What this test suite validates"

# EvalServer connection
evalserver:
  url: "http://localhost:8787"      # Server URL
  workflow_name: "myworkflow"       # Workflow identifier
  timeout: "180s"                   # Max execution time per test

# Semantic matching configuration
semantic:
  strategy: "llm-judge"             # "embedding", "llm-judge", or "hybrid"
  threshold: 0.70                   # Pass threshold (0.0-1.0)
  
  # For embedding strategy
  embedding:
    provider: "ollama"
    model: "nomic-embed-text"
  
  # For llm-judge or hybrid strategy
  llm:
    provider: "ollama"
    model: "llama3.2"
    temperature: 0.0
    max_tokens: 2000

# Test cases
tests:
  - name: "Test Case Name"
    input: "Input to workflow"
    expected_output: |
      Multi-line description of expected output.
      Focus on semantic meaning, not exact wording.
      
  - name: "Another Test"
    input: "Different input"
    expected_output: "Short expected output"
```

### Configuration Fields

#### EvalServer Section

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | Yes | HTTP endpoint of EvalServer |
| `workflow_name` | string | Yes | Workflow identifier (must match server registration) |
| `timeout` | duration | Yes | Max time per test (e.g., "180s", "3m") |

#### Semantic Section

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `strategy` | string | Yes | Matching strategy: `embedding`, `llm-judge`, `hybrid` |
| `threshold` | float | Yes | Pass threshold 0.0-1.0 (typically 0.60-0.80) |
| `embedding` | object | Conditional | Required for `embedding` or `hybrid` |
| `llm` | object | Conditional | Required for `llm-judge` or `hybrid` |

#### Test Case

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique test identifier |
| `input` | string | Yes | Input sent to workflow |
| `expected_output` | string | Yes | Semantic description of expected output |

---

## Semantic Matching Strategies

### 1. Embedding Strategy

Uses vector embeddings to compute similarity between expected and actual outputs.

**When to Use:**
- Fast execution needed (< 1 second per test)
- Checking if outputs cover similar topics/concepts
- High-volume testing (100+ test cases)
- Deterministic results required

**How It Works:**
1. Embeds expected output using `nomic-embed-text`
2. Embeds actual workflow output
3. Computes cosine similarity
4. Passes if similarity ≥ threshold

**Configuration:**
```yaml
semantic:
  strategy: "embedding"
  threshold: 0.70
  embedding:
    provider: "ollama"
    model: "nomic-embed-text"
```

**Pros:**
- ⚡ Very fast (< 1s)
- 🎯 Deterministic
- 📊 Good for semantic similarity

**Cons:**
- 🤔 Less nuanced than LLM judge
- ❌ May miss quality issues
- 📝 Better for content matching than quality

**Example Results:**
```
Test: Generate Article
Expected: "A technical article about AI safety"
Actual: "AI Safety: A Comprehensive Guide..."
Similarity: 0.82 ✓ PASSED
```

---

### 2. LLM-as-Judge Strategy

Uses an LLM to evaluate if actual output matches the expected description.

**When to Use:**
- Quality matters more than speed
- Nuanced evaluation needed (tone, completeness, accuracy)
- Expected outputs are descriptions, not exact text
- Need reasoning behind pass/fail decisions

**How It Works:**
1. Constructs a prompt with expected and actual outputs
2. Asks LLM: "Does actual match expected?"
3. LLM responds with YES/NO and confidence score
4. Provides reasoning for the decision

**Configuration:**
```yaml
semantic:
  strategy: "llm-judge"
  threshold: 0.70
  llm:
    provider: "ollama"
    model: "llama3.2"
    temperature: 0.0        # Use 0 for consistency
    max_tokens: 2000
```

**Custom Judge Prompt (Optional):**
```yaml
semantic:
  strategy: "llm-judge"
  threshold: 0.70
  llm:
    provider: "ollama"
    model: "llama3.2"
  judge_prompt: |
    You are evaluating AI-generated content.
    
    Expected: {expected}
    Actual: {actual}
    
    Does the actual output meet the expectations?
    Respond: YES <confidence> <reasoning> or NO <confidence> <reasoning>
```

**Pros:**
- 🧠 Nuanced understanding
- ✍️ Provides reasoning
- 🎯 Better quality assessment
- 📋 Handles complex criteria

**Cons:**
- 🐌 Slower (5-15s per test)
- 💰 More expensive (if using paid APIs)
- 🎲 Less deterministic
- 🔧 Requires good LLM

**Example Results:**
```
Test: Generate Report
Confidence: 0.90 ✓ PASSED

Reasoning:
"The actual output matches the expected description perfectly. 
It contains a comprehensive technical report with structured 
sections covering AI collaboration, applications, benefits, 
and future directions as specified."
```

---

### 3. Hybrid Strategy

Combines both embedding and LLM judge strategies.

**When to Use:**
- Maximum coverage needed
- Balance speed and quality
- Critical workflows that need double validation

**How It Works:**
1. Runs embedding similarity check
2. If passed, marks as PASSED
3. If embedding fails, runs LLM judge
4. Uses best result from either strategy

**Configuration:**
```yaml
semantic:
  strategy: "hybrid"
  threshold: 0.70
  embedding:
    provider: "ollama"
    model: "nomic-embed-text"
  llm:
    provider: "ollama"
    model: "llama3.2"
```

**Pros:**
- ✅ Highest accuracy
- 🎯 Catches edge cases
- ⚡ Fast when embedding passes

**Cons:**
- 🐌 Slower on failures
- 🔧 More complex configuration
- 💾 More resource intensive

**Strategy Comparison:**

| Factor | Embedding | LLM Judge | Hybrid |
|--------|-----------|-----------|--------|
| Speed | ⚡⚡⚡ | ⚡ | ⚡⚡ |
| Accuracy | ⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ |
| Cost | $ | $$$ | $$ |
| Reasoning | ❌ | ✅ | ✅ |
| Deterministic | ✅ | ⚠️ | ⚠️ |

---

## EvalServer Integration

### What is EvalServer?

EvalServer is an HTTP server mode that wraps your workflow for testing. It provides:
- Standardized HTTP endpoints
- Trace collection
- Timeout handling
- Error reporting

### Implementing EvalServer

```go
package main

import (
    "context"
    "os"
    agk "github.com/agenticgokit/agenticgokit/v1beta"
)

func main() {
    // Check for eval mode
    if os.Getenv("AGK_EVAL_MODE") == "true" {
        runEvalServer()
        return
    }
    runNormal()
}

func runEvalServer() {
    ctx := context.Background()
    
    // Load workflow (TOML, builder, or programmatic)
    workflow, err := agk.LoadWorkflowFromTOML("workflow-config.toml")
    if err != nil {
        log.Fatal(err)
    }
    
    if err := workflow.Initialize(ctx); err != nil {
        log.Fatal(err)
    }
    defer workflow.Shutdown(ctx)
    
    // Create server with options
    server := agk.NewEvalServer(
        agk.WithEvalWorkflow("myworkflow", workflow),
        agk.WithEvalPort(8787),
        agk.WithTraceDir("./eval-traces"),
    )
    
    fmt.Println("EvalServer listening on :8787")
    if err := server.ListenAndServe(); err != nil {
        log.Fatal(err)
    }
}
```

### EvalServer Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithEvalWorkflow(name, workflow)` | Register a workflow | Required |
| `WithEvalPort(port)` | HTTP port | `8787` |
| `WithTraceDir(dir)` | Trace storage directory | `./.agk/eval-traces` |

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| POST | `/invoke` | Invoke default workflow |
| POST | `/invoke/{name}` | Invoke named workflow |
| GET | `/traces/{id}` | Get trace by ID |

### Request Format

```json
{
  "input": "Your workflow input",
  "sessionID": "optional-session-id",
  "options": {
    "timeout": 120
  }
}
```

### Response Format

```json
{
  "output": "Workflow output text",
  "success": true,
  "duration": 45.2,
  "trace_id": "run-20260207-123456-12345678"
}
```

---

## Reports

The eval framework auto-generates professional markdown reports with detailed analysis.

### Report Structure

```markdown
# Test Report: Suite Name

> **Status: PASSED** - 5/6 tests completed successfully

## Summary

| Metric | Value | Progress |
|--------|-------|----------|
| Total Tests | 6 | |
| Passed | 5 | ✓✓✓✓✓ |
| Failed | 1 | ✗ |
| Pass Rate | 83.3% | [████████████████░░░░] |

## Detailed Test Results

### 1. Test Name

**Status:** PASSED | **Duration:** 45.2s
**Confidence Score:** 85%

[Progress bar visualization]

<details>
<summary>View Judge's Reasoning</summary>
...
</details>

<details>
<summary>Expected Output</summary>
...
</details>

<details>
<summary>Actual Output</summary>
...
</details>
```

### Report Location

Reports are saved to:
```
.agk/reports/eval-report-YYYYMMDD-HHMMSS.md
```

### Report Features

- ✅ **Executive Summary**: Quick pass/fail overview
- 📊 **Progress Bars**: Visual representation of success rates
- 📈 **Confidence Scores**: Numerical confidence with bar visualization
- 🔍 **Collapsible Sections**: Reduces clutter, expandable details
- 🔗 **Trace Links**: Direct links to execution traces
- 🎯 **Judge Reasoning**: Explanation for LLM judge decisions
- 🏷️ **AGK Branding**: Tool attribution footer

---

## Best Practices

### Threshold Selection

| Threshold | Use Case |
|-----------|----------|
| 0.90+ | Strict quality gates, production deployments |
| 0.70-0.89 | Standard testing, most use cases |
| 0.60-0.69 | Lenient matching, exploratory testing |
| < 0.60 | Not recommended (too permissive) |

### Writing Good Expected Outputs

**❌ Bad - Too specific:**
```yaml
expected_output: "The capital of France is Paris."
```

**✅ Good - Semantic description:**
```yaml
expected_output: |
  A factually correct statement identifying Paris as 
  the capital city of France
```

**❌ Bad - Exact template:**
```yaml
expected_output: |
  # Title
  ## Section 1
  Content here
  ## Section 2
  More content
```

**✅ Good - Structure description:**
```yaml
expected_output: |
  A well-structured document with:
  - A clear title
  - Multiple sections with headings
  - Professional formatting
  - Comprehensive content
```

### Test Organization

```yaml
# Group related tests
tests:
  # Basic functionality
  - name: "Basic Query"
    input: "simple question"
    expected_output: "direct answer"
  
  # Edge cases
  - name: "Empty Input"
    input: ""
    expected_output: "error message or helpful prompt"
  
  # Complex scenarios
  - name: "Multi-step Workflow"
    input: "complex requirements"
    expected_output: |
      Detailed multi-section output with...
```

### Performance Tips

1. **Use embedding for bulk tests**: Switch to `embedding` strategy for large test suites (50+ tests)
2. **Parallel execution**: Run multiple test suites in parallel
3. **Adjust timeouts**: Set realistic timeouts based on workflow complexity
4. **Cache embeddings**: Ollama automatically caches embeddings

---

## Troubleshooting

### EvalServer Connection Failed

**Symptom:**
```
Error: failed to connect to EvalServer at http://localhost:8787
```

**Solution:**
```bash
# Check if server is running
curl http://localhost:8787/health

# Start the server
AGK_EVAL_MODE=true ./myworkflow

# Verify correct port in tests.yaml
evalserver:
  url: "http://localhost:8787"
```

### Test Timeout

**Symptom:**
```
Error: test timed out after 180s
```

**Solution:**
```yaml
# Increase timeout in YAML
evalserver:
  timeout: "300s"  # 5 minutes

# Or use CLI flag
agk eval tests.yaml --timeout 300
```

### Low Confidence Scores

**Symptom:**
```
All tests failing with confidence ~0.40
```

**Solutions:**
1. **Check expected output**: Make it more semantic, less specific
2. **Lower threshold**: Try 0.60 instead of 0.70
3. **Switch strategy**: Try `llm-judge` if using `embedding`
4. **Verify workflow**: Manually run workflow to check actual output

### LLM Judge Not Available

**Symptom:**
```
Error: failed to initialize LLM judge: model not found
```

**Solution:**
```bash
# Install required model
ollama pull llama3.2

# Verify model name in tests.yaml
semantic:
  llm:
    model: "llama3.2"  # Must match exact model name
```

### Embedding Model Missing

**Symptom:**
```
Error: embedding model not available
```

**Solution:**
```bash
# Install embedding model
ollama pull nomic-embed-text

# Verify configuration
semantic:
  embedding:
    provider: "ollama"
    model: "nomic-embed-text"
```

---

## Advanced Usage

### Custom Judge Prompts

Override the default judge prompt for specialized evaluation:

```yaml
semantic:
  strategy: "llm-judge"
  judge_prompt: |
    You are a technical documentation reviewer.
    
    Expected Requirements:
    {expected}
    
    Actual Content:
    {actual}
    
    Evaluate if the content meets professional documentation standards.
    Consider: accuracy, clarity, completeness, formatting.
    
    Respond: YES <0.0-1.0> <reasoning> or NO <0.0-1.0> <reasoning>
```

### CI/CD Integration

```yaml
# .github/workflows/test.yml
name: AI Workflow Tests

on: [push, pull_request]

jobs:
  eval:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Install Ollama
        run: curl -fsSL https://ollama.com/install.sh | sh
      
      - name: Pull Models
        run: |
          ollama pull llama3.2
          ollama pull nomic-embed-text
      
      - name: Start EvalServer
        run: |
          cd myworkflow
          AGK_EVAL_MODE=true ./myworkflow &
          sleep 10
      
      - name: Run Tests
        run: |
          cd agk
          ./agk eval ../tests/semantic-tests.yaml --timeout 300
      
      - name: Upload Report
        uses: actions/upload-artifact@v3
        with:
          name: eval-report
          path: .agk/reports/
```

### Multiple Workflows

Test multiple workflows in one suite:

```yaml
# Start server with multiple workflows
server := agk.NewEvalServer(
    agk.WithEvalWorkflow("workflow1", wf1),
    agk.WithEvalWorkflow("workflow2", wf2),
)
```

```yaml
# Test different workflows
tests:
  - name: "Test Workflow 1"
    workflow_name: "workflow1"
    input: "..."
    
  - name: "Test Workflow 2"
    workflow_name: "workflow2"
    input: "..."
```

---

## Examples

### Example 1: Documentation Generator

```yaml
name: "Docs Generator Tests"
description: "Validate technical documentation quality"

evalserver:
  url: "http://localhost:8787"
  workflow_name: "docs"
  timeout: "120s"

semantic:
  strategy: "llm-judge"
  threshold: 0.75
  llm:
    provider: "ollama"
    model: "llama3.2"

tests:
  - name: "API Documentation"
    input: "Document the /api/users endpoint"
    expected_output: |
      Professional API documentation including:
      - Endpoint description
      - HTTP method and path
      - Request parameters
      - Response format
      - Example requests/responses
      - Error codes
```

### Example 2: Code Review

```yaml
name: "Code Review Tests"
description: "Automated code review quality"

evalserver:
  url: "http://localhost:8787"
  workflow_name: "reviewer"
  timeout: "90s"

semantic:
  strategy: "hybrid"
  threshold: 0.80
  embedding:
    provider: "ollama"
    model: "nomic-embed-text"
  llm:
    provider: "ollama"
    model: "llama3.2"

tests:
  - name: "Security Review"
    input: "Review this authentication code"
    expected_output: |
      A thorough security review identifying:
      - Potential vulnerabilities
      - Best practice violations
      - Specific recommendations
      - Risk severity levels
```

---

## See Also

- [Trace Documentation](trace.md) - Debugging with traces
- [AGK CLI Reference](../README.md) - Full command reference
- [Workflow Examples](../../test-eval-demo/) - Complete examples
