package tui

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// FocusArea represents which panel is currently focused
type FocusArea int

const (
	FocusTree FocusArea = iota
	FocusDetails
	FocusMetadata
)

// DetailTab represents the active tab in the details panel
type DetailTab int

const (
	TabOverview DetailTab = iota
	TabPrompt
	TabResponse
	TabAttributes
	TabTiming
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
	runID            string
	manifest         TraceRun
	roots            []*SpanNode
	visibleNodes     []*SpanNode
	cursor           int
	viewMode         ViewMode
	focusArea        FocusArea // Current focused panel
	selectedTab      DetailTab // Active tab in details panel
	treeViewport     viewport.Model
	detailViewport   viewport.Model
	metadataViewport viewport.Model
	ready            bool
	width            int
	height           int
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
	// Search state
	searchMode    bool
	searchQuery   string
	searchMatches []*SpanNode
	searchIndex   int
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
		runID:            runID,
		manifest:         manifest,
		roots:            roots,
		visibleNodes:     visible,
		cursor:           0,
		viewMode:         TreeView,
		focusArea:        FocusTree,
		selectedTab:      TabOverview,
		treeViewport:     viewport.New(40, 10),
		detailViewport:   viewport.New(40, 10),
		metadataViewport: viewport.New(30, 20),
		totalTokens:      totalTokens,
		estimatedCost:    estimatedCost,
		errorCount:       errorCount,
		slowestSpan:      slowest,
		top3Slowest:      top3,
		tracePath:        tracePath,
		lastOffset:       lastOffset,
		isLive:           tracePath != "",
		lastUpdate:       time.Now(),
		searchMode:       false,
		searchQuery:      "",
		searchMatches:    make([]*SpanNode, 0),
		searchIndex:      -1,
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
			// Handle search input mode
			if m.searchMode {
				return m.updateSearchInput(msg)
			}
			return m.updateTreeView(msg)
		case DetailView:
			return m.updateDetailView(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate panel dimensions
		availableWidth := msg.Width - 6
		availableHeight := msg.Height - 12

		leftWidth := (availableWidth * 66) / 100
		rightWidth := availableWidth - leftWidth
		treeHeight := (availableHeight * 40) / 100
		if treeHeight < 10 {
			treeHeight = 10
		}
		detailHeight := availableHeight - treeHeight
		if detailHeight < 8 {
			detailHeight = 8
		}

		if !m.ready {
			m.treeViewport = viewport.New(leftWidth-4, treeHeight-3)
			m.detailViewport = viewport.New(availableWidth-4, availableHeight-4)
			m.metadataViewport = viewport.New(rightWidth-4, availableHeight-3)
			m.ready = true
		} else {
			m.treeViewport.Width = leftWidth - 4
			m.treeViewport.Height = treeHeight - 3
			m.detailViewport.Width = availableWidth - 4
			m.detailViewport.Height = availableHeight - 4
			m.metadataViewport.Width = rightWidth - 4
			m.metadataViewport.Height = availableHeight - 3
		}
	}

	// Update the focused viewport
	switch m.focusArea {
	case FocusTree:
		m.treeViewport, cmd = m.treeViewport.Update(msg)
	case FocusDetails:
		m.detailViewport, cmd = m.detailViewport.Update(msg)
	case FocusMetadata:
		m.metadataViewport, cmd = m.metadataViewport.Update(msg)
	}

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

	case "tab":
		// Cycle focus forward: Tree -> Details -> Metadata -> Tree
		m.focusArea = (m.focusArea + 1) % 3
		return m, nil

	case "shift+tab":
		// Cycle focus backward
		m.focusArea = (m.focusArea + 2) % 3 // +2 mod 3 is same as -1
		return m, nil

	case "left":
		// Switch tabs left (always available)
		if m.selectedTab > 0 {
			m.selectedTab--
		} else {
			m.selectedTab = TabTiming // Wrap to last tab
		}
		return m, nil

	case "right":
		// Switch tabs right (always available)
		if m.selectedTab < TabTiming {
			m.selectedTab++
		} else {
			m.selectedTab = TabOverview // Wrap to first tab
		}
		return m, nil

	case "h":
		// Tree collapse only with 'h'
		m = m.handleTreeCollapse()

	case "l":
		// Tree expand only with 'l'
		m = m.handleTreeSelection()

	case "1":
		m.selectedTab = TabOverview
		return m, nil

	case "2":
		m.selectedTab = TabPrompt
		return m, nil

	case "3":
		m.selectedTab = TabResponse
		return m, nil

	case "4":
		m.selectedTab = TabAttributes
		return m, nil

	case "5":
		m.selectedTab = TabTiming
		return m, nil

	case "esc", "backspace":
		// Clear search if active
		if m.searchMode {
			m.searchMode = false
			m.searchQuery = ""
			m.searchMatches = nil
			m.searchIndex = -1
			return m, nil
		}
		// Go back to run list (if we have multiple runs)
		if len(m.allRuns) > 0 {
			m.viewMode = RunListView
			return m, nil
		}
		return m, tea.Quit

	case KeyUp, "k", KeyDown, "j":
		m = m.handleTreeNavigation(msg.String())

	case "enter":
		m = m.handleTreeSelection()

	case " ":
		m = m.handleTreeToggle()

	case "d":
		// Show details
		if m.cursor < len(m.visibleNodes) {
			m.viewMode = DetailView
			m.updateDetailViewport()
		}

	case "/":
		// Enter search mode
		m.searchMode = true
		m.searchQuery = ""
		return m, nil

	case "n":
		// Next search match
		if len(m.searchMatches) > 0 {
			m.searchIndex = (m.searchIndex + 1) % len(m.searchMatches)
			m = m.jumpToSearchMatch()
		}
		return m, nil

	case "N":
		// Previous search match
		if len(m.searchMatches) > 0 {
			if m.searchIndex <= 0 {
				m.searchIndex = len(m.searchMatches) - 1
			} else {
				m.searchIndex--
			}
			m = m.jumpToSearchMatch()
		}
		return m, nil

	case "e":
		// Jump to next error
		m = m.jumpToNextError()
		return m, nil

	case "E":
		// Jump to previous error
		m = m.jumpToPreviousError()
		return m, nil

	case "[", "]":
		m = m.handleRunSwitching(msg.String())
	}

	return m, nil
}

