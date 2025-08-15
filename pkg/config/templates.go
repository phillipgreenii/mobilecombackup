package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConfigTemplate holds template information
type ConfigTemplate struct {
	Name        string
	Description string
	Content     *Config
}

// GetConfigTemplates returns available configuration templates
func GetConfigTemplates() []ConfigTemplate {
	return []ConfigTemplate{
		{
			Name:        "default",
			Description: "Default configuration for general use",
			Content:     DefaultConfig(),
		},
		{
			Name:        "development",
			Description: "Development environment configuration with debug logging",
			Content:     GetEnvironmentDefaults(EnvironmentDevelopment),
		},
		{
			Name:        "production",
			Description: "Production environment configuration with structured logging",
			Content:     GetEnvironmentDefaults(EnvironmentProduction),
		},
		{
			Name:        "minimal",
			Description: "Minimal configuration for simple setups",
			Content:     getMinimalConfig(),
		},
		{
			Name:        "strict",
			Description: "Strict validation configuration for data integrity",
			Content:     getStrictConfig(),
		},
	}
}

// getMinimalConfig returns a minimal configuration
func getMinimalConfig() *Config {
	config := &Config{
		Repository: RepositoryConfig{
			Root: ".",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "console",
		},
	}
	return config
}

// getStrictConfig returns a strict validation configuration
func getStrictConfig() *Config {
	config := DefaultConfig()
	config.Validation.Strict = true
	config.Validation.CheckChecksums = true
	config.Validation.AutofixEnabled = false
	config.Validation.MaxErrors = 10
	config.Logging.Level = "debug"
	return config
}

// GenerateConfigFile generates a configuration file from a template
func GenerateConfigFile(templateName string, outputPath string) error {
	templates := GetConfigTemplates()

	var template *ConfigTemplate
	for _, tmpl := range templates {
		if tmpl.Name == templateName {
			template = &tmpl
			break
		}
	}

	if template == nil {
		return fmt.Errorf("template '%s' not found", templateName)
	}

	// Ensure output directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Marshal configuration to YAML
	data, err := yaml.Marshal(template.Content)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Add template header comment
	header := fmt.Sprintf("# Generated configuration file: %s\n# %s\n#\n# For more information, see: https://github.com/phillipgreen/mobilecombackup\n\n",
		template.Name, template.Description)
	content := header + string(data)

	// Write to file
	if err := os.WriteFile(outputPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	return nil
}

// GetConfigExample returns an example configuration with comments
func GetConfigExample() string {
	return `# mobilecombackup Configuration File
# 
# This file demonstrates all available configuration options.
# You can override any of these settings with:
# 1. Command line flags (highest priority)
# 2. Environment variables (prefix: MOBILECOMBACKUP_)
# 3. This configuration file
# 4. Built-in defaults (lowest priority)

# Repository configuration
repository:
  # Root directory for the mobile backup repository
  root: "."
  
  # File and directory permissions
  permissions:
    dir: 0755   # Directory permissions (octal)
    file: 0644  # File permissions (octal)

# Import configuration
import:
  # Number of records to process in each batch
  batch_size: 1000
  
  # Timeout for import operations
  timeout: "5m"
  
  # Number of retry attempts for failed imports
  retry_attempts: 3
  
  # Temporary directory for processing
  temp_dir: "tmp"
  
  # Data filtering options
  filters:
    # Types of data to import (calls, sms)
    allowed_types: ["calls", "sms"]
    
    # Patterns to exclude from import
    exclude_patterns: []
    
    # Date range filtering (optional)
    date_range:
      start: "2020-01-01T00:00:00Z"
      end: "2024-12-31T23:59:59Z"

# Validation configuration
validation:
  # Enable strict validation mode
  strict: false
  
  # Maximum number of validation errors to report
  max_errors: 100
  
  # Verify file checksums during validation
  check_checksums: true
  
  # Validation rules to skip
  skip_rules: []
  
  # Enable automatic fixing of validation issues
  autofix_enabled: true
  
  # Create backups before fixing issues
  backup_on_fix: true

# Logging configuration
logging:
  # Log level: debug, info, warn, error, off
  level: "info"
  
  # Log format: console, json
  format: "console"
  
  # Include timestamps in log output
  timestamps: true
  
  # Use colored output (console format only)
  color: true
  
  # Optional: write logs to file (empty = stdout/stderr only)
  file: ""

# Environment-specific settings
# You can create separate config files for different environments:
# - mobilecombackup-development.yaml
# - mobilecombackup-test.yaml  
# - mobilecombackup-production.yaml
`
}

// ListAvailableTemplates returns a formatted list of available templates
func ListAvailableTemplates() string {
	templates := GetConfigTemplates()
	result := "Available configuration templates:\n\n"

	for _, template := range templates {
		result += fmt.Sprintf("  %-12s %s\n", template.Name, template.Description)
	}

	result += "\nTo generate a configuration file:\n"
	result += "  mobilecombackup config init --template <name> [--output <path>]\n"

	return result
}
