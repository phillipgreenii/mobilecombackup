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
- [ ] Import validation is silent unless errors occur (no progress output)
- [ ] Validation violations displayed same as validate subcommand (detailed format)
- [ ] Import exits with code 2 on validation failure (consistent with other commands)
- [ ] User must fix repository and retry import (no auto-fix)
- [ ] Validation strictness identical to validate subcommand

### Non-Functional Requirements
- [ ] Repository validation should complete in < 1 second for repositories up to 10,000 entries
- [ ] Error messages should be actionable (tell user how to fix)
- [ ] No performance impact on valid repositories (validation is fast-fail check)

## Design
### Approach
Replace the basic directory existence check with full repository validation using the existing pkg/validation infrastructure.

### API/Interface
```go
// In pkg/importer/importer.go, replace validateRepository function
func (i *Importer) validateRepository() error {
    // Create readers required for validation
    callsReader := calls.NewXMLCallsReader(i.repoRoot)
    smsReader := sms.NewXMLSMSReader(i.repoRoot)
    attachmentReader := attachments.NewAttachmentManager(i.repoRoot)
    contactsReader := contacts.NewContactsManager(i.repoRoot)
    
    validator := validation.NewRepositoryValidator(
        i.repoRoot,
        callsReader,
        smsReader,
        attachmentReader,
        contactsReader,
    )
    
    report, err := validator.ValidateRepository()
    if err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    if report.Status != validation.Valid {
        return formatValidationError(report.Violations)
    }
    return nil
}

// Format validation violations same as validate subcommand
func formatValidationError(violations []validation.ValidationViolation) error {
    var messages []string
    for _, v := range violations {
        messages = append(messages, fmt.Sprintf("- %s: %s", v.Type, v.Message))
    }
    return fmt.Errorf("repository validation failed:\n%s", strings.Join(messages, "\n"))
}
```

### Implementation Notes
- Validation must occur before any file scanning begins
- Use existing validation.RepositoryValidator from pkg/validation
- Create all four required readers for comprehensive validation
- Format validation violations identical to validate subcommand output
- Silent operation (no progress output) unless validation fails
- Exit with code 2 on validation failure (consistent with validate command)
- Remove TODO comment about validation integration

## Tasks
- [ ] Replace validateRepository function at pkg/importer/importer.go:174
- [ ] Create required readers (calls, SMS, attachments, contacts) for validation
- [ ] Implement formatValidationError function matching validate subcommand output
- [ ] Ensure validation occurs before any file scanning begins
- [ ] Add exit code 2 behavior for validation failures
- [ ] Remove TODO comment about validation integration
- [ ] Write tests for import validation (valid/invalid repositories)
- [ ] Test silent operation (no output unless error)
- [ ] Update import command documentation

## Testing
### Unit Tests
- Test import on repository missing .mobilecombackup.yaml (should fail with exit code 2)
- Test import on repository missing required directories (should fail with exit code 2)
- Test import on valid repository (should proceed silently to file scanning)
- Test validation error message formatting matches validate subcommand
- Test reader creation for validation (all four readers initialized)

### Integration Tests
- Test import command with invalid repository returns exit code 2
- Test import command validation happens before any file scanning
- Test import produces no validation output on valid repository
- Test user workflow: fix repository → retry import succeeds
- Test validation violations displayed same format as validate subcommand

### Edge Cases
- Repository with permission issues (should fail validation)
- Repository with corrupted marker file (should fail validation)
- Repository with partial structure (should fail validation)
- Empty repository with valid structure (should pass validation)
- Repository with only one data type (calls OR SMS)

## Risks and Mitigations
- **Risk**: Performance impact from validation on large repositories
  - **Mitigation**: Validation is designed to be fast; target < 1 second for 10K entries
- **Risk**: Breaking existing import workflows that rely on partial repositories
  - **Mitigation**: Clear error messages guide users to fix repository; validates same as validate command
- **Risk**: Reader creation overhead during validation
  - **Mitigation**: Readers are lightweight; consider reusing for actual import process

## References
- Related features: FEAT-007 (Validate subcommand - provides validation infrastructure)
- Code locations: pkg/importer/importer.go:174 (current validateRepository function)
- Dependencies: pkg/validation (existing validation logic)

## Notes
Additional thoughts, questions, or considerations that arise during planning/implementation.
