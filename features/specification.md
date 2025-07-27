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

