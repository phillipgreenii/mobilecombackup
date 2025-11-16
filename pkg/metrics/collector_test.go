package metrics

import (
	"testing"
	"time"
)

func TestNewPrometheusMetrics(t *testing.T) {
	// Test with nil config
	metrics := NewPrometheusMetrics(nil)
	if metrics == nil {
		t.Fatal("NewPrometheusMetrics returned nil")
	}

	if metrics.enabled {
		t.Error("Expected metrics to be disabled with nil config")
	}

	// Test with enabled config
	config := &Config{
		Enabled:   true,
		Namespace: "test",
		Subsystem: "unit",
	}

	metrics = NewPrometheusMetrics(config)
	if !metrics.enabled {
		t.Error("Expected metrics to be enabled")
	}

	if metrics.registry == nil {
		t.Fatal("Expected registry to be created")
	}
}

func TestPrometheusMetrics_RecordImportDuration(t *testing.T) {
	config := &Config{
		Enabled:   true,
		Namespace: "test",
		Subsystem: "unit",
	}

	metrics := NewPrometheusMetrics(config)

	// Should not panic
	metrics.RecordImportDuration("calls", "success", time.Second)
	metrics.RecordImportDuration("sms", "error", 500*time.Millisecond)
}

func TestPrometheusMetrics_IncrementFilesProcessed(t *testing.T) {
	config := &Config{
		Enabled:   true,
		Namespace: "test",
		Subsystem: "unit",
	}

	metrics := NewPrometheusMetrics(config)

	// Should not panic
	metrics.IncrementFilesProcessed("calls")
	metrics.IncrementFilesProcessed("sms")
}

func TestPrometheusMetrics_IncrementRecordsProcessed(t *testing.T) {
	config := &Config{
		Enabled:   true,
		Namespace: "test",
		Subsystem: "unit",
	}

	metrics := NewPrometheusMetrics(config)

	// Should not panic
	metrics.IncrementRecordsProcessed("calls", 100)
	metrics.IncrementRecordsProcessed("sms", 250)
}

func TestPrometheusMetrics_RecordBatchSize(t *testing.T) {
	config := &Config{
		Enabled:   true,
		Namespace: "test",
		Subsystem: "unit",
	}

	metrics := NewPrometheusMetrics(config)

	// Should not panic
	metrics.RecordBatchSize("import", 1000)
	metrics.RecordBatchSize("validation", 500)
}

func TestPrometheusMetrics_ValidationMetrics(t *testing.T) {
	config := &Config{
		Enabled:   true,
		Namespace: "test",
		Subsystem: "unit",
	}

	metrics := NewPrometheusMetrics(config)

	// Should not panic
	metrics.IncrementValidationErrors("count_mismatch")
	metrics.IncrementValidationRules("structure_check")
	metrics.RecordValidationDuration("full_validation", 2*time.Second)
}

func TestPrometheusMetrics_FileMetrics(t *testing.T) {
	config := &Config{
		Enabled:   true,
		Namespace: "test",
		Subsystem: "unit",
	}

	metrics := NewPrometheusMetrics(config)

	// Should not panic
	metrics.RecordFileSize("calls", 1024*1024) // 1MB
	metrics.RecordAttachmentSize(512 * 1024)   // 512KB
}

func TestPrometheusMetrics_OperationMetrics(t *testing.T) {
	config := &Config{
		Enabled:   true,
		Namespace: "test",
		Subsystem: "unit",
	}

	metrics := NewPrometheusMetrics(config)

	// Should not panic
	metrics.SetActiveOperations("import", 3)
	metrics.IncrementOperationErrors("import", "timeout")
}

func TestPrometheusMetrics_PerformanceMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	config := &Config{
		Enabled:   true,
		Namespace: "test",
		Subsystem: "unit",
		Detailed:  true, // Enable detailed metrics
	}

	metrics := NewPrometheusMetrics(config)

	// Should not panic
	metrics.RecordMemoryUsage("import", 100*1024*1024) // 100MB
	metrics.RecordCPUUsage("validation", 75.5)         // 75.5%
}

func TestPrometheusMetrics_DisabledMetrics(t *testing.T) {
	config := &Config{
		Enabled: false,
	}

	metrics := NewPrometheusMetrics(config)

	// All methods should be safe to call when disabled
	metrics.RecordImportDuration("calls", "success", time.Second)
	metrics.IncrementFilesProcessed("calls")
	metrics.IncrementRecordsProcessed("calls", 100)
	metrics.RecordBatchSize("import", 1000)
	metrics.IncrementValidationErrors("count_mismatch")
	metrics.IncrementValidationRules("structure_check")
	metrics.RecordValidationDuration("full_validation", 2*time.Second)
	metrics.RecordFileSize("calls", 1024*1024)
	metrics.RecordAttachmentSize(512 * 1024)
	metrics.SetActiveOperations("import", 3)
	metrics.IncrementOperationErrors("import", "timeout")
	metrics.RecordMemoryUsage("import", 100*1024*1024)
	metrics.RecordCPUUsage("validation", 75.5)
}

func TestNullMetrics(t *testing.T) {
	metrics := NewNullMetrics()

	// All methods should be safe to call and do nothing
	metrics.RecordImportDuration("calls", "success", time.Second)
	metrics.IncrementFilesProcessed("calls")
	metrics.IncrementRecordsProcessed("calls", 100)
	metrics.RecordBatchSize("import", 1000)
	metrics.IncrementValidationErrors("count_mismatch")
	metrics.IncrementValidationRules("structure_check")
	metrics.RecordValidationDuration("full_validation", 2*time.Second)
	metrics.RecordFileSize("calls", 1024*1024)
	metrics.RecordAttachmentSize(512 * 1024)
	metrics.SetActiveOperations("import", 3)
	metrics.IncrementOperationErrors("import", "timeout")
	metrics.RecordMemoryUsage("import", 100*1024*1024)
	metrics.RecordCPUUsage("validation", 75.5)

	// Test passes if no panic occurs
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Enabled {
		t.Error("Expected metrics to be disabled by default")
	}

	if config.Namespace != "mobilecombackup" {
		t.Errorf("Expected namespace to be 'mobilecombackup', got %s", config.Namespace)
	}

	if config.Listen != ":9090" {
		t.Errorf("Expected listen address to be ':9090', got %s", config.Listen)
	}

	if config.Path != "/metrics" {
		t.Errorf("Expected path to be '/metrics', got %s", config.Path)
	}

	if config.Detailed {
		t.Error("Expected detailed metrics to be disabled by default")
	}
}

func TestPrometheusMetrics_GetPrometheusRegistry(t *testing.T) {
	config := &Config{
		Enabled:   true,
		Namespace: "test",
		Subsystem: "unit",
	}

	metrics := NewPrometheusMetrics(config)
	registry := metrics.GetPrometheusRegistry()

	if registry == nil {
		t.Error("GetPrometheusRegistry() returned nil for enabled metrics")
	}

}
