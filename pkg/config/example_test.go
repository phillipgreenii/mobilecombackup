package config_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/phillipgreen/mobilecombackup/pkg/config"
)

// ExampleLoader_LoadWithDefaults demonstrates loading configuration with default precedence
func ExampleLoader_LoadWithDefaults() {
	loader := config.NewConfigLoader()

	// Load configuration using standard precedence:
	// 1. Environment variables (MOBILECOMBACKUP_*)
	// 2. Config files in standard locations
	// 3. Built-in defaults
	cfg, err := loader.LoadWithDefaults()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	fmt.Printf("Repository root: %s\n", cfg.Repository.Root)
	fmt.Printf("Batch size: %d\n", cfg.Import.BatchSize)
	fmt.Printf("Log level: %s\n", cfg.Logging.Level)

	// Output:
	// Repository root: .
	// Batch size: 1000
	// Log level: info
}

// ExampleLoader_Load demonstrates loading configuration from a specific file
func ExampleLoader_Load() {
	loader := config.NewConfigLoader()

	// Create a temporary config file
	tempDir, _ := os.MkdirTemp("", "config-example")
	defer func() { _ = os.RemoveAll(tempDir) }()

	configFile := filepath.Join(tempDir, "mobilecombackup.yaml")
	configContent := `
repository:
  root: "/data/mobile-backup"
import:
  batch_size: 2000
  timeout: "10m"
logging:
  level: "debug"
  format: "json"
`

	_ = os.WriteFile(configFile, []byte(configContent), 0600)

	// Load from specific file
	cfg, err := loader.Load(configFile)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	fmt.Printf("Repository root: %s\n", cfg.Repository.Root)
	fmt.Printf("Batch size: %d\n", cfg.Import.BatchSize)
	fmt.Printf("Log level: %s\n", cfg.Logging.Level)
	fmt.Printf("Log format: %s\n", cfg.Logging.Format)

	// Output:
	// Repository root: /data/mobile-backup
	// Batch size: 2000
	// Log level: debug
	// Log format: json
}

// ExampleGetEnvironmentDefaults demonstrates environment-specific configurations
func ExampleGetEnvironmentDefaults() {
	// Development environment
	devConfig := config.GetEnvironmentDefaults(config.EnvironmentDevelopment)
	fmt.Printf("Development - Level: %s, Color: %t\n",
		devConfig.Logging.Level, devConfig.Logging.Color)

	// Production environment
	prodConfig := config.GetEnvironmentDefaults(config.EnvironmentProduction)
	fmt.Printf("Production - Level: %s, Format: %s, Strict: %t\n",
		prodConfig.Logging.Level, prodConfig.Logging.Format, prodConfig.Validation.Strict)

	// Output:
	// Development - Level: debug, Color: true
	// Production - Level: info, Format: json, Strict: true
}

// ExampleGenerateConfigFile demonstrates generating configuration files from templates
func ExampleGenerateConfigFile() {
	tempDir, _ := os.MkdirTemp("", "config-template")
	defer func() { _ = os.RemoveAll(tempDir) }()

	configPath := filepath.Join(tempDir, "mobilecombackup.yaml")

	// Generate a development configuration
	err := config.GenerateConfigFile("development", configPath)
	if err != nil {
		fmt.Printf("Error generating config: %v\n", err)
		return
	}

	// Don't print the actual path as it's temporary
	fmt.Println("Generated configuration file successfully")

	// Verify the file was created
	if _, err := os.Stat(configPath); err == nil {
		fmt.Println("Configuration file created successfully")
	}

	// Output:
	// Generated configuration file successfully
	// Configuration file created successfully
}

// ExampleGetConfigTemplates demonstrates listing available templates
func ExampleGetConfigTemplates() {
	templates := config.GetConfigTemplates()

	fmt.Println("Available configuration templates:")
	for _, template := range templates {
		fmt.Printf("  %-12s %s\n", template.Name, template.Description)
	}

	// Output:
	// Available configuration templates:
	//   default      Default configuration for general use
	//   development  Development environment configuration with debug logging
	//   production   Production environment configuration with structured logging
	//   minimal      Minimal configuration for simple setups
	//   strict       Strict validation configuration for data integrity
}

// Example_configValidation demonstrates configuration validation
func Example_configValidation() {
	loader := config.NewConfigLoader()

	// Valid configuration
	validConfig := config.DefaultConfig()
	validConfig.Repository.Root = "/valid/path"

	err := loader.Validate(validConfig)
	if err == nil {
		fmt.Println("Valid configuration passed validation")
	}

	// Invalid configuration
	invalidConfig := config.DefaultConfig()
	invalidConfig.Repository.Root = ""  // Empty root is invalid
	invalidConfig.Import.BatchSize = -1 // Negative batch size is invalid

	err = loader.Validate(invalidConfig)
	if err != nil {
		fmt.Printf("Invalid configuration failed validation: %v\n", err)
	}

	// Output:
	// Valid configuration passed validation
	// Invalid configuration failed validation: validation errors: repository.root cannot be empty; import.batch_size must be positive
}

// Example_configEnvironmentVariables demonstrates environment variable usage
func Example_configEnvironmentVariables() {
	// Set environment variables
	_ = os.Setenv("MOBILECOMBACKUP_REPOSITORY_ROOT", "/env/repo")
	_ = os.Setenv("MOBILECOMBACKUP_IMPORT_BATCH_SIZE", "5000")
	_ = os.Setenv("MOBILECOMBACKUP_LOGGING_LEVEL", "debug")

	defer func() {
		_ = os.Unsetenv("MOBILECOMBACKUP_REPOSITORY_ROOT")
		_ = os.Unsetenv("MOBILECOMBACKUP_IMPORT_BATCH_SIZE")
		_ = os.Unsetenv("MOBILECOMBACKUP_LOGGING_LEVEL")
	}()

	loader := config.NewConfigLoader()
	cfg, err := loader.LoadWithDefaults()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	fmt.Printf("Repository root from env: %s\n", cfg.Repository.Root)
	fmt.Printf("Batch size from env: %d\n", cfg.Import.BatchSize)
	fmt.Printf("Log level from env: %s\n", cfg.Logging.Level)

	// Output:
	// Repository root from env: /env/repo
	// Batch size from env: 5000
	// Log level from env: debug
}

// Example_configPrecedence demonstrates configuration precedence
func Example_configPrecedence() {
	// This example shows how configuration precedence works:
	// 1. Environment variables override config files
	// 2. Config files override defaults
	// 3. Defaults are used when nothing else is specified

	tempDir, _ := os.MkdirTemp("", "config-precedence")
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create config file with specific values
	configFile := filepath.Join(tempDir, "mobilecombackup.yaml")
	configContent := `
repository:
  root: "/config/file/repo"
import:
  batch_size: 3000
`
	_ = os.WriteFile(configFile, []byte(configContent), 0600)

	// Set environment variable to override config file
	_ = os.Setenv("MOBILECOMBACKUP_REPOSITORY_ROOT", "/env/override/repo")
	defer func() { _ = os.Unsetenv("MOBILECOMBACKUP_REPOSITORY_ROOT") }()

	loader := config.NewConfigLoader()
	cfg, err := loader.Load(configFile)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	fmt.Printf("Repository root (env override): %s\n", cfg.Repository.Root)
	fmt.Printf("Batch size (from config): %d\n", cfg.Import.BatchSize)
	fmt.Printf("Log level (default): %s\n", cfg.Logging.Level)

	// Output:
	// Repository root (env override): /env/override/repo
	// Batch size (from config): 3000
	// Log level (default): info
}
