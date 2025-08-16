# FEAT-055: Add Context Support Throughout Application

## Status
- **Priority**: medium
- **Review Status**: needs-work (requires Phase 0 completion before ready)

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
**ACTUAL INTERFACES FROM CODEBASE (Phase 0 Complete):**

```go
// calls.Reader interface from pkg/calls/reader.go
type CallsReader interface {
    // Current methods (to be enhanced with context versions)
    ReadCalls(year int) ([]Call, error)
    StreamCallsForYear(year int, callback func(Call) error) error
    GetAvailableYears() ([]int, error)
    GetCallsCount(year int) (int, error)
    ValidateCallsFile(year int) error
    
    // NEW: Context-aware versions
    ReadCallsContext(ctx context.Context, year int) ([]Call, error)
    StreamCallsForYearContext(ctx context.Context, year int, callback func(Call) error) error
    GetAvailableYearsContext(ctx context.Context) ([]int, error)
    GetCallsCountContext(ctx context.Context, year int) (int, error)
    ValidateCallsFileContext(ctx context.Context, year int) error
}

// sms.Reader interface from pkg/sms/reader.go
type SMSReader interface {
    // Current methods (to be enhanced with context versions)
    ReadMessages(year int) ([]Message, error)
    StreamMessagesForYear(year int, callback func(Message) error) error
    GetAttachmentRefs(year int) ([]string, error)
    GetAllAttachmentRefs() (map[string]bool, error)
    GetAvailableYears() ([]int, error)
    GetMessageCount(year int) (int, error)
    ValidateSMSFile(year int) error
    
    // NEW: Context-aware versions
    ReadMessagesContext(ctx context.Context, year int) ([]Message, error)
    StreamMessagesForYearContext(ctx context.Context, year int, callback func(Message) error) error
    GetAttachmentRefsContext(ctx context.Context, year int) ([]string, error)
    GetAllAttachmentRefsContext(ctx context.Context) (map[string]bool, error)
    GetAvailableYearsContext(ctx context.Context) ([]int, error)
    GetMessageCountContext(ctx context.Context, year int) (int, error)
    ValidateSMSFileContext(ctx context.Context, year int) error
}

// contacts.Reader interface from pkg/contacts/types.go
type ContactsReader interface {
    // Current methods (to be enhanced with context versions)
    LoadContacts() error
    GetContactByNumber(number string) (string, bool)
    GetNumbersByContact(name string) ([]string, bool)
    GetAllContacts() ([]*Contact, error)
    ContactExists(name string) bool
    IsKnownNumber(number string) bool
    GetContactsCount() int
    AddUnprocessedContacts(addresses, contactNames string) error
    GetUnprocessedEntries() []UnprocessedEntry
    
    // NEW: Context-aware versions
    LoadContactsContext(ctx context.Context) error
    GetAllContactsContext(ctx context.Context) ([]*Contact, error)
    AddUnprocessedContactsContext(ctx context.Context, addresses, contactNames string) error
}

// contacts.Writer interface from pkg/contacts/types.go
type ContactsWriter interface {
    // Current methods (to be enhanced with context versions)
    SaveContacts(path string) error
    
    // NEW: Context-aware versions
    SaveContactsContext(ctx context.Context, path string) error
}

// attachments.AttachmentReader interface from pkg/attachments/types.go
type AttachmentReader interface {
    // Current methods (to be enhanced with context versions)
    GetAttachment(hash string) (*Attachment, error)
    ReadAttachment(hash string) ([]byte, error)
    AttachmentExists(hash string) (bool, error)
    ListAttachments() ([]*Attachment, error)
    StreamAttachments(callback func(*Attachment) error) error
    VerifyAttachment(hash string) (bool, error)
    GetAttachmentPath(hash string) string
    FindOrphanedAttachments(referencedHashes map[string]bool) ([]*Attachment, error)
    ValidateAttachmentStructure() error
    
    // NEW: Context-aware versions
    GetAttachmentContext(ctx context.Context, hash string) (*Attachment, error)
    ReadAttachmentContext(ctx context.Context, hash string) ([]byte, error)
    AttachmentExistsContext(ctx context.Context, hash string) (bool, error)
    ListAttachmentsContext(ctx context.Context) ([]*Attachment, error)
    StreamAttachmentsContext(ctx context.Context, callback func(*Attachment) error) error
    VerifyAttachmentContext(ctx context.Context, hash string) (bool, error)
    FindOrphanedAttachmentsContext(ctx context.Context, referencedHashes map[string]bool) ([]*Attachment, error)
    ValidateAttachmentStructureContext(ctx context.Context) error
}

// attachments.AttachmentStorage interface from pkg/attachments/types.go
type AttachmentStorage interface {
    // Current methods (to be enhanced with context versions)
    Store(hash string, data []byte, metadata AttachmentInfo) error
    GetPath(hash string) (string, error)
    GetMetadata(hash string) (AttachmentInfo, error)
    Exists(hash string) bool
    
    // NEW: Context-aware versions
    StoreContext(ctx context.Context, hash string, data []byte, metadata AttachmentInfo) error
    GetPathContext(ctx context.Context, hash string) (string, error)
    GetMetadataContext(ctx context.Context, hash string) (AttachmentInfo, error)
    ExistsContext(ctx context.Context, hash string) bool
}

// coalescer.Coalescer interface from pkg/coalescer/types.go
type Coalescer[T Entry] interface {
    // Current methods (to be enhanced with context versions)
    LoadExisting(entries []T) error
    Add(entry T) bool
    GetAll() []T
    GetByYear(year int) []T
    GetSummary() Summary
    Reset()
    
    // NEW: Context-aware versions
    LoadExistingContext(ctx context.Context, entries []T) error
    AddContext(ctx context.Context, entry T) bool
    GetAllContext(ctx context.Context) []T
    GetByYearContext(ctx context.Context, year int) []T
}

// validation.RepositoryValidator interface from pkg/validation/repository.go
type RepositoryValidator interface {
    // Current methods (to be enhanced with context versions)
    ValidateRepository() (*Report, error)
    ValidateStructure() []Violation
    ValidateManifest() []Violation
    ValidateContent() []Violation
    ValidateConsistency() []Violation
    
    // NEW: Context-aware versions
    ValidateRepositoryContext(ctx context.Context) (*Report, error)
    ValidateStructureContext(ctx context.Context) []Violation
    ValidateManifestContext(ctx context.Context) []Violation
    ValidateContentContext(ctx context.Context) []Violation
    ValidateConsistencyContext(ctx context.Context) []Violation
}

// validation.ManifestValidator interface from pkg/validation/types.go
type ManifestValidator interface {
    // Current methods (to be enhanced with context versions)
    LoadManifest() (*FileManifest, error)
    ValidateManifestFormat(manifest *FileManifest) []Violation
    CheckManifestCompleteness(manifest *FileManifest) []Violation
    VerifyManifestChecksum() error
    
    // NEW: Context-aware versions
    LoadManifestContext(ctx context.Context) (*FileManifest, error)
    ValidateManifestFormatContext(ctx context.Context, manifest *FileManifest) []Violation
    CheckManifestCompletenessContext(ctx context.Context, manifest *FileManifest) []Violation
    VerifyManifestChecksumContext(ctx context.Context) error
}
```

