package config

import (
	"os"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/logging"
)

// Config represents the complete application configuration
type Config struct {
	Repository RepositoryConfig `yaml:"repository" mapstructure:"repository"`
	Import     ImportConfig     `yaml:"import" mapstructure:"import"`
	Validation ValidationConfig `yaml:"validation" mapstructure:"validation"`
	Logging    LoggingConfig    `yaml:"logging" mapstructure:"logging"`
}

// RepositoryConfig holds repository-related configuration
type RepositoryConfig struct {
	Root        string      `yaml:"root" mapstructure:"root"`
	Permissions Permissions `yaml:"permissions" mapstructure:"permissions"`
}

// Permissions holds file and directory permission settings
type Permissions struct {
	Dir  os.FileMode `yaml:"dir" mapstructure:"dir"`
	File os.FileMode `yaml:"file" mapstructure:"file"`
}

// ImportConfig holds import-related configuration
type ImportConfig struct {
	BatchSize     int           `yaml:"batch_size" mapstructure:"batch_size"`
	Timeout       time.Duration `yaml:"timeout" mapstructure:"timeout"`
	RetryAttempts int           `yaml:"retry_attempts" mapstructure:"retry_attempts"`
	TempDir       string        `yaml:"temp_dir" mapstructure:"temp_dir"`
	Filters       FilterConfig  `yaml:"filters" mapstructure:"filters"`
}

// FilterConfig holds data filtering configuration
type FilterConfig struct {
	AllowedTypes    []string `yaml:"allowed_types" mapstructure:"allowed_types"`
	ExcludePatterns []string `yaml:"exclude_patterns" mapstructure:"exclude_patterns"`
	DateRange       struct {
		Start time.Time `yaml:"start" mapstructure:"start"`
		End   time.Time `yaml:"end" mapstructure:"end"`
	} `yaml:"date_range" mapstructure:"date_range"`
}

// ValidationConfig holds validation-related configuration
type ValidationConfig struct {
	Strict         bool     `yaml:"strict" mapstructure:"strict"`
	MaxErrors      int      `yaml:"max_errors" mapstructure:"max_errors"`
	CheckChecksums bool     `yaml:"check_checksums" mapstructure:"check_checksums"`
	SkipRules      []string `yaml:"skip_rules" mapstructure:"skip_rules"`
	AutofixEnabled bool     `yaml:"autofix_enabled" mapstructure:"autofix_enabled"`
	BackupOnFix    bool     `yaml:"backup_on_fix" mapstructure:"backup_on_fix"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      logging.LogLevel  `yaml:"level" mapstructure:"level"`
	Format     logging.LogFormat `yaml:"format" mapstructure:"format"`
	TimeStamps bool              `yaml:"timestamps" mapstructure:"timestamps"`
	Color      bool              `yaml:"color" mapstructure:"color"`
	File       string            `yaml:"file" mapstructure:"file"` // Optional log file path
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Repository: RepositoryConfig{
			Root: ".",
			Permissions: Permissions{
				Dir:  0755,
				File: 0644,
			},
		},
		Import: ImportConfig{
			BatchSize:     1000,
			Timeout:       5 * time.Minute,
			RetryAttempts: 3,
			TempDir:       "tmp",
			Filters: FilterConfig{
				AllowedTypes:    []string{"calls", "sms"},
				ExcludePatterns: []string{},
			},
		},
		Validation: ValidationConfig{
			Strict:         false,
			MaxErrors:      100,
			CheckChecksums: true,
			SkipRules:      []string{},
			AutofixEnabled: true,
			BackupOnFix:    true,
		},
		Logging: LoggingConfig{
			Level:      logging.LevelInfo,
			Format:     logging.FormatConsole,
			TimeStamps: true,
			Color:      true,
			File:       "", // Empty means stdout/stderr only
		},
	}
}

// Environment represents different deployment environments
type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentTest        Environment = "test"
	EnvironmentProduction  Environment = "production"
)

// EnvConfig holds environment-specific configuration
type EnvConfig struct {
	Environment Environment `yaml:"environment" mapstructure:"environment"`
	Debug       bool        `yaml:"debug" mapstructure:"debug"`
}

// GetEnvironmentDefaults returns configuration defaults for a specific environment
func GetEnvironmentDefaults(env Environment) *Config {
	config := DefaultConfig()

	switch env {
	case EnvironmentDevelopment:
		config.Logging.Level = logging.LevelDebug
		config.Logging.Color = true
		config.Validation.Strict = false

	case EnvironmentTest:
		config.Logging.Level = logging.LevelWarn
		config.Logging.Color = false
		config.Logging.Format = logging.FormatJSON
		config.Validation.Strict = true

	case EnvironmentProduction:
		config.Logging.Level = logging.LevelInfo
		config.Logging.Color = false
		config.Logging.Format = logging.FormatJSON
		config.Validation.Strict = true
		config.Validation.MaxErrors = 50
	}

	return config
}
