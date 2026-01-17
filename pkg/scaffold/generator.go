package scaffold

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Generator handles file and directory generation
type Generator struct{}

// NewGenerator creates a new generator
func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateStructure creates the directory structure for a new project
func (g *Generator) GenerateStructure(ctx context.Context, projectPath string) error {
	dirs := []string{
		"workflow",
		"agents",
		"internal/config",
		"internal/utils",
		"pkg",
		"test/fixtures",
		"test/mocks",
		"docs",
		".agk",
	}

	for _, dir := range dirs {
		path := filepath.Join(projectPath, dir)
		if err := os.MkdirAll(path, 0750); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", path, err)
		}
	}

	return nil
}

// GenerateMainGo creates the main.go entry point for the project
func (g *Generator) GenerateMainGo(projectPath, projectName string) error {
	packageName := strings.ToLower(projectName)
	packageName = strings.ReplaceAll(packageName, "-", "_")
	_ = packageName // Use packageName to avoid unused variable

	content := `package main

import (
	"context"
	"log"

	"github.com/agenticgokit/agenticgokit/v1beta/core"
	"github.com/example/` + projectName + `/workflow"
)

func main() {
	ctx := context.Background()

	// Create workflow factory
	factory := workflow.NewFactory("openai", "gpt-4")

	// Create workflow
	wf, err := factory.CreateWorkflow()
	if err != nil {
		log.Fatalf("Failed to create workflow: %v", err)
	}

	// Execute workflow
	result, err := wf.Execute(ctx, "Hello, AgenticGoKit!")
	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	log.Printf("Result: %s", result)
}
`

	filePath := filepath.Join(projectPath, "main.go")
	return os.WriteFile(filePath, []byte(content), 0600)
}

// GenerateGoMod creates a go.mod file for the project
func (g *Generator) GenerateGoMod(projectPath, projectName string) error {
	// Convert project name to module path
	modulePath := "github.com/" + projectName

	content := `module ` + modulePath + `

go 1.21

require (
	github.com/agenticgokit/agenticgokit/v1beta v0.9.0
	github.com/spf13/cobra v1.8.0
	github.com/spf13/viper v1.18.0
		github.com/rs/zerolog v1.33.0
	github.com/fatih/color v1.16.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/google/uuid v1.5.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/log15 v2.3.2-0.20221150144038-414c3106be10+incompatible // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pelletier/go-toml/v2 v2.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20231127185646-65229373498e // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
`

	filePath := filepath.Join(projectPath, "go.mod")
	return os.WriteFile(filePath, []byte(content), 0600)
}

// GenerateTestFixtures creates test files and fixtures
func (g *Generator) GenerateTestFixtures(projectPath string) error {
	testMainContent := `package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExample(t *testing.T) {
	assert.True(t, true)
}
`

	testPath := filepath.Join(projectPath, "test/fixtures")
	mainTestPath := filepath.Join(projectPath, "main_test.go")

	if err := os.WriteFile(mainTestPath, []byte(testMainContent), 0600); err != nil {
		return fmt.Errorf("failed to create main_test.go: %w", err)
	}

	_ = testPath // Reserved for future use
	return nil
}
