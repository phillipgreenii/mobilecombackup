# FEAT-001: Repository Validation

## Status
- **Completed**: -
- **Priority**: high

## Overview
The repository is what will hold all of the records.  To ensure it stays trustworthy, we need to have define a validation process.
This feature will define how to validate the correctness of a given repository by leveraging the reader components (FEAT-002 through FEAT-005) and performing cross-file consistency checks. Details on how to create and modify a repository are outside the scope of this feature.

## Background
See section `Repository` in `specification.md`

## Requirements
### Functional Requirements

#### Files.yaml Validation
- [ ] A valid repository will always contain `files.yaml` and it will contain references to all files in the repository, except `files.yaml` and `files.yaml.sha256`.
- [ ] It is considered a violation for each file in the repository which is not listed in `files.yaml`.
- [ ] It is a considered a violation for each file in `files.yaml` which is not in the repository.
- [ ] A valid repository will always contain `files.yaml.sha256` and it will contain the SHA-256 of `files.yaml`.
- [ ] Verify that all file entries in `files.yaml` have valid SHA-256 format (64 hex characters)
- [ ] Verify that all file entries have positive `size_bytes` values
- [ ] Check that file paths are relative and don't contain ".." or absolute paths
- [ ] Validate that no duplicate file entries exist

#### SHA-256 Validation
- [ ] Calculate and verify SHA-256 for each file listed in `files.yaml` matches
- [ ] Verify `files.yaml.sha256` contains exactly 64 hex characters
- [ ] Calculate and verify `files.yaml` SHA-256 matches `files.yaml.sha256`

#### Repository Structure Validation
- [ ] Verify required directories exist: `calls/`, `sms/`, `attachments/`
- [ ] Verify required files exist: `contacts.yaml`, `summary.yaml`

#### Summary Validation
- [ ] Validate `summary.yaml` exists and has correct schema
- [ ] Verify summary data using readers from FEAT-002, FEAT-003, FEAT-004, FEAT-005

#### Cross-File Consistency
- [ ] Verify `summary.yaml` `counts` match actual record counts (using CallsReader and SMSReader)
- [ ] Verify `summary.yaml` `size_bytes` match actual file sizes
- [ ] Check that all attachments referenced in SMS files exist (using AttachmentReader)
- [ ] Verify no orphaned attachments (using AttachmentReader with SMS references)

### Non-Functional Requirements
- [ ] Scalable directory structure (avoid too many files per directory)
- [ ] Clear violation messages
- [ ] Performance: Validation should complete in reasonable time for large repositories (>1GB)
- [ ] Memory efficiency: Leverage streaming capabilities of reader components
- [ ] Detailed reporting: Generate validation report with specific violations and locations
- [ ] Exit codes: Clear exit codes (0=valid, 1=violations found, 2=validation error)

## Design Approach

### Validation Phases
1. **Structure Validation**: Check directory/file structure exists
2. **Manifest Validation**: Verify files.yaml completeness and accuracy
3. **Checksum Validation**: Verify all SHA-256 hashes
4. **Content Validation**: Use reader components to validate file contents
5. **Consistency Validation**: Cross-check references and counts using reader APIs

### Error Reporting Format
```yaml
validation_report:
  timestamp: "2024-01-15T10:30:00Z"
  repository_path: "/path/to/repo"
  status: "invalid"
  violations:
    - type: "missing_file"
      severity: "error"
      file: "summary.yaml"
      message: "Required file not found"
    - type: "checksum_mismatch"
      severity: "error"
      file: "calls/calls-2015.xml"
      expected: "abc123..."
      actual: "def456..."
    - type: "orphaned_attachment"
      severity: "warning"
      file: "attachments/ab/ab54363e39"
      message: "Attachment not referenced by any SMS"
```

## Implementation Tasks
- [ ] Implement file manifest reader and validator
- [ ] Implement SHA-256 calculation and verification
- [ ] Integrate CallsReader (FEAT-002) for calls validation
- [ ] Integrate SMSReader (FEAT-003) for SMS validation and attachment tracking
- [ ] Integrate AttachmentReader (FEAT-004) for attachment validation
- [ ] Integrate ContactsReader (FEAT-005) for contacts validation
- [ ] Build cross-file consistency validator using reader APIs
- [ ] Design and implement validation report generator
- [ ] Add performance optimizations for large repositories
- [ ] Write comprehensive test suite

## Testing Approach
- Unit tests for each validation component
- Integration tests with sample valid/invalid repositories
- Performance tests with large repositories
- Edge cases: empty repository, corrupted files, missing directories, missing files

## Dependencies
- FEAT-002: Read Calls from Repository (CallsReader)
- FEAT-003: Read SMS from Repository (SMSReader)
- FEAT-004: Read Attachments from Repository (AttachmentReader)
- FEAT-005: Read Contacts from Repository (ContactsReader)

## References
- See section `Repository` in `specification.md`
