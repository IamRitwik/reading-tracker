package tui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/IamRitwik/reading-tracker/internal/domain"
	"github.com/IamRitwik/reading-tracker/internal/service"
)

type viewState int

const (
	viewList viewState = iota
	viewDetail
	viewAddForm
	viewUpdateForm
	viewConfirmDelete
	viewStats
)

type Model struct {
	ctx    context.Context
	cancel context.CancelFunc
	svc    *service.BookService
	logger *slog.Logger

	width, height int

	view   viewState
	books  []*domain.Book
	cursor int
	filter service.ListInput

	selected *domain.Book // for detail / update / delete-confirm

	form formModel

	// transient status line
	status      string
	statusIsErr bool
}

func newModel(ctx context.Context, cancel context.CancelFunc, svc *service.BookService, logger *slog.Logger) Model {
	return Model{
		ctx:    ctx,
		cancel: cancel,
		svc:    svc,
		logger: logger,
		view:   viewList,
	}
}

// Init is called once by Bubble Tea after Run starts. We kick off the
// initial data load and give Bubble Tea some initial dimensions.
func (m Model) Init() tea.Cmd {
	return m.loadBooksCmd()
}

// --- Update ------------------------------------------------------------

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case booksLoadedMsg:
		if msg.err != nil {
			m.status = "load failed: " + msg.err.Error()
			m.statusIsErr = true
			return m, nil
		}
		m.books = msg.books
		if m.cursor >= len(m.books) && len(m.books) > 0 {
			m.cursor = len(m.books) - 1
		}
		if len(m.books) == 0 {
			m.cursor = 0
		}
		return m, nil

	case bookAddedMsg:
		if msg.err != nil {
			m.status = "add failed: " + msg.err.Error()
			m.statusIsErr = true
			return m, nil
		}
		m.status = "added: " + msg.book.Title
		m.statusIsErr = false
		m.view = viewList
		return m, m.loadBooksCmd()

	case bookUpdatedMsg:
		if msg.err != nil {
			m.status = "update failed: " + msg.err.Error()
			m.statusIsErr = true
			return m, nil
		}
		m.status = fmt.Sprintf("updated %q → %d/%d",
			msg.book.Title, msg.book.PagesRead, msg.book.TotalPages)
		m.statusIsErr = false
		m.view = viewList
		m.selected = nil
		return m, m.loadBooksCmd()

	case bookDeletedMsg:
		if msg.err != nil {
			m.status = "delete failed: " + msg.err.Error()
			m.statusIsErr = true
			return m, nil
		}
		m.status = "book deleted"
		m.statusIsErr = false
		m.view = viewList
		m.selected = nil
		return m, m.loadBooksCmd()
	}

	return m, nil
}

// handleKey routes key events based on the current view.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Forms have their own key semantics; forward into the form.
	if m.view == viewAddForm || m.view == viewUpdateForm {
		return m.handleFormKey(msg)
	}

	switch m.view {
	case viewList:
		return m.handleListKey(msg)
	case viewDetail:
		return m.handleDetailKey(msg)
	case viewConfirmDelete:
		return m.handleConfirmDeleteKey(msg)
	case viewStats:
		return m.handleStatsKey(msg)
	}
	return m, nil
}

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.cancel()
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.books)-1 {
			m.cursor++
		}

	case "g":
		m.cursor = 0
	case "G":
		if len(m.books) > 0 {
			m.cursor = len(m.books) - 1
		}

	case "r":
		m.status = "refreshing…"
		m.statusIsErr = false
		return m, m.loadBooksCmd()

	case "a":
		m.form = newAddForm(m.width)
		m.view = viewAddForm
		m.status = ""

	case "enter":
		if len(m.books) > 0 {
			m.selected = m.books[m.cursor]
			m.view = viewDetail
		}

	case "u":
		if len(m.books) > 0 {
			b := *m.books[m.cursor]
			m.selected = &b
			m.form = newUpdateProgressForm(m.width, b)
			m.view = viewUpdateForm
			m.status = ""
		}

	case "d":
		if len(m.books) > 0 {
			b := *m.books[m.cursor]
			m.selected = &b
			m.view = viewConfirmDelete
		}

	case "s":
		m.view = viewStats
	}
	return m, nil
}

