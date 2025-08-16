package logging

import (
	"context"

	"github.com/rs/zerolog"
)

// Logger provides structured logging interface
type Logger interface {
	// Level-based logging methods
	Debug() *zerolog.Event
	Info() *zerolog.Event
	Warn() *zerolog.Event
	Error() *zerolog.Event

	// Context methods for adding structured data
	With() *zerolog.Context
	WithStr(key, val string) Logger
	WithInt(key string, val int) Logger
	WithError(err error) Logger
	WithDuration(key string, val interface{}) Logger

	// Create a child logger with additional context
	WithContext(ctx context.Context) Logger
	WithFields(fields map[string]interface{}) Logger
	WithComponent(component string) Logger
	WithOperation(operation string) Logger
}

// ContextLogger extends Logger with context-aware functionality
type ContextLogger interface {
	Logger
	WithOperationID(operationID string) Logger
	WithRequestID(requestID string) Logger
}

// LogLevel represents the logging levels
type LogLevel string

const (
	// LevelDebug enables debug-level logging
	LevelDebug LogLevel = "debug"
	// LevelInfo enables info-level logging
	LevelInfo LogLevel = "info"
	// LevelWarn enables warn-level logging
	LevelWarn LogLevel = "warn"
	// LevelError enables error-level logging
	LevelError LogLevel = "error"
	// LevelOff disables all logging
	LevelOff LogLevel = "off"
)

// LogFormat represents the output format for logs
type LogFormat string

const (
	// FormatConsole outputs human-readable console logs
	FormatConsole LogFormat = "console"
	// FormatJSON outputs structured JSON logs
	FormatJSON LogFormat = "json"
)

// LogConfig holds logging configuration
type LogConfig struct {
	Level      LogLevel  `yaml:"level" json:"level"`
	Format     LogFormat `yaml:"format" json:"format"`
	TimeStamps bool      `yaml:"timestamps" json:"timestamps"`
	Color      bool      `yaml:"color" json:"color"`
}

// ContextKey represents a context key type for proper type safety
type ContextKey string

const (
	// RequestIDKey is the context key used for request ID tracking
	RequestIDKey ContextKey = "request_id"
	// OperationIDKey is the context key used for operation ID tracking
	OperationIDKey ContextKey = "operation_id"
)

// DefaultLogConfig returns the default logging configuration
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level:      LevelInfo,
		Format:     FormatConsole,
		TimeStamps: true,
		Color:      true,
	}
}
