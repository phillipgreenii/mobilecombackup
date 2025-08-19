# Specification

## Overview

Each day, SMS and Call logs are backed up on phone.  Historically, the backup would include all SMS and calls whaich are on the phone. Later backups include entries from the previous X number days.  Either way backups contain a lot of duplicates.  This is a major problem with SMS which backup the same images over and over again.  This leads to daily backup files of >700MB.  

To remedy this, this code will focus on doing two things.
 1) Remove duplicates.  We should be able to take all of the entries and keep only distinct ones. The format of the entries may make this difficult due to the same entry being stored in the backup in different ways.
    - Fields may change over time
    - Timestamp field with timezone uses the current location, which might be different from a previous backup.
 2) Store images external of the backup files.
    - By default, the images are stored in the backup files encoded with base64.
    - We want to change this so that Images will be written to disk in a location identified by hash.

### Expected Flow

1. Load Files
   - Read Existing Repository
   - Load contacts.yaml including any existing unprocessed entries
   - Any specified unprocessed backup file
2. Entry Transformation
   - Validate entries and extract contact names from contact_name attributes
   - **Extract Attachments (FEAT-012)**: Extract user-attached content from MMS parts
     - Filter by content type (extract images/videos/audio/documents, skip system files)
     - Store in hash-based structure: `attachments/[first-2-chars]/[full-hash]`
     - Update MMS parts with path references and metadata
   - Remove Duplicates (excluding contact_name from hash calculation)
3. Order Entries
   - Sort by Timestamp (UTC)
   - Partition by Year
4. Overwrite Repository
   - Write processed entries to repository
   - Save contacts.yaml with extracted contact names in unprocessed section

## File / Directory Schemas

### Call Backup (`calls.xml`)

Calls are stored as XML, schema is as follows:

- `calls`
  - attributes:
    - `count`: number of call entries in this file
  - children:
    - `call`
      - attributes:
        - `number`: phone number
          - possible formats: `ddddddd`, `dddddddddd`, `+1dddddddddd`
        - `duration`: duration of the call in seconds
        - `date`: timestamp in epoch milliseconds
        - `type`: enum to represent the type of call (matches Android CallLog.Calls constants)
          - 1: incoming (INCOMING_TYPE)
          - 2: outgoing (OUTGOING_TYPE)
          - 3: missed (MISSED_TYPE)
          - 4: voicemail (VOICEMAIL_TYPE)
        - `readable_date`: `date` formatted to be consumed by humans; may not be consistent, don't use in comparison
        - `contact_name`: name associated with phone number; may not be consistent, don't use in comparison

### SMS Backup (`sms.xml`)

SMS and MMS messages are stored as XML, schema is as follows:

- `smses`
  - attributes:
    - `count`: number of SMS/MMS entries in this file
  - children:
    - `sms`: regular SMS message
      - attributes:
        - `protocol`: protocol version (typically "0")
        - `address`: phone number or short code
          - possible formats: `dddd`, `ddddddd`, `dddddddddd`, `+1dddddddddd`
          - can also be email for MMS group messages: `user@domain.com`
        - `date`: timestamp in epoch milliseconds
        - `type`: enum to represent the type of message (matches Android Telephony.Sms constants)
          - 1: received (MESSAGE_TYPE_INBOX)
          - 2: sent (MESSAGE_TYPE_SENT)
        - `subject`: subject line (typically "null" for SMS)
        - `body`: message text content
        - `toa`: type of address (typically "null")
        - `sc_toa`: service center type of address (typically "null")
        - `service_center`: SMSC number (e.g., "+13123149623" or "null")
        - `read`: read status (0=unread, 1=read)
        - `status`: delivery status (typically "-1")
        - `locked`: locked status (0=unlocked, 1=locked)
        - `date_sent`: timestamp when sent in epoch milliseconds (0 for received)
        - `readable_date`: `date` formatted for humans; may not be consistent, don't use in comparison
        - `contact_name`: name associated with phone number; may not be consistent, don't use in comparison
    - `mms`: multimedia message
      - attributes:
        - `date`: timestamp in epoch milliseconds
        - `msg_box`: message box type (1=inbox, 2=sent)
        - `address`: recipient(s) phone number(s), can be multiple separated by "~" for group messages
        - `m_type`: MMS message type (128=send request, 132=retrieve confirmation)
        - `m_id`: unique message ID
        - `thread_id`: conversation thread ID (optional)
        - `readable_date`: `date` formatted for humans; may not be consistent, don't use in comparison
        - `contact_name`: name(s) associated with phone number(s); may not be consistent, don't use in comparison
        - Additional MMS-specific attributes: `callback_set`, `text_only`, `sub`, `retr_st`, `ct_cls`, `sub_cs`, `read`, `ct_l`, `tr_id`, `st`, `d_tm`, `read_status`, `ct_t`, `retr_txt_cs`, `deletable`, `d_rpt`, `date_sent`, `seen`, `reserved`, `v`, `exp`, `pri`, `hidden`, `msg_id`, `rr`, `app_id`, `resp_txt`, `rpt_a`, `locked`, `retr_txt`, `resp_st`, `m_size`, `m_cls`
      - children:
        - `parts`: container for message parts
          - `part`: individual content part
            - attributes:
              - `seq`: sequence number (-1 for SMIL layout defining MMS presentation, 0+ for actual content parts)
              - `ct`: content type (e.g., "application/smil", "text/plain", "image/png", "image/jpeg")
              - `name`: file name (can be "null")
              - `chset`: character set (e.g., "106" for UTF-8, can be "null")
              - `cd`: content disposition (e.g., "attachment", can be "null")
              - `fn`: filename (can be "null")
              - `cid`: content ID
              - `cl`: content location
              - `ctt_s`, `ctt_t`: content type start/type (typically "null")
              - `text`: text content or SMIL layout XML
              - `data`: base64-encoded binary data for attachments (images, etc.)
              - **FEAT-012 Enhanced Fields (after attachment extraction):**
                - `path`: repository-relative path to extracted attachment file
                - `original_size`: size of decoded attachment in bytes
                - `extraction_date`: ISO8601 timestamp when attachment was extracted
        - `addrs`: recipient addresses (for group MMS)
          - `addr`: individual address
            - attributes:
              - `address`: phone number
              - `type`: address type (137=from, 151=to)
              - `charset`: character encoding

### Repository

A repository is a file structure which will house all of the processed backup files.  The structure is as follows:

```
repository/
├── contacts.yaml
├── files.yaml
├── files.yaml.sha256
├── summary.yaml
├── calls/
│   ├── calls-2015.xml
│   └── calls-2016.xml
├── sms/
│   ├── sms-2015.xml
│   └── sms-2016.xml
└── attachments/
    └── [hash-based subdirectories]
```

#### contacts.yaml

`contacts.yaml` contains mapping from number to contact name, plus an optional unprocessed section for contact names extracted during import.

```yaml
contacts:
  - name: "Bob Ross"
    numbers: 
      - "+11111111111"
  - name: "<unknown>"
    numbers:
      - "8888888888"
      - "9999999999"

unprocessed:
  - phone_number: "5551234567"
    contact_names: ["John Doe", "Johnny"]
  - phone_number: "5559876543" 
    contact_names: ["Jane Smith"]
```

**Unprocessed Section (FEAT-035):**
- Contains contact names extracted from imported messages in structured format
- **Structure**: Each entry has `phone_number` and `contact_names` fields
- **Multi-Address Support**: Parses addresses with `~` separators and contact names with `,` separators
- **Validation**: Requires equal counts of phone numbers and contact names (rejects entire SMS entry if mismatch)
- **Contact Matching**: Excludes phone numbers that already exist in main contacts section using normalized phone number comparison
- **Duplicate Prevention**: Ensures contacts do not appear in both processed and unprocessed sections (BUG-069 fix)
- **Name Combining**: Multiple contact names for same phone number are combined into single entry
- **Consistent Ordering**: Entries consistently ordered by phone number lexicographically in saved files (BUG-070 fix)
- **Manual Review**: Requires manual promotion to main contacts section for canonical name selection

#### files.yaml

`files.yaml` lists all files in the repository with their associated sha256 and size.
 - Exclusions: `files.yaml` and `files.yaml.sha256`

```yaml
files:
  - file: summary.yaml
    sha256: ...
    size_bytes: 1234
  - file: contacts.yaml
    sha256: ...
    size_bytes: 567
  - file: calls/calls-2015.xml
    sha256: ...
    size_bytes: 89012
```

