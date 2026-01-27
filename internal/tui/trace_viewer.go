package tui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// tickMsg is sent periodically to check for file updates
type tickMsg time.Time

const (
	StatusUnset = "Unset"
	CtrlC       = "ctrl+c"
	KeyUp       = "up"
	KeyDown     = "down"
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
	// Hot reload / file watching
	tracePath  string    // Path to trace file being watched
	lastOffset int64     // Bytes read so far
	isLive     bool      // Whether we're watching for updates
	lastUpdate time.Time // Last time file was updated
}

func calculateMetrics(nodes []*SpanNode) (totalTokens int, errorCount int, slowest *SpanNode, top3 []*SpanNode) {
	calc := &MetricsCalculator{
		Top3: make([]*SpanNode, 0, 3),
	}

	for _, node := range nodes {
		calc.ProcessNode(node)
	}

	return calc.TotalTokens, calc.ErrorCount, calc.Slowest, calc.Top3
}

type MetricsCalculator struct {
	TotalTokens int
	ErrorCount  int
	Slowest     *SpanNode
	Top3        []*SpanNode
}

func (mc *MetricsCalculator) ProcessNode(node *SpanNode) {
	attrs := node.Span.GetAllAttributes()

	// Count tokens (from various possible attribute names)
	if tokens, ok := attrs["agk.stream.tokens"]; ok {
		if t, ok := tokens.(float64); ok {
			mc.TotalTokens += int(t)
		}
	}
	if tokens, ok := attrs["llm.usage.total_tokens"]; ok {
		if t, ok := tokens.(float64); ok {
			mc.TotalTokens += int(t)
		}
	}

	// Count errors
	if node.Span.Status.Code != "" && node.Span.Status.Code != StatusUnset && node.Span.Status.Code != "Ok" {
		mc.ErrorCount++
	}

	// Track slowest spans (only leaf nodes or LLM spans)
	if !node.HasChildren() || strings.Contains(strings.ToLower(node.Span.Name), "llm") {
		if mc.Slowest == nil || node.DurationMs > mc.Slowest.DurationMs {
			mc.Slowest = node
		}
		mc.updateTop3(node)
	}
}

