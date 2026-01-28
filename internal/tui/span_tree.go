package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Span represents a parsed OpenTelemetry span
type Span struct {
	Name                 string                   `json:"Name"`
	StartTime            string                   `json:"StartTime"`
	EndTime              string                   `json:"EndTime"`
	Attributes           []map[string]interface{} `json:"Attributes,omitempty"`
	SpanContext          SpanContext              `json:"SpanContext"`
	Parent               ParentSpan               `json:"Parent"`
	SpanKind             int                      `json:"SpanKind"`
	Status               SpanStatus               `json:"Status"`
	ChildSpanCount       int                      `json:"ChildSpanCount"`
	InstrumentationScope map[string]interface{}   `json:"InstrumentationScope"`
}

// SpanContext contains span identification
type SpanContext struct {
	TraceID string `json:"TraceID"`
	SpanID  string `json:"SpanID"`
}

// ParentSpan contains parent span reference
type ParentSpan struct {
	TraceID string `json:"TraceID"`
	SpanID  string `json:"SpanID"`
}

// SpanStatus contains span status
type SpanStatus struct {
	Code        string `json:"Code"`
	Description string `json:"Description,omitempty"`
}

// SpanNode represents a span in the hierarchical tree
type SpanNode struct {
	Span       Span
	Children   []*SpanNode
	Depth      int
	Expanded   bool
	Parent     *SpanNode
	DurationMs int64
}

// ParseSpans parses JSONL trace data into spans
func ParseSpans(data string) []Span {
	var spans []Span
	lines := strings.Split(data, "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		var span Span
		if err := json.Unmarshal([]byte(line), &span); err != nil {
			continue
		}
		spans = append(spans, span)
	}

	return spans
}

// BuildSpanTree builds a hierarchical tree from flat span list
func BuildSpanTree(spans []Span) []*SpanNode {
	// Create node map
	nodeMap := make(map[string]*SpanNode)
	for i := range spans {
		node := &SpanNode{
			Span:       spans[i],
			Children:   make([]*SpanNode, 0),
			Expanded:   true, // Start expanded
			DurationMs: calculateDuration(spans[i].StartTime, spans[i].EndTime),
		}
		nodeMap[spans[i].SpanContext.SpanID] = node
	}

	// Build tree structure
	var roots []*SpanNode
	for _, node := range nodeMap {
		parentID := node.Span.Parent.SpanID
		if parentID == "" || parentID == "0000000000000000" {
			// This is a root span
			node.Depth = 0
			roots = append(roots, node)
		} else if parent, ok := nodeMap[parentID]; ok {
			// Has a parent in our span set
			node.Parent = parent
			parent.Children = append(parent.Children, node)
		} else {
			// Parent not found, treat as root
			node.Depth = 0
			roots = append(roots, node)
		}
	}

	// Calculate depths
	for _, root := range roots {
		setDepths(root, 0)
	}

	// Sort roots by start time
	sortNodesByTime(roots)

	return roots
}

// setDepths recursively sets node depths
func setDepths(node *SpanNode, depth int) {
	node.Depth = depth
	sortNodesByTime(node.Children)
	for _, child := range node.Children {
		setDepths(child, depth+1)
	}
}

// sortNodesByTime sorts nodes by start time
func sortNodesByTime(nodes []*SpanNode) {
	for i := 0; i < len(nodes)-1; i++ {
		for j := i + 1; j < len(nodes); j++ {
			t1, _ := time.Parse(time.RFC3339, nodes[i].Span.StartTime)
			t2, _ := time.Parse(time.RFC3339, nodes[j].Span.StartTime)
			if t1.After(t2) {
				nodes[i], nodes[j] = nodes[j], nodes[i]
			}
		}
	}
}

// FlattenTree returns a flat list of visible nodes for display
func FlattenTree(roots []*SpanNode) []*SpanNode {
	var result []*SpanNode
	for _, root := range roots {
		flattenNode(root, &result)
	}
	return result
}

func flattenNode(node *SpanNode, result *[]*SpanNode) {
	*result = append(*result, node)
	if node.Expanded {
		for _, child := range node.Children {
			flattenNode(child, result)
		}
	}
}

// calculateDuration calculates duration in milliseconds
func calculateDuration(startTime, endTime string) int64 {
	if startTime == "" || endTime == "" {
		return 0
	}
	start, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return 0
	}
	end, err := time.Parse(time.RFC3339, endTime)
	if err != nil {
		return 0
	}
	return end.Sub(start).Milliseconds()
}

