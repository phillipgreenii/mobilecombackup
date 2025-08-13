# FEAT-034: Fix Attachment Extraction During Import

## Status
- **Completed**: YYYY-MM-DD
- **Priority**: high

## Overview
Fix the attachment extraction process that is currently failing during import, leaving binary data embedded in SMS records.

## Background
During import, attachments are completely failing to be extracted from SMS/MMS messages. Testing confirmed:
- Created new repository and imported SMS with attachments - no attachments extracted
- Imported all test data - no attachments found in attachments directory
- Base64 data remains embedded in SMS XML instead of being replaced with relative paths
- No files created in expected hash-based attachments directory structure

This suggests the attachment extraction code path is either not being executed or is silently failing.

## Requirements
### Functional Requirements
- [ ] Extract attachments from MMS messages during import (currently completely failing)
- [ ] Store attachments in hash-based directory structure (attachments/ab/ab123...)
- [ ] Replace base64 data in SMS XML with relative paths to extracted files
- [ ] Update attachment tracking in repository
- [ ] Handle duplicate attachments properly (same hash)
- [ ] Remove base64 data from XML after successful extraction

### Non-Functional Requirements
- [ ] Attachment extraction should not significantly slow import
- [ ] Support large attachment files without memory issues

## Design
### Root Cause Investigation Required
The attachment extraction code appears to exist but is not working:
- `pkg/sms/extractor.go` contains extraction logic
- `pkg/importer/sms.go:137` calls `ExtractAttachmentsFromMMS`
- Code path appears correct but complete failure suggests:
  - AttachmentExtractor not properly initialized
  - Silent failures in extraction process
  - Content type filtering too restrictive
  - File system operations failing

### Investigation Approach
1. **Debug with Test Data**: Use testdata files with known attachments
2. **Add Logging**: Trace execution through extraction code path
3. **Verify Setup**: Check AttachmentExtractor initialization
4. **Content Type Check**: Verify attachment content types are recognized

### Expected Behavior
```xml
<!-- Before extraction -->
<part ct="image/png" data="iVBORw0KGgoAAAANSUhEUgAAABQAAAAUCAYAAACNiR0N..." />

<!-- After extraction -->
<part ct="image/png" path="attachments/ab/ab123456789..." />
```

### API/Interface
```go
// Key extraction interface
type AttachmentExtractor interface {
    ExtractAttachmentsFromMMS(mms *MMS) error
}

// Current code location
// pkg/importer/sms.go:137
err := i.attachmentExtractor.ExtractAttachmentsFromMMS(mms)
```

### Implementation Notes
- Extraction must happen during import before XML is written
- Base64 data must be completely removed from XML after extraction
- File system permissions must allow attachment directory creation
- Hash calculation must be consistent with attachment manager

## Tasks
- [ ] Add debug logging to trace attachment extraction execution path
- [ ] Verify AttachmentExtractor initialization in importer setup
- [ ] Test extraction with testdata/sms-test.xml (contains duck.png attachment)
- [ ] Check content type filtering - ensure image/png, image/jpeg etc. are included
- [ ] Verify file system permissions for attachment directory creation
- [ ] Fix identified bug in extraction process
- [ ] Ensure base64 data is removed from XML after extraction
- [ ] Test with all test data files containing attachments
- [ ] Write comprehensive tests for extraction process

## Testing
### Unit Tests
- Test AttachmentExtractor with single MMS containing image attachment
- Test content type recognition (image/png, image/jpeg, video/mp4)
- Test base64 decoding and hash calculation
- Test XML update after extraction (data field removed, path field added)
- Test duplicate attachment handling (same hash)

### Integration Tests
- Test full import with testdata/sms-test.xml (duck.png attachment)
- Test import creates files in attachments/[hash]/[hash] structure
- Test imported SMS XML contains relative paths instead of base64 data
- Test repository validation passes with extracted attachments

### Edge Cases
- Corrupted base64 data in MMS parts
- Missing attachment directory (should be created)
- Permission errors during file creation
- Very large attachments (memory usage)
- MMS with multiple attachments
- MMS with no attachments (should not error)

## Risks and Mitigations
- **Risk**: Description
  - **Mitigation**: How to handle

## References
- Related features: FEAT-004 (Attachment management infrastructure)
- Code locations: pkg/sms/extractor.go (extraction logic)
- Code locations: pkg/importer/sms.go:137 (extraction call site)
- Test data: testdata/sms-test.xml (contains duck.png on line 56)
- Dependencies: pkg/attachments (hash-based storage)

## Notes
Additional thoughts, questions, or considerations that arise during planning/implementation.
