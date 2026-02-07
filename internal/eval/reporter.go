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

	// Executive Summary Banner
	if results.AllPassed() {
		fmt.Fprintf(w, "> **Status: PASSED** - %d/%d tests completed successfully in %s\n\n",
			results.PassedTests, results.TotalTests, formatDuration(results.Duration))
	} else {
		fmt.Fprintf(w, "> **Status: FAILED** - %d test(s) failed out of %d total tests. Pass rate: %.1f%%\n\n",
			results.FailedTests, results.TotalTests, results.PassRate())
	}

	fmt.Fprintf(w, "**Generated:** %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// Quick Stats with visual bars
	fmt.Fprintf(w, "## Summary\n\n")
	fmt.Fprintf(w, "| Metric | Value | Progress |\n")
	fmt.Fprintf(w, "|--------|-------|----------|\n")
	fmt.Fprintf(w, "| **Total Tests** | %d | |\n", results.TotalTests)
	fmt.Fprintf(w, "| **Passed** | %d | %s |\n", results.PassedTests, generateBar(results.PassedTests, results.TotalTests, "✓"))
	fmt.Fprintf(w, "| **Failed** | %d | %s |\n", results.FailedTests, generateBar(results.FailedTests, results.TotalTests, "✗"))
	fmt.Fprintf(w, "| **Pass Rate** | %.1f%% | %s |\n", results.PassRate(), generateProgressBar(results.PassRate()))
	fmt.Fprintf(w, "| **Duration** | %s | |\n\n", formatDuration(results.Duration))

	// Quick Navigation for failed tests
	if !results.AllPassed() {
		fmt.Fprintf(w, "### Failed Tests\n\n")
		for i, result := range results.Results {
			if !result.Passed {
				fmt.Fprintf(w, "- [%s](#%d---%s) - %.2fs\n",
					result.TestName, i+1, strings.ReplaceAll(strings.ToLower(result.TestName), " ", "-"), result.Duration.Seconds())
			}
		}
		fmt.Fprintf(w, "\n")
	}

	// Test Results section with enhanced formatting
	fmt.Fprintf(w, "---\n\n")
	fmt.Fprintf(w, "## Detailed Test Results\n\n")

	for i, result := range results.Results {
		statusBadge := "PASSED"
		if !result.Passed {
			statusBadge = "FAILED"
		}

		fmt.Fprintf(w, "### %d. %s\n\n", i+1, result.TestName)

		// Status badge
		fmt.Fprintf(w, "**Status:** `%s` | **Duration:** %s\n\n",
			statusBadge, formatDuration(result.Duration))

		// Semantic matching details with visual confidence
		if result.MatchStrategy != "" {
			fmt.Fprintf(w, "**Matching Strategy:** `%s`\n\n", result.MatchStrategy)

			if result.Confidence > 0 {
				confidenceBar := generateConfidenceBar(result.Confidence)
				fmt.Fprintf(w, "**Confidence Score:** %.0f%%\n\n", result.Confidence*100)
				fmt.Fprintf(w, "```\n%s\n```\n\n", confidenceBar)
			}

			// LLM Judge Evaluation
			if result.MatchStrategy == "llm-judge" && result.MatchDetails != nil {
				judgeResp, ok := result.MatchDetails["judge_response"].(string)
				if ok {
					fmt.Fprintf(w, "#### LLM Judge Evaluation\n\n")
					if judgeResp != "" {
						// Parse verdict from response
						verdict := "Unknown"
						if strings.HasPrefix(strings.ToUpper(judgeResp), "YES") {
							verdict = "Approved"
						} else if strings.HasPrefix(strings.ToUpper(judgeResp), "NO") {
							verdict = "Rejected"
						}
						fmt.Fprintf(w, "**Verdict:** %s\n\n", verdict)
						fmt.Fprintf(w, "<details>\n<summary>View Judge's Reasoning</summary>\n\n")
						fmt.Fprintf(w, "```\n%s\n```\n\n", judgeResp)
						fmt.Fprintf(w, "</details>\n\n")
					} else {
						fmt.Fprintf(w, "> *Judge returned empty response*\n\n")
					}
				}
			}

			// Other match details in compact format
			if len(result.MatchDetails) > 0 {
				fmt.Fprintf(w, "<details>\n<summary>Technical Details</summary>\n\n")
				for k, v := range result.MatchDetails {
					if k == "judge_response" && result.MatchStrategy == "llm-judge" {
						continue
					}
					fmt.Fprintf(w, "- **%s:** `%v`\n", k, v)
				}
				fmt.Fprintf(w, "\n</details>\n\n")
			}
		}

		// Trace information
		if result.TraceID != "" {
			fmt.Fprintf(w, "**Trace ID:** [`%s`](.agk/runs/%s/)\n\n", result.TraceID, result.TraceID)
		}

		// Error message - prominent for failed tests
		if !result.Passed && result.ErrorMessage != "" {
			fmt.Fprintf(w, "#### Failure Details\n\n")
			fmt.Fprintf(w, "```\n%s\n```\n\n", result.ErrorMessage)
		}

		// Expected vs Actual Comparison
		if result.ExpectedOutput != "" || result.ActualOutput != "" {
			fmt.Fprintf(w, "#### Output Comparison\n\n")

			// Show side-by-side if both present
			if result.ExpectedOutput != "" {
				fmt.Fprintf(w, "<details>\n<summary>Expected Output</summary>\n\n")
				fmt.Fprintf(w, "```\n%s\n```\n\n", result.ExpectedOutput)
				fmt.Fprintf(w, "</details>\n\n")
			}

			if result.ActualOutput != "" {
				fmt.Fprintf(w, "<details open>\n<summary>Actual Output</summary>\n\n")
				fmt.Fprintf(w, "```\n%s\n```\n\n", result.ActualOutput)
				fmt.Fprintf(w, "</details>\n\n")
			} else if !result.Passed {
				fmt.Fprintf(w, "> **Actual Output:** *(empty)*\n\n")
			}
		}

		// Additional metadata
		if len(result.Metadata) > 0 {
			fmt.Fprintf(w, "<details>\n<summary>Additional Metadata</summary>\n\n")
			for k, v := range result.Metadata {
				fmt.Fprintf(w, "- **%s:** %v\n", k, v)
			}
			fmt.Fprintf(w, "\n</details>\n\n")
		}

		fmt.Fprintf(w, "---\n\n")
	}

	// Trace analysis section with helpful tips
	fmt.Fprintf(w, "## Trace Analysis & Debugging\n\n")
	fmt.Fprintf(w, "All test execution traces are saved in `.agk/runs/` for detailed inspection.\n\n")

	if !results.AllPassed() {
		fmt.Fprintf(w, "### Debugging Tips\n\n")
		fmt.Fprintf(w, "1. **View detailed traces:** Use `agk trace show <trace-id>` to see step-by-step execution\n")
		fmt.Fprintf(w, "2. **Compare outputs:** Check the Expected vs Actual sections above\n")
		fmt.Fprintf(w, "3. **Check confidence scores:** Low scores may indicate semantic mismatch\n")
		fmt.Fprintf(w, "4. **Review LLM judge reasoning:** Expand the judge's evaluation for insights\n\n")
	}

	fmt.Fprintf(w, "### Commands\n\n")
	fmt.Fprintf(w, "```bash\n")
	fmt.Fprintf(w, "# View specific trace with full details\n")
	fmt.Fprintf(w, "agk trace show <trace-id>\n\n")
	fmt.Fprintf(w, "# List all available traces\n")
	fmt.Fprintf(w, "agk trace list\n\n")
	fmt.Fprintf(w, "# Re-run tests\n")
	fmt.Fprintf(w, "agk eval <test-file.yaml>\n")
	fmt.Fprintf(w, "```\n\n")

	// Final summary
	if results.AllPassed() {
		fmt.Fprintf(w, "---\n\n")
		fmt.Fprintf(w, "## Summary\n\n")
		fmt.Fprintf(w, "All tests passed successfully. Your system is performing as expected.\n\n")
	}

	// Report footer with generation details
	fmt.Fprintf(w, "---\n\n")
	fmt.Fprintf(w, "<div align=\"center\">\n\n")
	fmt.Fprintf(w, "**Report Generated by AGK Eval Tool**\n\n")
	fmt.Fprintf(w, "Date: %s\n\n", time.Now().Format("Monday, January 2, 2006 at 3:04 PM MST"))
	fmt.Fprintf(w, "Tool: AgenticGoKit (AGK) Evaluation Framework v1beta\n\n")
	fmt.Fprintf(w, "---\n\n")
	fmt.Fprintf(w, "*Powered by [AgenticGoKit](https://github.com/agenticgokit/agenticgokit)*\n\n")
	fmt.Fprintf(w, "</div>\n")

	return nil
}

// Helper functions

// generateBar creates a visual bar representation
func generateBar(count, total int, emoji string) string {
	if total == 0 {
		return ""
	}
	barLength := 10
	filled := (count * barLength) / total
	bar := strings.Repeat(emoji, filled)
	return bar
}

// generateProgressBar creates a progress bar for percentages
func generateProgressBar(percentage float64) string {
	barLength := 20
	filled := int(percentage * float64(barLength) / 100)
	empty := barLength - filled

	bar := "["
	bar += strings.Repeat("█", filled)
	bar += strings.Repeat("░", empty)
	bar += "]"

	return bar
}

// generateConfidenceBar creates a visual confidence meter
func generateConfidenceBar(confidence float64) string {
	percentage := confidence * 100
	barLength := 50
	filled := int(confidence * float64(barLength))
	empty := barLength - filled

	bar := ""
	if percentage >= 80 {
		bar += strings.Repeat("█", filled)
	} else if percentage >= 60 {
		bar += strings.Repeat("▓", filled)
	} else {
		bar += strings.Repeat("▒", filled)
	}
	bar += strings.Repeat("░", empty)
	bar += fmt.Sprintf(" %.0f%%", percentage)

	return bar
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
