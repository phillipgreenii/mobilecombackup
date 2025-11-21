package metrics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestServer_HandleHealth(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	config.Enabled = true
	healthRegistry := NewHealthRegistry()
	promRegistry := prometheus.NewRegistry()
	server := NewServer(config, healthRegistry, promRegistry)

	t.Run("no_health_registry", func(t *testing.T) {
		// Create server without health registry
		serverNoHealth := NewServer(config, nil, prometheus.NewRegistry())
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()

		serverNoHealth.handleHealth(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})

	t.Run("healthy_status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()

		server.handleHealth(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["status"] == nil {
			t.Error("Expected status field in response")
		}
	})
}

func TestServer_HandleHealthCheck(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	config.Enabled = true
	healthRegistry := NewHealthRegistry()
	healthRegistry.Register(NewSystemHealthChecker())
	promRegistry := prometheus.NewRegistry()
	server := NewServer(config, healthRegistry, promRegistry)

	t.Run("no_health_registry", func(t *testing.T) {
		serverNoHealth := NewServer(config, nil, prometheus.NewRegistry())
		req := httptest.NewRequest(http.MethodGet, "/health/system", nil)
		w := httptest.NewRecorder()

		serverNoHealth.handleHealthCheck(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})

	t.Run("empty_check_name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health/", nil)
		w := httptest.NewRecorder()

		server.handleHealthCheck(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("valid_check", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health/system", nil)
		w := httptest.NewRecorder()

		server.handleHealthCheck(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["status"] == nil {
			t.Error("Expected status field in response")
		}
	})
}

func TestServer_HandleReadiness(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	config.Enabled = true
	healthRegistry := NewHealthRegistry()
	promRegistry := prometheus.NewRegistry()
	server := NewServer(config, healthRegistry, promRegistry)

	t.Run("no_health_registry", func(t *testing.T) {
		serverNoHealth := NewServer(config, nil, prometheus.NewRegistry())
		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		w := httptest.NewRecorder()

		serverNoHealth.handleReadiness(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})

	t.Run("ready_status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		w := httptest.NewRecorder()

		server.handleReadiness(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		ready, ok := response["ready"].(bool)
		if !ok {
			t.Error("Expected ready field to be boolean")
		}

		if !ready {
			t.Error("Expected ready to be true")
		}
	})
}

func TestServer_HandleLiveness(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	server := NewServer(config, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	w := httptest.NewRecorder()

	server.handleLiveness(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	alive, ok := response["alive"].(bool)
	if !ok {
		t.Error("Expected alive field to be boolean")
	}

	if !alive {
		t.Error("Expected alive to be true")
	}
}

func TestServer_HandleInfo(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	config.Enabled = true
	config.Namespace = "testapp"
	config.Path = "/custom-metrics"
	server := NewServer(config, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/info", nil)
	w := httptest.NewRecorder()

	server.handleInfo(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["application"] != "mobilecombackup" {
		t.Errorf("Expected application to be 'mobilecombackup', got %v", response["application"])
	}

	metricsInfo, ok := response["metrics"].(map[string]interface{})
	if !ok {
		t.Error("Expected metrics field to be object")
	}

	if metricsInfo["enabled"] != true {
		t.Error("Expected metrics enabled to be true")
	}

	if metricsInfo["namespace"] != "testapp" {
		t.Errorf("Expected namespace to be 'testapp', got %v", metricsInfo["namespace"])
	}
}

func TestServer_WriteJSON(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	server := NewServer(config, nil, nil)

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{"message": "test"}

		server.writeJSON(w, http.StatusOK, data)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got %s", contentType)
		}

		var response map[string]string
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["message"] != "test" {
			t.Errorf("Expected message to be 'test', got %s", response["message"])
		}
	})
}

func TestServer_WriteError(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	server := NewServer(config, nil, nil)

	w := httptest.NewRecorder()
	server.writeError(w, http.StatusBadRequest, "test error")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "test error" {
		t.Errorf("Expected error to be 'test error', got %v", response["error"])
	}
}

func TestServerManager_RegisterHealthCheck(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	manager := NewServerManager(config)

	// Create a custom health checker
	customChecker := &mockHealthChecker{
		name:   "custom",
		status: HealthStatusHealthy,
	}

	manager.RegisterHealthCheck(customChecker)

	// Verify the check was registered by checking the registry
	result := manager.server.healthRegistry.CheckOne("custom")
	if result.Name != "custom" {
		t.Errorf("Expected check name 'custom', got %s", result.Name)
	}
}

// mockHealthChecker is a simple mock for testing
type mockHealthChecker struct {
	name   string
	status HealthStatus
}

func (m *mockHealthChecker) Name() string {
	return m.name
}

func (m *mockHealthChecker) Check() *HealthCheck {
	return &HealthCheck{
		Name:        m.name,
		Status:      m.status,
		LastChecked: time.Now(),
	}
}
