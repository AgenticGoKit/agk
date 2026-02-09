package audit

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/TyphonHill/go-mermaid/diagrams/flowchart"
)

// GenerateMermaid creates a Mermaid flowchart from a TraceObject
func GenerateMermaid(obj *TraceObject) string {
	return GenerateMermaidWithHierarchy(obj)
}

// GenerateMermaidWithHierarchy creates a Mermaid diagram respecting parent-child relationships
func GenerateMermaidWithHierarchy(obj *TraceObject) string {
	// Build parent map
	parentMap := make(map[string][]int)
	spanIDToIndex := make(map[string]int)
	childrenBySpan := make(map[string][]string)
	spanByIndex := make([]string, len(obj.Events))

	for i, event := range obj.Events {
		spanIDToIndex[event.SpanID] = i
		spanByIndex[i] = event.SpanID
		if event.ParentID != "" && event.ParentID != "0000000000000000" {
			parentMap[event.ParentID] = append(parentMap[event.ParentID], i)
			childrenBySpan[event.ParentID] = append(childrenBySpan[event.ParentID], event.SpanID)
		}
	}

	diagram := flowchart.NewFlowchart()
	diagram.EnableMarkdownFence()
	diagram.SetDirection(flowchart.FlowchartDirectionTopDown)
	diagram.Config.SetHtmlLabels(true)

	nodes := make([]*flowchart.Node, len(obj.Events))
	for i, event := range obj.Events {
		label := formatNodeLabel(event)
		node := diagram.AddNode(label)
		applyFlowchartShape(node, event.Type)
		if style := getFlowchartStyle(event.Type); style != nil {
			node.SetStyle(style)
		}
		nodes[i] = node
	}

	parentIndices := make([]int, 0, len(parentMap))
	for parentSpanID := range parentMap {
		if parentIdx, ok := spanIDToIndex[parentSpanID]; ok {
			parentIndices = append(parentIndices, parentIdx)
		}
	}
	sort.Ints(parentIndices)

	addedLinks := make(map[string]bool)
	addLink := func(fromIdx, toIdx int) {
		key := fmt.Sprintf("%d->%d", fromIdx, toIdx)
		if addedLinks[key] {
			return
		}
		addedLinks[key] = true
		diagram.AddLink(nodes[fromIdx], nodes[toIdx])
	}

	// Special handling for sequential workflows: chain steps and nest descendants
	sequentialParents := make([]int, 0)
	for _, parentIdx := range parentIndices {
		if isWorkflowSequential(obj.Events[parentIdx]) {
			sequentialParents = append(sequentialParents, parentIdx)
		}
	}

	if len(sequentialParents) > 0 {
		for _, parentIdx := range sequentialParents {
			parentSpanID := obj.Events[parentIdx].SpanID
			children := parentMap[parentSpanID]
			stepChildren := make([]int, 0)
			for _, childIdx := range children {
				if isWorkflowStep(obj.Events[childIdx]) {
					stepChildren = append(stepChildren, childIdx)
				}
			}

			sort.Slice(stepChildren, func(i, j int) bool {
				idxI := stepChildren[i]
				idxJ := stepChildren[j]
				stepI, okI := getStepIndex(obj.Events[idxI])
				stepJ, okJ := getStepIndex(obj.Events[idxJ])
				if okI && okJ {
					return stepI < stepJ
				}
				if okI != okJ {
					return okI
				}
				return obj.Events[idxI].Timestamp.Before(obj.Events[idxJ].Timestamp)
			})

			if len(stepChildren) > 0 {
				addLink(parentIdx, stepChildren[0])
				for i := 0; i < len(stepChildren)-1; i++ {
					addLink(stepChildren[i], stepChildren[i+1])
				}
			}

			for _, stepIdx := range stepChildren {
				descendants := collectDescendantIndices(spanByIndex[stepIdx], spanIDToIndex, childrenBySpan, obj)
				if len(descendants) == 0 {
					continue
				}
				sort.Slice(descendants, func(i, j int) bool {
					return obj.Events[descendants[i]].Timestamp.Before(obj.Events[descendants[j]].Timestamp)
				})
				addLink(stepIdx, descendants[0])
				for i := 0; i < len(descendants)-1; i++ {
					addLink(descendants[i], descendants[i+1])
				}
			}
		}

		return diagram.String()
	}

	for _, parentIdx := range parentIndices {
		parentEvent := obj.Events[parentIdx]
		parentSpanID := parentEvent.SpanID
		children := parentMap[parentSpanID]

		if isWorkflowSequential(parentEvent) {
			stepChildren := make([]int, 0)
			otherChildren := make([]int, 0)
			for _, childIdx := range children {
				if isWorkflowStep(obj.Events[childIdx]) {
					stepChildren = append(stepChildren, childIdx)
				} else {
					otherChildren = append(otherChildren, childIdx)
				}
			}

			if len(stepChildren) > 0 {
				sort.Slice(stepChildren, func(i, j int) bool {
					idxI := stepChildren[i]
					idxJ := stepChildren[j]
					stepI, okI := getStepIndex(obj.Events[idxI])
					stepJ, okJ := getStepIndex(obj.Events[idxJ])
					if okI && okJ {
						return stepI < stepJ
					}
					if okI != okJ {
						return okI
					}
					return obj.Events[idxI].Timestamp.Before(obj.Events[idxJ].Timestamp)
				})

				addLink(parentIdx, stepChildren[0])
				for i := 0; i < len(stepChildren)-1; i++ {
					addLink(stepChildren[i], stepChildren[i+1])
				}
			}

			sort.Ints(otherChildren)
			for _, childIdx := range otherChildren {
				addLink(parentIdx, childIdx)
			}
			continue
		}

		sort.Ints(children)
		for _, childIdx := range children {
			addLink(parentIdx, childIdx)
		}
	}

	return diagram.String()
}