#### files.yaml.sha256

`files.yaml.sha256` is the SHA256 of `files.yaml`.

#### summary.yaml

`summary.yaml` contains a summary of repository contents. This file is automatically generated after each import operation and reflects the current state of the repository.

**Implemented Structure (BUG-020):**
```yaml
last_updated: "2025-08-09T10:30:00Z"  # RFC3339 timestamp
statistics:
  total_calls: 1250        # Total number of call entries across all years
  total_sms: 3400          # Total number of SMS/MMS entries across all years 
  total_attachments: 450   # Total number of attachment files
  years_covered:           # List of years that have data
    - 2014
    - 2015
    - 2016
```

**Properties:**
- `last_updated`: ISO 8601/RFC3339 timestamp of when the summary was generated
- `statistics`: Repository-wide statistics
  - `total_calls`: Sum of all call entries across all year files
  - `total_sms`: Sum of all SMS and MMS entries across all year files
  - `total_attachments`: Count of unique attachment files in attachments directory
  - `years_covered`: Sorted list of years that have data (calls or SMS)

**Generation:**
- Created/updated at the end of each successful import (non-dry-run mode)
- If generation fails, a warning is logged but import continues
- Not created during repository initialization (only during import)

**Original Design (Not Implemented):**
|Property|Description|
|--------|-----------|
| `count`  | Total number of record of thie type |
| `size_bytes` | Total size, in bytes, of the records for this type. |
| `oldest`  | Timestamp of the oldest record of this type |
| `latest`  | Timestamp of the latest record of this type |

```yaml
summary:
  contacts:
    count: 12
    size_bytes: 1000
  calls:
    count: 234
    oldest: $timestamp
    latest: $timestamp2
    size_bytes: 2000
  sms:
    count: 2345
    oldest: $timestamp3
    latest: $timestamp4
    size_bytes: 123456
  attachments:
    count: 12
    size_bytes: 1000000
```

#### calls/

These files are of the same schema as outlined previousl in "Calls Backup".

There should be a file per year: 

```
calls/
├── calls-2015.xml
└── calls-2016.xml
```

#### sms/

These files are of the same schema as outlined previousl in "SMS Backup".

There should be a file per year: 

```
sms/
├── sms-2015.xml
└── sms-2016.xml
```

#### attachments/

Contents will be the attachments identified by their hash.  They will be grouped into directories based upon the first 2 characters of the hash:

```
attachments/
├── ab/
│   ├── ab54363e39
│   └── ab8bd623a0
└── b2/
    └── b28fe7c6ab
```

**Attachment Extraction Process (FEAT-012):**
- Only user-attached content is extracted (images, videos, audio, documents)
- System-generated content (SMIL, text, vCards) remains inline in MMS XML
- Base64-encoded data is decoded and stored with SHA-256 hash-based naming
- Files are deduplicated - multiple messages can reference the same attachment
- MMS parts are updated with `path`, `original_size`, and `extraction_date` fields
- Extraction occurs during SMS import after validation but before deduplication

**Supported File Types:**
- **Images**: JPEG, PNG, GIF, BMP, WebP
- **Videos**: MP4, 3GPP, QuickTime, AVI
- **Audio**: MPEG, MP4, AMR, WAV
- **Documents**: PDF, Word, Excel, PowerPoint (Office formats)

## Important Notes

### Contact Name Handling
The `contact_name` field should NOT be used for comparing or identifying duplicate records. This field may vary between backups due to:
- Contact name changes in the phone's address book
- Sync timing differences
- Unknown numbers showing as "(Unknown)" vs actual names

The phone number (`number` for calls, `address` for SMS/MMS) should be used as the primary identifier. The `contacts.yaml` file will maintain the authoritative mapping of numbers to names.

### Security and Path Validation
All file operations handling external input use the security path validator (`pkg/security/path.go`) to prevent directory traversal attacks. This includes:
- CLI argument processing and validation
- Import/export operations with user-provided paths
- Repository structure management and file operations
- Attachment extraction and storage operations
- Orphan removal and cleanup operations

Path validation ensures all file operations remain within repository boundaries and prevents access to parent directories or sensitive system locations.

### Timezone Considerations
- The `readable_date` field uses the timezone of where the backup was performed
- When writing to the repository, all `readable_date` fields will be recalculated using EST
- The `date` field (epoch milliseconds) is always UTC and should be used for sorting and year partitioning

### MMS Structure
- MMS messages can be group messages with multiple recipients separated by "~"
- SMIL (Synchronized Multimedia Integration Language) parts define the presentation layout
- Actual content (text, images) is stored in separate parts with positive sequence numbers
- Image attachments are base64-encoded in the `data` attribute

### Hash Algorithm
SHA-256 will be used consistently for:
- Attachment content hashing and storage
- Entry deduplication (excluding `readable_date`, `contact_name` and after attachment extraction)

## Reader Interfaces

### CallsReader Interface

The CallsReader provides access to call records stored in the repository:

```go
type CallsReader interface {
    // ReadCalls reads all calls from a specific year
    ReadCalls(year int) ([]Call, error)
    
    // StreamCalls streams calls for memory efficiency
    StreamCalls(year int, callback func(Call) error) error
    
    // GetAvailableYears returns list of years with call data
    GetAvailableYears() ([]int, error)
    
    // GetCallCount returns total number of calls for a year
    GetCallCount(year int) (int, error)
    
    // ValidateCallsFile validates XML structure and year consistency
    ValidateCallsFile(year int) error
}
```

**Key Features:**
- Streaming API for memory-efficient processing of large files
- Year-based validation ensuring call dates match filename years (UTC-based)
- XML schema validation including count attribute verification
- Support for all call types: incoming (1), outgoing (2), missed (3), voicemail (4)

### SMSReader Interface

The SMSReader provides access to SMS and MMS records stored in the repository:

```go
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

**Key Features:**
- Unified interface for SMS and MMS messages through Message interface
- Attachment reference tracking for cleanup operations
- Complex MMS parsing with parts and addresses
- Group message support (multiple recipients)
- SMIL part filtering and handling

### Message Types

#### Message Interface
```go
type Message interface {
    GetDate() time.Time
    GetAddress() string
    GetType() MessageType
    GetReadableDate() string
    GetContactName() string
}
```

#### MessageType Constants
```go
type MessageType int

const (
    ReceivedMessage MessageType = 1  // Inbox/received
    SentMessage     MessageType = 2  // Sent messages
)
```

**SMS Type Determination:** Uses `type` attribute (1=received, 2=sent)
**MMS Type Determination:** Uses `msg_box` attribute (1=received, 2=sent)

### ContactsReader Interface

The ContactsReader provides access to contact information stored in the repository:

```go
type ContactsReader interface {
    // LoadContacts loads all contacts from contacts.yaml
    LoadContacts() error
    
    // GetContactByNumber returns contact name for a phone number
    GetContactByNumber(number string) (string, bool)
    
    // GetNumbersByContact returns all numbers for a contact name
    GetNumbersByContact(name string) ([]string, bool)
    
    // GetAllContacts returns all contacts
    GetAllContacts() ([]*Contact, error)
    
    // ContactExists checks if a contact name exists
    ContactExists(name string) bool
    
    // IsKnownNumber checks if a number has a contact
    IsKnownNumber(number string) bool
    
    // GetContactsCount returns total number of contacts
    GetContactsCount() int
    
    // AddUnprocessedContacts processes SMS address and contact names (FEAT-035)
    AddUnprocessedContacts(address, contactNames string) error
    
    // GetUnprocessedEntries returns all unprocessed entries sorted by phone number
    GetUnprocessedEntries() []UnprocessedEntry
}
```

**Key Features:**
- Efficient O(1) phone number lookup using hash maps
- Phone number normalization for consistent matching
- Handles multiple number formats (+1XXXXXXXXXX, (XXX) XXX-XXXX, etc.)
- Special handling for `<unknown>` contact designation
- Duplicate number detection and validation
- Graceful handling of missing contacts.yaml files
- Unprocessed contact extraction and storage during import
- Atomic file operations for data integrity
- Support for multiple name variations per phone number
- **Consistency Fixes**: Normalized phone number comparison prevents contacts from appearing in both processed and unprocessed sections (BUG-069)
- **Deterministic Ordering**: SaveContacts produces consistent file output with lexicographically sorted unprocessed entries (BUG-070)

### ContactsWriter Interface

The ContactsWriter provides functionality for writing contact information to the repository:

```go
type ContactsWriter interface {
	// SaveContacts writes the current state to contacts.yaml
	SaveContacts(path string) error

	// AddUnprocessedContact adds a contact to the unprocessed section
	AddUnprocessedContact(phone, name string)

	// GetUnprocessedContacts returns all unprocessed contacts
	GetUnprocessedContacts() map[string][]string
}
```

**Key Features:**
- Contact name extraction during import process
- Unprocessed section management for manual review workflow
- Phone number normalization for consistent storage
- Atomic file operations to prevent data loss
- Preservation of existing processed contacts and unprocessed entries
- Support for multiple name variations per phone number
- Integration with SMS, MMS, and call import processes

#### Contact Structure
```go
type Contact struct {
    Name    string   `yaml:"name"`
    Numbers []string `yaml:"numbers"`
}