func (m Model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.cancel()
		return m, tea.Quit
	case "esc", "backspace":
		m.view = viewList
		m.selected = nil
	case "u":
		if m.selected != nil {
			m.form = newUpdateProgressForm(m.width, *m.selected)
			m.view = viewUpdateForm
		}
	case "d":
		if m.selected != nil {
			m.view = viewConfirmDelete
		}
	}
	return m, nil
}

func (m Model) handleConfirmDeleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.selected != nil {
			id := m.selected.ID
			m.view = viewList
			return m, m.deleteBookCmd(id)
		}
	case "n", "N", "esc":
		m.view = viewList
		m.selected = nil
	}
	return m, nil
}

func (m Model) handleStatsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.cancel()
		return m, tea.Quit
	case "esc", "backspace", "s":
		m.view = viewList
	}
	return m, nil
}

// handleFormKey handles keys while a form is active.
func (m Model) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.cancel()
		return m, tea.Quit
	case "esc":
		m.view = viewList
		m.form = formModel{}
		m.status = ""
		return m, nil
	case "tab", "down":
		m.form = m.form.Next()
		return m, nil
	case "shift+tab", "up":
		m.form = m.form.Prev()
		return m, nil
	case "enter":
		if m.form.IsLast() {
			return m.submitForm()
		}
		m.form = m.form.Next()
		return m, nil
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m Model) submitForm() (tea.Model, tea.Cmd) {
	switch m.form.mode {
	case formAdd:
		in, errMsg := m.form.ToAddInput()
		if errMsg != "" {
			m.status = errMsg
			m.statusIsErr = true
			return m, nil
		}
		return m, m.addBookCmd(in)

	case formUpdateProgress:
		pages, errMsg := m.form.ToPagesRead()
		if errMsg != "" {
			m.status = errMsg
			m.statusIsErr = true
			return m, nil
		}
		return m, m.updateProgressCmd(m.form.bookID, pages)
	}
	return m, nil
}

// --- View --------------------------------------------------------------

func (m Model) View() string {
	switch m.view {
	case viewDetail:
		return m.viewDetail()
	case viewAddForm:
		return m.viewForm("Add Book")
	case viewUpdateForm:
		return m.viewForm("Update Progress")
	case viewConfirmDelete:
		return m.viewConfirmDelete()
	case viewStats:
		return m.viewStats()
	default:
		return m.viewList()
	}
}

// --- Renderers ---------------------------------------------------------

func (m Model) viewList() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(" 📚 Reading Tracker "))
	if m.status != "" {
		b.WriteString("  ")
		if m.statusIsErr {
			b.WriteString(errStyle.Render(m.status))
		} else {
			b.WriteString(okStyle.Render(m.status))
		}
	}
	b.WriteString("\n\n")

	if len(m.books) == 0 {
		b.WriteString(dimStyle.Render("No books yet. Press 'a' to add one."))
		b.WriteString("\n")
	} else {
		for i, bk := range m.books {
			cursor := "  "
			if i == m.cursor {
				cursor = cursorStyle.Render("▸ ")
			}

			title := bk.Title
			if i == m.cursor {
				title = selectedTitleStyle.Render(title)
			} else {
				title = valueStyle.Render(title)
			}

			pct := bk.ProgressPercent()
			bar := renderBar(pct, 20)
			status := statusStyle(bk.Status).Render(statusBadge(bk.Status))

			fmt.Fprintf(&b, "%s%s\n", cursor, title)
			fmt.Fprintf(&b, "    %s  %s  %3.0f%%  %s\n",
				dimStyle.Render(bk.Author),
				bar,
				pct,
				status,
			)
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render(
		"↑/k ↓/j move · enter detail · a add · u update · d delete · s stats · r refresh · q quit"))
	return b.String()
}

