package scaffold

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DefaultOpenAIModel = "gpt-4o"

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

	case TemplateWorkflow:
		return NewWorkflowGenerator(), nil

	default:
		// Attempt to fallback to registry/external generator if not built-in?
		// But GetTemplateGenerator is for built-ins usually.
		return nil, fmt.Errorf("unknown built-in template type: %s", templateType)
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
		LLMModel:    getLLMModel(opts.LLMProvider), // Dynamic model selection
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

// Helper to get default model for provider
func getLLMModel(provider string) string {
	switch provider {
	case ProviderAnthropic:
		return "claude-sonnet-4-20250514"
	case ProviderOllama:
		return "llama3.2"
	case ProviderOpenAI:
		return "gpt-4o"
	default:
		return "gpt-4o"
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
