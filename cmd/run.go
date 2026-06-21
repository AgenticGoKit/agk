package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

var (
	runNoTrace    bool
	runTraceLevel string
	runWatchMode  bool
)

// runCmd builds and runs an AgenticGoKit project with tracing enabled,
// then prints a trace summary so the develop→run→observe loop stays in one tool.
var runCmd = &cobra.Command{
	Use:   "run [path]",
	Short: "Build and run an AgenticGoKit project with tracing enabled",
	Long: `Build and run an AgenticGoKit project (go run .) with tracing enabled.

By default this sets AGK_TRACE=true and AGK_TRACE_EXPORTER=file so the run is
captured to .agk/runs/<run-id>/. When the program exits, AGK prints a short trace
summary and a hint to open the interactive viewer.

Examples:
  # Run the project in the current directory
  agk run

  # Run a project in another directory
  agk run ./my-agent

  # Capture full prompts/responses for deep debugging
  agk run --trace-level detailed

  # Re-run automatically when .go files change
  agk run --watch

  # Run without tracing
  agk run --no-trace

Note: trace inspection commands (agk trace ...) read from the current directory's
.agk/runs. Run them from the same directory as the project for best results.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runProject,
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().BoolVar(&runNoTrace, "no-trace", false, "Disable automatic tracing")
	runCmd.Flags().StringVar(&runTraceLevel, "trace-level", "standard",
		"Trace detail level: minimal|standard|detailed")
	runCmd.Flags().BoolVarP(&runWatchMode, "watch", "w", false, "Re-run on .go file changes")
}

func runProject(_ *cobra.Command, args []string) error {
	projectPath := "."
	if len(args) > 0 {
		projectPath = args[0]
	}

	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	if err := validateGoProject(absPath); err != nil {
		color.Red("✗ %v", err)
		return err
	}

	if err := validateTraceLevel(runTraceLevel); err != nil {
		color.Red("✗ %v", err)
		return err
	}

	if runWatchMode {
		return runWithWatch(absPath)
	}
	return runOnce(absPath)
}

// validateGoProject ensures the target directory looks like a runnable Go module.
func validateGoProject(dir string) error {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("not a directory: %s", dir)
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		return fmt.Errorf("no go.mod found in %s (run 'agk init' first)", dir)
	}
	return nil
}

func validateTraceLevel(level string) error {
	switch level {
	case "minimal", "standard", "detailed":
		return nil
	default:
		return fmt.Errorf("invalid trace level %q (valid: minimal, standard, detailed)", level)
	}
}

// projectEnv returns the environment for the child process, layering tracing on top
// of the current environment unless tracing is disabled.
func projectEnv() []string {
	env := os.Environ()
	if runNoTrace {
		return env
	}
	env = append(env,
		"AGK_TRACE=true",
		"AGK_TRACE_LEVEL="+runTraceLevel,
		"AGK_TRACE_EXPORTER=file",
	)
	return env
}

// execGoRun runs `go run .` in dir with the configured environment, wiring the child
// process to the parent's stdio. The context cancels the run (used by watch mode and
// Ctrl+C handling).
func execGoRun(ctx context.Context, dir string) error {
	c := exec.CommandContext(ctx, "go", "run", ".")
	c.Dir = dir
	c.Env = projectEnv()
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func runOnce(dir string) error {
	runsDir := filepath.Join(dir, runsDirName)
	before := snapshotRuns(runsDir)

	color.Cyan("▶  Running project: %s", dir)
	if !runNoTrace {
		color.HiBlack("   tracing: on (level=%s) → %s", runTraceLevel, runsDir)
	}
	fmt.Println()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runErr := execGoRun(ctx, dir)
	if runErr != nil {
		color.Red("\n✗ Project exited with error: %v", runErr)
	} else {
		color.Green("\n✓ Project finished")
	}

	printNewTraceSummary(runsDir, before)
	return runErr
}

// snapshotRuns records the set of run directories present before a run so we can
// identify the run produced by this invocation afterwards.
func snapshotRuns(runsDir string) map[string]bool {
	seen := map[string]bool{}
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return seen
	}
	for _, e := range entries {
		if e.IsDir() {
			seen[e.Name()] = true
		}
	}
	return seen
}

// printNewTraceSummary finds the run directory created since `before` and prints a
// compact summary plus a hint to open the interactive viewer.
func printNewTraceSummary(runsDir string, before map[string]bool) {
	if runNoTrace {
		return
	}

	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return
	}

	type runEntry struct {
		name    string
		modTime time.Time
	}
	var newRuns []runEntry
	for _, e := range entries {
		if !e.IsDir() || before[e.Name()] {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		newRuns = append(newRuns, runEntry{name: e.Name(), modTime: info.ModTime()})
	}

	if len(newRuns) == 0 {
		color.HiBlack("\n(no trace captured — the project may not use AgenticGoKit observability)")
		return
	}

	// Newest first.
	sort.Slice(newRuns, func(i, j int) bool {
		return newRuns[i].modTime.After(newRuns[j].modTime)
	})

	runID := newRuns[0].name
	manifest, err := readManifest(filepath.Join(runsDir, runID))
	if err != nil {
		return
	}
	printRunSummary(manifest)
}

func printRunSummary(m TraceRun) {
	bar := strings.Repeat("─", 60)
	fmt.Println()
	color.HiBlack(bar)
	color.Cyan("📊 Trace Summary")
	fmt.Printf("   Run ID:      %s\n", m.RunID)
	fmt.Printf("   Duration:    %.2fs\n", m.Duration)
	fmt.Printf("   Spans:       %d\n", m.SpanCount)
	fmt.Printf("   LLM Calls:   %d\n", m.LLMCalls)
	fmt.Printf("   Tokens:      %d\n", m.TotalTokens)
	if m.EstimatedCost > 0 {
		fmt.Printf("   Est. Cost:   $%.4f\n", m.EstimatedCost)
	}
	color.HiBlack(bar)
	fmt.Printf("→ Inspect: %s\n", color.CyanString("agk trace view %s", m.RunID))
}

// runWithWatch re-runs the project whenever a .go file changes. Each change cancels
// the in-flight run and starts a fresh one (debounced to coalesce rapid saves).
func runWithWatch(dir string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	defer func() { _ = watcher.Close() }()

	if err := addGoDirs(watcher, dir); err != nil {
		return fmt.Errorf("failed to watch project: %w", err)
	}

	color.Cyan("👀 Watch mode — re-running on .go changes (Ctrl+C to stop)")
	if !runNoTrace {
		color.HiBlack("   tracing: on (level=%s)", runTraceLevel)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	var (
		mu     sync.Mutex
		cancel context.CancelFunc
	)

	startRun := func() {
		mu.Lock()
		defer mu.Unlock()
		if cancel != nil {
			cancel()
		}
		ctx, c := context.WithCancel(context.Background())
		cancel = c
		go func() {
			runsDir := filepath.Join(dir, runsDirName)
			before := snapshotRuns(runsDir)
			fmt.Println()
			color.Cyan("▶  Running...")
			err := execGoRun(ctx, dir)
			if ctx.Err() != nil {
				return // superseded by a newer run
			}
			if err != nil {
				color.Red("✗ exited: %v", err)
			}
			printNewTraceSummary(runsDir, before)
			color.HiBlack("\n— waiting for changes —")
		}()
	}

	startRun()

	var debounce *time.Timer
	for {
		select {
		case <-sigCh:
			mu.Lock()
			if cancel != nil {
				cancel()
			}
			mu.Unlock()
			fmt.Println("\n👋 Stopped")
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if filepath.Ext(event.Name) != ".go" {
				continue
			}
			if debounce != nil {
				debounce.Stop()
			}
			name := filepath.Base(event.Name)
			debounce = time.AfterFunc(300*time.Millisecond, func() {
				color.HiBlack("\n♻  change detected: %s", name)
				startRun()
			})

		case werr, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			color.Red("watch error: %v", werr)
		}
	}
}

// addGoDirs registers dir and its subdirectories with the watcher, skipping hidden
// directories, vendor, and node_modules.
func addGoDirs(w *fsnotify.Watcher, root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if path != root && (strings.HasPrefix(base, ".") || base == "vendor" || base == "node_modules") {
			return filepath.SkipDir
		}
		return w.Add(path)
	})
}
