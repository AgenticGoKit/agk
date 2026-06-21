package eval

import (
	"fmt"
	"strings"
)

// evalTrace mirrors the subset of the EvalServer's GET /traces/{id} response that
// trace assertions need. The server type lives in the framework (v1beta.EvalTrace);
// this is a decode-only copy so the CLI doesn't depend on the framework package.
type evalTrace struct {
	ID    string      `json:"id"`
	Spans []*evalSpan `json:"spans"`
}

type evalSpan struct {
	Name       string                 `json:"name"`
	Attributes map[string]interface{} `json:"attributes"`
}

// ObservedTrace is the normalized view of a run's behavior used for assertions.
type ObservedTrace struct {
	ToolCalls []string // distinct tool names invoked, in first-seen order
	LLMCalls  int      // number of LLM spans
	Path      []string // ordered span names (the execution path)
	Steps     int      // total spans (a proxy for execution steps)
}

// buildObservedTrace normalizes a fetched trace (and the tools_called list from the
// invoke response) into an ObservedTrace. Either source may be empty; tools_called is
// treated as the authoritative tool list and augmented with any tool spans found.
func buildObservedTrace(t *evalTrace, toolsCalled []string) ObservedTrace {
	obs := ObservedTrace{}
	seen := make(map[string]bool)

	addTool := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		obs.ToolCalls = append(obs.ToolCalls, name)
	}

	for _, name := range toolsCalled {
		addTool(name)
	}

	if t != nil {
		for _, sp := range t.Spans {
			if sp == nil {
				continue
			}
			obs.Path = append(obs.Path, sp.Name)

			lname := strings.ToLower(sp.Name)
			if strings.Contains(lname, "llm") {
				obs.LLMCalls++
			}
			if isToolSpan(lname) {
				addTool(toolNameFromSpan(sp))
			}
		}
		obs.Steps = len(t.Spans)
	}

	return obs
}

func isToolSpan(lowerName string) bool {
	return strings.Contains(lowerName, "tool.call") || strings.Contains(lowerName, "tool_call")
}

// toolNameFromSpan extracts a tool name from a tool span's attributes, trying the
// AgenticGoKit key first and a couple of common fallbacks.
func toolNameFromSpan(sp *evalSpan) string {
	for _, key := range []string{"agk.tool.name", "tool.name", "tool"} {
		if v, ok := sp.Attributes[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// ValidateTrace checks an ObservedTrace against a TraceExpectation and returns a list
// of human-readable failure messages (empty means all assertions passed).
//
// Semantics:
//   - tool_calls:     every listed tool must have been called (subset check)
//   - llm_calls:      when > 0, the observed LLM-call count must match exactly
//   - min_steps:      observed step count must be >= min
//   - max_steps:      observed step count must be <= max
//   - execution_path: the listed names must appear, in order, as a subsequence
func ValidateTrace(exp *TraceExpectation, obs ObservedTrace) []string {
	if exp == nil {
		return nil
	}

	var failures []string

	if len(exp.ToolCalls) > 0 {
		have := make(map[string]bool, len(obs.ToolCalls))
		for _, t := range obs.ToolCalls {
			have[t] = true
		}
		var missing []string
		for _, want := range exp.ToolCalls {
			if !have[want] {
				missing = append(missing, want)
			}
		}
		if len(missing) > 0 {
			failures = append(failures, fmt.Sprintf(
				"expected tool call(s) not found: %v (called: %v)", missing, orNone(obs.ToolCalls)))
		}
	}

	if exp.LLMCalls > 0 && obs.LLMCalls != exp.LLMCalls {
		failures = append(failures, fmt.Sprintf(
			"expected %d LLM call(s), observed %d", exp.LLMCalls, obs.LLMCalls))
	}

	if exp.MinSteps > 0 && obs.Steps < exp.MinSteps {
		failures = append(failures, fmt.Sprintf(
			"expected at least %d step(s), observed %d", exp.MinSteps, obs.Steps))
	}

	if exp.MaxSteps > 0 && obs.Steps > exp.MaxSteps {
		failures = append(failures, fmt.Sprintf(
			"expected at most %d step(s), observed %d", exp.MaxSteps, obs.Steps))
	}

	if len(exp.ExecutionPath) > 0 && !isOrderedSubsequence(exp.ExecutionPath, obs.Path) {
		failures = append(failures, fmt.Sprintf(
			"expected execution path %v not found (in order) within observed %v",
			exp.ExecutionPath, orNone(obs.Path)))
	}

	return failures
}

// isOrderedSubsequence reports whether want appears within have in order (gaps allowed).
func isOrderedSubsequence(want, have []string) bool {
	i := 0
	for _, h := range have {
		if i < len(want) && h == want[i] {
			i++
		}
	}
	return i == len(want)
}

func orNone(s []string) []string {
	if len(s) == 0 {
		return []string{"<none>"}
	}
	return s
}
