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
- [ ] Attachment extraction must only process whitelisted binary file types
- [ ] Text-based content must remain inline and not be extracted as separate files
- [ ] Unknown content types must be rejected with clear logging
- [ ] System must use explicit content type whitelist for extraction decisions
- [ ] Rejections should be logged with content type and reason for manual review

### Acceptance Criteria
- **Given** an MMS with image/jpeg content
- **When** attachment processing occurs
- **Then** the image should be extracted to attachments directory

- **Given** an MMS with text/plain content  
- **When** attachment processing occurs
- **Then** content should remain inline and no file should be created

- **Given** an MMS with unknown content type "application/custom"
- **When** attachment processing occurs
- **Then** entry should be rejected with logged reason for manual review

- **Given** an MMS with malformed/missing content type
- **When** attachment processing occurs
- **Then** entry should be rejected with clear explanation in logs

### Non-Functional Requirements
- [ ] Performance should be improved by skipping unnecessary text processing
- [ ] Storage efficiency should be maintained by avoiding duplicate text storage
- [ ] Rejected entries must be clearly logged for future analysis
- [ ] No backward compatibility concerns (no existing repositories to consider)

## Design
### Approach
Implement strict content type filtering using an explicit whitelist of supported binary types. Reject any content that is not explicitly whitelisted, with clear logging for manual review and future decision-making.

### API/Interface
```go
// Add content type filtering to attachment processing
type AttachmentProcessor interface {
    ShouldExtract(contentType string) (bool, string) // bool for extract, string for rejection reason
    ExtractAttachment(attachment Attachment) error
}
```

### Data Structures
```go
// Explicit whitelist of binary content types for extraction
var BinaryContentTypes = map[string]bool{
    // Common SMS image types
    "image/jpeg": true,
    "image/jpg":  true,  // Some systems use this variant
    "image/png":  true,
    "image/gif":  true,
    "image/bmp":  true,
    "image/webp": true,
}

// Text content types that should remain inline (for logging purposes)
var TextContentTypes = map[string]bool{
    "text/plain": true,
    "text/html":  true,
    "text/x-vCard": true,
    "application/smil": true,
}

type ContentDecision struct {
    ShouldExtract bool
    Reason        string
    ContentType   string
}
```

### Implementation Notes
- Identify current attachment extraction logic in `pkg/sms/extractor.go`
- Implement strict whitelist checking before extraction
- Reject unknown types with detailed logging including content type and size
- Log text types as "skipped - text content" 
- Log unknown types as "rejected - unknown type" for manual review
- Handle edge cases by rejecting with clear explanations

### Code Locations
- Primary implementation: `pkg/sms/extractor.go` - ExtractAttachmentsFromMMS method
- Test file: `pkg/sms/extractor_test.go`
- Content type logic: New function `shouldExtractContentType(contentType string)`

## Tasks
- [ ] Locate current attachment extraction code in pkg/sms/extractor.go
- [ ] Implement shouldExtractContentType() function with explicit whitelist
- [ ] Add comprehensive logging for all rejection reasons
- [ ] Modify ExtractAttachmentsFromMMS to check content type before processing
- [ ] Add logging for skipped text content and rejected unknown types
- [ ] Write tests covering whitelist, text rejection, and unknown type rejection
- [ ] Add test cases for edge cases (malformed content types, missing headers)

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
- **Missing content type**: Reject with reason "missing content type header"
- **Empty/whitespace content type**: Reject with reason "empty content type"
- **Malformed content type**: Reject with reason "malformed content type: [value]"
- **Unknown content types**: Reject with reason "unknown content type: [type]" 
- **Binary data with text MIME type**: Reject with reason "conflicting type/data" (handle later)
- **Rich text formats**: Reject unknown text variants for manual review

All edge cases should be rejected with clear logging for future analysis and decision-making.

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
This change will improve system efficiency and ensure proper separation between inline text content and extractable binary attachments. The conservative whitelist approach with rejection logging will allow for iterative improvement based on real-world data patterns.

Unknown content types will be rejected and logged for manual review, allowing for informed decisions about whether to add them to the whitelist in future updates.