func (mc *MetricsCalculator) updateTop3(node *SpanNode) {
	inserted := false
	for i, s := range mc.Top3 {
		if node.DurationMs > s.DurationMs {
			// Insert at position i
			mc.Top3 = append(mc.Top3[:i], append([]*SpanNode{node}, mc.Top3[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted && len(mc.Top3) < 3 {
		mc.Top3 = append(mc.Top3, node)
	}
	if len(mc.Top3) > 3 {
		mc.Top3 = mc.Top3[:3]
	}
}

// NewTraceViewer creates a new trace viewer model
func NewTraceViewer(runID string, manifest TraceRun, spans []Span) Model {
	return NewTraceViewerWithPath(runID, manifest, spans, "")
}

// NewTraceViewerWithPath creates a trace viewer with hot reload support
func NewTraceViewerWithPath(runID string, manifest TraceRun, spans []Span, tracePath string) Model {
	roots := BuildSpanTree(spans)
	visible := FlattenTree(roots)

	totalTokens, errorCount, slowest, top3 := calculateMetrics(visible)
	estimatedCost := float64(totalTokens) * 0.000002

	// Calculate initial file offset if path provided
	var lastOffset int64
	if tracePath != "" {
		if info, err := os.Stat(tracePath); err == nil {
			lastOffset = info.Size()
		}
	}

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
		tracePath:     tracePath,
		lastOffset:    lastOffset,
		isLive:        tracePath != "",
		lastUpdate:    time.Now(),
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
	m.totalTokens, m.errorCount, m.slowestSpan, m.top3Slowest = calculateMetrics(m.visibleNodes)
	m.estimatedCost = float64(m.totalTokens) * 0.000002
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	if m.isLive && m.tracePath != "" {
		return m.tickCmd()
	}
	return nil
}

// tickCmd returns a command that sends a tick after 500ms
func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tickMsg:
		// Check for file updates
		if m.isLive && m.tracePath != "" {
			if newSpans := m.checkFileUpdates(); len(newSpans) > 0 {
				// Add new spans and rebuild tree
				m = m.addNewSpans(newSpans)
				m.lastUpdate = time.Now()
			}
			return m, m.tickCmd()
		}
		return m, nil

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

// checkFileUpdates reads new lines from the trace file
func (m *Model) checkFileUpdates() []Span {
	info, err := os.Stat(m.tracePath)
	if err != nil {
		return nil
	}

	// No new data
	if info.Size() <= m.lastOffset {
		return nil
	}

	// Open file and seek to last position
	file, err := os.Open(m.tracePath)
	if err != nil {
		return nil
	}
	defer func() { _ = file.Close() }()

	if _, err := file.Seek(m.lastOffset, 0); err != nil {
		return nil
	}

	// Read new lines
	var newLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			newLines = append(newLines, line)
		}
	}

	// Update offset
	m.lastOffset = info.Size()

	// Parse new spans
	if len(newLines) == 0 {
		return nil
	}

	return ParseSpans(strings.Join(newLines, "\n"))
}

// addNewSpans adds new spans to the existing tree
func (m Model) addNewSpans(newSpans []Span) Model {
	// Get all existing spans
	existingSpans := m.collectAllSpans()

	// Add new spans
	allSpans := append(existingSpans, newSpans...)

	// Rebuild tree
	m.roots = BuildSpanTree(allSpans)
	m.visibleNodes = FlattenTree(m.roots)

	// Update metrics
	m.computeMetrics()

	// Update manifest span count
	m.manifest.SpanCount = len(allSpans)

	return m
}

// collectAllSpans extracts all spans from the tree
func (m Model) collectAllSpans() []Span {
	var spans []Span
	for _, node := range m.visibleNodes {
		spans = append(spans, node.Span)
	}
	return spans
}

// updateRunListView handles input in run list view
func (m Model) updateRunListView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", CtrlC:
		return m, tea.Quit

	case KeyUp, "k":
		if m.runCursor > 0 {
			m.runCursor--
		}

	case KeyDown, "j":
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

	case KeyUp, "k", KeyDown, "j":
		m = m.handleTreeNavigation(msg.String())

	case "enter", "l", "right":
		m = m.handleTreeSelection()

	case "h", "left":
		m = m.handleTreeCollapse()

	case " ":
		m = m.handleTreeToggle()

	case "d":
		// Show details
		if m.cursor < len(m.visibleNodes) {
			m.viewMode = DetailView
			m.updateDetailViewport()
		}

	case "[", "]":
		m = m.handleRunSwitching(msg.String())
	}

	return m, nil
}

func (m Model) handleTreeNavigation(key string) Model {
	switch key {
	case KeyUp, "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case KeyDown, "j":
		if m.cursor < len(m.visibleNodes)-1 {
			m.cursor++
		}
	}
	return m
}

func (m Model) handleTreeSelection() Model {
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
	return m
}

func (m Model) handleTreeCollapse() Model {
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
	return m
}

func (m Model) handleTreeToggle() Model {
	// Toggle expand/collapse with space
	if m.cursor < len(m.visibleNodes) {
		node := m.visibleNodes[m.cursor]
		if node.HasChildren() {
			node.ToggleExpanded()
			m.visibleNodes = FlattenTree(m.roots)
		}
	}
	return m
}

func (m Model) handleRunSwitching(key string) Model {
	if key == "[" {
		// Previous run
		if len(m.allRuns) > 0 && m.selectedRun > 0 {
			m.selectedRun--
			m.runCursor = m.selectedRun
			m.loadRun(m.selectedRun)
		}
	} else if key == "]" {
		// Next run
		if len(m.allRuns) > 0 && m.selectedRun < len(m.allRuns)-1 {
			m.selectedRun++
			m.runCursor = m.selectedRun
			m.loadRun(m.selectedRun)
		}
	}
	return m
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

	var content strings.Builder

	// 1. Global Header
	content.WriteString(m.renderGlobalHeader())
	content.WriteString("\n")

	// 2. Main Content
	var mainContent string
	switch m.viewMode {
	case RunListView:
		mainContent = m.renderRunListView()
	case TreeView:
		mainContent = m.renderTreeView()
	case DetailView:
		mainContent = m.renderDetailView()
	default:
		mainContent = m.renderRunListView()
	}
	content.WriteString(mainContent)
	content.WriteString("\n\n")

	// 3. Global Footer / Help
	// For now, let's let render methods handle their content but WITHOUT the header.
	// And wrap everything in BoxStyle here.

	return BoxStyle.Width(m.width - 2).Render(content.String())
}

func (m Model) renderGlobalHeader() string {
	var b strings.Builder

	// Main Title
	title := "AgenticGoKit Trace Explorer"
	if m.isLive {
		title = "🔴 LIVE  " + title
	}
	b.WriteString(TitleStyle.Render(title))

	// If a run is selected, show its context in the header too?
	// Or keeps it simple. User said "fixed header".

	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width-6))

	return b.String()
}

