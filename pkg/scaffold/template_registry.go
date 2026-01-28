package scaffold

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DefaultGpt4Turbo = "gpt-4-turbo"

	ProviderAnthropic = "anthropic"
	ProviderOllama    = "ollama"
	ProviderOpenAI    = "openai"
	ProviderAzure     = "azure"
)

// GetTemplateGenerator returns the appropriate template generator for the given template type
func GetTemplateGenerator(templateType TemplateType) (TemplateGenerator, error) {
	switch templateType {
	case TemplateQuickstart:
		return NewQuickstartGenerator(), nil

	case TemplateSingleAgent:
		return NewSingleAgentGenerator(), nil

	case TemplateMultiAgent:
		return NewMultiAgentGenerator(), nil

	case TemplateConfigDriven:
		return NewConfigDrivenGenerator(), nil

	case TemplateAdvanced:
		return NewAdvancedGenerator(), nil

	case TemplateMCPTools:
		return NewMCPToolsGenerator(), nil

	case TemplateWorkflow:
		return NewWorkflowGenerator(), nil

	default:
		return nil, fmt.Errorf("unknown template type: %s", templateType)
	}
}

// TemplateRegistry provides information about all available templates
type TemplateRegistry struct{}

// NewTemplateRegistry creates a new template registry
func NewTemplateRegistry() *TemplateRegistry {
	return &TemplateRegistry{}
}

// ListTemplates returns all available templates
func (r *TemplateRegistry) ListTemplates() []TemplateMetadata {
	return GetAllTemplates()
}

// GetTemplate returns metadata for a specific template
func (r *TemplateRegistry) GetTemplate(templateType TemplateType) (TemplateMetadata, error) {
	for _, tm := range GetAllTemplates() {
		if tm.Name == string(templateType) {
			return tm, nil
		}
	}
	return TemplateMetadata{}, fmt.Errorf("template not found: %s", templateType)
}

// ===== QUICKSTART GENERATOR =====

// QuickstartGenerator implements TemplateGenerator for quickstart template
type QuickstartGenerator struct{}

func NewQuickstartGenerator() *QuickstartGenerator {
	return &QuickstartGenerator{}
}

func (g *QuickstartGenerator) GetMetadata() TemplateMetadata {
	return TemplateMetadata{
		Name:        "Quickstart",
		Description: "Minimal setup - perfect for learning",
		Complexity:  "⭐",
		FileCount:   2,
		Features:    []string{"Agent", "Hardcoded Config"},
	}
}

func (g *QuickstartGenerator) Generate(ctx context.Context, opts GenerateOptions) error {
	// Create project directory
	if err := os.MkdirAll(opts.ProjectPath, 0750); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Prepare template data
	data := TemplateData{
		ProjectName: opts.ProjectName,
		LLMModel:    "gpt-4o-mini", // Default for quickstart
		LLMProvider: opts.LLMProvider,
		Description: opts.Description,
		AgentType:   opts.AgentType,
	}

	// Render go.mod from template
	goModContent, err := RenderTemplate("templates/quickstart/go.mod.tmpl", data)
	if err != nil {
		return err
	}

	goModPath := filepath.Join(opts.ProjectPath, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0600); err != nil {
		return fmt.Errorf("failed to create go.mod: %w", err)
	}

	// Render main.go from template
	mainGoContent, err := RenderTemplate("templates/quickstart/main.go.tmpl", data)
	if err != nil {
		return err
	}

	mainGoPath := filepath.Join(opts.ProjectPath, "main.go")
	if err := os.WriteFile(mainGoPath, []byte(mainGoContent), 0600); err != nil {
		return fmt.Errorf("failed to create main.go: %w", err)
	}

	return nil
}

// ===== GENERATORS =====

// SingleAgentGenerator generates a single-agent template
type SingleAgentGenerator struct{}

