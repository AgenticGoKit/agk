package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	VersionLatest = "latest"
)

// CachedTemplate represents a template stored in the local cache.
type CachedTemplate struct {
	Name        string            // Template name from manifest (e.g., "rag-agent")
	Source      string            // Source URL/Path (e.g., "github.com/user/repo")
	Version     string            // Version tag or "latest"
	Description string            // Description from manifest
	LocalPath   string            // Absolute path to the template in cache
	Manifest    *TemplateManifest // Parsed manifest
}

// CacheManager handles local storage of templates.
type CacheManager struct {
	BaseDir string // Root cache directory (e.g., ~/.agk/templates)
}

// NewCacheManager creates a new cache manager.
// If baseDir is empty, it defaults to ~/.agk/templates.
func NewCacheManager(baseDir string) (*CacheManager, error) {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		baseDir = filepath.Join(home, ".agk", "templates")
	}

	if err := os.MkdirAll(baseDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create cache directory %s: %w", baseDir, err)
	}

	return &CacheManager{BaseDir: baseDir}, nil
}

// GetPath returns the expected local path for a given source and version.
// Source should be a clean URL path like "github.com/user/repo".
// If version is empty, it uses "latest".
func (c *CacheManager) GetPath(source, version string) string {
	if version == "" {
		version = VersionLatest
	}
	// Clean source to avoid path traversal
	source = filepath.Clean(source)
	source = strings.TrimPrefix(source, "/")
	source = strings.TrimPrefix(source, "\\")

	// Example: ~/.agk/templates/github.com/user/repo/v1.0.0
	return filepath.Join(c.BaseDir, source, version)
}

// List returns all templates currently in the cache.
func (c *CacheManager) List() ([]CachedTemplate, error) {
	var templates []CachedTemplate

	// Walk the cache directory to find agk-template.toml files
	// We expect structure: BaseDir/HOST/jUSER/REPO/VERSION/agk-template.toml
	err := filepath.Walk(c.BaseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Skip dot directories like .git
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}

		if info.Name() == "agk-template.toml" {
			// Found a manifest, parse it
			manifest, err := ParseManifest(path)
			if err != nil {
				// Log error but continue? For now just skip
				return nil
			}

			// Determine source and version from path relative to BaseDir
			relPath, err := filepath.Rel(c.BaseDir, filepath.Dir(path))
			if err != nil {
				return nil
			}

			// Expected relPath: source/version
			// We take the parent of directory as source, and directory name as version
			version := filepath.Base(relPath)
			source := filepath.Dir(relPath)

			templates = append(templates, CachedTemplate{
				Name:        manifest.Template.Name,
				Source:      filepath.ToSlash(source),
				Version:     version,
				Description: manifest.Template.Description,
				LocalPath:   filepath.Dir(path),
				Manifest:    manifest,
			})
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk cache directory: %w", err)
	}

	// Sort by name
	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Name < templates[j].Name
	})

	return templates, nil
}

// Remove deletes a template from the cache.
// Source should include the domain, e.g., "github.com/user/repo".
// If version is provided, only that version is removed.
// If version is empty, ALL versions of that template are removed.
func (c *CacheManager) Remove(source, version string) error {
	path := c.GetPath(source, version)

	if version == "" {
		// Remove all versions: ~/.agk/templates/github.com/user/repo
		// Note: GetPath appends /latest if version is empty, so we need to grab Dir of that
		path = filepath.Dir(path)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("template not found: %s", path)
	}

	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to remove template %s: %w", path, err)
	}

	return nil
}

// Clear removes all cached templates.
func (c *CacheManager) Clear() error {
	if err := os.RemoveAll(c.BaseDir); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}
	return os.MkdirAll(c.BaseDir, 0750)
}
