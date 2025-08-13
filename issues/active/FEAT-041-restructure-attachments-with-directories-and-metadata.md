# FEAT-041: Restructure Attachments with Directories and Metadata

## Status
- **Priority**: high

## Overview
Enhance the attachment storage structure by organizing attachments in directories named by their hash, including proper file extensions based on MIME type, and maintaining metadata files. This improves organization, usability, and provides better attachment management.

## Background
Currently, attachments are stored as single files named with their SHA-256 hash. This approach has several limitations:
- No file extension, making it difficult to identify file types
- No metadata preservation about the original attachment
- Poor user experience when browsing attachments
- Missing contextual information about attachment source

The new structure will use directories for better organization and metadata preservation.

## Requirements
### Functional Requirements
- [ ] Create directory structure using hash as directory name (first 2 chars as subdirectory)
- [ ] Store attachment files with proper extensions based on MIME type
- [ ] Use original filename from SMS when available
- [ ] Generate generic filename with extension when no original name exists
- [ ] Create metadata.yaml file in each attachment directory
- [ ] Preserve all existing attachment functionality
- [ ] Maintain backward compatibility during migration

### Non-Functional Requirements
- [ ] No performance degradation in attachment processing
- [ ] Maintain existing hash-based deduplication
- [ ] Ensure atomic operations during attachment storage
- [ ] Support safe migration from old to new structure

## Design
### Approach
Transform the current flat file structure to a directory-based structure:

**Old Structure:**
```
attachments/e3/e37b1a09d9512c9f8741292195fb27b9f6c0708ca844e4172ee2db589a4261df
```

**New Structure:**
```
attachments/e3/e37b1a09d9512c9f8741292195fb27b9f6c0708ca844e4172ee2db589a4261df/
├── attachment.png (or original filename if available)
└── metadata.yaml
```

### API/Interface
```go
// AttachmentInfo represents attachment metadata
type AttachmentInfo struct {
    Hash         string    `yaml:"hash"`
    OriginalName string    `yaml:"original_name,omitempty"`
    MimeType     string    `yaml:"mime_type"`
    Size         int64     `yaml:"size"`
    CreatedAt    time.Time `yaml:"created_at"`
    SourceMMS    string    `yaml:"source_mms,omitempty"`
}

// AttachmentStorage interface with directory support
type AttachmentStorage interface {
    Store(hash string, data io.Reader, metadata AttachmentInfo) error
    GetPath(hash string) (string, error)
    GetMetadata(hash string) (AttachmentInfo, error)
    Exists(hash string) bool
}
```

### Data Structures
```go
// MIME type to extension mapping
var mimeExtensions = map[string]string{
    "image/png":  "png",
    "image/jpeg": "jpg",
    "image/gif":  "gif",
    "video/mp4":  "mp4",
    "audio/mp3":  "mp3",
    // ... additional mappings
}

// Metadata file structure
type AttachmentMetadata struct {
    Hash         string    `yaml:"hash"`
    OriginalName string    `yaml:"original_name,omitempty"`
    MimeType     string    `yaml:"mime_type"`
    Size         int64     `yaml:"size"`
    CreatedAt    time.Time `yaml:"created_at"`
    SourceMMS    string    `yaml:"source_mms,omitempty"`
}
```

### Implementation Notes
- Use MIME type detection libraries for accurate extension mapping
- Implement graceful fallback for unknown MIME types (use .bin extension)
- Ensure directory creation is atomic and safe for concurrent access
- Update all existing attachment references to use new path structure
- Implement migration strategy for existing installations

## Tasks
- [ ] Design MIME type to extension mapping system
- [ ] Update AttachmentStorage interface for directory-based storage
- [ ] Implement directory creation and file organization logic
- [ ] Add metadata.yaml creation and management
- [ ] Update attachment path resolution throughout codebase
- [ ] Implement migration logic for existing attachments
- [ ] Update all references in SMS/MMS processing
- [ ] Update validation logic for new structure
- [ ] Write comprehensive tests for new structure
- [ ] Update documentation and examples

## Testing
### Unit Tests
- MIME type to extension mapping functionality
- Directory creation and file storage
- Metadata file creation and parsing
- Path resolution with new structure
- Migration logic for existing attachments

### Integration Tests
- End-to-end attachment storage and retrieval
- SMS/MMS processing with new attachment paths
- Repository validation with new structure
- Full import/export cycle

### Edge Cases
- Unknown MIME types and extension fallback
- Duplicate filenames in same directory
- Invalid characters in original filenames
- Large attachment files
- Concurrent attachment storage
- Migration from partially corrupted old structure

## Risks and Mitigations
- **Risk**: Breaking existing installations during migration
  - **Mitigation**: Implement careful migration strategy with rollback capability
- **Risk**: Performance impact of directory operations
  - **Mitigation**: Benchmark and optimize directory creation and file operations
- **Risk**: Filename conflicts with original names
  - **Mitigation**: Implement collision detection and resolution strategy

## References
- Related features: FEAT-012 (Extract Attachments), FEAT-034 (Fix Attachment Extraction)
- Related bugs: BUG-040 (Attachment Validation Failing)
- Code locations: pkg/attachments/, pkg/sms/extractor.go
- Storage structure: .mobilecombackup.yaml specification

## Notes
This enhancement significantly improves the user experience when working with attachments while maintaining the robust hash-based deduplication system. The directory structure provides clear organization and the metadata files preserve important contextual information about each attachment.

The implementation should be done carefully to ensure existing repositories continue to work during and after the migration process.