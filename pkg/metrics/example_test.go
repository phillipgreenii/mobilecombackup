package metrics_test

import (
	"fmt"
	"log"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/metrics"
)

// Example of using Prometheus metrics for import operations
func ExamplePrometheusMetrics() {
	// Create metrics configuration
	config := &metrics.Config{
		Enabled:   true,
		Namespace: "mobilecombackup",
		Subsystem: "import",
		Listen:    ":9090",
		Path:      "/metrics",
		Detailed:  false,
	}

	// Create metrics collector
	metricsCollector := metrics.NewPrometheusMetrics(config)

	// Record some metrics during import
	metricsCollector.IncrementFilesProcessed("calls")
	metricsCollector.IncrementRecordsProcessed("calls", 150)
	metricsCollector.RecordImportDuration("calls", "success", 2*time.Second)
	metricsCollector.RecordFileSize("calls", 1024*1024) // 1MB

	fmt.Println("Import metrics recorded")
	// Output: Import metrics recorded
}

// Example of using the health check system
func ExampleHealthRegistry() {
	// Create health registry
	registry := metrics.NewHealthRegistry()

	// Register health checkers
	registry.Register(metrics.NewSystemHealthChecker())
	registry.Register(metrics.NewDiskSpaceHealthChecker(".", 0.9))

	// Check overall health status
	status := registry.OverallStatus()
	fmt.Printf("Overall health status: %s\n", status)

	// Check specific health check
	result := registry.CheckOne("system")
	fmt.Printf("System health: %s\n", result.Status)

	// Output:
	// Overall health status: healthy
	// System health: healthy
}

// Example of running metrics server with health checks
func ExampleServerManager() {
	config := &metrics.Config{
		Enabled: true,
		Listen:  ":9090",
		Path:    "/metrics",
	}

	// Create server manager with default health checks
	manager := metrics.NewServerManager(config)

	// Start the server (in production, handle errors appropriately)
	if err := manager.Start(); err != nil {
		log.Printf("Failed to start metrics server: %v", err)
		return
	}

	// Server is now running and serving:
	// - /metrics (Prometheus metrics)
	// - /health (overall health status)
	// - /health/system (system health check)
	// - /health/disk_space (disk space health check)
	// - /ready (readiness probe)
	// - /live (liveness probe)
	// - /info (application info)

	fmt.Println("Metrics server started")

	// Stop the server when done
	defer func() {
		if err := manager.Stop(); err != nil {
			log.Printf("Failed to stop metrics server: %v", err)
		}
	}()

	// Output: Metrics server started
}

// Example of creating custom health checks
func ExampleCustomHealthChecker() {
	// Define a custom health check function
	customCheck := func() *metrics.HealthCheck {
		// Perform custom health validation
		isHealthy := true // Your custom logic here

		status := metrics.HealthStatusHealthy
		message := "Custom service is running"

		if !isHealthy {
			status = metrics.HealthStatusUnhealthy
			message = "Custom service is not responding"
		}

		return &metrics.HealthCheck{
			Name:        "custom_service",
			Status:      status,
			Message:     message,
			Details:     map[string]interface{}{"checked_at": time.Now()},
			LastChecked: time.Now(),
			Duration:    10 * time.Millisecond,
		}
	}

	// Create custom health checker
	checker := metrics.NewCustomHealthChecker("custom_service", customCheck)

	// Register with health registry
	registry := metrics.NewHealthRegistry()
	registry.Register(checker)

	// Check the custom health status
	result := registry.CheckOne("custom_service")
	fmt.Printf("Custom health status: %s\n", result.Status)

	// Output: Custom health status: healthy
}

// Example of using default metrics configuration
func ExampleDefaultConfig() {
	// Get default configuration (metrics disabled by default)
	config := metrics.DefaultConfig()

	fmt.Printf("Enabled: %t\n", config.Enabled)
	fmt.Printf("Namespace: %s\n", config.Namespace)
	fmt.Printf("Listen: %s\n", config.Listen)
	fmt.Printf("Path: %s\n", config.Path)

	// Output:
	// Enabled: false
	// Namespace: mobilecombackup
	// Listen: :9090
	// Path: /metrics
}

// Example of using null metrics (when metrics are disabled)
func ExampleNullMetrics() {
	// Create null metrics collector (safe no-op implementation)
	metricsCollector := metrics.NewNullMetrics()

	// All method calls are safe and do nothing
	metricsCollector.IncrementFilesProcessed("calls")
	metricsCollector.RecordImportDuration("calls", "success", time.Second)
	metricsCollector.RecordFileSize("calls", 1024)

	fmt.Println("Null metrics operations completed")
	// Output: Null metrics operations completed
}
