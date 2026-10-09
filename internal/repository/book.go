package repository

import (
	"context"

	"github.com/IamRitwik/reading-tracker/internal/domain"
)

// BookRepository is the persistence contract. Note:
//   - Every method takes context.Context as its first arg (Go convention).
//   - It returns domain types, never sql.Row or *sql.DB.
//   - It's a plain interface with no imports of database/sql.
//
// This means the service layer never knows whether we're using SQLite,
// Postgres, or an in-memory fake.
type BookRepository interface {
	Create(ctx context.Context, b *domain.Book) error
	GetByID(ctx context.Context, id int64) (*domain.Book, error)
	List(ctx context.Context, f BookFilter) ([]*domain.Book, error)
	Update(cyx context.Context, b *domain.Book) error
	Delete(ctx context.Context, id int64) error
}

// BookFilter controls List results. All fields optional.
// Pointer fields distinguish "not set" from "zero value."
type BookFilter struct {
	Status   *domain.Status
	Category *string
	Author   *string
	Search   *string // substring across title, author, description
	Limit    int     // 0 = no limit
	Offset   int
}
