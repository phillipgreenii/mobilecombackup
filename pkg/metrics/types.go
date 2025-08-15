package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics interface defines the metrics collection API
type Metrics interface {
	// Import metrics
	RecordImportDuration(fileType, status string, duration time.Duration)
	IncrementFilesProcessed(fileType string)
	IncrementRecordsProcessed(recordType string, count int)
	RecordBatchSize(operation string, size int)

	// Validation metrics
	IncrementValidationErrors(violationType string)
	IncrementValidationRules(ruleType string)
	RecordValidationDuration(operation string, duration time.Duration)

	// File metrics
	RecordFileSize(fileType string, size int64)
	RecordAttachmentSize(size int64)

	// Operation metrics
	SetActiveOperations(operation string, count int)
	IncrementOperationErrors(operation, errorType string)

	// Performance metrics
	RecordMemoryUsage(operation string, bytes int64)
	RecordCPUUsage(operation string, percentage float64)
}

// Registry holds all Prometheus metrics
type Registry struct {
	// Import metrics
	ImportDuration   *prometheus.HistogramVec
	FilesProcessed   *prometheus.CounterVec
	RecordsProcessed *prometheus.CounterVec
	BatchSize        *prometheus.HistogramVec

	// Validation metrics
	ValidationErrors   *prometheus.CounterVec
	ValidationRules    *prometheus.CounterVec
	ValidationDuration *prometheus.HistogramVec

	// File metrics
	FileSize       *prometheus.HistogramVec
	AttachmentSize prometheus.Histogram

	// Operation metrics
	ActiveOperations *prometheus.GaugeVec
	OperationErrors  *prometheus.CounterVec

	// Performance metrics
	MemoryUsage *prometheus.GaugeVec
	CPUUsage    *prometheus.GaugeVec
}

// Config holds metrics configuration
type Config struct {
	Enabled   bool   `yaml:"enabled" mapstructure:"enabled"`
	Namespace string `yaml:"namespace" mapstructure:"namespace"`
	Subsystem string `yaml:"subsystem" mapstructure:"subsystem"`
	Listen    string `yaml:"listen" mapstructure:"listen"`     // Address to listen on for metrics endpoint
	Path      string `yaml:"path" mapstructure:"path"`         // HTTP path for metrics endpoint
	Detailed  bool   `yaml:"detailed" mapstructure:"detailed"` // Enable detailed metrics collection
}

// DefaultConfig returns the default metrics configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:   false, // Disabled by default
		Namespace: "mobilecombackup",
		Subsystem: "",
		Listen:    ":9090",
		Path:      "/metrics",
		Detailed:  false,
	}
}

// HealthStatus represents the health status of the application
type HealthStatus string

const (
	// HealthStatusHealthy indicates all systems are functioning normally
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusUnhealthy indicates critical systems are failing
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	// HealthStatusDegraded indicates some systems are impaired but still functional
	HealthStatusDegraded HealthStatus = "degraded"
)

// HealthCheck represents a health check result
type HealthCheck struct {
	Name        string                 `json:"name"`
	Status      HealthStatus           `json:"status"`
	Message     string                 `json:"message,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	LastChecked time.Time              `json:"last_checked"`
	Duration    time.Duration          `json:"duration"`
}

// HealthChecker interface for health checks
type HealthChecker interface {
	// Check performs a health check and returns the result
	Check() *HealthCheck

	// Name returns the name of the health check
	Name() string
}

// HealthRegistry manages health checks
type HealthRegistry interface {
	// Register adds a health checker
	Register(checker HealthChecker)

	// CheckAll runs all registered health checks
	CheckAll() map[string]*HealthCheck

	// CheckOne runs a specific health check by name
	CheckOne(name string) *HealthCheck

	// OverallStatus returns the overall health status
	OverallStatus() HealthStatus
}
