// Package scaffold provides project scaffolding and template generation functionality.
package scaffold

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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
	templateEngine  *Engine
	configGenerator *config.Generator
	generator       *Generator
}

// NewService creates a new scaffold service
func NewService(logger *zerolog.Logger) *Service {
	return &Service{
		logger:          logger,
		templateEngine:  NewEngine(),
		configGenerator: config.NewGenerator(),
		generator:       NewGenerator(),
	}
}

// GenerateProject generates a new project with the given options
func (s *Service) GenerateProject(ctx context.Context, opts GenerateOptions) error {
	if s.logger != nil {
		s.logger.Info().Str("project", opts.ProjectName).Msg("starting project generation")
	}

	// Collect user input if interactive
	projectConfig := &config.ProjectConfig{
		Name:        opts.ProjectName,
		Description: opts.Description,
		Template:    opts.Template,
		LLMProvider: opts.LLMProvider,
		AgentType:   opts.AgentType,
	}

	if opts.Interactive {
		var err error
		projectConfig, err = s.collectUserInput(ctx, projectConfig)
		if err != nil {
			return fmt.Errorf("failed to collect user input: %w", err)
		}
	}

	// Create project directory
	fmt.Println(color.CyanString("  ✓ Creating directory structure"))
	if err := os.MkdirAll(opts.ProjectPath, 0750); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Generate directory structure
	fmt.Println(color.CyanString("  ✓ Generating project structure"))
	if err := s.generator.GenerateStructure(ctx, opts.ProjectPath); err != nil {
		return fmt.Errorf("failed to generate project structure: %w", err)
	}

	// Generate configuration file
	fmt.Println(color.CyanString("  ✓ Generating agk.toml configuration"))
	configPath := filepath.Join(opts.ProjectPath, "agk.toml")
	if err := s.configGenerator.GenerateConfig(projectConfig, configPath); err != nil {
		return fmt.Errorf("failed to generate configuration: %w", err)
	}

	// Generate workflow files
	fmt.Println(color.CyanString("  ✓ Creating workflow definitions"))
	if err := s.templateEngine.RenderWorkflow(opts.ProjectPath, projectConfig); err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}

	// Generate main.go
	fmt.Println(color.CyanString("  ✓ Creating main.go entry point"))
	if err := s.generator.GenerateMainGo(opts.ProjectPath, opts.ProjectName); err != nil {
		return fmt.Errorf("failed to generate main.go: %w", err)
	}

	// Generate README
	fmt.Println(color.CyanString("  ✓ Generating README.md"))
	if err := s.templateEngine.RenderREADME(opts.ProjectPath, projectConfig); err != nil {
		return fmt.Errorf("failed to generate README: %w", err)
	}

	// Generate go.mod
	fmt.Println(color.CyanString("  ✓ Creating go.mod"))
	if err := s.generator.GenerateGoMod(opts.ProjectPath, opts.ProjectName); err != nil {
		return fmt.Errorf("failed to generate go.mod: %w", err)
	}

	// Generate test fixtures
	fmt.Println(color.CyanString("  ✓ Creating test fixtures"))
	if err := s.generator.GenerateTestFixtures(opts.ProjectPath); err != nil {
		return fmt.Errorf("failed to generate test fixtures: %w", err)
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