// UnprocessedEntry represents structured unprocessed contact data (FEAT-035)
type UnprocessedEntry struct {
    PhoneNumber  string   `yaml:"phone_number"`
    ContactNames []string `yaml:"contact_names"`
}

// ContactsData represents the YAML structure of contacts.yaml
type ContactsData struct {
    Contacts    []*Contact         `yaml:"contacts"`
    Unprocessed []UnprocessedEntry `yaml:"unprocessed,omitempty"`
}
```

## Validation Interface

### RepositoryValidator Interface

The RepositoryValidator provides comprehensive validation of repository structure and content:

```go
type RepositoryValidator interface {
    // ValidateRepository performs comprehensive repository validation
    ValidateRepository() (*ValidationReport, error)
    
    // ValidateStructure validates overall repository structure
    ValidateStructure() []ValidationViolation
    
    // ValidateManifest validates files.yaml completeness and accuracy
    ValidateManifest() []ValidationViolation
    
    // ValidateContent validates all content files
    ValidateContent() []ValidationViolation
    
    // ValidateConsistency performs cross-file consistency validation
    ValidateConsistency() []ValidationViolation
}
```

### ValidationReport Structure
```go
type ValidationReport struct {
    Timestamp      time.Time             `yaml:"timestamp"`
    RepositoryPath string                `yaml:"repository_path"`
    Status         ValidationStatus      `yaml:"status"`
    Violations     []ValidationViolation `yaml:"violations"`
}

type ValidationViolation struct {
    Type     ViolationType `yaml:"type"`
    Severity Severity      `yaml:"severity"`
    File     string        `yaml:"file"`
    Message  string        `yaml:"message"`
    Expected string        `yaml:"expected,omitempty"`
    Actual   string        `yaml:"actual,omitempty"`
}
```

### Validation Types and Severity
```go
type ViolationType string

const (
    MissingFile        ViolationType = "missing_file"
    ExtraFile          ViolationType = "extra_file"
    ChecksumMismatch   ViolationType = "checksum_mismatch"
    InvalidFormat      ViolationType = "invalid_format"
    OrphanedAttachment ViolationType = "orphaned_attachment"
    CountMismatch      ViolationType = "count_mismatch"
    SizeMismatch       ViolationType = "size_mismatch"
    StructureViolation ViolationType = "structure_violation"
    MissingMarkerFile  ViolationType = "missing_marker_file"
    UnsupportedVersion ViolationType = "unsupported_version"
    FormatMismatch     ViolationType = "format_mismatch"     // FEAT-025: File content doesn't match expected MIME type
    UnknownFormat      ViolationType = "unknown_format"      // FEAT-025: Unrecognized file format
)

type Severity string

const (
    SeverityError   Severity = "error"
    SeverityWarning Severity = "warning"
)

// FixableViolation extends ValidationViolation with suggested fix content
type FixableViolation struct {
    ValidationViolation
    SuggestedFix string `yaml:"suggested_fix,omitempty"`
}
```

### ReportGenerator Interface

The ReportGenerator creates formatted validation reports:

```go
type ReportGenerator interface {
    // GenerateReport creates a formatted validation report
    GenerateReport(report *ValidationReport, format ReportFormat, 
                  options *ReportFilterOptions) (string, error)
    
    // GenerateSummary creates a brief summary of validation results
    GenerateSummary(report *ValidationReport) (*ValidationSummary, error)
}

type ReportFormat string

const (
    FormatYAML ReportFormat = "yaml"
    FormatJSON ReportFormat = "json"
    FormatText ReportFormat = "text"
)
```

### OptimizedRepositoryValidator Interface

The OptimizedRepositoryValidator extends base validation with performance features:

```go
type OptimizedRepositoryValidator interface {
    RepositoryValidator
    
    // ValidateRepositoryWithOptions performs validation with performance controls
    ValidateRepositoryWithOptions(ctx context.Context, 
                                 options *PerformanceOptions) (*ValidationReport, error)
    
    // ValidateAsync performs validation asynchronously with progress reporting
    ValidateAsync(options *PerformanceOptions) (<-chan *ValidationReport, <-chan error)
    
    // GetMetrics returns current validation performance metrics
    GetMetrics() *ValidationMetrics
    
    // ClearCache clears the validation cache
    ClearCache()
}

type PerformanceOptions struct {
    ParallelValidation bool
    EarlyTermination   bool
    MaxConcurrency     int
    ProgressCallback   func(stage string, progress float64)
    Timeout           time.Duration
}
```

**Key Features:**
- Marker file validation as first step with version support checking
- Comprehensive validation of repository structure, manifest, content, and consistency
- Two-tier severity system (errors vs warnings)
- Multi-format report generation (YAML, JSON, human-readable text)
- Filtering and customization options for reports
- Performance optimizations for large repositories
- Parallel validation with configurable concurrency
- Progress reporting and metrics tracking
- Early termination on critical errors or unsupported versions
- Context-aware cancellation support
- Fixable violations with suggested fix content
- **Format Validation (FEAT-025)**: Magic byte detection for PNG, JPEG, GIF, MP4, PDF formats
  - Verifies file content matches expected MIME type from SMS/MMS metadata
  - Always-on validation with minimal performance overhead (&lt;5%)
  - Rejects unknown file formats to ensure repository integrity

#### Phone Number Normalization
- Removes all non-digit characters
- Strips leading "1" for US numbers (11 digits starting with 1)
- Provides consistent lookup regardless of input format
- Supports formats: +1XXXXXXXXXX, 1XXXXXXXXXX, XXXXXXXXXX, (XXX) XXX-XXXX, XXX-XXX-XXXX, XXX.XXX.XXXX

### AttachmentReader Interface

The AttachmentReader provides access to attachment files stored in the repository:

```go
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
    FindOrphanedAttachments(referencedHashes map[string]bool) ([]*Attachment, error)
    
    // ValidateAttachmentStructure validates the directory structure
    ValidateAttachmentStructure() error
}
```

**Key Features:**
- Hash-based content addressing using SHA-256
- Directory structure: `attachments/[first-2-chars]/[full-hash]`
- Content verification against stored hashes
- Efficient streaming for large attachment repositories
- Orphaned attachment detection for cleanup operations
- Directory structure validation
- Statistics collection for repository analysis

#### Attachment Structure
```go
type Attachment struct {
    Hash   string // SHA-256 hash in lowercase hex
    Path   string // Relative path: attachments/ab/ab54363e39...
    Size   int64  // File size in bytes
    Exists bool   // Whether the file exists on disk
}
```

#### Hash-Based Storage
- Uses first 2 characters of SHA-256 hash as subdirectory prefix
- Prevents filesystem performance issues with too many files in single directory
- Content-addressed storage ensures data integrity
- No file extensions needed (content type stored in MMS metadata)

### Marker File Validation

The `.mobilecombackup.yaml` marker file is validated first before any other repository validation:

#### MarkerFileValidator Interface
```go
type MarkerFileValidator interface {
    // ValidateMarkerFile checks the marker file exists and has valid content
    // Returns: violations, versionSupported, error
    ValidateMarkerFile() ([]ValidationViolation, bool, error)
    
    // GetSuggestedFix returns the suggested content for a missing marker file
    GetSuggestedFix() string
}
```

#### Marker File Requirements
- **Required Fields:**
  - `repository_structure_version`: Must be "1" (only supported version)
  - `created_at`: RFC3339 timestamp of repository creation
  - `created_by`: Tool and version that created the repository
- **Validation Process:**
  1. Check file exists (missing file is a fixable violation)
  2. Validate YAML syntax before field parsing
  3. Validate all required fields are present
  4. Validate RFC3339 timestamp format
  5. Check version is supported (only "1" currently)
  6. Log warnings for extra fields (not violations)
- **Version Support:**
  - If version is unsupported, validation stops immediately
  - Repository with unsupported version is marked Invalid
  - Prevents processing of incompatible repository structures

### Autofix Interface (FEAT-011)

The AutofixManager provides automatic fixing of validation violations through safe, well-known operations:

```go
type AutofixManager interface {
    // FixViolations attempts to fix all provided violations
    FixViolations(violations []ValidationViolation) (*AutofixReport, error)
    
    // CanFix returns whether a specific violation type can be automatically fixed
    CanFix(violationType ViolationType) bool
    
    // GetSupportedViolationTypes returns all violation types that can be fixed
    GetSupportedViolationTypes() []ViolationType
}

