package errors

import (
	"errors"
	"strings"
	"testing"
)

const (
	// Test constants
	testOperation = "test operation"
)

func TestNewValidationError(t *testing.T) {
	t.Parallel()

	underlyingErr := errors.New("test error")
	err := NewValidationError(testOperation, underlyingErr)

	// Check that it's a ValidationError
	if err == nil {
		t.Fatal("NewValidationError() returned nil")
	}

	// Check operation
	if err.Operation != testOperation {
		t.Errorf("Operation = %q, want %q", err.Operation, testOperation)
	}

	// Check underlying error
	if err.Err != underlyingErr {
		t.Errorf("Err = %v, want %v", err.Err, underlyingErr)
	}

	// Check default error code
	if err.Code != ErrCodeValidation {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeValidation)
	}

	// Check that file and line are set (from runtime.Caller)
	if err.File == "" {
		t.Error("File should be set by runtime.Caller")
	}
	if err.Line == 0 {
		t.Error("Line should be set by runtime.Caller")
	}

	// Check that error message contains file and line
	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "helpers_test.go") {
		t.Errorf("Error message should contain file name, got: %s", errorMsg)
	}
}

func TestNewValidationErrorWithCode(t *testing.T) {
	t.Parallel()

	underlyingErr := errors.New("test error")
	err := NewValidationErrorWithCode(testOperation, ErrCodeParsing, underlyingErr)

	// Check that the custom error code is set
	if err.Code != ErrCodeParsing {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeParsing)
	}

	// Check other fields
	if err.Operation != testOperation {
		t.Errorf("Operation = %q, want %q", err.Operation, testOperation)
	}
	if err.Err != underlyingErr {
		t.Errorf("Err = %v, want %v", err.Err, underlyingErr)
	}
}

func TestNewProcessingError(t *testing.T) {
	t.Parallel()

	underlyingErr := errors.New("test error")
	err := NewProcessingError("parsing", "input.xml", underlyingErr)

	if err.Stage != "parsing" {
		t.Errorf("Stage = %q, want %q", err.Stage, "parsing")
	}
	if err.InputFile != "input.xml" {
		t.Errorf("InputFile = %q, want %q", err.InputFile, "input.xml")
	}
	if err.Code != ErrCodeProcessing {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeProcessing)
	}
	if err.Err != underlyingErr {
		t.Errorf("Err = %v, want %v", err.Err, underlyingErr)
	}
}

func TestNewProcessingErrorWithCode(t *testing.T) {
	t.Parallel()

	underlyingErr := errors.New("test error")
	err := NewProcessingErrorWithCode("parsing", "input.xml", ErrCodeParsing, underlyingErr)

	if err.Code != ErrCodeParsing {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeParsing)
	}
}

func TestNewImportError(t *testing.T) {
	t.Parallel()

	underlyingErr := errors.New("test error")
	err := NewImportError("validation", "contacts", 100, underlyingErr)

	if err.Phase != "validation" {
		t.Errorf("Phase = %q, want %q", err.Phase, "validation")
	}
	if err.Entity != "contacts" {
		t.Errorf("Entity = %q, want %q", err.Entity, "contacts")
	}
	if err.Count != 100 {
		t.Errorf("Count = %d, want %d", err.Count, 100)
	}
	if err.Code != ErrCodeImport {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeImport)
	}
	if err.Err != underlyingErr {
		t.Errorf("Err = %v, want %v", err.Err, underlyingErr)
	}
}

func TestNewImportErrorWithCode(t *testing.T) {
	t.Parallel()

	underlyingErr := errors.New("test error")
	err := NewImportErrorWithCode("validation", "contacts", 100, ErrCodeValidation, underlyingErr)

	if err.Code != ErrCodeValidation {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeValidation)
	}
}

func TestNewFileError(t *testing.T) {
	t.Parallel()

	underlyingErr := errors.New("test error")
	err := NewFileError("/path/to/file.txt", "read", underlyingErr)

	if err.Path != "/path/to/file.txt" {
		t.Errorf("Path = %q, want %q", err.Path, "/path/to/file.txt")
	}
	if err.Operation != "read" {
		t.Errorf("Operation = %q, want %q", err.Operation, "read")
	}
	if err.Code != ErrCodeFileNotFound {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeFileNotFound)
	}
	if err.Err != underlyingErr {
		t.Errorf("Err = %v, want %v", err.Err, underlyingErr)
	}
}

func TestNewFileErrorWithCode(t *testing.T) {
	t.Parallel()

	underlyingErr := errors.New("test error")
	err := NewFileErrorWithCode("/path/to/file.txt", "write", ErrCodePermission, underlyingErr)

	if err.Code != ErrCodePermission {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodePermission)
	}
}

func TestNewConfigurationError(t *testing.T) {
	t.Parallel()

	underlyingErr := errors.New("test error")
	err := NewConfigurationError("max_files", "invalid", underlyingErr)

	if err.Key != "max_files" {
		t.Errorf("Key = %q, want %q", err.Key, "max_files")
	}
	if err.Value != "invalid" {
		t.Errorf("Value = %q, want %q", err.Value, "invalid")
	}
	if err.Code != ErrCodeConfiguration {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeConfiguration)
	}
	if err.Err != underlyingErr {
		t.Errorf("Err = %v, want %v", err.Err, underlyingErr)
	}
}

func TestNewConfigurationErrorWithCode(t *testing.T) {
	t.Parallel()

	underlyingErr := errors.New("test error")
	err := NewConfigurationErrorWithCode("key", "value", ErrCodeValidation, underlyingErr)

	if err.Code != ErrCodeValidation {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeValidation)
	}
}

