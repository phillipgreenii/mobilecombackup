# FEAT-034: Fix Attachment Extraction During Import

## Status
- **Completed**: YYYY-MM-DD
- **Priority**: high

## Overview
Fix the attachment extraction process that is currently failing during import, leaving binary data embedded in SMS records.

## Background
During import, attachments are not being extracted from SMS/MMS messages. The binary data remains embedded in the SMS records instead of being extracted to the attachments directory with proper hash-based naming.

## Requirements
### Functional Requirements
- [ ] Extract attachments from MMS messages during import
- [ ] Store attachments in hash-based directory structure
- [ ] Replace attachment data in SMS with relative paths
- [ ] Update attachment tracking in repository
- [ ] Handle duplicate attachments properly (same hash)

### Non-Functional Requirements
- [ ] Attachment extraction should not significantly slow import
- [ ] Support large attachment files without memory issues

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
- [ ] Investigate current attachment extraction logic
- [ ] Fix attachment extraction in SMS/MMS processing
- [ ] Ensure proper integration with AttachmentManager
- [ ] Test attachment extraction with real MMS data
- [ ] Verify hash-based storage works correctly
- [ ] Test duplicate attachment handling
- [ ] Write comprehensive tests for extraction process

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
