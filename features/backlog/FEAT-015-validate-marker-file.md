# FEAT-015: Validate .mobilecombackup Marker File

## Status
- **Completed**: Not yet started
- **Priority**: high (dependency for FEAT-014)

## Overview
Update repository validation (FEAT-001) to validate the presence and content of the `.mobilecombackup` marker file. This file contains key-value pairs including the repository structure version, enabling future migration paths.

## Background
FEAT-014 introduces a `.mobilecombackup` marker file to identify initialized repositories and track the repository structure version. This file needs to be validated as part of the repository validation process to ensure all repositories have proper version tracking.

## Requirements
### Functional Requirements
- [ ] Add .mobilecombackup to the list of required files in repository validation
- [ ] Validate .mobilecombackup file exists in repository root
- [ ] Validate .mobilecombackup contains valid YAML
- [ ] Validate repository_structure_version key exists
- [ ] Validate repository_structure_version value is a supported version (currently only "1")
- [ ] Validate created_at key exists and is a valid ISO timestamp
- [ ] Validate created_by key exists
- [ ] Include .mobilecombackup in files.yaml manifest
- [ ] Mark missing .mobilecombackup as a fixable violation (can be created with default values)

### Non-Functional Requirements
- [ ] Clear error messages for marker file violations
- [ ] Maintain backward compatibility with existing validation reports
- [ ] Follow existing validation patterns from FEAT-001

## Design
### Approach
Extend the existing RepositoryValidator to include marker file validation as part of the structure validation phase.

### API/Interface
```go
// MarkerFileValidator validates the .mobilecombackup marker file
type MarkerFileValidator struct {
    repoRoot string
}

// Validate checks the marker file exists and has valid content
func (v *MarkerFileValidator) Validate() ([]ValidationViolation, error) {
    // Check file exists
    // Parse YAML content
    // Validate required fields
    // Return violations
}

// MarkerFileContent represents the .mobilecombackup file structure
type MarkerFileContent struct {
    RepositoryStructureVersion string `yaml:"repository_structure_version"`
    CreatedAt                  string `yaml:"created_at"`
    CreatedBy                  string `yaml:"created_by"`
}
```

### Implementation Notes
- Add to structure validation phase before other file validations
- Use gopkg.in/yaml.v3 for parsing (already a dependency)
- Missing file violation should have type "missing_marker_file" with fixable flag
- Invalid content violations should specify what's wrong (missing key, invalid version, invalid timestamp format)
- Supported versions should be defined as constants

## Tasks
- [ ] Add MarkerFileValidator to pkg/validation
- [ ] Update RepositoryValidator to include marker file validation
- [ ] Add .mobilecombackup to required files list
- [ ] Implement YAML parsing and validation logic
- [ ] Mark missing marker file as fixable violation
- [ ] Write unit tests for marker file validation
- [ ] Update integration tests to include .mobilecombackup
- [ ] Update validation report format documentation

## Testing
### Unit Tests
- Test validation with missing .mobilecombackup file
- Test validation with invalid YAML content
- Test validation with missing repository_structure_version
- Test validation with missing created_at or created_by
- Test validation with invalid ISO timestamp format
- Test validation with unsupported version number
- Test validation with valid marker file

### Integration Tests
- Update existing repository validation tests to include .mobilecombackup
- Test fixable violation reporting

### Edge Cases
- Empty .mobilecombackup file
- Malformed YAML
- Extra unexpected keys in marker file
- Non-string version value

## Risks and Mitigations
- **Risk**: Breaking existing validation for repositories without marker file
  - **Mitigation**: Mark as fixable violation, not critical error
- **Risk**: Version compatibility issues
  - **Mitigation**: Define clear version support matrix

## References
- Pre-req: FEAT-014 (Add Init Subcommand)
- Extends: FEAT-001 (Repository Validation)
- Related: FEAT-011 (Validation Autofix) - will handle fixing missing marker file

## Notes
- This validation ensures all repositories can be tracked for future migrations
- The fixable nature allows existing repositories to be upgraded gracefully
- Future versions might add more fields to the marker file