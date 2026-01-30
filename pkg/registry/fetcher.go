package registry

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Fetcher defines the interface for fetching templates.
type Fetcher interface {
	// Fetch downloads the template from source/version to dest directory.
	Fetch(ctx context.Context, source, version, dest string) error
}

// GitFetcher downloads templates from Git repositories.
type GitFetcher struct{}

// Fetch implements Fetcher for Git repositories.
// It supports cloning specific tags or the latest default branch.
func (f *GitFetcher) Fetch(ctx context.Context, source, version, dest string) error {
	// Ensure destination directory doesn't exist to avoid git clone errors
	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("failed to clear destination: %w", err)
	}

	// Construct Git URL
	// Simple heuristic: if it looks like github.com/user/repo, add https://
	url := source
	if !strings.Contains(url, "://") && !strings.HasPrefix(url, "git@") {
		url = "https://" + url
	}

	cloneOpts := &git.CloneOptions{
		URL:      url,
		Progress: os.Stdout, // Should ideally be controlled by logger/context
		Depth:    1,         // Default to shallow clone
		Tags:     git.NoTags,
	}

	// If version is specified and not "latest", try to checkout that tag
	if version != "" && version != "latest" {
		cloneOpts.ReferenceName = plumbing.ReferenceName("refs/tags/" + version)
		cloneOpts.SingleBranch = true
		cloneOpts.Depth = 1 // Shallow clone of tag is supported
	}

	// Perform clone
	_, err := git.PlainCloneContext(ctx, dest, false, cloneOpts)
	if err != nil {
		// Fallback: If tag checkout failed, maybe try full clone then checkout?
		// But for now return error.
		return fmt.Errorf("git clone failed for %s@%s: %w", url, version, err)
	}

	// Cleanup .git directory as we don't need history in the template cache
	if err := os.RemoveAll(filepath.Join(dest, ".git")); err != nil {
		return fmt.Errorf("failed to remove .git directory: %w", err)
	}

	return nil
}

// LocalFetcher copies templates from a local path.
type LocalFetcher struct{}

// Fetch implements Fetcher for local paths.
// Source is assumed to be an absolute or relative file path.
// Version is ignored for local paths.
func (f *LocalFetcher) Fetch(ctx context.Context, source, version, dest string) error {
	// Ensure source exists
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("local source not found: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("local source %s is not a directory", source)
	}

	// Ensure destination directory doesn't exist
	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("failed to clear destination: %w", err)
	}

	// Copy directory
	return copyDir(source, dest)
}

// copyDir recursively copies a directory tree, attempting to preserve permissions.
func copyDir(src string, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Construct relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			// Create directory
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		if err := copyFile(path, dstPath, info.Mode()); err != nil {
			return fmt.Errorf("failed to copy file %s: %w", path, err)
		}
		return nil
	})

	return err
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return os.Chmod(dst, mode)
}
