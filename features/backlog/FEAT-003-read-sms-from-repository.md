# FEAT-003: Read SMS from Repository

## Status
- **Completed**: -
- **Priority**: high

## Overview
Implement functionality to read SMS and MMS records from the repository structure. This feature provides the foundation for accessing and processing SMS/MMS data stored in the repository's XML files, including validation of the XML structure, handling attachment references, and tracking which attachments are used.

## Background
The repository stores SMS/MMS records in XML files organized by year (e.g., `sms/sms-2015.xml`). SMS entries are simple text messages, while MMS entries are more complex with multiple parts, addresses, and attachment references. This feature will provide a clean interface for reading these records.

## Requirements
### Functional Requirements
- [ ] Read SMS/MMS records from `sms/sms-YYYY.xml` files
- [ ] Parse XML structure according to SMS and MMS schemas
- [ ] Validate XML structure matches expected schema
- [ ] Handle SMS attributes: protocol, address, date, type, subject, body, etc.
- [ ] Handle MMS structure with parts and addresses
- [ ] Extract attachment references from MMS parts
- [ ] Track which attachments are referenced by messages
- [ ] Support streaming for large files (memory efficiency)
- [ ] Convert epoch milliseconds to proper time.Time
- [ ] Handle group messages (multiple addresses)
- [ ] Parse SMIL presentation data
- [ ] Verify year in filename matches actual record dates (based on UTC)
- [ ] Provide total count of records per year
- [ ] Validate count attribute matches actual number of entries

### Non-Functional Requirements
- [ ] Memory efficient - stream processing for files >1GB
- [ ] Proper error handling with context
- [ ] Performance - process 5,000 messages/second minimum
- [ ] Support for schema variations between backup versions

## Design
### Approach
Create a dedicated reader for SMS/MMS XML files that:
1. Streams XML files efficiently
2. Differentiates between SMS and MMS entries
3. Handles complex MMS structure with parts
4. Extracts attachment references
5. Returns typed message structs

### API/Interface
```go
// Message is the base interface for SMS and MMS
type Message interface {
    GetDate() time.Time
    GetAddress() string
    GetType() MessageType
    GetReadableDate() string
    GetContactName() string
}

// SMS represents a simple text message
type SMS struct {
    Protocol      string
    Address       string
    Date          time.Time
    Type          MessageType
    Subject       string
    Body          string
    ServiceCenter string
    Read          bool
    Locked        bool
    DateSent      time.Time
    ReadableDate  string
    ContactName   string
}

// MMS represents a multimedia message
type MMS struct {
    Date         time.Time
    MsgBox       int
    Address      []string // Multiple for group messages
    MType        int
    MId          string
    ThreadId     string
    Parts        []MMSPart
    Addresses    []MMSAddress
    ReadableDate string
    ContactName  string
    // Additional MMS-specific fields...
}

// MMSPart represents a content part of an MMS
type MMSPart struct {
    Seq           int
    ContentType   string
    Name          string
    Text          string
    AttachmentRef string // Path reference if attachment
}

// MessageType represents the type of message
type MessageType int

const (
    ReceivedMessage MessageType = 1
    SentMessage     MessageType = 2
)

// SMSReader reads SMS/MMS records from repository
type SMSReader interface {
    // ReadMessages reads all messages from a specific year
    ReadMessages(year int) ([]Message, error)
    
    // StreamMessages streams messages for memory efficiency
    StreamMessages(year int, callback func(Message) error) error
    
    // GetAttachmentRefs returns all attachment references in a year
    GetAttachmentRefs(year int) ([]string, error)
    
    // GetAllAttachmentRefs returns all attachment references across all years
    GetAllAttachmentRefs() (map[string]bool, error)
    
    // GetAvailableYears returns list of years with SMS data
    GetAvailableYears() ([]int, error)
    
    // GetMessageCount returns total number of messages for a year
    GetMessageCount(year int) (int, error)
    
    // ValidateSMSFile validates XML structure and year consistency
    ValidateSMSFile(year int) error
}
```

### Implementation Notes
- Use encoding/xml with xml.Decoder for streaming
- Differentiate SMS vs MMS by element name
- Parse MMS parts carefully, skip SMIL parts for attachments
- Convert base64 references to attachment paths
- Handle group message addresses (separated by ~)

## Tasks
- [ ] Define Message interface and concrete types
- [ ] Create SMSReader interface
- [ ] Implement XML streaming parser
- [ ] Add XML schema validation
- [ ] Add SMS parsing logic
- [ ] Add MMS parsing with parts/addresses
- [ ] Implement attachment reference extraction and tracking
- [ ] Add year consistency validation
- [ ] Add date/time conversion utilities
- [ ] Write comprehensive unit tests
- [ ] Add integration tests
- [ ] Document complex MMS structure

## Testing
### Unit Tests
- Parse simple SMS with all fields
- Parse MMS with text and image parts
- Handle group MMS with multiple addresses
- Test SMIL part filtering
- Verify attachment reference extraction
- Test date conversions

### Integration Tests
- Read actual repository files
- Handle very large SMS files
- Process mixed SMS/MMS files
- Verify attachment references exist

### Edge Cases
- Empty SMS file (count=0)
- MMS without parts
- Corrupted attachment references
- Invalid date values
- Missing required fields
- Special characters in body text

## Risks and Mitigations
- **Risk**: Complex MMS structure variations
  - **Mitigation**: Flexible parsing, log unknown structures
- **Risk**: Large attachment data in XML
  - **Mitigation**: Skip data field parsing, use references only
- **Risk**: Memory usage with many messages
  - **Mitigation**: Streaming API mandatory

## References
- Related features: FEAT-004 (Attachments)
- Specification: See "SMS Backup" section in specification.md
- Code location: pkg/sms/reader.go (to be created)

## Notes
- MMS structure is significantly more complex than SMS
- SMIL parts define presentation, not actual content
- Attachment data should already be extracted to files
- Consider lazy loading of MMS parts for performance