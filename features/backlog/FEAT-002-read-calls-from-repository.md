# FEAT-002: Read Calls from Repository

## Status
- **Completed**: -
- **Priority**: high

## Overview
Implement functionality to read call records from the repository structure. This feature provides the foundation for accessing and processing call data stored in the repository's XML files.

## Background
The repository stores call records in XML files organized by year (e.g., `calls/calls-2015.xml`). Each file contains call entries with attributes like number, duration, date, type, etc. This feature will provide a clean interface for reading these records, which is essential for validation, analysis, and other operations.

## Requirements
### Functional Requirements
- [ ] Read call records from `calls/calls-YYYY.xml` files
- [ ] Parse XML structure according to the calls schema
- [ ] Handle all call attributes: number, duration, date, type, readable_date, contact_name
- [ ] Support streaming for large files (memory efficiency)
- [ ] Convert epoch milliseconds in `date` field to proper time.Time
- [ ] Handle missing or optional fields gracefully
- [ ] Return structured call data with proper types

### Non-Functional Requirements
- [ ] Memory efficient - stream processing for files >100MB
- [ ] Proper error handling with context (file name, line number)
- [ ] Performance - process 10,000 calls/second minimum
- [ ] Support for schema evolution (unknown attributes ignored, not error)

## Design
### Approach
Create a dedicated reader for call XML files that:
1. Opens and streams XML files
2. Validates basic structure (calls element with count attribute)
3. Parses individual call entries
4. Converts data types appropriately
5. Returns typed Call structs or errors

### API/Interface
```go
// Call represents a single call record
type Call struct {
    Number       string
    Duration     int
    Date         time.Time
    Type         CallType
    ReadableDate string
    ContactName  string
}

// CallType represents the type of call
type CallType int

const (
    IncomingCall CallType = 1
    OutgoingCall CallType = 2
    MissedCall   CallType = 3
    VoicemailCall CallType = 4
)

// CallsReader reads call records from repository
type CallsReader interface {
    // ReadCalls reads all calls from a specific year file
    ReadCalls(year int) ([]Call, error)
    
    // StreamCalls streams calls from a year file for memory efficiency
    StreamCalls(year int, callback func(Call) error) error
    
    // GetAvailableYears returns list of years with call data
    GetAvailableYears() ([]int, error)
}
```

### Implementation Notes
- Use encoding/xml with xml.Decoder for streaming
- Validate count attribute matches actual entries (warning if mismatch)
- Convert date from epoch milliseconds to time.Time
- Handle phone number format variations
- ContactName is informational only, not used for identity

## Tasks
- [ ] Define Call struct and CallType constants
- [ ] Create CallsReader interface
- [ ] Implement XML streaming parser
- [ ] Add date conversion utilities
- [ ] Implement count validation
- [ ] Add error handling with context
- [ ] Write comprehensive unit tests
- [ ] Add integration tests with sample files
- [ ] Document usage examples

## Testing
### Unit Tests
- Parse valid call XML with all fields
- Parse call with missing optional fields
- Handle malformed XML gracefully
- Verify count validation
- Test date conversion accuracy
- Test all call types

### Integration Tests
- Read actual repository files
- Handle large files (memory usage verification)
- Test with empty calls file
- Test with missing year files

### Edge Cases
- Empty calls file (count=0)
- Mismatched count attribute
- Invalid date values
- Unknown call types
- Malformed phone numbers
- XML encoding issues

## Risks and Mitigations
- **Risk**: Schema changes in future backups
  - **Mitigation**: Ignore unknown attributes, focus on core fields
- **Risk**: Large file memory usage
  - **Mitigation**: Implement streaming API
- **Risk**: Invalid XML from corrupted backups
  - **Mitigation**: Graceful error handling, partial results where possible

## References
- Related features: FEAT-001 (Repository Validation)
- Specification: See "Call Backup" section in specification.md
- Code location: pkg/calls/reader.go (to be created)

## Notes
- This is a read-only feature; no modification of repository
- Consider caching parsed results for repeated reads
- Phone number normalization might be needed in future