func NewSingleAgentGenerator() *SingleAgentGenerator {
	return &SingleAgentGenerator{}
}

func (g *SingleAgentGenerator) GetMetadata() TemplateMetadata {
	return TemplateMetadata{
		Name:        "Single-Agent",
		Description: "Single agent with tools and memory",
		Complexity:  "⭐⭐",
		FileCount:   5,
		Features:    []string{"Agent", "Tools/MCP", "Memory", ".env Config"},
	}
}

func (g *SingleAgentGenerator) Generate(ctx context.Context, opts GenerateOptions) error {
	// Create project directory
	if err := os.MkdirAll(opts.ProjectPath, 0750); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Determine LLM model based on provider
	llmModel := DefaultGpt4Turbo
	if opts.LLMProvider == ProviderAnthropic {
		llmModel = "claude-3-sonnet-20240229"
	} else if opts.LLMProvider == ProviderOllama {
		llmModel = "llama3.2"
	}

	// Prepare template data
	data := TemplateData{
		ProjectName: opts.ProjectName,
		LLMModel:    llmModel,
		LLMProvider: opts.LLMProvider,
		Description: opts.Description,
		AgentType:   opts.AgentType,
	}

	// Files to generate: go.mod, main.go, .env
	files := map[string]string{
		"go.mod":  "templates/single-agent/go.mod.tmpl",
		"main.go": "templates/single-agent/main.go.tmpl",
		".env":    "templates/single-agent/.env.tmpl",
	}

	for fileName, templatePath := range files {
		content, err := RenderTemplate(templatePath, data)
		if err != nil {
			return err
		}

		filePath := filepath.Join(opts.ProjectPath, fileName)
		if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to create %s: %w", fileName, err)
		}
	}

	return nil
}

// MultiAgentGenerator generates a multi-agent template
type MultiAgentGenerator struct{}

func NewMultiAgentGenerator() *MultiAgentGenerator {
	return &MultiAgentGenerator{}
}

func (g *MultiAgentGenerator) GetMetadata() TemplateMetadata {
	return TemplateMetadata{
		Name:        "Multi-Agent",
		Description: "Multiple agents with workflow pipeline",
		Complexity:  "⭐⭐⭐",
		FileCount:   8,
		Features:    []string{"Agents", "Workflow", "Sequential Pipeline", ".env Config"},
	}
}

func (g *MultiAgentGenerator) Generate(ctx context.Context, opts GenerateOptions) error {
	// TODO: Phase 2 - Implement multi-agent generator
	return fmt.Errorf("multi-agent template not yet implemented")
}

// ConfigDrivenGenerator generates a config-driven template
type ConfigDrivenGenerator struct{}

func NewConfigDrivenGenerator() *ConfigDrivenGenerator {
	return &ConfigDrivenGenerator{}
}

func (g *ConfigDrivenGenerator) GetMetadata() TemplateMetadata {
	return TemplateMetadata{
		Name:        "Config-Driven",
		Description: "Enterprise setup with TOML configuration",
		Complexity:  "⭐⭐⭐⭐",
		FileCount:   12,
		Features:    []string{"Agents", "Workflow", "Factory Pattern", "TOML Config", "Memory"},
	}
}

func (g *ConfigDrivenGenerator) Generate(ctx context.Context, opts GenerateOptions) error {
	// TODO: Phase 2 - Implement config-driven generator
	return fmt.Errorf("config-driven template not yet implemented")
}

// AdvancedGenerator generates an advanced template
type AdvancedGenerator struct{}

func NewAdvancedGenerator() *AdvancedGenerator {
	return &AdvancedGenerator{}
}

func (g *AdvancedGenerator) GetMetadata() TemplateMetadata {
	return TemplateMetadata{
		Name:        "Advanced",
		Description: "Full-stack with server, frontend, and Docker",
		Complexity:  "⭐⭐⭐⭐⭐",
		FileCount:   20,
		Features:    []string{"Agents", "Workflow", "Server", "Frontend", "WebSocket", "Docker", "TOML Config"},
	}
}

