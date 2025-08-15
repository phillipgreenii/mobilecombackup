package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	// Test constants
	debugLevel = "debug"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// Test repository defaults
	if config.Repository.Root != "." {
		t.Errorf("Expected repository root to be '.', got %s", config.Repository.Root)
	}

	if config.Repository.Permissions.Dir != 0755 {
		t.Errorf("Expected directory permissions to be 0755, got %o", config.Repository.Permissions.Dir)
	}

	// Test import defaults
	if config.Import.BatchSize != 1000 {
		t.Errorf("Expected batch size to be 1000, got %d", config.Import.BatchSize)
	}

	if config.Import.Timeout != 5*time.Minute {
		t.Errorf("Expected timeout to be 5m, got %v", config.Import.Timeout)
	}

	// Test validation defaults
	if config.Validation.MaxErrors != 100 {
		t.Errorf("Expected max errors to be 100, got %d", config.Validation.MaxErrors)
	}

	if !config.Validation.AutofixEnabled {
		t.Error("Expected autofix to be enabled by default")
	}
}

func TestGetEnvironmentDefaults(t *testing.T) {
	tests := []struct {
		env      Environment
		expected map[string]interface{}
	}{
		{
			env: EnvironmentDevelopment,
			expected: map[string]interface{}{
				"debug_level": true,
				"color":       true,
				"strict":      false,
			},
		},
		{
			env: EnvironmentProduction,
			expected: map[string]interface{}{
				"json_format": true,
				"no_color":    true,
				"strict":      true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.env), func(t *testing.T) {
			config := GetEnvironmentDefaults(tt.env)

			switch tt.env {
			case EnvironmentDevelopment:
				if config.Logging.Level != debugLevel {
					t.Errorf("Expected debug level for development, got %s", config.Logging.Level)
				}
				if !config.Logging.Color {
					t.Error("Expected color enabled for development")
				}
				if config.Validation.Strict {
					t.Error("Expected strict validation disabled for development")
				}

			case EnvironmentProduction:
				if config.Logging.Format != "json" {
					t.Errorf("Expected JSON format for production, got %s", config.Logging.Format)
				}
				if config.Logging.Color {
					t.Error("Expected color disabled for production")
				}
				if !config.Validation.Strict {
					t.Error("Expected strict validation enabled for production")
				}
			}
		})
	}
}

func TestViperConfigLoader_LoadWithDefaults(t *testing.T) {
	loader := NewConfigLoader()

	config, err := loader.LoadWithDefaults()
	if err != nil {
		t.Fatalf("LoadWithDefaults failed: %v", err)
	}

	if config == nil {
		t.Fatal("LoadWithDefaults returned nil config")
	}

	// Should return default values when no config file exists
	if config.Repository.Root != "." {
		t.Errorf("Expected default repository root, got %s", config.Repository.Root)
	}
}

func TestViperConfigLoader_LoadFromFile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yaml")

	configContent := `
repository:
  root: "/tmp/test-repo"
  permissions:
    dir: 0750
    file: 0640

import:
  batch_size: 500
  timeout: "10m"
  retry_attempts: 5

validation:
  strict: true
  max_errors: 50
  autofix_enabled: false

logging:
  level: debugLevel
  format: "json"
  timestamps: false
  color: false
`

	err := os.WriteFile(configFile, []byte(configContent), 0600)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	loader := NewConfigLoader()
	config, err := loader.Load(configFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify loaded values
	if config.Repository.Root != "/tmp/test-repo" {
		t.Errorf("Expected repository root '/tmp/test-repo', got %s", config.Repository.Root)
	}

	if config.Repository.Permissions.Dir != 0750 {
		t.Errorf("Expected directory permissions 0750, got %o", config.Repository.Permissions.Dir)
	}

	if config.Import.BatchSize != 500 {
		t.Errorf("Expected batch size 500, got %d", config.Import.BatchSize)
	}

	if config.Import.Timeout != 10*time.Minute {
		t.Errorf("Expected timeout 10m, got %v", config.Import.Timeout)
	}

	if config.Validation.Strict != true {
		t.Error("Expected strict validation to be true")
	}

	if config.Validation.AutofixEnabled != false {
		t.Error("Expected autofix to be disabled")
	}

	if config.Logging.Level != debugLevel {
		t.Errorf("Expected debug level, got %s", config.Logging.Level)
	}

	if config.Logging.Format != "json" {
		t.Errorf("Expected JSON format, got %s", config.Logging.Format)
	}
}

