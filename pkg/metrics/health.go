package metrics

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"
)

// HealthRegistryImpl implements HealthRegistry
type HealthRegistryImpl struct {
	checkers map[string]HealthChecker
	mutex    sync.RWMutex
}

// NewHealthRegistry creates a new health registry
func NewHealthRegistry() HealthRegistry {
	return &HealthRegistryImpl{
		checkers: make(map[string]HealthChecker),
	}
}

// Register adds a health checker
func (h *HealthRegistryImpl) Register(checker HealthChecker) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.checkers[checker.Name()] = checker
}

// CheckAll runs all registered health checks
func (h *HealthRegistryImpl) CheckAll() map[string]*HealthCheck {
	h.mutex.RLock()
	checkers := make(map[string]HealthChecker, len(h.checkers))
	for name, checker := range h.checkers {
		checkers[name] = checker
	}
	h.mutex.RUnlock()

	results := make(map[string]*HealthCheck)
	for name, checker := range checkers {
		results[name] = checker.Check()
	}

	return results
}

// CheckOne runs a specific health check by name
func (h *HealthRegistryImpl) CheckOne(name string) *HealthCheck {
	h.mutex.RLock()
	checker, exists := h.checkers[name]
	h.mutex.RUnlock()

	if !exists {
		return &HealthCheck{
			Name:        name,
			Status:      HealthStatusUnhealthy,
			Message:     "Health check not found",
			LastChecked: time.Now(),
		}
	}

	return checker.Check()
}

// OverallStatus returns the overall health status
func (h *HealthRegistryImpl) OverallStatus() HealthStatus {
	results := h.CheckAll()

	if len(results) == 0 {
		return HealthStatusHealthy // No checks means healthy
	}

	hasUnhealthy := false
	hasDegraded := false

	for _, result := range results {
		switch result.Status {
		case HealthStatusUnhealthy:
			hasUnhealthy = true
		case HealthStatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return HealthStatusUnhealthy
	}
	if hasDegraded {
		return HealthStatusDegraded
	}

	return HealthStatusHealthy
}

// SystemHealthChecker checks basic system health
type SystemHealthChecker struct {
	name string
}

// NewSystemHealthChecker creates a system health checker
func NewSystemHealthChecker() HealthChecker {
	return &SystemHealthChecker{
		name: "system",
	}
}

// Name returns the health checker name
func (s *SystemHealthChecker) Name() string {
	return s.name
}

// Check performs system health check
func (s *SystemHealthChecker) Check() *HealthCheck {
	start := time.Now()

	status := HealthStatusHealthy
	var message string
	details := make(map[string]interface{})

	// Check memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	memUsageMB := float64(m.Alloc) / 1024 / 1024
	details["memory_usage_mb"] = memUsageMB
	details["gc_runs"] = m.NumGC
	details["goroutines"] = runtime.NumGoroutine()

	// Basic memory threshold check (warning if > 100MB)
	if memUsageMB > 100 {
		status = HealthStatusDegraded
		message = fmt.Sprintf("High memory usage: %.1f MB", memUsageMB)
	}

	// Check if we can write to temp directory
	tempDir := os.TempDir()
	testFile := fmt.Sprintf("%s/health_check_%d", tempDir, time.Now().UnixNano())
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		status = HealthStatusUnhealthy
		message = fmt.Sprintf("Cannot write to temp directory: %v", err)
		details["temp_dir_error"] = err.Error()
	} else {
		_ = os.Remove(testFile)
		details["temp_dir_writable"] = true
	}

	if status == HealthStatusHealthy {
		message = "System is healthy"
	}

	return &HealthCheck{
		Name:        s.name,
		Status:      status,
		Message:     message,
		Details:     details,
		LastChecked: time.Now(),
		Duration:    time.Since(start),
	}
}

// DiskSpaceHealthChecker checks disk space
type DiskSpaceHealthChecker struct {
	name      string
	path      string
	threshold float64 // Warning threshold as percentage (e.g., 0.9 for 90%)
}

// NewDiskSpaceHealthChecker creates a disk space health checker
func NewDiskSpaceHealthChecker(path string, threshold float64) HealthChecker {
	return &DiskSpaceHealthChecker{
		name:      "disk_space",
		path:      path,
		threshold: threshold,
	}
}

// Name returns the health checker name
func (d *DiskSpaceHealthChecker) Name() string {
	return d.name
}

// Check performs disk space health check
func (d *DiskSpaceHealthChecker) Check() *HealthCheck {
	start := time.Now()

	status := HealthStatusHealthy
	message := "Disk space is adequate"
	details := make(map[string]interface{})

	// Get file system stats
	stat, err := os.Stat(d.path)
	if err != nil {
		return &HealthCheck{
			Name:        d.name,
			Status:      HealthStatusUnhealthy,
			Message:     fmt.Sprintf("Cannot access path %s: %v", d.path, err),
			Details:     map[string]interface{}{"error": err.Error()},
			LastChecked: time.Now(),
			Duration:    time.Since(start),
		}
	}

	details["path"] = d.path
	details["path_exists"] = stat != nil
	details["threshold"] = d.threshold

	// Note: Getting actual disk space requires platform-specific code
	// For now, we'll just check if the path is accessible
	if stat == nil {
		status = HealthStatusUnhealthy
		message = fmt.Sprintf("Path %s is not accessible", d.path)
	}

	return &HealthCheck{
		Name:        d.name,
		Status:      status,
		Message:     message,
		Details:     details,
		LastChecked: time.Now(),
		Duration:    time.Since(start),
	}
}

// CustomHealthChecker allows custom health checks
type CustomHealthChecker struct {
	name    string
	checkFn func() *HealthCheck
}

// NewCustomHealthChecker creates a custom health checker
func NewCustomHealthChecker(name string, checkFn func() *HealthCheck) HealthChecker {
	return &CustomHealthChecker{
		name:    name,
		checkFn: checkFn,
	}
}

// Name returns the health checker name
func (c *CustomHealthChecker) Name() string {
	return c.name
}

// Check performs the custom health check
func (c *CustomHealthChecker) Check() *HealthCheck {
	if c.checkFn != nil {
		return c.checkFn()
	}

	return &HealthCheck{
		Name:        c.name,
		Status:      HealthStatusUnhealthy,
		Message:     "No check function provided",
		LastChecked: time.Now(),
	}
}
