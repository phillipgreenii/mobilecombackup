# FEAT-012: Extract Attachments

## Status
- **Completed**: 2025-08-10
- **Priority**: high

## Overview
When importing SMS, if there is an file as part of the message, it should be extracted and added to the repository as an attachment.
The SMS will then be updated to reference the attachment using relative paths.

## Background
The backups encode the files in base64 in an attribute in the XML.  These large attributes can make working with the XML files difficult to work with. Not all attachments represent user content - many are system-generated files (SMIL presentation layouts, contact cards, etc.) that should be filtered out.

## Requirements
### Functional Requirements
- [ ] Attachment extraction will occur as part of the import process after FEAT-009 is complete
- [ ] Extract only user-attached content (images, videos, audio, documents) - skip system files
- [ ] Extract the file from the entry and hash it with SHA-256
- [ ] If the file does not exist in the repository, it will be added
- [ ] Store attachment as `attachments/[first-2-chars]/[full-hash]`
- [ ] SMS row will be updated to point to attachment using repository-relative path (e.g., `attachments/ab/abc123...`)
- [ ] Handle MMS with multiple parts (preserve attachment sequence order using `seq` attribute)
- [ ] If base64 decoding fails for any attachment, reject the entire SMS entry using rejection system
- [ ] `import` command will include the number of unique attachments added and how many times existing attachments were referenced
- [ ] Report duplicate attachment counts per type and year
- [ ] Handle empty attachments appropriately (skip extraction, leave inline)
- [ ] Skip system-generated content types:
  - `application/smil` (SMIL presentation layouts)
  - `text/plain` (message body text)
  - `text/x-vCard` (contact cards)
  - `application/vnd.wap.multipart.related` (WAP containers)

### Non-Functional Requirements
- [ ] Handle large datasets efficiently
- [ ] Stream large attachments to disk to avoid memory issues
- [ ] Performance target: Process 5,000 messages/second with attachments
- [ ] Memory usage: Stay under 100MB even with large attachments
- [ ] Implement checksum verification after extraction

## Attachment Extraction Criteria

### Content to Extract
Only extract user-generated content that represents actual attachments:
- **Images**: `image/jpeg`, `image/png`, `image/gif`, `image/bmp`, `image/webp`
- **Videos**: `video/mp4`, `video/3gpp`, `video/quicktime`, `video/avi`
- **Audio**: `audio/mpeg`, `audio/mp4`, `audio/amr`, `audio/wav`
- **Documents**: `application/pdf`, `application/msword`, `application/vnd.openxmlformats-*`

### Content to Skip (Leave Inline)
Do not extract system-generated or metadata content:
- **SMIL**: `application/smil` - Presentation layout definitions
- **Text**: `text/plain` - Message body content
- **vCards**: `text/x-vCard` - Contact information
- **WAP**: `application/vnd.wap.*` - WAP protocol containers
- **Empty data**: Parts with no data or null data attributes
- **Small files**: Files under 1KB (likely metadata)

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
     - Update `<part>` element:
       - Replace `data` attribute with `path` attribute
       - Add `original_size` and `extraction_date` attributes

### Enhanced Metadata Schema
Metadata remains in the SMS XML part element:
```xml
<part seq="0" ct="image/png" fn="duck.png" path="attachments/ab/abc123..." 
      original_size="45678" extraction_date="2024-01-15T10:30:00Z" />
```

Additional metadata attributes:
- `path`: Repository-relative path to extracted file
- `original_size`: Size of decoded attachment in bytes
- `extraction_date`: When the attachment was extracted (ISO8601)

### SMS Update Strategy
- Remove `data` attribute from `<part>` element
- Add `path` attribute with repository-relative path
- Preserve all other attributes in the `<part>` element

### Hash Calculation Impact
- Before FEAT-012: SMS hash includes full base64 attachment data
- After FEAT-012: SMS hash includes only the attachment reference path
- This ensures consistent deduplication behavior

### Error Handling
- Base64 decode failures: Reject entire SMS entry with reason "malformed-attachment"
- File write failures: Reject entire SMS entry with reason "attachment-write-error"
- Invalid content type: Skip extraction, log warning, keep data inline
- Maintain transaction semantics: If any user attachment fails, reject entire SMS

## Tasks
- [x] Create attachment extraction module with content type filtering
- [x] Implement content type detection and filtering logic
- [x] Implement base64 decoding with error handling
- [x] Implement SHA-256 hashing for content addressing
- [x] Create directory structure for attachments
- [x] Integrate with SMS import process (post-FEAT-009)
- [x] Update SMS XML structure to use path references for extracted attachments
- [x] Add attachment deduplication tracking with detailed statistics
- [x] Implement streaming for large attachments
- [x] Add comprehensive unit tests for various attachment types
- [x] Add tests for content type filtering (ensure SMIL/text not extracted)
- [x] Add integration tests for end-to-end attachment extraction
- [x] Test edge cases (empty, corrupt, very large attachments)
- [ ] Update import summary to show attachment statistics by content type (minor enhancement left for future)

## Testing Requirements
- [x] Unit tests for base64 decoding with various content types
- [x] Tests for corrupt/invalid base64 data
- [x] Tests for empty attachments
- [x] Tests for very large attachments (memory efficiency)
- [x] Tests for MMS with multiple attachments
- [x] Tests for attachment deduplication
- [x] Integration tests with full import flow

## References
- Depends on: FEAT-009-import-sms (must be completed first)
- Related: FEAT-003-read-sms-from-repository
- Related: FEAT-010-add-import-subcommand

