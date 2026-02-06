package eval

import (
	"fmt"
	"time"
)

// RunnerConfig configures the test runner
type RunnerConfig struct {
	Timeout      time.Duration
	Verbose      bool
	FailFast     bool
	OutputFormat string
}

// Runner executes test suites
type Runner struct {
	config  *RunnerConfig
	matcher *Matcher
}

// NewRunner creates a new test runner
func NewRunner(config *RunnerConfig) *Runner {
	return &Runner{
		config:  config,
		matcher: NewMatcher(),
	}
}

// Run executes a test suite and returns results
func (r *Runner) Run(suite *TestSuite) (*SuiteResults, error) {
	results := &SuiteResults{
		SuiteName:  suite.Name,
		TotalTests: len(suite.Tests),
		StartTime:  time.Now(),
		Results:    make([]TestResult, 0, len(suite.Tests)),
	}

	// Create target based on type
	var target *HTTPTarget
	if suite.Target.Type == "http" {
		target = NewHTTPTarget(suite.Target.URL, r.config.Timeout)

		// Health check
		if r.config.Verbose {
			fmt.Printf("\n🏥 Health check: %s\n", suite.Target.URL)
		}
		if err := target.Health(); err != nil {
			return nil, fmt.Errorf("target health check failed: %w", err)
		}
		if r.config.Verbose {
			fmt.Println("✓ Target is healthy")
		}
	} else {
		return nil, fmt.Errorf("unsupported target type: %s", suite.Target.Type)
	}

	// Run each test
	for i, test := range suite.Tests {
		if r.config.Verbose {
			fmt.Printf("\n[%d/%d] Running: %s\n", i+1, len(suite.Tests), test.Name)
		}

		result := r.runTest(test, target)
		results.Results = append(results.Results, result)

		if result.Passed {
			results.PassedTests++
			if r.config.Verbose {
				fmt.Printf("  ✓ PASSED (%.2fs)\n", result.Duration.Seconds())
			}
		} else {
			results.FailedTests++
			if r.config.Verbose {
				fmt.Printf("  ✗ FAILED: %s\n", result.ErrorMessage)
			}

			// Stop on first failure if fail-fast is enabled
			if r.config.FailFast {
				break
			}
		}
	}

	results.EndTime = time.Now()
	results.Duration = results.EndTime.Sub(results.StartTime)

	return results, nil
}

// runTest executes a single test
func (r *Runner) runTest(test Test, target *HTTPTarget) TestResult {
	result := TestResult{
		TestName: test.Name,
		Metadata: test.Metadata,
	}

	start := time.Now()

	// Get timeout for this test
	timeout := int(r.config.Timeout.Seconds())
	if test.Timeout > 0 {
		timeout = test.Timeout
	}

	// Invoke the target
	resp, err := target.Invoke(test.Input, timeout)
	result.Duration = time.Since(start)

	if err != nil {
		result.Passed = false
		result.ErrorMessage = fmt.Sprintf("invocation failed: %v", err)
		return result
	}

	if !resp.Success {
		result.Passed = false
		result.ErrorMessage = fmt.Sprintf("execution failed: %s", resp.Error)
		result.ActualOutput = resp.Output
		result.TraceID = resp.TraceID
		return result
	}

	// Store actual output and trace ID
	result.ActualOutput = resp.Output
	result.TraceID = resp.TraceID

	// Match output against expectations
	matched, errMsg := r.matcher.Match(resp.Output, test.Expect)
	if !matched {
		result.Passed = false
		result.ErrorMessage = errMsg
		return result
	}

	// TODO: Validate trace expectations if specified
	if test.Expect.Trace != nil {
		// This would require fetching trace data from /traces/{id}
		// For now, we'll skip trace validation
	}

	result.Passed = true
	return result
}
