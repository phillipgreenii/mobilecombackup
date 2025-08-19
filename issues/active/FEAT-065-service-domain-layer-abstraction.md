# FEAT-065: Introduce service-domain layer abstractions

## Status
- **Reported**: 2025-08-18
- **Completed**: 
- **Priority**: medium
- **Type**: architecture

## Overview
The importer service layer (`pkg/importer`) directly imports and depends on domain packages (`pkg/calls`, `pkg/sms`, `pkg/contacts`, `pkg/attachments`), creating tight coupling and violating the Dependency Inversion Principle. This makes testing difficult and prevents easy replacement of implementations.

## Requirements
Introduce abstraction interfaces between service and domain layers to reduce coupling, improve testability, and enable dependency injection.

## Design
### Current Problem
```go
// pkg/importer/importer.go lines 10-14
import (
    "github.com/phillipgreen/mobilecombackup/pkg/calls"
    "github.com/phillipgreen/mobilecombackup/pkg/sms"
    "github.com/phillipgreen/mobilecombackup/pkg/contacts"
    "github.com/phillipgreen/mobilecombackup/pkg/attachments"
)
```

### Proposed Solution
Create domain interfaces that the service layer depends on, with concrete implementations in domain packages.

```go
// Package structure clarification:
// - Leverage existing Reader interfaces in pkg/calls/reader.go and pkg/sms/reader.go  
// - Extend rather than replace existing patterns
// - Create pkg/service/interfaces.go for service-layer abstractions

// pkg/service/interfaces.go
type CallsReader interface {
    calls.Reader  // Embed existing Reader interface
    // Service-specific extensions if needed
}

type SMSReader interface {
    sms.Reader  // Embed existing Reader interface  
    // Service-specific extensions if needed
}

type ContactsManager interface {
    LoadContacts() error
    GetContactByNumber(number string) (string, bool)
    AddUnprocessedContacts(entries []ContactEntry) error
    SaveContacts() error
}

type AttachmentManager interface {
    StoreAttachment(hash string, data []byte, metadata AttachmentMetadata) error
    GetAttachment(hash string) (AttachmentInfo, error)
    AttachmentExists(hash string) bool
}
```

## Implementation Plan
### Phase 1: Interface Definition (Week 1)
- Create `pkg/service/interfaces.go` with abstraction interfaces
- Build upon existing Reader interfaces in pkg/calls and pkg/sms
- Define common types and data structures
- Ensure interfaces cover all current usage patterns

### Phase 2: Implementation Adaptation (Week 2)
- Update domain packages to implement interfaces
- Create adapter patterns where needed
- Ensure interface implementations are complete

### Phase 3: Service Layer Refactoring (Week 3)
- Update importer to depend on interfaces instead of concrete types
- Implement dependency injection for service components
- Update service constructors to accept interface dependencies

### Phase 4: Testing and Validation (Week 4)
- Create mock implementations for testing
- Update all tests to use dependency injection
- Validate no functionality regression

## Tasks
- [ ] Design domain interfaces for all service dependencies
- [ ] Create pkg/service/interfaces.go with complete interface definitions
- [ ] Update domain packages to implement new interfaces
- [ ] Refactor pkg/importer to use interfaces instead of concrete types
- [ ] Implement dependency injection in service constructors
- [ ] Create mock implementations for testing
- [ ] Update all service tests to use dependency injection
- [ ] Add integration tests to verify interface compliance
- [ ] Update documentation with new architecture patterns

## Testing
### Interface Compliance Tests
- Verify all domain implementations satisfy interfaces
- Test interface method signatures and behavior
- Validate error handling across interface boundaries

### Dependency Injection Tests
- Test service behavior with mock implementations
- Verify proper interface usage in service layer
- Test error propagation through interface calls

### Integration Tests
- Verify real implementations work through interfaces
- Test complete workflow with interface abstractions
- Validate no performance regression

## Acceptance Criteria
- [ ] Complete interface definitions for all domain dependencies
- [ ] All domain packages implement required interfaces
- [ ] Service layer depends only on interfaces, not concrete types
- [ ] Dependency injection available for all service components
- [ ] Mock implementations available for all interfaces
- [ ] All tests use dependency injection where appropriate
- [ ] No functionality or performance regression
- [ ] Clear documentation of new architecture patterns

## Technical Considerations
### Interface Design
- Keep interfaces focused and single-purpose
- Avoid over-abstraction - only abstract what varies
- Ensure interfaces are easy to mock and test

### Dependency Injection
- Use constructor injection for required dependencies
- Consider interface composition for complex dependencies
- Maintain clear object lifetimes and ownership

### Performance Impact
- Interfaces should have minimal performance overhead
- Avoid unnecessary allocations or indirection
- Profile critical paths to ensure no regression

## Migration Strategy
### Backward Compatibility
- Maintain existing public APIs during transition
- Use adapter patterns to bridge old and new interfaces
- Provide migration examples for external users

### Incremental Rollout
- Start with one domain interface (e.g., ContactsManager)
- Gradually migrate other domains
- Complete service layer refactoring last

## Related Issues
- Part of architecture cleanup initiative
- Enables better testing practices
- Supports future extensibility and modularity

## Notes
This change significantly improves the architecture by reducing coupling and enabling better testing. The interfaces should be designed based on actual usage patterns in the service layer, not theoretical abstractions.