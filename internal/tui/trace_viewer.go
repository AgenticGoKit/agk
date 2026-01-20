package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// ViewMode represents the current viewing mode
type ViewMode int

const (
	RunListView ViewMode = iota
	TreeView
	DetailView
)

// TraceRun contains trace run metadata
type TraceRun struct {
	RunID         string
	Command       string
	Status        string
	Duration      float64
	SpanCount     int
	LLMCalls      int
	TotalTokens   int
	EstimatedCost float64
}

// RunData contains a run with its parsed spans
type RunData struct {
	Manifest TraceRun
	Spans    []Span
}

// Model is the main bubbletea model for the trace viewer
type Model struct {
	// Multi-run support
	allRuns     []RunData
	runCursor   int
	selectedRun int

	// Current run data
	runID        string
	manifest     TraceRun
	roots        []*SpanNode
	visibleNodes []*SpanNode
	cursor       int
	viewMode     ViewMode
	viewport     viewport.Model
	ready        bool
	width        int
	height       int
	// Computed metrics
	totalTokens   int
	estimatedCost float64
	errorCount    int
	slowestSpan   *SpanNode
	top3Slowest   []*SpanNode
}

// NewTraceViewer creates a new trace viewer model
func NewTraceViewer(runID string, manifest TraceRun, spans []Span) Model {
	roots := BuildSpanTree(spans)
	visible := FlattenTree(roots)

	// Compute metrics from spans
	var totalTokens int
	var errorCount int
	var slowest *SpanNode
	top3 := make([]*SpanNode, 0, 3)

	for _, node := range visible {
		attrs := node.Span.GetAllAttributes()

		// Count tokens (from various possible attribute names)
		if tokens, ok := attrs["agk.stream.tokens"]; ok {
			if t, ok := tokens.(float64); ok {
				totalTokens += int(t)
			}
		}
		if tokens, ok := attrs["llm.usage.total_tokens"]; ok {
			if t, ok := tokens.(float64); ok {
				totalTokens += int(t)
			}
		}

		// Count errors
		if node.Span.Status.Code != "" && node.Span.Status.Code != "Unset" && node.Span.Status.Code != "Ok" {
			errorCount++
		}

		// Track slowest spans (only leaf nodes or LLM spans)
		if !node.HasChildren() || strings.Contains(strings.ToLower(node.Span.Name), "llm") {
			if slowest == nil || node.DurationMs > slowest.DurationMs {
				slowest = node
			}
			// Insert into top 3
			inserted := false
			for i, s := range top3 {
				if node.DurationMs > s.DurationMs {
					// Insert at position i
					top3 = append(top3[:i], append([]*SpanNode{node}, top3[i:]...)...)
					inserted = true
					break
				}
			}
			if !inserted && len(top3) < 3 {
				top3 = append(top3, node)
			}
			if len(top3) > 3 {
				top3 = top3[:3]
			}
		}
	}

	// Estimate cost (rough: $0.002 per 1K tokens for GPT-3.5 class)
	estimatedCost := float64(totalTokens) * 0.000002

	return Model{
		runID:         runID,
		manifest:      manifest,
		roots:         roots,
		visibleNodes:  visible,
		cursor:        0,
		viewMode:      TreeView,
		totalTokens:   totalTokens,
		estimatedCost: estimatedCost,
		errorCount:    errorCount,
		slowestSpan:   slowest,
		top3Slowest:   top3,
	}
}

// NewTraceExplorer creates a trace explorer with multiple runs (for `agk trace` command)
func NewTraceExplorer(runs []RunData) Model {
	m := Model{
		allRuns:   runs,
		runCursor: 0,
		viewMode:  RunListView,
	}

	// If we have runs, prepare the first one
	if len(runs) > 0 {
		m.loadRun(0)
	}

	return m
}

// loadRun loads a specific run's data into the model
func (m *Model) loadRun(index int) {
	if index < 0 || index >= len(m.allRuns) {
		return
	}

	run := m.allRuns[index]
	m.selectedRun = index
	m.runID = run.Manifest.RunID
	m.manifest = run.Manifest
	m.roots = BuildSpanTree(run.Spans)
	m.visibleNodes = FlattenTree(m.roots)
	m.cursor = 0

	// Recompute metrics
	m.computeMetrics()
}

