# Migration Guide: Error Handling Standardization

This guide helps you adopt the new standardized error handling patterns that provide better context and debugging information.

## Overview

The error handling across the codebase has been standardized to provide consistent error context, proper error wrapping, and improved debugging experience. While most of the changes are internal, this affects how errors are reported and should be handled.

## Key Changes

### Enhanced Error Context

Errors now include more contextual information about the operation that failed:

#### Before
```go
// Old error messages
err: "validation failed"
err: "failed to read file"
err: "processing error"
```

#### After
```go
// New error messages with context
err: "failed to import SMS file 'messages.xml': validation failed"
err: "failed to process file '/path/to/backup.xml': invalid XML structure"
err: "failed to process import files: stat: no such file or directory"
```

### Proper Error Wrapping

All errors now properly preserve the error chain using `fmt.Errorf` with `%w`:

#### Before
```go
// Error chain was lost
if err != nil {
    return fmt.Errorf("operation failed: %s", err.Error())
}
```

#### After
```go
// Error chain preserved
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

## Error Handling Patterns

### Standard Error Wrapping

When wrapping errors, always include operation context:

```go
// Include what operation failed and preserve error chain
if err := processFile(filename); err != nil {
    return fmt.Errorf("failed to process file %s: %w", filename, err)
}

// Include relevant parameters and context
if err := validateRepository(repoPath); err != nil {
    return fmt.Errorf("validation failed for repository %s: %w", repoPath, err)
}

// For internal operations, include function context
if err := parseXML(reader); err != nil {
    return fmt.Errorf("parseXML: failed to decode element: %w", err)
}
```

### Error Unwrapping

Take advantage of error unwrapping for better error handling:

```go
_, err := importer.Import()
if err != nil {
    // Check for specific error types in the chain
    var pathErr *os.PathError
    if errors.As(err, &pathErr) {
        // Handle file system errors specifically
        fmt.Printf("File system error: %s\n", pathErr.Path)
    }
    
    // Check for specific error conditions
    if errors.Is(err, os.ErrNotExist) {
        // Handle missing file case
        fmt.Println("File not found")
    }
    
    // Get the full error context
    fmt.Printf("Import failed: %v\n", err)
}
```

## Error Categories

Errors are now classified into clear categories:

### User Errors
Errors caused by invalid user input or configuration:
```go
// File not found, invalid paths, malformed XML
err: "failed to import SMS file '/missing/file.xml': no such file or directory"
```

### System Errors
Errors caused by system limitations or I/O issues:
```go
// Permission errors, disk space, network issues
err: "failed to write attachment to '/path/file': permission denied"
```

### Logic Errors
Programming errors or unexpected internal states:
```go
// Validation failures, assertion errors, unexpected conditions
err: "validation failed for repository '/repo': missing marker file"
```

## Migration Steps

### Step 1: Update Error Handling Code

Review your error handling code to take advantage of enhanced error context:

#### Before
```go
_, err := importer.Import()
if err != nil {
    log.Printf("Import failed: %s", err)
    return
}
```

#### After
```go
_, err := importer.Import()
if err != nil {
    // Error already contains full context
    log.Printf("Import failed: %v", err)
    
    // Or check for specific error types
    if errors.Is(err, os.ErrNotExist) {
        log.Printf("File not found during import")
    }
    return
}
```

### Step 2: Improve Error Logging

Take advantage of structured error information:

```go
_, err := importer.Import()
if err != nil {
    // Log with structured context
    logger.WithFields(map[string]interface{}{
        "error": err,
        "operation": "import",
    }).Error("Import operation failed")
    
    // Or extract specific error details
    var pathErr *os.PathError
    if errors.As(err, &pathErr) {
        logger.WithFields(map[string]interface{}{
            "path": pathErr.Path,
            "operation": pathErr.Op,
        }).Error("File system error during import")
    }
}
```

### Step 3: Update Tests

Update tests to check for improved error messages:

#### Before
```go
func TestImportError(t *testing.T) {
    _, err := importer.Import()
    if err == nil {
        t.Error("Expected error")
    }
    // Could only check that error occurred
}
```

#### After
```go
func TestImportError(t *testing.T) {
    _, err := importer.Import()
    if err == nil {
        t.Error("Expected error")
    }
    
    // Check for specific error context
    if !strings.Contains(err.Error(), "failed to process import files") {
        t.Errorf("Error missing expected context: %v", err)
    }
    
    // Check error chain preservation
    var pathErr *os.PathError
    if !errors.As(err, &pathErr) {
        t.Error("Expected os.PathError in error chain")
    }
}
```

## Error Message Guidelines

### Good Error Messages

Error messages should be actionable and include relevant context:

```go
// ✅ Good: Includes operation, file, and specific problem
"failed to import SMS file 'messages.xml': invalid XML structure at line 42"

