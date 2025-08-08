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
   - Read Exisiting Repository
   - Any specified unprocessed backup file
2. Entry Transformation
   - Extract Images
   - Remove Duplicates
3. Order Entries
   - Sort by Timestamp (UTC)
   - Partition by Year
4. Overwrite Repository

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

`contacts.yaml` contains mapping from number to contact name.

```yaml
contacts:
  - name: "Bob Ross"
    numbers: 
      - "+11111111111"
  - name: "<unknown>"
    numbers:
      - "8888888888"
      - "9999999999"
```

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

`summary.yaml` contains a summary of repository contents.

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

## Important Notes

### Contact Name Handling
The `contact_name` field should NOT be used for comparing or identifying duplicate records. This field may vary between backups due to:
- Contact name changes in the phone's address book
- Sync timing differences
- Unknown numbers showing as "(Unknown)" vs actual names

The phone number (`number` for calls, `address` for SMS/MMS) should be used as the primary identifier. The `contacts.yaml` file will maintain the authoritative mapping of numbers to names.

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
- Entry deduplication (excluding `readable_date` and after attachment extraction)

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
}
```

**Key Features:**
- Efficient O(1) phone number lookup using hash maps
- Phone number normalization for consistent matching
- Handles multiple number formats (+1XXXXXXXXXX, (XXX) XXX-XXXX, etc.)
- Special handling for `<unknown>` contact designation
- Duplicate number detection and validation
- Graceful handling of missing contacts.yaml files

#### Contact Structure
```go
type Contact struct {
    Name    string   `yaml:"name"`
    Numbers []string `yaml:"numbers"`
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
   - `summary.yaml` - Initial summary with zero counts
4. Displays tree-style output of created structure

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

### Helper Functions

- `PrintError(format string, args ...interface{})`: Consistent error output to stderr

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
- Files: `rejected/calls-[hash]-[timestamp]-rejects.xml`
- Violations: `rejected/calls-[hash]-[timestamp]-violations.yaml`
- Original XML structure preserved for re-import after correction

#### Import Process

1. **Repository Validation**: Ensure valid structure before import
2. **Load Existing Data**: Build deduplication index from repository
3. **Process Files**: 
   - Stream XML parsing for memory efficiency
   - Validate each entry
   - Check for duplicates via hash
   - Accumulate valid entries
4. **Single Write Operation**:
   - Merge existing and new entries
   - Sort by timestamp (stable for same timestamps)
   - Partition by year
   - Write atomically to repository

### Performance Characteristics

- **Memory Efficiency**: Streaming XML parsing, progress every 100 records
- **Thread Safety**: Concurrent-safe coalescer operations
- **Single Write**: Repository updated only once after all processing
- **Error Resilience**: Continue on individual failures, track statistics

