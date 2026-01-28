package audit

import (
	"time"
)

// EventType categorizes trace events for evaluation
type EventType string

const (
	// EventTypeThought represents internal reasoning/decision
	EventTypeThought EventType = "thought"
	// EventTypeToolCall represents a tool invocation
	EventTypeToolCall EventType = "tool_call"
	// EventTypeObservation represents tool output/result
	EventTypeObservation EventType = "observation"
	// EventTypeLLMCall represents an LLM API call
	EventTypeLLMCall EventType = "llm_call"
	// EventTypeDecision represents a decision point
	EventTypeDecision EventType = "decision"
)

// TraceEvent represents a single step in agent execution
type TraceEvent struct {
	Timestamp  time.Time      `json:"timestamp"`
	Type       EventType      `json:"type"`
	SpanID     string         `json:"span_id"`
	SpanName   string         `json:"span_name"`
	Content    string         `json:"content,omitempty"`     // Main content (prompt, response, etc)
	Metadata   map[string]any `json:"metadata,omitempty"`    // Additional context
	DurationMs int64          `json:"duration_ms,omitempty"` // Duration in milliseconds
	ParentID   string         `json:"parent_id,omitempty"`   // Parent span for hierarchy
}

// TraceObject is the complete trace for evaluation
type TraceObject struct {
	RunID       string       `json:"run_id"`
	Command     string       `json:"command,omitempty"`
	StartTime   time.Time    `json:"start_time"`
	EndTime     time.Time    `json:"end_time"`
	Events      []TraceEvent `json:"events"`
	FinalOutput string       `json:"final_output,omitempty"`
	Summary     TraceSummary `json:"summary"`
}

// TraceSummary provides aggregate metrics for the trace
type TraceSummary struct {
	TotalEvents     int     `json:"total_events"`
	ThoughtCount    int     `json:"thought_count"`
	ToolCallCount   int     `json:"tool_call_count"`
	LLMCallCount    int     `json:"llm_call_count"`
	TotalDurationMs int64   `json:"total_duration_ms"`
	TokensUsed      int     `json:"tokens_used,omitempty"`
	EstimatedCost   float64 `json:"estimated_cost,omitempty"`
	HasDetailedData bool    `json:"has_detailed_data"` // True if content captured
}

// ReasoningAnalysis provides evaluation-focused analysis of the trace
type ReasoningAnalysis struct {
	// Path shows the sequence of event types taken
	Path []EventType `json:"path"`
	// DecisionPoints where agent made choices
	DecisionPoints []TraceEvent `json:"decision_points,omitempty"`
	// ToolUsageCorrect indicates if tool calls were appropriate
	ToolUsageCorrect *bool `json:"tool_usage_correct,omitempty"`
	// ReasoningQuality is a 0-1 score for reasoning quality (set by judge)
	ReasoningQuality *float64 `json:"reasoning_quality,omitempty"`
}