func TestViperConfigLoader_Validate(t *testing.T) {
	loader := NewConfigLoader()

	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid config",
			config:      DefaultConfig(),
			expectError: false,
		},
		{
			name: "empty repository root",
			config: &Config{
				Repository: RepositoryConfig{Root: ""},
				Import:     DefaultConfig().Import,
				Validation: DefaultConfig().Validation,
				Logging:    DefaultConfig().Logging,
			},
			expectError: true,
			errorMsg:    "repository.root cannot be empty",
		},
		{
			name: "invalid batch size",
			config: &Config{
				Repository: DefaultConfig().Repository,
				Import: ImportConfig{
					BatchSize:     -1,
					Timeout:       5 * time.Minute,
					RetryAttempts: 3,
					TempDir:       "tmp",
				},
				Validation: DefaultConfig().Validation,
				Logging:    DefaultConfig().Logging,
			},
			expectError: true,
			errorMsg:    "import.batch_size must be positive",
		},
		{
			name: "invalid logging level",
			config: &Config{
				Repository: DefaultConfig().Repository,
				Import:     DefaultConfig().Import,
				Validation: DefaultConfig().Validation,
				Logging: LoggingConfig{
					Level:      "invalid",
					Format:     "console",
					TimeStamps: true,
					Color:      true,
				},
			},
			expectError: true,
			errorMsg:    "invalid logging level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loader.Validate(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected validation error, got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no validation error, got: %v", err)
				}
			}
		})
	}
}

func TestViperConfigLoader_LoadFromEnvironment(t *testing.T) {
	loader := NewConfigLoader()

	config, err := loader.LoadFromEnvironment(EnvironmentDevelopment)
	if err != nil {
		t.Fatalf("LoadFromEnvironment failed: %v", err)
	}

	// Should have development-specific defaults
	if config.Logging.Level != debugLevel {
		t.Errorf("Expected debug level for development environment, got %s", config.Logging.Level)
	}

	if !config.Logging.Color {
		t.Error("Expected color enabled for development environment")
	}
}

func TestGetConfigFiles(t *testing.T) {
	files := GetConfigFiles()

	if len(files) == 0 {
		t.Error("GetConfigFiles returned no files")
	}

	// Should include current directory
	found := false
	for _, file := range files {
		if strings.Contains(file, "./mobilecombackup.yaml") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected current directory config file in list")
	}

	// Should include system directory
	found = false
	for _, file := range files {
		if strings.Contains(file, "/etc/mobilecombackup") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected system config directory in list")
	}
}

func TestViperConfigLoader_LoadNonExistentFile(t *testing.T) {
	loader := NewConfigLoader()

	_, err := loader.Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error when loading nonexistent file")
	}
}

func TestViperConfigLoader_LoadInvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "invalid-config.yaml")

	// Write invalid YAML
	invalidYAML := `
repository:
  root: "/tmp/test"
  invalid: [ unclosed bracket
`

	err := os.WriteFile(configFile, []byte(invalidYAML), 0600)
	if err != nil {
		t.Fatalf("Failed to write invalid config file: %v", err)
	}

	loader := NewConfigLoader()
	_, err = loader.Load(configFile)
	if err == nil {
		t.Error("Expected error when loading invalid YAML")
	}
}
