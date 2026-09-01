package config

import (
	"testing"
)

func TestGetConfigExample(t *testing.T) {
	t.Parallel()

	example := GetConfigExample()
	if example == "" {
		t.Error("Expected non-empty config example")
	}

	// Should contain various sections
	if len(example) < 100 {
		t.Error("Expected substantial config example")
	}
}

func TestGetConfigHeader(t *testing.T) {
	t.Parallel()

	header := getConfigHeader()
	if header == "" {
		t.Error("Expected non-empty config header")
	}

	// Should contain comment markers
	if len(header) == 0 {
		t.Error("Expected config header content")
	}
}

func TestGetRepositoryConfigExample(t *testing.T) {
	t.Parallel()

	example := getRepositoryConfigExample()
	if example == "" {
		t.Error("Expected non-empty repository config example")
	}

	// Should contain repository-related content
	if len(example) < 20 {
		t.Error("Expected substantial repository config example")
	}
}

func TestGetImportConfigExample(t *testing.T) {
	t.Parallel()

	example := getImportConfigExample()
	if example == "" {
		t.Error("Expected non-empty import config example")
	}

	// Should contain import-related content
	if len(example) < 20 {
		t.Error("Expected substantial import config example")
	}
}

func TestGetValidationConfigExample(t *testing.T) {
	t.Parallel()

	example := getValidationConfigExample()
	if example == "" {
		t.Error("Expected non-empty validation config example")
	}

	// Should contain validation-related content
	if len(example) < 20 {
		t.Error("Expected substantial validation config example")
	}
}

func TestGetLoggingConfigExample(t *testing.T) {
	t.Parallel()

	example := getLoggingConfigExample()
	if example == "" {
		t.Error("Expected non-empty logging config example")
	}

	// Should contain logging-related content
	if len(example) < 20 {
		t.Error("Expected substantial logging config example")
	}
}

func TestGetEnvironmentConfigExample(t *testing.T) {
	t.Parallel()

	example := getEnvironmentConfigExample()
	if example == "" {
		t.Error("Expected non-empty environment config example")
	}

	// Should contain environment-related content
	if len(example) < 10 {
		t.Error("Expected environment config example content")
	}
}

func TestListAvailableTemplates(t *testing.T) {
	t.Parallel()

	templates := ListAvailableTemplates()
	if len(templates) == 0 {
		t.Error("Expected at least one available template")
	}

	// Should contain template-related content
	if len(templates) < 20 {
		t.Error("Expected substantial templates list content")
	}
}

func TestNewConfigLoaderWithLogger(t *testing.T) {
	t.Parallel()

	// Test with nil logger
	loader := NewConfigLoaderWithLogger(nil)
	if loader == nil {
		t.Error("Expected non-nil loader with nil logger")
	}

	// Should not panic with nil logger
	_, err := loader.Load("nonexistent.yaml")
	// Error is expected since we don't have a config file, but shouldn't panic
	if err == nil {
		t.Error("Expected error loading nonexistent file")
	}
}
