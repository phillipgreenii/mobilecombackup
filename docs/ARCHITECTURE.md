# Architecture Overview

This document provides a comprehensive overview of the mobilecombackup system architecture, package relationships, and key design decisions.

## System Purpose

The mobilecombackup tool processes mobile phone backup files (SMS/MMS and call logs in XML format), coalescing multiple backup sources while removing duplicates, extracting attachments, and organizing data by year for efficient long-term storage.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                           CLI Layer                              │
│  cmd/mobilecombackup - Cobra-based CLI with subcommands         │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│                     Service Layer                                │
│  - pkg/importer: Orchestrates import workflows                  │
│  - pkg/validation: Repository integrity validation              │
│  - pkg/autofix: Automatic repair of common issues              │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│                    Domain Layer                                  │
│  - pkg/calls: Call log processing and streaming                 │
│  - pkg/sms: SMS/MMS processing with attachment handling         │
│  - pkg/contacts: Contact management and YAML processing         │
│  - pkg/coalescer: Generic deduplication engine                  │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│                 Infrastructure Layer                             │
│  - pkg/attachments: Hash-based file storage                     │
│  - pkg/manifest: File integrity and change tracking             │
│  - pkg/security: Path validation and safety checks              │
│  - pkg/logging: Structured logging system                       │
│  - pkg/config: Configuration management                         │
│  - pkg/types: Generic utilities and type definitions            │
└─────────────────────────────────────────────────────────────────┘
```

## Package Relationships

### Core Processing Flow

1. **CLI Commands** (`cmd/mobilecombackup`)
   - Entry point for all operations
   - Implements subcommands: import, validate, info, init
   - Handles flag parsing and output formatting

2. **Import Orchestration** (`pkg/importer`)
   - Coordinates multi-step import process
   - Manages progress reporting and error collection
   - Integrates validation, coalescing, and file generation

3. **Data Processing** (`pkg/calls`, `pkg/sms`)
   - Stream-based XML parsing for memory efficiency
   - Year-based partitioning of data
   - Attachment extraction and referencing

4. **Storage Systems** (`pkg/attachments`, `pkg/manifest`)
   - Content-addressable attachment storage
   - File integrity tracking and verification

### Key Interfaces

#### Reader Interfaces
```go
// pkg/calls/reader.go
type CallsReader interface {
    ReadCalls(year int) ([]Call, error)
    StreamCallsForYear(year int, callback func(Call) error) error
    GetAvailableYears() ([]int, error)
    GetCallCount(year int) (int, error)
}

// pkg/sms/reader.go  
type SMSReader interface {
    ReadMessages(year int) ([]Message, error)
    StreamMessagesForYear(year int, callback func(Message) error) error
    GetAvailableYears() ([]int, error)
    GetMessageCount(year int) (int, error)
}
```

#### Storage Interfaces
```go
// pkg/attachments/types.go
type AttachmentStorage interface {
    Store(hash string, data []byte, info AttachmentInfo) error
    StoreFromReader(hash string, data io.Reader, info AttachmentInfo) error
    GetMetadata(hash string) (*AttachmentInfo, error)
    Exists(hash string) bool
    GetPath(hash string) string
}
```

#### Coalescing Interface
```go
// pkg/coalescer/types.go
type Entry interface {
    Hash() string
    Timestamp() time.Time
    Year() int
}

