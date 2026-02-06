package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/agenticgokit/agk/internal/eval"
)

var evalCmd = &cobra.Command{
	Use:   "eval <test-file>",
	Short: "Run evaluation tests against your agents/workflows",
	Long: `Run evaluation tests defined in YAML files against your agents and workflows.

Examples:
  # Run tests from a file
  agk eval tests.yaml
  
  # Run with custom timeout
  agk eval tests.yaml --timeout 300
  
  # Run with verbose output
  agk eval tests.yaml --verbose
  
  # Validate test file without running
  agk eval tests.yaml --validate-only`,
	Args: cobra.ExactArgs(1),
	RunE: runEval,
}

var (
	evalTimeout      int
	evalVerbose      bool
	evalValidateOnly bool
	evalOutputFormat string
	evalFailFast     bool
)

func init() {
	rootCmd.AddCommand(evalCmd)

	evalCmd.Flags().IntVar(&evalTimeout, "timeout", 300, "Timeout in seconds for each test")
	evalCmd.Flags().BoolVarP(&evalVerbose, "verbose", "v", false, "Verbose output")
	evalCmd.Flags().BoolVar(&evalValidateOnly, "validate-only", false, "Only validate test file, don't run tests")
	evalCmd.Flags().StringVarP(&evalOutputFormat, "format", "f", "console", "Output format (console, json, junit)")
	evalCmd.Flags().BoolVar(&evalFailFast, "fail-fast", false, "Stop on first test failure")
}

func runEval(cmd *cobra.Command, args []string) error {
	testFile := args[0]

	// Check if file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		return fmt.Errorf("test file not found: %s", testFile)
	}

	// Get absolute path
	absPath, err := filepath.Abs(testFile)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	if evalVerbose {
		fmt.Printf("📋 Loading test file: %s\n", absPath)
	}

	// Parse test file
	suite, err := eval.ParseTestFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to parse test file: %w", err)
	}

	if evalVerbose {
		fmt.Printf("✓ Loaded %d test(s) from suite: %s\n", len(suite.Tests), suite.Name)
	}

	// Validate only mode
	if evalValidateOnly {
		fmt.Println("✓ Test file is valid")
		return nil
	}

	// Create test runner
	runner := eval.NewRunner(&eval.RunnerConfig{
		Timeout:      time.Duration(evalTimeout) * time.Second,
		Verbose:      evalVerbose,
		FailFast:     evalFailFast,
		OutputFormat: evalOutputFormat,
	})

	// Run tests
	if evalVerbose {
		fmt.Println("\n🚀 Running tests...")
		fmt.Println("==================")
	}

	results, err := runner.Run(suite)
	if err != nil {
		return fmt.Errorf("test execution failed: %w", err)
	}

	// Generate report
	reporter := eval.NewReporter(evalOutputFormat)
	if err := reporter.Generate(results, os.Stdout); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Exit with error code if tests failed
	if !results.AllPassed() {
		os.Exit(1)
	}

	return nil
}
