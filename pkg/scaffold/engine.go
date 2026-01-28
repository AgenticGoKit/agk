// Package scaffold provides project scaffolding and template generation functionality.
// This file contains the template Engine for rendering workflow files.
package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/agenticgokit/agk/internal/config"
)

// capitalize converts the first letter to uppercase
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// Engine handles template rendering for scaffolding
type Engine struct{}

// NewEngine creates a new template engine
func NewEngine() *Engine {
	return &Engine{}
}

// RenderWorkflow generates the main workflow Go files
func (e *Engine) RenderWorkflow(projectPath string, cfg *config.ProjectConfig) error {
	workflowDir := filepath.Join(projectPath, "workflow")

	// Prepare template data
	packageName := strings.ToLower(cfg.Name)
	packageName = strings.ReplaceAll(packageName, "-", "_")
	workflowName := capitalize(packageName)

	data := TemplateData{
		ProjectName:  cfg.Name,
		WorkflowName: workflowName,
		Description:  cfg.Description,
		LLMProvider:  cfg.LLMProvider,
	}

	// Generate workflow.go
	workflowContent, err := RenderTemplate("templates/workflow/workflow.go.tmpl", data)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "workflow.go"), []byte(workflowContent), 0600); err != nil {
		return err
	}

	// Generate agents.go
	agentsContent, err := RenderTemplate("templates/workflow/agents.go.tmpl", data)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "agents.go"), []byte(agentsContent), 0600); err != nil {
		return err
	}

	// Generate factory.go
	factoryContent, err := RenderTemplate("templates/workflow/factory.go.tmpl", data)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "factory.go"), []byte(factoryContent), 0600); err != nil {
		return err
	}

	return nil
}

// RenderREADME generates the README.md file
func (e *Engine) RenderREADME(projectPath string, cfg *config.ProjectConfig) error {
	readmePath := filepath.Join(projectPath, "README.md")

	data := TemplateData{
		ProjectName: cfg.Name,
		Description: cfg.Description,
		LLMProvider: cfg.LLMProvider,
	}

	content, err := RenderTemplate("templates/workflow/README.md.tmpl", data)
	if err != nil {
		return err
	}

	if err := os.WriteFile(readmePath, []byte(content), 0600); err != nil {
		return err
	}

	return nil
}