type Coalescer[T Entry] interface {
    LoadExisting(entries []T) error
    Add(entry T) bool
    GetSorted() []T
    GetSummary() Summary
}
```

## Data Flow Architecture

### Import Process Flow

```
XML Backup Files
       │
       ▼
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│  XML Parser │───▶│  Coalescer   │───▶│ Year Writer │
│ (Streaming) │    │(Deduplicates)│    │  (Output)   │
└─────────────┘    └──────────────┘    └─────────────┘
       │                    │                   │
       ▼                    ▼                   ▼
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│ Attachment  │    │   Contact    │    │ Repository  │
│ Extractor   │    │  Processor   │    │   Files     │
└─────────────┘    └──────────────┘    └─────────────┘
       │                    │                   │
       ▼                    ▼                   ▼
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│  Content-   │    │ contacts.yaml│    │ Year-based  │
│Addressable  │    │    File      │    │   Storage   │
│  Storage    │    │              │    │             │
└─────────────┘    └──────────────┘    └─────────────┘
```

### Repository Structure

```
repository/
├── .mobilecombackup.yaml    # Repository marker file
├── calls/                   # Call records by year
│   ├── calls-2023.xml
│   ├── calls-2024.xml
│   └── summary.yaml
├── sms/                     # SMS/MMS records by year  
│   ├── sms-2023.xml
│   ├── sms-2024.xml
│   └── summary.yaml
├── attachments/             # Hash-based attachment storage
│   ├── ab/
│   │   └── abc123.../
│   │       ├── photo.jpg
│   │       └── metadata.yaml
│   └── summary.yaml
├── contacts.yaml            # Contact information
├── files.yaml              # File manifest
├── files.yaml.sha256       # Manifest checksum
└── rejected/                # Files that couldn't be processed
```

## Key Design Decisions

### Streaming Architecture
- **Why**: Handle files larger than available memory
- **How**: Interface-based streaming with callback functions
- **Trade-offs**: More complex code, but scales to any file size

### Content-Addressable Storage
- **Why**: Automatic deduplication of identical attachments
- **How**: SHA-256 hashes as primary keys with two-level directory structure
- **Trade-offs**: More complex directory layout, but eliminates duplicate storage

### Year-Based Partitioning
- **Why**: Enables efficient processing of specific time ranges
- **How**: All data partitioned by UTC year with separate files
- **Trade-offs**: More files to manage, but better performance for time-based queries

### Generic Coalescer
- **Why**: Type-safe deduplication for any entry type
- **How**: Go 1.18+ generics with hash-based deduplication
- **Trade-offs**: Requires newer Go version, but eliminates code duplication

### UTC Time Handling
- **Why**: Consistent timezone handling across all operations
- **How**: All timestamps normalized to UTC with explicit conversion
- **Trade-offs**: May lose original timezone info, but ensures consistency

## Security Architecture

### Path Validation (`pkg/security`)
- All file paths validated to prevent directory traversal
- Relative path resolution with security checks
- Null byte and control character filtering

### Hash Verification
- SHA-256 checksums for all stored files
- Streaming hash calculation during file operations
- Integrity verification on read operations

### Input Validation
- XML parser with resource limits to prevent DoS
- Fuzz testing for parser security
- Safe handling of malformed input data

## Performance Characteristics

### Memory Usage
- **Streaming Operations**: O(1) memory usage regardless of file size
- **Coalescing**: O(n) memory where n is unique entries
- **Attachment Storage**: O(1) memory per operation

### I/O Patterns
- **Sequential Reads**: Optimized for large XML file processing
- **Random Writes**: Content-addressable storage optimized for diverse access patterns
- **Atomic Operations**: Temporary files ensure consistency

### Scalability
- **Horizontal**: Multiple repositories can be processed independently
- **Vertical**: Streaming architecture handles arbitrarily large files
- **Concurrent**: Thread-safe operations support parallel processing

## Error Handling Strategy

### Error Categories
1. **Recoverable Errors**: Continue processing with error collection
2. **Fatal Errors**: Stop processing immediately (e.g., disk full)
3. **Validation Errors**: Collected for batch reporting and autofix

### Error Propagation
- Errors bubble up through layers with context
- Service layer collects and categorizes errors
- CLI layer formats errors for user consumption

### Autofix Integration
- Validation errors feed into autofix system
- Common issues automatically repaired when safe
- Dry-run mode for validation before applying fixes

## Testing Strategy

### Unit Tests
- Individual package functionality
- Interface compliance testing
- Edge case and error condition coverage

### Integration Tests
- End-to-end workflow testing
- Real file processing validation
- Cross-package interaction verification

### Performance Tests
- Benchmark tests for critical paths
- Memory usage validation
- Large file processing verification

### Security Tests
- Fuzz testing for parsers
- Path traversal attack prevention
- Input validation boundary testing

## Development Guidelines

### Code Organization
- Interface-first design for testability
- Package-level documentation with examples
- Clear separation of concerns between layers

### Error Handling
- Return errors, don't use os.Exit() in libraries
- Include context in error messages
- Collect errors for batch processing when appropriate

### Performance Considerations
- Stream large files rather than loading into memory
- Use content addressing for automatic deduplication
- Implement atomic operations for consistency

### Security Best Practices
- Validate all external input
- Use secure file operations
- Implement proper path sanitization

## Future Architecture Considerations

### Potential Extensions
- Plugin architecture for additional data sources
- Distributed processing for very large datasets
- Database backend option for metadata storage
- Compression support for stored files

### Scalability Improvements
- Parallel processing pipelines
- Incremental import capabilities
- Background processing options
- Cloud storage integration

This architecture balances performance, security, and maintainability while providing a solid foundation for processing large mobile communication backups efficiently and safely.