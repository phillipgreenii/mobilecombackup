# BUG-020: Import doesn't update summary.yaml

## Status
- **Reported**: 2025-08-08
- **Fixed**: 
- **Priority**: high
- **Severity**: major

## Overview
The import command does not create or update the `summary.yaml` file in the repository root, which should contain metadata about the repository contents and last import operations.

## Reproduction Steps
1. Initialize a new repository with `mobilecombackup init`
2. Import data with `mobilecombackup import`
3. Check the repository root for `summary.yaml`
4. Observe that the file is either missing or not updated with import information

## Expected Behavior
After import, `summary.yaml` should exist in the repository root containing the current state of the repository (not import history).

Example structure:
```yaml
last_updated: 2025-08-08T10:30:00Z
statistics:
  total_calls: 1250
  total_sms: 3400
  total_attachments: 450
  years_covered: [2014, 2015, 2016]
```

The summary should:
- Be written once at the end of import after all files are successfully processed
- Contain total counts of each entity type across all years
- List all years that have data in the repository
- Be updated with each import to reflect the current repository state

## Actual Behavior
The `summary.yaml` file is not created or updated during import operations.

## Environment
- Version: Current main branch
- OS: All platforms affected
- Go version: 1.24

## Root Cause Analysis
### Investigation Notes
The import command generates an ImportSummary struct containing all statistics but doesn't persist this to disk.

### Root Cause
The import command successfully processes all data and updates the repository but doesn't generate a summary.yaml file. The functionality to count total entities across all years and write this summary was never implemented in the import workflow.

## Fix Approach
1. Define a simple SummaryFile struct containing only repository statistics
2. After successful import, count all entities across year files
3. Generate years_covered by checking which year files exist
4. Write summary.yaml atomically at the end of import
5. Skip summary generation in dry-run mode

## Tasks
- [ ] Define SummaryFile struct in importer package
- [ ] Create summary generation function that counts entities from repository
- [ ] Implement year detection from existing year files
- [ ] Add summary writing at end of Import() method
- [ ] Handle dry-run mode (skip writing)
- [ ] Write unit tests for summary generation
- [ ] Write integration tests verifying summary.yaml creation
- [ ] Update README with summary.yaml documentation

## Testing
### Regression Tests
- Test summary creation on first import
- Test summary updates on subsequent imports
- Test with various data types and volumes
- Verify YAML formatting is valid

### Verification Steps
1. Run import on empty repository
2. Verify summary.yaml is created with correct data
3. Run another import
4. Verify summary.yaml is updated appropriately

## Workaround
Users can manually count entities by examining year files, but this is tedious and error-prone.

## Related Issues
- FEAT-010: Add Import subcommand (where summary should be generated)
- BUG-021: Import not generating files.yaml (similar metadata issue)

## Notes
- Summary reflects current repository state, not import history
- Only total counts are tracked, not duplicates or per-import statistics
- The file should be written atomically to prevent corruption
- Future enhancement: Could add validation timestamp if validate command updates it