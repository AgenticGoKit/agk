package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// watchCmd live-tails a trace run as it executes, printing each span as it lands.
var watchCmd = &cobra.Command{
	Use:   "watch [run-id]",
	Short: "Live-tail a trace run as it executes",
	Long: `Live-tail a trace run, printing each span as it is appended to trace.jsonl.

With no run ID, watch follows the currently-running run if one is live, otherwise
it waits for the next run to start (e.g. from 'agk run' in another terminal). The
watch ends when the run completes (its manifest.json is written) or on Ctrl+C.

Examples:
  agk trace watch              # follow the live run, or wait for the next one
  agk trace watch <run-id>     # follow a specific run`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := ""
		if len(args) > 0 {
			runID = args[0]
		}
		return watchTrace(runID)
	},
}

func init() {
	traceCmd.AddCommand(watchCmd)
}

func watchTrace(runID string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	resolved, err := resolveWatchRun(ctx, runsDirName, runID)
	if err != nil {
		return err
	}
	if resolved == "" {
		return nil // cancelled while waiting
	}

	return tailRun(ctx, filepath.Join(runsDirName, resolved), resolved)
}

// resolveWatchRun picks the run to follow: an explicit ID, the live latest run, or
// (when the latest run has already completed) the next run to appear.
func resolveWatchRun(ctx context.Context, runsDir, runID string) (string, error) {
	if runID != "" {
		if _, err := os.Stat(filepath.Join(runsDir, runID)); err != nil {
			return "", fmt.Errorf("trace not found: %s", runID)
		}
		return runID, nil
	}

	// A latest run with no manifest yet is still live — follow it.
	if latest := getLatestRunID(); latest != "" {
		if _, err := os.Stat(filepath.Join(runsDir, latest, "manifest.json")); err != nil {
			return latest, nil
		}
	}

	fmt.Println("⏳ Waiting for a new run to start (e.g. `agk run`)... Ctrl+C to stop")
	before := currentRunSet(runsDir)

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return "", nil
		case <-ticker.C:
			for name := range currentRunSet(runsDir) {
				if !before[name] {
					return name, nil
				}
			}
		}
	}
}

func currentRunSet(runsDir string) map[string]bool {
	set := map[string]bool{}
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return set
	}
	for _, e := range entries {
		if e.IsDir() {
			set[e.Name()] = true
		}
	}
	return set
}

func tailRun(ctx context.Context, runPath, runID string) error {
	tracePath := filepath.Join(runPath, "trace.jsonl")
	manifestPath := filepath.Join(runPath, "manifest.json")

	color.Cyan("👀 Watching %s  (Ctrl+C to stop)", runID)
	fmt.Println(strings.Repeat("─", 64))

	tailer := &traceTailer{path: tracePath}
	printNew := func() {
		for _, line := range tailer.poll() {
			var span map[string]interface{}
			if err := json.Unmarshal([]byte(line), &span); err == nil {
				fmt.Println(formatWatchSpan(span))
			}
		}
	}

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			printNew()
			fmt.Println("\n👋 Stopped")
			return nil
		case <-ticker.C:
			printNew()
			if _, err := os.Stat(manifestPath); err == nil {
				printNew() // final drain
				printWatchSummary(runPath, runID)
				return nil
			}
		}
	}
}

func printWatchSummary(runPath, runID string) {
	fmt.Println(strings.Repeat("─", 64))
	if m, err := readManifest(runPath); err == nil {
		color.Green("✅ Run complete — %d spans, %d LLM calls, %d tokens, $%.4f, %.2fs",
			m.SpanCount, m.LLMCalls, m.TotalTokens, m.EstimatedCost, m.Duration)
	}
	fmt.Printf("→ Inspect: %s\n", color.CyanString("agk trace view %s", runID))
}

// traceTailer incrementally reads new complete lines appended to a JSONL file,
// buffering any trailing partial line until it is finished.
type traceTailer struct {
	path   string
	offset int64
	buf    []byte
}

func (t *traceTailer) poll() []string {
	f, err := os.Open(t.path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	if _, err := f.Seek(t.offset, io.SeekStart); err != nil {
		return nil
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return nil
	}
	t.offset += int64(len(data))
	t.buf = append(t.buf, data...)

	var lines []string
	for {
		i := bytes.IndexByte(t.buf, '\n')
		if i < 0 {
			break
		}
		if line := strings.TrimSpace(string(t.buf[:i])); line != "" {
			lines = append(lines, line)
		}
		t.buf = t.buf[i+1:]
	}
	return lines
}

func formatWatchSpan(span map[string]interface{}) string {
	name, _ := span["Name"].(string)
	start := parseSpanTime(span["StartTime"])
	end := parseSpanTime(span["EndTime"])

	var durMs int64
	if !start.IsZero() && !end.IsZero() {
		durMs = end.Sub(start).Milliseconds()
	}
	ts := "--:--:--"
	if !start.IsZero() {
		ts = start.Format("15:04:05")
	}

	return fmt.Sprintf("%s  %s  %-44s %6dms", ts, watchIcon(strings.ToLower(name)), truncateName(name, 44), durMs)
}

func watchIcon(lowerName string) string {
	switch {
	case strings.Contains(lowerName, "llm"):
		return "🤖"
	case strings.Contains(lowerName, "tool"):
		return "🔧"
	case strings.Contains(lowerName, "workflow"):
		return "🔀"
	case strings.Contains(lowerName, "agent"):
		return "👤"
	default:
		return "•"
	}
}

func truncateName(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "…"
}

func parseSpanTime(v interface{}) time.Time {
	s, ok := v.(string)
	if !ok || s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Time{}
}
