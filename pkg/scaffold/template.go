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
	TemplateQuickstart   TemplateType = "quickstart"
	TemplateSingleAgent  TemplateType = "single-agent"
	TemplateMultiAgent   TemplateType = "multi-agent"
	TemplateConfigDriven TemplateType = "config-driven"
	TemplateAdvanced     TemplateType = "advanced"
	TemplateMCPTools     TemplateType = "mcp-tools"
	TemplateWorkflow     TemplateType = "workflow"
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
		"quickstart":    TemplateQuickstart,
		"single-agent":  TemplateSingleAgent,
		"multi-agent":   TemplateMultiAgent,
		"config-driven": TemplateConfigDriven,
		"advanced":      TemplateAdvanced,
		"mcp-tools":     TemplateMCPTools,
		"workflow":      TemplateWorkflow,
	}

	if tt, ok := validTemplates[templateStr]; ok {
		return tt, nil
	}

	return "", fmt.Errorf("invalid template '%s'. Valid options: quickstart, single-agent, multi-agent, config-driven, advanced, mcp-tools, workflow", templateStr)
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
			Name:        "Single-Agent",
			Description: "Single agent with tools and memory",
			Complexity:  "⭐⭐",
			FileCount:   5,
			Features:    []string{"Agent", "Tools/MCP", "Memory", ".env Config"},
		},
		{
			Name:        "Multi-Agent",
			Description: "Multiple agents with workflow pipeline",
			Complexity:  "⭐⭐⭐",
			FileCount:   8,
			Features:    []string{"Agents", "Workflow", "Sequential Pipeline", ".env Config"},
		},
		{
			Name:        "Config-Driven",
			Description: "Enterprise setup with TOML configuration",
			Complexity:  "⭐⭐⭐⭐",
			FileCount:   12,
			Features:    []string{"Agents", "Workflow", "Factory Pattern", "TOML Config", "Memory"},
		},
		{
			Name:        "Advanced",
			Description: "Full-stack with server, frontend, and Docker",
			Complexity:  "⭐⭐⭐⭐⭐",
			FileCount:   20,
			Features:    []string{"Agents", "Workflow", "Server", "Frontend", "WebSocket", "Docker", "TOML Config"},
		},
		{
			Name:        "MCP-Tools",
			Description: "Agent with MCP server tool integration",
			Complexity:  "⭐⭐",
			FileCount:   3,
			Features:    []string{"Agent", "MCP Tools", "Streaming", "Observability"},
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
