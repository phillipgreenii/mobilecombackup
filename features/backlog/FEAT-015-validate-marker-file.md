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
- [ ] Validate .mobilecombackup contains valid YAML (validate YAML structure before checking fields)
- [ ] Validate repository_structure_version key exists
- [ ] Validate repository_structure_version value is exactly "1" (only latest version supported)
- [ ] Validate created_at key exists and is a valid RFC3339 timestamp
- [ ] Validate created_by key exists
- [ ] Include .mobilecombackup in files.yaml manifest
- [ ] Mark missing .mobilecombackup as a fixable violation with suggested fix content
- [ ] Log warnings for extra fields but do not create validation violations

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
    // Validate YAML structure first
    // Parse YAML content
    // Validate required fields
    // Check for extra fields and log warnings
    // Return violations
}

// FixableViolation includes suggested fix content
type FixableViolation struct {
    ValidationViolation
    SuggestedFix string // YAML content that would fix the violation
}

// MarkerFileContent represents the .mobilecombackup file structure
type MarkerFileContent struct {
    RepositoryStructureVersion string `yaml:"repository_structure_version"`
    CreatedAt                  string `yaml:"created_at"`
    CreatedBy                  string `yaml:"created_by"`
}
```

### Implementation Notes
- Validate marker file first in structure validation phase (before other validations)
- Use gopkg.in/yaml.v3 for parsing (already a dependency)
- Missing file violation should have type "missing_marker_file" with fixable flag and suggested content
- Invalid content violations should specify what's wrong (missing key, invalid version, invalid RFC3339 format)
- Only version "1" is supported (reject all others)
- If version is unsupported, skip remaining repository validations
- Extra fields should be logged as warnings but not create violations
- Validate YAML structure before attempting to parse fields for clearer errors
- Include default marker file content in fixable violation for missing files

## Tasks
- [ ] Add MarkerFileValidator to pkg/validation
- [ ] Update RepositoryValidator to validate marker file first
- [ ] Add .mobilecombackup to required files list
- [ ] Implement YAML structure validation before field parsing
- [ ] Implement RFC3339 timestamp validation
- [ ] Include suggested fix content in fixable violations
- [ ] Add logic to skip further validation if version unsupported
- [ ] Implement warning logging for extra fields
- [ ] Write unit tests for marker file validation
- [ ] Update integration tests to include .mobilecombackup
- [ ] Update validation report format documentation

## Testing
### Unit Tests
- Test validation with missing .mobilecombackup file (verify suggested fix content)
- Test validation with malformed YAML content
- Test validation with valid YAML but missing repository_structure_version
- Test validation with missing created_at or created_by
- Test validation with invalid RFC3339 timestamp format
- Test validation with unsupported version number (not "1")
- Test validation with valid marker file
- Test that extra fields generate warnings but not violations
- Test that validation stops if version is unsupported

### Integration Tests
- Update existing repository validation tests to include .mobilecombackup
- Test fixable violation reporting with suggested fix content
- Test that validation halts when version is unsupported

### Edge Cases
- Empty .mobilecombackup file
- Malformed YAML (invalid syntax)
- Extra unexpected keys in marker file (should warn but not fail)
- Non-string version value
- Various invalid RFC3339 formats (missing timezone, wrong format)
- Version "2" or other future versions (should halt validation)

## Risks and Mitigations
- **Risk**: Breaking existing validation for repositories without marker file
  - **Mitigation**: Mark as fixable violation with clear suggested fix content
- **Risk**: Future version compatibility when repository structure changes
  - **Mitigation**: Only accept current version "1", future versions require migration
- **Risk**: Unclear error messages for YAML parsing failures
  - **Mitigation**: Validate YAML structure separately before field validation

## References
- Pre-req: FEAT-014 (Add Init Subcommand)
- Extends: FEAT-001 (Repository Validation)
- Related: FEAT-011 (Validation Autofix) - will handle fixing missing marker file

## Notes
- This validation ensures all repositories can be tracked for future migrations
- The fixable nature allows existing repositories to be upgraded gracefully
- Only version "1" is supported; future versions will require explicit migration
- Extra fields in the marker file generate warnings but are not validation violations
- RFC3339 is the only accepted timestamp format for consistency
- Validation should halt early if the repository version is unsupported to prevent confusion