// updateSearchInput handles keyboard input in search mode
func (m Model) updateSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel search
		m.searchMode = false
		m.searchQuery = ""
		return m, nil

	case "enter":
		// Execute search
		m.searchMode = false
		m = m.executeSearch()
		return m, nil

	case "backspace":
		// Delete character
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}
		return m, nil

	default:
		// Add character
		if len(msg.String()) == 1 {
			m.searchQuery += msg.String()
		}
		return m, nil
	}
}

// executeSearch performs the search and populates matches
func (m Model) executeSearch() Model {
	m.searchMatches = make([]*SpanNode, 0)
	m.searchIndex = -1

	if m.searchQuery == "" {
		return m
	}

	query := strings.ToLower(m.searchQuery)

	// Search through all visible nodes
	for _, node := range m.visibleNodes {
		if m.matchesSearch(node, query) {
			m.searchMatches = append(m.searchMatches, node)
		}
	}

	// Jump to first match if any
	if len(m.searchMatches) > 0 {
		m.searchIndex = 0
		m = m.jumpToSearchMatch()
	}

	return m
}

// matchesSearch checks if a node matches the search query
func (m Model) matchesSearch(node *SpanNode, query string) bool {
	// Search in span name
	if strings.Contains(strings.ToLower(node.Span.Name), query) {
		return true
	}

	// Search in friendly name
	if strings.Contains(strings.ToLower(node.Span.GetFriendlyName()), query) {
		return true
	}

	// Search in attributes
	attrs := node.Span.GetAllAttributes()
	for k, v := range attrs {
		if strings.Contains(strings.ToLower(k), query) {
			return true
		}
		if strings.Contains(strings.ToLower(fmt.Sprintf("%v", v)), query) {
			return true
		}
	}

	// Search in status
	if strings.Contains(strings.ToLower(node.Span.Status.Code), query) {
		return true
	}
	if strings.Contains(strings.ToLower(node.Span.Status.Description), query) {
		return true
	}

	return false
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
	var cmd tea.Cmd

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc", "backspace":
		m.viewMode = TreeView
		return m, nil

	case "left":
		// Switch tabs left
		if m.selectedTab > 0 {
			m.selectedTab--
		} else {
			m.selectedTab = TabTiming
		}
		// Update viewport content for new tab
		node := m.visibleNodes[m.cursor]
		var content string
		switch m.selectedTab {
		case TabOverview:
			content = m.renderOverviewTab(node)
		case TabPrompt:
			content = m.renderPromptTab(node)
		case TabResponse:
			content = m.renderResponseTab(node)
		case TabAttributes:
			content = m.renderAttributesTab(node)
		case TabTiming:
			content = m.renderTimingTab(node)
		}
		m.detailViewport.SetContent(content)
		return m, nil

	case "right":
		// Switch tabs right
		if m.selectedTab < TabTiming {
			m.selectedTab++
		} else {
			m.selectedTab = TabOverview
		}
		// Update viewport content for new tab
		node := m.visibleNodes[m.cursor]
		var content string
		switch m.selectedTab {
		case TabOverview:
			content = m.renderOverviewTab(node)
		case TabPrompt:
			content = m.renderPromptTab(node)
		case TabResponse:
			content = m.renderResponseTab(node)
		case TabAttributes:
			content = m.renderAttributesTab(node)
		case TabTiming:
			content = m.renderTimingTab(node)
		}
		m.detailViewport.SetContent(content)
		return m, nil

	case "1":
		m.selectedTab = TabOverview
		m.detailViewport.SetContent(m.renderOverviewTab(m.visibleNodes[m.cursor]))
		return m, nil
	case "2":
		m.selectedTab = TabPrompt
		m.detailViewport.SetContent(m.renderPromptTab(m.visibleNodes[m.cursor]))
		return m, nil
	case "3":
		m.selectedTab = TabResponse
		m.detailViewport.SetContent(m.renderResponseTab(m.visibleNodes[m.cursor]))
		return m, nil
	case "4":
		m.selectedTab = TabAttributes
		m.detailViewport.SetContent(m.renderAttributesTab(m.visibleNodes[m.cursor]))
		return m, nil
	case "5":
		m.selectedTab = TabTiming
		m.detailViewport.SetContent(m.renderTimingTab(m.visibleNodes[m.cursor]))
		return m, nil

	default:
		// Pass all other keys to viewport for scrolling
		m.detailViewport, cmd = m.detailViewport.Update(msg)
	}

	return m, cmd
}

