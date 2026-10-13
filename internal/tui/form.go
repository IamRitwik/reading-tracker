package tui

import (
	"strconv"
	"strings"

	"github.com/IamRitwik/reading-tracker/internal/domain"
	"github.com/IamRitwik/reading-tracker/internal/service"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type formMode int

const (
	formAdd formMode = iota
	formUpdateProgress
)

type formModel struct {
	mode    formMode
	labels  []string
	inputs  []textinput.Model
	focused int

	// context for update-progress mode
	bookID   int64
	maxPages int
}

// newAddForm creates a blank add form.
func newAddForm(width int) formModel {
	labels := []string{"Title", "Author", "Category", "Description", "Total Pages"}
	inputs := make([]textinput.Model, len(labels))
	for i := range inputs {
		ti := textinput.New()
		ti.CharLimit = 200
		ti.Width = max(20, width-16)
		ti.Prompt = ""
		inputs[i] = ti
	}
	inputs[0].Focus()
	return formModel{mode: formAdd, labels: labels, inputs: inputs}
}

// newUpdateProgressForm creates a form to update a book's pages read.
func newUpdateProgressForm(width int, b domain.Book) formModel {
	labels := []string{"Pages Read"}
	inputs := make([]textinput.Model, 1)
	ti := textinput.New()
	ti.CharLimit = 10
	ti.Width = max(10, width-16)
	ti.Prompt = ""
	ti.SetValue(strconv.Itoa(b.PagesRead))
	ti.CursorEnd()
	inputs[0] = ti
	inputs[0].Focus()
	return formModel{
		mode:     formUpdateProgress,
		labels:   labels,
		inputs:   inputs,
		bookID:   b.ID,
		maxPages: b.TotalPages,
	}
}

// Next moves focus forward (wraps around).
func (f formModel) Next() formModel {
	f.inputs[f.focused].Blur()
	f.focused = (f.focused + 1) % len(f.inputs)
	f.inputs[f.focused].Focus()
	return f
}

// Prev moves focus backward (wraps around).
func (f formModel) Prev() formModel {
	f.inputs[f.focused].Blur()
	f.focused = (f.focused - 1 + len(f.inputs)) % len(f.inputs)
	f.inputs[f.focused].Focus()
	return f
}

// IsLast reports whether the final field is focused.
func (f formModel) IsLast() bool { return f.focused == len(f.inputs)-1 }

// Update forwards a message to the focused input.
func (f formModel) Update(msg tea.Msg) (formModel, tea.Cmd) {
	var cmd tea.Cmd
	f.inputs[f.focused], cmd = f.inputs[f.focused].Update(msg)
	return f, cmd
}

// Values returns the trimmed values of all inputs.
func (f formModel) Values() []string {
	out := make([]string, len(f.inputs))
	for i, in := range f.inputs {
		out[i] = strings.TrimSpace(in.Value())
	}
	return out
}

// ToAddInput converts form values into a service.AddBookInput.
// Returns an error string suitable for display if parsing fails.
func (f formModel) ToAddInput() (service.AddBookInput, string) {
	v := f.Values()
	pages, err := strconv.Atoi(v[4])
	if err != nil {
		return service.AddBookInput{}, "Total Pages must be a number"
	}
	return service.AddBookInput{
		Title:       v[0],
		Author:      v[1],
		Category:    v[2],
		Description: v[3],
		TotalPages:  pages,
	}, ""
}

// ToPagesRead parses the update-progress form value.
func (f formModel) ToPagesRead() (int, string) {
	v := f.Values()[0]
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, "Pages Read must be a number"
	}
	if n < 0 || n > f.maxPages {
		return 0, "Pages Read must be between 0 and " + strconv.Itoa(f.maxPages)
	}
	return n, ""
}
