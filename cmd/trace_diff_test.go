package cmd

import "testing"

func TestDeltaDirection(t *testing.T) {
	cases := []struct {
		name string
		m    diffMetric
		want int
	}{
		{"lower-better improvement", diffMetric{A: 10, B: 5, LowerBetter: true, Colorize: true}, 1},
		{"lower-better regression", diffMetric{A: 5, B: 10, LowerBetter: true, Colorize: true}, -1},
		{"equal is neutral", diffMetric{A: 7, B: 7, LowerBetter: true, Colorize: true}, 0},
		{"not colorized is neutral", diffMetric{A: 10, B: 5, LowerBetter: true, Colorize: false}, 0},
		{"higher-better improvement", diffMetric{A: 5, B: 10, LowerBetter: false, Colorize: true}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := deltaDirection(c.m); got != c.want {
				t.Errorf("deltaDirection = %d, want %d", got, c.want)
			}
		})
	}
}

func TestRunDiffMetrics(t *testing.T) {
	a := TraceRun{Duration: 2.0, SpanCount: 5, LLMCalls: 2, TotalTokens: 1000, EstimatedCost: 0.0100}
	b := TraceRun{Duration: 1.0, SpanCount: 4, LLMCalls: 1, TotalTokens: 500, EstimatedCost: 0.0050}

	metrics := runDiffMetrics(a, b)
	if len(metrics) != 5 {
		t.Fatalf("expected 5 metrics, got %d", len(metrics))
	}

	byLabel := make(map[string]diffMetric, len(metrics))
	for _, m := range metrics {
		byLabel[m.Label] = m
	}

	// Tokens halved → improvement.
	if dir := deltaDirection(byLabel["Tokens"]); dir != 1 {
		t.Errorf("Tokens direction = %d, want 1 (improvement)", dir)
	}
	// Cost halved → improvement.
	if dir := deltaDirection(byLabel["Est. Cost"]); dir != 1 {
		t.Errorf("Cost direction = %d, want 1 (improvement)", dir)
	}
	// Spans is not colorized → neutral even though it changed.
	if dir := deltaDirection(byLabel["Spans"]); dir != 0 {
		t.Errorf("Spans direction = %d, want 0 (neutral)", dir)
	}
}

func TestFormatDelta(t *testing.T) {
	if got := formatDelta(diffMetric{A: 10, B: 10, Format: fmtCount}); got != "—" {
		t.Errorf("equal delta = %q, want em dash", got)
	}
	// 1000 -> 500 tokens: -500, 50% lower.
	got := formatDelta(diffMetric{A: 1000, B: 500, Format: fmtCount})
	if got != "-500 ▼ -50%" {
		t.Errorf("formatDelta = %q, want %q", got, "-500 ▼ -50%")
	}
}

func TestResolveDiffRunsExplicit(t *testing.T) {
	a, b, err := resolveDiffRuns([]string{"run-1", "run-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a != "run-1" || b != "run-2" {
		t.Errorf("got (%s, %s), want (run-1, run-2)", a, b)
	}
}