func (m *Model) updateDetailViewport() {
	if m.cursor >= len(m.visibleNodes) {
		return
	}
	node := m.visibleNodes[m.cursor]
	content := m.renderDetailContent(node)
	m.detailViewport.SetContent(content)
}

// View renders the model
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Use a fixed-height container to prevent scrolling
	var lines []string

	// 1. Global Header
	lines = append(lines, m.renderGlobalHeader())

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
	lines = append(lines, mainContent)

	// 3. Fixed Status/Help Bar at bottom
	lines = append(lines, m.renderStatusBar())

	// Join all parts
	output := strings.Join(lines, "\n")

	// Ensure we don't exceed terminal height but keep the status bar visible
	outputLines := strings.Split(output, "\n")
	if len(outputLines) > m.height {
		// Keep first lines (header) and last line (status bar), truncate middle
		keepTop := 5    // Header lines
		keepBottom := 1 // Status bar
		if len(outputLines) > keepTop+keepBottom {
			middle := m.height - keepTop - keepBottom
			if middle > 0 {
				outputLines = append(outputLines[:keepTop+middle], outputLines[len(outputLines)-keepBottom:]...)
			}
		}
		output = strings.Join(outputLines, "\n")
	}

	return output
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

func (m Model) renderStatusBar() string {
	var b strings.Builder

	// Separator line
	b.WriteString(strings.Repeat("─", m.width-4))
	b.WriteString("\n")

	// Build status bar based on current view and state
	var statusParts []string

	// Current view/focus indicator
	focusIndicator := ""
	switch m.viewMode {
	case RunListView:
		focusIndicator = "Run List"
	case TreeView:
		switch m.focusArea {
		case FocusTree:
			focusIndicator = "Tree"
		case FocusDetails:
			tabs := []string{"Overview", "Prompt", "Response", "Attributes", "Timing"}
			focusIndicator = "Details:" + tabs[m.selectedTab]
		case FocusMetadata:
			focusIndicator = "Metadata"
		}
	case DetailView:
		tabs := []string{"Overview", "Prompt", "Response", "Attributes", "Timing"}
		focusIndicator = "Detail:" + tabs[m.selectedTab]
	}
	statusParts = append(statusParts, SelectedStyle.Render(" "+focusIndicator+" "))

	// Key bindings based on current state
	var keys []string

	if m.searchMode {
		keys = []string{
			HelpKeyStyle.Render("[Type]") + " Search",
			HelpKeyStyle.Render("[Enter]") + " Confirm",
			HelpKeyStyle.Render("[Esc]") + " Cancel",
		}
	} else {
		switch m.viewMode {
		case RunListView:
			keys = []string{
				HelpKeyStyle.Render("[↑↓]") + " Navigate",
				HelpKeyStyle.Render("[Enter]") + " Open",
				HelpKeyStyle.Render("[q]") + " Quit",
			}
		case TreeView:
			keys = []string{
				HelpKeyStyle.Render("[Tab]") + " Focus",
				HelpKeyStyle.Render("[←→]") + " Tabs",
				HelpKeyStyle.Render("[↑↓]") + " Nav",
				HelpKeyStyle.Render("[h/l]") + " Fold",
				HelpKeyStyle.Render("[d]") + " Detail",
				HelpKeyStyle.Render("[/]") + " Search",
				HelpKeyStyle.Render("[e]") + " Errors",
				HelpKeyStyle.Render("[q]") + " Quit",
			}
		case DetailView:
			keys = []string{
				HelpKeyStyle.Render("[←→]") + " Tabs",
				HelpKeyStyle.Render("[1-5]") + " Jump",
				HelpKeyStyle.Render("[↑↓]") + " Scroll",
				HelpKeyStyle.Render("[Esc]") + " Back",
				HelpKeyStyle.Render("[q]") + " Quit",
			}
		}
	}

	// Add search status if active
	if len(m.searchMatches) > 0 && !m.searchMode {
		statusParts = append(statusParts, SuccessStyle.Render(fmt.Sprintf("🔍 %d matches", len(m.searchMatches))))
	}

	// Combine status and keys
	statusLine := strings.Join(statusParts, " ")
	if len(keys) > 0 {
		statusLine += "  " + strings.Join(keys, " ")
	}

	b.WriteString(HelpStyle.Render(statusLine))

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

	b.WriteString("\n")
	return b.String()
}

