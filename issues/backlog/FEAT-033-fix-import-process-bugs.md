# FEAT-033: Fix Import Process Bugs

## Status
- **Completed**: YYYY-MM-DD
- **Priority**: high

## Overview
Fix critical bugs in the import process that allow invalid operations and provide incorrect reporting.

## Background
Two critical issues were identified with the import process:
- Import runs even when the repository is invalid (should fail fast)
- During import, yearly summary shows 0 new and 0 duplicates even when totals > 0 (incorrect reporting)

## Requirements
### Functional Requirements
- [ ] Import command validates repository before processing
- [ ] Import fails fast with clear error if repository is invalid
- [ ] Import yearly summary correctly reports new and duplicate counts
- [ ] Import summary matches actual processing results

### Non-Functional Requirements
- [ ] Import validation should be fast (< 1 second)
- [ ] Error messages should be clear and actionable

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
- [ ] Add repository validation to import command startup
- [ ] Fix yearly summary counting logic for new entries
- [ ] Fix yearly summary counting logic for duplicates
- [ ] Add proper error handling for invalid repository state
- [ ] Write tests for repository validation in import
- [ ] Write tests for correct summary reporting
- [ ] Update error messages and documentation

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
