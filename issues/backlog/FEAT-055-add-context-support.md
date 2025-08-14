# FEAT-055: Add Context Support Throughout Application

## Status
- **Priority**: medium

## Overview
Add context.Context propagation throughout the application to enable graceful cancellation and timeout handling for long-running operations.

## Background
The current codebase lacks context.Context support, making it impossible to gracefully cancel long-running operations like large file imports or validation processes. This is a significant gap for production usage.

## Requirements
### Functional Requirements
- [ ] Add context.Context parameters to key interfaces and functions
- [ ] Implement cancellation handling in long-running operations
- [ ] Support timeout configuration through context
- [ ] Maintain backward compatibility where possible

### Non-Functional Requirements
- [ ] Operations should respond to cancellation within reasonable time
- [ ] Context cancellation should clean up resources properly
- [ ] Performance impact should be minimal

## Design
### Approach
Gradually introduce context support starting with core interfaces and working outward to implementation details.

### Complete Interface Updates
**All interfaces requiring context.Context support:**

```go
// File readers
type CallsReader interface {
    ReadCalls(ctx context.Context, year int) ([]Call, error)
    StreamCallsForYear(ctx context.Context, year int, callback func(Call) error) error
    GetAvailableYears(ctx context.Context) ([]int, error)
    GetCallsCount(ctx context.Context, year int) (int, error)
}

type SMSReader interface {
    ReadSMS(ctx context.Context, year int) ([]SMS, error)
    ReadMMS(ctx context.Context, year int) ([]MMS, error)
    StreamMessagesForYear(ctx context.Context, year int, callback func(Message) error) error
    GetAvailableYears(ctx context.Context) ([]int, error)
}

type ContactsReader interface {
    ReadContacts(ctx context.Context) ([]Contact, error)
    UpdateContacts(ctx context.Context, contacts []Contact) error
}

// Storage interfaces
type AttachmentStorage interface {
    Store(ctx context.Context, hash string, data []byte, metadata AttachmentInfo) error
    GetPath(ctx context.Context, hash string) (string, error)
    GetMetadata(ctx context.Context, hash string) (AttachmentInfo, error)
    Exists(ctx context.Context, hash string) bool
}

// Processing interfaces
type Coalescer[T Entry] interface {
    LoadExisting(ctx context.Context, entries []T) error
    Add(ctx context.Context, entry T) bool
    GetAll(ctx context.Context) []T
}

// Validation interfaces  
type RepositoryValidator interface {
    ValidateRepository(ctx context.Context) (*ValidationReport, error)
    ValidateStructure(ctx context.Context) []ValidationViolation
    ValidateManifest(ctx context.Context) []ValidationViolation
}
```

### Backward Compatibility Strategy
**Option: New methods with context, keep old ones**
```go
type CallsReader interface {
    // Legacy methods (deprecated but maintained)
    ReadCalls(year int) ([]Call, error)
    StreamCallsForYear(year int, callback func(Call) error) error
    
    // New context-aware methods  
    ReadCallsContext(ctx context.Context, year int) ([]Call, error)
    StreamCallsForYearContext(ctx context.Context, year int, callback func(Call) error) error
}

// Adapter function for migration
func (r *XMLCallsReader) ReadCalls(year int) ([]Call, error) {
    return r.ReadCallsContext(context.Background(), year)
}
```

### Implementation Patterns
**Context Check Pattern for Loops:**
```go
// Check context every 100 iterations to balance performance and responsiveness
func (r *XMLSMSReader) StreamMessagesForYear(ctx context.Context, year int, callback func(Message) error) error {
    const checkInterval = 100
    for i, msg := range messages {
        if i%checkInterval == 0 {
            select {
            case <-ctx.Done():
                return fmt.Errorf("operation cancelled: %w", ctx.Err())
            default:
            }
        }
        if err := callback(msg); err != nil {
            return err
        }
    }
    return nil
}
```

**Resource Cleanup Pattern:**
```go
func (s *DirectoryAttachmentStorage) Store(ctx context.Context, hash string, data []byte, metadata AttachmentInfo) error {
    // Create temp file
    tempFile, err := os.CreateTemp(s.tempDir, "attachment-*.tmp")
    if err != nil {
        return err
    }
    
    // Always cleanup temp file
    defer func() {
        tempFile.Close()
        os.Remove(tempFile.Name())
    }()
    
    // Check context before expensive operation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // Perform work with context awareness...
}
```