func (m Model) renderTreeView() string {
	var b strings.Builder

	// Count lines used for non-panel content
	usedLines := 0

	// Back indicator
	if len(m.allRuns) > 0 {
		backHint := MutedStyle.Render(fmt.Sprintf("[Esc] Back to list  |  Run %d/%d", m.selectedRun+1, len(m.allRuns)))
		b.WriteString(backHint)
		b.WriteString("\n")
		usedLines += 2
	}

	// Run Details Header (Specific to this view)
	header := m.renderRunSummary()
	b.WriteString(header)
	b.WriteString("\n")
	// Count lines in header (approximately 4-6 lines)
	usedLines += strings.Count(header, "\n") + 2

	// Calculate dimensions for 3-panel layout
	// Account for: global header (3), run header (counted above), status bar (2), search (1 if active), padding
	headerFooterLines := 3 + usedLines + 2 // status bar
	if m.searchMode {
		headerFooterLines += 1
	}

	availableWidth := m.width - 6
	availableHeight := m.height - headerFooterLines
	if availableHeight < 20 {
		availableHeight = 20 // Minimum height
	}

	// Responsive layout check
	if availableWidth < 100 {
		// Stack vertically for narrow terminals
		return m.renderStackedLayout()
	}

	// Panel widths: Left 66%, Right 34%
	leftWidth := (availableWidth * 66) / 100
	rightWidth := availableWidth - leftWidth

	// Left panel heights: Tree 40%, Details 60%
	treeHeight := (availableHeight * 40) / 100
	if treeHeight < 10 {
		treeHeight = 10
	}
	detailHeight := availableHeight - treeHeight
	if detailHeight < 8 {
		detailHeight = 8
		treeHeight = availableHeight - detailHeight
	}

	// Render three panels
	treeContent := m.renderTreePanel()
	detailContent := m.renderDetailPanel()
	metadataContent := m.renderMetadataPanel()

	// Apply focus styling
	treeStyle := LeftPaneStyle.Width(leftWidth).Height(treeHeight)
	detailStyle := LeftPaneStyle.Width(leftWidth).Height(detailHeight)
	metadataStyle := RightPaneStyle.Width(rightWidth).Height(availableHeight)

	if m.focusArea == FocusTree {
		treeStyle = treeStyle.BorderForeground(lipgloss.Color("#06B6D4")).BorderStyle(lipgloss.ThickBorder())
	}
	if m.focusArea == FocusDetails {
		detailStyle = detailStyle.BorderForeground(lipgloss.Color("#06B6D4")).BorderStyle(lipgloss.ThickBorder())
	}
	if m.focusArea == FocusMetadata {
		metadataStyle = metadataStyle.BorderForeground(lipgloss.Color("#06B6D4")).BorderStyle(lipgloss.ThickBorder())
	}

	// Build left column (tree + details stacked)
	leftColumn := lipgloss.JoinVertical(
		lipgloss.Left,
		treeStyle.Render(treeContent),
		detailStyle.Render(detailContent),
	)

	// Join left and right columns
	splitView := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftColumn,
		metadataStyle.Render(metadataContent),
	)
	b.WriteString(splitView)

	// Search bar (if active)
	if m.searchMode {
		b.WriteString("\n")
		b.WriteString(m.renderSearchBar())
	}

	return b.String()
}

// renderTreePanel renders the trace tree panel
func (m Model) renderTreePanel() string {
	var b strings.Builder

	title := "Trace Tree"
	if m.focusArea == FocusTree {
		title = "▶ " + title
	}
	b.WriteString(HeaderStyle.Render(title))
	b.WriteString("\n")

	// Build full content for viewport
	var content strings.Builder
	for i, node := range m.visibleNodes {
		line := m.renderSpanLine(node, i == m.cursor)
		content.WriteString(line)
		content.WriteString("\n")
	}

	// Set viewport content
	m.treeViewport.SetContent(content.String())

	// Auto-scroll to cursor
	if m.cursor < len(m.visibleNodes) {
		// Calculate line position and ensure it's visible
		if m.cursor < m.treeViewport.YOffset {
			m.treeViewport.YOffset = m.cursor
		} else if m.cursor >= m.treeViewport.YOffset+m.treeViewport.Height {
			m.treeViewport.YOffset = m.cursor - m.treeViewport.Height + 1
		}
	}

	b.WriteString(m.treeViewport.View())
	return b.String()
}

