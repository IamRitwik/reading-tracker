package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/IamRitwik/reading-tracker/internal/domain"
	"github.com/IamRitwik/reading-tracker/internal/repository"
)

// Package service contains the business logic of the reading tracker.
//
// The service layer is the only place that:
//   - Validates user input against domain rules
//   - Enforces state-machine rules (status transitions)
//   - Emits domain-level logs
//
// It never imports database/sql, cobra, or bubbletea.
// It depends only on the repository interface and the domain package.

type BookService struct {
	repo           repository.BookRepository
	logger         *slog.Logger
	defaultTimeout time.Duration
}

// NewBookService wires up the service.
//   - logger: child logger (e.g. logger.With("component", "book-service")).
//     Never nil — pass slog.Default() if you truly have nothing.
//   - defaultTimeout: applied per operation. Pass 0 to disable.
//     A caller-supplied shorter deadline always wins (Go's context
//     semantics: the effective deadline is the minimum).
func NewBookService(
	repo repository.BookRepository,
	logger *slog.Logger,
	defaultTimeout time.Duration,
) *BookService {
	if logger == nil {
		logger = slog.Default()
	}
	return &BookService{
		repo:           repo,
		logger:         logger,
		defaultTimeout: defaultTimeout,
	}

}

// --- Inputs ------------------------------------------------------------

// AddBookInput is the parameter object for Add. Separate from domain.Book
// on purpose: what a user provides is not the same as what we store.
// Fields like ID, CreatedAt, PagesRead, Status are derived, not provided.
type AddBookInput struct {
	Title       string
	Author      string
	Category    string
	Description string
	TotalPages  int
}

// UpdateDetailsInput edits metadata. Pointer fields = "only update if set."
// This is the standard partial-update pattern in Go.
type UpdateDetailsInput struct {
	Title       *string
	Author      *string
	Category    *string
	Description *string
	TotalPages  *int
}

// ListInput mirrors repository.BookFilter but lives in the service package
// so callers never import repository. Small duplication, big decoupling win.
type ListInput struct {
	Status   *domain.Status
	Category *string
	Author   *string
	Search   *string
	Limit    int
	Offset   int
}

// --- Public API --------------------------------------------------------

// Add creates a new book in the "want_to_read" state.
func (s *BookService) Add(ctx context.Context, in AddBookInput) (*domain.Book, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	in.Title = strings.TrimSpace(in.Title)
	in.Author = strings.TrimSpace(in.Author)
	in.Category = strings.TrimSpace(in.Category)

	err := validateAdd(in)
	if err != nil {
		s.logger.WarnContext(ctx, "add rejected",
			slog.String("reason", err.Error()),
			slog.String("title", in.Title),
		)
		return nil, err
	}

	b := &domain.Book{
		Title:       in.Title,
		Author:      in.Author,
		Category:    in.Category,
		Description: in.Description,
		TotalPages:  in.TotalPages,
		PagesRead:   0,
		Status:      domain.StatusWantToRead,
	}

	err = s.repo.Create(ctx, b)
	if err != nil {
		return nil, fmt.Errorf("create book: %w", err)
	}

	s.logger.InfoContext(ctx, "book added",
		slog.Int64("id", b.ID),
		slog.String("title", b.Title),
		slog.String("author", b.Author),
	)

	return b, nil
}

