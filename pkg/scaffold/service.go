// Package scaffold provides project scaffolding and template generation functionality.
package scaffold

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/rs/zerolog"

	"github.com/agenticgokit/agk/internal/config"
)

// GenerateOptions contains options for project generation
type GenerateOptions struct {
	ProjectName string
	ProjectPath string
	Template    string
	Interactive bool
	Force       bool
	Description string
	LLMProvider string
	AgentType   string
}

// Service handles project scaffolding and generation
type Service struct {
	logger          *zerolog.Logger
	configGenerator *config.Generator
}

// NewService creates a new scaffold service
func NewService(logger *zerolog.Logger) *Service {
	return &Service{
		logger:          logger,
		configGenerator: config.NewGenerator(),
	}
}

// GenerateProject generates a new project with the given options
func (s *Service) GenerateProject(ctx context.Context, opts GenerateOptions) error {
	if s.logger != nil {
		s.logger.Info().Str("project", opts.ProjectName).Msg("starting project generation")
	}

	// Resolve template type
	var templateType TemplateType
	switch opts.Template {
	case "quickstart":
		templateType = TemplateQuickstart
	case "single-agent":
		templateType = TemplateSingleAgent
	case "multi-agent":
		templateType = TemplateMultiAgent
	case "mcp-tools":
		templateType = TemplateMCPTools
	case "workflow":
		templateType = TemplateWorkflow
	default:
		// Default to single-agent if not specified or unknown
		if opts.Template == "" {
			templateType = TemplateSingleAgent
		} else {
			// Try to match string to type, otherwise error
			templateType = TemplateType(opts.Template)
		}
	}

	// Get generator for template
	generator, err := GetTemplateGenerator(templateType)
	if err != nil {
		return fmt.Errorf("failed to get template generator: %w", err)
	}

	// Execute generation
	fmt.Println(color.CyanString("  ✓ Generating %s project...", templateType))
	if err := generator.Generate(ctx, opts); err != nil {
		return fmt.Errorf("project generation failed: %w", err)
	}

	if s.logger != nil {
		s.logger.Info().Str("project", opts.ProjectName).Str("path", opts.ProjectPath).Msg("project generation completed successfully")
	}
	return nil
}

// collectUserInput gathers configuration from the user
func (s *Service) collectUserInput(ctx context.Context, cfg *config.ProjectConfig) (*config.ProjectConfig, error) {
	// For now, return the provided config
	// TODO: Implement interactive prompts using survey/v2
	return cfg, nil
}
