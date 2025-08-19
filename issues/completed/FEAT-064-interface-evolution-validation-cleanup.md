# FEAT-064: Clean up duplicate methods in validation interfaces

## Status
- **Reported**: 2025-08-18
- **Completed**: 2025-08-19
- **Priority**: high
- **Type**: architecture

## Overview
The `RepositoryValidator` interface in `pkg/validation/repository.go` contains duplicate method sets - legacy methods without context alongside context-aware versions, creating confusion about which methods to use and violating the Interface Segregation Principle.

## Requirements
Clean up the validation interfaces to provide clear, single-purpose APIs while maintaining backward compatibility during transition.

## Design
### Current Problem
```go
// pkg/validation/repository.go - actual interface names:
type RepositoryValidator interface {
    // Legacy methods without context
    ValidateRepository() (*Report, error)
    ValidateStructure() []Violation
    
    // Context-aware methods (newer) - note: actual names use "Context" suffix
    ValidateRepositoryContext(ctx context.Context) (*Report, error)
    ValidateStructureContext(ctx context.Context) []Violation
}
```

### Proposed Solution
1. **Deprecate Legacy Methods**: Add deprecation warnings to non-context methods
2. **Interface Evolution Strategy**: Gradual migration over 2 releases
3. **Clean API Design**: Context-aware methods become the primary interface
4. **Migration Tools**: Provide automated migration assistance

## Implementation Plan
### Phase 1: Deprecation (Week 1)
- Add `// Deprecated:` comments to legacy methods
- Update all internal usage to context-aware methods
- Add compilation warnings for deprecated method usage

### Phase 2: Migration Support (Week 2)  
- Create migration guide with examples
- Add automated migration tool/script
- Update documentation to recommend context methods

### Phase 3: Cleanup (6 months after deprecation)
- Remove deprecated methods after 2 release cycles
- Clean up interface definitions
- Update tests to use new interface

## Deprecation Comment Example
```go
// Deprecated: ValidateRepository is deprecated. Use ValidateRepositoryContext instead.
// This method will be removed in v2.0.0 (estimated 6 months).
ValidateRepository() (*Report, error)
```

## Tasks
- [ ] Add deprecation warnings to legacy validation methods
- [ ] Update all internal code to use context-aware methods
- [ ] Create comprehensive migration guide with examples
- [ ] Add linting rules to discourage deprecated method usage
- [ ] Implement automated migration tool/script
- [ ] Update API documentation to reflect preferred methods
- [ ] Add context-aware method tests
- [ ] Plan deprecation timeline and communication

## Testing
### Migration Tests
- Verify all existing functionality works with context methods
- Test context cancellation behavior
- Validate error propagation with context
- Ensure no performance regression

### Compatibility Tests  
- Verify deprecated methods still work during transition
- Test mixed usage scenarios (old + new methods)
- Validate warning messages are clear and helpful

## Acceptance Criteria
- [ ] All legacy methods marked as deprecated with clear migration path
- [ ] Internal codebase uses only context-aware methods
- [ ] Migration guide provides clear examples for all use cases
- [ ] Linting rules prevent new usage of deprecated methods
- [ ] Context cancellation works properly in all validation operations
- [ ] No breaking changes during deprecation period

## Technical Considerations
### Backward Compatibility
- Maintain API compatibility for 2 full releases
- Provide clear deprecation timeline
- Offer automated migration tools

### Context Handling
- Ensure proper context propagation through validation chain
- Handle context cancellation gracefully
- Maintain timeout and deadline support

### Error Handling
- Preserve existing error types and messages
- Add context information to errors where beneficial
- Maintain error chain compatibility

## Related Issues
- Part of architecture cleanup initiative
- Follows Go interface design best practices
- Supports better testing with context control

## Notes
This change improves API clarity while maintaining stability. The context-aware methods provide better control for testing and operation cancellation, which is important for long-running validation operations.