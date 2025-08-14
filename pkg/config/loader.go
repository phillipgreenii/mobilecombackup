package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// ConfigLoader interface for loading configuration
type ConfigLoader interface {
	Load(path string) (*Config, error)
	LoadWithDefaults() (*Config, error)
	LoadFromEnvironment(env Environment) (*Config, error)
	Validate(config *Config) error
}

// ViperConfigLoader implements ConfigLoader using Viper
type ViperConfigLoader struct {
	logger interface{} // Will be logging.Logger but using interface{} to avoid circular dependencies
}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader() ConfigLoader {
	return &ViperConfigLoader{}
}

// NewConfigLoaderWithLogger creates a new configuration loader with a logger
func NewConfigLoaderWithLogger(logger interface{}) ConfigLoader {
	return &ViperConfigLoader{
		logger: logger,
	}
}

// Load loads configuration from a specific file
func (l *ViperConfigLoader) Load(path string) (*Config, error) {
	v := viper.New()

	// Set up Viper
	l.setupViper(v)

	// Load from file
	if path != "" {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
		}
	}

	return l.parseConfig(v)
}

// LoadWithDefaults loads configuration using the standard precedence:
// 1. Command line flags (handled by caller)
// 2. Environment variables
// 3. Configuration files (in standard locations)
// 4. Defaults
func (l *ViperConfigLoader) LoadWithDefaults() (*Config, error) {
	v := viper.New()

	// Set up Viper
	l.setupViper(v)

	// Set config name and paths for automatic discovery
	v.SetConfigName("mobilecombackup")
	v.SetConfigType("yaml")

	// Add config paths following XDG Base Directory Specification
	v.AddConfigPath(".") // Current directory
	v.AddConfigPath("$HOME/.config/mobilecombackup")
	v.AddConfigPath("/etc/mobilecombackup")

	// Try to read config file (it's ok if it doesn't exist)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	return l.parseConfig(v)
}

// LoadFromEnvironment loads configuration with environment-specific defaults
func (l *ViperConfigLoader) LoadFromEnvironment(env Environment) (*Config, error) {
	// Start with environment defaults
	defaults := GetEnvironmentDefaults(env)

	v := viper.New()
	l.setupViper(v)

	// Set defaults from environment configuration
	l.setViperDefaults(v, defaults)

	// Set environment-specific config file name
	v.SetConfigName(fmt.Sprintf("mobilecombackup-%s", env))
	v.SetConfigType("yaml")

	// Add config paths
	v.AddConfigPath(".")
	v.AddConfigPath("$HOME/.config/mobilecombackup")
	v.AddConfigPath("/etc/mobilecombackup")

	// Try to read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	return l.parseConfig(v)
}

