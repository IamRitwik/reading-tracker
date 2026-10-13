package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/IamRitwik/reading-tracker/internal/domain"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	selectedTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	errStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	okStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
)

// statusStyle returns a colored style for a book status.
func statusStyle(s domain.Status) lipgloss.Style {
	switch s {
	case domain.StatusReading:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true)
	case domain.StatusCompleted:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	case domain.StatusAbandoned:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	default: // want_to_read
		return lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	}
}

// statusBadge returns a short human label for a status.
func statusBadge(s domain.Status) string {
	switch s {
	case domain.StatusWantToRead:
		return "WANT"
	case domain.StatusReading:
		return "READING"
	case domain.StatusCompleted:
		return "DONE"
	case domain.StatusAbandoned:
		return "DROP"
	}
	return strings.ToUpper(string(s))
}

// renderBar draws a compact one-line progress bar of the given width.
func renderBar(pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int(float64(width) * pct / 100.0)
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}
