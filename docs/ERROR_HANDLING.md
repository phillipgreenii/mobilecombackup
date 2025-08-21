# Error Handling Guidelines

This document provides guidelines for consistent error handling throughout the mobilecombackup project using the structured error types from `pkg/errors`.

## Overview

The project uses structured error types that provide context information and maintain compatibility with Go 1.13+ error handling patterns. These errors help with debugging by capturing file locations, operation contexts, and error categories.

## Error Types

### Available Error Types

The `pkg/errors` package provides several structured error types:

- **ValidationError**: For validation failures with file/line context
- **ProcessingError**: For data processing errors with stage and input file context  
- **ImportError**: For import operation errors with phase, entity, and count context
- **FileError**: For file operation errors with path and operation context
- **ConfigurationError**: For configuration-related errors with key/value context

### Error Codes

Each error type includes a category code for programmatic handling:

```go
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
```

## Usage Patterns

### Creating Structured Errors

Use helper functions to create structured errors with automatic context:

```go
import customerrors "github.com/phillipgreenii/mobilecombackup/pkg/errors"

// Validation error with automatic file/line context
err := customerrors.NewValidationError("validate user input", originalErr)

// Processing error with stage and file context
err := customerrors.NewProcessingError("parsing XML", "backup.xml", originalErr)

// Import error with phase, entity, and count context
err := customerrors.NewImportError("validation", "contacts", 150, originalErr)

// File operation error
err := customerrors.NewFileError("/path/to/file", "read", originalErr)

// Configuration error
err := customerrors.NewConfigurationError("max_files", "invalid", originalErr)
```

### Wrapping Existing Errors

Use wrapper functions to add context to existing errors:

```go
// Wrap with validation context
if err := validateInput(data); err != nil {
    return customerrors.WrapWithValidation("validate user data", err)
}

// Wrap with processing context  
if err := parseFile(filename); err != nil {
    return customerrors.WrapWithProcessing("parsing", filename, err)
}

// Wrap with file operation context
if err := os.Open(path); err != nil {
    return customerrors.WrapWithFile(path, "open", err)
}
```

### Error Code Checking

Check error types programmatically using error codes:

```go
if customerrors.IsErrorCode(err, customerrors.ErrCodeValidation) {
    // Handle validation errors specifically
    log.Warn("Validation failed, using defaults")
}

if code, ok := customerrors.GetErrorCode(err); ok {
    switch code {
    case customerrors.ErrCodeFileNotFound:
        // Handle missing files
    case customerrors.ErrCodePermission:
        // Handle permission errors
    }
}
```

### Error Chaining

Structured errors work seamlessly with Go 1.13+ error handling:

```go
// Create error chain
originalErr := fmt.Errorf("connection failed")
processingErr := customerrors.NewProcessingError("fetch data", "remote.xml", originalErr)
validationErr := customerrors.NewValidationError("validate remote data", processingErr)

// Check for specific errors in the chain
if errors.Is(validationErr, originalErr) {
    // originalErr found in chain
}

// Extract specific error types from chain
var procErr *customerrors.ProcessingError
if errors.As(validationErr, &procErr) {
    fmt.Printf("Processing failed at stage: %s\n", procErr.Stage)
}
```

## Best Practices

### When to Use Structured Errors

- **Always** use structured errors for new code
- **Prefer** wrapping existing errors rather than creating new ones
- **Use** appropriate error types based on context:
  - ValidationError: Input validation, schema validation, constraint checking
  - ProcessingError: Data transformation, parsing, computation stages
  - ImportError: Import operations with progress tracking
  - FileError: File I/O operations
  - ConfigurationError: Configuration parsing and validation

### Error Context Guidelines

- **Be specific** about the operation being performed
- **Include relevant identifiers** (file names, entity types, counts)
- **Use consistent terminology** across similar operations
- **Avoid sensitive information** in error messages

### Error Message Format

Structured errors automatically format messages with context:

```
ValidationError: "validator.go:42: validate input failed: value must be positive"
ProcessingError: "processing failed at stage 'parsing' for file 'backup.xml': unexpected EOF"
ImportError: "import failed during 'validation' phase for contacts after processing 150 items: duplicate name"
```

### Performance Considerations

- Structured errors have minimal overhead
- Automatic file/line capture uses `runtime.Caller(1)`
- Context information is captured only when errors occur
- Error codes enable efficient programmatic handling

## Migration Strategy

### Incremental Adoption

1. **New code**: Always use structured errors
2. **Critical paths**: Update error handling in validation and import flows
3. **Existing code**: Update opportunistically during maintenance
4. **Testing**: Ensure error handling tests work with both old and new patterns

### Compatibility

Structured errors are fully compatible with existing error handling:

```go
// Works with existing error checking
if err != nil {
    return err  // Error interface is maintained
}

// Works with error wrapping
return fmt.Errorf("operation failed: %w", structuredErr)

// Works with errors.Is/As
if errors.Is(err, specificErr) { ... }
```

## Testing Error Handling

### Unit Tests

Test error types and context preservation:

```go
func TestValidationErrorContext(t *testing.T) {
    originalErr := errors.New("invalid value")
    err := customerrors.NewValidationError("validate input", originalErr)
    
    // Test error message contains context
    if !strings.Contains(err.Error(), "validate input failed") {
        t.Error("Error message should contain operation context")
    }
    
    // Test error unwrapping
    if !errors.Is(err, originalErr) {
        t.Error("Should be able to unwrap to original error")
    }
    
    // Test error code
    if !customerrors.IsErrorCode(err, customerrors.ErrCodeValidation) {
        t.Error("Should have validation error code")
    }
}
```

### Integration Tests

Test error propagation through operation chains:

```go
func TestErrorPropagation(t *testing.T) {
    // Simulate operation that produces structured error
    err := performOperation()
    
    // Verify error type and context are preserved
    var validationErr *customerrors.ValidationError
    if !errors.As(err, &validationErr) {
        t.Error("Should propagate ValidationError type")
    }
    
    // Verify specific context is available
    if validationErr.Operation != "expected operation" {
        t.Error("Operation context should be preserved")
    }
}
```

## Examples

### Complete Error Handling Example

```go
func ProcessBackupFile(filename string) error {
    // Validate input
    if filename == "" {
        return customerrors.NewValidationError("validate filename", 
            errors.New("filename cannot be empty"))
    }
    
    // Open file
    file, err := os.Open(filename)
    if err != nil {
        return customerrors.WrapWithFile(filename, "open", err)
    }
    defer file.Close()
    
    // Parse content
    data, err := parseXML(file)
    if err != nil {
        return customerrors.WrapWithProcessing("parsing XML", filename, err)
    }
    
    // Validate content
    if err := validateData(data); err != nil {
        return customerrors.WrapWithValidation("validate parsed data", err)
    }
    
    return nil
}

// Usage with error handling
if err := ProcessBackupFile("backup.xml"); err != nil {
    // Log with context
    log.Error("Backup processing failed", "error", err)
    
    // Handle specific error types
    if customerrors.IsErrorCode(err, customerrors.ErrCodeFileNotFound) {
        return fmt.Errorf("backup file not found, please check the path")
    }
    
    if customerrors.IsErrorCode(err, customerrors.ErrCodeValidation) {
        return fmt.Errorf("backup file contains invalid data")
    }
    
    return fmt.Errorf("backup processing failed: %w", err)
}
```

This error handling approach provides rich context for debugging while maintaining compatibility with existing Go error handling patterns.