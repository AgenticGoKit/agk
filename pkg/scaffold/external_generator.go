package scaffold

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/agenticgokit/agk/pkg/registry"
)

// ExternalGenerator generates a project from a cached external template
type ExternalGenerator struct {
	Cached *registry.CachedTemplate
}

func NewExternalGenerator(cached *registry.CachedTemplate) *ExternalGenerator {
	return &ExternalGenerator{Cached: cached}
}

func (g *ExternalGenerator) GetMetadata() TemplateMetadata {
	return TemplateMetadata{
		Name:        g.Cached.Name,
		Description: g.Cached.Description,
		Complexity:  "External", // Could come from manifest
		FileCount:   0,          // Could calculate this
		Features:    []string{"External Template", g.Cached.Version},
	}
}

func (g *ExternalGenerator) Generate(ctx context.Context, opts GenerateOptions) error {
	// Create project directory
	if err := os.MkdirAll(opts.ProjectPath, 0750); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	manifest := g.Cached.Manifest
	srcDir := g.Cached.LocalPath

	// Prepare template data
	data := TemplateData{
		ProjectName: opts.ProjectName,
		LLMModel:    getLLMModel(opts.LLMProvider),
		LLMProvider: opts.LLMProvider,
		Description: opts.Description,
		AgentType:   opts.AgentType,
		APIKeyEnv:   getAPIKeyEnv(opts.LLMProvider),
	}

	// Walk through the template directory
	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Skip .git and ignored directories
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}

			// Create directory in destination
			destPath := filepath.Join(opts.ProjectPath, relPath)
			return os.MkdirAll(destPath, 0750)
		}

		// Skip manifest file
		if info.Name() == "agk-template.toml" {
			return nil
		}

		// Skip excluded files (simple check for now)
		if shouldExclude(relPath, manifest.Template.Files.Exclude) {
			return nil
		}

		// Use text/template if it's a .tmpl file or generally text?
		// Usually external templates might just be normal files we treat as templates
		// OR they explicitly have .tmpl extension.
		// For simplicity/power, let's try to render ALL non-binary files.
		// Or stick to .tmpl convention?
		// Most "cookiecutter" style tools render everything.

		destPath := filepath.Join(opts.ProjectPath, relPath)
		// Remove .tmpl extension if present
		destPath = strings.TrimSuffix(destPath, ".tmpl")

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Attempt to render
		rendered, err := renderContent(string(content), data)
		if err != nil {
			// If render fails (e.g. binary file), just copy original
			// Ideally check for binary before rendering
			return os.WriteFile(destPath, content, info.Mode())
		}

		return os.WriteFile(destPath, []byte(rendered), info.Mode())
	})

	return err
}

func renderContent(content string, data TemplateData) (string, error) {
	// Create template with Sprig functions
	tmpl, err := template.New("external").Funcs(sprig.TxtFuncMap()).Parse(content)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func shouldExclude(path string, patterns []string) bool {
	for _, p := range patterns {
		matched, _ := filepath.Match(p, path)
		if matched {
			return true
		}
	}
	return false
}