// setupViper configures Viper with common settings
func (l *ViperConfigLoader) setupViper(v *viper.Viper) {
	// Set environment variable prefix
	v.SetEnvPrefix("MOBILECOMBACKUP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	// Set default values
	defaults := DefaultConfig()
	l.setViperDefaults(v, defaults)
}

// setViperDefaults sets default values in Viper from a Config struct
func (l *ViperConfigLoader) setViperDefaults(v *viper.Viper, config *Config) {
	// Repository defaults
	v.SetDefault("repository.root", config.Repository.Root)
	v.SetDefault("repository.permissions.dir", config.Repository.Permissions.Dir)
	v.SetDefault("repository.permissions.file", config.Repository.Permissions.File)

	// Import defaults
	v.SetDefault("import.batch_size", config.Import.BatchSize)
	v.SetDefault("import.timeout", config.Import.Timeout)
	v.SetDefault("import.retry_attempts", config.Import.RetryAttempts)
	v.SetDefault("import.temp_dir", config.Import.TempDir)
	v.SetDefault("import.filters.allowed_types", config.Import.Filters.AllowedTypes)
	v.SetDefault("import.filters.exclude_patterns", config.Import.Filters.ExcludePatterns)

	// Validation defaults
	v.SetDefault("validation.strict", config.Validation.Strict)
	v.SetDefault("validation.max_errors", config.Validation.MaxErrors)
	v.SetDefault("validation.check_checksums", config.Validation.CheckChecksums)
	v.SetDefault("validation.skip_rules", config.Validation.SkipRules)
	v.SetDefault("validation.autofix_enabled", config.Validation.AutofixEnabled)
	v.SetDefault("validation.backup_on_fix", config.Validation.BackupOnFix)

	// Logging defaults
	v.SetDefault("logging.level", config.Logging.Level)
	v.SetDefault("logging.format", config.Logging.Format)
	v.SetDefault("logging.timestamps", config.Logging.TimeStamps)
	v.SetDefault("logging.color", config.Logging.Color)
	v.SetDefault("logging.file", config.Logging.File)
}

// parseConfig parses Viper configuration into Config struct
func (l *ViperConfigLoader) parseConfig(v *viper.Viper) (*Config, error) {
	config := &Config{}

	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// Validate the configuration
	if err := l.Validate(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// Validate validates a configuration
func (l *ViperConfigLoader) Validate(config *Config) error {
	var errors []string

	// Validate repository configuration
	if config.Repository.Root == "" {
		errors = append(errors, "repository.root cannot be empty")
	}

	// Validate permissions
	if config.Repository.Permissions.Dir == 0 {
		errors = append(errors, "repository.permissions.dir must be set")
	}
	if config.Repository.Permissions.File == 0 {
		errors = append(errors, "repository.permissions.file must be set")
	}

	// Validate import configuration
	if config.Import.BatchSize <= 0 {
		errors = append(errors, "import.batch_size must be positive")
	}
	if config.Import.RetryAttempts < 0 {
		errors = append(errors, "import.retry_attempts cannot be negative")
	}
	if config.Import.Timeout <= 0 {
		errors = append(errors, "import.timeout must be positive")
	}

	// Validate validation configuration
	if config.Validation.MaxErrors < 0 {
		errors = append(errors, "validation.max_errors cannot be negative")
	}

	// Validate allowed types
	for _, allowedType := range config.Import.Filters.AllowedTypes {
		if allowedType != "calls" && allowedType != "sms" {
			errors = append(errors, fmt.Sprintf("invalid allowed type: %s", allowedType))
		}
	}

	// Validate logging configuration
	validLevels := []string{"debug", "info", "warn", "error", "off"}
	levelValid := false
	for _, level := range validLevels {
		if string(config.Logging.Level) == level {
			levelValid = true
			break
		}
	}
	if !levelValid {
		errors = append(errors, fmt.Sprintf("invalid logging level: %s", config.Logging.Level))
	}

	validFormats := []string{"console", "json"}
	formatValid := false
	for _, format := range validFormats {
		if string(config.Logging.Format) == format {
			formatValid = true
			break
		}
	}
	if !formatValid {
		errors = append(errors, fmt.Sprintf("invalid logging format: %s", config.Logging.Format))
	}

	// Validate log file path if specified
	if config.Logging.File != "" {
		dir := filepath.Dir(config.Logging.File)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			errors = append(errors, fmt.Sprintf("log file directory does not exist: %s", dir))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// GetConfigFiles returns a list of configuration files that would be checked
func GetConfigFiles() []string {
	var files []string

	// Current directory
	files = append(files, "./mobilecombackup.yaml")
	files = append(files, "./mobilecombackup.yml")

	// User config directory
	if home, err := os.UserHomeDir(); err == nil {
		userConfig := filepath.Join(home, ".config", "mobilecombackup")
		files = append(files, filepath.Join(userConfig, "mobilecombackup.yaml"))
		files = append(files, filepath.Join(userConfig, "mobilecombackup.yml"))
	}

	// System config directory
	files = append(files, "/etc/mobilecombackup/mobilecombackup.yaml")
	files = append(files, "/etc/mobilecombackup/mobilecombackup.yml")

	return files
}
