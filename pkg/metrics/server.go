package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server provides HTTP endpoints for metrics and health checks
type Server struct {
	config         *Config
	healthRegistry HealthRegistry
	promRegistry   *prometheus.Registry
	server         *http.Server
}

// NewServer creates a new metrics server
func NewServer(config *Config, healthRegistry HealthRegistry, promRegistry *prometheus.Registry) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	if promRegistry == nil {
		promRegistry = prometheus.NewRegistry()
	}

	return &Server{
		config:         config,
		healthRegistry: healthRegistry,
		promRegistry:   promRegistry,
	}
}

// Start starts the metrics server
func (s *Server) Start() error {
	if !s.config.Enabled {
		return nil // Don't start server if metrics are disabled
	}

	mux := http.NewServeMux()

	// Prometheus metrics endpoint
	mux.Handle(s.config.Path, promhttp.HandlerFor(s.promRegistry, promhttp.HandlerOpts{}))

	// Health check endpoints
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/health/", s.handleHealthCheck)
	mux.HandleFunc("/ready", s.handleReadiness)
	mux.HandleFunc("/live", s.handleLiveness)

	// Info endpoint
	mux.HandleFunc("/info", s.handleInfo)

	s.server = &http.Server{
		Addr:         s.config.Listen,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s.server.ListenAndServe()
}

// Stop stops the metrics server
func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// handleHealth handles the main health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	if s.healthRegistry == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Health registry not configured")
		return
	}

	results := s.healthRegistry.CheckAll()
	overallStatus := s.healthRegistry.OverallStatus()

	response := map[string]interface{}{
		"status":    overallStatus,
		"checks":    results,
		"timestamp": time.Now(),
	}

	var statusCode int
	switch overallStatus {
	case HealthStatusHealthy:
		statusCode = http.StatusOK
	case HealthStatusDegraded:
		statusCode = http.StatusOK // Still OK, but degraded
	case HealthStatusUnhealthy:
		statusCode = http.StatusServiceUnavailable
	default:
		statusCode = http.StatusInternalServerError
	}

	s.writeJSON(w, statusCode, response)
}

// handleHealthCheck handles individual health check endpoints
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if s.healthRegistry == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Health registry not configured")
		return
	}

	// Extract health check name from path
	name := r.URL.Path[len("/health/"):]
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "Health check name required")
		return
	}

	result := s.healthRegistry.CheckOne(name)

	var statusCode int
	switch result.Status {
	case HealthStatusHealthy:
		statusCode = http.StatusOK
	case HealthStatusDegraded:
		statusCode = http.StatusOK
	case HealthStatusUnhealthy:
		statusCode = http.StatusServiceUnavailable
	default:
		statusCode = http.StatusInternalServerError
	}

	s.writeJSON(w, statusCode, result)
}

// handleReadiness handles Kubernetes-style readiness probe
func (s *Server) handleReadiness(w http.ResponseWriter, _ *http.Request) {
	if s.healthRegistry == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Health registry not configured")
		return
	}

	overallStatus := s.healthRegistry.OverallStatus()

	response := map[string]interface{}{
		"ready":     overallStatus == HealthStatusHealthy,
		"status":    overallStatus,
		"timestamp": time.Now(),
	}

	var statusCode int
	if overallStatus == HealthStatusHealthy {
		statusCode = http.StatusOK
	} else {
		statusCode = http.StatusServiceUnavailable
	}

	s.writeJSON(w, statusCode, response)
}

// handleLiveness handles Kubernetes-style liveness probe
func (s *Server) handleLiveness(w http.ResponseWriter, _ *http.Request) {
	// Liveness is simpler - just check if the service is running
	response := map[string]interface{}{
		"alive":     true,
		"timestamp": time.Now(),
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleInfo handles application info endpoint
func (s *Server) handleInfo(w http.ResponseWriter, _ *http.Request) {
	response := map[string]interface{}{
		"application": "mobilecombackup",
		"version":     "dev", // This would be injected at build time
		"metrics": map[string]interface{}{
			"enabled":   s.config.Enabled,
			"namespace": s.config.Namespace,
			"endpoint":  s.config.Path,
		},
		"timestamp": time.Now(),
	}

	s.writeJSON(w, http.StatusOK, response)
}

// writeJSON writes a JSON response
func (s *Server) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Fallback error handling
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// writeError writes an error response
func (s *Server) writeError(w http.ResponseWriter, statusCode int, message string) {
	response := map[string]interface{}{
		"error":     message,
		"timestamp": time.Now(),
	}
	s.writeJSON(w, statusCode, response)
}

// ServerManager manages the metrics server lifecycle
type ServerManager struct {
	server  *Server
	started bool
}

// NewServerManager creates a new metrics server manager
func NewServerManager(config *Config) *ServerManager {
	healthRegistry := NewHealthRegistry()

	// Register default health checks
	healthRegistry.Register(NewSystemHealthChecker())
	healthRegistry.Register(NewDiskSpaceHealthChecker(".", 0.9))

	// Create a separate prometheus registry for the server
	promRegistry := prometheus.NewRegistry()

	server := NewServer(config, healthRegistry, promRegistry)

	return &ServerManager{
		server: server,
	}
}

// Start starts the metrics server in a goroutine
func (m *ServerManager) Start() error {
	if m.started {
		return fmt.Errorf("metrics server already started")
	}

	if !m.server.config.Enabled {
		return nil // Don't start if disabled
	}

	go func() {
		if err := m.server.Start(); err != nil && err != http.ErrServerClosed {
			// Log error (would use logger in real implementation)
			fmt.Printf("Metrics server error: %v\n", err)
		}
	}()

	m.started = true
	return nil
}

// Stop stops the metrics server
func (m *ServerManager) Stop() error {
	if !m.started {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := m.server.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop metrics server: %w", err)
	}

	m.started = false
	return nil
}

// RegisterHealthCheck registers a custom health check
func (m *ServerManager) RegisterHealthCheck(checker HealthChecker) {
	if m.server.healthRegistry != nil {
		m.server.healthRegistry.Register(checker)
	}
}