// renderDetailPanel renders the details panel for selected node
func (m Model) renderDetailPanel() string {
	var b strings.Builder

	// Tab bar
	tabs := []string{"Overview", "Prompt", "Response", "Attributes", "Timing"}
	var tabBar strings.Builder
	for i, tab := range tabs {
		if DetailTab(i) == m.selectedTab {
			// Active tab - highlighted
			if m.focusArea == FocusDetails {
				tabBar.WriteString(SelectedStyle.Bold(true).Render(" " + tab + " "))
			} else {
				tabBar.WriteString(SelectedStyle.Render(" " + tab + " "))
			}
		} else {
			// Inactive tab
			tabBar.WriteString(MutedStyle.Render(" " + tab + " "))
		}
		if i < len(tabs)-1 {
			tabBar.WriteString(MutedStyle.Render("│"))
		}
	}
	b.WriteString(tabBar.String())
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n")

	if m.cursor >= len(m.visibleNodes) {
		b.WriteString(MutedStyle.Render("No span selected"))
		return b.String()
	}

	node := m.visibleNodes[m.cursor]

	// Render content based on selected tab
	var content string
	switch m.selectedTab {
	case TabOverview:
		content = m.renderOverviewTab(node)
	case TabPrompt:
		content = m.renderPromptTab(node)
	case TabResponse:
		content = m.renderResponseTab(node)
	case TabAttributes:
		content = m.renderAttributesTab(node)
	case TabTiming:
		content = m.renderTimingTab(node)
	}

	// Set viewport content
	m.detailViewport.SetContent(content)
	b.WriteString(m.detailViewport.View())

	return b.String()
}

// renderOverviewTab renders the overview tab content
func (m Model) renderOverviewTab(node *SpanNode) string {
	var b strings.Builder

	b.WriteString(SectionHeaderStyle.Render("Overview"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("%-12s %s\n", "Name:", node.Span.GetFriendlyName()))
	b.WriteString(fmt.Sprintf("%-12s %s\n", "Type:", node.Span.GetSpanType()))
	b.WriteString(fmt.Sprintf("%-12s %dms\n", "Duration:", node.DurationMs))

	// Status
	statusText := "OK"
	statusStyle := SuccessStyle
	if node.Span.Status.Code != "" && node.Span.Status.Code != StatusUnset && node.Span.Status.Code != "Ok" {
		statusText = node.Span.Status.Code
		statusStyle = ErrorStyle
	}
	b.WriteString(fmt.Sprintf("%-12s %s\n", "Status:", statusStyle.Render(statusText)))

	if node.Span.Status.Description != "" {
		b.WriteString(fmt.Sprintf("%-12s %s\n", "Message:", node.Span.Status.Description))
	}

	// Resource usage summary
	attrs := node.Span.GetAllAttributes()
	if tokens, ok := attrs["llm.usage.total_tokens"]; ok {
		b.WriteString("\n")
		b.WriteString(SectionHeaderStyle.Render("Resource Usage"))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("%-12s %v\n", "Tokens:", tokens))
		if promptTokens, ok := attrs["llm.usage.prompt_tokens"]; ok {
			b.WriteString(fmt.Sprintf("%-12s %v\n", "  Prompt:", promptTokens))
		}
		if completionTokens, ok := attrs["llm.usage.completion_tokens"]; ok {
			b.WriteString(fmt.Sprintf("%-12s %v\n", "  Response:", completionTokens))
		}
	}

	if model, ok := attrs["llm.model"]; ok {
		b.WriteString(fmt.Sprintf("%-12s %v\n", "Model:", model))
	}

	return b.String()
}

