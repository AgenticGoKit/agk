package audit

import (
	"fmt"
	"strings"
)

// GenerateMermaid creates a Mermaid flowchart from a TraceObject
func GenerateMermaid(obj *TraceObject) string {
	var b strings.Builder

	b.WriteString("```mermaid\n")
	b.WriteString("flowchart TD\n")

	// Create nodes for each event
	for i, event := range obj.Events {
		nodeID := fmt.Sprintf("N%d", i)
		label := formatNodeLabel(event)
		shape := getNodeShape(event.Type)

		b.WriteString(fmt.Sprintf("    %s%s\n", nodeID, shape(label)))
	}

	b.WriteString("\n")

	// Create edges between consecutive events
	for i := 0; i < len(obj.Events)-1; i++ {
		b.WriteString(fmt.Sprintf("    N%d --> N%d\n", i, i+1))
	}

	// Add styling
	b.WriteString("\n")
	b.WriteString("    %% Styling\n")
	for i, event := range obj.Events {
		style := getNodeStyle(event.Type)
		if style != "" {
			b.WriteString(fmt.Sprintf("    style N%d %s\n", i, style))
		}
	}

	b.WriteString("```\n")

	return b.String()
}

// GenerateMermaidWithHierarchy creates a Mermaid diagram respecting parent-child relationships
func GenerateMermaidWithHierarchy(obj *TraceObject) string {
	var b strings.Builder

	// Build parent map
	parentMap := make(map[string][]int)
	spanIDToIndex := make(map[string]int)

	for i, event := range obj.Events {
		spanIDToIndex[event.SpanID] = i
		if event.ParentID != "" && event.ParentID != "0000000000000000" {
			parentMap[event.ParentID] = append(parentMap[event.ParentID], i)
		}
	}

	b.WriteString("```mermaid\n")
	b.WriteString("flowchart TD\n")

	// Create nodes
	for i, event := range obj.Events {
		nodeID := fmt.Sprintf("N%d", i)
		label := formatNodeLabel(event)
		shape := getNodeShape(event.Type)
		b.WriteString(fmt.Sprintf("    %s%s\n", nodeID, shape(label)))
	}

	b.WriteString("\n")

	// Create edges based on parent-child relationships
	for parentSpanID, children := range parentMap {
		if parentIdx, ok := spanIDToIndex[parentSpanID]; ok {
			for _, childIdx := range children {
				b.WriteString(fmt.Sprintf("    N%d --> N%d\n", parentIdx, childIdx))
			}
		}
	}

	// For orphan nodes (no parent in trace), connect sequentially
	hasParent := make(map[int]bool)
	for _, children := range parentMap {
		for _, idx := range children {
			hasParent[idx] = true
		}
	}

	// Add styling
	b.WriteString("\n")
	for i, event := range obj.Events {
		style := getNodeStyle(event.Type)
		if style != "" {
			b.WriteString(fmt.Sprintf("    style N%d %s\n", i, style))
		}
	}

	b.WriteString("```\n")

	return b.String()
}

// formatNodeLabel creates a concise label for the node
func formatNodeLabel(event TraceEvent) string {
	// Start with event type icon
	icon := getEventIcon(event.Type)

	// Get a short description
	desc := event.SpanName
	if len(desc) > 25 {
		desc = desc[:22] + "..."
	}

	// Add duration if significant
	duration := ""
	if event.DurationMs > 100 {
		duration = fmt.Sprintf(" (%dms)", event.DurationMs)
	}

	return fmt.Sprintf("%s %s%s", icon, desc, duration)
}

// getEventIcon returns an emoji for the event type
func getEventIcon(eventType EventType) string {
	switch eventType {
	case EventTypeThought:
		return "💭"
	case EventTypeToolCall:
		return "🔧"
	case EventTypeObservation:
		return "👁"
	case EventTypeLLMCall:
		return "🤖"
	case EventTypeDecision:
		return "⚡"
	default:
		return "○"
	}
}

// getNodeShape returns a function that wraps the label in the appropriate shape
func getNodeShape(eventType EventType) func(string) string {
	switch eventType {
	case EventTypeThought:
		return func(label string) string { return fmt.Sprintf("([%s])", label) } // Stadium
	case EventTypeToolCall:
		return func(label string) string { return fmt.Sprintf("[[%s]]", label) } // Subroutine
	case EventTypeObservation:
		return func(label string) string { return fmt.Sprintf("[/%s/]", label) } // Parallelogram
	case EventTypeLLMCall:
		return func(label string) string { return fmt.Sprintf("{%s}", label) } // Rhombus
	case EventTypeDecision:
		return func(label string) string { return fmt.Sprintf("{{%s}}", label) } // Hexagon
	default:
		return func(label string) string { return fmt.Sprintf("[%s]", label) }
	}
}

// getNodeStyle returns Mermaid styling for the event type
func getNodeStyle(eventType EventType) string {
	switch eventType {
	case EventTypeThought:
		return "fill:#e1f5fe,stroke:#01579b"
	case EventTypeToolCall:
		return "fill:#e8f5e9,stroke:#1b5e20"
	case EventTypeObservation:
		return "fill:#fff3e0,stroke:#e65100"
	case EventTypeLLMCall:
		return "fill:#f3e5f5,stroke:#4a148c"
	case EventTypeDecision:
		return "fill:#fce4ec,stroke:#880e4f"
	default:
		return ""
	}
}
