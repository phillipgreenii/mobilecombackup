package logging_test

import (
	"context"
	"errors"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/logging"
)

// ExampleLogger demonstrates basic logging usage
func ExampleLogger() {
	// Create a logger with default configuration
	logger := logging.NewLogger(nil)

	// Basic logging
	logger.Info().Msg("Application starting")
	logger.Debug().Str("component", "main").Msg("Debug information")
	logger.Warn().Int("attempts", 3).Msg("Retry attempt")
	logger.Error().Err(errors.New("connection failed")).Msg("Failed to connect")

	// Structured logging with fields
	logger.Info().
		Str("user", "john").
		Int("age", 30).
		Bool("admin", false).
		Msg("User logged in")
}

// ExampleLogger_withComponent shows component-specific logging
func ExampleLogger_withComponent() {
	logger := logging.NewLogger(nil)

	// Create component-specific loggers
	importerLogger := logger.WithComponent("importer")
	validatorLogger := logger.WithComponent("validator")

	importerLogger.Info().
		Str("file", "calls.xml").
		Int("records", 1500).
		Msg("Processing import file")

	validatorLogger.Warn().
		Str("rule", "count_mismatch").
		Str("file", "calls-2024.xml").
		Msg("Validation warning")
}

// ExampleLogger_withOperation demonstrates operation tracking
func ExampleLogger_withOperation() {
	logger := logging.NewLogger(nil)

	// Track a specific operation
	opLogger := logger.WithOperation("import_sms")

	opLogger.Info().Msg("Starting SMS import")

	// Add more context as operation progresses
	opLogger.Info().
		Str("file", "sms-2024.xml").
		Int("size_mb", 45).
		Msg("Processing file")

	opLogger.Info().
		Int("processed", 2500).
		Dur("elapsed", 30*time.Second).
		Msg("Import completed")
}

// ExampleLogger_withContext shows context-aware logging
func ExampleLogger_withContext() {
	logger := logging.NewLogger(nil)

	// Create context with request tracking
	ctx := context.WithValue(context.Background(), logging.RequestIDKey, "req-12345")
	ctx = context.WithValue(ctx, logging.OperationIDKey, "op-67890")

	// Logger will automatically extract context values
	ctxLogger := logger.WithContext(ctx)

	ctxLogger.Info().
		Str("action", "validate_repository").
		Msg("Starting validation")
}

// ExampleLogger_configuration demonstrates different logging configurations
func ExampleLogger_configuration() {
	// Console format with colors (development)
	devConfig := &logging.LogConfig{
		Level:      logging.LevelDebug,
		Format:     logging.FormatConsole,
		TimeStamps: true,
		Color:      true,
	}
	devLogger := logging.NewLogger(devConfig)
	devLogger.Debug().Msg("Development environment")

	// JSON format (production)
	prodConfig := &logging.LogConfig{
		Level:      logging.LevelInfo,
		Format:     logging.FormatJSON,
		TimeStamps: true,
		Color:      false,
	}
	prodLogger := logging.NewLogger(prodConfig)
	prodLogger.Info().
		Str("environment", "production").
		Str("version", "1.0.0").
		Msg("Application started")

	// Quiet mode (minimal logging)
	quietConfig := &logging.LogConfig{
		Level:      logging.LevelError,
		Format:     logging.FormatConsole,
		TimeStamps: false,
		Color:      false,
	}
	quietLogger := logging.NewLogger(quietConfig)
	quietLogger.Error().Msg("Critical error only")
}

// ExampleContextLogger demonstrates advanced context-aware features
func ExampleContextLogger() {
	contextLogger := logging.NewContextAwareLogger(nil)

	// Track operations across system boundaries
	opLogger := contextLogger.WithOperationID("import-batch-001")

	opLogger.Info().
		Str("batch_id", "batch-001").
		Int("file_count", 5).
		Msg("Starting batch import")

	// Track requests in web/API context
	reqLogger := contextLogger.WithRequestID("api-req-456")

	reqLogger.Info().
		Str("endpoint", "/api/v1/import").
		Str("method", "POST").
		Msg("API request received")
}
