package observability_test

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/IamRitwik/reading-tracker/internal/domain"
	"github.com/IamRitwik/reading-tracker/internal/observability"
	"github.com/IamRitwik/reading-tracker/internal/repository"
	"github.com/IamRitwik/reading-tracker/internal/repository/memory"
)

func newInstrumented(t *testing.T) (*observability.InstrumentedBookRepo, *observability.Metrics) {
	t.Helper()
	m := observability.NewMetrics()
	inner := memory.NewBookRepo()
	return observability.NewInstrumentedBookRepo(inner, m, nil), m
}

func TestInstrumentedRepo_PassesThrough(t *testing.T) {
	repo, _ := newInstrumented(t)
	ctx := context.Background()

	// Behavior must be identical to the inner repo — no surprises added.
	b := &domain.Book{Title: "T", Author: "A", TotalPages: 10}
	if err := repo.Create(ctx, b); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByID(ctx, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "T" {
		t.Errorf("Title: got %q, want %q", got.Title, "T")
	}
}

func TestInstrumentedRepo_CountsCreates(t *testing.T) {
	repo, m := newInstrumented(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if err := repo.Create(ctx, &domain.Book{Title: "T", Author: "A", TotalPages: 1}); err != nil {
			t.Fatal(err)
		}
	}

	if got := testutil.ToFloat64(m.BooksAdded); got != 3 {
		t.Errorf("BooksAdded: got %v, want 3", got)
	}
}

func TestInstrumentedRepo_CountsUpdatesAndDeletes(t *testing.T) {
	repo, m := newInstrumented(t)
	ctx := context.Background()

	b := &domain.Book{Title: "T", Author: "A", TotalPages: 10}
	if err := repo.Create(ctx, b); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(ctx, b); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, b.ID); err != nil {
		t.Fatal(err)
	}

	if got := testutil.ToFloat64(m.BooksUpdated); got != 1 {
		t.Errorf("BooksUpdated: got %v, want 1", got)
	}
	if got := testutil.ToFloat64(m.BooksDeleted); got != 1 {
		t.Errorf("BooksDeleted: got %v, want 1", got)
	}
}

// failingRepo returns a preset error from every method. Used to verify that
// error metrics are recorded without needing a real DB failure.
type failingRepo struct{ err error }

func (f *failingRepo) Create(context.Context, *domain.Book) error { return f.err }
func (f *failingRepo) GetByID(context.Context, int64) (*domain.Book, error) {
	return nil, f.err
}
func (f *failingRepo) List(context.Context, repository.BookFilter) ([]*domain.Book, error) {
	return nil, f.err
}
func (f *failingRepo) Update(context.Context, *domain.Book) error { return f.err }
func (f *failingRepo) Delete(context.Context, int64) error        { return f.err }

func TestInstrumentedRepo_CountsErrors(t *testing.T) {
	m := observability.NewMetrics()
	sentinel := errors.New("boom")
	repo := observability.NewInstrumentedBookRepo(&failingRepo{err: sentinel}, m, nil)

	_ = repo.Create(context.Background(), &domain.Book{})

	if got := testutil.ToFloat64(m.DBQueryErrors.WithLabelValues("create")); got != 1 {
		t.Errorf("DBQueryErrors{create}: got %v, want 1", got)
	}
	// Successful-only counter must NOT have incremented.
	if got := testutil.ToFloat64(m.BooksAdded); got != 0 {
		t.Errorf("BooksAdded after failure: got %v, want 0", got)
	}
}
