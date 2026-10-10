package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/IamRitwik/reading-tracker/internal/domain"
	"github.com/IamRitwik/reading-tracker/internal/repository/memory"
	"github.com/IamRitwik/reading-tracker/internal/service"
)

// newSvc returns a BookService wired to an in-memory repo and a silent logger.
// Silent because test output should be *intentional* — logs from successful
// operations just add noise. When a test fails, t.Fatalf gives us everything
// we need.
func newSvc(t *testing.T) *service.BookService {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return service.NewBookService(memory.NewBookRepo(), logger, 0)
}

func mustAdd(t *testing.T, svc *service.BookService, title string, pages int) *domain.Book {
	t.Helper()
	b, err := svc.Add(context.Background(), service.AddBookInput{
		Title:      title,
		Author:     "Test Author",
		Category:   "Test",
		TotalPages: pages,
	})
	if err != nil {
		t.Fatalf("Add(%q): %v", title, err)
	}
	return b
}

// --- Add -----------------------------------------------------------------

func TestService_Add_HappyPath(t *testing.T) {
	svc := newSvc(t)
	b := mustAdd(t, svc, "Effective Go", 200)

	if b.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if b.Status != domain.StatusWantToRead {
		t.Errorf("new book status: got %q, want %q", b.Status, domain.StatusWantToRead)
	}
	if b.PagesRead != 0 {
		t.Errorf("new book PagesRead: got %d, want 0", b.PagesRead)
	}
}

