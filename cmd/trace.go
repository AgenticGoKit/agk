package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/agenticgokit/agk/internal/audit"
	"github.com/agenticgokit/agk/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

const runsDirName = ".agk/runs"

// traceCmd represents the trace command
var traceCmd = &cobra.Command{
	Use:   "trace",
	Short: "Manage and view execution traces",
	Long: `Manage and view execution traces from AgenticGoKit runs.

Traces are automatically stored in .agk/runs/<run-id>/ when AGK_TRACE=true.

Examples:
  agk trace                   # Launch interactive trace explorer
  agk trace list              # List all stored traces
  agk trace show <run-id>     # Display trace details in TUI
  agk trace view <run-id>     # Show run manifest/summary
  agk trace export <run-id>   # Export trace for external tools
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return launchTraceExplorer()
	},
}

// listCmd shows all stored traces
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all stored traces",
	RunE: func(cmd *cobra.Command, args []string) error {
		return listTraces()
	},
}

// showCmd displays trace details in interactive viewer
var showCmd = &cobra.Command{
	Use:   "show [run-id]",
	Short: "Show trace in interactive viewer",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := ""
		if len(args) > 0 {
			runID = args[0]
		}
		return showTrace(runID)
	},
}

// viewCmd shows run manifest/summary
var viewCmd = &cobra.Command{
	Use:   "view [run-id]",
	Short: "View run summary and manifest",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := ""
		if len(args) > 0 {
			runID = args[0]
		}
		return viewRun(runID)
	},
}

// exportCmd exports trace for external tools
var exportCmd = &cobra.Command{
	Use:   "export [run-id]",
	Short: "Export trace for external tools",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := ""
		if len(args) > 0 {
			runID = args[0]
		}

		format, _ := cmd.Flags().GetString("format")
		output, _ := cmd.Flags().GetString("output")

		return exportTraceInternal(runID, format, output)
	},
}

// auditCmd analyzes trace for reasoning patterns
var auditCmd = &cobra.Command{
	Use:   "audit [run-id]",
	Short: "Analyze trace for reasoning patterns",
	Long: `Analyze a trace to extract reasoning events for evaluation.

Outputs a TraceObject with events categorized as:
  - thought: Internal reasoning/decisions
  - tool_call: Tool invocations with arguments
  - observation: Tool outputs/results
  - llm_call: LLM API calls

Use AGK_TRACE_LEVEL=detailed when running your agent to capture
full content (prompts, responses, tool args/outputs).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := ""
		if len(args) > 0 {
			runID = args[0]
		}
		return auditTrace(runID)
	},
}

// mermaidCmd generates Mermaid diagram from trace
var mermaidCmd = &cobra.Command{
	Use:   "mermaid [run-id]",
	Short: "Generate Mermaid diagram from trace",
	Long: `Generate a Mermaid flowchart visualizing the agent's execution path.

The diagram shows the sequence of thoughts, tool calls, and decisions
made by the agent. Output is Markdown with embedded Mermaid code.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := ""
		if len(args) > 0 {
			runID = args[0]
		}
		output, _ := cmd.Flags().GetString("output")
		return generateMermaid(runID, output)
	},
}

func init() {
	rootCmd.AddCommand(traceCmd)
	traceCmd.AddCommand(listCmd)
	traceCmd.AddCommand(showCmd)
	traceCmd.AddCommand(viewCmd)
	traceCmd.AddCommand(exportCmd)
	traceCmd.AddCommand(auditCmd)
	traceCmd.AddCommand(mermaidCmd)

	// Export flags
	exportCmd.Flags().String("format", "json", "Export format: json, jaeger, otel")
	exportCmd.Flags().String("output", "", "Output file (default: stdout)")
}

// TraceRun represents a stored trace run
type TraceRun struct {
	RunID         string    `json:"run_id"`
	Command       string    `json:"command"`
	Status        string    `json:"status"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	Duration      float64   `json:"duration_seconds"`
	SpanCount     int       `json:"span_count"`
	LLMCalls      int       `json:"llm_calls"`
	TotalTokens   int       `json:"total_tokens"`
	EstimatedCost float64   `json:"estimated_cost"`
}

