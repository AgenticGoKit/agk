package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/agenticgokit/agenticgokit/observability"
	"github.com/fatih/color"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/agenticgokit/agk/pkg/registry"
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

  # Initialize with built-in template
	agk init my-project --template quickstart

  # Initialize from a community template (registry)
	agk init my-project --template <registry-template-name>

  # Initialize from a GitHub repository
  agk init my-project --template github.com/username/my-template

  # Initialize from a specific version
  agk init my-project --template github.com/username/my-template@v1.0.0

	# Non-interactive initialization
	agk init my-project --template quickstart --llm openai --force

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
	// Create observability span for command execution
	tracer := observability.GetTracer("agk-cli")
	ctx, span := tracer.Start(cmd.Context(), "agk.init")
	defer span.End()

	// Handle --list flag
	if initListTemplates {
		span.SetAttributes(attribute.Bool("list_templates", true))
		span.SetStatus(codes.Ok, "listed templates")
		listTemplates()
		return nil
	}

	projectName := args[0]
	span.SetAttributes(
		attribute.String("project_name", projectName),
		attribute.String("template", initTemplate),
		attribute.Bool("force", initForce),
	)

	// Validate project name
	if err := validateProjectName(projectName); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid project name")
		color.Red("✗ Invalid project name: %v", err)
		return err
	}

	projectPath := filepath.Join(initOutputDir, projectName)

	// Check if path already exists
	if _, err := os.Stat(projectPath); err == nil && !initForce {
		err := fmt.Errorf("project directory already exists")
		span.RecordError(err)
		span.SetStatus(codes.Error, "directory exists")
		color.Red("✗ Directory already exists: %s", projectPath)
		color.Yellow("Use --force to overwrite")
		return err
	}

	// Try to get generator (built-in or external)
	var generator scaffold.TemplateGenerator
	var metadata scaffold.TemplateMetadata
	var templateType scaffold.TemplateType

	// First, check if it's a built-in template
	builtInType, err := scaffold.ValidateTemplate(initTemplate)
	if err == nil {
		// It is built-in
		templateType = builtInType
		span.SetAttributes(attribute.String("template_type", string(templateType)))

		gen, err := scaffold.GetTemplateGenerator(templateType)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to get generator")
			color.Red("✗ Failed to get template generator: %v", err)
			return err
		}
		generator = gen
		metadata = gen.GetMetadata()
	} else {
		// Not built-in, try resolving as external template
		color.Cyan("ℹ️  Template '%s' not found locally, checking registry...", initTemplate)

		cm, err := registry.NewCacheManager("")
		if err != nil {
			return fmt.Errorf("failed to init cache manager: %w", err)
		}
		resolver := registry.NewResolver(cm)

		cached, err := resolver.Resolve(ctx, initTemplate)
		if err != nil {
			// Failed both built-in and external
			err = fmt.Errorf("template '%s' not found (neither built-in nor registry): %w", initTemplate, err)
			span.RecordError(err)
			span.SetStatus(codes.Error, "invalid template")
			color.Red("✗ %v", err)
			return err
		}

		gen := scaffold.NewExternalGenerator(cached)
		generator = gen
		metadata = gen.GetMetadata()
		templateType = scaffold.TemplateType("external") // Dummy type for next steps

		span.SetAttributes(
			attribute.String("template_type", "external"),
			attribute.String("external_source", cached.Source),
		)
		color.Green("✓ Found template '%s' version %s", cached.Name, cached.Version)
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
	metadata = generator.GetMetadata()
	color.Cyan("\n📦 Creating new AgenticGoKit project: %s\n", projectName)
	color.Cyan("   Template: %s (%s) - %s\n", metadata.Name, metadata.Complexity, metadata.Description)
	color.Cyan("   Files: %d | Features: %v\n", metadata.FileCount, metadata.Features)

	// Generate project using the template generator
	if err := generator.Generate(ctx, opts); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "generation failed")
		color.Red("✗ Project generation failed: %v", err)
		if logger != nil {
			logger.Error().Err(err).Msg("project generation failed")
		} else {
			// Fallback stderr
			l := zerolog.New(os.Stderr)
			l.Error().Err(err).Msg("project generation failed")
		}
		return err
	}

	// Print success message
	color.Green("\n✅ Project initialized successfully!\n")

	// Record success metrics
	span.SetAttributes(
		attribute.Int("file_count", metadata.FileCount),
		attribute.StringSlice("features", metadata.Features),
	)
	span.SetStatus(codes.Ok, "project initialized")

	// Print next steps
	printNextSteps(projectName, projectPath, templateType, metadata)

	return nil
}

