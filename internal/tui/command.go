package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/IamRitwik/reading-tracker/internal/domain"
	"github.com/IamRitwik/reading-tracker/internal/service"
)

// --- Messages ----------------------------------------------------------

type booksLoadedMsg struct {
	books []*domain.Book
	err   error
}

type bookAddedMsg struct {
	book *domain.Book
	err  error
}

type bookUpdatedMsg struct {
	book *domain.Book
	err  error
}

type bookDeletedMsg struct {
	id  int64
	err error
}

// --- Command constructors ---------------------------------------------
//
// Each returns a tea.Cmd: a func() tea.Msg that Bubble Tea runs in a
// goroutine. State never mutates directly inside a Cmd — it returns a Msg,
// and Update handles it on the main loop. This is what makes Bubble Tea
// race-free without locks.
//
// Every command derives its context from the model's base context so that
// quitting the TUI cancels in-flight work.

const opTimeout = 5 * time.Second

func (m Model) loadBooksCmd() tea.Cmd {
	ctx, svc, filter := m.ctx, m.svc, m.filter
	return func() tea.Msg {
		opCtx, cancel := context.WithTimeout(ctx, opTimeout)
		defer cancel()
		books, err := svc.List(opCtx, filter)
		return booksLoadedMsg{books: books, err: err}
	}
}

func (m Model) addBookCmd(in service.AddBookInput) tea.Cmd {
	ctx, svc := m.ctx, m.svc
	return func() tea.Msg {
		opCtx, cancel := context.WithTimeout(ctx, opTimeout)
		defer cancel()
		b, err := svc.Add(opCtx, in)
		return bookAddedMsg{book: b, err: err}
	}
}

func (m Model) updateProgressCmd(id int64, pages int) tea.Cmd {
	ctx, svc := m.ctx, m.svc
	return func() tea.Msg {
		opCtx, cancel := context.WithTimeout(ctx, opTimeout)
		defer cancel()
		b, err := svc.UpdateProgress(opCtx, id, pages)
		return bookUpdatedMsg{book: b, err: err}
	}
}

func (m Model) deleteBookCmd(id int64) tea.Cmd {
	ctx, svc := m.ctx, m.svc
	return func() tea.Msg {
		opCtx, cancel := context.WithTimeout(ctx, opTimeout)
		defer cancel()
		err := svc.Delete(opCtx, id)
		return bookDeletedMsg{id: id, err: err}
	}
}
