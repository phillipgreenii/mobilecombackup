# FEAT-009: Import SMSs

## Status
- **Completed**: -
- **Priority**: high

## Overview
Import valid sms from SMS backups (`sms-*.xml`).  Invalid sms will be rejected, valid sms will be added to the repository if they aren't already in there.

## Background
Daily backups contain many duplicates from previous days. Duplicates should not occur within the repository.

## Requirements
### Functional Requirements
- [ ] The repository will not contain any duplicate sms
- [ ] Track total number of sms added per year.
- [ ] Track total number of sms not added because they would be duplicates.
- [ ] Repository will persist sms ordered by timestamp and partioned by year.
- [ ] Preserve original entry order for same timestamps
- [ ] Invalid sms will be dumped into a file, with the same relative path, but in a `rejected/` directory.
- [ ] Display import summary with counts (added, duplicates, rejected)
- [ ] Process one file at a time
- [ ] Continue processing remaining SMS after encountering invalid entries

### Non-Functional Requirements
- [ ] Handle large datasets efficiently (thousands of SMS messages)
- [ ] Maintain stability for entries with identical timestamps

## Design/Implementation Approach
### Deduplication Strategy
- Use SHA-256 hash-based approach for detecting duplicates
- Hash calculation excludes `readable_date` field (timezone-dependent)
- If SMS hash equals one already in repository, SMS will not be added
- Otherwise, append new SMS to repository
- Memory store using map[string][V] keyed by hash
- Process existing repository first, then new files

### Validation Criteria
SMS are validated using criteria from FEAT-003:
- Required fields:
  - `date`: Valid timestamp for year partitioning
  - `address`: Phone number or contact identifier
  - `type`: Message type (sent/received/draft)
  - `body`: Message content (can be empty for MMS)
- Optional fields handled gracefully
- Malformed XML structure causes rejection

### Error Handling
Different error types result in rejection:
- `missing-timestamp`: No valid date field
- `malformed-xml`: XML parsing failure
- `invalid-field`: Required field missing or invalid format
- Continue processing other entries on error
- Track rejection counts in summary

### Rejection File Format
Rejected rows will be written to a file in `rejected/` directory
- It will follow the naming pattern of `sms-$originalFileHash-rejects.xml` where `$originalFileHash` is the SHA-256 of the file being imported
- It will be structured the same as a valid "SMS Backup", except each row will be the rejected SMS.
  - The idea behind this is that it should be easy to import the rejected SMS once they have been corrected.
Next to the rejects file, will be a file called `sms-$originalFileHash-violations.yaml` which will contain the violations of each SMS.  It will look similar to 
```yaml
violations:
  - line: 0
    violations:
      - missing-timestamp
      - unknown-type
  - line: 1
    violations:
      - missing-timestamp
```

### Performance Considerations
- Batch processing for large SMS imports
- Progress reporting every 100 entries
- Memory-efficient streaming for XML parsing
- Summary statistics displayed at completion

## Tasks
- [ ] Extend coalescer to handle SMS entries
- [ ] Implement SMS-specific validation rules (reuse FEAT-003 logic)
- [ ] Add SMS hash calculation (exclude `readable_date`)
- [ ] Create rejection file writer for invalid SMS
- [ ] Implement progress reporting for large imports
- [ ] Add SMS import to main command flow
- [ ] Write integration tests for SMS importing
- [ ] Update summary output to include SMS statistics

## References
- Related: FEAT-001-repository-validation
- Related: FEAT-003-sms-from-repository
- Related: FEAT-010-add-import-subcommand
- Related: FEAT-012-extract-attachments
