// Package metrics provides functionality for collecting and managing application metrics.
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusMetrics implements the Metrics interface using Prometheus
type PrometheusMetrics struct {
	registry     *Registry
	promRegistry *prometheus.Registry
	enabled      bool
}

// NewPrometheusMetrics creates a new Prometheus metrics collector
func NewPrometheusMetrics(config *Config) *PrometheusMetrics {
	if config == nil {
		config = DefaultConfig()
	}

	registry := &Registry{}
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
func createMetricsRegistry(config *Config, promRegistry *prometheus.Registry) *Registry {
	registry := &Registry{}

	// Initialize different metric categories
	createImportMetrics(registry, config)
	createValidationMetrics(registry, config)
	createFileMetrics(registry, config)
	createOperationMetrics(registry, config)

	// Create performance metrics if detailed mode enabled
	if config.Detailed {
		createPerformanceMetrics(registry, config)
	}

	// Register metrics with prometheus
	registerCoreMetrics(registry, promRegistry)
	if config.Detailed {
		registerPerformanceMetrics(registry, promRegistry)
	}

	return registry
}

// createImportMetrics initializes import-related metrics
func createImportMetrics(registry *Registry, config *Config) {
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
}

// createValidationMetrics initializes validation-related metrics
func createValidationMetrics(registry *Registry, config *Config) {
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
}

// createFileMetrics initializes file-related metrics
func createFileMetrics(registry *Registry, config *Config) {
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
}

// createOperationMetrics initializes operation-related metrics
func createOperationMetrics(registry *Registry, config *Config) {
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
}

// createPerformanceMetrics initializes performance-related metrics
func createPerformanceMetrics(registry *Registry, config *Config) {
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

// registerCoreMetrics registers core metrics with prometheus
func registerCoreMetrics(registry *Registry, promRegistry *prometheus.Registry) {
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
}

// registerPerformanceMetrics registers performance metrics with prometheus
func registerPerformanceMetrics(registry *Registry, promRegistry *prometheus.Registry) {
	promRegistry.MustRegister(
		registry.MemoryUsage,
		registry.CPUUsage,
	)
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

// RecordImportDuration records the duration of an import operation (no-op for NullMetrics).
func (n *NullMetrics) RecordImportDuration(_, _ string, _ time.Duration) {}

// IncrementFilesProcessed increments the count of processed files (no-op for NullMetrics).
func (n *NullMetrics) IncrementFilesProcessed(_ string) {}

// IncrementRecordsProcessed increments the count of processed records (no-op for NullMetrics).
func (n *NullMetrics) IncrementRecordsProcessed(_ string, _ int) {}

// RecordBatchSize records the size of a batch operation (no-op for NullMetrics).
func (n *NullMetrics) RecordBatchSize(_ string, _ int) {}

// IncrementValidationErrors increments the count of validation errors (no-op for NullMetrics).
func (n *NullMetrics) IncrementValidationErrors(_ string) {}

// IncrementValidationRules increments the count of validation rules (no-op for NullMetrics).
func (n *NullMetrics) IncrementValidationRules(_ string) {}

// RecordValidationDuration records the duration of a validation operation (no-op for NullMetrics).
func (n *NullMetrics) RecordValidationDuration(_ string, _ time.Duration) {}

// RecordFileSize records the size of a file (no-op for NullMetrics).
func (n *NullMetrics) RecordFileSize(_ string, _ int64) {}

// RecordAttachmentSize records attachment size (no-op for NullMetrics)
func (n *NullMetrics) RecordAttachmentSize(_ int64) {}

// SetActiveOperations sets active operation count (no-op for NullMetrics)
func (n *NullMetrics) SetActiveOperations(_ string, _ int) {}

// IncrementOperationErrors increments operation error counts (no-op for NullMetrics)
func (n *NullMetrics) IncrementOperationErrors(_, _ string) {}

// RecordMemoryUsage records memory usage (no-op for NullMetrics)
func (n *NullMetrics) RecordMemoryUsage(_ string, _ int64) {}

// RecordCPUUsage records CPU usage (no-op for NullMetrics)
func (n *NullMetrics) RecordCPUUsage(_ string, _ float64) {}