func (m Model) renderRunListView() string {
	var b strings.Builder

	// HEADER REMOVED

	if len(m.allRuns) == 0 {
		b.WriteString("\n")
		b.WriteString(MutedStyle.Render("No traces found. Run with AGK_TRACE=true to generate traces."))
		b.WriteString("\n")
	} else {
		// Calculate visible area
		// Adjust height for header (approx 2 lines) and footer/padding
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
	// Help bar (keeping here for now as it changes per view)
	help := HelpKeyStyle.Render("[↑↓]") + " Navigate  " +
		HelpKeyStyle.Render("[Enter]") + " View spans  " +
		HelpKeyStyle.Render("[q]") + " Quit"
	b.WriteString(HelpStyle.Render(help))

	return b.String() // Return raw string, View() wraps it
}

func (m Model) renderTreeView() string {
	var b strings.Builder

	// Back indicator
	if len(m.allRuns) > 0 {
		backHint := MutedStyle.Render(fmt.Sprintf("[Esc] Back to list  |  Run %d/%d", m.selectedRun+1, len(m.allRuns)))
		b.WriteString(backHint)
		b.WriteString("\n")
	}

	// Run Details Header (Specific to this view)
	// We keep this as "sub-header"
	header := m.renderRunSummary() // Renamed from renderHeader to avoid confusion
	b.WriteString(header)
	b.WriteString("\n")

	// Span tree
	treeContent := m.renderSpanTree()
	b.WriteString(treeContent)

	// Help bar
	help := m.renderHelpBar()
	b.WriteString("\n")
	b.WriteString(help)

	return b.String()
}

func (m Model) renderRunSummary() string { // Previously renderHeader
	var lines []string

	// Title line with run ID
	title := fmt.Sprintf("Run: %s", m.runID) // Simplified since we have global header
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

	return b.String()
}

func (m Model) renderDetailContent(node *SpanNode) string {
	var b strings.Builder

	// --- Hero Section (Overview) ---
	b.WriteString(m.renderOverviewSection(node))

	// --- Content Section (Audit Data - prompts, responses) ---
	b.WriteString(m.renderContentSection(node))

	// --- Attributes Grouping ---
	b.WriteString(m.renderAttributeSection(node))

	return b.String()
}

func (m Model) renderOverviewSection(node *SpanNode) string {
	var b strings.Builder
	b.WriteString(SectionHeaderStyle.Render("Overview"))
	b.WriteString("\n")

	// Grid layout for basic stats
	stats := []struct {
		Label string
		Value string
	}{
		{"Duration", DurationStyle.Render(fmt.Sprintf("%dms", node.DurationMs))},
		{"Status", func() string {
			if node.Span.Status.Code == "" || node.Span.Status.Code == "Unset" || node.Span.Status.Code == "Ok" {
				return SuccessStyle.Render("OK")
			}
			return ErrorStyle.Render(node.Span.Status.Code)
		}()},
		{"Span ID", MutedStyle.Render(node.Span.SpanContext.SpanID)},
		{"Parent ID", func() string {
			if node.Parent != nil {
				return MutedStyle.Render(node.Span.Parent.SpanID)
			}
			return MutedStyle.Render("-")
		}()},
	}

	for _, stat := range stats {
		b.WriteString(fmt.Sprintf("%-15s %s\n", AttributeKeyStyle.Render(stat.Label+":"), stat.Value))
	}
	return b.String()
}

// renderContentSection displays audit content (prompts, responses, tool args)
// Only shown when detailed trace data is available (AGK_TRACE_LEVEL=detailed)
func (m Model) renderContentSection(node *SpanNode) string {
	var b strings.Builder
	attrs := node.Span.GetAllAttributes()

	// Content keys to look for
	contentKeys := []struct {
		Key   string
		Icon  string
		Label string
	}{
		{"agk.prompt.user", "📝", "User Prompt"},
		{"agk.prompt.system", "🖥️", "System Prompt"},
		{"agk.llm.response", "🤖", "LLM Response"},
		{"agk.tool.arguments", "📥", "Tool Arguments"},
		{"agk.tool.result", "📤", "Tool Result"},
	}

	// Check if any content is available
	hasContent := false
	for _, ck := range contentKeys {
		if _, ok := attrs[ck.Key]; ok {
			hasContent = true
			break
		}
	}

	if !hasContent {
		return ""
	}

	b.WriteString("\n")
	b.WriteString(SectionHeaderStyle.Render("Content (Detailed Trace)"))
	b.WriteString("\n")

	for _, ck := range contentKeys {
		if val, ok := attrs[ck.Key]; ok {
			content := fmt.Sprintf("%v", val)

			// Header with icon
			b.WriteString(fmt.Sprintf("\n%s ", ck.Icon))
			b.WriteString(AttributeKeyStyle.Render(ck.Label))
			b.WriteString("\n")
			b.WriteString(MutedStyle.Render(strings.Repeat("─", 40)))
			b.WriteString("\n")

			// Content (truncate if too long)
			maxLen := 500
			if len(content) > maxLen {
				content = content[:maxLen-3] + "..."
				b.WriteString(content)
				b.WriteString("\n")
				b.WriteString(MutedStyle.Render("[truncated]"))
			} else {
				b.WriteString(content)
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	return b.String()
}

func (m Model) renderAttributeSection(node *SpanNode) string {
	var b strings.Builder
	attrs := node.Span.GetAllAttributes()
	if len(attrs) == 0 {
		b.WriteString(SectionHeaderStyle.Render("Attributes"))
		b.WriteString("\n")
		b.WriteString(MutedStyle.Render("  No attributes available"))
		b.WriteString("\n")
		return b.String()
	}

	// Group attributes
	var (
		llmAttrs      = make(map[string]interface{})
		workflowAttrs = make(map[string]interface{})
		httpAttrs     = make(map[string]interface{})
		otherAttrs    = make(map[string]interface{})
	)

	for k, v := range attrs {
		if strings.HasPrefix(k, "agk.llm.") || strings.HasPrefix(k, "llm.") {
			llmAttrs[k] = v
		} else if strings.HasPrefix(k, "agk.workflow.") || strings.HasPrefix(k, "workflow.") {
			workflowAttrs[k] = v
		} else if strings.HasPrefix(k, "http.") {
			httpAttrs[k] = v
		} else {
			otherAttrs[k] = v
		}
	}

	m.renderAttributeGroup(&b, "LLM Configuration", llmAttrs)
	m.renderAttributeGroup(&b, "Workflow Context", workflowAttrs)
	m.renderAttributeGroup(&b, "HTTP Details", httpAttrs)
	m.renderAttributeGroup(&b, "Metadata", otherAttrs)

	return b.String()
}

func (m Model) renderAttributeGroup(b *strings.Builder, title string, group map[string]interface{}) {
	if len(group) == 0 {
		return
	}
	b.WriteString(SectionHeaderStyle.Render(title))
	b.WriteString("\n")

	// Sort keys
	keys := make([]string, 0, len(group))
	for k := range group {
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
		val := group[key]
		// Clean up key display
		displayKey := key
		if strings.Contains(key, ".") {
			parts := strings.Split(key, ".")
			displayKey = parts[len(parts)-1] // Show only the last part
		}

		keyStyled := AttributeKeyStyle.Render(fmt.Sprintf("  %-20s", displayKey))
		valStyled := AttributeValueStyle.Render(fmt.Sprintf("%v", val))
		b.WriteString(keyStyled)
		b.WriteString(valStyled)
		b.WriteString("\n")
	}
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
