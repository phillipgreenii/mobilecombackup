# FEAT-036: Add Repository Validation to Import Command

## Status
- **Completed**: YYYY-MM-DD
- **Priority**: high

## Overview
Add full repository validation to the import command to prevent import operations on invalid repositories.

## Background
Currently, the import command runs even when the repository is invalid. The existing validation only checks if the directory exists (pkg/importer/importer.go:174) with a TODO comment to integrate proper validation from FEAT-007.

This allows imports to proceed on repositories missing essential structure (.mobilecombackup.yaml, required directories, etc.), leading to unexpected behavior and potential data corruption.

## Requirements
### Functional Requirements
- [ ] Import command uses full repository validation before any file scanning
- [ ] Import fails fast with clear error message if repository is invalid
- [ ] Validation occurs before scanning files to import (early failure)
- [ ] Error message includes specific validation violations found
- [ ] User must fix repository and retry import (no auto-fix)

### Non-Functional Requirements
- [ ] Repository validation should complete in < 1 second for typical repositories
- [ ] Error messages should be actionable (tell user how to fix)

## Design
### Approach
Replace the basic directory existence check with full repository validation using the existing pkg/validation infrastructure.

### API/Interface
```go
// In pkg/importer/importer.go, replace validateRepository function
func (i *Importer) validateRepository() error {
    validator := validation.NewRepositoryValidator(i.repoRoot, validation.Config{})
    report, err := validator.ValidateRepository()
    if err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    if report.Status != validation.Valid {
        return formatValidationError(report.Violations)
    }
    return nil
}
```

### Implementation Notes
- Validation must occur before any file scanning begins
- Use existing validation.RepositoryValidator from pkg/validation
- Format validation violations into user-friendly error messages
- Remove TODO comment about validation integration

## Tasks
- [ ] Replace basic directory check with full validation from pkg/validation
- [ ] Integrate validation call before file scanning in import workflow
- [ ] Add proper error handling and user-friendly error messages
- [ ] Remove TODO comment about validation integration
- [ ] Write tests for import validation (valid/invalid repositories)
- [ ] Update import command documentation
- [ ] Test integration with existing validation code

## Testing
### Unit Tests
- Test import on repository missing .mobilecombackup.yaml (should fail)
- Test import on repository missing required directories (should fail)
- Test import on valid repository (should proceed)
- Test validation error message formatting

### Integration Tests
- Test import command with invalid repository returns exit code 2
- Test import command validation happens before file scanning
- Test user workflow: fix repository → retry import succeeds

### Edge Cases
- Repository with permission issues
- Repository with corrupted marker file
- Repository with partial structure

## Risks and Mitigations
- **Risk**: Description
  - **Mitigation**: How to handle

## References
- Related features: FEAT-007 (Validate subcommand - provides validation infrastructure)
- Code locations: pkg/importer/importer.go:174 (current validateRepository function)
- Dependencies: pkg/validation (existing validation logic)

## Notes
Additional thoughts, questions, or considerations that arise during planning/implementation.