// computeMetrics calculates metrics for the current run
func (m *Model) computeMetrics() {
	var totalTokens int
	var errorCount int
	var slowest *SpanNode
	top3 := make([]*SpanNode, 0, 3)

	for _, node := range m.visibleNodes {
		attrs := node.Span.GetAllAttributes()

		if tokens, ok := attrs["agk.stream.tokens"]; ok {
			if t, ok := tokens.(float64); ok {
				totalTokens += int(t)
			}
		}
		if tokens, ok := attrs["llm.usage.total_tokens"]; ok {
			if t, ok := tokens.(float64); ok {
				totalTokens += int(t)
			}
		}

		if node.Span.Status.Code != "" && node.Span.Status.Code != "Unset" && node.Span.Status.Code != "Ok" {
			errorCount++
		}

		if !node.HasChildren() || strings.Contains(strings.ToLower(node.Span.Name), "llm") {
			if slowest == nil || node.DurationMs > slowest.DurationMs {
				slowest = node
			}
			inserted := false
			for i, s := range top3 {
				if node.DurationMs > s.DurationMs {
					top3 = append(top3[:i], append([]*SpanNode{node}, top3[i:]...)...)
					inserted = true
					break
				}
			}
			if !inserted && len(top3) < 3 {
				top3 = append(top3, node)
			}
			if len(top3) > 3 {
				top3 = top3[:3]
			}
		}
	}

	m.totalTokens = totalTokens
	m.estimatedCost = float64(totalTokens) * 0.000002
	m.errorCount = errorCount
	m.slowestSpan = slowest
	m.top3Slowest = top3
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.viewMode {
		case RunListView:
			return m.updateRunListView(msg)
		case TreeView:
			return m.updateTreeView(msg)
		case DetailView:
			return m.updateDetailView(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.viewport = viewport.New(msg.Width-4, msg.Height-10)
			m.viewport.YPosition = 0
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = msg.Height - 10
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// updateRunListView handles input in run list view
func (m Model) updateRunListView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if m.runCursor > 0 {
			m.runCursor--
		}

	case "down", "j":
		if m.runCursor < len(m.allRuns)-1 {
			m.runCursor++
		}

	case "enter", "l", "right":
		if m.runCursor < len(m.allRuns) {
			m.loadRun(m.runCursor)
			m.viewMode = TreeView
		}
	}

	return m, nil
}

func (m Model) updateTreeView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc", "backspace":
		// Go back to run list (if we have multiple runs)
		if len(m.allRuns) > 0 {
			m.viewMode = RunListView
			return m, nil
		}
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.visibleNodes)-1 {
			m.cursor++
		}

	case "enter", "l", "right":
		if m.cursor < len(m.visibleNodes) {
			node := m.visibleNodes[m.cursor]
			if node.HasChildren() {
				node.ToggleExpanded()
				m.visibleNodes = FlattenTree(m.roots)
			} else {
				// Show detail view for leaf nodes
				m.viewMode = DetailView
				m.updateDetailViewport()
			}
		}

	case "h", "left":
		if m.cursor < len(m.visibleNodes) {
			node := m.visibleNodes[m.cursor]
			if node.HasChildren() && node.Expanded {
				node.Expanded = false
				m.visibleNodes = FlattenTree(m.roots)
			} else if node.Parent != nil {
				// Navigate to parent
				for i, n := range m.visibleNodes {
					if n == node.Parent {
						m.cursor = i
						break
					}
				}
			}
		}

	case " ":
		// Toggle expand/collapse with space
		if m.cursor < len(m.visibleNodes) {
			node := m.visibleNodes[m.cursor]
			if node.HasChildren() {
				node.ToggleExpanded()
				m.visibleNodes = FlattenTree(m.roots)
			}
		}

	case "d":
		// Show details
		if m.cursor < len(m.visibleNodes) {
			m.viewMode = DetailView
			m.updateDetailViewport()
		}

	case "[":
		// Previous run
		if len(m.allRuns) > 0 && m.selectedRun > 0 {
			m.selectedRun--
			m.runCursor = m.selectedRun
			m.loadRun(m.selectedRun)
		}

	case "]":
		// Next run
		if len(m.allRuns) > 0 && m.selectedRun < len(m.allRuns)-1 {
			m.selectedRun++
			m.runCursor = m.selectedRun
			m.loadRun(m.selectedRun)
		}
	}

	return m, nil
}

