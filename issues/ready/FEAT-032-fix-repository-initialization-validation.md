# FEAT-032: Fix Repository Initialization and Validation

## Status
- **Ready for Implementation**: 2025-08-13
- **Priority**: high

## Overview
Fix critical issues with repository initialization and validation that prevent proper repository setup and file tracking.

## Background
Multiple issues were identified with repository initialization and validation:
- `init` command creates incomplete repository structure (missing files.yaml and files.yaml.sha256)
- `validate --autofix` creates files.yaml but not files.yaml.sha256
- .mobilecombackup.yaml is not tracked in files.yaml
- `info` command on empty repo doesn't show entity counts (should show 0s)
- Files.yaml validation rules don't align with generation logic
- Directory path discrepancies causing validation failures

## Requirements
### Functional Requirements
- [ ] `init` command creates complete repository structure: directories (calls/, sms/, attachments/), required files (files.yaml, files.yaml.sha256, summary.yaml, contacts.yaml), and marker file (.mobilecombackup.yaml)
- [ ] `validate --autofix` always regenerates files.yaml; only creates files.yaml.sha256 if missing
- [ ] .mobilecombackup.yaml is properly tracked in files.yaml (but files.yaml does not include itself or files.yaml.sha256)
- [ ] `info` command shows counts for all entity types (0 for empty repo)
- [ ] Fix relative path interpretation in files.yaml validation (files showing as both missing and unexpected)
- [ ] Files.yaml validation rules align with generation logic

### Non-Functional Requirements
- [ ] Repository initialization should be atomic (all-or-nothing)
- [ ] Validation should be fast and comprehensive

## Design
### Approach

#### Complete Repository Structure
The `init` command will create a complete, valid repository with this structure:
```
repository/
├── .mobilecombackup.yaml    # Repository marker with metadata
├── files.yaml               # File manifest (excludes itself and .sha256)
├── files.yaml.sha256        # SHA-256 checksum of files.yaml
├── summary.yaml             # Empty import summary
├── contacts.yaml            # Empty contacts list
├── calls/                   # Empty directory for call logs
├── sms/                     # Empty directory for SMS/MMS
└── attachments/             # Empty directory for attachments
```
Note: `rejected/` directory is created only when needed during import.

#### Manifest Generation Strategy
- Extract manifest generation logic from existing autofix code into shared utility
- `files.yaml` includes: .mobilecombackup.yaml, summary.yaml, contacts.yaml, and any files in calls/, sms/, attachments/
- `files.yaml` does NOT include itself, files.yaml.sha256, or anything from rejected/
- Use consistent relative path format with forward slashes (filepath.ToSlash())

#### Path Consistency Fix
The validation failure where files appear as both "missing" and "unexpected" indicates a path interpretation mismatch between generation and validation logic. Fix by:
- Ensuring both use same working directory reference
- Normalizing all paths to forward slashes in YAML
- Using consistent relative path resolution

### API/Interface
```go
// Shared manifest utilities
func GenerateFileManifest(repoRoot string) (*FileManifest, error)
func WriteManifestFiles(repoRoot string, manifest *FileManifest) error
func ValidateManifest(repoRoot string) ([]ValidationViolation, error)

// Updated init command
func InitializeRepository(repoRoot string, quiet bool) error
```

### Implementation Notes
- Autofix behavior: always regenerate files.yaml, only create files.yaml.sha256 if missing
- Atomic init: create all files/directories or rollback on failure
- Path normalization: use filepath.ToSlash() for cross-platform YAML compatibility
- Info command: show "0" counts for all entity types on empty but valid repository

## Tasks
- [ ] Extract manifest generation from autofix into shared utility in pkg/manifest/
- [ ] Update init command to create complete repository structure (core directories and files, not rejected/)
- [ ] Fix autofix logic: always regenerate files.yaml, only create .sha256 if missing
- [ ] Add .mobilecombackup.yaml to files.yaml manifest (exclude files.yaml itself and rejected/ contents)
- [ ] Fix path interpretation mismatch between manifest generation and validation
- [ ] Update info command to show "0" counts for all entity types on empty repo
- [ ] Ensure atomic init operation with proper cleanup on failure
- [ ] Add cross-platform path normalization (filepath.ToSlash)
- [ ] Write comprehensive tests covering all fixed scenarios
- [ ] Update CLI documentation and help text

## Testing
### Unit Tests
- Test manifest generation utility with various repository states
- Test path normalization across Windows/Unix
- Test autofix logic: files.yaml regeneration vs files.yaml.sha256 creation
- Test atomic init rollback on failures

### Integration Tests
- Test complete init → validate → info workflow
- Test init creates all required files and directories (excluding rejected/)
- Test info shows "0" counts on fresh repository
- Test validate passes on init-created repository
- Test autofix on repository missing files.yaml or files.yaml.sha256

### Edge Cases
- Init on existing repository (should fail or warn)
- Init with insufficient permissions
- Validate on repository with path inconsistencies
- Info on partially-created repository
- Cross-platform path handling (Windows backslashes)

## Risks and Mitigations
- **Risk**: Description
  - **Mitigation**: How to handle

## References
- Related features: FEAT-XXX
- Code locations: pkg/module/file.go
- External docs: Link to any external documentation

## Notes
Additional thoughts, questions, or considerations that arise during planning/implementation.