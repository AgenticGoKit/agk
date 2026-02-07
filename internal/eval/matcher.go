package eval

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// MatchResult represents the result of a match operation
type MatchResult struct {
	Matched     bool                   // Whether the output matched the expectation
	Confidence  float64                // Confidence score (0.0 - 1.0)
	Explanation string                 // Human-readable explanation
	Strategy    string                 // Strategy used (exact, contains, regex, semantic)
	Details     map[string]interface{} // Strategy-specific details
}

// MatcherInterface defines the interface for output validation
type MatcherInterface interface {
	// Match checks if actual output matches expected criteria
	Match(ctx context.Context, actual string, expected Expectation) (*MatchResult, error)

	// Name returns the matcher strategy name
	Name() string
}

// MatcherFactory creates matchers based on configuration
type MatcherFactory struct {
	semanticConfig *SemanticConfig
}

// NewMatcherFactory creates a new matcher factory
func NewMatcherFactory(config *SemanticConfig) *MatcherFactory {
	return &MatcherFactory{semanticConfig: config}
}

// CreateMatcher creates appropriate matcher for expectation type
func (f *MatcherFactory) CreateMatcher(exp Expectation) (MatcherInterface, error) {
	switch exp.Type {
	case "exact":
		return NewExactMatcher(), nil
	case "contains":
		return NewContainsMatcher(), nil
	case "regex":
		return NewRegexMatcher(), nil
	case "semantic":
		return f.createSemanticMatcher(exp)
	default:
		return nil, fmt.Errorf("unknown expectation type: %s", exp.Type)
	}
}

// createSemanticMatcher creates a semantic matcher with merged configuration
func (f *MatcherFactory) createSemanticMatcher(exp Expectation) (MatcherInterface, error) {
	// Merge global config with test-specific overrides
	config := f.mergeSemanticConfig(exp)

	// Determine strategy
	strategy := MatcherStrategyLLMJudge // default
	if config.Strategy != "" {
		strategy = config.Strategy
	}

	// Create appropriate matcher
	switch strategy {
	case MatcherStrategyEmbedding:
		return NewEmbeddingMatcher(config)
	case MatcherStrategyLLMJudge:
		return NewLLMJudgeMatcher(config)
	case MatcherStrategyHybrid:
		return NewHybridMatcher(config)
	default:
		return nil, fmt.Errorf("unknown semantic strategy: %s", strategy)
	}
}

// mergeSemanticConfig merges global semantic config with test-specific overrides
func (f *MatcherFactory) mergeSemanticConfig(exp Expectation) *SemanticConfig {
	// Start with global config or defaults
	config := &SemanticConfig{
		Strategy:  MatcherStrategyLLMJudge,
		Threshold: 0.85,
	}

	if f.semanticConfig != nil {
		// Copy global config
		config.Strategy = f.semanticConfig.Strategy
		config.Threshold = f.semanticConfig.Threshold
		config.JudgePrompt = f.semanticConfig.JudgePrompt

		if f.semanticConfig.LLM != nil {
			llmCopy := *f.semanticConfig.LLM
			config.LLM = &llmCopy
		}

		if f.semanticConfig.Embedding != nil {
			embCopy := *f.semanticConfig.Embedding
			config.Embedding = &embCopy
		}
	}

	// Apply test-specific overrides
	if exp.Strategy != "" {
		config.Strategy = exp.Strategy
	}

	if exp.Threshold != nil {
		config.Threshold = *exp.Threshold
	}

	if exp.JudgePrompt != "" {
		config.JudgePrompt = exp.JudgePrompt
	}

	if exp.LLM != nil {
		config.LLM = exp.LLM
	}

	if exp.Embedding != nil {
		config.Embedding = exp.Embedding
	}

	return config
}

// ========================================
// Built-in Matchers
// ========================================

// ExactMatcher checks for exact string match
type ExactMatcher struct{}

func NewExactMatcher() *ExactMatcher {
	return &ExactMatcher{}
}

func (m *ExactMatcher) Match(ctx context.Context, actual string, exp Expectation) (*MatchResult, error) {
	expected := exp.Value
	if expected == "" && len(exp.Values) > 0 {
		expected = exp.Values[0]
	}

	matched := actual == expected
	confidence := 1.0
	if !matched {
		confidence = 0.0
	}

	explanation := "exact match"
	if !matched {
		explanation = fmt.Sprintf("expected exact match: %q, got: %q", expected, actual)
	}

	return &MatchResult{
		Matched:     matched,
		Confidence:  confidence,
		Strategy:    "exact",
		Explanation: explanation,
	}, nil
}

func (m *ExactMatcher) Name() string {
	return "exact"
}

// ContainsMatcher checks if actual contains expected values
type ContainsMatcher struct{}

func NewContainsMatcher() *ContainsMatcher {
	return &ContainsMatcher{}
}

func (m *ContainsMatcher) Match(ctx context.Context, actual string, exp Expectation) (*MatchResult, error) {
	values := exp.Values
	if len(values) == 0 && exp.Value != "" {
		values = []string{exp.Value}
	}

	actualLower := strings.ToLower(actual)
	var missing []string

	for _, value := range values {
		if !strings.Contains(actualLower, strings.ToLower(value)) {
			missing = append(missing, value)
		}
	}

	matched := len(missing) == 0
	confidence := 1.0
	if !matched {
		confidence = 0.0
	}

	explanation := "contains all expected values"
	if !matched {
		explanation = fmt.Sprintf("missing expected values: %v", missing)
	}

	return &MatchResult{
		Matched:     matched,
		Confidence:  confidence,
		Strategy:    "contains",
		Explanation: explanation,
		Details: map[string]interface{}{
			"expected": values,
			"missing":  missing,
		},
	}, nil
}

func (m *ContainsMatcher) Name() string {
	return "contains"
}

// RegexMatcher checks if actual matches regex pattern
type RegexMatcher struct{}

func NewRegexMatcher() *RegexMatcher {
	return &RegexMatcher{}
}

func (m *RegexMatcher) Match(ctx context.Context, actual string, exp Expectation) (*MatchResult, error) {
	pattern := exp.Pattern
	if pattern == "" && exp.Value != "" {
		pattern = exp.Value
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	matched := re.MatchString(actual)
	confidence := 1.0
	if !matched {
		confidence = 0.0
	}

	explanation := "matches regex pattern"
	if !matched {
		explanation = fmt.Sprintf("does not match regex pattern: %s", pattern)
	}

	return &MatchResult{
		Matched:     matched,
		Confidence:  confidence,
		Strategy:    "regex",
		Explanation: explanation,
		Details: map[string]interface{}{
			"pattern": pattern,
		},
	}, nil
}

func (m *RegexMatcher) Name() string {
	return "regex"
}

// ========================================
// Legacy Matcher (for backward compatibility)
// ========================================

// Matcher validates test outputs against expectations (legacy)
type Matcher struct{}

// NewMatcher creates a new matcher
func NewMatcher() *Matcher {
	return &Matcher{}
}

// Match checks if actual output matches the expectation (legacy method)
func (m *Matcher) Match(actual string, expect Expectation) (bool, string) {
	ctx := context.Background()
	factory := NewMatcherFactory(nil)

	matcher, err := factory.CreateMatcher(expect)
	if err != nil {
		return false, err.Error()
	}

	result, err := matcher.Match(ctx, actual, expect)
	if err != nil {
		return false, err.Error()
	}

	return result.Matched, result.Explanation
}
