package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agenticgokit/agk/internal/config"
)

// Engine handles template rendering
type Engine struct{}

// NewEngine creates a new template engine
func NewEngine() *Engine {
	return &Engine{}
}

// RenderWorkflow generates the main workflow Go files
func (e *Engine) RenderWorkflow(projectPath string, cfg *config.ProjectConfig) error {
	workflowDir := filepath.Join(projectPath, "workflow")

	// Generate workflow.go
	workflowContent := e.generateWorkflowContent(cfg)
	if err := os.WriteFile(filepath.Join(workflowDir, "workflow.go"), []byte(workflowContent), 0644); err != nil {
		return fmt.Errorf("failed to write workflow.go: %w", err)
	}

	// Generate agents.go
	agentsContent := e.generateAgentsContent(cfg)
	if err := os.WriteFile(filepath.Join(workflowDir, "agents.go"), []byte(agentsContent), 0644); err != nil {
		return fmt.Errorf("failed to write agents.go: %w", err)
	}

	// Generate factory.go
	factoryContent := e.generateFactoryContent(cfg)
	if err := os.WriteFile(filepath.Join(workflowDir, "factory.go"), []byte(factoryContent), 0644); err != nil {
		return fmt.Errorf("failed to write factory.go: %w", err)
	}

	return nil
}

// RenderREADME generates the README.md file
func (e *Engine) RenderREADME(projectPath string, cfg *config.ProjectConfig) error {
	readmePath := filepath.Join(projectPath, "README.md")

	content := e.generateREADMEContent(cfg)

	if err := os.WriteFile(readmePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write README: %w", err)
	}

	return nil
}

func (e *Engine) generateWorkflowContent(cfg *config.ProjectConfig) string {
	packageName := strings.ToLower(cfg.Name)
	packageName = strings.ReplaceAll(packageName, "-", "_")

	return `package workflow

import (
	"context"
	"fmt"

	"github.com/agenticgokit/agenticgokit/v1beta/core"
)

// ` + strings.Title(packageName) + `Workflow represents the main workflow
type ` + strings.Title(packageName) + `Workflow struct {
	agents map[string]core.Agent
}

// New` + strings.Title(packageName) + `Workflow creates a new workflow instance
func New` + strings.Title(packageName) + `Workflow() (*` + strings.Title(packageName) + `Workflow, error) {
	// TODO: Initialize agents
	agents := make(map[string]core.Agent)

	return &` + strings.Title(packageName) + `Workflow{
		agents: agents,
	}, nil
}

// Execute runs the workflow
func (w *` + strings.Title(packageName) + `Workflow) Execute(ctx context.Context, input string) (string, error) {
	// TODO: Implement workflow logic
	return fmt.Sprintf("Processing: %s", input), nil
}
`
}

func (e *Engine) generateAgentsContent(cfg *config.ProjectConfig) string {
	return `package workflow

import (
	"github.com/agenticgokit/agenticgokit/v1beta/core"
)

// CreateAgents initializes all agents for the workflow
func CreateAgents(llmProvider, model string) (map[string]core.Agent, error) {
	agents := make(map[string]core.Agent)

	// TODO: Create agents based on your workflow requirements
	// Example:
	// mainAgent, err := core.NewAgent(core.AgentConfig{
	//     Name:        "main-agent",
	//     LLMProvider: llmProvider,
	//     Model:       model,
	// })
	// if err != nil {
	//     return nil, err
	// }
	// agents["main"] = mainAgent

	return agents, nil
}
`
}

func (e *Engine) generateFactoryContent(cfg *config.ProjectConfig) string {
	return `package workflow

import (
	"fmt"

	"github.com/agenticgokit/agenticgokit/v1beta/core"
)

// Factory handles workflow component creation
type Factory struct {
	llmProvider string
	model       string
}

// NewFactory creates a new workflow factory
func NewFactory(llmProvider, model string) *Factory {
	return &Factory{
		llmProvider: llmProvider,
		model:       model,
	}
}

// CreateAgent creates an agent with the given configuration
func (f *Factory) CreateAgent(name string, agentType string) (core.Agent, error) {
	// TODO: Implement agent creation based on type
	return nil, fmt.Errorf("agent type %s not implemented", agentType)
}

// CreateWorkflow creates a new workflow
func (f *Factory) CreateWorkflow() (*` + strings.Title(strings.ReplaceAll(cfg.Name, "-", "_")) + `Workflow, error) {
	return New` + strings.Title(strings.ReplaceAll(cfg.Name, "-", "_")) + `Workflow()
}
`
}

func (e *Engine) generateREADMEContent(cfg *config.ProjectConfig) string {
	content := `# ` + cfg.Name + `

` + cfg.Description + `

## Getting Started

### Prerequisites

- Go 1.21 or later
- AgenticGoKit CLI
- ` + cfg.LLMProvider + ` API key

### Setup

1. Install dependencies:
   ` + "`" + `bash
   go mod tidy
   ` + "`" + `

2. Configure your environment:
   ` + "`" + `bash
   export OPENAI_API_KEY=your-key-here
   ` + "`" + `

3. Run the project:
   ` + "`" + `bash
   agk workflow run --workflow workflow/main.yaml
   ` + "`" + `

## Project Structure

` + "`" + `
.
├── agk.toml              # Project configuration
├── go.mod                # Go module definition
├── README.md             # This file
├── workflow/             # Workflow definitions
│   └── main.yaml         # Main workflow
├── agents/               # Agent implementations
├── internal/             # Internal packages
├── pkg/                  # Public packages
├── test/                 # Tests and fixtures
└── docs/                 # Documentation
` + "`" + `

## Configuration

Edit ` + "`" + `agk.toml` + "`" + ` to configure:
- LLM provider and model
- Agent types and count
- Workflow execution options
- Server settings

## Development

### Running Locally

` + "`" + `bash
agk serve --port 8080
` + "`" + `

### Running Tests

` + "`" + `bash
go test ./...
` + "`" + `

## Resources

- [AgenticGoKit Documentation](https://github.com/agenticgokit/agenticgokit)
- [Workflow Examples](https://github.com/agenticgokit/agenticgokit/tree/main/examples)
- [API Reference](https://pkg.go.dev/github.com/agenticgokit/agenticgokit)

## License

MIT License - See LICENSE file for details
`
	return content
}
