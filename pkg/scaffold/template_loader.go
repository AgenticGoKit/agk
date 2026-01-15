package scaffold

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

//go:embed templates/*/*.tmpl
var templateFS embed.FS

// TemplateData holds data for template rendering
type TemplateData struct {
	ProjectName string
	LLMModel    string
	LLMProvider string
	Description string
	AgentType   string
}

// RenderTemplate renders a template file with the provided data
func RenderTemplate(templatePath string, data TemplateData) (string, error) {
	// Read template file
	content, err := templateFS.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	// Parse template
	tmpl, err := template.New("template").Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", templatePath, err)
	}

	return buf.String(), nil
}