### Backward Compatibility Strategy
**DECISION: Context Suffix Approach (Option A)**

After analyzing the codebase dependencies, we're using the Context suffix approach:
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
### Phase 0: Interface Alignment and Strategy (Week 1) - **REQUIRED BEFORE READY**
- [ ] **CRITICAL**: Complete full interface audit and update spec with actual current interfaces:
  - `pkg/calls/reader.go`: Document exact CallsReader interface methods
  - `pkg/sms/reader.go`: Document exact SMSReader interface methods  
  - `pkg/contacts/types.go`: Document exact ContactsReader interface methods
  - `pkg/attachments/types.go`: Document exact AttachmentStorage interface methods
  - `pkg/validation/types.go`: Document validator interfaces (if any)
  - **Replace ALL "VERIFY" sections in this spec** with actual interface definitions

- [x] **Choose and document backward compatibility strategy**: Context suffix approach chosen
  ```
  DECISION: Option A - Context suffix approach (ReadCallsContext, etc.)
  
  RATIONALE:
  - Minimizes breaking changes for existing code
  - Clear naming convention indicates context-aware operations
  - Allows gradual migration of codebase
  - Standard Go practice (see database/sql package evolution)
  - Legacy methods delegate to context versions with context.Background()
  ```

- [x] **Create dependency map**: Document which packages/files use each interface
  ```
  DEPENDENCY MAP:
  
  calls.Reader:
  - pkg/validation/calls.go (CallsValidator)
  - pkg/validation/repository.go (RepositoryValidatorImpl)
  - pkg/validation/reader_adapters.go (validation adapters)
  - cmd/mobilecombackup/cmd/info.go (CLI info command)
  
  sms.Reader:
  - pkg/validation/sms.go (SMSValidator)
  - pkg/validation/attachments.go (AttachmentsValidator)
  - pkg/validation/repository.go (RepositoryValidatorImpl)
  - pkg/validation/reader_adapters.go (validation adapters)
  - cmd/mobilecombackup/cmd/info.go (CLI info command)
  - cmd/mobilecombackup/cmd/validate.go (CLI validate command)
  
  contacts.Reader:
  - pkg/validation/contacts.go (ContactsValidator)
  - pkg/validation/repository.go (RepositoryValidatorImpl)
  - cmd/mobilecombackup/cmd/info.go (CLI info command)
  
  attachments.AttachmentReader:
  - pkg/validation/attachments.go (AttachmentsValidator)
  - pkg/validation/repository.go (RepositoryValidatorImpl)
  - cmd/mobilecombackup/cmd/info.go (CLI info command)
  
  MIGRATION PRIORITY:
  1. Core readers (calls, sms, contacts, attachments) - used by validation
  2. Validation interfaces - used by CLI commands
  3. CLI commands - final integration point
  ```
