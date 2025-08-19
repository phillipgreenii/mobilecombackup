# FEAT-066: Standardize error handling patterns with context

## Status
- **Reported**: 2025-08-18
- **Completed**: 2025-08-19
- **Priority**: medium
- **Type**: code-quality

## Overview
The codebase has inconsistent error handling patterns across packages - some functions return raw errors without context, others use different wrapping strategies, and error context is frequently lost through the call stack, making debugging difficult.

## Requirements
Establish consistent error handling patterns with proper context preservation, standardized error wrapping, and clear error classification for better debugging and user experience.

## Design
### Current Problems
- Mix of raw errors and wrapped errors across packages
- Inconsistent error context information
- Loss of error context through call chains
- No standardized error classification or severity

### Proposed Solution
Standardize error handling with consistent patterns:

```go
// Standard error wrapping pattern
if err != nil {
    return fmt.Errorf("operation failed at %s: %w", location, err)
}

// Context-rich errors with operation details
if err := processFile(filename); err != nil {
    return fmt.Errorf("failed to process file %s: %w", filename, err)
}

// Structured error information
type ProcessingError struct {
    Operation string
    File      string
    Line      int
    Cause     error
}
```

## Implementation Plan
### Phase 1: Error Standards Definition (Week 1)
- Define standardized error wrapping patterns
- Create error handling guidelines and examples
- Design error context structures

### Phase 2: Package-by-Package Migration (Week 2-3)
- Start with core packages (pkg/importer, pkg/validation)
- Update error handling to follow standards
- Ensure proper error context preservation

### Phase 3: Tooling and Validation (Week 4)
- Add linting rules for error handling patterns
- Create error handling examples and documentation
- Add error context tests

## Tasks
- [ ] Define comprehensive error handling standards
- [ ] Create error wrapping utility functions in pkg/errors
- [ ] Update pkg/importer error handling to follow standards
- [ ] Update pkg/validation error handling for consistency
- [ ] Update pkg/calls, pkg/sms, pkg/contacts error handling
- [ ] Add golangci-lint rules for error handling patterns
- [ ] Create error handling examples and best practices guide
- [ ] Add tests for error context preservation
- [ ] Update existing tests to validate error messages

## Error Handling Standards
### Wrapping Pattern
```go
// Always include context about the operation
return fmt.Errorf("failed to import SMS file %s: %w", filename, err)

// Include relevant identifiers
return fmt.Errorf("validation failed for repository %s: %w", repoPath, err)

// For internal operations, include function context
return fmt.Errorf("parseXML: failed to decode element: %w", err)
```

### Error Classification
- **User Errors**: Invalid input, missing files, configuration errors
- **System Errors**: I/O failures, permission errors, resource exhaustion  
- **Logic Errors**: Programming errors, assertion failures, unexpected states

### Context Requirements
- Operation being performed
- Relevant file paths, identifiers, or parameters
- Function or module context where error occurred
- Preserve original error through error wrapping

## Testing
### Error Context Tests
- Verify error messages contain sufficient context
- Test error wrapping preserves original error
- Validate error unwrapping works correctly

### Error Propagation Tests
- Test error context preservation through call chains
- Verify error information reaches user appropriately
- Test error handling in concurrent operations

## Acceptance Criteria
- [ ] Consistent error wrapping patterns across all packages
- [ ] All errors include sufficient context for debugging
- [ ] Error unwrapping works correctly throughout codebase
- [ ] Linting rules enforce error handling standards
- [ ] Clear documentation and examples for error handling
- [ ] No loss of error context through call chains
- [ ] User-friendly error messages for common failure cases
- [ ] Structured error types where appropriate

## Technical Considerations
### Error Performance
- Minimize error allocation overhead
- Avoid excessive string formatting in happy path
- Use efficient error wrapping techniques

### Backward Compatibility
- Maintain existing error types where possible
- Ensure error detection logic still works
- Preserve error codes and classifications

### Context Preservation
- Always use `fmt.Errorf` with `%w` verb for wrapping
- Include operation context in error messages
- Preserve stack trace information where helpful

## Error Message Guidelines
### Good Examples
```go
"failed to import SMS file 'messages.xml': invalid XML structure at line 42"
"repository validation failed at /path/repo: missing marker file"
"attachment extraction failed for hash abc123: file size exceeds limit"
```

### Avoid
```go
"error"
"failed"
"something went wrong"
"validation error"
```

## Related Issues
- Improves debugging and troubleshooting experience
- Enables better error reporting and logging
- Part of code quality improvement initiative

## Notes
Focus on user-facing operations first (CLI commands, import/export), then internal operations. Error messages should provide actionable information when possible, guiding users toward resolution.