func (g *AdvancedGenerator) Generate(ctx context.Context, opts GenerateOptions) error {
	// TODO: Phase 2 - Implement advanced generator
	return fmt.Errorf("advanced template not yet implemented")
}

// MCPToolsGenerator generates an MCP-enabled agent template
type MCPToolsGenerator struct{}

func NewMCPToolsGenerator() *MCPToolsGenerator {
	return &MCPToolsGenerator{}
}

func (g *MCPToolsGenerator) GetMetadata() TemplateMetadata {
	return TemplateMetadata{
		Name:        "MCP-Tools",
		Description: "Agent with MCP server tool integration",
		Complexity:  "⭐⭐",
		FileCount:   3,
		Features:    []string{"Agent", "MCP Tools", "Streaming", "Observability"},
	}
}

func generateTemplateFiles(opts GenerateOptions, files map[string]string) error {
	if err := os.MkdirAll(opts.ProjectPath, 0750); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	data := TemplateData{
		ProjectName: opts.ProjectName,
		LLMModel:    getLLMModel(opts.LLMProvider),
		LLMProvider: opts.LLMProvider,
		Description: opts.Description,
		AgentType:   opts.AgentType,
		APIKeyEnv:   getAPIKeyEnv(opts.LLMProvider),
	}

	for fileName, templatePath := range files {
		content, err := RenderTemplate(templatePath, data)
		if err != nil {
			return err
		}

		filePath := filepath.Join(opts.ProjectPath, fileName)
		if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to create %s: %w", fileName, err)
		}
	}

	return nil
}

func (g *MCPToolsGenerator) Generate(ctx context.Context, opts GenerateOptions) error {
	files := map[string]string{
		"go.mod":    "templates/mcp-tools/go.mod.tmpl",
		"main.go":   "templates/mcp-tools/main.go.tmpl",
		"README.md": "templates/mcp-tools/README.md.tmpl",
	}
	return generateTemplateFiles(opts, files)
}

// WorkflowGenerator generates a streaming workflow template
type WorkflowGenerator struct{}

func NewWorkflowGenerator() *WorkflowGenerator {
	return &WorkflowGenerator{}
}

func (g *WorkflowGenerator) GetMetadata() TemplateMetadata {
	return TemplateMetadata{
		Name:        "Workflow",
		Description: "Multi-step streaming workflow pipeline",
		Complexity:  "⭐⭐⭐",
		FileCount:   3,
		Features:    []string{"Workflow", "Multi-Agent", "Streaming", "Step Tracking"},
	}
}

func (g *WorkflowGenerator) Generate(ctx context.Context, opts GenerateOptions) error {
	files := map[string]string{
		"go.mod":    "templates/workflow/go.mod.tmpl",
		"main.go":   "templates/workflow/main.go.tmpl",
		"README.md": "templates/workflow/README.md.tmpl",
	}
	return generateTemplateFiles(opts, files)
}

// Helper to get default model for provider
func getLLMModel(provider string) string {
	switch provider {
	case ProviderAnthropic:
		return "claude-sonnet-4-20250514"
	case ProviderOllama:
		return "llama3.2"
	case ProviderOpenAI:
		return "gpt-4-turbo"
	default:
		return "gpt-4-turbo"
	}
}

// Helper to get the API key environment variable name for provider
func getAPIKeyEnv(provider string) string {
	switch provider {
	case ProviderAnthropic:
		return "ANTHROPIC_API_KEY"
	case ProviderOllama:
		return "OLLAMA_HOST" // Ollama doesn't need API key, but uses host
	case ProviderOpenAI:
		return "OPENAI_API_KEY"
	case ProviderAzure:
		return "AZURE_OPENAI_API_KEY"
	default:
		return "OPENAI_API_KEY"
	}
}