func (m Model) viewDetail() string {
	if m.selected == nil {
		return dimStyle.Render("no book selected")
	}
	b := *m.selected

	var sb strings.Builder
	sb.WriteString(titleStyle.Render(b.Title))
	sb.WriteString("\n\n")

	row := func(k, v string) {
		fmt.Fprintf(&sb, "%s %s\n", labelStyle.Render(fmt.Sprintf("%-12s", k)), valueStyle.Render(v))
	}

	row("Author", b.Author)
	if b.Category != "" {
		row("Category", b.Category)
	}
	row("Pages", fmt.Sprintf("%d / %d", b.PagesRead, b.TotalPages))
	row("Status", statusStyle(b.Status).Render(string(b.Status)))
	row("Progress", fmt.Sprintf("%s %.1f%%", renderBar(b.ProgressPercent(), 30), b.ProgressPercent()))
	if b.Description != "" {
		row("Description", b.Description)
	}
	row("Added", b.CreatedAt.Local().Format("2006-01-02 15:04"))
	row("Updated", b.UpdatedAt.Local().Format("2006-01-02 15:04"))

	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("esc back · u update progress · d delete · q quit"))
	return sb.String()
}

func (m Model) viewForm(heading string) string {
	var sb strings.Builder
	sb.WriteString(titleStyle.Render(" " + heading + " "))
	sb.WriteString("\n\n")

	for i, label := range m.form.labels {
		marker := "  "
		if i == m.form.focused {
			marker = cursorStyle.Render("▸ ")
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", marker, labelStyle.Render(label)))
		sb.WriteString(fmt.Sprintf("  %s\n\n", m.form.inputs[i].View()))
	}

	if m.status != "" {
		if m.statusIsErr {
			sb.WriteString(errStyle.Render(m.status))
		} else {
			sb.WriteString(okStyle.Render(m.status))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("tab/↓ next · shift+tab/↑ prev · enter submit · esc cancel"))
	return sb.String()
}

func (m Model) viewConfirmDelete() string {
	if m.selected == nil {
		return dimStyle.Render("nothing selected")
	}
	var sb strings.Builder
	sb.WriteString(titleStyle.Render(" Delete Book "))
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("Delete %s by %s?\n\n",
		selectedTitleStyle.Render(m.selected.Title),
		dimStyle.Render(m.selected.Author)))
	sb.WriteString(helpStyle.Render("y confirm · n cancel"))
	return sb.String()
}

func (m Model) viewStats() string {
	total := len(m.books)
	byStatus := map[domain.Status]int{}
	var pagesRead, pagesTotal int
	for _, bk := range m.books {
		byStatus[bk.Status]++
		pagesRead += bk.PagesRead
		pagesTotal += bk.TotalPages
	}

	var sb strings.Builder
	sb.WriteString(titleStyle.Render(" Statistics "))
	sb.WriteString("\n\n")

	row := func(k, v string) {
		fmt.Fprintf(&sb, "%s %s\n", labelStyle.Render(fmt.Sprintf("%-16s", k)), valueStyle.Render(v))
	}

	row("Total books", fmt.Sprintf("%d", total))
	row("Want to read", fmt.Sprintf("%d", byStatus[domain.StatusWantToRead]))
	row("Reading", fmt.Sprintf("%d", byStatus[domain.StatusReading]))
	row("Completed", fmt.Sprintf("%d", byStatus[domain.StatusCompleted]))
	row("Abandoned", fmt.Sprintf("%d", byStatus[domain.StatusAbandoned]))
	row("Pages read", fmt.Sprintf("%d", pagesRead))
	row("Pages total", fmt.Sprintf("%d", pagesTotal))
	if pagesTotal > 0 {
		row("Overall", fmt.Sprintf("%.1f%%", float64(pagesRead)/float64(pagesTotal)*100))
	}

	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("esc back · q quit"))
	return sb.String()
}
