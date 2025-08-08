# FEAT-012: Extract Attachments

## Status
- **Completed**: -
- **Priority**: high

## Overview
When importing SMS, if there is an file as part of the message, it should be extracted and added to the repository as an attachment.
The SMS will then be updated to reference the attachment using relative paths.

## Background
The backups encode the files in base64 in an attribute in the XML.  These large attributes can make working with the XML files difficult to work with.

## Requirements
### Functional Requirements
- [ ] Attachment extraction will occur as part of the import process after FEAT-009 is complete
- [ ] Extract the file from the entry and hash it with SHA-256
- [ ] If the file does not exist in the repository, it will be added
- [ ] Store attachment as `attachments/[first-2-chars]/[full-hash]`
- [ ] Store metadata as `attachments/[first-2-chars]/[full-hash].metadata.yaml` with original properties (filename, content type, etc.)
- [ ] SMS row will be updated to point to attachment using repository-relative path (e.g., `attachments/ab/abc123...`)
- [ ] Support all MIME types for attachments
- [ ] Handle MMS with multiple parts (preserve attachment sequence order using `seq` attribute)
- [ ] If base64 decoding fails for any attachment, reject the entire SMS entry (atomic processing)
- [ ] `import` command will include the number of unique attachments added and how many times existing attachments were referenced
- [ ] Report duplicate attachment counts per type and year
- [ ] Handle empty attachments appropriately (store empty file with hash)
- [ ] Preserve all attachment metadata including:
  - Original filename (`fn` attribute)
  - Content type (`ct` attribute)
  - Sequence number (`seq` attribute)
  - Any other attributes present in the part element

### Non-Functional Requirements
- [ ] Handle large datasets efficiently
- [ ] Stream large attachments to disk to avoid memory issues
- [ ] Consider concurrent attachment extraction for performance
- [ ] Define maximum attachment size limits (configurable)
- [ ] Implement checksum verification after extraction

## Design/Implementation Approach

### Attachment Detection
- Parse MMS entries looking for `<parts>` elements
- Each `<part>` element contains:
  - `data`: Base64-encoded file content
  - `ct`: Content type (MIME type)
  - `fn`: Original filename
  - `seq`: Sequence order
  - Other metadata attributes

### Extraction Process
1. For each SMS entry during import:
   - Check if entry contains `<parts>` element
   - For each `<part>`:
     - Decode base64 data
     - Calculate SHA-256 hash of decoded content
     - Check if `attachments/[first-2]/[hash]` exists
     - If not exists:
       - Write decoded content to `attachments/[first-2]/[hash]`
       - Write metadata to `attachments/[first-2]/[hash].metadata.yaml`
     - Update `<part>` element to replace `data` attribute with `path` attribute

### Metadata File Format
```yaml
filename: "duck.png"
content_type: "image/png"
sequence: 0
original_attributes:
  name: "null"
  # any other attributes from the original part element
```

### SMS Update Strategy
- Remove `data` attribute from `<part>` element
- Add `path` attribute with repository-relative path
- Preserve all other attributes in the `<part>` element

### Hash Calculation Impact
- Before FEAT-012: SMS hash includes full base64 attachment data
- After FEAT-012: SMS hash includes only the attachment reference path
- This ensures consistent deduplication behavior

### Error Handling
- Base64 decode failures result in SMS rejection
- File write failures result in SMS rejection
- Maintain transaction semantics: all attachments for an SMS must succeed

## Tasks
- [ ] Create attachment extraction module
- [ ] Implement base64 decoding with error handling
- [ ] Implement SHA-256 hashing for content addressing
- [ ] Create directory structure for attachments
- [ ] Implement metadata YAML writer
- [ ] Integrate with SMS import process (post-FEAT-009)
- [ ] Update SMS XML structure to use path references
- [ ] Add attachment deduplication tracking
- [ ] Implement streaming for large attachments
- [ ] Add comprehensive unit tests for various attachment types
- [ ] Add integration tests for end-to-end attachment extraction
- [ ] Test edge cases (empty, corrupt, very large attachments)
- [ ] Update import summary to show attachment statistics

## Testing Requirements
- [ ] Unit tests for base64 decoding with various content types
- [ ] Tests for corrupt/invalid base64 data
- [ ] Tests for empty attachments
- [ ] Tests for very large attachments (memory efficiency)
- [ ] Tests for MMS with multiple attachments
- [ ] Tests for attachment deduplication
- [ ] Integration tests with full import flow

## References
- Depends on: FEAT-009-import-sms (must be completed first)
- Related: FEAT-003-read-sms-from-repository
- Related: FEAT-010-add-import-subcommand

