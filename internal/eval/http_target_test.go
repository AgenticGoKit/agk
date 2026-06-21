package eval

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchTrace(t *testing.T) {
	const traceID = "trace-abc"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/traces/"+traceID {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "trace-abc",
			"spans": [
				{"name": "agk.agent.run"},
				{"name": "agk.llm.generate"},
				{"name": "agk.tool.call", "attributes": {"agk.tool.name": "search"}}
			]
		}`))
	}))
	defer server.Close()

	target := NewHTTPTarget(server.URL, 5*time.Second)

	trace, err := target.FetchTrace(traceID)
	if err != nil {
		t.Fatalf("FetchTrace error: %v", err)
	}
	if trace.ID != traceID {
		t.Errorf("trace ID = %q, want %q", trace.ID, traceID)
	}
	if len(trace.Spans) != 3 {
		t.Fatalf("got %d spans, want 3", len(trace.Spans))
	}

	// Round-trip through the normalizer to confirm the wire format is consumable.
	obs := buildObservedTrace(trace, nil)
	if obs.LLMCalls != 1 {
		t.Errorf("LLMCalls = %d, want 1", obs.LLMCalls)
	}
	if len(obs.ToolCalls) != 1 || obs.ToolCalls[0] != "search" {
		t.Errorf("ToolCalls = %v, want [search]", obs.ToolCalls)
	}
}

func TestFetchTraceErrors(t *testing.T) {
	target := NewHTTPTarget("http://127.0.0.1:0", time.Second)

	if _, err := target.FetchTrace(""); err == nil {
		t.Error("expected error for empty trace id")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer server.Close()

	target = NewHTTPTarget(server.URL, 5*time.Second)
	if _, err := target.FetchTrace("missing"); err == nil {
		t.Error("expected error for 404 trace fetch")
	}
}