// launchTraceExplorer launches the unified trace explorer TUI
func launchTraceExplorer() error {
	runsDir := runsDirName

	// Check if directory exists
	if _, err := os.Stat(runsDir); os.IsNotExist(err) {
		fmt.Println("No traces found. Run with AGK_TRACE=true to generate traces.")
		return nil
	}

	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return fmt.Errorf("failed to read runs directory: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No traces found. Run with AGK_TRACE=true to generate traces.")
		return nil
	}

	// Load all runs with their spans
	var runDataList []tui.RunData
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		runPath := filepath.Join(runsDir, entry.Name())
		manifest, err := readManifest(runPath)
		if err != nil {
			continue
		}

		// Read spans
		tracePath := filepath.Join(runPath, "trace.jsonl")
		data, err := os.ReadFile(tracePath)
		if err != nil {
			continue
		}
		spans := tui.ParseSpans(string(data))

		runDataList = append(runDataList, tui.RunData{
			Manifest: tui.TraceRun{
				RunID:         manifest.RunID,
				Command:       manifest.Command,
				Status:        manifest.Status,
				Duration:      manifest.Duration,
				SpanCount:     manifest.SpanCount,
				LLMCalls:      manifest.LLMCalls,
				TotalTokens:   manifest.TotalTokens,
				EstimatedCost: manifest.EstimatedCost,
			},
			Spans: spans,
		})
	}

	if len(runDataList) == 0 {
		fmt.Println("No valid traces found.")
		return nil
	}

	// Sort by newest first (assuming run IDs contain timestamps)
	sort.Slice(runDataList, func(i, j int) bool {
		return runDataList[i].Manifest.RunID > runDataList[j].Manifest.RunID
	})

	// Create and run TUI explorer
	model := tui.NewTraceExplorer(runDataList)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

func listTraces() error {
	runsDir := runsDirName

	// Create directory if it doesn't exist
	if _, err := os.Stat(runsDir); os.IsNotExist(err) {
		fmt.Println("No traces found. Run with AGK_TRACE=true to generate traces.")
		return nil
	}

	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return fmt.Errorf("failed to read runs directory: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No traces found. Run with AGK_TRACE=true to generate traces.")
		return nil
	}

	// Parse all runs
	var runs []TraceRun
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifest, err := readManifest(filepath.Join(runsDir, entry.Name()))
		if err != nil {
			continue // Skip runs without valid manifest
		}
		runs = append(runs, manifest)
	}

	if len(runs) == 0 {
		fmt.Println("No valid traces found.")
		return nil
	}

	// Sort by start time (newest first)
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].StartTime.After(runs[j].StartTime)
	})

	// Print table
	fmt.Println()
	fmt.Printf("%-40s %-12s %-8s %-10s %-10s %-12s\n",
		"Run ID", "Command", "Status", "Duration", "LLM Calls", "Tokens")
	fmt.Println(strings.Repeat("-", 92))

	for _, run := range runs {
		status := "✅ OK"
		if run.Status != "completed" && run.Status != "ok" {
			status = "❌ ERROR"
		}

		duration := fmt.Sprintf("%.2fs", run.Duration)
		llmCalls := fmt.Sprintf("%d", run.LLMCalls)
		tokens := fmt.Sprintf("%d", run.TotalTokens)

		fmt.Printf("%-40s %-12s %-8s %-10s %-10s %-12s\n",
			run.RunID, run.Command, status, duration, llmCalls, tokens)
	}
	fmt.Println()

	return nil
}

