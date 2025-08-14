# FEAT-059: Improve Generic Types Usage

## Status
- **Priority**: low

## Overview
Leverage Go 1.18+ generics to reduce code duplication and improve type safety throughout the codebase where appropriate.

## Background
The current codebase makes minimal use of Go generics, missing opportunities to reduce code duplication and improve type safety, particularly in areas like result handling, data structures, and common patterns.

## Requirements
### Functional Requirements
- [ ] Identify opportunities for generic types to reduce duplication
- [ ] Implement generic result and option types
- [ ] Create generic collection utilities where beneficial
- [ ] Maintain backward compatibility with existing APIs

### Non-Functional Requirements
- [ ] Generic usage should improve code maintainability
- [ ] Should not negatively impact compilation time significantly
- [ ] Should improve type safety without complexity overhead

## Design
### Approach
Identify common patterns that can benefit from generics and implement them incrementally without disrupting existing functionality.

### API/Interface
```go
// Generic result type for operations
type Result[T any] struct {
    Value T
    Error error
}

// Generic paginated response
type Page[T any] struct {
    Items      []T
    Total      int
    PageNumber int
    PageSize   int
}

// Generic optional type
type Optional[T any] struct {
    value *T
}
```

### Implementation Notes
- Focus on areas with clear type safety benefits
- Avoid over-engineering with unnecessary generic abstractions
- Consider generic constraints for more specific use cases
- Maintain clear documentation for generic types

## Tasks
### Phase 1: Concrete Opportunity Identification
- [ ] Identify duplicate pattern in coalescer.go: Entry interface could benefit from constraints
- [ ] Review validation package for duplicate error handling patterns
- [ ] Analyze result/error patterns in reader interfaces (calls, sms, contacts)
- [ ] Document specific functions with repeated type patterns

### Phase 2: Targeted Generic Implementation
- [ ] Create generic Result[T] type for operations that return value+error
- [ ] Implement generic Coalescer[T Entry] with type constraints
- [ ] Add generic Optional[T] for nullable values in parsing
- [ ] Create generic Validator[T] interface for different validation types

### Phase 3: Integration and Testing
- [ ] Update 2-3 specific packages to use generic types (start with pkg/coalescer)
- [ ] Add comprehensive tests for generic type implementations
- [ ] Benchmark performance impact vs non-generic versions
- [ ] Update documentation with concrete usage examples from updated packages

## Implementation Targets
**Specific opportunities identified:**
- `pkg/coalescer/types.go`: Entry interface and Coalescer can be more type-safe
- `pkg/calls/types.go` and `pkg/sms/types.go`: Similar timestamp conversion patterns
- Error handling patterns in `pkg/validation`: repeated error wrapping logic
- Reader interfaces: common streaming patterns across calls/sms/contacts

## Testing
### Unit Tests
- Test generic types with various concrete types
- Test type safety and constraint validation
- Test generic utility functions

### Integration Tests
- Test generic types in real usage scenarios
- Verify backward compatibility

### Edge Cases
- Generic type inference scenarios
- Complex constraint handling
- Performance with generic types

## Risks and Mitigations
- **Risk**: Over-engineering with excessive generic abstractions
  - **Mitigation**: Focus on clear wins with measurable benefits
- **Risk**: Increased complexity for minimal benefit
  - **Mitigation**: Only implement generics where they clearly improve the code

## References
- Source: CODE_IMPROVEMENT_REPORT.md item #11

## Notes
This is a lower priority enhancement that should be implemented thoughtfully. Focus on areas where generics provide clear benefits rather than applying them everywhere.