func TestService_Add_Validation(t *testing.T) {
	svc := newSvc(t)
	ctx := context.Background()

	cases := []struct {
		name string
		in   service.AddBookInput
		want error
	}{
		{"empty title", service.AddBookInput{Author: "A", TotalPages: 10}, domain.ErrInvalidTitle},
		{"whitespace title", service.AddBookInput{Title: "   ", Author: "A", TotalPages: 10}, domain.ErrInvalidTitle},
		{"empty author", service.AddBookInput{Title: "T", TotalPages: 10}, domain.ErrInvalidAuthor},
		{"zero pages", service.AddBookInput{Title: "T", Author: "A"}, domain.ErrInvalidTotalPages},
		{"negative pages", service.AddBookInput{Title: "T", Author: "A", TotalPages: -5}, domain.ErrInvalidTotalPages},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Add(ctx, tc.in)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

// --- UpdateProgress ------------------------------------------------------

func TestService_UpdateProgress_Transitions(t *testing.T) {
	svc := newSvc(t)
	ctx := context.Background()

	b := mustAdd(t, svc, "Progress Book", 100)

	// 0 pages → still want_to_read
	got, err := svc.UpdateProgress(ctx, b.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StatusWantToRead {
		t.Errorf("at 0/100: got %q, want %q", got.Status, domain.StatusWantToRead)
	}

	// Mid → reading
	got, err = svc.UpdateProgress(ctx, b.ID, 40)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StatusReading {
		t.Errorf("at 40/100: got %q, want %q", got.Status, domain.StatusReading)
	}

	// Exact total → completed
	got, err = svc.UpdateProgress(ctx, b.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StatusCompleted {
		t.Errorf("at 100/100: got %q, want %q", got.Status, domain.StatusCompleted)
	}
}

func TestService_UpdateProgress_OutOfRange(t *testing.T) {
	svc := newSvc(t)
	ctx := context.Background()
	b := mustAdd(t, svc, "Out of Range", 50)

	for _, pages := range []int{-1, 51, 1000} {
		_, err := svc.UpdateProgress(ctx, b.ID, pages)
		if !errors.Is(err, domain.ErrInvalidProgress) {
			t.Errorf("pages=%d: got %v, want ErrInvalidProgress", pages, err)
		}
	}
}

func TestService_UpdateProgress_NotFound(t *testing.T) {
	svc := newSvc(t)
	_, err := svc.UpdateProgress(context.Background(), 9999, 10)
	if !errors.Is(err, domain.ErrBookNotFound) {
		t.Fatalf("got %v, want ErrBookNotFound", err)
	}
}

// --- SetStatus -----------------------------------------------------------

func TestService_SetStatus_AllowedAndDisallowed(t *testing.T) {
	svc := newSvc(t)
	ctx := context.Background()
	b := mustAdd(t, svc, "State Machine", 200)

	// want_to_read → completed is disallowed (must read it first)
	if _, err := svc.SetStatus(ctx, b.ID, domain.StatusCompleted); err == nil {
		t.Fatal("want_to_read → completed should be rejected")
	}

	// want_to_read → reading is allowed
	if _, err := svc.SetStatus(ctx, b.ID, domain.StatusReading); err != nil {
		t.Fatalf("want_to_read → reading: %v", err)
	}

	// reading → completed is allowed; completion forces pagesRead = total
	got, err := svc.SetStatus(ctx, b.ID, domain.StatusCompleted)
	if err != nil {
		t.Fatal(err)
	}
	if got.PagesRead != got.TotalPages {
		t.Errorf("completed: PagesRead=%d, want %d", got.PagesRead, got.TotalPages)
	}

	// completed → reading is allowed (reread)
	if _, err := svc.SetStatus(ctx, b.ID, domain.StatusReading); err != nil {
		t.Fatalf("completed → reading (reread): %v", err)
	}
}

func TestService_SetStatus_Idempotent(t *testing.T) {
	svc := newSvc(t)
	b := mustAdd(t, svc, "Idempotent", 100)

	got, err := svc.SetStatus(context.Background(), b.ID, domain.StatusWantToRead)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != b.ID {
		t.Fatal("idempotent SetStatus should return same book")
	}
}

func TestService_SetStatus_InvalidTarget(t *testing.T) {
	svc := newSvc(t)
	b := mustAdd(t, svc, "Bad Target", 100)

	_, err := svc.SetStatus(context.Background(), b.ID, domain.Status("garbage"))
	if !errors.Is(err, domain.ErrInvalidStatus) {
		t.Fatalf("got %v, want ErrInvalidStatus", err)
	}
}

// --- UpdateDetails -------------------------------------------------------

func TestService_UpdateDetails_ShrinkTotalPagesBelowProgress(t *testing.T) {
	svc := newSvc(t)
	ctx := context.Background()

	b := mustAdd(t, svc, "Shrink", 300)
	if _, err := svc.UpdateProgress(ctx, b.ID, 150); err != nil {
		t.Fatal(err)
	}

	newTotal := 100 // less than pagesRead (150)
	_, err := svc.UpdateDetails(ctx, b.ID, service.UpdateDetailsInput{
		TotalPages: &newTotal,
	})
	if !errors.Is(err, domain.ErrInvalidProgress) {
		t.Fatalf("got %v, want ErrInvalidProgress", err)
	}
}

func TestService_UpdateDetails_ShrinkToExactProgress(t *testing.T) {
	svc := newSvc(t)
	ctx := context.Background()

	b := mustAdd(t, svc, "Shrink to exact", 300)
	if _, err := svc.UpdateProgress(ctx, b.ID, 150); err != nil {
		t.Fatal(err)
	}

	newTotal := 150
	got, err := svc.UpdateDetails(ctx, b.ID, service.UpdateDetailsInput{
		TotalPages: &newTotal,
	})
	if err != nil {
		t.Fatal(err)
	}
	// pagesRead == total now → status should auto-promote to completed
	if got.Status != domain.StatusCompleted {
		t.Errorf("got %q, want %q", got.Status, domain.StatusCompleted)
	}
}

// --- Context -------------------------------------------------------------

func TestService_ContextCancellation(t *testing.T) {
	svc := newSvc(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.Add(ctx, service.AddBookInput{
		Title: "Cancelled", Author: "A", TotalPages: 1,
	})
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
}

// TestService_TimeoutRespectsShorterCallerDeadline proves our withTimeout
// helper doesn't override a caller's stricter deadline.
func TestService_TimeoutRespectsShorterCallerDeadline(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.NewBookService(memory.NewBookRepo(), logger, 10*time.Second)

	// Caller gives 1ns — will expire before the repo even runs.
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond) // ensure the 1ns has definitely expired

	_, err := svc.Add(ctx, service.AddBookInput{
		Title: "Too slow", Author: "A", TotalPages: 1,
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v, want context.DeadlineExceeded", err)
	}
}
