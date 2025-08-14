package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestValidationError(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	validationErr := &ValidationError{
		File:      "test.go",
		Line:      42,
		Operation: "validate input",
		Code:      ErrCodeValidation,
		Err:       underlyingErr,
	}

	// Test Error() method
	expected := "test.go:42: validate input failed: underlying error"
	if validationErr.Error() != expected {
		t.Errorf("Error() = %q, want %q", validationErr.Error(), expected)
	}

	// Test Unwrap() method
	if validationErr.Unwrap() != underlyingErr {
		t.Errorf("Unwrap() = %v, want %v", validationErr.Unwrap(), underlyingErr)
	}

	// Test ErrorCode() method
	if validationErr.ErrorCode() != ErrCodeValidation {
		t.Errorf("ErrorCode() = %v, want %v", validationErr.ErrorCode(), ErrCodeValidation)
	}

	// Test errors.Is() compatibility
	if !errors.Is(validationErr, underlyingErr) {
		t.Error("errors.Is() should return true for wrapped error")
	}

	// Test errors.As() compatibility
	var target *ValidationError
	if !errors.As(validationErr, &target) {
		t.Error("errors.As() should return true for ValidationError")
	}
	if target != validationErr {
		t.Error("errors.As() should set target to the same ValidationError")
	}
}

func TestValidationError_WithoutFileContext(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	validationErr := &ValidationError{
		Operation: "validate input",
		Code:      ErrCodeValidation,
		Err:       underlyingErr,
	}

	// Test Error() method without file context
	expected := "validate input failed: underlying error"
	if validationErr.Error() != expected {
		t.Errorf("Error() = %q, want %q", validationErr.Error(), expected)
	}
}

func TestProcessingError(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	processingErr := &ProcessingError{
		Stage:     "parsing",
		InputFile: "input.xml",
		Code:      ErrCodeProcessing,
		Err:       underlyingErr,
	}

	// Test Error() method
	expected := "processing failed at stage 'parsing' for file 'input.xml': underlying error"
	if processingErr.Error() != expected {
		t.Errorf("Error() = %q, want %q", processingErr.Error(), expected)
	}

	// Test Unwrap() method
	if processingErr.Unwrap() != underlyingErr {
		t.Errorf("Unwrap() = %v, want %v", processingErr.Unwrap(), underlyingErr)
	}

	// Test ErrorCode() method
	if processingErr.ErrorCode() != ErrCodeProcessing {
		t.Errorf("ErrorCode() = %v, want %v", processingErr.ErrorCode(), ErrCodeProcessing)
	}
}

func TestProcessingError_WithoutInputFile(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	processingErr := &ProcessingError{
		Stage: "parsing",
		Code:  ErrCodeProcessing,
		Err:   underlyingErr,
	}

	// Test Error() method without input file
	expected := "processing failed at stage 'parsing': underlying error"
	if processingErr.Error() != expected {
		t.Errorf("Error() = %q, want %q", processingErr.Error(), expected)
	}
}

func TestImportError(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	importErr := &ImportError{
		Phase:  "validation",
		Entity: "contacts",
		Count:  150,
		Code:   ErrCodeImport,
		Err:    underlyingErr,
	}

	// Test Error() method
	expected := "import failed during 'validation' phase for contacts after processing 150 items: underlying error"
	if importErr.Error() != expected {
		t.Errorf("Error() = %q, want %q", importErr.Error(), expected)
	}

	// Test Unwrap() method
	if importErr.Unwrap() != underlyingErr {
		t.Errorf("Unwrap() = %v, want %v", importErr.Unwrap(), underlyingErr)
	}

	// Test ErrorCode() method
	if importErr.ErrorCode() != ErrCodeImport {
		t.Errorf("ErrorCode() = %v, want %v", importErr.ErrorCode(), ErrCodeImport)
	}
}

func TestImportError_WithoutCount(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	importErr := &ImportError{
		Phase:  "validation",
		Entity: "contacts",
		Count:  0,
		Code:   ErrCodeImport,
		Err:    underlyingErr,
	}

	// Test Error() method without count
	expected := "import failed during 'validation' phase for contacts: underlying error"
	if importErr.Error() != expected {
		t.Errorf("Error() = %q, want %q", importErr.Error(), expected)
	}
}

func TestFileError(t *testing.T) {
	underlyingErr := errors.New("file not found")
	fileErr := &FileError{
		Path:      "/path/to/file.txt",
		Operation: "read",
		Code:      ErrCodeFileNotFound,
		Err:       underlyingErr,
	}

	// Test Error() method
	expected := "file read failed for '/path/to/file.txt': file not found"
	if fileErr.Error() != expected {
		t.Errorf("Error() = %q, want %q", fileErr.Error(), expected)
	}

	// Test Unwrap() method
	if fileErr.Unwrap() != underlyingErr {
		t.Errorf("Unwrap() = %v, want %v", fileErr.Unwrap(), underlyingErr)
	}

	// Test ErrorCode() method
	if fileErr.ErrorCode() != ErrCodeFileNotFound {
		t.Errorf("ErrorCode() = %v, want %v", fileErr.ErrorCode(), ErrCodeFileNotFound)
	}
}

