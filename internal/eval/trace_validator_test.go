package eval

import (
	"reflect"
	"testing"
)

func TestValidateTrace(t *testing.T) {
	tests := []struct {
		name        string
		exp         *TraceExpectation
		obs         ObservedTrace
		wantFailure bool
	}{
		{
			name: "tool calls present",
			exp:  &TraceExpectation{ToolCalls: []string{"search"}},
			obs:  ObservedTrace{ToolCalls: []string{"search", "calculator"}},
		},
		{
			name:        "tool call missing",
			exp:         &TraceExpectation{ToolCalls: []string{"search", "weather"}},
			obs:         ObservedTrace{ToolCalls: []string{"search"}},
			wantFailure: true,
		},
		{
			name: "llm calls exact match",
			exp:  &TraceExpectation{LLMCalls: 3},
			obs:  ObservedTrace{LLMCalls: 3},
		},
		{
			name:        "llm calls mismatch",
			exp:         &TraceExpectation{LLMCalls: 3},
			obs:         ObservedTrace{LLMCalls: 2},
			wantFailure: true,
		},
		{
			name: "min steps satisfied",
			exp:  &TraceExpectation{MinSteps: 2},
			obs:  ObservedTrace{Steps: 5},
		},
		{
			name:        "min steps violated",
			exp:         &TraceExpectation{MinSteps: 5},
			obs:         ObservedTrace{Steps: 2},
			wantFailure: true,
		},
		{
			name:        "max steps violated",
			exp:         &TraceExpectation{MaxSteps: 3},
			obs:         ObservedTrace{Steps: 7},
			wantFailure: true,
		},
		{
			name: "execution path ordered subsequence",
			exp:  &TraceExpectation{ExecutionPath: []string{"research", "format"}},
			obs:  ObservedTrace{Path: []string{"start", "research", "summarize", "format", "done"}},
		},
		{
			name:        "execution path out of order",
			exp:         &TraceExpectation{ExecutionPath: []string{"format", "research"}},
			obs:         ObservedTrace{Path: []string{"research", "format"}},
			wantFailure: true,
		},
		{
			name: "nil expectation passes",
			exp:  nil,
			obs:  ObservedTrace{},
		},
		{
			name:        "multiple failures combine",
			exp:         &TraceExpectation{ToolCalls: []string{"x"}, LLMCalls: 2, MaxSteps: 1},
			obs:         ObservedTrace{ToolCalls: []string{"y"}, LLMCalls: 5, Steps: 9},
			wantFailure: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			failures := ValidateTrace(tc.exp, tc.obs)
			if got := len(failures) > 0; got != tc.wantFailure {
				t.Fatalf("wantFailure=%v, got failures=%v", tc.wantFailure, failures)
			}
		})
	}
}

func TestBuildObservedTrace(t *testing.T) {
	tr := &evalTrace{
		ID: "trace-1",
		Spans: []*evalSpan{
			{Name: "agk.agent.run"},
			{Name: "agk.llm.generate"},
			{Name: "agk.tool.call", Attributes: map[string]interface{}{"agk.tool.name": "search"}},
			{Name: "agk.llm.generate"},
		},
	}

	obs := buildObservedTrace(tr, []string{"calculator"})

	if obs.LLMCalls != 2 {
		t.Errorf("LLMCalls = %d, want 2", obs.LLMCalls)
	}
	if obs.Steps != 4 {
		t.Errorf("Steps = %d, want 4", obs.Steps)
	}
	// tools_called ("calculator") plus the tool span ("search"), de-duplicated and ordered.
	wantTools := []string{"calculator", "search"}
	if !reflect.DeepEqual(obs.ToolCalls, wantTools) {
		t.Errorf("ToolCalls = %v, want %v", obs.ToolCalls, wantTools)
	}
	wantPath := []string{"agk.agent.run", "agk.llm.generate", "agk.tool.call", "agk.llm.generate"}
	if !reflect.DeepEqual(obs.Path, wantPath) {
		t.Errorf("Path = %v, want %v", obs.Path, wantPath)
	}
}

func TestBuildObservedTraceNilTrace(t *testing.T) {
	obs := buildObservedTrace(nil, []string{"search", "search", ""})
	// Only tools_called is available; duplicates and empties removed.
	if !reflect.DeepEqual(obs.ToolCalls, []string{"search"}) {
		t.Errorf("ToolCalls = %v, want [search]", obs.ToolCalls)
	}
	if obs.Steps != 0 || obs.LLMCalls != 0 {
		t.Errorf("expected zero Steps/LLMCalls with nil trace, got steps=%d llm=%d", obs.Steps, obs.LLMCalls)
	}
}

func TestIsOrderedSubsequence(t *testing.T) {
	cases := []struct {
		want, have []string
		ok         bool
	}{
		{[]string{"a", "c"}, []string{"a", "b", "c"}, true},
		{[]string{"a", "b", "c"}, []string{"a", "b", "c"}, true},
		{[]string{"c", "a"}, []string{"a", "b", "c"}, false},
		{[]string{"a", "d"}, []string{"a", "b", "c"}, false},
		{[]string{}, []string{"a"}, true},
	}
	for _, c := range cases {
		if got := isOrderedSubsequence(c.want, c.have); got != c.ok {
			t.Errorf("isOrderedSubsequence(%v, %v) = %v, want %v", c.want, c.have, got, c.ok)
		}
	}
}
