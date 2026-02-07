package eval

import "time"

// TestSuite represents a collection of tests
type TestSuite struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Target      Target            `yaml:"target"`
	Semantic    *SemanticConfig   `yaml:"semantic,omitempty"` // Global semantic matching config
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
	Type        string            `yaml:"type"` // exact, contains, regex, semantic
	Value       string            `yaml:"value,omitempty"`
	Values      []string          `yaml:"values,omitempty"`
	Pattern     string            `yaml:"pattern,omitempty"`
	Threshold   *float64          `yaml:"threshold,omitempty"` // For semantic matching (pointer for override detection)
	Description string            `yaml:"description,omitempty"`
	Trace       *TraceExpectation `yaml:"trace,omitempty"`

	// Semantic matching overrides (optional, per-test)
	Strategy    string           `yaml:"strategy,omitempty"`     // Override global strategy
	LLM         *LLMConfig       `yaml:"llm,omitempty"`          // Override global LLM config
	Embedding   *EmbeddingConfig `yaml:"embedding,omitempty"`    // Override global embedding config
	JudgePrompt string           `yaml:"judge_prompt,omitempty"` // Override global judge prompt
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
	TestName       string
	Passed         bool
	Duration       time.Duration
	ActualOutput   string
	ExpectedOutput string
	ErrorMessage   string
	TraceID        string
	Metadata       map[string]interface{}

	// Semantic matching results
	MatchStrategy string                 `json:"match_strategy,omitempty"` // embedding, llm-judge, hybrid
	Confidence    float64                `json:"confidence,omitempty"`     // 0.0 - 1.0
	MatchDetails  map[string]interface{} `json:"match_details,omitempty"`  // Strategy-specific details
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

// SemanticConfig defines semantic matching configuration
type SemanticConfig struct {
	Strategy    string           `yaml:"strategy"`               // embedding | llm-judge | hybrid
	LLM         *LLMConfig       `yaml:"llm,omitempty"`          // LLM configuration for llm-judge strategy
	Embedding   *EmbeddingConfig `yaml:"embedding,omitempty"`    // Embedding configuration
	Threshold   float64          `yaml:"threshold"`              // Similarity threshold (0.0 - 1.0)
	JudgePrompt string           `yaml:"judge_prompt,omitempty"` // Custom judge prompt template
}

// LLMConfig for LLM-based semantic matching
type LLMConfig struct {
	Provider    string  `yaml:"provider"`           // ollama | openai | anthropic
	Model       string  `yaml:"model"`              // Model name
	Temperature float64 `yaml:"temperature"`        // Temperature for generation
	MaxTokens   int     `yaml:"max_tokens"`         // Max tokens for response
	BaseURL     string  `yaml:"base_url,omitempty"` // Optional base URL
}

// EmbeddingConfig for embedding-based semantic matching
type EmbeddingConfig struct {
	Provider string `yaml:"provider"`           // ollama | openai
	Model    string `yaml:"model"`              // Embedding model name
	BaseURL  string `yaml:"base_url,omitempty"` // Optional base URL
}
