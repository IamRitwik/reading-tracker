package memory

// Package memory provides an in-memory BookRepository.
//
// It is used in service-layer tests where we want to exercise business logic
// without touching SQLite. It is intentionally minimal: it does not emulate
// transactions, WAL, or concurrency beyond a single mutex.

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/IamRitwik/reading-tracker/internal/domain"
	"github.com/IamRitwik/reading-tracker/internal/repository"
)

var _ repository.BookRepository = (*BookRepo)(nil)

type BookRepo struct {
	mu     sync.RWMutex
	nextID int64
	books  map[int64]*domain.Book
}

func NewBookRepo() *BookRepo {
	return &BookRepo{
		nextID: 1,
		books:  make(map[int64]*domain.Book),
	}
}

func (r *BookRepo) Create(ctx context.Context, b *domain.Book) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	b.ID = r.nextID
	r.nextID++
	b.CreatedAt = now
	b.UpdatedAt = now

	// Copy so the caller can't mutate our internal state.
	cp := *b
	r.books[b.ID] = &cp
	return nil
}

func (r *BookRepo) GetByID(ctx context.Context, id int64) (*domain.Book, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	b, ok := r.books[id]
	if !ok {
		return nil, domain.ErrBookNotFound
	}
	cp := *b
	return &cp, nil
}

func (r *BookRepo) List(ctx context.Context, f repository.BookFilter) ([]*domain.Book, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []*domain.Book
	for _, b := range r.books {
		if f.Status != nil && b.Status != *f.Status {
			continue
		}
		if f.Category != nil && b.Category != *f.Category {
			continue
		}
		if f.Author != nil && b.Author != *f.Author {
			continue
		}
		if f.Search != nil && *f.Search != "" {
			q := strings.ToLower(*f.Search)
			hit := strings.Contains(strings.ToLower(b.Title), q) ||
				strings.Contains(strings.ToLower(b.Author), q) ||
				strings.Contains(strings.ToLower(b.Description), q) ||
				strings.Contains(strings.ToLower(b.Category), q)
			if !hit {
				continue
			}
		}
		cp := *b
		out = append(out, &cp)
	}

	// Match the SQL implementation: newest updated first.
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})

	if f.Offset > 0 && f.Offset < len(out) {
		out = out[f.Offset:]
	} else if f.Offset >= len(out) {
		out = nil
	}
	if f.Limit > 0 && f.Limit < len(out) {
		out = out[:f.Limit]
	}
	return out, nil
}

func (r *BookRepo) Update(ctx context.Context, b *domain.Book) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.books[b.ID]; !ok {
		return domain.ErrBookNotFound
	}
	b.UpdatedAt = time.Now().UTC()
	cp := *b
	r.books[b.ID] = &cp
	return nil
}

func (r *BookRepo) Delete(ctx context.Context, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.books[id]; !ok {
		return domain.ErrBookNotFound
	}
	delete(r.books, id)
	return nil
}