func TestWrapperFunctions(t *testing.T) {
	underlyingErr := errors.New("test error")

	// Test WrapWithValidation
	err := WrapWithValidation(testOperation, underlyingErr)
	if validationErr, ok := err.(*ValidationError); ok {
		if validationErr.Operation != testOperation {
			t.Errorf("WrapWithValidation operation = %q, want %q", validationErr.Operation, testOperation)
		}
		if validationErr.Err != underlyingErr {
			t.Errorf("WrapWithValidation err = %v, want %v", validationErr.Err, underlyingErr)
		}
	} else {
		t.Errorf("WrapWithValidation should return *ValidationError, got %T", err)
	}

	// Test WrapWithProcessing
	err = WrapWithProcessing("parsing", "input.xml", underlyingErr)
	if processingErr, ok := err.(*ProcessingError); ok {
		if processingErr.Stage != "parsing" {
			t.Errorf("WrapWithProcessing stage = %q, want %q", processingErr.Stage, "parsing")
		}
	} else {
		t.Errorf("WrapWithProcessing should return *ProcessingError, got %T", err)
	}

	// Test WrapWithImport
	err = WrapWithImport("validation", "contacts", 100, underlyingErr)
	if importErr, ok := err.(*ImportError); ok {
		if importErr.Phase != "validation" {
			t.Errorf("WrapWithImport phase = %q, want %q", importErr.Phase, "validation")
		}
	} else {
		t.Errorf("WrapWithImport should return *ImportError, got %T", err)
	}

	// Test WrapWithFile
	err = WrapWithFile("/path/to/file.txt", "read", underlyingErr)
	if fileErr, ok := err.(*FileError); ok {
		if fileErr.Path != "/path/to/file.txt" {
			t.Errorf("WrapWithFile path = %q, want %q", fileErr.Path, "/path/to/file.txt")
		}
	} else {
		t.Errorf("WrapWithFile should return *FileError, got %T", err)
	}

	// Test WrapWithConfiguration
	err = WrapWithConfiguration("key", "value", underlyingErr)
	if configErr, ok := err.(*ConfigurationError); ok {
		if configErr.Key != "key" {
			t.Errorf("WrapWithConfiguration key = %q, want %q", configErr.Key, "key")
		}
	} else {
		t.Errorf("WrapWithConfiguration should return *ConfigurationError, got %T", err)
	}
}

func TestWrapperFunctions_NilError(t *testing.T) {
	// Test that wrapper functions return nil when given nil error
	if err := WrapWithValidation("test", nil); err != nil {
		t.Errorf("WrapWithValidation(nil) = %v, want nil", err)
	}
	if err := WrapWithProcessing("test", "file", nil); err != nil {
		t.Errorf("WrapWithProcessing(nil) = %v, want nil", err)
	}
	if err := WrapWithImport("test", "entity", 0, nil); err != nil {
		t.Errorf("WrapWithImport(nil) = %v, want nil", err)
	}
	if err := WrapWithFile("path", "op", nil); err != nil {
		t.Errorf("WrapWithFile(nil) = %v, want nil", err)
	}
	if err := WrapWithConfiguration("key", "value", nil); err != nil {
		t.Errorf("WrapWithConfiguration(nil) = %v, want nil", err)
	}
}

func TestGetErrorCode(t *testing.T) {
	// Test with structured error
	structuredErr := &ValidationError{
		Operation: "test",
		Code:      ErrCodeValidation,
		Err:       errors.New("test"),
	}

	code, ok := GetErrorCode(structuredErr)
	if !ok {
		t.Error("GetErrorCode should return true for structured error")
	}
	if code != ErrCodeValidation {
		t.Errorf("GetErrorCode = %v, want %v", code, ErrCodeValidation)
	}

	// Test with standard error
	standardErr := errors.New("standard error")
	code, ok = GetErrorCode(standardErr)
	if ok {
		t.Error("GetErrorCode should return false for standard error")
	}
	if code != "" {
		t.Errorf("GetErrorCode code should be empty for standard error, got %v", code)
	}
}

func TestIsErrorCode(t *testing.T) {
	// Test with structured error
	structuredErr := &ValidationError{
		Operation: "test",
		Code:      ErrCodeValidation,
		Err:       errors.New("test"),
	}

	if !IsErrorCode(structuredErr, ErrCodeValidation) {
		t.Error("IsErrorCode should return true for matching error code")
	}
	if IsErrorCode(structuredErr, ErrCodeProcessing) {
		t.Error("IsErrorCode should return false for non-matching error code")
	}

	// Test with standard error
	standardErr := errors.New("standard error")
	if IsErrorCode(standardErr, ErrCodeValidation) {
		t.Error("IsErrorCode should return false for standard error")
	}
}

func TestChainedErrorCode(t *testing.T) {
	// Test that error codes work through error chains
	originalErr := errors.New("original")
	processingErr := NewProcessingError("parsing", "file.xml", originalErr)
	validationErr := NewValidationError("validate", processingErr)

	// Should be able to get the validation error code from the outer error
	if !IsErrorCode(validationErr, ErrCodeValidation) {
		t.Error("Should be able to get ValidationError code from outer error")
	}

	// Should be able to get the processing error code from the chain
	code, ok := GetErrorCode(processingErr)
	if !ok || code != ErrCodeProcessing {
		t.Error("Should be able to get ProcessingError code from chain")
	}
}
