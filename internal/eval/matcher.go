package eval

import (
	"fmt"
	"regexp"
	"strings"
)

// Matcher validates test outputs against expectations
type Matcher struct{}

// NewMatcher creates a new matcher
func NewMatcher() *Matcher {
	return &Matcher{}
}

// Match checks if actual output matches the expectation
func (m *Matcher) Match(actual string, expect Expectation) (bool, string) {
	switch expect.Type {
	case "exact":
		return m.matchExact(actual, expect.Value)
	case "contains":
		return m.matchContains(actual, expect.Values)
	case "regex":
		return m.matchRegex(actual, expect.Pattern)
	case "semantic":
		return m.matchSemantic(actual, expect.Value, expect.Threshold)
	default:
		return false, fmt.Sprintf("unknown expectation type: %s", expect.Type)
	}
}

// matchExact checks for exact string match
func (m *Matcher) matchExact(actual, expected string) (bool, string) {
	if actual == expected {
		return true, ""
	}
	return false, fmt.Sprintf("expected exact match:\n  Expected: %s\n  Actual:   %s", expected, actual)
}

// matchContains checks if actual contains all expected values
func (m *Matcher) matchContains(actual string, values []string) (bool, string) {
	actualLower := strings.ToLower(actual)
	var missing []string

	for _, value := range values {
		if !strings.Contains(actualLower, strings.ToLower(value)) {
			missing = append(missing, value)
		}
	}

	if len(missing) > 0 {
		return false, fmt.Sprintf("missing expected values: %v", missing)
	}

	return true, ""
}

// matchRegex checks if actual matches the regex pattern
func (m *Matcher) matchRegex(actual, pattern string) (bool, string) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, fmt.Sprintf("invalid regex pattern: %v", err)
	}

	if re.MatchString(actual) {
		return true, ""
	}

	return false, fmt.Sprintf("output does not match regex pattern: %s", pattern)
}

// matchSemantic performs semantic similarity matching
// For now, this is a simple implementation - can be enhanced with embeddings
func (m *Matcher) matchSemantic(actual, expected string, threshold float64) (bool, string) {
	// Simple implementation: check for significant word overlap
	actualWords := strings.Fields(strings.ToLower(actual))
	expectedWords := strings.Fields(strings.ToLower(expected))

	// Count matching words
	matches := 0
	for _, ew := range expectedWords {
		for _, aw := range actualWords {
			if ew == aw {
				matches++
				break
			}
		}
	}

	// Calculate similarity (simple word overlap ratio)
	similarity := float64(matches) / float64(len(expectedWords))

	if threshold == 0 {
		threshold = 0.7 // Default threshold
	}

	if similarity >= threshold {
		return true, ""
	}

	return false, fmt.Sprintf("semantic similarity %.2f below threshold %.2f", similarity, threshold)
}