// ✅ Good: Includes repository path and specific issue
"repository validation failed at /path/repo: missing marker file"

// ✅ Good: Includes hash and specific constraint violated
"attachment extraction failed for hash abc123: file size exceeds limit"
```

### Messages to Avoid

Avoid generic or unhelpful error messages:

```go
// ❌ Avoid: Too generic
"error"
"failed"
"something went wrong"

// ❌ Avoid: No context
"validation error"
"file error"
"processing failed"
```

## Error Context Preservation

The error handling improvements ensure that context is preserved through the entire call chain:

```go
// Error flows from specific failure through multiple layers
// Each layer adds relevant context while preserving the original error

// Layer 1: File system error
err := os.Open("/missing/file.xml")
// => "open /missing/file.xml: no such file or directory"

// Layer 2: Processing layer adds context
err = fmt.Errorf("failed to process SMS file %s: %w", filename, err)
// => "failed to process SMS file 'messages.xml': open /missing/file.xml: no such file or directory"

// Layer 3: Import layer adds operation context
err = fmt.Errorf("failed to process import files: %w", err)
// => "failed to process import files: failed to process SMS file 'messages.xml': open /missing/file.xml: no such file or directory"
```

## Best Practices

### Always Include Context
```go
// Include relevant identifiers and operation details
return fmt.Errorf("failed to import %s file %s: %w", fileType, filename, err)
```

### Preserve Error Chains
```go
// Always use %w for error wrapping
return fmt.Errorf("operation context: %w", err)
```

### Use Error Checking Functions
```go
// Take advantage of errors.Is() and errors.As()
if errors.Is(err, os.ErrNotExist) {
    // Handle missing file case
}

var pathErr *os.PathError
if errors.As(err, &pathErr) {
    // Handle file system errors
}
```

### Test Error Context
```go
// Test that errors contain expected context
if !strings.Contains(err.Error(), "expected context") {
    t.Errorf("Error missing context: %v", err)
}
```

## Troubleshooting

### Debugging Error Chains

Use error unwrapping to debug complex error chains:

```go
func debugError(err error) {
    fmt.Printf("Error: %v\n", err)
    
    // Walk the error chain
    for err != nil {
        fmt.Printf("  Caused by: %v\n", err)
        err = errors.Unwrap(err)
    }
}
```

### Testing Error Types

Test for specific error types in the chain:

```go
func TestSpecificError(t *testing.T) {
    _, err := someOperation()
    
    // Check if specific error type exists in chain
    var targetErr *MyCustomError
    if !errors.As(err, &targetErr) {
        t.Error("Expected MyCustomError in error chain")
    }
}
```

## Backward Compatibility

The error handling improvements are largely backward compatible:

- **Error checking**: `if err != nil` continues to work
- **Error messages**: May be more detailed but basic checks still work
- **Error types**: Specific error types are preserved in the chain

However, code that depends on exact error message text may need updates to accommodate the additional context information.

## Support

For questions about error handling patterns or migration issues, please refer to the project documentation or file an issue in the repository.