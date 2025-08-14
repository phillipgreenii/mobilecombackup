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

### Technical Design
**Performance Requirements:**
- Support files up to 2GB with < 100MB memory usage
- Maximum buffer size: 32KB for streaming operations
- Performance target: < 10% degradation compared to current implementation

**Streaming Implementation Approach:**
```go
func (das *DirectoryAttachmentStorage) StoreFromReader(hash string, data io.Reader, metadata AttachmentInfo) error {
    // 1. Create temp file with proper permissions
    tempFile := filepath.Join(tempDir, fmt.Sprintf("attachment-%s.tmp", uuid.New()))
    file, err := os.Create(tempFile)
    if err != nil {
        return err
    }
    defer os.Remove(tempFile) // cleanup on error
    
    // 2. Create hash writer for verification
    hasher := sha256.New()
    multiWriter := io.MultiWriter(file, hasher)
    
    // 3. Stream with fixed buffer size
    _, err = io.CopyBuffer(multiWriter, data, make([]byte, 32*1024))
    if err != nil {
        return err
    }
    
    // 4. Verify hash matches expected
    calculatedHash := hex.EncodeToString(hasher.Sum(nil))
    if calculatedHash != hash {
        return fmt.Errorf("hash mismatch: expected %s, got %s", hash, calculatedHash)
    }
    
    // 5. Atomic rename to final location
    finalPath := filepath.Join(dirPath, filename)
    return os.Rename(tempFile, finalPath)
}
```

### Implementation Notes
- Use temp files with atomic rename for consistency
- Calculate hash during streaming using io.MultiWriter
- 32KB buffer size balances memory usage and performance
- Proper cleanup with defer statements for all error paths

## Tasks
### Phase 1: Core Implementation
- [ ] Create custom error types: `ErrHashMismatch`, `ErrPartialWrite`, `ErrDiskFull`
- [ ] Implement streaming StoreFromReader with hash verification
- [ ] Add temp file management with atomic operations
- [ ] Ensure proper cleanup with defer blocks for all error paths
- [ ] Add progress callback support (optional for large files)

### Phase 2: Testing and Validation
- [ ] Create test files of various sizes: 1MB, 100MB, 500MB, 1GB
- [ ] Write unit tests for streaming vs memory implementation
- [ ] Add memory usage validation tests (monitor with runtime.ReadMemStats)
- [ ] Test hash verification and mismatch handling
- [ ] Test error conditions: disk full, permission denied, network interruption

### Phase 3: Performance and Integration  
- [ ] Benchmark memory usage before/after with large files
- [ ] Benchmark processing speed comparison (should be within 10% of original)
- [ ] Integration tests with real MMS attachment processing
- [ ] Update attachment storage documentation with streaming examples

## Testing
### Unit Tests
- Test streaming with specific file sizes: 1MB, 100MB, 500MB, 1GB
- Test hash verification success and failure scenarios
- Test temp file cleanup on various error conditions
- Test atomic rename operations
- Test buffer size variations and performance impact

### Memory Usage Tests
- Monitor memory usage with `runtime.ReadMemStats` during large file processing
- Verify memory stays under 100MB for files up to 2GB
- Compare memory usage: old implementation vs streaming implementation
- Test concurrent attachment processing memory usage

### Integration Tests
- End-to-end MMS attachment extraction and storage with large files
- Test with real-world attachment sizes from mobile backups
- Verify file integrity after streaming storage

### Performance Benchmarks
- Benchmark processing speed: old vs new implementation
- Benchmark memory allocation patterns
- Test scalability with multiple large attachments
- Measure impact on overall import performance

### Error Recovery Tests
- Simulate disk full during write operation
- Test behavior with permission denied errors
- Simulate network interruption for network-based readers
- Test cleanup after system crash/kill (manual test)

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