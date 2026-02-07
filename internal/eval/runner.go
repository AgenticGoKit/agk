package eval

import (
	"context"
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
	config         *RunnerConfig
	matcher        *Matcher        // Legacy matcher (deprecated)
	matcherFactory *MatcherFactory // New matcher factory
}

// NewRunner creates a new test runner
func NewRunner(config *RunnerConfig) *Runner {
	return &Runner{
		config:         config,
		matcher:        NewMatcher(), // Keep for backward compatibility
		matcherFactory: nil,          // Will be created when needed
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

	// Create matcher factory with semantic config from suite
	r.matcherFactory = NewMatcherFactory(suite.Semantic)

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

	if r.config.Verbose {
		fmt.Printf("  [HTTP Response] Success=%v, Error=%q, Output=%q (length: %d bytes)\n",
			resp != nil && resp.Success,
			func() string {
				if resp != nil {
					return resp.Error
				}
				return ""
			}(),
			func() string {
				if resp != nil {
					return resp.Output
				}
				return ""
			}(),
			func() int {
				if resp != nil {
					return len(resp.Output)
				}
				return 0
			}())
	}

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

	// Store expected output for reporting
	if test.Expect.Value != "" {
		result.ExpectedOutput = test.Expect.Value
	} else if len(test.Expect.Values) > 0 {
		result.ExpectedOutput = fmt.Sprintf("One of: %v", test.Expect.Values)
	} else if test.Expect.Pattern != "" {
		result.ExpectedOutput = fmt.Sprintf("Pattern: %s", test.Expect.Pattern)
	}

	// Match output against expectations using new matcher factory
	ctx := context.Background()
	matcher, err := r.matcherFactory.CreateMatcher(test.Expect)
	if err != nil {
		result.Passed = false
		result.ErrorMessage = fmt.Sprintf("failed to create matcher: %v", err)
		return result
	}

	matchResult, err := matcher.Match(ctx, resp.Output, test.Expect)
	if err != nil {
		result.Passed = false
		result.ErrorMessage = fmt.Sprintf("match error: %v", err)
		return result
	}

	// Store semantic matching results
	result.MatchStrategy = matchResult.Strategy
	result.Confidence = matchResult.Confidence
	result.MatchDetails = matchResult.Details

	if !matchResult.Matched {
		result.Passed = false
		result.ErrorMessage = matchResult.Explanation
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
