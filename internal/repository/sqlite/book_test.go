package sqlite_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/IamRitwik/reading-tracker/internal/db"
	"github.com/IamRitwik/reading-tracker/internal/domain"
	"github.com/IamRitwik/reading-tracker/internal/repository"
	"github.com/IamRitwik/reading-tracker/internal/repository/sqlite"
)

// newTestRepo returns a repo backed by a fresh SQLite file in t.TempDir().
// Why a file and not :memory:?
//  1. Exercises the same WAL / pragma path as production.
//  2. t.TempDir() auto-cleans, including -wal and -shm files.
//  3. Avoids :memory: + connection pool interactions that surprise people.
func newTestRepo(t *testing.T) *sqlite.BookRepo {
	t.Helper() // tells t.Fatalf to blame the caller, not this helper.
	ctx := context.Background()

	path := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(ctx, path)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	return sqlite.NewBookRepo(database)
}

func sampleBook() *domain.Book {
	return &domain.Book{
		Title:       "The Go Programming Language",
		Author:      "Donovan & Kernighan",
		Category:    "Programming",
		Description: "The definitive Go reference.",
		TotalPages:  380,
		Status:      domain.StatusWantToRead,
	}
}

func TestBookRepo_CreateAndGetByID(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	b := sampleBook()
	if err := repo.Create(ctx, b); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if b.ID == 0 {
		t.Fatal("Create did not set ID")
	}
	if b.CreatedAt.IsZero() {
		t.Fatal("Create did not set CreatedAt")
	}

	got, err := repo.GetByID(ctx, b.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Title != b.Title {
		t.Errorf("Title: got %q, want %q", got.Title, b.Title)
	}
	if got.Status != domain.StatusWantToRead {
		t.Errorf("Status: got %q, want %q", got.Status, domain.StatusWantToRead)
	}
	if !got.CreatedAt.Equal(b.CreatedAt) {
		t.Errorf("CreatedAt roundtrip: got %v, want %v", got.CreatedAt, b.CreatedAt)
	}
}

func TestBookRepo_GetByID_NotFound(t *testing.T) {
	repo := newTestRepo(t)

	_, err := repo.GetByID(context.Background(), 9999)
	if !errors.Is(err, domain.ErrBookNotFound) {
		t.Fatalf("got %v, want ErrBookNotFound", err)
	}
}

func TestBookRepo_List_Filters(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	seed := []*domain.Book{
		{Title: "A", Author: "X", Category: "Sci-Fi", TotalPages: 100, Status: domain.StatusReading},
		{Title: "B", Author: "Y", Category: "Sci-Fi", TotalPages: 200, Status: domain.StatusCompleted},
		{Title: "C", Author: "X", Category: "Biography", TotalPages: 300, Status: domain.StatusWantToRead},
	}
	for _, b := range seed {
		if err := repo.Create(ctx, b); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	t.Run("by status", func(t *testing.T) {
		s := domain.StatusCompleted
		got, err := repo.List(ctx, repository.BookFilter{Status: &s})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Title != "B" {
			t.Fatalf("got %d books, expected 1 (B)", len(got))
		}
	})

	t.Run("by author", func(t *testing.T) {
		a := "X"
		got, err := repo.List(ctx, repository.BookFilter{Author: &a})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d books, expected 2", len(got))
		}
	})

	t.Run("search", func(t *testing.T) {
		q := "Sci"
		got, err := repo.List(ctx, repository.BookFilter{Search: &q})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d books, expected 2 (category match)", len(got))
		}
	})
}

func TestBookRepo_Update(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	b := sampleBook()
	if err := repo.Create(ctx, b); err != nil {
		t.Fatal(err)
	}

	b.PagesRead = 200
	b.Status = domain.StatusReading
	if err := repo.Update(ctx, b); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.GetByID(ctx, b.ID)
	if got.PagesRead != 200 {
		t.Errorf("PagesRead: got %d, want 200", got.PagesRead)
	}
	if got.Status != domain.StatusReading {
		t.Errorf("Status: got %q", got.Status)
	}
}

func TestBookRepo_Update_NotFound(t *testing.T) {
	repo := newTestRepo(t)

	b := sampleBook()
	b.ID = 9999
	err := repo.Update(context.Background(), b)
	if !errors.Is(err, domain.ErrBookNotFound) {
		t.Fatalf("got %v, want ErrBookNotFound", err)
	}
}

func TestBookRepo_Delete(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	b := sampleBook()
	_ = repo.Create(ctx, b)

	if err := repo.Delete(ctx, b.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.GetByID(ctx, b.ID); !errors.Is(err, domain.ErrBookNotFound) {
		t.Fatalf("after delete: got %v, want ErrBookNotFound", err)
	}
}

// TestBookRepo_ContextCancellation is the most important test in this file.
// It proves context actually reaches the driver: a cancelled context should
// abort the query. If this test passes, context propagation is real, not
// decorative.
func TestBookRepo_ContextCancellation(t *testing.T) {
	repo := newTestRepo(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := repo.Create(ctx, sampleBook())
	if err == nil {
		t.Fatal("expected error on cancelled context, got nil")
	}
	// Note: we don't assert errors.Is(err, context.Canceled) because
	// the driver may wrap it. If you want strict checking, unwrap and
	// inspect. For this tutorial, "any error" is the assertion.
}
