package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
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
	evalReportFile   string
	evalServe        bool
	evalServeDir     string
	evalServeCmd     string
	evalServeWait    int
)

func init() {
	rootCmd.AddCommand(evalCmd)

	evalCmd.Flags().IntVar(&evalTimeout, "timeout", 300, "Timeout in seconds for each test")
	evalCmd.Flags().BoolVarP(&evalVerbose, "verbose", "v", false, "Verbose output")
	evalCmd.Flags().BoolVar(&evalValidateOnly, "validate-only", false, "Only validate test file, don't run tests")
	evalCmd.Flags().StringVarP(&evalOutputFormat, "format", "f", "console", "Output format (console, json, junit, markdown)")
	evalCmd.Flags().BoolVar(&evalFailFast, "fail-fast", false, "Stop on first test failure")
	evalCmd.Flags().StringVarP(&evalReportFile, "report", "r", "", "Save detailed report to file (auto-generated if not specified)")
	evalCmd.Flags().BoolVar(&evalServe, "serve", false, "Build & launch the project in EvalServer mode, run tests, then stop it")
	evalCmd.Flags().StringVar(&evalServeDir, "serve-dir", ".", "Project directory to launch when --serve is set")
	evalCmd.Flags().StringVar(&evalServeCmd, "serve-cmd", "", "Custom command to launch the server (default: go run .)")
	evalCmd.Flags().IntVar(&evalServeWait, "serve-timeout", 90, "Seconds to wait for the server to become healthy")
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

	// Optionally launch the project's EvalServer for the duration of the run.
	var srv *evalServer
	if evalServe {
		s, err := launchAndWait(evalServeDir, evalServeCmd, suite.Target.URL, evalServeWait, evalVerbose)
		if err != nil {
			return err
		}
		srv = s
		defer srv.Stop()

		// Ensure the server is stopped if the user interrupts the run.
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigCh
			srv.Stop()
			os.Exit(130)
		}()
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

	// Save detailed markdown report to file (by default)
	reportPath := evalReportFile
	if reportPath == "" {
		// Auto-generate report filename
		timestamp := time.Now().Format("20060102-150405")
		reportDir := ".agk/reports"
		if err := os.MkdirAll(reportDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create report directory: %v\n", err)
		} else {
			reportPath = filepath.Join(reportDir, fmt.Sprintf("eval-report-%s.md", timestamp))
		}
	}

	if reportPath != "" {
		reportFile, err := os.Create(reportPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create report file: %v\n", err)
		} else {
			defer reportFile.Close()
			mdReporter := eval.NewReporter("markdown")
			if err := mdReporter.Generate(results, reportFile); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to write markdown report: %v\n", err)
			} else {
				fmt.Printf("\n📄 Detailed report saved to: %s\n", reportPath)
			}
		}
	}

	// Exit with error code if tests failed. os.Exit skips deferred calls, so stop
	// the server explicitly first.
	if !results.AllPassed() {
		if srv != nil {
			srv.Stop()
		}
		os.Exit(1)
	}

	return nil
}
