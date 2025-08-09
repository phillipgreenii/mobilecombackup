# BUG-021: Import doesn't generate files.yaml

## Status
- **Reported**: 2025-08-08
- **Fixed**: 
- **Priority**: high
- **Severity**: major

## Overview
The import command does not generate a `files.yaml` manifest in the repository root, which should list all files with checksums for integrity verification.

## Reproduction Steps
1. Initialize a new repository with `mobilecombackup init`
2. Import data with `mobilecombackup import`
3. Check for `files.yaml` in the repository root
4. Observe that this manifest file is missing

## Expected Behavior
After import, the repository root should contain a `files.yaml` manifest:

```yaml
version: "1.0"
generated: "2025-08-08T10:30:00Z"
generator: "mobilecombackup v1.2.3"
files:
  - name: ".mobilecombackup.yaml"
    size: 125
    checksum: "sha256:abc123..."
    modified: "2025-08-08T10:30:00Z"
  - name: "summary.yaml"
    size: 1250
    checksum: "sha256:def456..."
    modified: "2025-08-08T10:30:00Z"
  - name: "calls/calls-2014.xml"
    size: 125000
    checksum: "sha256:abc123..."
    modified: "2025-08-08T10:30:00Z"
  - name: "calls/calls-2015.xml"
    size: 89000
    checksum: "sha256:def456..."
    modified: "2025-08-08T10:30:00Z"
  - name: "sms/sms-2014.xml"
    size: 225000
    checksum: "sha256:ghi789..."
    modified: "2025-08-08T10:30:00Z"
  - name: "attachments/ab/ab123456789..."
    size: 45000
    checksum: "sha256:jkl012..."
    modified: "2025-08-08T10:30:00Z"
  - name: "contacts/contacts.yaml"
    size: 5000
    checksum: "sha256:mno345..."
    modified: "2025-08-08T10:30:00Z"
```

## Actual Behavior
No `files.yaml` manifest is generated in the repository root during import.

## Environment
- Version: Current main branch
- OS: All platforms affected
- Go version: 1.24

## Root Cause Analysis
### Investigation Notes
The import process needs to generate a repository-wide manifest file that includes all files in the repository.

### Root Cause
The manifest generation functionality was not implemented in the import command.

## Fix Approach
1. Define the `files.yaml` structure with version, timestamp, and generator info
2. Walk the entire repository after import to collect all files
3. Calculate SHA-256 checksums for each file
4. Write the manifest to the repository root
5. Replace entire manifest on each import (no merging)

## Tasks
- [ ] Define Go structs for files.yaml (FileManifest, FileEntry)
- [ ] Implement repository walker to collect all files
- [ ] Implement SHA-256 checksum calculation
- [ ] Create manifest writer using yaml.v3
- [ ] Integrate manifest generation at end of import workflow
- [ ] Add manifest generation to import summary stats
- [ ] Write unit tests for manifest generation
- [ ] Write integration tests for import with manifest
- [ ] Update validation command to verify files.yaml correctness

## Testing
### Regression Tests
- Test manifest generation includes all file types
- Test checksum accuracy for various file sizes
- Test manifest replacement on re-import
- Verify YAML formatting and structure
- Test with empty repository
- Test with large repository (1000+ files)
- Test file permission errors during manifest generation

### Verification Steps
1. Run import with test data
2. Verify files.yaml exists in repository root
3. Verify all repository files are listed in manifest
4. Verify checksums match actual files
5. Run validation to ensure manifest is verified
6. Re-import and verify manifest is replaced

## Workaround
None - this is required for repository integrity verification.

## Related Issues
- BUG-020: Import not updating summary.yaml (similar metadata issue)
- FEAT-007: Repository validation (will verify manifest correctness)
- FEAT-006: Init command (files.yaml not created during init, only during import)

## Notes
- This is critical for repository integrity
- The validation command will verify manifest correctness as part of its checks
- The manifest is generated at the end of import, after all files are written
- SHA-256 checksums are used for all files
- The manifest is replaced entirely on each import (no merging)
- The manifest includes ALL files in the repository (including .mobilecombackup.yaml, summary.yaml, etc.)