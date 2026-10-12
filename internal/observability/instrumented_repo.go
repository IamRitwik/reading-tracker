package observability

import (
	"context"
	"log/slog"
	"time"

	"github.com/IamRitwik/reading-tracker/internal/domain"
	"github.com/IamRitwik/reading-tracker/internal/repository"
)

// slowQueryThreshold is where we start warning. For a local SQLite this
// should basically never trigger; if it does, something is wrong (disk
// contention, huge table, missing index).
const slowQueryThreshold = 100 * time.Millisecond

// InstrumentedBookRepo wraps any repository.BookRepository and records
// Prometheus metrics around every call.
//
// This is the DECORATOR pattern. The inner repo (sqlite, memory, whatever)
// has no idea metrics exist. We can wrap, unwrap, and stack decorators
// freely. Ordering matters: if you later add a caching decorator, put it
// INSIDE instrumentation so cache hits are still counted as operations.
//
// Compile-time assertion: if someone changes the interface, this fails to build.
var _ repository.BookRepository = (*InstrumentedBookRepo)(nil)

type InstrumentedBookRepo struct {
	next    repository.BookRepository
	metrics *Metrics
	logger  *slog.Logger
}

// NewInstrumentedBookRepo returns a repo that records metrics for every
// operation. Pass logger=nil to use slog.Default().
func NewInstrumentedBookRepo(
	next repository.BookRepository,
	metrics *Metrics,
	logger *slog.Logger,
) *InstrumentedBookRepo {
	if logger == nil {
		logger = slog.Default()
	}
	return &InstrumentedBookRepo{next: next, metrics: metrics, logger: logger}
}

func (r *InstrumentedBookRepo) Create(ctx context.Context, b *domain.Book) error {
	start := time.Now()
	err := r.next.Create(ctx, b)
	r.observe(ctx, "create", start, err)
	if err == nil {
		r.metrics.BooksAdded.Inc()
	}
	return err
}

func (r *InstrumentedBookRepo) GetByID(ctx context.Context, id int64) (*domain.Book, error) {
	start := time.Now()
	b, err := r.next.GetByID(ctx, id)
	r.observe(ctx, "get_by_id", start, err)
	return b, err
}

func (r *InstrumentedBookRepo) List(ctx context.Context, f repository.BookFilter) ([]*domain.Book, error) {
	start := time.Now()
	books, err := r.next.List(ctx, f)
	r.observe(ctx, "list", start, err)
	return books, err
}

func (r *InstrumentedBookRepo) Update(ctx context.Context, b *domain.Book) error {
	start := time.Now()
	err := r.next.Update(ctx, b)
	r.observe(ctx, "update", start, err)
	if err == nil {
		r.metrics.BooksUpdated.Inc()
	}
	return err
}

func (r *InstrumentedBookRepo) Delete(ctx context.Context, id int64) error {
	start := time.Now()
	err := r.next.Delete(ctx, id)
	r.observe(ctx, "delete", start, err)
	if err == nil {
		r.metrics.BooksDeleted.Inc()
	}
	return err
}

// observe is the one place where we record duration + error + slow warnings.
// Every method funnels through here, so the metric names and label values
// stay consistent no matter how many methods we add.
func (r *InstrumentedBookRepo) observe(ctx context.Context, op string, start time.Time, err error) {
	dur := time.Since(start)
	result := "ok"
	if err != nil {
		result = "error"
		r.metrics.DBQueryErrors.WithLabelValues(op).Inc()
	}
	r.metrics.DBQueryDuration.WithLabelValues(op, result).Observe(dur.Seconds())

	if dur > slowQueryThreshold {
		r.logger.WarnContext(ctx, "slow repository call",
			slog.String("operation", op),
			slog.Duration("duration", dur),
			slog.String("result", result),
		)
	}
}
