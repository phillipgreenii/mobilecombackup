# FEAT-008: Import Calls

## Status
- **Completed**: -
- **Priority**: high

## Overview
Import valid calls from call backups (`calls-*.xml`).  Invalid calls will be rejected, valid calls will be added to the repository if they aren't already in there.

The import process follows this flow:
1. Validate the repository structure
2. Load the existing repository for deduplication
3. Process each file, accumulating valid calls
4. Write/update the repository only once after all files are processed

## Background
Daily backups contain many duplicates from previous days. Duplicates should not occur within the repository.

## Requirements
### Functional Requirements
- [ ] The repository will not contain any duplicate calls
- [ ] Track total number of calls added per year.
- [ ] Track total number of calls not added because they would be duplicates.
- [ ] Repository will persist calls ordered by timestamp and partioned by year.
- [ ] Preserve original entry order for same timestamps
- [ ] Invalid calls will be dumped into a file, with the same relative path, but in a `rejected/` directory.

### Non-Functional Requirements
- [ ] Handle large datasets efficiently
- [ ] Maintain stability for entries with identical timestamps

## Design/Implementation Approach
### Processing Flow
1. **Validate Repository**: Use validation from FEAT-007 to ensure repository is valid
2. **Load Repository**: Load existing calls for deduplication checking using FEAT-002 reader
3. **Process Files**: For each import file:
   - Parse and validate each call entry
   - Calculate hash (excluding `readable_date` and `contact_name`)
   - Check against loaded repository for duplicates
   - Accumulate non-duplicates for later writing
   - Write invalid entries to rejection files
4. **Write Repository**: After all files processed, perform a single repository update:
   - Combine existing entries with new entries
   - Sort by timestamp and partition by year
   - Write to final repository location

### Deduplication Strategy
- Use the same hash-based approach from FEAT-001
- Hash calculation excludes `readable_date` field (timezone-dependent) and `contact_name` (unreliable field that may change over time)
- Build deduplication index from existing repository
- Check all new entries against this index before accepting them

### Implementation Options
- **Option A**: Keep everything in memory (simpler but requires more RAM)
- **Option B**: Use temporary staging area (more complex but handles larger datasets)
- The key requirement is that the repository is only written/modified once

### Validation Criteria
Calls are validated using the same logic as FEAT-002:
- Required fields:
  - `date`: Valid timestamp for year partitioning
  - `number`: Phone number (any valid phone number format accepted)
  - `type`: Call type (incoming/outgoing/missed)
  - `duration`: Call duration in seconds
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
- It will follow the naming pattern of `calls-$originalFileHash-$timestamp-rejects.xml` where:
  - `$originalFileHash` is the SHA-256 of the file being imported
  - `$timestamp` is the import timestamp in format YYYYMMDD-HHMMSS
- It will be structured the same as a valid "Calls Backup", except each row will be the rejected calls.
  - The idea behind this is that it should be easy to import the rejected calls once they have been corrected.
Next to the rejects file, will be a file called `calls-$originalFileHash-$timestamp-violations.yaml` which will contain the violations of each call.  It will look similar to 
```yaml
violations:
  - line: 1  # 1-based line numbers
    violations:
      - missing-timestamp
      - unknown-type
  - line: 2
    violations:
      - missing-timestamp
```

### Performance Considerations
- Initial repository load optimized for deduplication checking
- Progress reporting every 100 entries during file processing
- Streaming XML parsing for import files to handle large inputs
- Single repository write operation at end to ensure consistency
- Summary statistics displayed at completion
- Implementation should handle large repositories efficiently

## Tasks
- [ ] Design accumulator structure for new calls (in-memory or staging)
- [ ] Implement repository loading for deduplication index
- [ ] Extend coalescer to handle call entries
- [ ] Implement call-specific validation rules (reuse FEAT-002 logic)
- [ ] Add call hash calculation (exclude `readable_date` and `contact_name`)
- [ ] Create rejection file writer for invalid calls with timestamp in filename
- [ ] Implement progress reporting for large imports
- [ ] Create single-write repository update mechanism:
  - [ ] Merge existing and new entries
  - [ ] Sort by timestamp
  - [ ] Partition by year
  - [ ] Write to repository atomically
- [ ] Add call import to main command flow (functionality only, CLI in FEAT-010)
- [ ] Write unit tests for accumulator operations
- [ ] Write unit tests for call validation logic
- [ ] Write unit tests for hash calculation with contact_name exclusion
- [ ] Write integration test: Import calls into empty repository
- [ ] Write integration test: Import calls with existing repository (duplicate detection)
- [ ] Write integration test: Import calls with invalid entries (rejection handling)
- [ ] Write integration test: Import large dataset (performance test)
- [ ] Write integration test: Import calls with same timestamp (order preservation)
- [ ] Write integration test: Verify repository is updated only once
- [ ] Update summary output to include call statistics

## References
- Related: FEAT-001-repository-validation
- Related: FEAT-002-calls-from-repository
- Related: FEAT-010-add-import-subcommand
