package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/IamRitwik/reading-tracker/internal/domain"
)

// renderBooksTable writes an aligned table using text/tabwriter.
// tabwriter handles column widths automatically — no external dependency.
func renderBooksTable(w io.Writer, books []*domain.Book) {
	if len(books) == 0 {
		fmt.Fprintln(w, "(no books)")
		return
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tTITLE\tAUTHOR\tCATEGORY\tPROGRESS\tSTATUS")
	for _, b := range books {
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\n",
			b.ID,
			truncate(b.Title, 40),
			truncate(b.Author, 25),
			truncate(b.Category, 15),
			fmt.Sprintf("%d/%d (%.0f%%)", b.PagesRead, b.TotalPages, b.ProgressPercent()),
			string(b.Status),
		)
	}
	_ = tw.Flush()
}

func renderBook(w io.Writer, b *domain.Book) {
	row := func(k string, v any) {
		fmt.Fprintf(w, "%-12s %v\n", k+":", v)
	}
	row("ID", b.ID)
	row("Title", b.Title)
	row("Author", b.Author)
	if b.Category != "" {
		row("Category", b.Category)
	}
	row("Pages", fmt.Sprintf("%d / %d (%.1f%%)", b.PagesRead, b.TotalPages, b.ProgressPercent()))
	row("Status", string(b.Status))
	if b.Description != "" {
		row("Description", b.Description)
	}
	row("Added", b.CreatedAt.Local().Format("2006-01-02 15:04"))
	row("Updated", b.UpdatedAt.Local().Format("2006-01-02 15:04"))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}

// promptYes asks the user a yes/no question. Used by `delete` when --yes
// is absent. Reads a line from stdin.
func promptYes(prompt string) bool {
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", prompt)
	var line string
	_, _ = fmt.Fscanln(os.Stdin, &line)
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes"
}