func (m Model) updateDetailView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc", "backspace", "h", "left":
		m.viewMode = TreeView
		return m, nil
	}

	// Let viewport handle scrolling
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *Model) updateDetailViewport() {
	if m.cursor >= len(m.visibleNodes) {
		return
	}
	node := m.visibleNodes[m.cursor]
	content := m.renderDetailContent(node)
	m.viewport.SetContent(content)
}

// View renders the model
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	switch m.viewMode {
	case RunListView:
		return m.renderRunListView()
	case TreeView:
		return m.renderTreeView()
	case DetailView:
		return m.renderDetailView()
	default:
		return m.renderRunListView()
	}
}

func (m Model) renderRunListView() string {
	var b strings.Builder

	// Title
	b.WriteString(HeaderStyle.Render("Trace Runs"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width-6))
	b.WriteString("\n\n")

	if len(m.allRuns) == 0 {
		b.WriteString(MutedStyle.Render("No traces found. Run with AGK_TRACE=true to generate traces."))
		b.WriteString("\n")
	} else {
		// Calculate visible area
		maxVisible := m.height - 8
		if maxVisible < 5 {
			maxVisible = 5
		}

		// Scroll offset
		scrollOffset := 0
		if m.runCursor >= maxVisible {
			scrollOffset = m.runCursor - maxVisible + 1
		}

		for i, run := range m.allRuns {
			if i < scrollOffset || i >= scrollOffset+maxVisible {
				continue
			}

			// Status
			status := SuccessStyle.Render("[OK]")
			if run.Manifest.Status != "completed" && run.Manifest.Status != "ok" {
				status = ErrorStyle.Render("[FAIL]")
			}

			// Format line
			runLine := fmt.Sprintf("%-28s  %-12s  %6.2fs  %d LLM  %s",
				run.Manifest.RunID,
				run.Manifest.Command,
				run.Manifest.Duration,
				run.Manifest.LLMCalls,
				status,
			)

			if i == m.runCursor {
				b.WriteString(CursorStyle.Render("→ "))
				b.WriteString(SelectedStyle.Render(runLine))
			} else {
				b.WriteString("  ")
				b.WriteString(runLine)
			}
			b.WriteString("\n")
		}

		// Scroll indicator
		if len(m.allRuns) > maxVisible {
			b.WriteString("\n")
			b.WriteString(MutedStyle.Render(fmt.Sprintf("[%d/%d runs]", m.runCursor+1, len(m.allRuns))))
		}
	}

	b.WriteString("\n\n")
	// Help bar
	help := HelpKeyStyle.Render("[↑↓]") + " Navigate  " +
		HelpKeyStyle.Render("[Enter]") + " View spans  " +
		HelpKeyStyle.Render("[q]") + " Quit"
	b.WriteString(HelpStyle.Render(help))

	return BoxStyle.Width(m.width - 2).Render(b.String())
}

func (m Model) renderTreeView() string {
	var b strings.Builder

	// Back indicator if we have multiple runs
	if len(m.allRuns) > 0 {
		backHint := MutedStyle.Render(fmt.Sprintf("[Esc] Back to list  |  Run %d/%d", m.selectedRun+1, len(m.allRuns)))
		b.WriteString(backHint)
		b.WriteString("\n")
	}

	// Header
	header := m.renderHeader()
	b.WriteString(header)
	b.WriteString("\n")

	// Span tree
	treeContent := m.renderSpanTree()
	b.WriteString(treeContent)

	// Help bar
	help := m.renderHelpBar()
	b.WriteString("\n")
	b.WriteString(help)

	return BoxStyle.Width(m.width - 2).Render(b.String())
}

