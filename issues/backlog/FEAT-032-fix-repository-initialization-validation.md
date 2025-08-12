# FEAT-032: Fix Repository Initialization and Validation

## Status
- **Completed**: YYYY-MM-DD
- **Priority**: high

## Overview
Fix critical issues with repository initialization and validation that prevent proper repository setup and file tracking.

## Background
Multiple issues were identified with repository initialization and validation:
- `init` command creates incomplete repository structure (missing files.yaml and files.yaml.sha256)
- `validate --autofix` creates files.yaml but not files.yaml.sha256
- .mobilecombackup.yaml is not tracked in files.yaml
- `info` command on empty repo doesn't show entity counts (should show 0s)
- Files.yaml validation rules don't align with generation logic
- Directory path discrepancies causing validation failures

## Requirements
### Functional Requirements
- [ ] `init` command creates complete repository structure including files.yaml and files.yaml.sha256
- [ ] `validate --autofix` creates both files.yaml and files.yaml.sha256
- [ ] .mobilecombackup.yaml is properly tracked in files.yaml
- [ ] `info` command shows counts for all entity types (0 for empty repo)
- [ ] Files.yaml validation rules align with generation logic
- [ ] Consistent directory paths between validation and generation

### Non-Functional Requirements
- [ ] Repository initialization should be atomic (all-or-nothing)
- [ ] Validation should be fast and comprehensive

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
- [ ] Analyze current init command implementation
- [ ] Fix init command to create complete repository structure
- [ ] Fix validate --autofix to create both files.yaml and files.yaml.sha256
- [ ] Add .mobilecombackup.yaml to files.yaml tracking
- [ ] Fix info command to show all entity counts
- [ ] Align files.yaml validation with generation logic
- [ ] Fix directory path consistency issues
- [ ] Write comprehensive tests for all fixes
- [ ] Update documentation

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
