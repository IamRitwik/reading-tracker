package domain

import "time"

// Status represents a book's reading state.
// Using a named string type gives us compile-time distinction
// between a Status and, say, a title.
type Status string

const (
	StatusWantToRead Status = "want_to_read"
	StatusReading    Status = "reading"
	StatusCompleted  Status = "completed"
	StatusAbandoned  Status = "abandoned"
)

// Valid reports whether s is a recognized status.
// We validate at every boundary (service, DB CHECK constraint).
func (s Status) Valid() bool {
	switch s {
	case StatusWantToRead, StatusReading, StatusCompleted, StatusAbandoned:
		return true
	}
	return false
}

// Book - core domain entity
type Book struct {
	ID          int64
	Title       string
	Author      string
	Category    string
	Description string
	TotalPages  int
	PagesRead   int
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ProgressPercent returns progress from 0-100
func (b Book) ProgressPercent() float64 {
	if b.TotalPages <= 0 {
		return 0
	}
	return float64(b.PagesRead) / float64(b.TotalPages) * 100
}