type AutofixReport struct {
    FixedViolations    []FixedViolation `json:"fixed_violations"`
    UnfixedViolations  []ValidationViolation `json:"unfixed_violations"`
    Errors            []AutofixError   `json:"errors"`
    Summary           AutofixSummary   `json:"summary"`
}

type FixedViolation struct {
    ValidationViolation
    Action      string `json:"action"`        // Description of fix applied
    NewValue    string `json:"new_value,omitempty"` // New value created/set
}

type AutofixError struct {
    ViolationType ViolationType `json:"violation_type"`
    File         string        `json:"file"`
    Message      string        `json:"message"`
    Action       string        `json:"attempted_action"`
}

type AutofixSummary struct {
    TotalViolations int `json:"total_violations"`
    Fixed          int `json:"fixed"`
    Unfixed        int `json:"unfixed"`
    Errors         int `json:"errors"`
}
```

#### Supported Violation Types

The AutofixManager can automatically fix the following violation types:

| ViolationType | Fix Action | Safety Level |
|---------------|------------|--------------|
| `missing_directory` | Create directory with proper permissions | Safe |
| `missing_file` | Generate file with appropriate content | Safe |
| `missing_files_yaml` | Scan repository and generate complete file list | Safe |
| `missing_checksum` | Generate SHA-256 for files.yaml.sha256 | Safe |
| `incorrect_size` | Update size_bytes in files.yaml entries | Safe |
| `missing_file_entry` | Add missing files to files.yaml | Safe |
| `stale_file_entry` | Remove non-existent entries from files.yaml | Safe |

#### Safety Principles

- **Conservative Operations**: Only performs well-known, safe fixes that cannot cause data loss
- **Atomic Operations**: Uses atomic file operations (write to temp, then rename) to prevent corruption
- **Integrity Preservation**: Never auto-corrects SHA-256 mismatches to preserve integrity violation detection
- **Idempotent Behavior**: Running autofix multiple times is safe and produces consistent results
- **Error Isolation**: Individual fix failures do not stop the overall process

#### CLI Integration

The autofix functionality integrates seamlessly with the validate command:

```bash
# Run validation with automatic fixing
mobilecombackup validate --autofix

# Preview fixes without applying them (future enhancement)
mobilecombackup validate --autofix --dry-run

# Combine with JSON output for programmatic use
mobilecombackup validate --autofix --output-json
```

**Exit Codes:**
- `0`: All violations were successfully fixed
- `1`: Some violations remain after autofix attempts
- `2`: Errors occurred during autofix (but process continued)
- `3`: Fatal error prevented autofix from running

### Validation Requirements

Both CallsReader and SMSReader implementations provide validation capabilities:

1. **XML Schema Validation**
   - Verify count attribute matches actual number of entries
   - Validate required fields are present and properly formatted
   - Handle null values appropriately (convert "null" strings to empty values)

2. **Year Consistency Validation**
   - Ensure all records in a file belong to the year specified in filename
   - Use UTC timezone for year determination
   - Report specific violations with context (file path, record details)

3. **Date Conversion**
   - Convert epoch milliseconds to time.Time objects
   - Handle timezone-independent processing using UTC
   - Validate date ranges and detect anomalies

### Performance Characteristics

- **Streaming Processing:** Both readers support streaming APIs to handle large files (>1GB) without memory issues
- **Performance Targets:**
  - Calls: 10,000 records/second minimum
  - SMS: 5,000 messages/second minimum
- **Memory Efficiency:** Use encoding/xml.Decoder for streaming XML parsing
- **Error Resilience:** Continue processing on individual record failures, collect errors for reporting

## Command-Line Interface

### Overview

The mobilecombackup CLI provides a command-line interface built on the Cobra framework for interacting with mobile backup repositories.

### Global Flags

All commands support the following global flags:

- `--repo-root string`: Path to repository root (default ".")
- `--quiet`: Suppress non-error output
- `--version, -v`: Display version information
- `--help, -h`: Display help information

### Version Information

The CLI supports version injection during build:
```bash
go build -ldflags "-X main.Version=1.2.3" -o mobilecombackup github.com/phillipgreen/mobilecombackup/cmd/mobilecombackup
```

Version output format: `mobilecombackup version X.Y.Z`

### Exit Codes

- `0`: Success
- `1`: Command/flag error or execution failure
- `2`: Runtime error (used by subcommands)

### Error Handling

- All errors are prefixed with "Error: " and written to stderr
- Unknown commands and flags produce specific error messages
- Help is displayed when no arguments are provided

### Package Structure

```
cmd/mobilecombackup/
├── main.go           # Entry point with version injection
└── cmd/
    ├── root.go       # Root command definition and global flags
    └── init.go       # Init subcommand implementation
```

### Subcommands

#### Init Command

Initializes a new mobilecombackup repository with the required directory structure.

**Usage:**
```bash
mobilecombackup init [flags]
```

**Flags:**
- `--dry-run`: Preview actions without creating directories

**Behavior:**
1. Validates target directory (must be empty or non-existent)
2. Creates directory structure:
   - `calls/` - For call log XML files
   - `sms/` - For SMS/MMS XML files
   - `attachments/` - For extracted attachment files
3. Creates metadata files:
   - `.mobilecombackup.yaml` - Repository marker with version metadata
   - `contacts.yaml` - Empty contacts file
   - Does NOT create `summary.yaml` (only created during import)
4. Displays tree-style output of created structure
5. Does NOT create summary.yaml (only created during import)

**Marker File Format:**
```yaml
repository_structure_version: "1"
created_at: "2024-01-15T10:30:00Z"  # RFC3339 format
created_by: "mobilecombackup v0.1.0"
```

**Error Conditions:**
- Directory already contains a repository (has .mobilecombackup.yaml)
- Directory appears to be a repository (has calls/, sms/, or attachments/)
- Directory is not empty
- Path exists but is not a directory

#### Validate Command

Validates a mobilecombackup repository for structure, content, and consistency.

**Usage:**
```bash
mobilecombackup validate [flags]
```

**Flags:**
- `--verbose`: Show detailed progress information
- `--output-json`: Output results in JSON format
- `--remove-orphan-attachments`: Remove orphaned attachment files (FEAT-013)
- `--dry-run`: Show what would be done without making changes

**Behavior:**
1. Resolves repository root from (in order):
   - `--repo-root` flag
   - `MB_REPO_ROOT` environment variable
   - Current directory
2. Creates all necessary readers (calls, SMS, attachments, contacts)
3. Runs comprehensive validation using `RepositoryValidator`
4. Outputs results in selected format (text or JSON)
5. Returns appropriate exit code

**Exit Codes:**
- `0`: Repository is valid
- `1`: Validation violations detected
- `2`: Runtime error (repository not found, I/O error)

**Output Formats:**

Text output (default):
- Shows progress indicator during validation
- Groups violations by type with counts
- Displays file path and message for each violation

JSON output (`--output-json`):
```json
{
  "valid": boolean,
  "violations": [
    {
      "Type": "violation_type",
      "Severity": "error|warning",
      "File": "path/to/file",
      "Message": "description",
      "Expected": "expected_value",
      "Actual": "actual_value"
    }
  ]
}
```

**Quiet Mode:**
- With `--quiet` flag, suppresses all output except violations
- Valid repositories produce no output
- Invalid repositories show only violation details

**Verbose Mode:**
- With `--verbose` flag, shows detailed progress for each phase
- Displays "Completed X validation" messages

**Orphan Attachment Removal (FEAT-013):**
The `--remove-orphan-attachments` flag enables removal of attachment files that are no longer referenced by any SMS/MMS messages:

- **Safe Operation**: Only removes files confirmed to have no references
- **Security**: Uses path validation to ensure removal operations stay within repository boundaries (BUG-056 fix)
- **Dry-Run Support**: Use `--dry-run` to preview what would be removed
- **Progress Reporting**: Shows scanning and removal progress
- **Error Handling**: Continues processing if individual files fail to remove
- **Directory Cleanup**: Removes empty subdirectories after file removal
- **Statistics**: Reports attachments scanned, orphans found, removed, and bytes freed

**Output with Orphan Removal:**
```
Repository validation completed:
  Calls: 12,345 processed, 0 violations found
  SMS: 23,456 processed, 0 violations found
  Attachments: 3,456 processed, 0 violations found
  
