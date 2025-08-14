package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusMetrics implements the Metrics interface using Prometheus
type PrometheusMetrics struct {
	registry     *MetricsRegistry
	promRegistry *prometheus.Registry
	enabled      bool
}

// NewPrometheusMetrics creates a new Prometheus metrics collector
func NewPrometheusMetrics(config *MetricsConfig) *PrometheusMetrics {
	if config == nil {
		config = DefaultMetricsConfig()
	}

	registry := &MetricsRegistry{}
	promRegistry := prometheus.NewRegistry()

	// Only create metrics if enabled
	if config.Enabled {
		registry = createMetricsRegistry(config, promRegistry)
	}

	return &PrometheusMetrics{
		registry:     registry,
		promRegistry: promRegistry,
		enabled:      config.Enabled,
	}
}

// createMetricsRegistry creates and registers all Prometheus metrics
func createMetricsRegistry(config *MetricsConfig, promRegistry *prometheus.Registry) *MetricsRegistry {
	registry := &MetricsRegistry{}

	// Import metrics
	registry.ImportDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "import_duration_seconds",
			Help:      "Time spent importing files by type and status",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"file_type", "status"},
	)

	registry.FilesProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "files_processed_total",
			Help:      "Total number of files processed by type",
		},
		[]string{"file_type"},
	)

	registry.RecordsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "records_processed_total",
			Help:      "Total number of records processed by type",
		},
		[]string{"record_type"},
	)

	registry.BatchSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "batch_size",
			Help:      "Size of processing batches by operation",
			Buckets:   []float64{10, 50, 100, 500, 1000, 5000, 10000},
		},
		[]string{"operation"},
	)

	// Validation metrics
	registry.ValidationErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "validation_errors_total",
			Help:      "Total number of validation errors by type",
		},
		[]string{"violation_type"},
	)

	registry.ValidationRules = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "validation_rules_total",
			Help:      "Total number of validation rules executed by type",
		},
		[]string{"rule_type"},
	)

	registry.ValidationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "validation_duration_seconds",
			Help:      "Time spent on validation operations",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	// File metrics
	registry.FileSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "file_size_bytes",
			Help:      "Size of processed files by type",
			Buckets:   []float64{1024, 10240, 102400, 1048576, 10485760, 104857600, 1073741824}, // 1KB to 1GB
		},
		[]string{"file_type"},
	)

	registry.AttachmentSize = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "attachment_size_bytes",
			Help:      "Size of processed attachments",
			Buckets:   []float64{1024, 10240, 102400, 1048576, 10485760, 104857600}, // 1KB to 100MB
		},
	)

	// Operation metrics
	registry.ActiveOperations = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "active_operations",
			Help:      "Number of active operations by type",
		},
		[]string{"operation"},
	)

	registry.OperationErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "operation_errors_total",
			Help:      "Total number of operation errors by operation and error type",
		},
		[]string{"operation", "error_type"},
	)

	// Performance metrics (only if detailed metrics are enabled)
	if config.Detailed {
		registry.MemoryUsage = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "memory_usage_bytes",
				Help:      "Memory usage by operation",
			},
			[]string{"operation"},
		)

		registry.CPUUsage = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "cpu_usage_percent",
				Help:      "CPU usage percentage by operation",
			},
			[]string{"operation"},
		)
	}

	// Register all metrics with the custom registry
	promRegistry.MustRegister(
		registry.ImportDuration,
		registry.FilesProcessed,
		registry.RecordsProcessed,
		registry.BatchSize,
		registry.ValidationErrors,
		registry.ValidationRules,
		registry.ValidationDuration,
		registry.FileSize,
		registry.AttachmentSize,
		registry.ActiveOperations,
		registry.OperationErrors,
	)

	if config.Detailed {
		promRegistry.MustRegister(
			registry.MemoryUsage,
			registry.CPUUsage,
		)
	}

	return registry
}

// GetPrometheusRegistry returns the internal prometheus registry
func (m *PrometheusMetrics) GetPrometheusRegistry() *prometheus.Registry {
	return m.promRegistry
}

// RecordImportDuration records the duration of import operations
func (m *PrometheusMetrics) RecordImportDuration(fileType, status string, duration time.Duration) {
	if !m.enabled || m.registry.ImportDuration == nil {
		return
	}
	m.registry.ImportDuration.WithLabelValues(fileType, status).Observe(duration.Seconds())
}

// IncrementFilesProcessed increments the count of files processed
func (m *PrometheusMetrics) IncrementFilesProcessed(fileType string) {
	if !m.enabled || m.registry.FilesProcessed == nil {
		return
	}
	m.registry.FilesProcessed.WithLabelValues(fileType).Inc()
}