func showTrace(runID string) error {
	runsDir := runsDirName

	// If no run ID provided, use latest
	if runID == "" {
		runID = getLatestRunID()
		if runID == "" {
			fmt.Println("No traces found. Run with AGK_TRACE=true to generate traces.")
			return nil
		}
	}

	runPath := filepath.Join(runsDir, runID)

	// Check if run exists
	if _, err := os.Stat(runPath); os.IsNotExist(err) {
		return fmt.Errorf("trace not found: %s", runID)
	}

	// Read trace file
	tracePath := filepath.Join(runPath, "trace.jsonl")
	data, err := os.ReadFile(tracePath)
	if err != nil {
		return fmt.Errorf("failed to read trace: %w", err)
	}

	// Parse spans using TUI package
	spans := tui.ParseSpans(string(data))
	manifest, _ := readManifest(runPath)

	// Convert manifest to TUI format
	tuiManifest := tui.TraceRun{
		RunID:         manifest.RunID,
		Command:       manifest.Command,
		Status:        manifest.Status,
		Duration:      manifest.Duration,
		SpanCount:     manifest.SpanCount,
		LLMCalls:      manifest.LLMCalls,
		TotalTokens:   manifest.TotalTokens,
		EstimatedCost: manifest.EstimatedCost,
	}

	// Create and run TUI with hot reload support
	model := tui.NewTraceViewerWithPath(runID, tuiManifest, spans, tracePath)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

func viewRun(runID string) error {
	runsDir := runsDirName

	// If no run ID provided, use latest
	if runID == "" {
		runID = getLatestRunID()
		if runID == "" {
			fmt.Println("No traces found. Run with AGK_TRACE=true to generate traces.")
			return nil
		}
	}

	runPath := filepath.Join(runsDir, runID)

	// Check if run exists
	if _, err := os.Stat(runPath); os.IsNotExist(err) {
		return fmt.Errorf("trace not found: %s", runID)
	}

	manifest, err := readManifest(runPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	// Display manifest
	fmt.Println()
	fmt.Printf("Run Information\n")
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Run ID:              %s\n", manifest.RunID)
	fmt.Printf("Command:             %s\n", manifest.Command)
	fmt.Printf("Status:              ✅ %s\n", manifest.Status)
	fmt.Printf("Started:             %s\n", manifest.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Completed:           %s\n", manifest.EndTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Duration:            %.2fs\n", manifest.Duration)
	fmt.Println()
	fmt.Printf("Execution Stats\n")
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Spans:               %d\n", manifest.SpanCount)
	fmt.Printf("LLM Calls:           %d\n", manifest.LLMCalls)
	fmt.Printf("Total Tokens:        %d\n", manifest.TotalTokens)
	fmt.Printf("Estimated Cost:      $%.4f\n", manifest.EstimatedCost)
	fmt.Println()
	fmt.Printf("Files\n")
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Trace:               %s/trace.jsonl\n", runPath)
	fmt.Printf("Events:              %s/events.jsonl\n", runPath)
	fmt.Printf("Manifest:            %s/manifest.json\n", runPath)
	fmt.Println()

	return nil
}

func exportTraceInternal(runID, format, output string) error {
	runsDir := runsDirName

	// If no run ID provided, use latest
	if runID == "" {
		runID = getLatestRunID()
		if runID == "" {
			fmt.Println("No traces found. Run with AGK_TRACE=true to generate traces.")
			return nil
		}
	}

	runPath := filepath.Join(runsDir, runID)
	tracePath := filepath.Join(runPath, "trace.jsonl")

	// Read trace data
	data, err := os.ReadFile(tracePath)
	if err != nil {
		return fmt.Errorf("failed to read trace: %w", err)
	}

	// Parse JSONL into spans
	lines := strings.Split(string(data), "\n")
	var spans []map[string]interface{}
	for _, line := range lines {
		if line == "" {
			continue
		}
		var span map[string]interface{}
		if err := json.Unmarshal([]byte(line), &span); err != nil {
			continue
		}
		spans = append(spans, span)
	}

	// Format and export based on format flag
	var exportData interface{}

	switch format {
	case "json":
		// Raw JSONL as JSON array
		exportData = spans

	case "jaeger":
		// Convert to Jaeger format
		exportData = convertToJaegerFormat(spans, runID)

	case "otel", "otlp":
		// Convert to OpenTelemetry format
		exportData = convertToOTLPFormat(spans, runID)

	default:
		return fmt.Errorf("unknown format: %s (supported: json, jaeger, otel)", format)
	}

	// Marshal data
	exportBytes, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Write output
	if output != "" {
		if err := os.WriteFile(output, exportBytes, 0600); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("✅ Exported trace to %s (format: %s)\n", output, format)
	} else {
		fmt.Println(string(exportBytes))
	}

	return nil
}

// convertToJaegerFormat converts OpenTelemetry spans to Jaeger format
func convertToJaegerFormat(spans []map[string]interface{}, _ string) map[string]interface{} {
	jaegerSpans := make([]map[string]interface{}, 0)

	for _, span := range spans {
		jaegerSpan := map[string]interface{}{}

		// Extract and map fields
		if traceID, ok := span["SpanContext"].(map[string]interface{})["TraceID"]; ok {
			jaegerSpan["traceID"] = traceID
		}
		if spanID, ok := span["SpanContext"].(map[string]interface{})["SpanID"]; ok {
			jaegerSpan["spanID"] = spanID
		}
		if name, ok := span["Name"]; ok {
			jaegerSpan["operationName"] = name
		}
		if startTime, ok := span["StartTime"]; ok {
			jaegerSpan["startTime"] = startTime
		}
		if endTime, ok := span["EndTime"]; ok {
			jaegerSpan["endTime"] = endTime
		}

		// Map attributes to tags
		if attrs, ok := span["Attributes"].([]interface{}); ok {
			tags := make([]map[string]interface{}, 0)
			for _, attr := range attrs {
				if attrMap, ok := attr.(map[string]interface{}); ok {
					tag := map[string]interface{}{
						"key":   attrMap["Key"],
						"value": attrMap["Value"],
					}
					tags = append(tags, tag)
				}
			}
			jaegerSpan["tags"] = tags
		}

		jaegerSpans = append(jaegerSpans, jaegerSpan)
	}

	return map[string]interface{}{
		"traceID": getTraceID(spans),
		"spans":   jaegerSpans,
	}
}

// convertToOTLPFormat converts to OpenTelemetry Protocol format
func convertToOTLPFormat(spans []map[string]interface{}, _ string) map[string]interface{} {
	return map[string]interface{}{
		"resourceSpans": []map[string]interface{}{
			{
				"resource": map[string]interface{}{
					"attributes": []map[string]interface{}{
						{
							"key": "service.name",
							"value": map[string]interface{}{
								"stringValue": "agenticgokit",
							},
						},
						{
							"key": "service.version",
							"value": map[string]interface{}{
								"stringValue": "0.6.0",
							},
						},
					},
				},
				"scopeSpans": []map[string]interface{}{
					{
						"scope": map[string]interface{}{
							"name": "agenticgokit",
						},
						"spans": spans,
					},
				},
			},
		},
	}
}

// getTraceID extracts the trace ID from spans
func getTraceID(spans []map[string]interface{}) string {
	if len(spans) > 0 {
		if spanCtx, ok := spans[0]["SpanContext"].(map[string]interface{}); ok {
			if traceID, ok := spanCtx["TraceID"]; ok {
				return traceID.(string)
			}
		}
	}
	return ""
}

// Helper functions

func readManifest(runPath string) (TraceRun, error) {
	// First try to read manifest.json if it exists
	manifestPath := filepath.Join(runPath, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err == nil {
		var manifest TraceRun
		if err := json.Unmarshal(data, &manifest); err == nil {
			return manifest, nil
		}
	}

	// Fallback: parse trace.jsonl and create synthetic manifest
	return parseTraceFile(runPath)
}

// parseTraceFile reads trace.jsonl and creates a TraceRun from the trace data
// parseTraceFile reads trace.jsonl and creates a TraceRun from the trace data
func parseTraceFile(runPath string) (TraceRun, error) {
	tracePath := filepath.Join(runPath, "trace.jsonl")
	data, err := os.ReadFile(tracePath)
	if err != nil {
		return TraceRun{}, fmt.Errorf("no trace file found: %w", err)
	}

	runID := filepath.Base(runPath)
	stats := &RunStats{}

	// Parse JSONL to extract span information
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := scanner.Bytes()
		var span map[string]interface{}
		if err := json.Unmarshal(line, &span); err != nil {
			continue
		}
		stats.Update(span)
	}

	if stats.FirstSpan.IsZero() {
		stats.FirstSpan = time.Now()
	}
	if stats.LastSpan.IsZero() {
		stats.LastSpan = stats.FirstSpan
	}

	// Parse run ID to extract command name
	// Format: run-{timestamp} or run-{timestamp}-{command}
	command := "agent"
	if parts := strings.Split(runID, "-"); len(parts) > 2 {
		command = strings.Join(parts[2:], "-")
	}

	durationSeconds := stats.LastSpan.Sub(stats.FirstSpan).Seconds()
	estimatedCost := float64(stats.TotalTokens) * 0.00001 // Rough estimate

	return TraceRun{
		RunID:         runID,
		Command:       command,
		Status:        "completed",
		StartTime:     stats.FirstSpan,
		EndTime:       stats.LastSpan,
		Duration:      durationSeconds,
		SpanCount:     stats.SpanCount,
		LLMCalls:      stats.LLMCalls,
		TotalTokens:   stats.TotalTokens,
		EstimatedCost: estimatedCost,
	}, nil
}

type RunStats struct {
	SpanCount   int
	LLMCalls    int
	TotalTokens int
	FirstSpan   time.Time
	LastSpan    time.Time
}

func (s *RunStats) Update(span map[string]interface{}) {
	s.SpanCount++

	// Check if this is an LLM span
	if spanName, ok := span["Name"].(string); ok {
		if strings.Contains(spanName, "llm") {
			s.LLMCalls++
		}
	}

	// Extract token count from attributes
	if attrs, ok := span["Attributes"].([]interface{}); ok {
		s.extractTokens(attrs)
	}

	// Extract start and end times
	s.updateTimes(span)
}

func (s *RunStats) extractTokens(attrs []interface{}) {
	for _, attr := range attrs {
		if attrMap, ok := attr.(map[string]interface{}); ok {
			if key, ok := attrMap["Key"].(string); ok {
				// Look for token-related attributes
				if key == "llm.usage.completion_tokens" || key == "llm.completion_tokens" {
					if val, ok := attrMap["Value"].(map[string]interface{}); ok {
						if tokenVal, ok := val["Value"]; ok {
							if tokenInt, err := toInt64(tokenVal); err == nil {
								s.TotalTokens += int(tokenInt)
							}
						}
					}
				}
			}
		}
	}
}

func (s *RunStats) updateTimes(span map[string]interface{}) {
	// Extract start and end times from span
	// Format: "2026-01-19T18:36:38.897+09:00"
	if st, ok := span["StartTime"].(string); ok {
		// Try to parse with timezone
		if t, err := time.Parse(time.RFC3339, st); err == nil {
			if s.FirstSpan.IsZero() || t.Before(s.FirstSpan) {
				s.FirstSpan = t
			}
			if t.After(s.LastSpan) {
				s.LastSpan = t
			}
		}
	}

	// Also check EndTime to get the latest time
	if et, ok := span["EndTime"].(string); ok {
		if t, err := time.Parse(time.RFC3339, et); err == nil {
			if t.After(s.LastSpan) {
				s.LastSpan = t
			}
		}
	}
}

// toInt64 safely converts a value to int64
func toInt64(v interface{}) (int64, error) {
	switch val := v.(type) {
	case float64:
		return int64(val), nil
	case int:
		return int64(val), nil
	case int64:
		return val, nil
	case string:
		i, err := strconv.ParseInt(val, 10, 64)
		return i, err
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", v)
	}
}

func getLatestRunID() string {
	entries, err := os.ReadDir(runsDirName)
	if err != nil {
		return ""
	}

	var latest os.FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "run-") {
			if latest == nil || info.ModTime().After(latest.ModTime()) {
				latest = info
			}
		}
	}

	if latest != nil {
		return latest.Name()
	}
	return ""
}

