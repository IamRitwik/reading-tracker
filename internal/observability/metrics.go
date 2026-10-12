package observability

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const metricsNamespace = "reading_tracker"

// Metrics holds every Prometheus collector the app uses.
// Construct once in main, pass by pointer wherever needed.
// All fields are safe for concurrent use (Prometheus guarantees this).
type Metrics struct {
	// Registry is private to this app — we do NOT use the global default.
	// Using a private registry keeps tests isolated (each test gets a fresh
	// registry) and prevents duplicate-registration panics when multiple
	// components ship their own collectors.
	Registry *prometheus.Registry

	BooksAdded   prometheus.Counter
	BooksUpdated prometheus.Counter
	BooksDeleted prometheus.Counter

	DBQueryDuration *prometheus.HistogramVec // labels: operation, result
	DBQueryErrors   *prometheus.CounterVec   // label:  operation
}

// NewMetrics constructs and registers all collectors on a private registry.
func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()

	m := &Metrics{
		Registry: reg,

		BooksAdded: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "books_added_total",
			Help:      "Total number of books added.",
		}),
		BooksUpdated: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "books_updated_total",
			Help:      "Total number of successful book updates.",
		}),
		BooksDeleted: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "books_deleted_total",
			Help:      "Total number of books deleted.",
		}),

		// Custom buckets instead of prometheus.DefBuckets. Default buckets
		// start at 5ms, which is too coarse for a local SQLite — everything
		// would land in the first bucket and we'd learn nothing.
		DBQueryDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "db_query_duration_seconds",
			Help:      "Duration of repository operations in seconds.",
			Buckets:   []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		}, []string{"operation", "result"}),

		DBQueryErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "db_query_errors_total",
			Help:      "Total number of repository errors by operation.",
		}, []string{"operation"}),
	}

	reg.MustRegister(
		m.BooksAdded,
		m.BooksUpdated,
		m.BooksDeleted,
		m.DBQueryDuration,
		m.DBQueryErrors,

		// Go runtime and process metrics. Optional but standard in prod:
		//   reading_tracker_go_goroutines
		//   reading_tracker_go_memstats_alloc_bytes
		//   reading_tracker_process_cpu_seconds_total
		// Comment these out if you want a cleaner /metrics output.
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return m
}

// Handler returns an http.Handler that renders the registry in
// Prometheus exposition format.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{
		// If a collector panics during scrape, return 500 rather than
		// crashing the whole /metrics endpoint.
		ErrorHandling: promhttp.HTTPErrorOnError,
	})
}
