package registry

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFakeTemplate creates a cached template at BaseDir/<source>/<version>/agk-template.toml.
func writeFakeTemplate(t *testing.T, baseDir, source, version, name string) {
	t.Helper()
	dir := filepath.Join(baseDir, filepath.FromSlash(source), version)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	manifest := "[template]\nname = \"" + name + "\"\nversion = \"" + version + "\"\ndescription = \"test template\"\n"
	if err := os.WriteFile(filepath.Join(dir, "agk-template.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func newTestCache(t *testing.T) *CacheManager {
	t.Helper()
	cm, err := NewCacheManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewCacheManager: %v", err)
	}
	return cm
}

func TestFindByName(t *testing.T) {
	cm := newTestCache(t)
	writeFakeTemplate(t, cm.BaseDir, "github.com/acme/rag", "latest", "rag-agent")
	writeFakeTemplate(t, cm.BaseDir, "github.com/acme/chat", "latest", "chat-agent")

	// Match by manifest name.
	byName, err := cm.FindByName("rag-agent")
	if err != nil {
		t.Fatalf("FindByName: %v", err)
	}
	if len(byName) != 1 || byName[0].Name != "rag-agent" {
		t.Fatalf("byName = %+v, want one rag-agent", byName)
	}

	// Match by source path.
	bySource, err := cm.FindByName("github.com/acme/chat")
	if err != nil {
		t.Fatalf("FindByName: %v", err)
	}
	if len(bySource) != 1 || bySource[0].Name != "chat-agent" {
		t.Fatalf("bySource = %+v, want one chat-agent", bySource)
	}

	// No match.
	none, err := cm.FindByName("does-not-exist")
	if err != nil {
		t.Fatalf("FindByName: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("expected no matches, got %+v", none)
	}
}

func TestRemoveByName(t *testing.T) {
	cm := newTestCache(t)
	writeFakeTemplate(t, cm.BaseDir, "github.com/acme/rag", "latest", "rag-agent")
	writeFakeTemplate(t, cm.BaseDir, "github.com/acme/rag", "v1.0.0", "rag-agent")

	// Two cached versions share the same manifest name; both should be removed.
	n, err := cm.RemoveByName("rag-agent")
	if err != nil {
		t.Fatalf("RemoveByName: %v", err)
	}
	if n != 2 {
		t.Fatalf("removed %d, want 2", n)
	}

	remaining, err := cm.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected empty cache, got %+v", remaining)
	}
}

func TestRemoveByNameNotFound(t *testing.T) {
	cm := newTestCache(t)
	if _, err := cm.RemoveByName("ghost"); err == nil {
		t.Fatal("expected error removing a non-existent template")
	}
}
