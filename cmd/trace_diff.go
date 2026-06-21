package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// diffMetric is a single comparable metric between two runs.
type diffMetric struct {
	Label       string
	A, B        float64
	LowerBetter bool             // a lower B than A is an improvement
	Colorize    bool             // whether the delta should be colored good/bad
	Format      func(float64) string
}

// diffCmd compares two trace runs to answer "did my change help?".
var diffCmd = &cobra.Command{
	Use:   "diff [run-a] [run-b]",
	Short: "Compare two trace runs (duration, tokens, cost, ...)",
	Long: `Compare two trace runs and show the deltas for duration, spans, LLM calls,
tokens, and estimated cost.

Run selection:
  agk trace diff                 # compare the two most recent runs
  agk trace diff <run-a>         # compare <run-a> (baseline) against the latest run
  agk trace diff <run-a> <run-b> # compare two explicit runs

The first run is treated as the baseline (A); the second is the new run (B).
For duration, tokens, and cost, a lower value in B is shown as an improvement.`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTraceDiff(args)
	},
}

func init() {
	traceCmd.AddCommand(diffCmd)
}

func runTraceDiff(args []string) error {
	idA, idB, err := resolveDiffRuns(args)
	if err != nil {
		return err
	}

	for _, id := range []string{idA, idB} {
		if _, err := os.Stat(filepath.Join(runsDirName, id)); err != nil {
			return fmt.Errorf("trace not found: %s", id)
		}
	}

	manifestA, err := readManifest(filepath.Join(runsDirName, idA))
	if err != nil {
		return fmt.Errorf("failed to read run %s: %w", idA, err)
	}
	manifestB, err := readManifest(filepath.Join(runsDirName, idB))
	if err != nil {
		return fmt.Errorf("failed to read run %s: %w", idB, err)
	}

	printDiff(manifestA, manifestB)
	return nil
}

// resolveDiffRuns determines the two run IDs to compare based on the args provided.
func resolveDiffRuns(args []string) (string, string, error) {
	switch len(args) {
	case 2:
		return args[0], args[1], nil
	case 1:
		latest := ""
		for _, id := range recentRunIDs(runsDirName) {
			if id != args[0] {
				latest = id
				break
			}
		}
		if latest == "" {
			return "", "", fmt.Errorf("need a second run to diff against %s", args[0])
		}
		return args[0], latest, nil
	default: // 0 args
		ids := recentRunIDs(runsDirName)
		if len(ids) < 2 {
			return "", "", fmt.Errorf("need at least two trace runs to diff (found %d)", len(ids))
		}
		// ids[0] is newest; baseline is the older of the two most recent.
		return ids[1], ids[0], nil
	}
}

// recentRunIDs returns run directory names sorted newest-first by modification time.
func recentRunIDs(runsDir string) []string {
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return nil
	}
	type run struct {
		name string
		mod  time.Time
	}
	var runs []run
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		runs = append(runs, run{e.Name(), info.ModTime()})
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].mod.After(runs[j].mod) })

	ids := make([]string, len(runs))
	for i, r := range runs {
		ids[i] = r.name
	}
	return ids
}

func runDiffMetrics(a, b TraceRun) []diffMetric {
	return []diffMetric{
		{Label: "Duration", A: a.Duration, B: b.Duration, LowerBetter: true, Colorize: true, Format: fmtSeconds},
		{Label: "Spans", A: float64(a.SpanCount), B: float64(b.SpanCount), Format: fmtCount},
		{Label: "LLM Calls", A: float64(a.LLMCalls), B: float64(b.LLMCalls), Format: fmtCount},
		{Label: "Tokens", A: float64(a.TotalTokens), B: float64(b.TotalTokens), LowerBetter: true, Colorize: true, Format: fmtCount},
		{Label: "Est. Cost", A: a.EstimatedCost, B: b.EstimatedCost, LowerBetter: true, Colorize: true, Format: fmtUSD},
	}
}

// deltaDirection returns +1 if B is an improvement over A, -1 if a regression,
// and 0 if unchanged or not colorized.
func deltaDirection(m diffMetric) int {
	d := m.B - m.A
	if d == 0 || !m.Colorize {
		return 0
	}
	if (d < 0) == m.LowerBetter {
		return 1
	}
	return -1
}

func printDiff(a, b TraceRun) {
	fmt.Println()
	color.Cyan("📊 Trace Diff")
	fmt.Printf("  A (baseline): %s\n", a.RunID)
	fmt.Printf("  B (new):      %s\n", b.RunID)
	fmt.Println(strings.Repeat("─", 64))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "METRIC\tA (baseline)\tB (new)\tΔ")
	for _, m := range runDiffMetrics(a, b) {
		delta := formatDelta(m)
		switch deltaDirection(m) {
		case 1:
			delta = color.GreenString(delta)
		case -1:
			delta = color.RedString(delta)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", m.Label, m.Format(m.A), m.Format(m.B), delta)
	}
	_ = w.Flush()
	fmt.Println()
}

func formatDelta(m diffMetric) string {
	d := m.B - m.A
	if d == 0 {
		return "—"
	}
	sign, arrow := "+", "▲"
	if d < 0 {
		sign, arrow = "-", "▼"
	}
	out := fmt.Sprintf("%s%s %s", sign, m.Format(absF(d)), arrow)
	if m.A != 0 {
		out += fmt.Sprintf(" %.0f%%", d/m.A*100)
	}
	return out
}

func fmtSeconds(v float64) string { return fmt.Sprintf("%.2fs", v) }
func fmtCount(v float64) string   { return fmt.Sprintf("%.0f", v) }
func fmtUSD(v float64) string     { return fmt.Sprintf("$%.4f", v) }

func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