type Span struct {
	Name                 string                   `json:"Name"`
	StartTime            string                   `json:"StartTime"`
	EndTime              string                   `json:"EndTime"`
	Attributes           []map[string]interface{} `json:"Attributes,omitempty"`
	ParentSpanID         string                   `json:"ParentSpanId,omitempty"`
	SpanID               string                   `json:"SpanId"`
	SpanKind             int                      `json:"SpanKind"`
	Status               map[string]interface{}   `json:"Status"`
	ChildSpanCount       int                      `json:"ChildSpanCount"`
	InstrumentationScope map[string]interface{}   `json:"InstrumentationScope"`
}

// auditTrace analyzes a trace and outputs a TraceObject for evaluation
func auditTrace(runID string) error {
	runsDir := runsDirName

	// If no run ID provided, use latest
	if runID == "" {
		runID = getLatestRunID()
		if runID == "" {
			fmt.Println("No traces found. Run with AGK_TRACE=true to generate traces.")
			return nil
		}
	}

	runPath := filepath.Join(runsDir, runID)

	// Check if run exists
	if _, err := os.Stat(runPath); os.IsNotExist(err) {
		return fmt.Errorf("trace not found: %s", runID)
	}

	// Use the audit package to collect events
	collector, err := audit.NewCollector(runPath)
	if err != nil {
		return fmt.Errorf("failed to create collector: %w", err)
	}

	traceObj, err := collector.Collect()
	if err != nil {
		return fmt.Errorf("failed to collect trace: %w", err)
	}

	// Output as JSON
	output, err := json.MarshalIndent(traceObj, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal trace object: %w", err)
	}

	fmt.Println(string(output))
	return nil
}

