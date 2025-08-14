package logging

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name   string
		config *LogConfig
		want   LogLevel
	}{
		{
			name:   "default config",
			config: nil,
			want:   LevelInfo,
		},
		{
			name: "debug config",
			config: &LogConfig{
				Level:  LevelDebug,
				Format: FormatJSON,
			},
			want: LevelDebug,
		},
		{
			name: "error config",
			config: &LogConfig{
				Level:  LevelError,
				Format: FormatConsole,
			},
			want: LevelError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.config)
			if logger == nil {
				t.Errorf("NewLogger() returned nil")
			}

			// Test that logger is functional
			logger.Info().Msg("test message")
		})
	}
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer

	// Create logger with JSON format for easier parsing
	logger := zerolog.New(&buf).Level(zerolog.DebugLevel)
	zeroLogger := &ZeroLogger{logger: logger}

	tests := []struct {
		name     string
		logFunc  func() *zerolog.Event
		expected string
	}{
		{
			name:     "debug level",
			logFunc:  zeroLogger.Debug,
			expected: `"level":"debug"`,
		},
		{
			name:     "info level",
			logFunc:  zeroLogger.Info,
			expected: `"level":"info"`,
		},
		{
			name:     "warn level",
			logFunc:  zeroLogger.Warn,
			expected: `"level":"warn"`,
		},
		{
			name:     "error level",
			logFunc:  zeroLogger.Error,
			expected: `"level":"error"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc().Msg("test message")

			output := buf.String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("Expected log output to contain %s, got: %s", tt.expected, output)
			}
		})
	}
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	zeroLogger := &ZeroLogger{logger: logger}

	// Test WithStr
	strLogger := zeroLogger.WithStr("key", "value")
	strLogger.Info().Msg("test")

	output := buf.String()
	if !strings.Contains(output, `"key":"value"`) {
		t.Errorf("Expected log output to contain key:value, got: %s", output)
	}

	// Test WithInt
	buf.Reset()
	intLogger := zeroLogger.WithInt("count", 42)
	intLogger.Info().Msg("test")

	output = buf.String()
	if !strings.Contains(output, `"count":42`) {
		t.Errorf("Expected log output to contain count:42, got: %s", output)
	}

	// Test WithFields
	buf.Reset()
	fieldsLogger := zeroLogger.WithFields(map[string]interface{}{
		"field1": "value1",
		"field2": 123,
	})
	fieldsLogger.Info().Msg("test")

	output = buf.String()
	if !strings.Contains(output, `"field1":"value1"`) || !strings.Contains(output, `"field2":123`) {
		t.Errorf("Expected log output to contain both fields, got: %s", output)
	}
}

func TestWithComponent(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	zeroLogger := &ZeroLogger{logger: logger}

	componentLogger := zeroLogger.WithComponent("importer")
	componentLogger.Info().Msg("test message")

	output := buf.String()
	if !strings.Contains(output, `"component":"importer"`) {
		t.Errorf("Expected log output to contain component field, got: %s", output)
	}
}

func TestWithOperation(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	zeroLogger := &ZeroLogger{logger: logger}

	opLogger := zeroLogger.WithOperation("import_calls")
	opLogger.Info().Msg("starting operation")

	output := buf.String()
	if !strings.Contains(output, `"operation":"import_calls"`) {
		t.Errorf("Expected log output to contain operation field, got: %s", output)
	}
}

func TestWithContext(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	zeroLogger := &ZeroLogger{logger: logger}

	// Test with context containing request_id
	ctx := context.WithValue(context.Background(), RequestIDKey, "req-123")
	ctxLogger := zeroLogger.WithContext(ctx)
	ctxLogger.Info().Msg("test message")

	output := buf.String()
	if !strings.Contains(output, `"request_id":"req-123"`) {
		t.Errorf("Expected log output to contain request_id from context, got: %s", output)
	}

	// Test with context containing operation_id
	buf.Reset()
	ctx = context.WithValue(context.Background(), OperationIDKey, "op-456")
	ctxLogger = zeroLogger.WithContext(ctx)
	ctxLogger.Info().Msg("test message")

	output = buf.String()
	if !strings.Contains(output, `"operation_id":"op-456"`) {
		t.Errorf("Expected log output to contain operation_id from context, got: %s", output)
	}
}

func TestNullLogger(t *testing.T) {
	// Null logger should not produce any output
	logger := NewNullLogger()

	// These should not panic and should produce no output
	logger.Debug().Msg("debug message")
	logger.Info().Msg("info message")
	logger.Warn().Msg("warn message")
	logger.Error().Msg("error message")
}

func TestContextAwareLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	zeroLogger := &ZeroLogger{logger: logger}
	contextLogger := &ContextAwareLogger{ZeroLogger: zeroLogger}

	// Test WithOperationID
	opLogger := contextLogger.WithOperationID("op-789")
	opLogger.Info().Msg("test operation")

	output := buf.String()
	if !strings.Contains(output, `"operation_id":"op-789"`) {
		t.Errorf("Expected log output to contain operation_id, got: %s", output)
	}

	// Test WithRequestID
	buf.Reset()
	reqLogger := contextLogger.WithRequestID("req-789")
	reqLogger.Info().Msg("test request")

	output = buf.String()
	if !strings.Contains(output, `"request_id":"req-789"`) {
		t.Errorf("Expected log output to contain request_id, got: %s", output)
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    LogLevel
		expected zerolog.Level
	}{
		{LevelDebug, zerolog.DebugLevel},
		{LevelInfo, zerolog.InfoLevel},
		{LevelWarn, zerolog.WarnLevel},
		{LevelError, zerolog.ErrorLevel},
		{LevelOff, zerolog.Disabled},
		{"invalid", zerolog.InfoLevel}, // default
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := parseLogLevel(tt.input)
			if result != tt.expected {
				t.Errorf("parseLogLevel(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDefaultLogConfig(t *testing.T) {
	config := DefaultLogConfig()

	if config.Level != LevelInfo {
		t.Errorf("Expected default level to be info, got %s", config.Level)
	}

	if config.Format != FormatConsole {
		t.Errorf("Expected default format to be console, got %s", config.Format)
	}

	if !config.TimeStamps {
		t.Error("Expected default timestamps to be enabled")
	}

	if !config.Color {
		t.Error("Expected default color to be enabled")
	}
}
