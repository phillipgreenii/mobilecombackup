# FEAT-052: Optimize Attachment Storage for Large Files

## Status
- **Priority**: high

## Overview
Current attachment storage reads entire attachments into memory before storing them, which could cause out-of-memory errors with large attachments. This should be optimized to use streaming I/O.

## Background
The current `StoreFromReader` method in `pkg/attachments/storage.go:61-65` loads all attachment data into memory before writing to disk, which is inefficient and potentially problematic for large files.

## Requirements
### Functional Requirements
- [ ] Stream attachments directly from input to disk without loading into memory
- [ ] Maintain existing API compatibility where possible
- [ ] Preserve all current functionality (metadata, hash verification, etc.)

### Non-Functional Requirements
- [ ] Reduce memory usage for large attachment processing
- [ ] Maintain or improve processing speed
- [ ] Handle I/O errors gracefully during streaming

## Design
### Approach
Replace the current approach of reading all data into memory with direct streaming from input reader to output file using buffered I/O.

### API/Interface
```go
func (das *DirectoryAttachmentStorage) StoreFromReader(hash string, data io.Reader, metadata AttachmentInfo) error {
    // Stream directly to disk with proper error handling
}
```

### Implementation Notes
- Use `io.CopyBuffer` with appropriate buffer size (32KB recommended)
- Ensure proper cleanup if streaming fails partway through
- Consider atomic file operations (write to temp file, then rename)

## Tasks
- [ ] Design review of streaming approach
- [ ] Implement streaming storage method
- [ ] Add proper error handling for partial writes
- [ ] Write tests for large file handling
- [ ] Add benchmarks comparing old vs new approach
- [ ] Update documentation

## Testing
### Unit Tests
- Test streaming with various file sizes
- Test error conditions during streaming
- Test metadata preservation

### Integration Tests
- End-to-end attachment processing with large files
- Memory usage validation

### Edge Cases
- Network interruption during streaming
- Disk space exhaustion during write
- Permission errors during file creation

## Risks and Mitigations
- **Risk**: Breaking API compatibility
  - **Mitigation**: Maintain existing method signatures, change implementation only
- **Risk**: Data corruption during streaming errors
  - **Mitigation**: Use atomic file operations (temp file + rename)

## References
- Code locations: pkg/attachments/storage.go:61-65
- Source: CODE_IMPROVEMENT_REPORT.md item #4

## Notes
This improvement will significantly reduce memory usage for processing large MMS attachments, making the tool more suitable for handling large backup files.