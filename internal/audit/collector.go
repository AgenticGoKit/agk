package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Collector extracts trace events from stored span data
type Collector struct {
	runPath string
	spans   []RawSpan
}

// RawSpan represents a parsed span from trace.jsonl
type RawSpan struct {
	Name        string                   `json:"Name"`
	SpanContext SpanContext              `json:"SpanContext"`
	Parent      SpanContext              `json:"Parent"`
	StartTime   string                   `json:"StartTime"`
	EndTime     string                   `json:"EndTime"`
	Attributes  []map[string]interface{} `json:"Attributes"`
	Status      SpanStatus               `json:"Status"`
}

// SpanContext contains span identification
type SpanContext struct {
	TraceID string `json:"TraceID"`
	SpanID  string `json:"SpanID"`
}

// SpanStatus contains span status
type SpanStatus struct {
	Code        string `json:"Code"`
	Description string `json:"Description,omitempty"`
}

// NewCollector creates a collector from a run path
func NewCollector(runPath string) (*Collector, error) {
	tracePath := filepath.Join(runPath, "trace.jsonl")
	data, err := os.ReadFile(tracePath)
	if err != nil {
		return nil, err
	}

	var spans []RawSpan
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var span RawSpan
		if err := json.Unmarshal([]byte(line), &span); err != nil {
			continue
		}
		spans = append(spans, span)
	}

	return &Collector{
		runPath: runPath,
		spans:   spans,
	}, nil
}

// Collect extracts TraceObject from the spans
func (c *Collector) Collect() (*TraceObject, error) {
	runID := filepath.Base(c.runPath)

	obj := &TraceObject{
		RunID:  runID,
		Events: make([]TraceEvent, 0),
		Summary: TraceSummary{
			HasDetailedData: false,
		},
	}

	for _, span := range c.spans {
		event := c.spanToEvent(span)
		obj.Events = append(obj.Events, event)

		// Update summary counts
		switch event.Type {
		case EventTypeThought:
			obj.Summary.ThoughtCount++
		case EventTypeToolCall:
			obj.Summary.ToolCallCount++
		case EventTypeLLMCall:
			obj.Summary.LLMCallCount++
		}

		// Check for detailed data
		if event.Content != "" {
			obj.Summary.HasDetailedData = true
		}

		// Update timing
		if obj.StartTime.IsZero() || event.Timestamp.Before(obj.StartTime) {
			obj.StartTime = event.Timestamp
		}
		if event.Timestamp.After(obj.EndTime) {
			obj.EndTime = event.Timestamp
		}
	}

	obj.Summary.TotalEvents = len(obj.Events)
	obj.Summary.TotalDurationMs = obj.EndTime.Sub(obj.StartTime).Milliseconds()

	// Sort events by timestamp
	sort.Slice(obj.Events, func(i, j int) bool {
		return obj.Events[i].Timestamp.Before(obj.Events[j].Timestamp)
	})

	return obj, nil
}

// spanToEvent converts a raw span to a TraceEvent
func (c *Collector) spanToEvent(span RawSpan) TraceEvent {
	event := TraceEvent{
		SpanID:   span.SpanContext.SpanID,
		SpanName: span.Name,
		ParentID: span.Parent.SpanID,
		Metadata: make(map[string]any),
	}

	// Parse timestamp
	if t, err := time.Parse(time.RFC3339, span.StartTime); err == nil {
		event.Timestamp = t
	}

	// Calculate duration
	if start, err := time.Parse(time.RFC3339, span.StartTime); err == nil {
		if end, err := time.Parse(time.RFC3339, span.EndTime); err == nil {
			event.DurationMs = end.Sub(start).Milliseconds()
		}
	}

	// Determine event type and extract attributes
	event.Type = c.classifySpan(span.Name)

	// Extract attributes
	for _, attr := range span.Attributes {
		key, ok := attr["Key"].(string)
		if !ok {
			continue
		}
		value, ok := attr["Value"].(map[string]interface{})
		if !ok {
			continue
		}
		val, ok := value["Value"]
		if !ok {
			continue
		}

		// Store in metadata
		event.Metadata[key] = val

		// Check for content fields (detailed trace level)
		switch key {
		case "agk.prompt.user", "agk.llm.response":
			event.Content = val.(string)
		case "agk.tool.arguments":
			if event.Type == EventTypeToolCall {
				event.Content = val.(string)
			}
		case "agk.tool.result":
			if event.Type == EventTypeObservation {
				event.Content = val.(string)
			}
		}
	}

	return event
}

// classifySpan determines the event type based on span name
func (c *Collector) classifySpan(name string) EventType {
	nameLower := strings.ToLower(name)

	switch {
	case strings.Contains(nameLower, "tool"):
		return EventTypeToolCall
	case strings.Contains(nameLower, "llm"):
		return EventTypeLLMCall
	case strings.Contains(nameLower, "agent"):
		return EventTypeThought
	case strings.Contains(nameLower, "workflow"):
		return EventTypeDecision
	default:
		return EventTypeThought
	}
}

// GetReasoningPath extracts the sequence of event types
func (c *Collector) GetReasoningPath(obj *TraceObject) []EventType {
	path := make([]EventType, len(obj.Events))
	for i, event := range obj.Events {
		path[i] = event.Type
	}
	return path
}
