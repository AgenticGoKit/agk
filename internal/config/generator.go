package config

import (
	"fmt"
	"os"
	"strings"
)

// ProjectConfig holds the configuration for a project
type ProjectConfig struct {
	Name        string
	Description string
	Template    string
	LLMProvider string
	AgentType   string
}

// Generator generates configuration files
type Generator struct{}

// NewGenerator creates a new config generator
func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateConfig generates the agk.toml configuration file
func (g *Generator) GenerateConfig(cfg *ProjectConfig, outputPath string) error {
	// Set defaults if not provided
	if cfg.Template == "" {
		cfg.Template = "simple-agent"
	}
	if cfg.LLMProvider == "" {
		cfg.LLMProvider = "openai"
	}
	if cfg.AgentType == "" {
		cfg.AgentType = "single"
	}

	// Convert project name to valid Go package name
	packageName := strings.ToLower(cfg.Name)
	packageName = strings.ReplaceAll(packageName, "-", "_")

	content := g.generateConfigContent(cfg, packageName)

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (g *Generator) generateConfigContent(cfg *ProjectConfig, packageName string) string {
	description := cfg.Description
	if description == "" {
		description = fmt.Sprintf("AgenticGoKit project: %s", cfg.Name)
	}

	return fmt.Sprintf(`# AGK Project Configuration
# Learn more at: https://github.com/agenticgokit/agenticgokit

[project]
name = "%s"
description = "%s"
version = "0.1.0"
authors = ["Your Name <your.email@example.com>"]

[build]
output_dir = "./build"
templates_dir = "./templates"

[llm]
provider = "%s"
model = "gpt-4"
api_key = "${OPENAI_API_KEY}"
timeout = "30s"

[agents]
type = "%s"
max_agents = 5
memory_type = "in-memory"

[workflow]
type = "sequential"
default_workflow = "workflow/main.yaml"

[server]
port = 8080
host = "localhost"
debug = false

[logging]
level = "info"
format = "json"
output = "stdout"

[mcp]
enabled = true
auto_discover = true

`, cfg.Name, description, cfg.LLMProvider, cfg.AgentType)
}
