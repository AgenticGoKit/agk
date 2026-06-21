package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTraceTailer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.jsonl")
	if err := os.WriteFile(path, []byte("line1\nline2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tl := &traceTailer{path: path}

	// First poll returns both complete lines.
	if got := tl.poll(); len(got) != 2 || got[0] != "line1" || got[1] != "line2" {
		t.Fatalf("first poll = %v, want [line1 line2]", got)
	}

	// Nothing new yet.
	if got := tl.poll(); len(got) != 0 {
		t.Fatalf("expected no new lines, got %v", got)
	}

	// A partial line (no newline) is buffered, not emitted.
	appendString(t, path, "partial")
	if got := tl.poll(); len(got) != 0 {
		t.Fatalf("partial line should buffer, got %v", got)
	}

	// Completing the line emits it whole.
	appendString(t, path, "-done\n")
	if got := tl.poll(); len(got) != 1 || got[0] != "partial-done" {
		t.Fatalf("poll = %v, want [partial-done]", got)
	}
}

func TestFormatWatchSpan(t *testing.T) {
	span := map[string]interface{}{
		"Name":      "agk.llm.generate",
		"StartTime": "2026-06-22T10:00:00Z",
		"EndTime":   "2026-06-22T10:00:01Z",
	}
	line := formatWatchSpan(span)
	if !strings.Contains(line, "agk.llm.generate") {
		t.Errorf("missing span name: %q", line)
	}
	if !strings.Contains(line, "1000ms") {
		t.Errorf("expected 1000ms duration: %q", line)
	}
	if !strings.Contains(line, "🤖") {
		t.Errorf("expected LLM icon: %q", line)
	}
}

func TestWatchIconAndTruncate(t *testing.T) {
	if watchIcon("agk.tool.call") != "🔧" {
		t.Error("tool icon mismatch")
	}
	if watchIcon("agk.workflow.run") != "🔀" {
		t.Error("workflow icon mismatch")
	}
	if got := truncateName("abcdefghij", 5); got != "abcd…" {
		t.Errorf("truncateName = %q, want abcd…", got)
	}
	if got := truncateName("short", 10); got != "short" {
		t.Errorf("truncateName should not change short strings, got %q", got)
	}
}

func appendString(t *testing.T, path, s string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(s); err != nil {
		t.Fatal(err)
	}
}
