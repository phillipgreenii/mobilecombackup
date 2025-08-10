# FEAT-025: Add Format Validation to Attachment Files

## Status
- **Completed**: 2025-08-10
- **Priority**: medium

## Overview
Extend the existing AttachmentsValidator to verify that attachment files match their expected file formats by checking magic bytes. This enhancement adds format validation to the current file validation process, ensuring that file content matches declared MIME types and rejecting unknown file formats.

## Background
The existing AttachmentsValidator currently validates file placement, naming, and hash verification. However, it does not verify that file content matches the expected format based on MIME types declared in SMS/MMS messages. This can lead to situations where:
- A file declared as image/png actually contains JPEG data
- Corrupted files with invalid headers pass validation
- Unknown or potentially dangerous file formats are accepted

This feature extends the existing validation framework to add magic byte checking for file format verification.

## Requirements
### Functional Requirements
- [ ] Extend AttachmentsValidator to check file format using magic bytes
- [ ] Verify file format matches declared MIME type from SMS/MMS metadata
- [ ] Report format mismatches (e.g., JPEG data with PNG MIME type)
- [ ] Reject unknown or unrecognized file formats during validation
- [ ] Add format validation to existing file validation process (always enabled)
- [ ] Support common mobile backup formats (PNG, JPEG, GIF, MP4, PDF)
- [ ] Provide clear error messages indicating expected vs actual format

### Non-Functional Requirements
- [ ] Performance: Minimal overhead to existing validation (< 5% slower)
- [ ] Memory efficiency: Read only file headers for format detection
- [ ] Clear error reporting with file path, expected format, and detected format
- [ ] Integrate seamlessly with existing validation framework

## Design
### Approach
Extend the existing AttachmentsValidator implementation to include magic byte checking during the file validation process. This leverages the existing validation framework and adds format checking as an additional validation step.

### API/Interface
```go
// Extend existing AttachmentsValidatorImpl to include format checking
type AttachmentsValidatorImpl struct {
    repositoryRoot   string
    attachmentReader attachments.AttachmentReader
    smsReader        sms.SMSReader  // Add to get MIME type information
}

// Add new violation types to existing ViolationType enum
const (
    // ... existing types ...
    FormatMismatch ViolationType = "format_mismatch"
    UnknownFormat  ViolationType = "unknown_format"
)

// Extend ValidateAttachmentIntegrity to include format checking
// The method will now also verify file formats match expected MIME types

// Add helper function to detect file format
func detectFileFormat(filePath string) (mimeType string, err error) {
    // Read first 512 bytes for format detection
    // Return detected MIME type or error
}

// Known format signatures
var formatSignatures = []struct {
    mimeType  string
    magic     []byte
    offset    int
}{
    {"image/png", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, 0},
    {"image/jpeg", []byte{0xFF, 0xD8, 0xFF}, 0},
    {"image/gif", []byte{0x47, 0x49, 0x46, 0x38}, 0},
    {"video/mp4", []byte{0x66, 0x74, 0x79, 0x70}, 4}, // 'ftyp' at offset 4
    {"application/pdf", []byte{0x25, 0x50, 0x44, 0x46}, 0}, // '%PDF'
}

// Add method to get MIME types for attachments from SMS/MMS data
func (v *AttachmentsValidatorImpl) getAttachmentMimeTypes() (map[string]string, error) {
    // Returns map of hash -> MIME type from SMS/MMS messages
}
```

### Implementation Notes
- Integrate format checking into existing `ValidateAttachmentIntegrity()` method
- Add format check in the loop after hash verification (line 105-139)
- Read only first 512 bytes of files for efficient format detection
- Cross-reference with SMS/MMS data to get expected MIME types
- Add violations using existing ValidationViolation structure
- Format validation is always enabled (no CLI flags needed)
- Reject files with unknown formats as validation failures
- File format detection patterns:
  - PNG: Check for exact signature at offset 0
  - JPEG: Check for FFD8FF at offset 0
  - GIF: Check for GIF87a or GIF89a at offset 0
  - MP4: Check for 'ftyp' at offset 4-7
  - PDF: Check for %PDF at offset 0
- Integration points:
  - Modify `NewAttachmentsValidator` to accept SMSReader parameter
  - Add format detection after line 137 in `ValidateAttachmentIntegrity`
  - Use existing violation reporting mechanism

## Tasks
- [x] Add `detectFileFormat()` function in `pkg/validation/attachments.go`
- [x] Add `FormatMismatch` and `UnknownFormat` to ViolationType constants
- [x] Modify `AttachmentsValidatorImpl` struct to include `smsReader` field
- [x] Update `NewAttachmentsValidator()` to accept SMSReader parameter
- [x] Add `getAttachmentMimeTypes()` method to retrieve MIME types from SMS/MMS
- [x] Extend `ValidateAttachmentIntegrity()` to include format checking after hash verification
- [x] Update repository validator to pass SMSReader to AttachmentsValidator
- [x] Write unit tests for format detection function
- [x] Update existing attachment validator tests to include format validation
- [x] Update validate command to create SMSReader for attachment validation

## Testing
### Unit Tests
- Test format detection for all supported types (PNG, JPEG, GIF, MP4, PDF)
- Test format mismatch detection (PNG data with JPEG MIME type)
- Test unknown format rejection
- Test corrupted file headers
- Test integration with existing validation violations
- Test performance impact on large repositories
- Test edge cases in magic byte detection

### Integration Tests
- Full repository validation including format checking
- Test with real SMS/MMS data containing various attachment types
- Verify format violations appear in validation results
- Performance test showing minimal overhead
- Test JSON output includes new violation types

### Edge Cases
- Files too small to contain valid headers
- Truncated files with partial headers
- Files with valid headers but corrupted content
- Multiple valid format signatures (e.g., TIFF can start like JPEG)
- Zero-byte files
- Files with read permissions but corrupted filesystem metadata
- Attachment referenced by multiple messages with different MIME types

## Risks and Mitigations
- **Risk**: Format detection may slow down validation
  - **Mitigation**: Read only first 512 bytes; cache results if needed
- **Risk**: False positives in format detection
  - **Mitigation**: Use conservative detection; handle ambiguous cases
- **Risk**: Breaking existing validation behavior
  - **Mitigation**: Add format checking as additional step, preserve existing checks

## References
- Related features: FEAT-012 (attachment extraction), FEAT-004 (attachment reader)
- Depends on: FEAT-003 (SMS reader for MIME types), existing AttachmentsValidator
- Code locations: pkg/attachments/validator.go, pkg/attachments/format.go
- Extends: Current attachment validation framework

## Notes
- Format validation is always enabled - no configuration needed
- Unknown formats are rejected to ensure repository integrity
- This enhances existing validation without changing the validation API
- Future enhancement: Support additional formats as needed