// listTemplates prints all available templates with their metadata
func listTemplates() {
	color.Cyan("\n📋 Available AgenticGoKit Templates\n")
	color.Cyan("═══════════════════════════════════\n\n")

	// Built-in templates
	templates := scaffold.GetAllTemplates()
	color.Cyan("Built-in:\n")
	for i, tmpl := range templates {
		color.Green("%d. %s %s\n", i+1, tmpl.Name, tmpl.Complexity)
		fmt.Printf("   %s\n", color.YellowString(tmpl.Description))
		if len(tmpl.Features) > 0 {
			fmt.Printf("   Features: %v\n", color.CyanString("%v", tmpl.Features))
		}
		fmt.Printf("   Files: %s\n", color.MagentaString("%d", tmpl.FileCount))
		fmt.Printf("   Usage: %s\n", color.HiBlackString("agk init my-project --template %s", strings.ToLower(tmpl.Name)))
		if i < len(templates)-1 {
			fmt.Println()
		}
	}

	// Registry templates
	color.Cyan("\nRegistry:\n")
	index, err := registry.FetchIndex(registry.DefaultRegistryURL)
	if err != nil {
		fmt.Printf("   %s\n", color.YellowString("Unable to fetch registry templates: %v", err))
		fmt.Println()
		return
	}

	if len(index.Templates) == 0 {
		fmt.Printf("   %s\n", color.YellowString("No templates found in registry."))
		fmt.Println()
		return
	}

	registryNames := make([]string, 0, len(index.Templates))
	for name := range index.Templates {
		registryNames = append(registryNames, name)
	}
	sort.Strings(registryNames)
	for i, name := range registryNames {
		source := index.Templates[name]
		color.Green("%d. %s\n", i+1, name)
		fmt.Printf("   Source: %s\n", color.HiBlackString(source))
		fmt.Printf("   Usage: %s\n", color.HiBlackString("agk init my-project --template %s", name))
		if i < len(registryNames)-1 {
			fmt.Println()
		}
	}

	fmt.Println()
	fmt.Println()
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
func printNextSteps(_ string, projectPath string, templateType scaffold.TemplateType, _ scaffold.TemplateMetadata) {
	relPath, _ := filepath.Rel(".", projectPath)

	fmt.Println(color.BlueString("📖 Next Steps:"))
	fmt.Printf("  1. %s\n", color.CyanString("cd %s", relPath))
	fmt.Printf("  2. %s\n", color.CyanString("go mod tidy"))
	fmt.Printf("  3. %s\n", color.CyanString("export OPENAI_API_KEY=your-key-here  # Set your LLM API key"))
	fmt.Printf("  4. %s\n", color.CyanString("go run main.go                        # Run the project"))

	fmt.Println()
	fmt.Println(color.BlueString("📚 Project Structure:"))

	// Show actual structure based on template
	switch templateType {
	case scaffold.TemplateQuickstart:
		fmt.Printf("  • %s\n", color.CyanString("main.go                    # Entry point with hardcoded agent config"))
		fmt.Printf("  • %s\n", color.CyanString("go.mod                     # Go module definition"))
	case scaffold.TemplateWorkflow:
		fmt.Printf("  • %s\n", color.CyanString("main.go                    # Multi-step workflow pipeline"))
		fmt.Printf("  • %s\n", color.CyanString("README.md                  # Documentation for workflow"))
		fmt.Printf("  • %s\n", color.CyanString("go.mod                     # Go module definition"))
	default:
		// Generic structure for other templates
		fmt.Printf("  • %s\n", color.CyanString("main.go                    # Entry point"))
		fmt.Printf("  • %s\n", color.CyanString("go.mod                     # Go module definition"))
	}

	fmt.Println()
	fmt.Println(color.BlueString("💡 Development Tips:"))

	// Template-specific tips
	switch templateType {
	case scaffold.TemplateQuickstart:
		fmt.Printf("  • Edit the %s configuration in %s\n", color.CyanString("LLMConfig"), color.CyanString("main.go"))
		fmt.Printf("  • Modify the %s to customize the agent behavior\n", color.CyanString("SystemPrompt"))
		fmt.Printf("  • Try different LLM providers: %s, %s, %s\n",
			color.CyanString("openai"), color.CyanString("anthropic"), color.CyanString("ollama"))
	case scaffold.TemplateWorkflow:
		fmt.Printf("  • Add new step in %s using .AddStep()\n", color.CyanString("main.go"))
		fmt.Printf("  • Monitor step progress via streaming output\n")
		fmt.Printf("  • Use %s to debug workflow execution\n", color.CyanString("agk trace"))
	default:
		fmt.Printf("  • Configure your LLM provider and API keys\n")
		fmt.Printf("  • Explore the generated code to understand the structure\n")
	}

	fmt.Printf("  • Framework docs: %s\n", color.HiBlackString("https://docs.agenticgokit.com/"))
	fmt.Println()
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Define flags
	initCmd.Flags().BoolVar(&initListTemplates, "list", false, "List available templates")
	initCmd.Flags().StringVarP(&initTemplate, "template", "t", "quickstart",
		"Template name (built-in: quickstart, workflow; or a registry template)")
	initCmd.Flags().StringVarP(&initOutputDir, "output", "o", ".", "Output directory for the project")
	initCmd.Flags().BoolVarP(&initInteractive, "interactive", "i", false, "Enable interactive prompts")
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Force overwrite existing files")
	initCmd.Flags().StringVar(&initLLMProvider, "llm", "", "LLM provider (openai, anthropic, ollama)")
	initCmd.Flags().StringVar(&initAgentType, "agent-type", "", "Agent type (single, multi, specialized)")
	initCmd.Flags().StringVar(&initDescription, "description", "", "Project description")
}