- [x] **Integration with existing context**: Document how to integrate with existing context usage in logging/metrics packages
  ```
  INTEGRATION PLAN:
  
  Existing Context Usage:
  - pkg/logging/types.go: Logger.WithContext(ctx) for request/operation tracking
  - pkg/metrics/server.go: Server.Stop(ctx) for graceful shutdown
  - Context keys: RequestIDKey, OperationIDKey for tracing
  
  Integration Strategy:
  1. Propagate request/operation IDs through context chain
  2. Use existing ContextLogger for operations with context
  3. Add timeout context creation in CLI commands
  4. Integrate with metrics server shutdown patterns
  5. Use context values for operation tracking in long-running tasks
  
  Context Flow:
  CLI Command -> Context with timeout/signals -> Reader operations -> Logger with context
  ```
- [ ] **Define migration timeline**: Which interfaces/packages to update first based on dependencies

### Phase 1: Core Interface Updates (Week 2)
- [ ] Update interfaces based on Phase 0 alignment
- [ ] Implement chosen backward compatibility strategy consistently
- [ ] Update interface implementations to accept context
- [ ] Add context checking patterns to streaming operations

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

## Specification Review Feedback (2025-08-15)

### Critical Issues Requiring Resolution
1. **Incomplete Phase 0**: Contains placeholder interface definitions that don't match actual codebase
   - `CallsReader` has 5 methods, not placeholder shown
   - `SMSReader` has 8 methods, not placeholder shown  
   - `AttachmentStorage` interface exists with 4 methods
   - `ContactsReader`/`ContactsWriter` are separate interfaces
   - `RepositoryValidator` has different methods than shown
   - `Coalescer[T Entry]` is already generic with 6 methods

2. **Missing backward compatibility decision**: Spec presents three options (A, B, C) but doesn't commit to specific approach

3. **Undefined integration strategy**: Doesn't specify how to integrate with existing context usage in logging/metrics packages

### Required Actions Before Ready
1. Complete full interface audit from actual codebase files
2. Replace ALL placeholder code blocks with actual interface definitions
3. Choose and document backward compatibility strategy
4. Create dependency graph showing interface usage patterns
5. Define integration points with existing context usage

### Assessment
The specification shows excellent planning and design work but cannot proceed to implementation until Phase 0 gaps are filled with actual codebase information. Once Phase 0 is complete, this will be ready for implementation.

## Notes
This is a foundational improvement that will enable better control over long-running operations. Consider implementing in phases, starting with the most critical operations.