// generateMermaid creates a Mermaid flowchart from trace data
func generateMermaid(runID, output string) error {
	runsDir := runsDirName

	// If no run ID provided, use latest
	if runID == "" {
		runID = getLatestRunID()
		if runID == "" {
			fmt.Println("No traces found. Run with AGK_TRACE=true to generate traces.")
			return nil
		}
	}

	runPath := filepath.Join(runsDir, runID)

	// Check if run exists
	if _, err := os.Stat(runPath); os.IsNotExist(err) {
		return fmt.Errorf("trace not found: %s", runID)
	}

	// Use the audit package to collect events
	collector, err := audit.NewCollector(runPath)
	if err != nil {
		return fmt.Errorf("failed to create collector: %w", err)
	}

	traceObj, err := collector.Collect()
	if err != nil {
		return fmt.Errorf("failed to collect trace: %w", err)
	}

	// Generate Mermaid diagram
	mermaid := audit.GenerateMermaidWithHierarchy(traceObj)

	// Build output content
	var content strings.Builder
	content.WriteString(fmt.Sprintf("# Agent Trace: %s\n\n", runID))
	content.WriteString(fmt.Sprintf("**Events:** %d | **Duration:** %dms\n\n",
		traceObj.Summary.TotalEvents, traceObj.Summary.TotalDurationMs))
	content.WriteString("## Execution Flow\n\n")
	content.WriteString(mermaid)

	// Write to file or stdout
	if output != "" {
		if err := os.WriteFile(output, []byte(content.String()), 0600); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("✅ Generated Mermaid diagram: %s\n", output)
	} else {
		fmt.Println(content.String())
	}

	return nil
}
