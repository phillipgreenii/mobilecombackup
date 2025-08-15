// Package errors provides custom error types and utility functions for error handling
// across the mobilecombackup application. It includes structured error types with
// categorization, wrapping capabilities, and contextual information.
package errors

import (
	"path/filepath"
	"runtime"
)

// NewValidationError creates a new ValidationError with automatic caller context
func NewValidationError(operation string, err error) *ValidationError {
	_, file, line, _ := runtime.Caller(1)
	return &ValidationError{
		File:      filepath.Base(file),
		Line:      line,
		Operation: operation,
		Code:      ErrCodeValidation,
		Err:       err,
	}
}

// NewValidationErrorWithCode creates a new ValidationError with a specific error code
func NewValidationErrorWithCode(operation string, code ErrorCode, err error) *ValidationError {
	_, file, line, _ := runtime.Caller(1)
	return &ValidationError{
		File:      filepath.Base(file),
		Line:      line,
		Operation: operation,
		Code:      code,
		Err:       err,
	}
}

// NewProcessingError creates a new ProcessingError
func NewProcessingError(stage, inputFile string, err error) *ProcessingError {
	return &ProcessingError{
		Stage:     stage,
		InputFile: inputFile,
		Code:      ErrCodeProcessing,
		Err:       err,
	}
}

// NewProcessingErrorWithCode creates a new ProcessingError with a specific error code
func NewProcessingErrorWithCode(stage, inputFile string, code ErrorCode, err error) *ProcessingError {
	return &ProcessingError{
		Stage:     stage,
		InputFile: inputFile,
		Code:      code,
		Err:       err,
	}
}

// NewImportError creates a new ImportError
func NewImportError(phase, entity string, count int, err error) *ImportError {
	return &ImportError{
		Phase:  phase,
		Entity: entity,
		Count:  count,
		Code:   ErrCodeImport,
		Err:    err,
	}
}

// NewImportErrorWithCode creates a new ImportError with a specific error code
func NewImportErrorWithCode(phase, entity string, count int, code ErrorCode, err error) *ImportError {
	return &ImportError{
		Phase:  phase,
		Entity: entity,
		Count:  count,
		Code:   code,
		Err:    err,
	}
}

// NewFileError creates a new FileError
func NewFileError(path, operation string, err error) *FileError {
	return &FileError{
		Path:      path,
		Operation: operation,
		Code:      ErrCodeFileNotFound, // Default to file not found, can be overridden
		Err:       err,
	}
}

// NewFileErrorWithCode creates a new FileError with a specific error code
func NewFileErrorWithCode(path, operation string, code ErrorCode, err error) *FileError {
	return &FileError{
		Path:      path,
		Operation: operation,
		Code:      code,
		Err:       err,
	}
}

// NewConfigurationError creates a new ConfigurationError
func NewConfigurationError(key, value string, err error) *ConfigurationError {
	return &ConfigurationError{
		Key:   key,
		Value: value,
		Code:  ErrCodeConfiguration,
		Err:   err,
	}
}

// NewConfigurationErrorWithCode creates a new ConfigurationError with a specific error code
func NewConfigurationErrorWithCode(key, value string, code ErrorCode, err error) *ConfigurationError {
	return &ConfigurationError{
		Key:   key,
		Value: value,
		Code:  code,
		Err:   err,
	}
}

// WrapWithValidation wraps an error with validation context
func WrapWithValidation(operation string, err error) error {
	if err == nil {
		return nil
	}
	return NewValidationError(operation, err)
}

// WrapWithProcessing wraps an error with processing context
func WrapWithProcessing(stage, inputFile string, err error) error {
	if err == nil {
		return nil
	}
	return NewProcessingError(stage, inputFile, err)
}

// WrapWithImport wraps an error with import context
func WrapWithImport(phase, entity string, count int, err error) error {
	if err == nil {
		return nil
	}
	return NewImportError(phase, entity, count, err)
}

// WrapWithFile wraps an error with file operation context
func WrapWithFile(path, operation string, err error) error {
	if err == nil {
		return nil
	}
	return NewFileError(path, operation, err)
}

// WrapWithConfiguration wraps an error with configuration context
func WrapWithConfiguration(key, value string, err error) error {
	if err == nil {
		return nil
	}
	return NewConfigurationError(key, value, err)
}

// GetErrorCode extracts the error code from an error if it implements ErrorCoder
func GetErrorCode(err error) (ErrorCode, bool) {
	if coder, ok := err.(ErrorCoder); ok {
		return coder.ErrorCode(), true
	}
	return "", false
}

// IsErrorCode checks if an error has a specific error code
func IsErrorCode(err error, code ErrorCode) bool {
	if errCode, ok := GetErrorCode(err); ok {
		return errCode == code
	}
	return false
}
