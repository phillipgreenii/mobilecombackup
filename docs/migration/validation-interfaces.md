# Migration Guide: Validation Interface Evolution

This guide helps you migrate from legacy validation methods to the new context-aware interface methods.

## Overview

As of version 1.1.0, the validation interfaces have been enhanced with context-aware methods that provide better control over validation operations, including timeout handling and cancellation support.

## Timeline

- **Version 1.0.0**: Legacy methods available
- **Version 1.1.0**: Context-aware methods added, legacy methods deprecated
- **Version 2.0.0**: Legacy methods will be removed (planned 6 months from 1.1.0)

## Migration Steps

### Step 1: Update Method Calls

Replace legacy method calls with their context-aware equivalents:

#### Before (Legacy)
```go
// Legacy validation methods (deprecated)
report, err := validator.ValidateRepository()
violations := validator.ValidateStructure()
manifestViolations := validator.ValidateManifest()
contentViolations := validator.ValidateContent()
consistencyViolations := validator.ValidateConsistency()
```

#### After (Context-aware)
```go
// Context-aware validation methods (recommended)
ctx := context.Background()
report, err := validator.ValidateRepositoryContext(ctx)
violations := validator.ValidateStructureContext(ctx)
manifestViolations := validator.ValidateManifestContext(ctx)
contentViolations := validator.ValidateContentContext(ctx)
consistencyViolations := validator.ValidateConsistencyContext(ctx)
```

### Step 2: Add Context Management

Take advantage of context features for better control:

#### Timeout Control
```go
// Set validation timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

report, err := validator.ValidateRepositoryContext(ctx)
if errors.Is(err, context.DeadlineExceeded) {
    // Handle timeout
    fmt.Println("Validation timed out")
}
```

#### Cancellation Support
```go
// Cancellable validation
ctx, cancel := context.WithCancel(context.Background())

// Cancel validation on signal
go func() {
    <-interrupt
    cancel()
}()

report, err := validator.ValidateRepositoryContext(ctx)
if errors.Is(err, context.Canceled) {
    // Handle cancellation
    fmt.Println("Validation cancelled")
}
```

### Step 3: Update Tests

Modify tests to use context-aware methods:

#### Before
```go
func TestValidation(t *testing.T) {
    validator := NewRepositoryValidator(repoPath)
    report, err := validator.ValidateRepository()
    if err != nil {
        t.Errorf("Validation failed: %v", err)
    }
}
```

#### After
```go
func TestValidation(t *testing.T) {
    validator := NewRepositoryValidator(repoPath)
    ctx := context.Background()
    report, err := validator.ValidateRepositoryContext(ctx)
    if err != nil {
        t.Errorf("Validation failed: %v", err)
    }
}
```

### Step 4: Handle Context Errors

Add proper error handling for context-related errors:

```go
ctx := context.Background()
report, err := validator.ValidateRepositoryContext(ctx)

switch {
case errors.Is(err, context.Canceled):
    // User cancelled the operation
    return fmt.Errorf("validation was cancelled")
case errors.Is(err, context.DeadlineExceeded):
    // Operation timed out
    return fmt.Errorf("validation timed out")
case err != nil:
    // Other validation error
    return fmt.Errorf("validation failed: %w", err)
}

// Process successful validation report
```

## Interface Changes

### RepositoryValidator Interface

#### Deprecated Methods
```go
// These methods are deprecated and will be removed in v2.0.0
ValidateRepository() (*Report, error)
ValidateStructure() []Violation
ValidateManifest() []Violation
ValidateContent() []Violation
ValidateConsistency() []Violation
```

#### New Context-Aware Methods
```go
// Use these methods instead
ValidateRepositoryContext(ctx context.Context) (*Report, error)
ValidateStructureContext(ctx context.Context) []Violation
ValidateManifestContext(ctx context.Context) []Violation
ValidateContentContext(ctx context.Context) []Violation
ValidateConsistencyContext(ctx context.Context) []Violation
```

## Backward Compatibility

During the deprecation period (v1.1.0 to v2.0.0):

1. **Legacy methods still work**: All existing code continues to function
2. **Deprecation warnings**: Legacy methods include deprecation notices
3. **Internal delegation**: Legacy methods delegate to context-aware versions
4. **No breaking changes**: Behavior remains consistent

## Best Practices

### Use Context.Background() for Simple Cases
```go
// For simple validation without timeout/cancellation needs
ctx := context.Background()
report, err := validator.ValidateRepositoryContext(ctx)
```

### Set Reasonable Timeouts
```go
// For long-running validation operations
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()
```

### Propagate Context from Callers
```go
func MyValidationFunc(ctx context.Context, repoPath string) error {
    validator := NewRepositoryValidator(repoPath)
    _, err := validator.ValidateRepositoryContext(ctx)
    return err
}
```

## Automated Migration

### Linting Rules

We recommend adding linting rules to detect deprecated method usage:

```yaml
# .golangci.yml
linters:
  enable:
    - staticcheck

linters-settings:
  staticcheck:
    checks: ["all", "SA1019"] # Detect deprecated API usage
```

### Migration Script

A simple find-and-replace script can help with basic migration:

```bash
#!/bin/bash
# migrate-validation.sh

# Replace method calls (adjust patterns as needed)
find . -name "*.go" -exec sed -i.bak \
  -e 's/ValidateRepository()/ValidateRepositoryContext(context.Background())/g' \
  -e 's/ValidateStructure()/ValidateStructureContext(context.Background())/g' \
  -e 's/ValidateManifest()/ValidateManifestContext(context.Background())/g' \
  -e 's/ValidateContent()/ValidateContentContext(context.Background())/g' \
  -e 's/ValidateConsistency()/ValidateConsistencyContext(context.Background())/g' \
  {} \;

# Add context import if not present
find . -name "*.go" -exec goimports -w {} \;
```

## Troubleshooting

### "context" Package Not Imported
Add the context import:
```go
import "context"
```

### Tests Failing with Context Errors
Make sure to use `context.Background()` in tests unless specifically testing context behavior.

### Performance Concerns
Context-aware methods have negligible performance overhead compared to legacy methods.

## Support

If you encounter issues during migration:

1. Check that you're using Go 1.13+ (required for context package)
2. Ensure all imports are correct (`context` package)
3. Review error handling for context-specific errors
4. Consider gradual migration rather than wholesale changes

For questions or issues, please file an issue in the project repository.