package errors

import (
	"fmt"
)

// ErrorCode represents a category of error for programmatic handling
type ErrorCode string

const (
	ErrCodeValidation    ErrorCode = "VALIDATION_ERROR"
	ErrCodeFileNotFound  ErrorCode = "FILE_NOT_FOUND"
	ErrCodeParsing       ErrorCode = "PARSE_ERROR"
	ErrCodePermission    ErrorCode = "PERMISSION_ERROR"
	ErrCodeStorage       ErrorCode = "STORAGE_ERROR"
	ErrCodeIntegrity     ErrorCode = "INTEGRITY_ERROR"
	ErrCodeProcessing    ErrorCode = "PROCESSING_ERROR"
	ErrCodeImport        ErrorCode = "IMPORT_ERROR"
	ErrCodeConfiguration ErrorCode = "CONFIG_ERROR"
)

// ValidationError represents an error that occurred during validation
// with context about the location and operation being performed
type ValidationError struct {
	File      string    // Source file where error occurred
	Line      int       // Line number in source file
	Operation string    // Operation being performed when error occurred
	Code      ErrorCode // Error category code
	Err       error     // Underlying error
}

func (e *ValidationError) Error() string {
	if e.File != "" && e.Line != 0 {
		return fmt.Sprintf("%s:%d: %s failed: %v", e.File, e.Line, e.Operation, e.Err)
	}
	return fmt.Sprintf("%s failed: %v", e.Operation, e.Err)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

func (e *ValidationError) ErrorCode() ErrorCode {
	return e.Code
}

// ProcessingError represents an error that occurred during data processing
// with context about the stage and input file being processed
type ProcessingError struct {
	Stage     string    // Processing stage where error occurred
	InputFile string    // Input file being processed
	Code      ErrorCode // Error category code
	Err       error     // Underlying error
}

func (e *ProcessingError) Error() string {
	if e.InputFile != "" {
		return fmt.Sprintf("processing failed at stage '%s' for file '%s': %v", e.Stage, e.InputFile, e.Err)
	}
	return fmt.Sprintf("processing failed at stage '%s': %v", e.Stage, e.Err)
}

func (e *ProcessingError) Unwrap() error {
	return e.Err
}

func (e *ProcessingError) ErrorCode() ErrorCode {
	return e.Code
}

// ImportError represents an error that occurred during import operations
// with context about the import phase and entity being processed
type ImportError struct {
	Phase  string    // Import phase where error occurred (e.g., "scanning", "processing", "validation")
	Entity string    // Entity being imported (e.g., "calls", "sms", "attachments")
	Count  int       // Number of entities processed before error
	Code   ErrorCode // Error category code
	Err    error     // Underlying error
}

func (e *ImportError) Error() string {
	if e.Count > 0 {
		return fmt.Sprintf("import failed during '%s' phase for %s after processing %d items: %v", e.Phase, e.Entity, e.Count, e.Err)
	}
	return fmt.Sprintf("import failed during '%s' phase for %s: %v", e.Phase, e.Entity, e.Err)
}

func (e *ImportError) Unwrap() error {
	return e.Err
}

func (e *ImportError) ErrorCode() ErrorCode {
	return e.Code
}

// FileError represents an error related to file operations
// with context about the file path and operation being performed
type FileError struct {
	Path      string    // File path that caused the error
	Operation string    // File operation being performed (e.g., "read", "write", "delete")
	Code      ErrorCode // Error category code
	Err       error     // Underlying error
}

func (e *FileError) Error() string {
	return fmt.Sprintf("file %s failed for '%s': %v", e.Operation, e.Path, e.Err)
}

func (e *FileError) Unwrap() error {
	return e.Err
}

func (e *FileError) ErrorCode() ErrorCode {
	return e.Code
}

// ConfigurationError represents an error related to configuration
// with context about the configuration key and value
type ConfigurationError struct {
	Key   string    // Configuration key that caused the error
	Value string    // Configuration value that caused the error
	Code  ErrorCode // Error category code
	Err   error     // Underlying error
}

func (e *ConfigurationError) Error() string {
	if e.Value != "" {
		return fmt.Sprintf("configuration error for key '%s' with value '%s': %v", e.Key, e.Value, e.Err)
	}
	return fmt.Sprintf("configuration error for key '%s': %v", e.Key, e.Err)
}

func (e *ConfigurationError) Unwrap() error {
	return e.Err
}

func (e *ConfigurationError) ErrorCode() ErrorCode {
	return e.Code
}

// ErrorCoder interface for errors that have error codes
type ErrorCoder interface {
	ErrorCode() ErrorCode
}
