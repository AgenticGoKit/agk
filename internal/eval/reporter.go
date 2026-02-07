package eval

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// Reporter generates test reports in various formats
type Reporter struct {
	format string
}

// NewReporter creates a new reporter
func NewReporter(format string) *Reporter {
	return &Reporter{format: format}
}

// Generate creates a report and writes it to the writer
func (r *Reporter) Generate(results *SuiteResults, w io.Writer) error {
	switch r.format {
	case "console":
		return r.generateConsole(results, w)
	case "json":
		return r.generateJSON(results, w)
	case "junit":
		return r.generateJUnit(results, w)
	case "markdown":
		return r.generateMarkdown(results, w)
	default:
		return fmt.Errorf("unsupported format: %s", r.format)
	}
}

// generateConsole creates a human-readable console report
func (r *Reporter) generateConsole(results *SuiteResults, w io.Writer) error {
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "═══════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(w, "  TEST RESULTS: %s\n", results.SuiteName)
	fmt.Fprintf(w, "═══════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(w, "\n")

	// Summary
	fmt.Fprintf(w, "Total Tests:    %d\n", results.TotalTests)
	fmt.Fprintf(w, "Passed:         %d ✓\n", results.PassedTests)
	fmt.Fprintf(w, "Failed:         %d ✗\n", results.FailedTests)
	fmt.Fprintf(w, "Pass Rate:      %.1f%%\n", results.PassRate())
	fmt.Fprintf(w, "Duration:       %s\n", formatDuration(results.Duration))
	fmt.Fprintf(w, "\n")

	// Failed tests details
	if results.FailedTests > 0 {
		fmt.Fprintf(w, "───────────────────────────────────────────────────────────────\n")
		fmt.Fprintf(w, "  FAILED TESTS\n")
		fmt.Fprintf(w, "───────────────────────────────────────────────────────────────\n")
		fmt.Fprintf(w, "\n")

		for _, result := range results.Results {
			if !result.Passed {
				fmt.Fprintf(w, "✗ %s\n", result.TestName)
				fmt.Fprintf(w, "  Duration: %s\n", formatDuration(result.Duration))

				// Show semantic matching details if available
				if result.MatchStrategy != "" {
					fmt.Fprintf(w, "  Strategy: %s", result.MatchStrategy)
					if result.Confidence > 0 {
						fmt.Fprintf(w, " (confidence: %.2f)", result.Confidence)
					}
					fmt.Fprintf(w, "\n")
				}

				if result.TraceID != "" {
					fmt.Fprintf(w, "  Trace ID: %s\n", result.TraceID)
					fmt.Fprintf(w, "  💡 View detailed trace: agk trace show %s\n", result.TraceID)
					fmt.Fprintf(w, "  📁 Trace location: .agk/runs/%s/\n", result.TraceID)
				}
				fmt.Fprintf(w, "  Error: %s\n", result.ErrorMessage)
				if result.ActualOutput != "" {
					fmt.Fprintf(w, "  Output:\n")
					fmt.Fprintf(w, "    %s\n", truncate(result.ActualOutput, 200))
				}
				fmt.Fprintf(w, "\n")
			}
		}
	}

	// Overall status
	fmt.Fprintf(w, "───────────────────────────────────────────────────────────────\n")
	if results.AllPassed() {
		fmt.Fprintf(w, "  ✓ ALL TESTS PASSED\n")
	} else {
		fmt.Fprintf(w, "  ✗ SOME TESTS FAILED\n")
	}
	fmt.Fprintf(w, "───────────────────────────────────────────────────────────────\n")
	fmt.Fprintf(w, "\n")

	// Trace analysis instructions
	fmt.Fprintf(w, "📊 DETAILED ANALYSIS:\n")
	fmt.Fprintf(w, "  • All traces saved in: .agk/runs/\n")
	fmt.Fprintf(w, "  • Use 'agk trace show <trace-id>' for detailed execution analysis\n")
	fmt.Fprintf(w, "  • Use 'agk trace list' to see all available traces\n")
	fmt.Fprintf(w, "\n")

	return nil
}

// generateJSON creates a JSON report
func (r *Reporter) generateJSON(results *SuiteResults, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

// generateJUnit creates a JUnit XML report
func (r *Reporter) generateJUnit(results *SuiteResults, w io.Writer) error {
	fmt.Fprintf(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	fmt.Fprintf(w, "<testsuite name=\"%s\" tests=\"%d\" failures=\"%d\" time=\"%.3f\">\n",
		results.SuiteName, results.TotalTests, results.FailedTests, results.Duration.Seconds())

	for _, result := range results.Results {
		fmt.Fprintf(w, "  <testcase name=\"%s\" time=\"%.3f\">\n",
			escapeXML(result.TestName), result.Duration.Seconds())

		if !result.Passed {
			fmt.Fprintf(w, "    <failure message=\"%s\">\n", escapeXML(result.ErrorMessage))
			fmt.Fprintf(w, "      Actual Output: %s\n", escapeXML(result.ActualOutput))
			fmt.Fprintf(w, "    </failure>\n")
		}

		fmt.Fprintf(w, "  </testcase>\n")
	}

	fmt.Fprintf(w, "</testsuite>\n")
	return nil
}

// generateMarkdown creates a detailed Markdown report
func (r *Reporter) generateMarkdown(results *SuiteResults, w io.Writer) error {
	fmt.Fprintf(w, "# Test Report: %s\n\n", results.SuiteName)
	fmt.Fprintf(w, "**Generated:** %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// Summary section
	fmt.Fprintf(w, "## Summary\n\n")
	fmt.Fprintf(w, "| Metric | Value |\n")
	fmt.Fprintf(w, "|--------|-------|\n")
	fmt.Fprintf(w, "| Total Tests | %d |\n", results.TotalTests)
	fmt.Fprintf(w, "| Passed | %d ✓ |\n", results.PassedTests)
	fmt.Fprintf(w, "| Failed | %d ✗ |\n", results.FailedTests)
	fmt.Fprintf(w, "| Pass Rate | %.1f%% |\n", results.PassRate())
	fmt.Fprintf(w, "| Duration | %s |\n\n", formatDuration(results.Duration))

	if results.AllPassed() {
		fmt.Fprintf(w, "### ✓ All Tests Passed\n\n")
	} else {
		fmt.Fprintf(w, "### ✗ Some Tests Failed\n\n")
	}

	// Test Results section
	fmt.Fprintf(w, "## Test Results\n\n")

	for i, result := range results.Results {
		if result.Passed {
			fmt.Fprintf(w, "### %d. ✓ %s\n\n", i+1, result.TestName)
		} else {
			fmt.Fprintf(w, "### %d. ✗ %s\n\n", i+1, result.TestName)
		}

		fmt.Fprintf(w, "**Status:** ")
		if result.Passed {
			fmt.Fprintf(w, "PASSED ✓\n\n")
		} else {
			fmt.Fprintf(w, "FAILED ✗\n\n")
		}

		fmt.Fprintf(w, "**Duration:** %s\n\n", formatDuration(result.Duration))

		// Semantic matching details
		if result.MatchStrategy != "" {
			fmt.Fprintf(w, "**Matching Strategy:** %s\n\n", result.MatchStrategy)
			if result.Confidence > 0 {
				fmt.Fprintf(w, "**Confidence Score:** %.2f\n\n", result.Confidence)
			}

			// Show LLM judge response prominently
			if result.MatchStrategy == "llm-judge" && result.MatchDetails != nil {
				judgeResp, ok := result.MatchDetails["judge_response"].(string)
				if ok {
					if judgeResp != "" {
						fmt.Fprintf(w, "**LLM Judge Evaluation:**\n\n```\n%s\n```\n\n", judgeResp)
					} else {
						fmt.Fprintf(w, "**LLM Judge Evaluation:** *(empty response)*\n\n")
					}
				}
			}

			// Show other match details
			if len(result.MatchDetails) > 0 {
				fmt.Fprintf(w, "**Match Details:**\n\n")
				for k, v := range result.MatchDetails {
					// Skip judge_response since we already showed it prominently
					if k == "judge_response" && result.MatchStrategy == "llm-judge" {
						continue
					}
					fmt.Fprintf(w, "- **%s:** %v\n", k, v)
				}
				fmt.Fprintf(w, "\n")
			}
		}

		if result.TraceID != "" {
			fmt.Fprintf(w, "**Trace ID:** `%s`\n\n", result.TraceID)
			fmt.Fprintf(w, "**Trace Location:** `.agk/runs/%s/`\n\n", result.TraceID)
		}

		if !result.Passed {
			fmt.Fprintf(w, "**Error:**\n\n```\n%s\n```\n\n", result.ErrorMessage)
		}

		if result.ExpectedOutput != "" {
			fmt.Fprintf(w, "**Expected Output:**\n\n```\n%s\n```\n\n", result.ExpectedOutput)
		}

		// Always show actual output for failed tests
		if result.ActualOutput != "" {
			fmt.Fprintf(w, "**Actual Output:**\n\n```\n%s\n```\n\n", result.ActualOutput)
		} else if !result.Passed {
			fmt.Fprintf(w, "**Actual Output:** *(empty)*\n\n")
		}

		if result.Metadata != nil && len(result.Metadata) > 0 {
			fmt.Fprintf(w, "**Metadata:**\n\n")
			for k, v := range result.Metadata {
				fmt.Fprintf(w, "- **%s:** %v\n", k, v)
			}
			fmt.Fprintf(w, "\n")
		}

		fmt.Fprintf(w, "---\n\n")
	}

	// Trace analysis section
	fmt.Fprintf(w, "## Trace Analysis\n\n")
	fmt.Fprintf(w, "All test execution traces are saved in `.agk/runs/`\n\n")
	fmt.Fprintf(w, "To view detailed trace information:\n\n")
	fmt.Fprintf(w, "```bash\n")
	fmt.Fprintf(w, "# View specific trace\n")
	fmt.Fprintf(w, "agk trace show <trace-id>\n\n")
	fmt.Fprintf(w, "# List all traces\n")
	fmt.Fprintf(w, "agk trace list\n")
	fmt.Fprintf(w, "```\n\n")

	return nil
}

// Helper functions

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d.Milliseconds()))
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
