package registry

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	FetcherTypeGit   = "git"
	FetcherTypeLocal = "local"
)

// Resolver handles resolving template references to cached templates.
// It orchestrates fetching and caching.
type Resolver struct {
	cache    *CacheManager
	fetchers map[string]Fetcher // "git", "local"
}

// NewResolver creates a new template resolver.
func NewResolver(cache *CacheManager) *Resolver {
	return &Resolver{
		cache: cache,
		fetchers: map[string]Fetcher{
			FetcherTypeGit:   &GitFetcher{},
			FetcherTypeLocal: &LocalFetcher{},
		},
	}
}

// Resolve locates a template, fetching it if necessary, and returns the cached template.
// Source can be:
// - GitHub URL: github.com/user/repo or https://github.com/user/repo
// - Versioned: github.com/user/repo@v1.0.0
// - Local path: ./my-template or /abs/path/to/template
// Resolve locates a template, fetching it if necessary, and returns the cached template.
// Source can be:
// - GitHub URL: github.com/user/repo or https://github.com/user/repo
// - Versioned: github.com/user/repo@v1.0.0
// - Local path: ./my-template or /abs/path/to/template
func (r *Resolver) Resolve(ctx context.Context, sourceRef string) (*CachedTemplate, error) {
	source, version := parseSourceRef(sourceRef)
	isLocal := isLocalPath(source)

	fetcherType, resolvedSource, err := r.resolveFetcherType(source, isLocal)
	if err != nil {
		return nil, err
	}
	source = resolvedSource

	// Determine cache path
	cacheKey := source
	if isLocal {
		cacheKey = "local/" + filepath.Base(source)
	}

	destPath := r.cache.GetPath(cacheKey, version)

	// Check if exists in cache
	if _, err := ParseManifest(filepath.Join(destPath, "agk-template.toml")); err == nil {
		return r.loadFromCache(destPath, cacheKey, version)
	}

	// Fetch it
	fetcher, ok := r.fetchers[fetcherType]
	if !ok {
		return nil, fmt.Errorf("no fetcher for type %s", fetcherType)
	}

	if err := fetcher.Fetch(ctx, source, version, destPath); err != nil {
		return nil, fmt.Errorf("failed to fetch template: %w", err)
	}

	return r.loadFromCache(destPath, cacheKey, version)
}

func (r *Resolver) resolveFetcherType(source string, isLocal bool) (string, string, error) {
	if isLocal {
		absPath, err := filepath.Abs(source)
		if err == nil {
			source = absPath
		}
		return FetcherTypeLocal, source, nil
	}

	// Check if valid URL or git source
	if strings.Contains(source, "://") || strings.HasPrefix(source, "git@") || strings.Contains(source, "github.com") {
		return FetcherTypeGit, source, nil
	}

	// Try registry lookup
	return r.resolveFromRegistry(source)
}

func (r *Resolver) resolveFromRegistry(source string) (string, string, error) {
	index, err := FetchIndex(DefaultRegistryURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch registry index to resolve '%s': %w", source, err)
	}

	if repoURL, ok := index.Templates[source]; ok {
		return FetcherTypeGit, repoURL, nil
	}

	if strings.HasPrefix(source, "agk/") {
		stripped := strings.TrimPrefix(source, "agk/")
		if repoURL, ok := index.Templates[stripped]; ok {
			return FetcherTypeGit, repoURL, nil
		}
		return "", "", fmt.Errorf("template '%s' (nor '%s') not found in registry", source, stripped)
	}

	return "", "", fmt.Errorf("template '%s' not found in registry and is not a valid URL or local path", source)
}

func (r *Resolver) loadFromCache(path, source, version string) (*CachedTemplate, error) {
	manifest, err := ParseManifest(filepath.Join(path, "agk-template.toml"))
	if err != nil {
		return nil, fmt.Errorf("invalid template (missing or invalid agk-template.toml): %w", err)
	}

	return &CachedTemplate{
		Name:        manifest.Template.Name,
		Source:      source,
		Version:     version,
		Description: manifest.Template.Description,
		LocalPath:   path,
		Manifest:    manifest,
	}, nil
}

// parseSourceRef splits "source@version" into "source" and "version"
func parseSourceRef(ref string) (string, string) {
	parts := strings.Split(ref, "@")
	if len(parts) > 1 {
		// Handle cases like "git@github.com:..." where @ is part of auth
		// If using https/github.com style, last @ is version
		lastIdx := strings.LastIndex(ref, "@")
		if lastIdx > 0 {
			// Check if it looks like git user? git@...
			// If contains / and @ is before /, it's auth.
			// Version @ is usually at the end.
			return ref[:lastIdx], ref[lastIdx+1:]
		}
	}
	return ref, VersionLatest
}

func isLocalPath(s string) bool {
	return strings.HasPrefix(s, ".") ||
		strings.HasPrefix(s, "/") ||
		strings.HasPrefix(s, "\\") ||
		filepath.IsAbs(s) ||
		strings.Contains(s, string(filepath.Separator)) && !strings.Contains(s, "://") && !strings.HasPrefix(s, "github.com")
}
