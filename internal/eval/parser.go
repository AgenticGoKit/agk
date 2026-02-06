package eval

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ParseTestFile parses a YAML test file into a TestSuite
func ParseTestFile(filePath string) (*TestSuite, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var suite TestSuite
	if err := yaml.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate suite
	if err := validateSuite(&suite); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &suite, nil
}

// validateSuite validates the test suite structure
func validateSuite(suite *TestSuite) error {
	if suite.Name == "" {
		return fmt.Errorf("suite name is required")
	}

	if suite.Target.Type == "" {
		return fmt.Errorf("target type is required")
	}

	if suite.Target.Type == "http" && suite.Target.URL == "" {
		return fmt.Errorf("target URL is required for HTTP targets")
	}

	if len(suite.Tests) == 0 {
		return fmt.Errorf("at least one test is required")
	}

	// Validate each test
	for i, test := range suite.Tests {
		if test.Name == "" {
			return fmt.Errorf("test %d: name is required", i)
		}
		if test.Input == "" {
			return fmt.Errorf("test '%s': input is required", test.Name)
		}
		if test.Expect.Type == "" {
			return fmt.Errorf("test '%s': expect.type is required", test.Name)
		}

		// Validate expectation based on type
		switch test.Expect.Type {
		case "exact":
			if test.Expect.Value == "" {
				return fmt.Errorf("test '%s': expect.value is required for 'exact' type", test.Name)
			}
		case "contains":
			if len(test.Expect.Values) == 0 {
				return fmt.Errorf("test '%s': expect.values is required for 'contains' type", test.Name)
			}
		case "regex":
			if test.Expect.Pattern == "" {
				return fmt.Errorf("test '%s': expect.pattern is required for 'regex' type", test.Name)
			}
		case "semantic":
			if test.Expect.Value == "" {
				return fmt.Errorf("test '%s': expect.value is required for 'semantic' type", test.Name)
			}
		}
	}

	return nil
}