**CLI Integration Pattern:**
```go
// Command with timeout support
func runImportCommand(cmd *cobra.Command, args []string) error {
    ctx := context.Background()
    
    // Add timeout if specified
    if importTimeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, importTimeout)
        defer cancel()
    }
    
    // Handle interrupt signals
    ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
    defer cancel()
    
    return importer.ImportFiles(ctx, options)
}
```

### Performance Requirements
- Context cancellation must be detected within **100ms** for streaming operations
- Performance overhead must not exceed **5%** for typical operations  
- All file handles must be closed within **1 second** of context cancellation
- Default timeouts:
  - File operations: **5 minutes**
  - Full import: **30 minutes**  
  - Validation: **10 minutes**

## Tasks
### Phase 1: Core Interface Updates (Week 1)
- [ ] Update CallsReader interface with context methods
- [ ] Update SMSReader interface with context methods  
- [ ] Update ContactsReader interface with context methods
- [ ] Update AttachmentStorage interface with context methods
- [ ] Add backward compatibility adapter methods
- [ ] Update interface implementations to accept context

### Phase 2: Processing Components (Week 2)
- [ ] Add context support to Coalescer interfaces and implementations
- [ ] Add context support to validation interfaces and implementations
- [ ] Implement context checking patterns in streaming operations
- [ ] Add resource cleanup patterns for context cancellation
- [ ] Update progress reporters to handle context cancellation

### Phase 3: CLI Integration (Week 3)
- [ ] Add timeout flags to CLI commands: `--timeout`, `--import-timeout`
- [ ] Add signal handling for graceful cancellation (SIGINT, SIGTERM)
- [ ] Update all command handlers to create and pass contexts
- [ ] Add context creation patterns for different operation types
- [ ] Test CLI timeout and cancellation behavior

### Phase 4: Testing and Documentation (Week 4)
- [ ] Add unit tests for context cancellation in all updated interfaces
- [ ] Add integration tests for end-to-end cancellation scenarios
- [ ] Add performance benchmarks to measure context overhead
- [ ] Create migration guide for developers using the interfaces
- [ ] Update API documentation with context usage patterns

## Testing
### Context Cancellation Tests
- **TestStreamingCancellation**: Cancel context during message streaming
- **TestFileOperationTimeout**: Test timeout during large file operations
- **TestValidationCancellation**: Cancel context during repository validation
- **TestImportCancellation**: Cancel full import operation mid-stream
- **TestResourceCleanup**: Verify all resources cleaned up on cancellation

### Performance Impact Tests  
- **BenchmarkContextOverhead**: Measure performance impact of context checks
- **TestCancellationResponsiveness**: Verify 100ms cancellation detection target
- **BenchmarkTimeoutOperations**: Compare performance with/without timeouts

### CLI Integration Tests
- **TestSignalHandling**: Test SIGINT/SIGTERM handling during operations
- **TestTimeoutFlags**: Test --timeout flag with various values
- **TestGracefulShutdown**: Verify clean shutdown with context cancellation

### Edge Cases and Error Scenarios
- Context cancellation during partial file write
- Multiple concurrent context cancellations
- Context cancellation with concurrent goroutines
- Nested context timeouts (operation timeout within command timeout)
- Context cancellation during transaction-like operations

### Migration Compatibility Tests
- Test backward compatibility with legacy method calls
- Verify adapter methods work correctly
- Test mixed usage (some callers use context, others don't)

## Risks and Mitigations
- **Risk**: Breaking API changes for existing interfaces
  - **Mitigation**: Provide adapter functions for backward compatibility
- **Risk**: Performance overhead from frequent context checks
  - **Mitigation**: Add context checks at appropriate intervals, not every iteration

## References
- [Go Concurrency Patterns: Context](https://blog.golang.org/context)
- Source: CODE_IMPROVEMENT_REPORT.md item #7

## Notes
This is a foundational improvement that will enable better control over long-running operations. Consider implementing in phases, starting with the most critical operations.