// IncrementRecordsProcessed increments the count of records processed
func (m *PrometheusMetrics) IncrementRecordsProcessed(recordType string, count int) {
	if !m.enabled || m.registry.RecordsProcessed == nil {
		return
	}
	m.registry.RecordsProcessed.WithLabelValues(recordType).Add(float64(count))
}

// RecordBatchSize records the size of processing batches
func (m *PrometheusMetrics) RecordBatchSize(operation string, size int) {
	if !m.enabled || m.registry.BatchSize == nil {
		return
	}
	m.registry.BatchSize.WithLabelValues(operation).Observe(float64(size))
}

// IncrementValidationErrors increments validation error count
func (m *PrometheusMetrics) IncrementValidationErrors(violationType string) {
	if !m.enabled || m.registry.ValidationErrors == nil {
		return
	}
	m.registry.ValidationErrors.WithLabelValues(violationType).Inc()
}

// IncrementValidationRules increments validation rule execution count
func (m *PrometheusMetrics) IncrementValidationRules(ruleType string) {
	if !m.enabled || m.registry.ValidationRules == nil {
		return
	}
	m.registry.ValidationRules.WithLabelValues(ruleType).Inc()
}

// RecordValidationDuration records validation operation duration
func (m *PrometheusMetrics) RecordValidationDuration(operation string, duration time.Duration) {
	if !m.enabled || m.registry.ValidationDuration == nil {
		return
	}
	m.registry.ValidationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordFileSize records file size metrics
func (m *PrometheusMetrics) RecordFileSize(fileType string, size int64) {
	if !m.enabled || m.registry.FileSize == nil {
		return
	}
	m.registry.FileSize.WithLabelValues(fileType).Observe(float64(size))
}

// RecordAttachmentSize records attachment size metrics
func (m *PrometheusMetrics) RecordAttachmentSize(size int64) {
	if !m.enabled || m.registry.AttachmentSize == nil {
		return
	}
	m.registry.AttachmentSize.Observe(float64(size))
}

// SetActiveOperations sets the count of active operations
func (m *PrometheusMetrics) SetActiveOperations(operation string, count int) {
	if !m.enabled || m.registry.ActiveOperations == nil {
		return
	}
	m.registry.ActiveOperations.WithLabelValues(operation).Set(float64(count))
}

// IncrementOperationErrors increments operation error count
func (m *PrometheusMetrics) IncrementOperationErrors(operation, errorType string) {
	if !m.enabled || m.registry.OperationErrors == nil {
		return
	}
	m.registry.OperationErrors.WithLabelValues(operation, errorType).Inc()
}

// RecordMemoryUsage records memory usage (only if detailed metrics enabled)
func (m *PrometheusMetrics) RecordMemoryUsage(operation string, bytes int64) {
	if !m.enabled || m.registry.MemoryUsage == nil {
		return
	}
	m.registry.MemoryUsage.WithLabelValues(operation).Set(float64(bytes))
}

// RecordCPUUsage records CPU usage (only if detailed metrics enabled)
func (m *PrometheusMetrics) RecordCPUUsage(operation string, percentage float64) {
	if !m.enabled || m.registry.CPUUsage == nil {
		return
	}
	m.registry.CPUUsage.WithLabelValues(operation).Set(percentage)
}

// NullMetrics implements the Metrics interface but does nothing (for disabled metrics)
type NullMetrics struct{}

// NewNullMetrics creates a null metrics collector
func NewNullMetrics() *NullMetrics {
	return &NullMetrics{}
}

// All methods are no-ops for NullMetrics
func (n *NullMetrics) RecordImportDuration(fileType, status string, duration time.Duration) {}
func (n *NullMetrics) IncrementFilesProcessed(fileType string)                              {}
func (n *NullMetrics) IncrementRecordsProcessed(recordType string, count int)               {}
func (n *NullMetrics) RecordBatchSize(operation string, size int)                           {}
func (n *NullMetrics) IncrementValidationErrors(violationType string)                       {}
func (n *NullMetrics) IncrementValidationRules(ruleType string)                             {}
func (n *NullMetrics) RecordValidationDuration(operation string, duration time.Duration)    {}
func (n *NullMetrics) RecordFileSize(fileType string, size int64)                           {}
func (n *NullMetrics) RecordAttachmentSize(size int64)                                      {}
func (n *NullMetrics) SetActiveOperations(operation string, count int)                      {}
func (n *NullMetrics) IncrementOperationErrors(operation, errorType string)                 {}
func (n *NullMetrics) RecordMemoryUsage(operation string, bytes int64)                      {}
func (n *NullMetrics) RecordCPUUsage(operation string, percentage float64)                  {}
