package eval

import "time"

// TestSuite represents a collection of tests
type TestSuite struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Target      Target            `yaml:"target"`
	Tests       []Test            `yaml:"tests"`
	Metadata    map[string]string `yaml:"metadata,omitempty"`
}

// Target defines where tests will be executed
type Target struct {
	Type string `yaml:"type"` // http, grpc, etc.
	URL  string `yaml:"url"`  // Base URL for HTTP targets
}

// Test represents a single test case
type Test struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description,omitempty"`
	Input       string                 `yaml:"input"`
	Expect      Expectation            `yaml:"expect"`
	Timeout     int                    `yaml:"timeout,omitempty"` // Override suite timeout
	Metadata    map[string]interface{} `yaml:"metadata,omitempty"`
}

// Expectation defines what to expect from test execution
type Expectation struct {
	Type      string   `yaml:"type"`      // exact, contains, regex, semantic
	Value     string   `yaml:"value,omitempty"`
	Values    []string `yaml:"values,omitempty"`
	Pattern   string   `yaml:"pattern,omitempty"`
	Threshold float64  `yaml:"threshold,omitempty"` // For semantic matching
	Trace     *TraceExpectation `yaml:"trace,omitempty"`
}

// TraceExpectation defines expectations for trace data
type TraceExpectation struct {
	ToolCalls     []string `yaml:"tool_calls,omitempty"`
	LLMCalls      int      `yaml:"llm_calls,omitempty"`
	ExecutionPath []string `yaml:"execution_path,omitempty"`
	MinSteps      int      `yaml:"min_steps,omitempty"`
	MaxSteps      int      `yaml:"max_steps,omitempty"`
}

// TestResult represents the result of a single test
type TestResult struct {
	TestName    string
	Passed      bool
	Duration    time.Duration
	ActualOutput string
	ExpectedOutput string
	ErrorMessage string
	TraceID     string
	Metadata    map[string]interface{}
}

// SuiteResults represents results for an entire test suite
type SuiteResults struct {
	SuiteName   string
	TotalTests  int
	PassedTests int
	FailedTests int
	Duration    time.Duration
	Results     []TestResult
	StartTime   time.Time
	EndTime     time.Time
}

// AllPassed returns true if all tests passed
func (sr *SuiteResults) AllPassed() bool {
	return sr.FailedTests == 0
}

// PassRate returns the pass rate as a percentage
func (sr *SuiteResults) PassRate() float64 {
	if sr.TotalTests == 0 {
		return 0
	}
	return float64(sr.PassedTests) / float64(sr.TotalTests) * 100
}
