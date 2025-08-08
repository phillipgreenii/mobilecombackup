# FEAT-004: Read Attachments from Repository

## Status
- **Completed**: 2025-01-27
- **Priority**: high

## Overview
Implement functionality to read and manage attachments stored in the repository. This feature provides access to attachment files that are referenced by SMS/MMS messages, stored in a content-addressed structure, and validates the attachment directory structure.

## Background
Attachments are stored in the repository under `attachments/` using a hash-based directory structure (e.g., `attachments/ab/ab54363e39`). These files are referenced by MMS messages and need to be accessible for validation, export, and analysis purposes. The structure uses the first two characters of the hash as a subdirectory to avoid too many files in a single directory.

## Requirements
### Functional Requirements
- [ ] Read attachment files from `attachments/[prefix]/[hash]` structure
- [ ] Verify attachment exists given a reference path
- [ ] Retrieve attachment content by hash
- [ ] List all attachments in repository
- [ ] Get attachment metadata (size, hash)
- [ ] Validate hash matches actual file content
- [ ] Validate attachment files follow the `[2-char-prefix]/[full-hash]` structure
- [ ] Verify attachment directory names are exactly 2 lowercase hex characters
- [ ] Support for finding orphaned attachments (not referenced by any SMS)
- [ ] Track which attachments are referenced vs orphaned

### Non-Functional Requirements
- [ ] Efficient directory traversal for large attachment stores
- [ ] Minimal memory usage when listing attachments
- [ ] Fast lookup by hash
- [ ] Proper error handling for missing/corrupted files

## Design
### Approach
Create an attachment reader that:
1. Navigates the hash-based directory structure
2. Provides efficient lookup by hash
3. Can enumerate all attachments
4. Validates file integrity
5. Supports finding unreferenced attachments

### API/Interface
```go
// Attachment represents a stored attachment file
type Attachment struct {
    Hash     string
    Path     string      // Relative path: attachments/ab/ab54363e39
    Size     int64
    Exists   bool
}

// AttachmentReader reads attachments from repository
type AttachmentReader interface {
    // GetAttachment retrieves attachment info by hash
    GetAttachment(hash string) (*Attachment, error)
    
    // ReadAttachment reads the actual file content
    ReadAttachment(hash string) ([]byte, error)
    
    // AttachmentExists checks if attachment exists
    AttachmentExists(hash string) (bool, error)
    
    // ListAttachments returns all attachments in repository
    ListAttachments() ([]*Attachment, error)
    
    // StreamAttachments streams attachment info for memory efficiency
    StreamAttachments(callback func(*Attachment) error) error
    
    // VerifyAttachment checks if file content matches its hash
    VerifyAttachment(hash string) (bool, error)
    
    // GetAttachmentPath returns the expected path for a hash
    GetAttachmentPath(hash string) string
    
    // FindOrphanedAttachments returns attachments not referenced by any messages
    // Requires a set of referenced attachment hashes from SMS reader
    FindOrphanedAttachments(referencedHashes map[string]bool) ([]*Attachment, error)
    
    // ValidateAttachmentStructure validates the directory structure
    ValidateAttachmentStructure() error
}

// AttachmentStats provides statistics about attachments
type AttachmentStats struct {
    TotalCount      int
    TotalSize       int64
    OrphanedCount   int  // Not referenced by any SMS
    CorruptedCount  int  // Hash mismatch
}
```

### Implementation Notes
- Hash format: lowercase hexadecimal SHA-256
- Directory structure: first 2 chars as subdirectory
- No file extension needed (content type in MMS)
- Support both full hash and path lookups
- Cache directory listings for performance

## Tasks
- [x] Define Attachment struct and related types
- [x] Create AttachmentReader interface
- [x] Implement hash-based path resolution
- [x] Add directory traversal logic
- [x] Implement content reading with verification
- [x] Add directory structure validation
- [x] Implement orphaned attachment detection
- [x] Add streaming API for large repositories
- [x] Create attachment statistics collector
- [x] Write unit tests
- [x] Add integration tests
- [x] Performance optimization for large stores

## Testing
### Unit Tests
- Path generation from hash
- Hash validation (format and length)
- Directory structure navigation
- Content hash verification
- Missing file handling

### Integration Tests
- Read attachments from sample repository
- List all attachments performance test
- Verify large attachment handling
- Cross-reference with SMS messages

### Edge Cases
- Empty attachments directory
- Invalid hash formats
- Missing subdirectories
- Corrupted files (hash mismatch)
- Very large attachment files
- Symbolic links or special files

## Risks and Mitigations
- **Risk**: Large number of files impacting performance
  - **Mitigation**: Streaming APIs, directory caching
- **Risk**: Corrupted attachments
  - **Mitigation**: Hash verification on read
- **Risk**: Storage of sensitive content
  - **Mitigation**: Document security considerations

## References
- Related features: FEAT-003 (SMS Reading)
- Specification: See "Repository/attachments" section
- Code location: pkg/attachments/reader.go (to be created)

## Notes
- Attachments are content-addressed (hash-based)
- No reference counting needed (append-only system)
- Consider adding MIME type detection in future
- May need export functionality later