# FEAT-025: Validate Attachment Files

## Status
- **Completed**: Not yet started
- **Priority**: medium

## Overview
Add validation capabilities to verify that attachment files referenced in SMS/MMS messages actually exist in the repository, have correct content hashes, and match their expected file formats. This feature ensures data integrity and helps identify corrupted, missing, or misidentified attachments.

## Background
During SMS/MMS import, attachments are extracted and stored using their content hash. However, there's currently no way to verify that:
1. Referenced attachment files actually exist in the repository
2. The file content matches the expected hash (no corruption)
3. All attachments in the repository are referenced by at least one message
4. File content matches the declared MIME type (e.g., PNG files have PNG headers)

This feature was identified during FEAT-012 planning as a separate concern from attachment extraction.

## Requirements
### Functional Requirements
- [ ] Validate all attachment references in SMS/MMS messages point to existing files
- [ ] Verify file content matches the hash in the filename
- [ ] Verify file format matches declared MIME type by checking file headers/magic bytes
- [ ] Report missing attachments with context (which messages reference them)
- [ ] Report corrupted attachments (hash mismatch)
- [ ] Report format mismatches (e.g., JPEG data with PNG MIME type)
- [ ] Optionally check for orphaned attachments (files not referenced by any message)
- [ ] Support both full validation and quick validation modes

### Non-Functional Requirements
- [ ] Performance: Process 10,000+ attachments in under 60 seconds
- [ ] Memory efficiency: Stream processing to handle large repositories
- [ ] Clear error reporting with actionable information

## Design
### Approach
Extend the existing validate subcommand to include attachment file validation. Use the AttachmentReader for file operations and SMSReader to get attachment references.

### API/Interface
```go
// Add to existing validation types
type AttachmentValidation struct {
    Missing         []MissingAttachment
    Corrupted       []CorruptedAttachment
    FormatMismatch  []FormatMismatchAttachment
    Orphaned        []OrphanedAttachment  // Optional check
}

type MissingAttachment struct {
    Hash         string
    ReferencedBy []MessageReference  // Which messages reference this
}

type CorruptedAttachment struct {
    Path         string
    ExpectedHash string
    ActualHash   string
    Error        string
}

type FormatMismatchAttachment struct {
    Path         string
    DeclaredType string  // MIME type from SMS/MMS
    ActualType   string  // Detected from file content
    Error        string
}

type MessageReference struct {
    Year    int
    Type    string  // "SMS" or "MMS"
    ID      string  // Message ID or unique identifier
}

// Validation modes
type ValidationMode int
const (
    QuickValidation ValidationMode = iota  // Check existence only
    FullValidation                         // Check existence, hash, and format
)
```

### Implementation Notes
- Use streaming to handle large numbers of attachments
- Calculate hashes only in full validation mode
- Cross-reference with SMS reader to find which messages use each attachment
- Report results in both human-readable and JSON formats
- File format detection using magic bytes:
  - PNG: `89 50 4E 47 0D 0A 1A 0A`
  - JPEG: `FF D8 FF`
  - GIF: `47 49 46 38`
  - MP4: `00 00 00 XX 66 74 79 70` (where XX is box size)
  - PDF: `25 50 44 46`

## Tasks
- [ ] Define validation types and interfaces
- [ ] Implement attachment reference collection from SMS/MMS messages
- [ ] Add quick validation mode (existence check only)
- [ ] Add full validation mode (existence + hash verification)
- [ ] Implement file format detection using magic bytes
- [ ] Add format validation against declared MIME types
- [ ] Implement orphaned attachment detection (optional flag)
- [ ] Add validation to existing validate subcommand
- [ ] Write comprehensive tests
- [ ] Update documentation and validate command help

## Testing
### Unit Tests
- Validate with missing attachments
- Validate with corrupted attachments (wrong hash)
- Validate with format mismatches (PNG data with JPEG MIME type)
- Validate with orphaned attachments
- Test both quick and full validation modes
- Test with empty attachment directory
- Test file format detection for common types

### Integration Tests
- Full repository validation with mixed issues
- Performance test with 1000+ attachments
- Test JSON output format

### Edge Cases
- Attachment referenced multiple times
- Very large attachment files (100MB+)
- Permission errors reading files
- Symbolic links in attachment directory
- Files with misleading extensions (e.g., .png file containing JPEG data)
- Corrupted file headers
- Unknown/unsupported file formats

## Risks and Mitigations
- **Risk**: Hash calculation for large files may be slow
  - **Mitigation**: Implement quick mode that skips hash verification; show progress
- **Risk**: Memory usage with many attachments
  - **Mitigation**: Use streaming; process in batches if needed

## References
- Related features: FEAT-012 (attachment extraction), FEAT-004 (attachment reader)
- Depends on: FEAT-003 (SMS reader for getting references)
- Code locations: pkg/attachments/, cmd/mobilecombackup/cmd/validate.go

## Notes
- Consider adding a --fix flag in future to remove orphaned attachments
- Hash verification is important for detecting disk corruption
- Quick mode useful for routine checks; full mode for thorough validation