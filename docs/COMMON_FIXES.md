# Common Issue Fixes

This document provides specific fix patterns for common errors encountered during development and verification.

## Overview

This guide helps agents automatically fix common issues that arise during the verification workflow. It covers test failures, lint violations, and build errors with specific fix patterns.

## When to Auto-Fix vs Ask User

### Auto-Fix These Issues:
- Import errors and missing dependencies
- Unused variables and dead code
- Formatting and style violations
- Simple type conversions
- Missing error handling patterns
- Documentation comments

### Ask User for Guidance When:
- Test logic appears incorrect (wrong expected values)
- Multiple valid approaches to fix an issue
- Fix would significantly change program behavior
- Unfamiliar error patterns not covered here
- Repeated failures after multiple fix attempts

## Test Failure Patterns

### Import and Dependency Errors

**Pattern**: `undefined: functionName`
**Fix**: 
```go
// Add missing import
import "github.com/phillipgreenii/mobilecombackup/pkg/missing"

// Or fix typo in function name
```

**Pattern**: `package not found`
**Fix**:
```bash
# Add missing dependency
go mod tidy

# Or check import path is correct
```

### Type Conversion Errors

**Pattern**: `cannot use x (type A) as type B`
**Fix**:
```go
// Add explicit type conversion
result := TypeB(x)

// Or use proper interface/struct field access
result := x.Field
```

### Unused Variable Errors

**Pattern**: `declared but not used`
**Fix**:
```go
// Option 1: Remove unused variable
// var unused string ← Delete this line

// Option 2: Use underscore if intentionally unused
_ = someValue

// Option 3: Actually use the variable
fmt.Println(previouslyUnused)
```

### Missing Test Data

**Pattern**: `no such file or directory: testdata/...`
**Fix**:
```bash
# Create missing test data directory
mkdir -p testdata/

# Create missing test files
touch testdata/expected_output.xml
```

### Permission Errors

**Pattern**: `permission denied`
**Fix**:
```bash
# Fix file permissions
chmod 644 file.go
chmod 755 directory/
```

## Lint Violation Patterns

### Unused Code

**Pattern**: `declared but not used`
**Fixes**:
```go
// Remove unused variables
// var unused string ← Delete

// Remove unused imports
// import "unused/package" ← Delete

// Remove unused functions (if not exported)
// func unusedFunc() {} ← Delete or export if needed
```

### Error Handling

**Pattern**: `Error return value is not checked`
**Fixes**:
```go
// Option 1: Handle the error
err := someFunction()
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}

// Option 2: Explicitly ignore (rare cases)
_ = someFunction()

// Option 3: Log and continue
if err := someFunction(); err != nil {
    log.Printf("warning: %v", err)
}
```

### Missing Documentation

**Pattern**: `should have comment or be unexported`
**Fix**:
```go
// Add documentation comment for exported functions
// ProcessData processes the input data according to specification
func ProcessData(input string) error {
    // ... implementation
}

// Or make function unexported if it's internal
func processData(input string) error {
    // ... implementation
}
```

### Formatting Issues

**Pattern**: Various formatting violations
**Fix**:
```bash
# Run formatter (should be automatic in workflow)
devbox run formatter

# Or direct gofmt
gofmt -w .
```

### Import Ordering

**Pattern**: Import order violations
**Fix**:
```bash
# Use goimports to fix import organization
goimports -w .

# Should be handled by devbox run formatter
```

## Build Failure Patterns

### Missing Imports

**Pattern**: `undefined: SomeType`
**Fix**:
```go
// Add missing import
import "github.com/phillipgreenii/mobilecombackup/pkg/types"

// Or import from standard library
import "fmt"
import "time"
```

### Syntax Errors

**Pattern**: Various syntax error messages
**Fixes**:
```go
// Missing closing brace
func example() {
    // code here
} // ← Add this

// Missing comma in struct
type Config struct {
    Field1 string,  // ← Add comma
    Field2 int
}

// Incorrect function signature
func (r *Receiver) Method() error { // ← Fix signature
    return nil
}
```

### Missing Dependencies

**Pattern**: `cannot find package`
**Fix**:
```bash
# Download missing dependencies
go mod download

# Tidy up module dependencies
go mod tidy

# Verify module integrity
go mod verify
```

## Go-Specific Common Fixes

### Interface Implementation

**Pattern**: Type doesn't implement interface
**Fix**:
```go
// Implement missing methods
func (t *Type) MissingMethod() error {
    return nil
}

// Check method signatures match interface exactly
```

### Receiver Issues

**Pattern**: Method set issues
**Fix**:
```go
// Use pointer receiver for mutating methods
func (t *Type) SetValue(v string) {
    t.value = v
}

// Use value receiver for read-only methods
func (t Type) GetValue() string {
    return t.value
}
```

### Context Usage

**Pattern**: Context-related issues
**Fix**:
```go
// Add context parameter
func ProcessData(ctx context.Context, data string) error {
    // implementation
}

// Use context in calls
ctx := context.Background()
err := ProcessData(ctx, "data")
```

## Project-Specific Patterns

### Import Paths

**Pattern**: Incorrect import paths
**Fix**:
```go
// Use full import path
import "github.com/phillipgreenii/mobilecombackup/pkg/calls"

// Not relative paths
// import "../calls" ← Wrong
```

### Test Data Location

**Pattern**: Test data in wrong location
**Fix**:
```go
// Use testdata/ directory
testFile := "testdata/sample.xml"

// Not /tmp or other locations
// testFile := "/tmp/sample.xml" ← Wrong
```

### Timestamp Handling

**Pattern**: Timestamp conversion issues
**Fix**:
```go
// Timestamps are milliseconds, divide by 1000 for Unix time
unixTime := timestamp / 1000
t := time.Unix(unixTime, 0).UTC()

// Always use UTC for consistency
```

## Fix Application Strategy

### Systematic Approach

1. **Read error message carefully** - understand the exact issue
2. **Identify pattern** - match to known patterns above
3. **Apply appropriate fix** - use the specific solution
4. **Re-run verification** - ensure fix worked
5. **Document unknown patterns** - add to this guide

### Multiple Errors

When multiple errors exist:
1. **Fix formatting first** - run `devbox run formatter`
2. **Fix imports and dependencies** - resolve import issues
3. **Fix syntax errors** - basic compilation issues
4. **Fix logic errors** - test and lint violations
5. **Re-run verification** - after each major fix category

### Error Cascades

Some errors cause others:
- Missing imports → undefined symbols → test failures
- Formatting issues → lint violations
- Syntax errors → build failures → test failures

Fix root causes first, then re-run verification.

## Integration with Workflows

This document supports:
- [Verification Workflow](VERIFICATION_WORKFLOW.md) - provides fixes for verification failures
- [Task Completion](TASK_COMPLETION.md) - enables automatic issue resolution
- All agent implementations - standard fix patterns

## Adding New Patterns

When you encounter a new error pattern:

1. **Document the pattern** - exact error message
2. **Document the fix** - specific solution steps
3. **Add to appropriate section** - test/lint/build
4. **Include examples** - code snippets showing fix
5. **Update this document** - for future reference

## Important Notes

- **Always re-run verification** after applying fixes
- **Fix root causes** not just symptoms
- **Test your fixes** don't assume they work
- **Ask for help** when unsure about fix approach
- **Document new patterns** for future use