package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/agenticgokit/agk/pkg/scaffold"
)

var (
	// Init command flags
	initTemplate      string
	initOutputDir     string
	initInteractive   bool
	initForce         bool
	initLLMProvider   string
	initAgentType     string
	initDescription   string
	initListTemplates bool
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new AgenticGoKit project",
	Long: `Initialize a new AgenticGoKit project with scaffolding.

The init command creates a complete project structure for building agentic
AI systems with AgenticGoKit, including:

- Project configuration (agk.toml)
- Workflow definitions
- Agent implementations
- Frontend/TUI setup (optional)
- Test fixtures and examples

Examples:
  # Initialize project interactively
  agk init my-project

  # Initialize with specific template
  agk init my-project --template simple-agent

  # Non-interactive initialization
  agk init my-project --template simple-agent --llm openai --agent-type single --force

  # Initialize in specific directory
  agk init my-project --output ./projects

  # List available templates
  agk init --list`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Allow zero args only when listing templates
		if initListTemplates {
			return nil
		}
		return cobra.ExactArgs(1)(cmd, args)
	},
	RunE: runInitCommand,
}

// runInitCommand executes the init command
func runInitCommand(cmd *cobra.Command, args []string) error {
	// Handle --list flag
	if initListTemplates {
		return listTemplates()
	}

	projectName := args[0]

	// Validate project name
	if err := validateProjectName(projectName); err != nil {
		color.Red("✗ Invalid project name: %v", err)
		return err
	}

	projectPath := filepath.Join(initOutputDir, projectName)

	// Check if path already exists
	if _, err := os.Stat(projectPath); err == nil && !initForce {
		color.Red("✗ Directory already exists: %s", projectPath)
		color.Yellow("Use --force to overwrite")
		return fmt.Errorf("project directory already exists")
	}

	// Validate and get template type
	templateType, err := scaffold.ValidateTemplate(initTemplate)
	if err != nil {
		color.Red("✗ %v", err)
		return err
	}

	// Get template generator
	generator, err := scaffold.GetTemplateGenerator(templateType)
	if err != nil {
		color.Red("✗ Failed to get template generator: %v", err)
		return err
	}

	// Prepare generation options
	opts := scaffold.GenerateOptions{
		ProjectName: projectName,
		ProjectPath: projectPath,
		Template:    initTemplate,
		Interactive: initInteractive,
		Force:       initForce,
		Description: initDescription,
		LLMProvider: initLLMProvider,
		AgentType:   initAgentType,
	}

	// Print header with template info
	metadata := generator.GetMetadata()
	color.Cyan("\n📦 Creating new AgenticGoKit project: %s\n", projectName)
	color.Cyan("   Template: %s (%s) - %s\n", metadata.Name, metadata.Complexity, metadata.Description)
	color.Cyan("   Files: %d | Features: %v\n", metadata.FileCount, metadata.Features)

	// Generate project using the template generator
	if err := generator.Generate(cmd.Context(), opts); err != nil {
		color.Red("✗ Project generation failed: %v", err)
		logger.Error("project generation failed", zap.Error(err))
		return err
	}

	// Print success message
	color.Green("\n✅ Project initialized successfully!\n")

	// Print next steps
	printNextSteps(projectName, projectPath)

	return nil
}

// listTemplates prints all available templates with their metadata
func listTemplates() error {
	color.Cyan("\n📋 Available AgenticGoKit Templates\n")
	color.Cyan("═══════════════════════════════════\n\n")

	templates := scaffold.GetAllTemplates()
	for i, tmpl := range templates {
		// Template name and complexity
		color.Green("%d. %s %s\n", i+1, tmpl.Name, tmpl.Complexity)

		// Description
		fmt.Printf("   %s\n", color.YellowString(tmpl.Description))

		// Features
		if len(tmpl.Features) > 0 {
			fmt.Printf("   Features: %v\n", color.CyanString("%v", tmpl.Features))
		}

		// File count
		fmt.Printf("   Files: %s\n", color.MagentaString("%d", tmpl.FileCount))

		// Usage example
		templateID := ""
		switch tmpl.Name {
		case "Quickstart":
			templateID = "quickstart"
		case "Single-Agent":
			templateID = "single-agent"
		case "Multi-Agent":
			templateID = "multi-agent"
		case "Config-Driven":
			templateID = "config-driven"
		case "Advanced":
			templateID = "advanced"
		}
		fmt.Printf("   Usage: %s\n", color.HiBlackString("agk init my-project --template %s", templateID))

		if i < len(templates)-1 {
			fmt.Println()
		}
	}

	fmt.Println()
	return nil
}

// validateProjectName validates the project name format
func validateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	// Check for invalid characters
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
			return fmt.Errorf("project name can only contain alphanumeric characters, hyphens, and underscores")
		}
	}

	return nil
}

// printNextSteps prints the next steps after project initialization
func printNextSteps(projectName, projectPath string) {
	relPath, _ := filepath.Rel(".", projectPath)

	fmt.Println(color.BlueString("📖 Next Steps:"))
	fmt.Printf("  1. %s\n", color.CyanString("cd %s", relPath))
	fmt.Printf("  2. %s\n", color.CyanString("go mod tidy"))
	fmt.Printf("  3. %s\n", color.CyanString("export OPENAI_API_KEY=your-key-here  # Set your LLM API key"))
	fmt.Printf("  4. %s\n", color.CyanString("go run main.go                        # Run the project"))

	fmt.Println()
	fmt.Println(color.BlueString("📚 Project Structure:"))
	fmt.Printf("  • %s\n", color.CyanString("main.go                    # Entry point"))
	fmt.Printf("  • %s\n", color.CyanString("workflow/                  # Workflow logic"))
	fmt.Printf("    - %s\n", color.CyanString("workflow.go                # Main workflow definition"))
	fmt.Printf("    - %s\n", color.CyanString("factory.go                 # Agent/workflow factory"))
	fmt.Printf("    - %s\n", color.CyanString("agents.go                  # Agent creation helpers"))
	fmt.Printf("  • %s\n", color.CyanString("agk.toml                   # Project configuration"))
	fmt.Printf("  • %s\n", color.CyanString("go.mod                     # Go module definition"))

	fmt.Println()
	fmt.Println(color.BlueString("💡 Development Tips:"))
	fmt.Printf("  • Edit %s to configure LLM providers and agents\n", color.CyanString("agk.toml"))
	fmt.Printf("  • Implement agents in %s\n", color.CyanString("workflow/agents.go"))
	fmt.Printf("  • Define workflow logic in %s\n", color.CyanString("workflow/workflow.go"))
	fmt.Printf("  • Use the framework: https://github.com/agenticgokit/agenticgokit\n")
	fmt.Println()
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Define flags
	initCmd.Flags().BoolVar(&initListTemplates, "list", false, "List available templates")
	initCmd.Flags().StringVarP(&initTemplate, "template", "t", "quickstart",
		"Template type: quickstart, single-agent, multi-agent, config-driven, advanced")
	initCmd.Flags().StringVarP(&initOutputDir, "output", "o", ".", "Output directory for the project")
	initCmd.Flags().BoolVarP(&initInteractive, "interactive", "i", false, "Enable interactive prompts")
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Force overwrite existing files")
	initCmd.Flags().StringVar(&initLLMProvider, "llm", "", "LLM provider (openai, anthropic, ollama)")
	initCmd.Flags().StringVar(&initAgentType, "agent-type", "", "Agent type (single, multi, specialized)")
	initCmd.Flags().StringVar(&initDescription, "description", "", "Project description")
}
