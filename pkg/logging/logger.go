package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// ZeroLogger implements the Logger interface using zerolog
type ZeroLogger struct {
	logger zerolog.Logger
}

// NewLogger creates a new logger with the given configuration
func NewLogger(config *LogConfig) Logger {
	if config == nil {
		config = DefaultLogConfig()
	}

	// Configure output writer
	var output io.Writer = os.Stdout
	if config.Format == FormatConsole {
		if config.Color {
			output = zerolog.ConsoleWriter{
				Out:        os.Stdout,
				TimeFormat: "15:04:05",
				NoColor:    false,
			}
		} else {
			output = zerolog.ConsoleWriter{
				Out:        os.Stdout,
				TimeFormat: "15:04:05",
				NoColor:    true,
			}
		}
	}

	// Create base logger
	logger := zerolog.New(output)

	// Configure timestamps
	if config.TimeStamps {
		logger = logger.With().Timestamp().Logger()
	}

	// Set log level
	level := parseLogLevel(config.Level)
	logger = logger.Level(level)

	return &ZeroLogger{
		logger: logger,
	}
}

// NewNullLogger creates a logger that discards all output (for testing)
func NewNullLogger() Logger {
	logger := zerolog.New(io.Discard).Level(zerolog.Disabled)
	return &ZeroLogger{
		logger: logger,
	}
}

// Debug returns a debug level event
func (l *ZeroLogger) Debug() *zerolog.Event {
	return l.logger.Debug()
}

// Info returns an info level event
func (l *ZeroLogger) Info() *zerolog.Event {
	return l.logger.Info()
}

// Warn returns a warn level event
func (l *ZeroLogger) Warn() *zerolog.Event {
	return l.logger.Warn()
}

// Error returns an error level event
func (l *ZeroLogger) Error() *zerolog.Event {
	return l.logger.Error()
}

// With returns a context for adding structured data
func (l *ZeroLogger) With() *zerolog.Context {
	ctx := l.logger.With()
	return &ctx
}

// WithStr adds a string field and returns a new logger
func (l *ZeroLogger) WithStr(key, val string) Logger {
	return &ZeroLogger{
		logger: l.logger.With().Str(key, val).Logger(),
	}
}

// WithInt adds an integer field and returns a new logger
func (l *ZeroLogger) WithInt(key string, val int) Logger {
	return &ZeroLogger{
		logger: l.logger.With().Int(key, val).Logger(),
	}
}

// WithError adds an error field and returns a new logger
func (l *ZeroLogger) WithError(err error) Logger {
	return &ZeroLogger{
		logger: l.logger.With().Err(err).Logger(),
	}
}

// WithDuration adds a duration field and returns a new logger
func (l *ZeroLogger) WithDuration(key string, val interface{}) Logger {
	if dur, ok := val.(time.Duration); ok {
		return &ZeroLogger{
			logger: l.logger.With().Dur(key, dur).Logger(),
		}
	}
	// Fallback to string representation if not a duration
	return &ZeroLogger{
		logger: l.logger.With().Str(key, fmt.Sprintf("%v", val)).Logger(),
	}
}

// WithContext creates a logger from context (extracts request ID, operation ID, etc.)
func (l *ZeroLogger) WithContext(ctx context.Context) Logger {
	logger := l.logger

	// Extract common context values if they exist
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		if id, ok := requestID.(string); ok {
			logger = logger.With().Str("request_id", id).Logger()
		}
	}

	if operationID := ctx.Value(OperationIDKey); operationID != nil {
		if id, ok := operationID.(string); ok {
			logger = logger.With().Str("operation_id", id).Logger()
		}
	}

	return &ZeroLogger{
		logger: logger,
	}
}

// WithFields adds multiple fields and returns a new logger
func (l *ZeroLogger) WithFields(fields map[string]interface{}) Logger {
	ctx := l.logger.With()
	for key, value := range fields {
		ctx = ctx.Interface(key, value)
	}
	return &ZeroLogger{
		logger: ctx.Logger(),
	}
}

// WithComponent adds a component field for identifying which part of the system is logging
func (l *ZeroLogger) WithComponent(component string) Logger {
	return &ZeroLogger{
		logger: l.logger.With().Str("component", component).Logger(),
	}
}

// WithOperation adds an operation field for tracking what operation is being performed
func (l *ZeroLogger) WithOperation(operation string) Logger {
	return &ZeroLogger{
		logger: l.logger.With().Str("operation", operation).Logger(),
	}
}

// parseLogLevel converts our LogLevel to zerolog.Level
func parseLogLevel(level LogLevel) zerolog.Level {
	switch strings.ToLower(string(level)) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "off", "disabled":
		return zerolog.Disabled
	default:
		return zerolog.InfoLevel
	}
}

// ContextAwareLogger extends ZeroLogger with context-specific functionality
type ContextAwareLogger struct {
	*ZeroLogger
}

// NewContextAwareLogger creates a context-aware logger
func NewContextAwareLogger(config *LogConfig) ContextLogger {
	base := NewLogger(config).(*ZeroLogger)
	return &ContextAwareLogger{
		ZeroLogger: base,
	}
}

// WithOperationID adds an operation ID for tracking operations across the system
func (l *ContextAwareLogger) WithOperationID(operationID string) Logger {
	return &ContextAwareLogger{
		ZeroLogger: l.WithStr("operation_id", operationID).(*ZeroLogger),
	}
}

// WithRequestID adds a request ID for tracking requests across the system
func (l *ContextAwareLogger) WithRequestID(requestID string) Logger {
	return &ContextAwareLogger{
		ZeroLogger: l.WithStr("request_id", requestID).(*ZeroLogger),
	}
}
