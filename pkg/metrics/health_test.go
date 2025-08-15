package metrics

import (
	"strings"
	"testing"
	"time"
)

const (
	testHealthCheckName = "system"
)

func TestNewHealthRegistry(t *testing.T) {
	registry := NewHealthRegistry()
	if registry == nil {
		t.Fatal("NewHealthRegistry returned nil")
	}
}

func TestHealthRegistry_Register(t *testing.T) {
	registry := NewHealthRegistry()
	checker := NewSystemHealthChecker()

	registry.Register(checker)

	// Check that the checker was registered
	result := registry.CheckOne(testHealthCheckName)
	if result == nil {
		t.Fatal("Expected health check result, got nil")
	}

	if result.Name != testHealthCheckName {
		t.Errorf("Expected health check name 'system', got %s", result.Name)
	}
}

func TestHealthRegistry_CheckAll(t *testing.T) {
	registry := NewHealthRegistry()

	// Register multiple checkers
	registry.Register(NewSystemHealthChecker())
	registry.Register(NewDiskSpaceHealthChecker(".", 0.9))

	results := registry.CheckAll()
	if len(results) != 2 {
		t.Errorf("Expected 2 health check results, got %d", len(results))
	}

	if _, exists := results[testHealthCheckName]; !exists {
		t.Error("Expected 'system' health check in results")
	}

	if _, exists := results["disk_space"]; !exists {
		t.Error("Expected 'disk_space' health check in results")
	}
}

func TestHealthRegistry_CheckOne(t *testing.T) {
	registry := NewHealthRegistry()
	registry.Register(NewSystemHealthChecker())

	// Test existing checker
	result := registry.CheckOne(testHealthCheckName)
	if result == nil {
		t.Fatal("Expected health check result, got nil")
	}

	if result.Name != testHealthCheckName {
		t.Errorf("Expected name 'system', got %s", result.Name)
	}

	// Test non-existent checker
	result = registry.CheckOne("nonexistent")
	if result.Status != HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy status for non-existent checker, got %s", result.Status)
	}

	if !strings.Contains(result.Message, "not found") {
		t.Errorf("Expected 'not found' in message, got: %s", result.Message)
	}
}

func TestHealthRegistry_OverallStatus(t *testing.T) {
	registry := NewHealthRegistry()

	// Test with no checkers
	status := registry.OverallStatus()
	if status != HealthStatusHealthy {
		t.Errorf("Expected healthy status with no checkers, got %s", status)
	}

	// Test with healthy checker
	registry.Register(NewSystemHealthChecker())
	status = registry.OverallStatus()
	// System health check should generally be healthy
	if status == HealthStatusUnhealthy {
		t.Errorf("Expected non-unhealthy status, got %s", status)
	}
}

func TestSystemHealthChecker(t *testing.T) {
	checker := NewSystemHealthChecker()

	if checker.Name() != testHealthCheckName {
		t.Errorf("Expected name 'system', got %s", checker.Name())
	}

	result := checker.Check()
	if result == nil {
		t.Fatal("Expected health check result, got nil")
	}

	if result.Name != testHealthCheckName {
		t.Errorf("Expected result name 'system', got %s", result.Name)
	}

	if result.LastChecked.IsZero() {
		t.Error("Expected LastChecked to be set")
	}

	if result.Duration == 0 {
		t.Error("Expected Duration to be set")
	}

	// Check that details are populated
	if result.Details == nil {
		t.Fatal("Expected details to be set")
	}

	if _, exists := result.Details["memory_usage_mb"]; !exists {
		t.Error("Expected memory_usage_mb in details")
	}

	if _, exists := result.Details["goroutines"]; !exists {
		t.Error("Expected goroutines in details")
	}

	if _, exists := result.Details["temp_dir_writable"]; !exists {
		t.Error("Expected temp_dir_writable in details")
	}
}

func TestDiskSpaceHealthChecker(t *testing.T) {
	checker := NewDiskSpaceHealthChecker(".", 0.9)

	if checker.Name() != "disk_space" {
		t.Errorf("Expected name 'disk_space', got %s", checker.Name())
	}

	result := checker.Check()
	if result == nil {
		t.Fatal("Expected health check result, got nil")
	}

	if result.Name != "disk_space" {
		t.Errorf("Expected result name 'disk_space', got %s", result.Name)
	}

	// Check that details are populated
	if result.Details == nil {
		t.Fatal("Expected details to be set")
	}

	if path, exists := result.Details["path"]; !exists || path != "." {
		t.Errorf("Expected path '.' in details, got %v", path)
	}

	if threshold, exists := result.Details["threshold"]; !exists || threshold != 0.9 {
		t.Errorf("Expected threshold 0.9 in details, got %v", threshold)
	}
}

func TestDiskSpaceHealthChecker_InvalidPath(t *testing.T) {
	checker := NewDiskSpaceHealthChecker("/nonexistent/path", 0.9)

	result := checker.Check()
	if result.Status != HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy status for invalid path, got %s", result.Status)
	}

	if !strings.Contains(result.Message, "Cannot access path") {
		t.Errorf("Expected 'Cannot access path' in message, got: %s", result.Message)
	}
}

func TestCustomHealthChecker(t *testing.T) {
	// Test with valid check function
	customCheck := func() *HealthCheck {
		return &HealthCheck{
			Name:        "custom",
			Status:      HealthStatusHealthy,
			Message:     "Custom check passed",
			LastChecked: time.Now(),
		}
	}

	checker := NewCustomHealthChecker("custom", customCheck)

	if checker.Name() != "custom" {
		t.Errorf("Expected name 'custom', got %s", checker.Name())
	}

	result := checker.Check()
	if result.Status != HealthStatusHealthy {
		t.Errorf("Expected healthy status, got %s", result.Status)
	}

	if result.Message != "Custom check passed" {
		t.Errorf("Expected custom message, got: %s", result.Message)
	}
}

func TestCustomHealthChecker_NoFunction(t *testing.T) {
	checker := NewCustomHealthChecker("custom", nil)

	result := checker.Check()
	if result.Status != HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy status with no check function, got %s", result.Status)
	}

	if !strings.Contains(result.Message, "No check function") {
		t.Errorf("Expected 'No check function' in message, got: %s", result.Message)
	}
}

func TestHealthStatus_String(t *testing.T) {
	tests := []struct {
		status   HealthStatus
		expected string
	}{
		{HealthStatusHealthy, "healthy"},
		{HealthStatusUnhealthy, "unhealthy"},
		{HealthStatusDegraded, "degraded"},
	}

	for _, test := range tests {
		if string(test.status) != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, string(test.status))
		}
	}
}