// GetAttribute gets an attribute value by key
func (s *Span) GetAttribute(key string) (interface{}, bool) {
	for _, attr := range s.Attributes {
		if k, ok := attr["Key"].(string); ok && k == key {
			if value, ok := attr["Value"].(map[string]interface{}); ok {
				if val, ok := value["Value"]; ok {
					return val, true
				}
			}
		}
	}
	return nil, false
}

// GetImportantAttributes returns filtered important attributes
func (s *Span) GetImportantAttributes() map[string]interface{} {
	important := make(map[string]interface{})
	importantKeys := []string{
		"agk.llm.provider", "agk.llm.model", "agk.llm.max_tokens", "agk.llm.temperature",
		"agk.stream.tokens", "agk.llm.latency_ms",
		"agk.workflow.step_name", "agk.workflow.step_index", "agk.workflow.mode",
		"agk.workflow.success", "agk.workflow.latency_ms", "agk.workflow.id",
		"agk.tools.count", "agk.tool.name",
		"http.status_code", "llm.streaming",
		"error.message", "error.type",
	}

	for _, attr := range s.Attributes {
		if key, ok := attr["Key"].(string); ok {
			for _, impKey := range importantKeys {
				if key == impKey {
					if value, ok := attr["Value"].(map[string]interface{}); ok {
						if val, ok := value["Value"]; ok {
							important[key] = val
						}
					}
					break
				}
			}
		}
	}

	return important
}

// GetAllAttributes returns all attributes as key-value pairs
func (s *Span) GetAllAttributes() map[string]interface{} {
	attrs := make(map[string]interface{})
	for _, attr := range s.Attributes {
		if key, ok := attr["Key"].(string); ok {
			if value, ok := attr["Value"].(map[string]interface{}); ok {
				if val, ok := value["Value"]; ok {
					attrs[key] = val
				}
			}
		}
	}
	return attrs
}

// HasChildren returns true if the span has children
func (n *SpanNode) HasChildren() bool {
	return len(n.Children) > 0
}

// ToggleExpanded toggles the expanded state
func (n *SpanNode) ToggleExpanded() {
	n.Expanded = !n.Expanded
}

// GetSpanType returns the type of span for styling
func (s *Span) GetSpanType() string {
	name := strings.ToLower(s.Name)
	switch {
	case strings.Contains(name, "workflow"):
		return "workflow"
	case strings.Contains(name, "agent"):
		return "agent"
	case strings.Contains(name, "llm"):
		return "llm"
	case strings.Contains(name, "tool"), strings.Contains(name, "mcp"):
		return "tool"
	default:
		return "other"
	}
}

// GetFriendlyName returns a user-friendly display name for the span
func (s *Span) GetFriendlyName() string {
	attrs := s.GetAllAttributes()
	name := strings.ToLower(s.Name)

	// For workflow steps, use the step name as primary label
	if strings.Contains(name, "workflow.step") {
		if stepName, ok := attrs["agk.workflow.step_name"]; ok {
			return fmt.Sprintf("🔹 %v", stepName)
		}
	}

	// For workflow root spans, show workflow mode
	if strings.Contains(name, "agk.workflow.sequential") {
		return "📋 Sequential Workflow"
	}
	if strings.Contains(name, "agk.workflow.parallel") {
		return "⚡ Parallel Workflow"
	}
	if strings.Contains(name, "agk.workflow.dag") {
		return "🔀 DAG Workflow"
	}
	if strings.Contains(name, "agk.workflow.loop") {
		return "🔄 Loop Workflow"
	}

	// For LLM spans, show provider and model
	if strings.Contains(name, "llm") {
		if model, ok := attrs["agk.llm.model"]; ok {
			provider := "llm"
			if p, ok := attrs["agk.llm.provider"]; ok {
				provider = fmt.Sprintf("%v", p)
			}
			return fmt.Sprintf("🤖 %s [%v]", provider, model)
		}
	}

	// For agent spans, simplify
	if strings.Contains(name, "agk.agent.run") {
		if model, ok := attrs["agk.llm.model"]; ok {
			return fmt.Sprintf("🤖 Agent [%v]", model)
		}
		return "🤖 Agent"
	}

	// Default: return original name
	return s.Name
}

// IsWorkflowStep returns true if this span represents a workflow step
func (s *Span) IsWorkflowStep() bool {
	return strings.Contains(strings.ToLower(s.Name), "workflow.step")
}

// IsInternalSpan returns true if this span should be hidden by default (detail level)
func (s *Span) IsInternalSpan() bool {
	name := strings.ToLower(s.Name)
	// These are internal implementation details, show only when parent expanded
	return strings.Contains(name, "agk.agent.run.stream") ||
		strings.Contains(name, "agk.agent.run.execute") ||
		strings.Contains(name, "transform")
}
