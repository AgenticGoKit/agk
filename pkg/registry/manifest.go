// Package registry provides template registry functionality for AGK.
// It handles fetching, caching, and validating templates from various sources
// including GitHub repositories, local paths, and the official AGK registry.
package registry

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// TemplateManifest represents the agk-template.toml file structure.
// This is the main configuration file for AGK templates.
type TemplateManifest struct {
	Template TemplateInfo `toml:"template"`
}

// TemplateInfo contains metadata and configuration for a template.
type TemplateInfo struct {
	// Basic metadata
	Name        string `toml:"name"`
	Version     string `toml:"version"`
	Description string `toml:"description"`
	Author      string `toml:"author"`
	License     string `toml:"license"`

	// Compatibility
	MinAGKVersion string `toml:"min_agk_version"`

	// Template variables that users can customize
	Variables map[string]Variable `toml:"variables"`

	// File inclusion/exclusion rules
	Files FileConfig `toml:"files"`

	// Hooks to run after template generation
	Hooks HookConfig `toml:"hooks"`

	// Dependencies required by generated project
	Dependencies map[string]string `toml:"dependencies"`
}

// Variable defines a template variable that can be customized during init.
type Variable struct {
	Type        string   `toml:"type"` // "string", "bool", "choice"
	Description string   `toml:"description"`
	Required    bool     `toml:"required"`
	Default     any      `toml:"default"`
	Options     []string `toml:"options"` // For "choice" type
}

// FileConfig specifies which files to include/exclude from the template.
type FileConfig struct {
	Include []string `toml:"include"` // Glob patterns to include
	Exclude []string `toml:"exclude"` // Glob patterns to exclude
}

// HookConfig defines commands to run after template generation.
type HookConfig struct {
	PostCreate []string `toml:"post_create"` // Commands like "go mod tidy"
}

// ParseManifest reads and parses an agk-template.toml file.
func ParseManifest(path string) (*TemplateManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	return ParseManifestData(data)
}

// ParseManifestData parses manifest content from bytes.
func ParseManifestData(data []byte) (*TemplateManifest, error) {
	var manifest TemplateManifest
	if err := toml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

// FindManifest searches for agk-template.toml in the given directory.
func FindManifest(dir string) (string, error) {
	manifestPath := filepath.Join(dir, "agk-template.toml")
	if _, err := os.Stat(manifestPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no agk-template.toml found in %s", dir)
		}
		return "", fmt.Errorf("failed to check manifest: %w", err)
	}
	return manifestPath, nil
}

// Validate checks if the manifest is valid.
func (m *TemplateManifest) Validate() error {
	if m.Template.Name == "" {
		return fmt.Errorf("template name is required")
	}
	if m.Template.Version == "" {
		return fmt.Errorf("template version is required")
	}

	// Validate variables
	for name, v := range m.Template.Variables {
		if err := validateVariable(name, v); err != nil {
			return err
		}
	}

	return nil
}

// validateVariable checks if a variable definition is valid.
func validateVariable(name string, v Variable) error {
	validTypes := map[string]bool{
		"string": true,
		"bool":   true,
		"choice": true,
	}

	if !validTypes[v.Type] {
		return fmt.Errorf("variable %q has invalid type %q (must be string, bool, or choice)", name, v.Type)
	}

	if v.Type == "choice" && len(v.Options) == 0 {
		return fmt.Errorf("variable %q is type 'choice' but has no options", name)
	}

	return nil
}

// GetVariable returns a variable by name, or nil if not found.
func (m *TemplateManifest) GetVariable(name string) *Variable {
	if v, ok := m.Template.Variables[name]; ok {
		return &v
	}
	return nil
}

// HasHooks returns true if the manifest defines any hooks.
func (m *TemplateManifest) HasHooks() bool {
	return len(m.Template.Hooks.PostCreate) > 0
}
