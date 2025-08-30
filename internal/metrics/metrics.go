package metrics

import (
	"context"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all application metrics
type Metrics struct {
	// API metrics
	apiRequestsTotal   *prometheus.CounterVec
	apiRequestDuration *prometheus.HistogramVec
	apiRequestErrors   *prometheus.CounterVec

	// Database metrics
	dbOperationsTotal   *prometheus.CounterVec
	dbOperationDuration *prometheus.HistogramVec
	dbConnectionPool    *prometheus.GaugeVec

	// Business metrics
	programsDiscovered *prometheus.CounterVec
	assetsDiscovered   *prometheus.CounterVec
	scansCompleted     *prometheus.CounterVec
	scansFailed        *prometheus.CounterVec

	// System metrics
	memoryUsage         *prometheus.GaugeVec
	goroutineCount      *prometheus.GaugeVec
	circuitBreakerState *prometheus.GaugeVec

	// Platform-specific metrics
	platformRequestsTotal   *prometheus.CounterVec
	platformRequestDuration *prometheus.HistogramVec
	platformErrors          *prometheus.CounterVec
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		// API metrics
		apiRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "monitor_agent_api_requests_total",
				Help: "Total number of API requests",
			},
			[]string{"method", "endpoint", "status_code"},
		),
		apiRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "monitor_agent_api_request_duration_seconds",
				Help:    "API request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),
		apiRequestErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "monitor_agent_api_request_errors_total",
				Help: "Total number of API request errors",
			},
			[]string{"method", "endpoint", "error_type"},
		),

		// Database metrics
		dbOperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "monitor_agent_db_operations_total",
				Help: "Total number of database operations",
			},
			[]string{"operation", "table"},
		),
		dbOperationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "monitor_agent_db_operation_duration_seconds",
				Help:    "Database operation duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation", "table"},
		),
		dbConnectionPool: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "monitor_agent_db_connection_pool",
				Help: "Database connection pool status",
			},
			[]string{"status"},
		),

		// Business metrics
		programsDiscovered: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "monitor_agent_programs_discovered_total",
				Help: "Total number of programs discovered",
			},
			[]string{"platform"},
		),
		assetsDiscovered: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "monitor_agent_assets_discovered_total",
				Help: "Total number of assets discovered",
			},
			[]string{"platform", "source"},
		),
		scansCompleted: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "monitor_agent_scans_completed_total",
				Help: "Total number of scans completed",
			},
			[]string{"platform"},
		),
		scansFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "monitor_agent_scans_failed_total",
				Help: "Total number of scans failed",
			},
			[]string{"platform", "error_type"},
		),

		// System metrics
		memoryUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "monitor_agent_memory_usage_bytes",
				Help: "Memory usage in bytes",
			},
			[]string{"type"},
		),
		goroutineCount: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "monitor_agent_goroutines_total",
				Help: "Number of goroutines",
			},
			[]string{},
		),
		circuitBreakerState: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "monitor_agent_circuit_breaker_state",
				Help: "Circuit breaker state (0=closed, 1=half-open, 2=open)",
			},
			[]string{"service"},
		),

		// Platform-specific metrics
		platformRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "monitor_agent_platform_requests_total",
				Help: "Total number of platform API requests",
			},
			[]string{"platform", "endpoint", "status_code"},
		),
		platformRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "monitor_agent_platform_request_duration_seconds",
				Help:    "Platform API request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"platform", "endpoint"},
		),
		platformErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "monitor_agent_platform_errors_total",
				Help: "Total number of platform API errors",
			},
			[]string{"platform", "error_type"},
		),
	}
}

// RecordAPIRequest records an API request
func (m *Metrics) RecordAPIRequest(method, endpoint string, statusCode int, duration time.Duration) {
	m.apiRequestsTotal.WithLabelValues(method, endpoint, string(rune(statusCode))).Inc()
	m.apiRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// RecordAPIError records an API error
func (m *Metrics) RecordAPIError(method, endpoint, errorType string) {
	m.apiRequestErrors.WithLabelValues(method, endpoint, errorType).Inc()
}

// RecordDatabaseOperation records a database operation
func (m *Metrics) RecordDatabaseOperation(operation, table string, duration time.Duration) {
	m.dbOperationsTotal.WithLabelValues(operation, table).Inc()
	m.dbOperationDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// UpdateConnectionPool updates connection pool metrics
func (m *Metrics) UpdateConnectionPool(open, idle int) {
	m.dbConnectionPool.WithLabelValues("open").Set(float64(open))
	m.dbConnectionPool.WithLabelValues("idle").Set(float64(idle))
}

// RecordProgramDiscovered records a discovered program
func (m *Metrics) RecordProgramDiscovered(platform string) {
	m.programsDiscovered.WithLabelValues(platform).Inc()
}

// RecordAssetDiscovered records a discovered asset
func (m *Metrics) RecordAssetDiscovered(platform, source string) {
	m.assetsDiscovered.WithLabelValues(platform, source).Inc()
}

// RecordScanCompleted records a completed scan
func (m *Metrics) RecordScanCompleted(platform string) {
	m.scansCompleted.WithLabelValues(platform).Inc()
}

// RecordScanFailed records a failed scan
func (m *Metrics) RecordScanFailed(platform, errorType string) {
	m.scansFailed.WithLabelValues(platform, errorType).Inc()
}

// UpdateSystemMetrics updates system metrics
func (m *Metrics) UpdateSystemMetrics() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.memoryUsage.WithLabelValues("alloc").Set(float64(memStats.Alloc))
	m.memoryUsage.WithLabelValues("sys").Set(float64(memStats.Sys))
	m.memoryUsage.WithLabelValues("heap_alloc").Set(float64(memStats.HeapAlloc))
	m.memoryUsage.WithLabelValues("heap_sys").Set(float64(memStats.HeapSys))

	m.goroutineCount.WithLabelValues().Set(float64(runtime.NumGoroutine()))
}

// UpdateCircuitBreakerState updates circuit breaker state
func (m *Metrics) UpdateCircuitBreakerState(service, state string) {
	var stateValue float64
	switch state {
	case "closed":
		stateValue = 0
	case "half-open":
		stateValue = 1
	case "open":
		stateValue = 2
	}
	m.circuitBreakerState.WithLabelValues(service).Set(stateValue)
}

// RecordPlatformRequest records a platform API request
func (m *Metrics) RecordPlatformRequest(platform, endpoint string, statusCode int, duration time.Duration) {
	m.platformRequestsTotal.WithLabelValues(platform, endpoint, string(rune(statusCode))).Inc()
	m.platformRequestDuration.WithLabelValues(platform, endpoint).Observe(duration.Seconds())
}

// RecordPlatformError records a platform API error
func (m *Metrics) RecordPlatformError(platform, errorType string) {
	m.platformErrors.WithLabelValues(platform, errorType).Inc()
}

// StartMetricsCollection starts periodic metrics collection
func (m *Metrics) StartMetricsCollection(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.UpdateSystemMetrics()
		}
	}
}
