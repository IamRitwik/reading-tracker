package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/IamRitwik/reading-tracker/internal/domain"
	"github.com/IamRitwik/reading-tracker/internal/repository"
)

// Compile-time assertion that *BookRepo satisfies repository.BookRepository.
// If you ever rename a method or change a signature, this line fails to build.
// This is idiomatic Go — cheap, zero runtime cost.
var _ repository.BookRepository = (*BookRepo)(nil)

type BookRepo struct {
	db *sql.DB
}

func NewBookRepo(db *sql.DB) *BookRepo {
	return &BookRepo{db: db}
}

const bookColumns = `id, title, author, category, description,
	total_pages, pages_read, status, created_at, updated_at`

func (r *BookRepo) Create(ctx context.Context, b *domain.Book) error {
	now := time.Now().UTC()
	b.CreatedAt = now
	b.UpdatedAt = now

	res, err := r.db.ExecContext(ctx, `
		INSERT INTO books
			(title, author, category, description, total_pages, pages_read, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		b.Title, b.Author, b.Category, b.Description,
		b.TotalPages, b.PagesRead, string(b.Status),
		toDBTime(b.CreatedAt), toDBTime(b.UpdatedAt),
	)

	if err != nil {
		return fmt.Errorf("insert book: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("last insert id: %w", err)
	}
	b.ID = id
	return nil
}

func (r *BookRepo) GetByID(ctx context.Context, id int64) (*domain.Book, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+bookColumns+` FROM books WHERE id = ?`, id,
	)
	b, err := scanBook(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrBookNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get book %d: %w", id, err)
	}
	return b, nil
}

func (r *BookRepo) List(ctx context.Context, f repository.BookFilter) ([]*domain.Book, error) {
	// Build the WHERE clause dynamically. We use ? placeholders exclusively —
	// NEVER string-concatenate user input. Placeholders protect against SQL
	// injection at the driver level.
	var (
		where []string
		args  []any
	)
	if f.Status != nil {
		where = append(where, "status = ?")
		args = append(args, string(*f.Status))
	}
	if f.Category != nil {
		where = append(where, "category = ?")
		args = append(args, *f.Category)
	}
	if f.Author != nil {
		where = append(where, "author = ?")
		args = append(args, *f.Author)
	}
	if f.Search != nil && *f.Search != "" {
		where = append(where,
			"(title LIKE ? OR author LIKE ? OR description LIKE ? OR category LIKE ?)")
		like := "%" + *f.Search + "%"
		args = append(args, like, like, like, like)
	}

	query := "SELECT " + bookColumns + " FROM books"
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY updated_at DESC"

	if f.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, f.Limit)
		if f.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, f.Offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query books: %w", err)
	}
	// defer Close is mandatory. Leaked rows = leaked connections = deadlock
	// the moment the pool is exhausted. With MaxOpenConns(1), one leak = hang.
	defer rows.Close()

	var books []*domain.Book
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, fmt.Errorf("scan book: %w", err)
		}
		books = append(books, b)
	}
	// ALWAYS check rows.Err() after the loop. It catches mid-stream errors
	// (network drop, context cancel) that Next() swallows.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate books: %w", err)
	}
	return books, nil
}

func (r *BookRepo) Update(ctx context.Context, b *domain.Book) error {
	b.UpdatedAt = time.Now().UTC()

	res, err := r.db.ExecContext(ctx, `
		UPDATE books
		SET title = ?, author = ?, category = ?, description = ?,
		    total_pages = ?, pages_read = ?, status = ?, updated_at = ?
		WHERE id = ?
	`,
		b.Title, b.Author, b.Category, b.Description,
		b.TotalPages, b.PagesRead, string(b.Status),
		toDBTime(b.UpdatedAt), b.ID,
	)
	if err != nil {
		return fmt.Errorf("update book %d: %w", b.ID, err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrBookNotFound
	}
	return nil
}

func (r *BookRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM books WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete book %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrBookNotFound
	}
	return nil
}

// scanBook works with both *sql.Row and *sql.Rows because both implement
// the same Scan(...any) error method. One implementation, two callers.
func scanBook(scanner interface{ Scan(...any) error }) (*domain.Book, error) {
	var (
		b                            domain.Book
		status, createdAt, updatedAt string
	)
	if err := scanner.Scan(
		&b.ID, &b.Title, &b.Author, &b.Category, &b.Description,
		&b.TotalPages, &b.PagesRead, &status, &createdAt, &updatedAt,
	); err != nil {
		return nil, err
	}
	b.Status = domain.Status(status)

	var err error
	if b.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
		return nil, fmt.Errorf("parse created_at %q: %w", createdAt, err)
	}
	if b.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt); err != nil {
		return nil, fmt.Errorf("parse updated_at %q: %w", updatedAt, err)
	}
	return &b, nil
}

func toDBTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}
