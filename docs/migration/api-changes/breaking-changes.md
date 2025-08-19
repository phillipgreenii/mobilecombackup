# Breaking Changes and Migration Guide

This document tracks all breaking changes and provides migration guidance for each major version.

## Version 2.0.0 (Planned)

### Validation Interface Cleanup

**Breaking Change**: Removal of deprecated validation methods.

#### Removed Methods
- `ValidateRepository() (*Report, error)`
- `ValidateStructure() []Violation`
- `ValidateManifest() []Violation`
- `ValidateContent() []Violation`
- `ValidateConsistency() []Violation`

#### Migration
Replace with context-aware equivalents:
```go
// Before (v1.x)
report, err := validator.ValidateRepository()

// After (v2.0+)
ctx := context.Background()
report, err := validator.ValidateRepositoryContext(ctx)
```

**Timeline**: 
- v1.1.0: Methods deprecated with warnings
- v2.0.0: Methods removed

**Migration Guide**: See [validation-interfaces.md](../validation-interfaces.md)

---

## Version 1.1.0 (Current)

### Service Layer Interface Abstractions

**Non-Breaking Change**: Added dependency injection support.

#### New Constructors Added
- `NewImporterWithDependencies()`
- `NewCallsImporterWithDependencies()`
- `NewSMSImporter()` (now accepts interface)

#### Migration
Optional migration to use dependency injection:
```go
// Traditional constructor (still supported)
importer := NewImporter(options)

// New dependency injection constructor
importer, err := NewImporterWithDependencies(
    options,
    contactsManager,
    callsReader,
    smsReader,
    attachmentStorage,
)
```

**Benefit**: Better testability and modularity.

### Error Handling Standardization

**Non-Breaking Change**: Enhanced error context and messages.

#### What Changed
- Error messages now include more context
- Error chains properly preserved with `%w`
- Better debugging information

#### Migration
No code changes required, but you may benefit from enhanced error handling:
```go
// Error messages are now more descriptive
_, err := importer.Import()
if err != nil {
    // Error now includes operation context
    log.Printf("Import failed: %v", err)
    
    // Can unwrap to specific error types
    var pathErr *os.PathError
    if errors.As(err, &pathErr) {
        log.Printf("File error: %s", pathErr.Path)
    }
}
```

**Migration Guide**: See [error-handling.md](../error-handling.md)

### Validation Interface Evolution

**Non-Breaking Change**: Added context-aware validation methods.

#### Deprecated Methods
Legacy methods deprecated but still functional:
- `ValidateRepository()`
- `ValidateStructure()`
- And others...

#### New Methods Added
Context-aware versions with same functionality:
- `ValidateRepositoryContext(ctx context.Context)`
- `ValidateStructureContext(ctx context.Context)`
- And others...

#### Migration
Gradual migration recommended:
```go
// Old way (deprecated but works)
report, err := validator.ValidateRepository()

// New way (recommended)
ctx := context.Background()
report, err := validator.ValidateRepositoryContext(ctx)
```

**Migration Guide**: See [validation-interfaces.md](../validation-interfaces.md)

---

## Version 1.0.0 (Baseline)

Initial stable release with core functionality:
- XML-based backup processing
- Hash-based attachment storage
- Repository structure and validation
- CLI interface

---

## Migration Strategy

### Gradual Migration Approach

1. **Update to v1.1.0**: Adopt new features gradually
   - Use deprecated methods initially
   - Migrate to context-aware methods when convenient
   - Adopt dependency injection for new code

2. **Prepare for v2.0.0**: Complete migration before next major version
   - Replace all deprecated method calls
   - Test with strict linting to catch deprecated usage
   - Update documentation and examples

### Automated Migration Tools

#### Linting for Deprecated APIs
Add to `.golangci.yml`:
```yaml
linters:
  enable:
    - staticcheck
linters-settings:
  staticcheck:
    checks: ["all", "SA1019"] # Detect deprecated usage
```

#### Find and Replace Scripts
```bash
# Find deprecated validation method usage
grep -r "ValidateRepository()" --include="*.go" .
grep -r "ValidateStructure()" --include="*.go" .

# Basic replacement (review before applying)
find . -name "*.go" -exec sed -i.bak \
  's/ValidateRepository()/ValidateRepositoryContext(context.Background())/g' {} \;
```

### Testing Migration

#### Validate Backward Compatibility
```go
func TestBackwardCompatibility(t *testing.T) {
    validator := NewRepositoryValidator(testRepoPath)
    
    // Test deprecated methods still work
    report1, err1 := validator.ValidateRepository()
    
    // Test new methods work
    ctx := context.Background()
    report2, err2 := validator.ValidateRepositoryContext(ctx)
    
    // Results should be identical
    if !reflect.DeepEqual(report1, report2) {
        t.Error("Deprecated and new methods produce different results")
    }
}
```

#### Test Error Handling Changes
```go
func TestErrorHandling(t *testing.T) {
    // Test that errors contain expected context
    _, err := failingOperation()
    if err == nil {
        t.Fatal("Expected error")
    }
    
    // Check error contains operation context
    errorMsg := err.Error()
    if !strings.Contains(errorMsg, "operation context") {
        t.Errorf("Error missing context: %s", errorMsg)
    }
    
    // Check error chain preservation
    var pathErr *os.PathError
    if !errors.As(err, &pathErr) {
        t.Error("Expected os.PathError in error chain")
    }
}
```

## Support and Resources

### Documentation
- [Validation Interface Migration](../validation-interfaces.md)
- [Error Handling Migration](../error-handling.md)
- [Architecture Decision Records](../../adr/)

### Migration Assistance
- Review deprecation warnings during compilation
- Use static analysis tools to detect deprecated usage
- Test thoroughly with new methods before v2.0.0 upgrade

### Getting Help
- File issues for migration problems
- Check examples in test code for usage patterns
- Review ADRs for architectural context