func collectDescendantIndices(rootSpanID string, spanIDToIndex map[string]int, childrenBySpan map[string][]string, obj *TraceObject) []int {
	var result []int
	queue := []string{rootSpanID}
	visited := make(map[string]bool)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if visited[current] {
			continue
		}
		visited[current] = true
		children := childrenBySpan[current]
		for _, childSpan := range children {
			idx, ok := spanIDToIndex[childSpan]
			if ok {
				if isWorkflowStep(obj.Events[idx]) || isWorkflowSequential(obj.Events[idx]) {
					// Skip other workflow step nodes to avoid cross-linking
				} else {
					result = append(result, idx)
				}
			}
			queue = append(queue, childSpan)
		}
	}

	return result
}

// formatNodeLabel creates a concise label for the node
func formatNodeLabel(event TraceEvent) string {
	// Start with event type icon
	icon := getEventIcon(event.Type)

	// Get a short description
	desc := event.SpanName
	if stepName, ok := event.Metadata["agk.workflow.step_name"].(string); ok && stepName != "" {
		desc = "step:" + stepName
	}
	if agentName, ok := event.Metadata["agk.agent.name"].(string); ok && agentName != "" {
		desc = fmt.Sprintf("%s @%s", desc, agentName)
	}
	if len(desc) > 60 {
		desc = desc[:57] + "..."
	}

	// Add duration on new line
	duration := ""
	if event.DurationMs > 0 {
		duration = fmt.Sprintf("<br/>%dms", event.DurationMs)
	}

	return fmt.Sprintf("%s %s%s", icon, desc, duration)
}

func isWorkflowSequential(event TraceEvent) bool {
	name := strings.ToLower(event.SpanName)
	return strings.Contains(name, "workflow.sequential")
}

func isWorkflowStep(event TraceEvent) bool {
	name := strings.ToLower(event.SpanName)
	if strings.Contains(name, "workflow.step") {
		return true
	}
	if stepName, ok := event.Metadata["agk.workflow.step_name"].(string); ok && stepName != "" {
		return true
	}
	return false
}

func getStepIndex(event TraceEvent) (int, bool) {
	if raw, ok := event.Metadata["agk.workflow.step_index"]; ok {
		switch v := raw.(type) {
		case int:
			return v, true
		case int64:
			return int(v), true
		case float64:
			return int(v), true
		case float32:
			return int(v), true
		case string:
			if parsed, err := parseInt(v); err == nil {
				return parsed, true
			}
		}
	}
	if !event.Timestamp.IsZero() {
		return int(event.Timestamp.UnixMilli()), false
	}
	return 0, false
}

func parseInt(value string) (int, error) {
	return strconv.Atoi(value)
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
func applyFlowchartShape(node *flowchart.Node, eventType EventType) {
	switch eventType {
	case EventTypeThought:
		node.SetShape(flowchart.NodeShapeTerminal)
	case EventTypeToolCall:
		node.SetShape(flowchart.NodeShapeSubprocess)
	case EventTypeObservation:
		node.SetShape(flowchart.NodeShapeInputOutput)
	case EventTypeLLMCall:
		node.SetShape(flowchart.NodeShapeDecision)
	case EventTypeDecision:
		node.SetShape(flowchart.NodeShapePrepare)
	default:
		node.SetShape(flowchart.NodeShapeProcess)
	}
}

// getFlowchartStyle returns Mermaid styling for the event type
func getFlowchartStyle(eventType EventType) *flowchart.NodeStyle {
	style := flowchart.NewNodeStyle()
	style.StrokeWidth = 1

	switch eventType {
	case EventTypeThought:
		style.Fill = "#e1f5fe"
		style.Stroke = "#01579b"
	case EventTypeToolCall:
		style.Fill = "#e8f5e9"
		style.Stroke = "#1b5e20"
	case EventTypeObservation:
		style.Fill = "#fff3e0"
		style.Stroke = "#e65100"
	case EventTypeLLMCall:
		style.Fill = "#f3e5f5"
		style.Stroke = "#4a148c"
	case EventTypeDecision:
		style.Fill = "#fce4ec"
		style.Stroke = "#880e4f"
	default:
		return nil
	}

	return style
}
