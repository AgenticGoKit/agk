// Package tui provides interactive terminal UI components for agk CLI.
package tui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	primaryColor   = lipgloss.Color("#7C3AED") // Purple
	secondaryColor = lipgloss.Color("#06B6D4") // Cyan
	successColor   = lipgloss.Color("#10B981") // Green
	errorColor     = lipgloss.Color("#EF4444") // Red
	warningColor   = lipgloss.Color("#F59E0B") // Amber
	mutedColor     = lipgloss.Color("#6B7280") // Gray
	accentColor    = lipgloss.Color("#F472B6") // Pink
)

// Box styles
var (
	// BoxStyle is the main container style
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1)

	// HeaderStyle for headers
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Padding(0, 1)

	// TitleStyle for main titles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(primaryColor).
			Padding(0, 2)

	// SectionHeaderStyle for detail view sections
	SectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(secondaryColor).
				Padding(0, 1).
				Margin(1, 0, 0, 0)
)

// Text styles
var (
	// SelectedStyle for selected items
	SelectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(secondaryColor)

	// CursorStyle for the cursor indicator
	CursorStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	// MutedStyle for less important text
	MutedStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// SuccessStyle for success indicators
	SuccessStyle = lipgloss.NewStyle().
			Foreground(successColor)

	// ErrorStyle for error indicators
	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	// WarningStyle for warnings
	WarningStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	// DurationStyle for duration values
	DurationStyle = lipgloss.NewStyle().
			Foreground(accentColor)

	// AttributeKeyStyle for attribute keys
	AttributeKeyStyle = lipgloss.NewStyle().
				Foreground(secondaryColor)

	// AttributeValueStyle for attribute values
	AttributeValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF"))
)

// Span type styles
var (
	WorkflowSpanStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8B5CF6")) // Violet

	AgentSpanStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3B82F6")) // Blue

	LLMSpanStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")) // Emerald

	ToolSpanStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")) // Amber
)

// Help bar style
var (
	HelpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Padding(0, 1)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)
)

// GetSpanStyle returns the appropriate style based on span name
func GetSpanStyle(spanName string) lipgloss.Style {
	switch {
	case contains(spanName, "workflow"):
		return WorkflowSpanStyle
	case contains(spanName, "agent"):
		return AgentSpanStyle
	case contains(spanName, "llm"):
		return LLMSpanStyle
	case contains(spanName, "tool"), contains(spanName, "mcp"):
		return ToolSpanStyle
	default:
		return lipgloss.NewStyle()
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