// Get returns a single book by ID.
func (s *BookService) Get(ctx context.Context, id int64) (*domain.Book, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	b, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// List returns books matching the filter.
func (s *BookService) List(ctx context.Context, in ListInput) ([]*domain.Book, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	return s.repo.List(ctx, repository.BookFilter{
		Status:   in.Status,
		Category: in.Category,
		Author:   in.Author,
		Search:   in.Search,
		Limit:    in.Limit,
		Offset:   in.Offset,
	})
}

// UpdateProgress sets pages read and derives status automatically.
//
// Rules:
//   - pagesRead must be in [0, totalPages]
//   - pagesRead == 0          → status want_to_read
//   - pagesRead == totalPages → status completed
//   - otherwise               → status reading
//
// This is the operation a user hits most often, so it optimizes for the
// common case: "I read up to page N."
func (s *BookService) UpdateProgress(ctx context.Context, id int64, pagesRead int) (*domain.Book, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	b, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if pagesRead < 0 || pagesRead > b.TotalPages {
		return nil, fmt.Errorf("%w: got %d, max %d", domain.ErrInvalidProgress, pagesRead, b.TotalPages)
	}

	prevStatus := b.Status
	b.PagesRead = pagesRead
	b.Status = deriveStatus(pagesRead, b.TotalPages)

	err = s.repo.Update(ctx, b)
	if err != nil {
		return nil, fmt.Errorf("update progress: %w", err)
	}

	logArgs := []any{
		slog.Int64("id", b.ID),
		slog.Int("pages_read", b.PagesRead),
		slog.Int("total_pages", b.TotalPages),
		slog.String("status", string(b.Status)),
	}
	if prevStatus != b.Status {
		logArgs = append(logArgs,
			slog.String("prev_status", string(prevStatus)),
		)
	}
	s.logger.InfoContext(ctx, "book progress updated", logArgs...)

	// A completion is a milestone — log it at info with a distinct message.
	if prevStatus != domain.StatusCompleted && b.Status == domain.StatusCompleted {
		s.logger.InfoContext(ctx, "book completed",
			slog.Int64("id", b.ID),
			slog.String("title", b.Title),
			slog.String("author", b.Author),
		)
	}
	return b, nil
}

// SetStatus explicitly changes status, enforcing transition rules.
// Used when the user wants to abandon, restart, or manually mark complete.
func (s *BookService) SetStatus(ctx context.Context, id int64, target domain.Status) (*domain.Book, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	if !target.Valid() {
		return nil, fmt.Errorf("%w: %q", domain.ErrInvalidStatus, target)
	}

	b, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if b.Status == target {
		return b, nil // idempotent
	}
	if !canTransition(b.Status, target) {
		return nil, fmt.Errorf("cannot transition from %q to %q",
			b.Status, target)
	}

	prev := b.Status
	b.Status = target

	// Keep pagesRead consistent with the new status.
	switch target {
	case domain.StatusWantToRead:
		b.PagesRead = 0
	case domain.StatusCompleted:
		b.PagesRead = b.TotalPages
	}

	if err := s.repo.Update(ctx, b); err != nil {
		return nil, fmt.Errorf("set status: %w", err)
	}

	s.logger.InfoContext(ctx, "book status changed",
		slog.Int64("id", b.ID),
		slog.String("from", string(prev)),
		slog.String("to", string(target)),
	)
	return b, nil
}

// UpdateDetails edits metadata. TotalPages changes are checked against
// current pagesRead so we never end up with pagesRead > totalPages.
func (s *BookService) UpdateDetails(ctx context.Context, id int64, in UpdateDetailsInput) (*domain.Book, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	b, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Title != nil {
		v := strings.TrimSpace(*in.Title)
		if v == "" {
			return nil, domain.ErrInvalidTitle
		}
		b.Title = v
	}
	if in.Author != nil {
		v := strings.TrimSpace(*in.Author)
		if v == "" {
			return nil, domain.ErrInvalidAuthor
		}
		b.Author = v
	}
	if in.Category != nil {
		b.Category = strings.TrimSpace(*in.Category)
	}
	if in.Description != nil {
		b.Description = *in.Description
	}
	if in.TotalPages != nil {
		if *in.TotalPages <= 0 {
			return nil, domain.ErrInvalidTotalPages
		}
		if *in.TotalPages < b.PagesRead {
			return nil, fmt.Errorf("%w: new total %d < pages read %d",
				domain.ErrInvalidProgress, *in.TotalPages, b.PagesRead)
		}
		b.TotalPages = *in.TotalPages
		// If the book now fits exactly, keep status honest.
		if b.PagesRead == b.TotalPages && b.Status != domain.StatusCompleted {
			b.Status = domain.StatusCompleted
		}
	}

	if err := s.repo.Update(ctx, b); err != nil {
		return nil, fmt.Errorf("update details: %w", err)
	}

	s.logger.InfoContext(ctx, "book details updated", slog.Int64("id", b.ID))
	return b, nil
}

// Delete removes a book.
func (s *BookService) Delete(ctx context.Context, id int64) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.logger.InfoContext(ctx, "book deleted", slog.Int64("id", id))
	return nil
}

// --- Internals ---------------------------------------------------------

// withTimeout wraps ctx with the service's default timeout if — and only
// if — the caller hasn't already set a deadline. Go's context.WithTimeout
// picks the *earlier* deadline automatically, so if the caller set 100ms
// and we set 2s, the caller's 100ms wins. That's the behavior we want:
// the caller knows best how long it's willing to wait.
func (s *BookService) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.defaultTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, s.defaultTimeout)
}

func validateAdd(in AddBookInput) error {
	if in.Title == "" {
		return domain.ErrInvalidTitle
	}
	if in.Author == "" {
		return domain.ErrInvalidAuthor
	}
	if in.TotalPages <= 0 {
		return domain.ErrInvalidTotalPages
	}
	return nil
}

// deriveStatus implements the auto-status rules documented on UpdateProgress.
func deriveStatus(pagesRead, totalPages int) domain.Status {
	switch {
	case pagesRead <= 0:
		return domain.StatusWantToRead
	case pagesRead >= totalPages:
		return domain.StatusCompleted
	default:
		return domain.StatusReading
	}
}

// allowedTransitions encodes the state machine. If you later need a
// "paused" status, this is the only function that changes.
var allowedTransitions = map[domain.Status][]domain.Status{
	domain.StatusWantToRead: {domain.StatusReading, domain.StatusAbandoned},
	domain.StatusReading:    {domain.StatusCompleted, domain.StatusAbandoned, domain.StatusWantToRead},
	domain.StatusCompleted:  {domain.StatusReading}, // reread
	domain.StatusAbandoned:  {domain.StatusReading, domain.StatusWantToRead},
}

func canTransition(from domain.Status, to domain.Status) bool {
	for _, allowed := range allowedTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}