func (m Model) renderHeader() string {
	var lines []string

	// Title line with run ID
	title := fmt.Sprintf("Trace: %s", m.runID)
	lines = append(lines, HeaderStyle.Render(title))

	// Status indicator
	status := SuccessStyle.Render("[OK]")
	if m.manifest.Status != "completed" && m.manifest.Status != "ok" {
		status = ErrorStyle.Render("[FAIL]")
	}

	// Combined stats line: Duration | Spans | LLM Calls | Tokens | Cost | Status
	var statParts []string
	statParts = append(statParts, fmt.Sprintf("Duration: %s", DurationStyle.Render(fmt.Sprintf("%.2fs", m.manifest.Duration))))
	statParts = append(statParts, fmt.Sprintf("Spans: %d", m.manifest.SpanCount))
	statParts = append(statParts, fmt.Sprintf("LLM: %d", m.manifest.LLMCalls))
	if m.totalTokens > 0 {
		statParts = append(statParts, fmt.Sprintf("Tokens: %s", DurationStyle.Render(fmt.Sprintf("%d", m.totalTokens))))
		statParts = append(statParts, fmt.Sprintf("Cost: %s", WarningStyle.Render(fmt.Sprintf("$%.4f", m.estimatedCost))))
	}
	statParts = append(statParts, fmt.Sprintf("Status: %s", status))

	// Error count inline if present
	if m.errorCount > 0 {
		statParts = append(statParts, ErrorStyle.Render(fmt.Sprintf("Errors: %d", m.errorCount)))
	}

	statsLine := strings.Join(statParts, "  |  ")
	lines = append(lines, MutedStyle.Render(statsLine))

	// Slowest span on separate line (only if meaningful)
	if m.slowestSpan != nil && m.slowestSpan.DurationMs > 100 {
		slowestName := m.slowestSpan.Span.Name
		attrs := m.slowestSpan.Span.GetAllAttributes()
		if stepName, ok := attrs["agk.workflow.step_name"]; ok {
			slowestName = fmt.Sprintf("%v", stepName)
		} else if model, ok := attrs["agk.llm.model"]; ok {
			slowestName = fmt.Sprintf("%s [%v]", m.slowestSpan.Span.Name, model)
		}
		slowestLine := fmt.Sprintf(
			"Bottleneck: %s %s",
			MutedStyle.Render(slowestName),
			DurationStyle.Render(fmt.Sprintf("(%dms)", m.slowestSpan.DurationMs)),
		)
		lines = append(lines, slowestLine)
	}

	return strings.Join(lines, "\n") + "\n" + strings.Repeat("─", m.width-6)
}

func (m Model) renderSpanTree() string {
	var b strings.Builder

	// Calculate visible area
	maxVisible := m.height - 12
	if maxVisible < 5 {
		maxVisible = 5
	}

	// Calculate scroll offset
	scrollOffset := 0
	if m.cursor >= maxVisible {
		scrollOffset = m.cursor - maxVisible + 1
	}

	for i, node := range m.visibleNodes {
		if i < scrollOffset || i >= scrollOffset+maxVisible {
			continue
		}

		line := m.renderSpanLine(node, i == m.cursor)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(m.visibleNodes) > maxVisible {
		indicator := MutedStyle.Render(fmt.Sprintf("  [%d/%d spans]", m.cursor+1, len(m.visibleNodes)))
		b.WriteString(indicator)
	}

	return b.String()
}

func (m Model) renderSpanLine(node *SpanNode, selected bool) string {
	// Indentation
	indent := strings.Repeat("  ", node.Depth)

	// Tree connector
	var prefix string
	if node.HasChildren() {
		if node.Expanded {
			prefix = "▼ "
		} else {
			prefix = "▶ "
		}
	} else {
		prefix = "  "
	}

	// Span name with styling
	spanStyle := GetSpanStyle(node.Span.Name)
	name := spanStyle.Render(node.Span.Name)

	// Get additional context from attributes
	var context string
	attrs := node.Span.GetAllAttributes()

	// For workflow steps, show the step name
	if stepName, ok := attrs["agk.workflow.step_name"]; ok {
		context = MutedStyle.Render(fmt.Sprintf(" [%v]", stepName))
	}

	// For LLM spans, show the model
	if model, ok := attrs["agk.llm.model"]; ok {
		context = MutedStyle.Render(fmt.Sprintf(" [%v]", model))
	}

	// For agent spans, show provider info
	if provider, ok := attrs["agk.llm.provider"]; ok {
		if context == "" {
			context = MutedStyle.Render(fmt.Sprintf(" [%v]", provider))
		}
	}

	// Error indicator
	errorIndicator := ""
	if node.Span.Status.Code != "" && node.Span.Status.Code != "Unset" && node.Span.Status.Code != "Ok" {
		errorIndicator = ErrorStyle.Render(" [ERR]")
	}

	// Duration
	duration := DurationStyle.Render(fmt.Sprintf("(%dms)", node.DurationMs))

	// Build line
	line := fmt.Sprintf("%s%s%s%s%s %s", indent, prefix, name, context, errorIndicator, duration)

	// Apply selection styling
	if selected {
		line = CursorStyle.Render("→ ") + SelectedStyle.Render(line)
	} else {
		line = "  " + line
	}

	return line
}

func (m Model) renderDetailView() string {
	var b strings.Builder

	if m.cursor >= len(m.visibleNodes) {
		return "No span selected"
	}

	node := m.visibleNodes[m.cursor]

	// Header
	title := fmt.Sprintf("📋 Span: %s", node.Span.Name)
	b.WriteString(HeaderStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width-6))
	b.WriteString("\n\n")

	// Viewport content
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Help bar
	help := HelpKeyStyle.Render("[Esc]") + " Back  " +
		HelpKeyStyle.Render("[↑↓]") + " Scroll  " +
		HelpKeyStyle.Render("[q]") + " Quit"
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render(help))

	return BoxStyle.Width(m.width - 2).Render(b.String())
}