func TestConfigurationError(t *testing.T) {
	underlyingErr := errors.New("invalid value")
	configErr := &ConfigurationError{
		Key:   "max_files",
		Value: "not_a_number",
		Code:  ErrCodeConfiguration,
		Err:   underlyingErr,
	}

	// Test Error() method
	expected := "configuration error for key 'max_files' with value 'not_a_number': invalid value"
	if configErr.Error() != expected {
		t.Errorf("Error() = %q, want %q", configErr.Error(), expected)
	}

	// Test Unwrap() method
	if configErr.Unwrap() != underlyingErr {
		t.Errorf("Unwrap() = %v, want %v", configErr.Unwrap(), underlyingErr)
	}

	// Test ErrorCode() method
	if configErr.ErrorCode() != ErrCodeConfiguration {
		t.Errorf("ErrorCode() = %v, want %v", configErr.ErrorCode(), ErrCodeConfiguration)
	}
}

func TestConfigurationError_WithoutValue(t *testing.T) {
	underlyingErr := errors.New("missing key")
	configErr := &ConfigurationError{
		Key:  "required_key",
		Code: ErrCodeConfiguration,
		Err:  underlyingErr,
	}

	// Test Error() method without value
	expected := "configuration error for key 'required_key': missing key"
	if configErr.Error() != expected {
		t.Errorf("Error() = %q, want %q", configErr.Error(), expected)
	}
}

func TestErrorCodes(t *testing.T) {
	tests := []struct {
		name string
		code ErrorCode
		want string
	}{
		{"validation", ErrCodeValidation, "VALIDATION_ERROR"},
		{"file not found", ErrCodeFileNotFound, "FILE_NOT_FOUND"},
		{"parsing", ErrCodeParsing, "PARSE_ERROR"},
		{"permission", ErrCodePermission, "PERMISSION_ERROR"},
		{"storage", ErrCodeStorage, "STORAGE_ERROR"},
		{"integrity", ErrCodeIntegrity, "INTEGRITY_ERROR"},
		{"processing", ErrCodeProcessing, "PROCESSING_ERROR"},
		{"import", ErrCodeImport, "IMPORT_ERROR"},
		{"configuration", ErrCodeConfiguration, "CONFIG_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.code) != tt.want {
				t.Errorf("ErrorCode %s = %q, want %q", tt.name, string(tt.code), tt.want)
			}
		})
	}
}

func TestErrorChaining(t *testing.T) {
	// Create a chain of errors
	originalErr := errors.New("original error")

	processingErr := &ProcessingError{
		Stage:     "parsing",
		InputFile: "input.xml",
		Code:      ErrCodeParsing,
		Err:       originalErr,
	}

	validationErr := &ValidationError{
		File:      "validator.go",
		Line:      100,
		Operation: "validate XML",
		Code:      ErrCodeValidation,
		Err:       processingErr,
	}

	// Test that errors.Is() works through the chain
	if !errors.Is(validationErr, originalErr) {
		t.Error("errors.Is() should find original error through the chain")
	}
	if !errors.Is(validationErr, processingErr) {
		t.Error("errors.Is() should find processing error in the chain")
	}

	// Test that errors.As() works through the chain
	var targetProcessing *ProcessingError
	if !errors.As(validationErr, &targetProcessing) {
		t.Error("errors.As() should find ProcessingError in the chain")
	}
	if targetProcessing != processingErr {
		t.Error("errors.As() should return the correct ProcessingError")
	}

	// Test error message contains context from both errors
	errorMsg := validationErr.Error()
	if !strings.Contains(errorMsg, "validator.go:100") {
		t.Error("Error message should contain file and line context")
	}
	if !strings.Contains(errorMsg, "validate XML failed") {
		t.Error("Error message should contain operation context")
	}
}

func TestErrorInterface(t *testing.T) {
	// Test that all error types implement the error interface
	var err error

	err = &ValidationError{Operation: "test", Err: errors.New("test")}
	if err.Error() == "" {
		t.Error("ValidationError should implement error interface")
	}

	err = &ProcessingError{Stage: "test", Err: errors.New("test")}
	if err.Error() == "" {
		t.Error("ProcessingError should implement error interface")
	}

	err = &ImportError{Phase: "test", Entity: "test", Err: errors.New("test")}
	if err.Error() == "" {
		t.Error("ImportError should implement error interface")
	}

	err = &FileError{Path: "test", Operation: "test", Err: errors.New("test")}
	if err.Error() == "" {
		t.Error("FileError should implement error interface")
	}

	err = &ConfigurationError{Key: "test", Err: errors.New("test")}
	if err.Error() == "" {
		t.Error("ConfigurationError should implement error interface")
	}
}

func TestErrorCoderInterface(t *testing.T) {
	// Test that all error types implement the ErrorCoder interface
	tests := []struct {
		name string
		err  ErrorCoder
		want ErrorCode
	}{
		{
			"ValidationError",
			&ValidationError{Code: ErrCodeValidation, Operation: "test", Err: errors.New("test")},
			ErrCodeValidation,
		},
		{
			"ProcessingError",
			&ProcessingError{Code: ErrCodeProcessing, Stage: "test", Err: errors.New("test")},
			ErrCodeProcessing,
		},
		{
			"ImportError",
			&ImportError{Code: ErrCodeImport, Phase: "test", Entity: "test", Err: errors.New("test")},
			ErrCodeImport,
		},
		{
			"FileError",
			&FileError{Code: ErrCodeFileNotFound, Path: "test", Operation: "test", Err: errors.New("test")},
			ErrCodeFileNotFound,
		},
		{
			"ConfigurationError",
			&ConfigurationError{Code: ErrCodeConfiguration, Key: "test", Err: errors.New("test")},
			ErrCodeConfiguration,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.ErrorCode() != tt.want {
				t.Errorf("%s.ErrorCode() = %v, want %v", tt.name, tt.err.ErrorCode(), tt.want)
			}
		})
	}
}
