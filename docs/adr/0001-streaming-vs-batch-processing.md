# ADR-0001: Streaming vs Batch Processing

**Status:** Accepted
**Date:** 2024-01-15
**Author:** Development Team
**Deciders:** Core development team

## Context

When processing large XML files (calls, SMS/MMS backup files), we needed to choose between two fundamental approaches:

1. **Batch Processing**: Load entire XML files into memory, parse everything at once, then process the complete dataset
2. **Streaming Processing**: Read and process XML files incrementally using streaming parsers with callback patterns

Mobile backup files can be extremely large (hundreds of MB to several GB), containing thousands of records. Memory usage and performance characteristics differ significantly between these approaches.

## Decision

We chose **streaming XML processing with callback patterns** for all mobile backup file processing.

## Rationale

### Memory Efficiency
- Streaming processing maintains constant memory usage regardless of file size
- Batch processing would require loading entire files into memory, potentially causing OOM errors
- Users often have backup files exceeding available system memory

### Scalability
- Streaming enables processing arbitrarily large files without performance degradation
- Batch processing memory requirements scale linearly with file size
- Streaming supports real-time progress reporting during import operations

### Performance Characteristics
- Streaming provides consistent, predictable performance
- Early termination possible for validation or error conditions
- Reduced garbage collection pressure compared to large object allocations

### Alternatives Considered
1. **Hybrid approach**: Stream for large files, batch for small files
   - Rejected: Complexity of dual code paths outweighed benefits
   - Different behavior for different file sizes creates testing complexity

2. **Memory-mapped files**: Map files to virtual memory
   - Rejected: XML parsing still requires structured processing
   - Platform-specific behavior complications

3. **Chunked batch processing**: Process files in fixed-size chunks
   - Rejected: XML structure doesn't align with arbitrary chunk boundaries
   - Still vulnerable to individual record size variations

## Consequences

### Positive Consequences
- **Memory efficiency**: Constant memory usage regardless of file size
- **Scalability**: Can process files limited only by storage, not memory
- **Progress reporting**: Real-time feedback during long-running operations
- **Error resilience**: Can continue processing after individual record failures
- **Interruptible operations**: Can cleanly stop processing on user request

### Negative Consequences
- **API complexity**: Callback-based APIs are more complex than simple function calls
- **Error handling**: Error recovery requires careful state management
- **Testing complexity**: Streaming behavior requires more sophisticated test scenarios
- **Sequential processing**: Cannot easily implement random access or parallel processing

## Implementation

### Core Components
- Custom XML streaming parser with security protections
- Callback-based processing interfaces for calls, SMS, MMS records
- Progress reporting through callback mechanisms
- Error collection and continuation strategies

### Key Patterns
```go
// Streaming reader interface
type Reader interface {
    StreamCalls(callback func(*Call) error) error
    StreamSMS(callback func(*Message) error) error
}

// Usage pattern
reader.StreamCalls(func(call *Call) error {
    // Process individual call record
    return processor.HandleCall(call)
})
```

### Error Handling Strategy
- Individual record failures collected but don't stop processing
- Fatal errors (file corruption, I/O errors) terminate processing
- Detailed error reporting with record context

## Related Decisions

- **ADR-0003**: XML Security Approach - Security requirements influenced streaming implementation
- **ADR-0002**: Hash-based Storage - Streaming enables efficient attachment processing during parse