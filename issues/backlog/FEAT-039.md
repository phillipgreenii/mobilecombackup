# FEAT-039: Limit Attachment Extraction to Binary Types Only

## Status
- **Reported**: 2025-08-13
- **Priority**: medium

## Overview
Attachment extraction should only process binary file types and skip text-based content. Currently, the system may attempt to extract text types as attachments when they should be handled differently.

## Background
Mobile backup files contain various types of data including text messages with embedded content and binary attachments like images, videos, and documents. Text content should remain inline with messages, while only binary attachments should be extracted to separate files for proper organization and storage efficiency.

## Requirements
### Functional Requirements
- [ ] Attachment extraction must only process binary file types (images, videos, documents, etc.)
- [ ] Text-based content should remain inline and not be extracted as separate files
- [ ] System should properly identify content types to determine extraction eligibility
- [ ] Clear logging should indicate when content is skipped due to being text-based

### Non-Functional Requirements
- [ ] Performance should be improved by skipping unnecessary text processing
- [ ] Storage efficiency should be maintained by avoiding duplicate text storage
- [ ] Backward compatibility with existing extracted attachments

## Design
### Approach
Implement content type detection to filter out text-based content before attempting attachment extraction. Use MIME type detection or file extension analysis to determine if content is binary.

### API/Interface
```go
// Add content type filtering to attachment processing
type AttachmentProcessor interface {
    ShouldExtract(contentType string, data []byte) bool
    ExtractAttachment(attachment Attachment) error
}
```

### Data Structures
```go
// Define supported binary content types
var BinaryContentTypes = map[string]bool{
    "image/jpeg": true,
    "image/png":  true,
    "video/mp4":  true,
    "application/pdf": true,
    // Add other binary types as needed
}
```

### Implementation Notes
- Identify current attachment extraction logic
- Add content type detection before extraction
- Create whitelist of binary MIME types that should be extracted
- Update extraction logic to skip text types
- Ensure proper error handling and logging

## Tasks
- [ ] Locate current attachment extraction code
- [ ] Implement content type detection function
- [ ] Create binary content type whitelist
- [ ] Modify extraction logic to check content type before processing
- [ ] Add logging for skipped text content
- [ ] Write tests for content type filtering
- [ ] Update any related documentation

## Testing
### Unit Tests
- Test content type detection for various file types
- Test extraction skipping for text-based content
- Test extraction proceeding for binary content
- Test edge cases with unknown or missing content types

### Integration Tests
- End-to-end test with mixed content types in backup files
- Verify only binary attachments are extracted to filesystem
- Verify text content remains inline in message data

### Edge Cases
- Handle missing or invalid content type information
- Handle edge cases like rich text formats
- Handle malformed or corrupted content type data

## Risks and Mitigations
- **Risk**: Over-filtering may skip legitimate binary content
  - **Mitigation**: Comprehensive testing with real backup data, conservative whitelist approach
- **Risk**: Content type detection may be unreliable
  - **Mitigation**: Use multiple detection methods (MIME type + file signature analysis)

## References
- Related features: FEAT-012 (Extract Attachments), FEAT-034 (Fix Attachment Extraction)
- Code locations: pkg/attachments/ directory
- Related issues: Attachment processing functionality

## Notes
This change should improve system efficiency and ensure proper separation between inline text content and extractable binary attachments. Consider backwards compatibility with existing repositories that may have incorrectly extracted text files.