# FEAT-001: Repository Validation

## Status
- **Completed**: -
- **Priority**: high

## Overview
The repository is what will hold all of the records.  To ensure it stays trustworthy, we need to have define a validation process.
This feature will define how to validate the correctness of a given repository. Details on how to create and modify a repository are outside the scope of this feature.

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
- [ ] Check attachment files follow the `[2-char-prefix]/[full-hash]` structure
- [ ] Verify attachment directory names are exactly 2 lowercase hex characters

#### Content Validation
- [ ] Validate XML schema for all `calls-YYYY.xml` files
- [ ] Validate XML schema for all `sms-YYYY.xml` files
- [ ] Verify year in filename matches actual record dates (e.g., `calls-2015.xml` contains only calls for the year 2015, as per UTC)
- [ ] Check that attachment references in SMS/MMS point to existing files
- [ ] Validate `contacts.yaml` and `summary.yaml` against their schemas

#### Cross-File Consistency
- [ ] Verify `summary.yaml` `counts` match actual record counts
- [ ] Verify `summary.yaml` `size_bytes` match actual file sizes
- [ ] Check that all attachments referenced in SMS files exist in `attachments/`
- [ ] Verify no orphaned attachments (files not referenced by any SMS)

### Non-Functional Requirements
- [ ] Scalable directory structure (avoid too many files per directory)
- [ ] Clear violation messages
- [ ] Performance: Validation should complete in reasonable time for large repositories (>1GB)
- [ ] Memory efficiency: Stream processing for large XML files to avoid loading all into memory
- [ ] Detailed reporting: Generate validation report with specific violations and locations
- [ ] Exit codes: Clear exit codes (0=valid, 1=violations found, 2=validation error)

## Design Approach

### Validation Phases
1. **Structure Validation**: Check directory/file structure exists
2. **Manifest Validation**: Verify files.yaml completeness and accuracy
3. **Checksum Validation**: Verify all SHA-256 hashes
4. **Schema Validation**: Check XML/YAML files against schemas
5. **Consistency Validation**: Cross-check references and counts

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
- [ ] Create XML schema validators for calls and SMS
- [ ] Create YAML schema validators for contacts and summary
- [ ] Implement attachment reference checker
- [ ] Build cross-file consistency validator
- [ ] Design and implement validation report generator
- [ ] Add performance optimizations for large repositories
- [ ] Write comprehensive test suite

## Testing Approach
- Unit tests for each validation component
- Integration tests with sample valid/invalid repositories
- Performance tests with large repositories
- Edge cases: empty repository, corrupted files, missing directories, missing files

## References
- See section `Repository` in `specification.md`
