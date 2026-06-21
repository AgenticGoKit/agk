package pricing

import "testing"

func TestEstimate(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		in       int
		out      int
		want     float64
		matched  bool
		approxOK bool // compare with tolerance instead of exact
	}{
		{name: "gpt-4o exact", model: "gpt-4o", in: 1_000_000, out: 1_000_000, want: 12.50, matched: true},
		{name: "gpt-4o-mini", model: "gpt-4o-mini", in: 1_000_000, out: 0, want: 0.15, matched: true},
		{name: "gpt-4o dated suffix", model: "gpt-4o-2024-08-06", in: 1_000_000, out: 0, want: 2.50, matched: true},
		{name: "longest prefix prefers mini", model: "gpt-4o-mini-2024-07-18", in: 1_000_000, out: 0, want: 0.15, matched: true},
		{name: "provider prefix stripped", model: "openai/gpt-4o", in: 0, out: 1_000_000, want: 10.00, matched: true},
		{name: "claude sonnet 4 dated", model: "claude-sonnet-4-20250514", in: 1_000_000, out: 0, want: 3.00, matched: true},
		{name: "case insensitive", model: "GPT-4O", in: 1_000_000, out: 0, want: 2.50, matched: true},
		{name: "zero tokens still matched", model: "gpt-4o", in: 0, out: 0, want: 0, matched: true},
		{name: "local model unknown", model: "llama3.2", in: 1_000_000, out: 1_000_000, want: 0, matched: false},
		{name: "empty model", model: "", in: 100, out: 100, want: 0, matched: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, matched := Estimate(tc.model, tc.in, tc.out)
			if matched != tc.matched {
				t.Fatalf("matched = %v, want %v", matched, tc.matched)
			}
			if diff := got - tc.want; diff > 1e-9 || diff < -1e-9 {
				t.Fatalf("cost = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEstimatePartialTokens(t *testing.T) {
	// 1,500 input + 500 output on gpt-4o:
	// 1500/1e6*2.50 + 500/1e6*10.00 = 0.00375 + 0.005 = 0.00875
	got, matched := Estimate("gpt-4o", 1500, 500)
	if !matched {
		t.Fatal("expected gpt-4o to match")
	}
	want := 0.00875
	if diff := got - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("cost = %v, want %v", got, want)
	}
}