Orphan attachment removal:
  Attachments scanned: 3,456
  Orphans found: 23
  Orphans removed: 22 (44.4 MB freed)
  Removal failures: 1
    - attachments/ab/ab12345...: Permission denied
```

**Usage Examples:**
```bash
# Validate and remove orphans
mobilecombackup validate --remove-orphan-attachments

# Preview orphan removal without changes
mobilecombackup validate --remove-orphan-attachments --dry-run

# Combine with JSON output
mobilecombackup validate --remove-orphan-attachments --output-json
```

**Autofix Functionality (FEAT-011 - Completed):**
Automatic fixing of validation violations is fully implemented with production-ready functionality. All autofix operations are complete:

- **CLI Integration**: Complete `--autofix` flag support in validate command with dry-run and progress reporting
- **Directory Structure Creation**: Automatic creation of missing directories (calls/, sms/, attachments/)
- **Metadata File Generation**: 
  - `.mobilecombackup.yaml` marker file with version and timestamp metadata
  - `contacts.yaml` empty structure generation
  - `summary.yaml` repository statistics generation
- **Files.yaml Management**: Complete file list generation and updates with SHA-256 hashing
- **Files.yaml.sha256 Generation**: Integrity verification file creation
- **XML Count Fixes**: Automatic correction of mismatched count attributes in XML files using streaming parsing
- **Enhanced Progress Reporting**: Detailed operation counts and verbose mode support
- **Enhanced Dry-run Mode**: Permission checking and detailed previews without execution
- **Atomic Operations**: Safe file operations with proper error handling and rollback
- **Exit Codes**: Comprehensive exit code support (0=all fixed, 1=violations remain, 2=errors, 3=fatal)
- **Output Formatting**: Both JSON and text output with detailed violation reporting
- **Comprehensive Testing**: 18 total tests (12 unit, 6 integration) achieving 78.2% coverage

**All Operations Complete (10/10 major tasks finished):**
1. Directory structure creation - ✓ Complete
2. Marker file generation - ✓ Complete  
3. Metadata file generation - ✓ Complete
4. Files.yaml generation/updates - ✓ Complete
5. Files.yaml.sha256 generation - ✓ Complete
6. Atomic file operations - ✓ Complete
7. XML count attribute fixes - ✓ Complete
8. Enhanced progress reporting - ✓ Complete
9. Enhanced dry-run mode - ✓ Complete
10. Comprehensive testing - ✓ Complete

#### Info Command

Shows comprehensive information about a mobilecombackup repository including statistics, metadata, and validation status.

**Usage:**
```bash
mobilecombackup info [flags]
```

**Flags:**
- `--json`: Output information in JSON format

**Behavior:**
1. Resolves repository root from (in order):
   - `--repo-root` flag
   - `MB_REPO_ROOT` environment variable 
   - Current directory
2. Reads repository metadata from `.mobilecombackup.yaml`
3. Gathers comprehensive statistics:
   - Call counts by year with earliest/latest date ranges
   - SMS/MMS counts by year with type breakdown and date ranges
   - Attachment counts with total size and MIME type distribution
   - Orphaned attachment detection (unreferenced by messages)
   - Contact counts
   - Rejection file counts (from `rejected/` directory)
   - Error counts from reader operations
4. Performs basic validation status check
5. Outputs results in selected format (text or JSON)

**Exit Codes:**
- `0`: Success
- `2`: Runtime error (repository not found, I/O error)

**Output Features:**
- Human-readable formatting with thousands separators
- Byte size formatting (B, KB, MB, GB)
- Date ranges displayed as "Jan 2 - Dec 28" format
- MIME types sorted by count (descending)
- Graceful handling of missing components

**Text Output Example:**
```
Repository: /home/user/backup
Version: 1
Created: 2024-01-15T10:30:00Z

Calls:
  2023: 1,234 calls (Jan 5 - Dec 28)
  2024: 567 calls (Jan 2 - Jun 15)
  Total: 1,801 calls

Messages:
  2023: 5,432 messages (4,321 SMS, 1,111 MMS) (Jan 1 - Dec 31)
  2024: 2,345 messages (2,000 SMS, 345 MMS) (Jan 1 - Jun 20)
  Total: 7,777 messages (6,321 SMS, 1,456 MMS)

Attachments:
  Count: 1,456
  Total Size: 245.3 MB
  Types:
    image/jpeg: 1,200
    image/png: 200
    video/mp4: 56
  Orphaned: 12

Contacts: 123

Validation: OK
```

**JSON Output:**
```json
{
  "version": "1",
  "created_at": "2024-01-15T10:30:00Z",
  "calls": {
    "2023": {
      "count": 1234,
      "earliest": "2023-01-05T12:30:00Z",
      "latest": "2023-12-28T18:45:00Z"
    }
  },
  "sms": {
    "2023": {
      "total_count": 5432,
      "sms_count": 4321,
      "mms_count": 1111,
      "earliest": "2023-01-01T00:00:00Z",
      "latest": "2023-12-31T23:59:00Z"
    }
  },
  "attachments": {
    "count": 1456,
    "total_size": 257368064,
    "orphaned_count": 12,
    "by_type": {
      "image/jpeg": 1200,
      "image/png": 200,
      "video/mp4": 56
    }
  },
  "contacts": 123,
  "validation_ok": true
}
```

**MIME Type Detection:**
- Based on file extensions in attachment paths
- Supports: image/jpeg, image/png, image/gif, video/mp4, video/3gpp, audio/mp3, audio/amr
- Default: application/octet-stream for unknown types

**Performance Characteristics:**
- Uses streaming APIs for memory efficiency
- No modification of repository data (read-only operation)
- Handles large repositories efficiently
- Progress indication for operations taking >2 seconds

#### Import Command
Imports mobile backup files into the repository with deduplication and validation.

**Usage:**
```bash
mobilecombackup import [flags] [paths...]
```

**Arguments:**
- `paths`: Files or directories to import (default: current directory)

**Flags:**
- `--dry-run`: Preview import without making changes
- `--verbose`: Enable verbose output
- `--json`: Output summary in JSON format
- `--filter string`: Process only specific type: calls, sms
- `--no-error-on-rejects`: Don't exit with error code if rejects found

**Repository Resolution:**
The repository location is determined by (in order of precedence):
1. `--repo-root` flag
2. `MB_REPO_ROOT` environment variable
3. Current directory

**Behavior:**
1. **Repository Validation**: Comprehensive validation using pkg/validation before any import operations (FEAT-036)
   - Validates repository structure, manifest, and content integrity
   - Fast-fail behavior with detailed error messages if validation fails
   - Silent operation unless validation errors occur
   - Exit code 2 for validation failures (consistent with validate command)
2. Loads existing repository data for deduplication
3. Scans paths for backup files (`calls*.xml`, `sms*.xml`)
   - Follows symlinks
   - Skips hidden directories (starting with `.`)
   - Excludes files already in repository structure
4. Processes each file:
   - Reports progress every 100 records
   - Validates entries
   - Detects duplicates via hash
   - Accumulates valid entries
5. Single repository write operation:
   - Merges existing and new entries
   - Sorts by timestamp
   - Partitions by year
   - Skips write if `--dry-run`
6. Displays summary with statistics

**Exit Codes:**
- `0`: Success
- `1`: Import completed with rejected entries (unless `--no-error-on-rejects`)
- `2`: Import failed (validation error, I/O error)

**Output Formats:**

Default output:
```
Processing files...
  Processing: backup-2024-01-15.xml (100 records)... (200 records)... done
  Processing: calls-2024-02-01.xml (100 records)... done

Import Summary:
              Initial     Final     Delta     Duplicates    Rejected
Calls Total        10        45        35             12           3
  2023              5        15        10              3           1
  2024              5        30        25              9           2
SMS Total          23        78        55             20           5
  2023             13        38        25              8           2
  2024             10        40        30             12           3