// renderPromptTab renders the prompt tab content
func (m Model) renderPromptTab(node *SpanNode) string {
	var b strings.Builder
	attrs := node.Span.GetAllAttributes()

	// System Prompt
	if systemPrompt, ok := attrs["agk.prompt.system"]; ok {
		b.WriteString(SectionHeaderStyle.Render("System Prompt"))
		b.WriteString("\n\n")
		b.WriteString(systemPrompt.(string))
		b.WriteString("\n\n")
	}

	// User Prompt
	if userPrompt, ok := attrs["agk.prompt.user"]; ok {
		b.WriteString(SectionHeaderStyle.Render("User Prompt"))
		b.WriteString("\n\n")
		b.WriteString(userPrompt.(string))
		b.WriteString("\n\n")
	}

	// Messages (if structured)
	if messages, ok := attrs["llm.request.messages"]; ok {
		b.WriteString(SectionHeaderStyle.Render("Messages"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("%v", messages))
		b.WriteString("\n\n")
	}

	if b.Len() == 0 {
		b.WriteString(MutedStyle.Render("No prompt data available for this span"))
	}

	return b.String()
}

// renderResponseTab renders the response tab content
func (m Model) renderResponseTab(node *SpanNode) string {
	var b strings.Builder
	attrs := node.Span.GetAllAttributes()

	// Response Text
	if response, ok := attrs["agk.llm.response"]; ok {
		b.WriteString(SectionHeaderStyle.Render("Response Text"))
		b.WriteString("\n\n")
		b.WriteString(response.(string))
		b.WriteString("\n\n")
	}

	// Tool Results
	if toolResult, ok := attrs["agk.tool.result"]; ok {
		b.WriteString(SectionHeaderStyle.Render("Tool Result"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("%v", toolResult))
		b.WriteString("\n\n")
	}

	// Finish Reason
	if finishReason, ok := attrs["llm.response.finish_reason"]; ok {
		b.WriteString(SectionHeaderStyle.Render("Finish Reason"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("%v", finishReason))
		b.WriteString("\n\n")
	}

	if b.Len() == 0 {
		b.WriteString(MutedStyle.Render("No response data available for this span"))
	}

	return b.String()
}

// renderAttributesTab renders all attributes in table format
func (m Model) renderAttributesTab(node *SpanNode) string {
	var b strings.Builder
	attrs := node.Span.GetAllAttributes()

	b.WriteString(SectionHeaderStyle.Render("All Attributes"))
	b.WriteString("\n\n")

	if len(attrs) == 0 {
		b.WriteString(MutedStyle.Render("No attributes available"))
		return b.String()
	}

	// Sort keys for consistent display
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Display as key-value table
	for _, k := range keys {
		v := attrs[k]
		// Clean up key for display
		displayKey := k
		displayKey = strings.TrimPrefix(displayKey, "agk.")
		displayKey = strings.TrimPrefix(displayKey, "llm.")
		displayKey = strings.TrimPrefix(displayKey, "workflow.")

		b.WriteString(fmt.Sprintf("%-30s %v\n", AttributeKeyStyle.Render(displayKey+":"), v))
	}

	return b.String()
}

// renderTimingTab renders timing information and breakdown
func (m Model) renderTimingTab(node *SpanNode) string {
	var b strings.Builder
	attrs := node.Span.GetAllAttributes()

	b.WriteString(SectionHeaderStyle.Render("Timing Details"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("%-15s %dms\n", "Duration:", node.DurationMs))
	b.WriteString(fmt.Sprintf("%-15s %s\n", "Start Time:", node.Span.StartTime))
	b.WriteString(fmt.Sprintf("%-15s %s\n", "End Time:", node.Span.EndTime))
	b.WriteString("\n")

	// Timing breakdown if child spans exist
	if len(node.Children) > 0 {
		b.WriteString(SectionHeaderStyle.Render("Child Spans"))
		b.WriteString("\n\n")

		var totalChildTime int64
		for _, child := range node.Children {
			totalChildTime += child.DurationMs
			percentage := float64(child.DurationMs) / float64(node.DurationMs) * 100

			bar := ""
			barWidth := int(percentage / 2) // 50 chars max
			if barWidth > 0 {
				bar = strings.Repeat("█", barWidth)
			}

			b.WriteString(fmt.Sprintf("%-30s %5dms %6.1f%% %s\n",
				child.Span.GetFriendlyName(),
				child.DurationMs,
				percentage,
				DurationStyle.Render(bar)))
		}

		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("%-30s %5dms\n", "Total Child Time:", totalChildTime))

		selfTime := node.DurationMs - totalChildTime
		if selfTime > 0 {
			b.WriteString(fmt.Sprintf("%-30s %5dms\n", "Self Time:", selfTime))
		}
	}

	// Performance markers if available
	if ttft, ok := attrs["llm.time_to_first_token"]; ok {
		b.WriteString("\n")
		b.WriteString(SectionHeaderStyle.Render("Performance Metrics"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("%-25s %v\n", "Time to First Token:", ttft))
	}

	return b.String()
}

// renderMetadataPanel renders the metadata/diagnostics panel
func (m Model) renderMetadataPanel() string {
	var b strings.Builder

	title := "Metadata"
	if m.focusArea == FocusMetadata {
		title = "▶ " + title
	}
	b.WriteString(HeaderStyle.Render(title))
	b.WriteString(" " + MutedStyle.Render("[↑↓] Scroll"))
	b.WriteString("\n")

	if m.cursor >= len(m.visibleNodes) {
		return b.String()
	}

	node := m.visibleNodes[m.cursor]
	attrs := node.Span.GetAllAttributes()

	// Build full content for viewport
	var content strings.Builder

	// === PINNED SECTIONS ===

	// Identity
	content.WriteString(SectionHeaderStyle.Render("Identity"))
	content.WriteString("\n")
	content.WriteString(fmt.Sprintf("%-12s %s\n", "Type:", node.Span.GetSpanType()))
	content.WriteString(fmt.Sprintf("%-12s %s\n", "Span ID:", MutedStyle.Render(node.Span.SpanContext.SpanID[:8]+"...")))
	if node.Parent != nil {
		content.WriteString(fmt.Sprintf("%-12s %s\n", "Parent:", MutedStyle.Render(node.Parent.Span.SpanContext.SpanID[:8]+"...")))
	}
	content.WriteString("\n")

	// Status
	content.WriteString(SectionHeaderStyle.Render("Status"))
	content.WriteString("\n")
	statusText := "OK"
	statusStyle := SuccessStyle
	if node.Span.Status.Code != "" && node.Span.Status.Code != StatusUnset && node.Span.Status.Code != "Ok" {
		statusText = node.Span.Status.Code
		statusStyle = ErrorStyle
	}
	content.WriteString(fmt.Sprintf("%-12s %s\n", "Status:", statusStyle.Render(statusText)))
	content.WriteString("\n")

	// Timing
	content.WriteString(SectionHeaderStyle.Render("Timing"))
	content.WriteString("\n")
	content.WriteString(fmt.Sprintf("%-12s %dms\n", "Duration:", node.DurationMs))
	content.WriteString(fmt.Sprintf("%-12s %s\n", "Start:", MutedStyle.Render(node.Span.StartTime)))
	content.WriteString("\n")

	// === SCROLLABLE SECTIONS ===

	// Resources
	if tokens, ok := attrs["llm.usage.total_tokens"]; ok {
		content.WriteString(SectionHeaderStyle.Render("Resources"))
		content.WriteString("\n")
		content.WriteString(fmt.Sprintf("%-12s %v\n", "Tokens:", tokens))
		if cost := float64(node.DurationMs) * 0.000001; cost > 0 {
			content.WriteString(fmt.Sprintf("%-12s %s\n", "Est. Cost:", WarningStyle.Render(fmt.Sprintf("$%.6f", cost))))
		}
		content.WriteString("\n")
	}

	// Errors
	if node.Span.Status.Code != "" && node.Span.Status.Code != StatusUnset && node.Span.Status.Code != "Ok" {
		content.WriteString(SectionHeaderStyle.Render("Error"))
		content.WriteString("\n")
		content.WriteString(ErrorStyle.Render(node.Span.Status.Description))
		content.WriteString("\n\n")
	}

	// Tags (all attributes)
	content.WriteString(SectionHeaderStyle.Render("All Attributes"))
	content.WriteString("\n")
	for k, v := range attrs {
		shortKey := strings.TrimPrefix(k, "agk.")
		shortKey = strings.TrimPrefix(shortKey, "llm.")
		shortKey = strings.TrimPrefix(shortKey, "workflow.")
		content.WriteString(fmt.Sprintf("%-20s %v\n", shortKey+":", v))
	}

	// Set viewport content
	m.metadataViewport.SetContent(content.String())

	b.WriteString(m.metadataViewport.View())
	return b.String()
}

// renderStackedLayout renders panels vertically for narrow terminals
func (m Model) renderStackedLayout() string {
	var b strings.Builder

	b.WriteString(WarningStyle.Render("⚠ Terminal narrow - stacked layout"))
	b.WriteString("\n\n")

	// Tree first
	treeContent := m.renderTreePanel()
	b.WriteString(BoxStyle.Render(treeContent))
	b.WriteString("\n")

	// Details second
	if m.cursor < len(m.visibleNodes) {
		detailContent := m.renderDetailPanel()
		b.WriteString(BoxStyle.Render(detailContent))
		b.WriteString("\n")
	}

	// Metadata last
	if m.cursor < len(m.visibleNodes) {
		metadataContent := m.renderMetadataPanel()
		b.WriteString(BoxStyle.Render(metadataContent))
	}

	return b.String()
}

// renderSearchBar renders the search input bar
func (m Model) renderSearchBar() string {
	prompt := "Search: " + m.searchQuery + "█"
	if len(m.searchMatches) > 0 {
		prompt += fmt.Sprintf(" (%d matches)", len(m.searchMatches))
	}
	return BoxStyle.Render(prompt)
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

	// Use friendly name for cleaner display
	friendlyName := node.Span.GetFriendlyName()
	spanStyle := GetSpanStyle(node.Span.Name)
	name := spanStyle.Render(friendlyName)

	// Get additional context from attributes (only if not already in friendly name)
	var context string
	attrs := node.Span.GetAllAttributes()

	// For workflow steps, show model info as context
	if node.Span.IsWorkflowStep() {
		if model, ok := attrs["agk.llm.model"]; ok {
			context = MutedStyle.Render(fmt.Sprintf(" [%v]", model))
		}
	}

	// Error indicator
	errorIndicator := ""
	if node.Span.Status.Code != "" && node.Span.Status.Code != "Unset" && node.Span.Status.Code != "Ok" {
		errorIndicator = ErrorStyle.Render(" [ERR]")
	}

	// Search match indicator
	searchIndicator := ""
	if m.isSearchMatch(node) {
		searchIndicator = " 🔍"
	}

	// Duration
	duration := DurationStyle.Render(fmt.Sprintf("(%dms)", node.DurationMs))

	// Build line
	line := fmt.Sprintf("%s%s%s%s%s%s %s", indent, prefix, name, context, errorIndicator, searchIndicator, duration)

	// Apply selection styling
	if selected {
		line = CursorStyle.Render("→ ") + SelectedStyle.Render(line)
	} else {
		line = "  " + line
	}

	return line
}

// isSearchMatch checks if a node is in current search results
func (m Model) isSearchMatch(node *SpanNode) bool {
	for _, match := range m.searchMatches {
		if match == node {
			return true
		}
	}
	return false
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
	b.WriteString(strings.Repeat("─", m.width-4))
	b.WriteString("\n")

	// Tab bar (same as in renderDetailPanel)
	tabs := []string{"Overview", "Prompt", "Response", "Attributes", "Timing"}
	var tabBar strings.Builder
	for i, tab := range tabs {
		if DetailTab(i) == m.selectedTab {
			tabBar.WriteString(SelectedStyle.Bold(true).Render(" " + tab + " "))
		} else {
			tabBar.WriteString(MutedStyle.Render(" " + tab + " "))
		}
		if i < len(tabs)-1 {
			tabBar.WriteString(MutedStyle.Render("│"))
		}
	}
	b.WriteString(tabBar.String())
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width-4))
	b.WriteString("\n\n")

	// Viewport content
	b.WriteString(m.detailViewport.View())

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

// jumpToSearchMatch moves cursor to current search match
func (m Model) jumpToSearchMatch() Model {
	if m.searchIndex < 0 || m.searchIndex >= len(m.searchMatches) {
		return m
	}

	match := m.searchMatches[m.searchIndex]
	// Find this node in visible nodes
	for i, node := range m.visibleNodes {
		if node == match {
			m.cursor = i
			m.focusArea = FocusTree
			break
		}
	}
	return m
}

// jumpToNextError finds and jumps to the next error node
func (m Model) jumpToNextError() Model {
	if m.errorCount == 0 {
		return m
	}

	// Search from current cursor position forward
	for i := m.cursor + 1; i < len(m.visibleNodes); i++ {
		if m.isErrorNode(m.visibleNodes[i]) {
			m.cursor = i
			m.focusArea = FocusTree
			// Expand parent if needed
			m = m.ensureNodeVisible(m.visibleNodes[i])
			return m
		}
	}

	// Wrap around to beginning
	for i := 0; i <= m.cursor; i++ {
		if m.isErrorNode(m.visibleNodes[i]) {
			m.cursor = i
			m.focusArea = FocusTree
			m = m.ensureNodeVisible(m.visibleNodes[i])
			return m
		}
	}

	return m
}

// jumpToPreviousError finds and jumps to the previous error node
func (m Model) jumpToPreviousError() Model {
	if m.errorCount == 0 {
		return m
	}

	// Search from current cursor position backward
	for i := m.cursor - 1; i >= 0; i-- {
		if m.isErrorNode(m.visibleNodes[i]) {
			m.cursor = i
			m.focusArea = FocusTree
			m = m.ensureNodeVisible(m.visibleNodes[i])
			return m
		}
	}

	// Wrap around to end
	for i := len(m.visibleNodes) - 1; i >= m.cursor; i-- {
		if m.isErrorNode(m.visibleNodes[i]) {
			m.cursor = i
			m.focusArea = FocusTree
			m = m.ensureNodeVisible(m.visibleNodes[i])
			return m
		}
	}

	return m
}

// isErrorNode checks if a node has an error status
func (m Model) isErrorNode(node *SpanNode) bool {
	return node.Span.Status.Code != "" &&
		node.Span.Status.Code != StatusUnset &&
		node.Span.Status.Code != "Ok"
}

// ensureNodeVisible expands parent nodes to make a node visible
func (m Model) ensureNodeVisible(node *SpanNode) Model {
	// Walk up the tree and expand all parents
	current := node.Parent
	for current != nil {
		if !current.Expanded {
			current.Expanded = true
		}
		current = current.Parent
	}
	// Rebuild visible list
	m.visibleNodes = FlattenTree(m.roots)
	return m
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
