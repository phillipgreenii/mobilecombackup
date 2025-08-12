# FEAT-035: Fix Contacts Processing and Format Issues

## Status
- **Completed**: YYYY-MM-DD
- **Priority**: high

## Overview
Fix multiple issues with contacts processing including format, parsing, matching, and sorting problems.

## Background
Several issues were identified with contacts processing:
- contacts.yaml unprocessed format is incorrect (needs phone_number and contact_names properties)
- Unprocessed entries are not sorted by phone number
- SMS address field may contain multiple phone numbers separated by `~` with corresponding contacts separated by `,`
- Numbers that exist in contacts.yaml are being added to unprocessed instead of being matched

## Requirements
### Functional Requirements
- [ ] Fix contacts.yaml unprocessed format to use phone_number and contact_names properties
- [ ] Sort unprocessed entries by phone number
- [ ] Parse multiple phone numbers from SMS address field (separated by `~`)
- [ ] Parse corresponding contact names (separated by `,`)
- [ ] Properly match existing contacts instead of adding to unprocessed
- [ ] Handle edge cases in phone number and contact parsing

### Non-Functional Requirements
- [ ] Contact matching should be efficient for large contact lists
- [ ] Phone number normalization should handle various formats

## Design
### Approach
Detailed design decisions and rationale.

### API/Interface
```go
// Key interfaces and contracts
```

### Data Structures
```go
// Important types and structures
```

### Implementation Notes
Technical considerations, constraints, and decisions.

## Tasks
- [ ] Fix contacts.yaml unprocessed format structure
- [ ] Add sorting by phone number for unprocessed entries
- [ ] Implement multi-phone number parsing from SMS address
- [ ] Implement multi-contact name parsing
- [ ] Fix contact matching logic to prevent duplicate unprocessed entries
- [ ] Add comprehensive parsing tests for edge cases
- [ ] Test contact matching with various phone number formats
- [ ] Update documentation for new contacts.yaml format

## Testing
### Unit Tests
- Test scenario 1
- Test scenario 2

### Integration Tests
- End-to-end scenario

### Edge Cases
- Edge case handling

## Risks and Mitigations
- **Risk**: Description
  - **Mitigation**: How to handle

## References
- Related features: FEAT-XXX
- Code locations: pkg/module/file.go
- External docs: Link to any external documentation

## Notes
Additional thoughts, questions, or considerations that arise during planning/implementation.