Files processed: 2
Time taken: 2.3s
```

JSON output (`--json`):
```json
{
  "files_processed": 2,
  "duration_seconds": 2.3,
  "total": {
    "initial": 33,
    "final": 123,
    "added": 90,
    "duplicates": 32,
    "rejected": 8,
    "errors": 0
  },
  "years": {
    "2023": {"final": 53, "added": 35, "duplicates": 11, "rejected": 3},
    "2024": {"final": 70, "added": 55, "duplicates": 21, "rejected": 5}
  },
  "rejection_files": []
}
```

**Examples:**
```bash
# Import specific files
mobilecombackup import --repo-root /path/to/repo backup1.xml backup2.xml

# Scan directory for backup files
mobilecombackup import --repo-root /path/to/repo /path/to/backups/

# Preview import without changes
mobilecombackup import --repo-root /path/to/repo --dry-run backup.xml

# Import only call logs
mobilecombackup import --repo-root /path/to/repo --filter calls backups/

# Import with JSON output
mobilecombackup import --repo-root /path/to/repo --json backups/
```

#### Completion Command

Generates shell completion scripts for mobilecombackup to enable tab-completion for commands, flags, and arguments.

**Usage:**
```bash
mobilecombackup completion [bash|zsh|fish|powershell]
```

**Supported Shells:**
- **bash**: Requires bash-completion package
- **zsh**: Requires compinit to be loaded
- **fish**: Built-in completion support
- **powershell**: Built-in completion support

**Setup Instructions:**

The completion command provides comprehensive setup instructions for each shell including:
- Temporary setup (current session only)
- Permanent setup (persistent across sessions)
- OS-specific notes and requirements
- Troubleshooting tips for common issues

**Behavior:**
1. Validates shell argument (must be one of: bash, zsh, fish, powershell)
2. Generates appropriate completion script for the specified shell
3. Outputs script to stdout for piping or redirection
4. Returns error for invalid shell arguments

**Examples:**
```bash
# View setup instructions
mobilecombackup completion --help

# Generate bash completion script
mobilecombackup completion bash > mobilecombackup-completion.bash

# Install fish completion directly
mobilecombackup completion fish > ~/.config/fish/completions/mobilecombackup.fish

# Load zsh completion temporarily
source <(mobilecombackup completion zsh)
```

**Help Text Features:**
- Complete setup instructions for all supported shells
- Both temporary and permanent installation methods
- Verification steps to confirm completion is working
- Troubleshooting section with common solutions
- OS-specific notes (Linux, macOS, Windows)

### Helper Functions

- `PrintError(format string, args ...interface{})`: Consistent error output to stderr

## Repository Initialization and Manifest Management

### Overview

Repository initialization creates a complete, valid repository structure with all required files and directories. The manifest system tracks all repository files for integrity validation.

### Manifest Package (pkg/manifest/)

The manifest package provides shared functionality for generating and managing file manifests across repository operations.

#### Core Types
```go
// FileEntry represents a single file entry in files.yaml
type FileEntry struct {
    File      string `yaml:"file"`
    SHA256    string `yaml:"sha256"`
    SizeBytes int64  `yaml:"size_bytes"`
}

// FileManifest represents the structure of files.yaml
type FileManifest struct {
    Files []FileEntry `yaml:"files"`
}
```

#### ManifestGenerator Interface
```go
type ManifestGenerator struct {
    repositoryRoot string
}

// Core methods
func NewManifestGenerator(repositoryRoot string) *ManifestGenerator
func (g *ManifestGenerator) GenerateFileManifest() (*FileManifest, error)
func (g *ManifestGenerator) WriteManifestFiles(manifest *FileManifest) error
func (g *ManifestGenerator) WriteManifestOnly(manifest *FileManifest) error
func (g *ManifestGenerator) WriteChecksumOnly() error
```

#### Manifest Generation Rules
- **Included Files**: `.mobilecombackup.yaml`, `summary.yaml`, `contacts.yaml`, and all files in `calls/`, `sms/`, `attachments/`
- **Excluded Files**: `files.yaml` (itself), `files.yaml.sha256`, temporary files (`.tmp`), rejected entries (`rejected/`), hidden files (except `.mobilecombackup.yaml`)
- **Path Format**: Cross-platform consistency using forward slashes via `filepath.ToSlash()`
- **Atomic Operations**: Uses temporary files with atomic rename to prevent corruption

#### Integration Points
- **Repository Initialization**: Creates complete manifest during `init` command
- **Validation Autofix**: Regenerates manifest when inconsistencies detected
- **Import Operations**: Updates manifest after successful imports

### Repository Structure Creation

#### Complete Initialization (FEAT-032)
The `init` command creates a complete, valid repository structure:

```
repository/
├── .mobilecombackup.yaml    # Repository marker with version metadata
├── files.yaml               # File manifest (excludes itself and .sha256)
├── files.yaml.sha256        # SHA-256 checksum of files.yaml
├── summary.yaml             # Empty import summary
├── contacts.yaml            # Empty contacts list
├── calls/                   # Empty directory for call logs
├── sms/                     # Empty directory for SMS/MMS
└── attachments/             # Empty directory for attachments
```

Note: `rejected/` directory is created only when needed during import operations.

#### Autofix Behavior
- **files.yaml**: Always regenerated from repository scan to ensure completeness
- **files.yaml.sha256**: Created only if missing (preserves existing checksums)
- **Marker File**: Generated with current timestamp and tool version metadata
- **Directory Structure**: Missing directories created with proper permissions (0750)

## Import Functionality

### Overview

The import functionality provides the ability to import call and SMS backup files into the repository with automatic deduplication, validation, and rejection handling.

### Architecture

#### Coalescer System

**Generic Coalescer Interface:**
```go
type Entry interface {
    Hash() string         // Unique identifier for deduplication
    Timestamp() time.Time // For sorting
    Year() int           // For partitioning
}

type Coalescer[T Entry] interface {
    LoadExisting(entries []T) error
    Add(entry T) bool              // Returns true if added (not duplicate)
    GetAll() []T                   // Returns sorted by timestamp
    GetByYear(year int) []T
    GetSummary() Summary
    Reset()
}
```

**Hash-Based Deduplication:**
- SHA-256 hashes for content addressing
- Excludes volatile fields (`readable_date`, `contact_name`)
- O(1) duplicate detection using map[string]Entry

#### Call Import

**CallEntry Implementation:**
- Wraps `Call` struct to implement `Entry` interface
- Hash calculation includes: number, duration, date, type
- Excludes: readable_date, contact_name (may change over time)

**Validation Rules:**
- Required fields: date (>0), number (non-empty), valid type (1-4)
- Duration must be non-negative
- Invalid entries written to rejection files

**Rejection Handling:**
- Directory structure created only when rejections occur (lazy creation)
- Thread-safe directory creation using sync.Once pattern
- Mirrored repository structure:
  ```
  rejected/
  ├── calls/
  │   └── calls-[hash]-[timestamp].xml
  └── sms/
      └── sms-[hash]-[timestamp].xml
  ```
- Original XML structure preserved for re-import after correction
- Files named with 8-character hash prefix and timestamp
- Import summary displays "Rejected entries saved to: rejected/" when applicable

#### SMS Import

**MessageEntry Implementation:**
- Wraps `Message` interface (SMS/MMS) to implement `Entry` interface
- Hash calculation includes all fields except: readable_date, contact_name
- Supports both SMS and MMS messages with polymorphic handling

**Validation Rules:**
- Required fields: date (>0), address (non-empty)
- SMS: valid type (1=received, 2=sent)
- MMS: valid msg_box (1=received, 2=sent)
- MMS parts validated for content type and structure
- Invalid entries marked for rejection (not yet written to files)

**Type System:**
- All timestamps stored as `int64` (epoch milliseconds)
- Boolean fields stored as `int` (0 or 1) for XML compatibility
- Consistent with calls implementation for uniformity

#### Import Process

1. **Repository Validation**: Comprehensive validation ensuring valid structure before any import operations (FEAT-036)
2. **Load Existing Data**: 
   - Build deduplication index from repository
   - Load contacts.yaml including existing unprocessed entries
   - Track initial entry counts per year for summary reporting (FEAT-037)
3. **Process Files**: 
   - Stream XML parsing for memory efficiency
   - Validate each entry
   - Extract contact names after validation but before deduplication (FEAT-035)
   - Extract attachments from MMS messages with debug logging (FEAT-034)
   - Check for duplicates via hash (excluding contact_name)
   - Track added/duplicate entries per year during processing (FEAT-037)
   - Accumulate valid entries
4. **Single Write Operation**:
   - Merge existing and new entries
   - Sort by timestamp (stable for same timestamps)
   - Partition by year
   - Write atomically to repository
   - Save contacts.yaml with extracted names in structured unprocessed format (FEAT-035)
   - Generate accurate yearly summary with mathematics validation (FEAT-037)

#### Year-Based Statistics Tracking (FEAT-037)

**YearTracker Implementation:**
```go
type YearTracker struct {
    initial    map[int]int  // Entries loaded from existing repository
    added      map[int]int  // New entries added during import
    duplicates map[int]int  // Duplicate entries found during import
}

