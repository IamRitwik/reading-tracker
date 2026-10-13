package tui

import (
	"context"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/IamRitwik/reading-tracker/internal/service"
)

// Run launches the TUI. It blocks until the user quits (q / Ctrl+C) or the
// parent context is cancelled (e.g. SIGTERM handled in main).
//
// The parent ctx is what signal.NotifyContext gives us. We derive a child
// so we can cancel in-flight DB commands independently when the user quits
// without cancelling the whole process context.
func Run(parent context.Context, svc *service.BookService, logger *slog.Logger) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	m := newModel(ctx, cancel, svc, logger)

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),  // full-screen mode; restores the terminal on exit
		tea.WithContext(ctx), // when ctx is cancelled, the program exits
	)
	_, err := p.Run()
	return err
}