func (m Model) renderDetailContent(node *SpanNode) string {
	var b strings.Builder

	// Basic info
	b.WriteString(AttributeKeyStyle.Render("Duration: "))
	b.WriteString(DurationStyle.Render(fmt.Sprintf("%dms", node.DurationMs)))
	b.WriteString("\n")

	b.WriteString(AttributeKeyStyle.Render("Status: "))
	if node.Span.Status.Code == "" || node.Span.Status.Code == "Unset" || node.Span.Status.Code == "Ok" {
		b.WriteString(SuccessStyle.Render("OK"))
	} else {
		b.WriteString(ErrorStyle.Render(node.Span.Status.Code))
	}
	b.WriteString("\n")

	b.WriteString(AttributeKeyStyle.Render("Span ID: "))
	b.WriteString(MutedStyle.Render(node.Span.SpanContext.SpanID))
	b.WriteString("\n")

	if node.Parent != nil {
		b.WriteString(AttributeKeyStyle.Render("Parent ID: "))
		b.WriteString(MutedStyle.Render(node.Span.Parent.SpanID))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Attributes section
	attrs := node.Span.GetAllAttributes()
	if len(attrs) > 0 {
		b.WriteString(HeaderStyle.Render("Attributes"))
		b.WriteString("\n")
		b.WriteString(strings.Repeat("─", 40))
		b.WriteString("\n")

		// Sort keys for consistent display
		keys := make([]string, 0, len(attrs))
		for k := range attrs {
			keys = append(keys, k)
		}
		// Simple sort
		for i := 0; i < len(keys)-1; i++ {
			for j := i + 1; j < len(keys); j++ {
				if keys[i] > keys[j] {
					keys[i], keys[j] = keys[j], keys[i]
				}
			}
		}

		for _, key := range keys {
			val := attrs[key]
			keyStyled := AttributeKeyStyle.Render(fmt.Sprintf("  %s: ", key))
			valStyled := AttributeValueStyle.Render(fmt.Sprintf("%v", val))
			b.WriteString(keyStyled)
			b.WriteString(valStyled)
			b.WriteString("\n")
		}
	} else {
		b.WriteString(MutedStyle.Render("No attributes"))
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderHelpBar() string {
	var help string
	if len(m.allRuns) > 0 {
		help = HelpKeyStyle.Render("[↑↓]") + " Navigate  " +
			HelpKeyStyle.Render("[Enter]") + " Expand  " +
			HelpKeyStyle.Render("[d]") + " Details  " +
			HelpKeyStyle.Render("[/]") + " Prev/Next Run  " +
			HelpKeyStyle.Render("[Esc]") + " List  " +
			HelpKeyStyle.Render("[q]") + " Quit"
	} else {
		help = HelpKeyStyle.Render("[↑↓]") + " Navigate  " +
			HelpKeyStyle.Render("[Enter/→]") + " Expand/Details  " +
			HelpKeyStyle.Render("[←]") + " Collapse  " +
			HelpKeyStyle.Render("[d]") + " Details  " +
			HelpKeyStyle.Render("[q]") + " Quit"
	}

	return HelpStyle.Render(help)
}

// Width returns a copy with updated width
func (m Model) Width(w int) Model {
	m.width = w
	return m
}

// Height returns a copy with updated height
func (m Model) Height(h int) Model {
	m.height = h
	return m
}