// Core methods
func (yt *YearTracker) trackInitialEntry(entry BackupEntry)
func (yt *YearTracker) trackImportEntry(entry BackupEntry, wasAdded bool)
func (yt *YearTracker) validateMathematics() error
```

**Statistics Validation:**
- **Mathematics Check**: Validates Initial + Added = Final for each year
- **Entry-Level Tracking**: Tracks statistics at individual entry level during processing
- **Memory Efficient**: Uses simple map[int]int counters (minimal overhead)
- **Error Detection**: Logs warnings if mathematics validation fails

#### Attachment Extraction (FEAT-012)

The AttachmentExtractor module handles extraction of user-attached content from MMS messages during the SMS import process.

**Integration Point:**
- Attachment extraction occurs after validation but before deduplication
- Only user-attached content is extracted (images, videos, audio, documents)
- System-generated content (SMIL, text, vCards) is left inline

**Content Type Filtering:**
```go
// Extractable content types
ExtractableTypes: []string{
    // Images
    "image/jpeg", "image/png", "image/gif", "image/bmp", "image/webp",
    // Videos  
    "video/mp4", "video/3gpp", "video/quicktime", "video/avi",
    // Audio
    "audio/mpeg", "audio/mp4", "audio/amr", "audio/wav",
    // Documents
    "application/pdf", "application/msword", 
    "application/vnd.openxmlformats-*",
}

// Skipped content types (left inline)
SkippedTypes: []string{
    "application/smil",           // SMIL presentation layouts
    "text/plain",                 // Message body text
    "text/x-vCard",              // Contact cards
    "application/vnd.wap.*",     // WAP protocol containers
}
```

**Extraction Process:**
1. Parse MMS parts for `data` attributes containing base64-encoded content
2. Filter by content type (extract only user content, skip system content)
3. Skip files smaller than 1KB (likely metadata)
4. Decode base64 data and calculate SHA-256 hash
5. Store in `attachments/[first-2-chars]/[full-hash]` structure
6. Update MMS part with path reference and metadata
7. Handle deduplication (reference existing files, extract only new ones)

**Debug Logging (FEAT-034):**
- **Comprehensive Tracing**: Debug logs track every step of attachment extraction pipeline
- **Content Type Filtering**: Logs decisions about which parts to extract vs skip
- **Deduplication Tracking**: Logs when attachments already exist vs new extractions
- **Error Context**: Detailed error logging for base64 decode failures and file operations
- **Performance Monitoring**: Logs processing time and attachment statistics
- **Verification Steps**: Logs file size validation and hash calculation results

**Enhanced MMSPart Structure:**
After extraction, MMS parts are updated with new fields:
- `path`: Repository-relative path to extracted file
- `original_size`: Size of decoded attachment in bytes
- `extraction_date`: ISO8601 timestamp when extracted
- `data` attribute is removed after successful extraction

**Error Handling:**
- Base64 decode failures: Reject entire SMS with reason "malformed-attachment"
- File write failures: Reject entire SMS with reason "attachment-write-error"
- Invalid/unknown content types: Skip extraction, leave data inline
- Transaction semantics: If any user attachment fails, reject entire SMS

### Performance Characteristics

- **Memory Efficiency**: Streaming XML parsing, progress every 100 records
- **Thread Safety**: Concurrent-safe coalescer operations
- **Single Write**: Repository updated only once after all processing
- **Error Resilience**: Continue on individual failures, track statistics

## Manual Review Workflow for Extracted Contacts

After importing messages with contact names, users should follow this workflow to review and organize extracted contacts:

### 1. Review Extracted Contacts
After running import, check `contacts.yaml` for the `unprocessed` section:
```yaml
unprocessed:
  - "5551234567: John Doe"
  - "5551234567: Johnny"
  - "5551234567: J. Doe"
  - "5559876543: Jane Smith"
```

### 2. Identify Duplicate Variations
Look for the same phone number with different name variations:
- Multiple entries for `5551234567` suggest the same person with different name formats
- Decide which name is the canonical/preferred version

### 3. Manual Editing Process
Edit `contacts.yaml` to promote names from `unprocessed` to the main `contacts` section:

**Before:**
```yaml
contacts:
  - name: "Existing Contact"
    numbers: ["5550000000"]

unprocessed:
  - "5551234567: John Doe"
  - "5551234567: Johnny"
  - "5559876543: Jane Smith"
```

**After manual review:**
```yaml
contacts:
  - name: "Existing Contact"
    numbers: ["5550000000"]
  - name: "John Doe"  # Promoted from unprocessed
    numbers: ["5551234567"]
  - name: "Jane Smith"  # Promoted from unprocessed
    numbers: ["5559876543"]

unprocessed:
  # Removed processed entries, keep any that need further review
```

### 4. Best Practices
- **Consolidate variations**: Choose one canonical name per person
- **Verify phone numbers**: Ensure numbers are correct before promoting
- **Keep context**: Use message timestamps to help identify contacts
- **Incremental review**: Process a few contacts at a time to avoid overwhelm
- **Backup first**: Keep a backup of contacts.yaml before major edits

### 5. Common Scenarios

**Scenario 1: Same person, multiple name formats**
```yaml
unprocessed:
  - "5551234567: John Doe"
  - "5551234567: Johnny"
  - "5551234567: J. Doe"
```
**Resolution**: Choose "John Doe" as canonical, remove duplicates

**Scenario 2: Unknown contacts**
```yaml
unprocessed:
  - "5559999999: Unknown"
  - "5558888888: Mom"
```
**Resolution**: Keep "Mom", research or skip "Unknown"

**Scenario 3: Business vs personal**
```yaml
unprocessed:
  - "5551234567: John Doe"
  - "5551234567: Acme Corp"
```
**Resolution**: Determine if John works at Acme, choose appropriate name

### 6. Future Import Behavior
- Subsequent imports will add new unprocessed entries
- Existing processed contacts remain in main section
- New variations of existing numbers will still appear in unprocessed
- Manual review process repeats for new entries only

## Development Workflow with Claude Commands

### Code Formatting (FEAT-027)

Automatic code formatting is integrated into the development workflow to ensure consistent code style:

**Formatting Requirements:**
- All Go code must be formatted using `devbox run formatter` (executes `go fmt ./...`)
- Formatting is mandatory before testing and committing
- Follows standard Go formatting conventions without custom configuration

**Integration Points:**
- **Quality Pipeline**: format → test → lint → build (enforced order)
- **Agent Behavior**: All agents automatically format code before auto-committing
- **Task Completion**: Formatting verification required before marking tasks complete
- **Development Commands**: Documented formatting commands in CLAUDE.md

**Implementation:**
- Uses existing `devbox run formatter` script in devbox.json
- Integrated into agent auto-commit workflow in `.claude/commands/implement-issue.md`
- Quality verification process updated to include formatting as first step
- Comprehensive documentation in CLAUDE.md Code Formatting Best Practices section

### Auto-Commit Behavior

All Claude commands and agents are configured with automatic commit functionality to streamline the development workflow:

#### When Auto-Commit Occurs
- After completing each TodoWrite task in `/implement-issue`
- After creating feature documents with `/create-feature`  
- After creating bug documents with `/create-bug`
- After completing documentation updates in agents

#### File Detection Strategy
Agents use git status comparison to identify only the files they modified during a task:

```bash
# Before starting task
git status --porcelain > /tmp/before_task

# After completing task
git status --porcelain > /tmp/after_task

# Stage only changed files (never use git add .)
comm -13 /tmp/before_task /tmp/after_task | cut -c4- | xargs -r git add
```

#### Commit Message Format
```
[ISSUE-ID]: [Brief task description]

[Optional: Details about implementation]

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

#### Key Features
- **Smart File Detection**: Only stages files actually modified during the task
- **Verification First**: Auto-commit only occurs after successful tests, build, and lint verification
- **Error Handling**: Agents stop and ask for user guidance if commits fail unexpectedly
- **Standard Format**: Consistent commit messages with issue references and proper attribution

