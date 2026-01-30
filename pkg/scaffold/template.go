// Package scaffold provides project scaffolding and template generation functionality.
package scaffold

import (
	"context"
	"fmt"
)

// TemplateType represents the type of template to generate
type TemplateType string

// Template type constants
const (
	TemplateQuickstart TemplateType = "quickstart"
	TemplateWorkflow   TemplateType = "workflow"
)

// TemplateMetadata contains information about a template
type TemplateMetadata struct {
	Name        string
	Description string
	Complexity  string
	FileCount   int
	Features    []string
}

// TemplateGenerator defines the interface for template generators
type TemplateGenerator interface {
	// Generate creates the project structure and files for the template
	Generate(ctx context.Context, opts GenerateOptions) error

	// GetMetadata returns metadata about the template
	GetMetadata() TemplateMetadata
}

// ValidateTemplate validates and returns a TemplateType from a string
func ValidateTemplate(templateStr string) (TemplateType, error) {
	validTemplates := map[string]TemplateType{
		"quickstart": TemplateQuickstart,
		"workflow":   TemplateWorkflow,
	}

	if tt, ok := validTemplates[templateStr]; ok {
		return tt, nil
	}

	return "", fmt.Errorf("invalid template '%s'. Valid options: quickstart, workflow", templateStr)
}

// GetAllTemplates returns all available templates
func GetAllTemplates() []TemplateMetadata {
	return []TemplateMetadata{
		{
			Name:        "Quickstart",
			Description: "Minimal setup - perfect for learning",
			Complexity:  "⭐",
			FileCount:   2,
			Features:    []string{"Agent", "Hardcoded Config"},
		},
		{
			Name:        "Workflow",
			Description: "Multi-step streaming workflow pipeline",
			Complexity:  "⭐⭐⭐",
			FileCount:   3,
			Features:    []string{"Workflow", "Multi-Agent", "Streaming", "Step Tracking"},
		},
	}
}
