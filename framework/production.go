package framework

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// --- Prometheus Metrics ---

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gails_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gails_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	dbQueryDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gails_db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	jobProcessedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gails_job_processed_total",
			Help: "Total number of background jobs processed",
		},
		[]string{"job_type", "status"},
	)

	cacheHitsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gails_cache_hits_total",
			Help: "Total number of cache hits",
		},
	)

	cacheMissesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gails_cache_misses_total",
			Help: "Total number of cache misses",
		},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(dbQueryDuration)
	prometheus.MustRegister(jobProcessedTotal)
	prometheus.MustRegister(cacheHitsTotal)
	prometheus.MustRegister(cacheMissesTotal)
}

// MetricsHandler returns the Prometheus metrics HTTP handler.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// RecordHTTPRequest records a metric for an HTTP request.
func RecordHTTPRequest(method, path string, status int, duration time.Duration) {
	httpRequestsTotal.WithLabelValues(method, path, http.StatusText(status)).Inc()
	httpRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// RecordDBQuery records a metric for a database query.
func RecordDBQuery(duration time.Duration) {
	dbQueryDuration.Observe(duration.Seconds())
}

// RecordJobProcessed records a metric for a background job.
func RecordJobProcessed(jobType, status string) {
	jobProcessedTotal.WithLabelValues(jobType, status).Inc()
}

// RecordCacheHit records a cache hit metric.
func RecordCacheHit() {
	cacheHitsTotal.Inc()
}

// RecordCacheMiss records a cache miss metric.
func RecordCacheMiss() {
	cacheMissesTotal.Inc()
}

// --- Panic Breadcrumbs ---

// Breadcrumb is a log entry captured for panic diagnostics.
type Breadcrumb struct {
	Timestamp time.Time
	Level     string
	Message   string
	RequestID string
}

// BreadcrumbRing is a thread-safe ring buffer of breadcrumbs per request ID.
type BreadcrumbRing struct {
	mu      sync.Mutex
	entries map[string][]Breadcrumb
	maxSize int
}

// GlobalBreadcrumbs is the global breadcrumb ring buffer.
var GlobalBreadcrumbs = NewBreadcrumbRing(10)

// NewBreadcrumbRing creates a new ring buffer with the given max size per request.
func NewBreadcrumbRing(maxSize int) *BreadcrumbRing {
	return &BreadcrumbRing{
		entries: make(map[string][]Breadcrumb),
		maxSize: maxSize,
	}
}

// Add adds a breadcrumb for a request ID.
func (br *BreadcrumbRing) Add(requestID, level, message string) {
	br.mu.Lock()
	defer br.mu.Unlock()

	crumbs := br.entries[requestID]
	crumbs = append(crumbs, Breadcrumb{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		RequestID: requestID,
	})
	// Keep only the last maxSize entries
	if len(crumbs) > br.maxSize {
		crumbs = crumbs[len(crumbs)-br.maxSize:]
	}
	br.entries[requestID] = crumbs
}

// Get retrieves breadcrumbs for a request ID.
func (br *BreadcrumbRing) Get(requestID string) []Breadcrumb {
	br.mu.Lock()
	defer br.mu.Unlock()
	return br.entries[requestID]
}

// Clear removes breadcrumbs for a request ID.
func (br *BreadcrumbRing) Clear(requestID string) {
	br.mu.Lock()
	defer br.mu.Unlock()
	delete(br.entries, requestID)
}