This auto-commit behavior ensures a clear development history while reducing manual overhead in the issue implementation workflow.

## Continuous Integration with Devbox (FEAT-026)

### Overview

The project uses devbox for continuous integration to ensure consistency between local development and CI environments. All CI workflows use the same tool versions and configurations as local development.

### CI Pipeline

The CI pipeline is defined as a single devbox script that executes all quality checks in sequence:

```json
{
  "scripts": {
    "ci": [
      "devbox run formatter",
      "devbox run tests", 
      "devbox run linter",
      "devbox run build-cli"
    ]
  }
}
```

### GitHub Actions Integration

Both CI workflows use the `jetify-com/devbox-install-action@v0.11.0` instead of direct tool installation:

#### Test Workflow (.github/workflows/test.yml)
- Triggers on: pushes to main, pull requests to main, manual dispatch
- Single job that runs `devbox run ci`
- Replaces separate lint and test jobs with unified CI pipeline

#### Release Workflow (.github/workflows/release.yml) 
- Triggers on: tag pushes, manual dispatch
- Runs CI pipeline before building release binaries via `pre_command: devbox run ci`
- Uses same Go 1.24 version as local development

### Benefits

**Environment Consistency:**
- Same Go version (1.24) in local development and CI
- Same golangci-lint version and configuration
- Same build and test commands
- No version drift between environments

**Simplified Maintenance:**
- Tool versions managed in single location (devbox.json)
- No need to update multiple workflow files when dependencies change
- Reduced CI configuration complexity

**Developer Experience:**
- `devbox run ci` provides local CI simulation
- Developers can verify changes before pushing
- Consistent failure modes between local and CI environments

### Pipeline Steps

1. **Formatting** (`go fmt ./...`): Ensures consistent code style
2. **Testing** (`go test -v -covermode=set ./...`): Runs full test suite with coverage
3. **Linting** (`golangci-lint run`): Static analysis and code quality checks
4. **Building** (`go build` with version injection): Validates compilation

### Migration Details

**Previous Approach:**
- Direct Go installation using `actions/setup-go@v2` with Go 1.16.x
- Separate golangci-lint action installation
- Different tool versions between local development and CI

**Current Approach:**
- Unified devbox environment using `jetify-com/devbox-install-action@v0.11.0`
- Go 1.24 (significant version upgrade from 1.16.x)
- All tools and versions consistent with local development environment
- Single `devbox run ci` command for complete pipeline execution

## Consistent Versioning Scheme (FEAT-031)

### Overview

The project uses a git tag-based versioning system with VERSION file fallback, providing a single source of truth for version information across CLI, SonarQube, and GitHub Actions.

### Version Formats

- **Development builds**: `2.0.0-dev-g1234567` (VERSION file base + git hash)
- **Release builds**: `2.0.0` (clean semantic version from git tags)
- **VERSION file content**: `2.0.0-dev` (base version with -dev suffix)
- **Git tag format**: `v2.0.0` (with "v" prefix following Go conventions)

### Version Extraction Logic

The build system follows this priority order:

1. **Git tags** (release builds): Use `git describe --tags --exact-match` for tagged commits
2. **VERSION file + git hash** (development builds): Combine base version with git commit hash
3. **VERSION file only** (fallback): When git is unavailable or in CI shallow clones
4. **Hardcoded fallback**: `"dev"` when neither git nor VERSION file available

### Implementation Components

#### Version Extraction Script (`scripts/build-version.sh`)
- Handles all version extraction scenarios with proper edge case handling
- Supports git tag-based releases and development builds
- Graceful fallbacks for missing git or VERSION file
- Used by devbox build-cli command for consistent version injection

#### Build System Integration
- **devbox.json**: `build-cli` command uses version extraction script
- **GitHub Actions**: Updated workflows with proper git history access (`fetch-depth: 0`)
- **SonarQube Integration**: Dynamic version passing via command-line arguments

#### Validation and Documentation
- **Version Validation Script** (`scripts/validate-version.sh`): Verifies VERSION file format
- **Comprehensive Documentation**: Developer workflow, troubleshooting, and checklists
- **User Documentation**: README.md updated with versioning section

### Benefits

**Single Source of Truth:**
- VERSION file provides base version for all scenarios
- Git tags override for clean release versions
- No version duplication across configuration files

**Automated Integration:**
- GitHub Actions automatically extract and use appropriate versions
- SonarQube receives clean semantic versions for project tracking
- CLI binaries always have meaningful version information

**Developer Experience:**
- Simple `devbox run build-cli` for version-aware builds
- `devbox run validate-version` for version file verification
- Clear checklists for version update workflows
- Automatic handling of all edge cases and fallbacks

### Versioning Workflow

#### Development Phase
1. VERSION file contains base version with `-dev` suffix (e.g., `2.1.0-dev`)
2. Builds show format: `2.1.0-dev-g1234567` (base + git hash)
3. SonarQube uses base version: `2.1.0` (clean semantic version)

#### Release Phase
1. Create git tag: `git tag v2.1.0`
2. Tagged builds show clean version: `2.1.0`
3. GitHub Actions triggered automatically
4. Update VERSION file for next development cycle: `2.2.0-dev`

This versioning scheme provides consistency, automation, and clear upgrade paths while minimizing maintenance overhead.

## SonarQube Cloud Integration (FEAT-030)

### Overview

The project integrates with SonarQube Cloud for automated code quality analysis, providing continuous monitoring of code quality metrics, test coverage, and security vulnerabilities. This integration complements the existing CI pipeline and development practices.

### Key Features

**Automated Analysis:**
- Quality Gate enforcement to prevent merging substandard code
- Real-time coverage tracking with historical trends
- Security vulnerability detection and reporting
- Code smell identification for maintainability improvements
- Duplication analysis and technical debt tracking

**Seamless Integration:**
- Runs automatically on pull requests and main branch commits
- Integrates with existing devbox-based CI pipeline
- Dynamic version extraction using FEAT-031 versioning system
- Coverage reports generated using standard Go tooling

### Configuration Files

#### SonarQube Properties (`sonar-project.properties`)
- **Organization**: `phillipgreenii`
- **Project Key**: `phillipgreenii_mobilecombackup`
- **Dynamic Version**: Uses base version from FEAT-031 (e.g., `2.0.0` without `-dev` suffix)
- **Go-specific Settings**: Configured for Go project structure and testing patterns
- **Coverage Integration**: Uses `coverage.out` from `go test -coverprofile` command

#### GitHub Actions Integration
- Extended existing CI workflow with SonarQube analysis step
- Uses `SonarSource/sonarqube-scan-action@v2` for analysis
- Requires `SONAR_TOKEN` repository secret for authentication
- Coverage generation via `devbox run go test -coverprofile=coverage.out`

### Quality Metrics and Badges

**README.md Badges:**
- Quality Gate Status: Overall project health indicator
- Coverage: Test coverage percentage with trending
- Maintainability Rating: Code maintainability assessment

**Analysis Exclusions:**
- Test files (`**/*_test.go`): Excluded from source analysis, included in test analysis
- Test data (`**/testdata/**`): Excluded from all analysis
- Vendor dependencies (`**/vendor/**`): Excluded from all analysis
- Temporary files (`**/tmp/**`): Excluded from all analysis

### Integration Benefits

**Code Quality Assurance:**
- Prevents regression in code quality through quality gates
- Identifies maintainability issues before they accumulate
- Tracks technical debt and provides remediation guidance

**Coverage Monitoring:**
- Monitors test coverage trends over time
- Identifies untested code paths and components
- Integrates with existing comprehensive test suite

**Security Analysis:**
- Scans for common Go security vulnerabilities
- Provides security hotspot identification
- Complements existing linting and static analysis

**Team Collaboration:**
- Centralized quality metrics accessible to all team members
- Historical tracking for project health assessment
- Integration with pull request workflows for quality discussions

### CI Pipeline Integration

The SonarQube analysis integrates seamlessly with the existing CI pipeline:

1. **Code Formatting**: `devbox run formatter` (existing)
2. **Test Execution**: `devbox run go test -coverprofile=coverage.out` (enhanced for coverage)
3. **Linting**: `devbox run linter` (existing)  
4. **CLI Build**: `devbox run build-cli` (existing)
5. **SonarQube Analysis**: Runs after successful completion of all previous steps

This integration maintains the established quality workflow while adding comprehensive code quality analysis and historical tracking capabilities.

