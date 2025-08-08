# FEAT-009: Import SMSs

## Status
- **Completed**: 2025-08-08
- **Priority**: high

## Overview
Import valid sms from SMS backups (`sms-*.xml`).  Invalid sms will be rejected, valid sms will be added to the repository if they aren't already in there.

The import process follows this flow:
1. Validate the repository structure
2. Load the existing repository for deduplication
3. Process each file, accumulating valid messages
4. Write/update the repository only once after all files are processed

MMS attachments remain in their base64 encoded form during import - extraction is handled separately by FEAT-012.

## Background
Daily backups contain many duplicates from previous days. Duplicates should not occur within the repository.

## Requirements
### Functional Requirements
- [ ] The repository will not contain any duplicate sms
- [ ] Track total number of sms added per year.
- [ ] Track total number of sms not added because they would be duplicates.
- [ ] Repository will persist sms ordered by timestamp and partioned by year.
- [ ] Preserve original entry order for same timestamps
- [ ] Invalid sms will be dumped into a file, with the same relative path, but in a `rejected/` directory.
- [ ] Display import summary with counts (added, duplicates, rejected)
- [ ] Process one file at a time
- [ ] Continue processing remaining SMS after encountering invalid entries

### Non-Functional Requirements
- [ ] Handle large datasets efficiently (thousands of SMS messages)
- [ ] Maintain stability for entries with identical timestamps

## Design/Implementation Approach
### Processing Flow
1. **Validate Repository**: Use validation from FEAT-007 to ensure repository is valid
2. **Load Repository**: Load existing SMS/MMS for deduplication checking using FEAT-003 reader
3. **Process Files**: For each import file:
   - Parse and validate each message entry
   - Calculate hash (excluding `readable_date` and `contact_name`)
   - Check against loaded repository for duplicates
   - Accumulate non-duplicates for later writing
   - Write invalid entries to rejection files
4. **Write Repository**: After all files processed, perform a single repository update:
   - Combine existing entries with new entries
   - Sort by timestamp and partition by year
   - Write to final repository location

### Deduplication Strategy
- Use SHA-256 hash-based approach for detecting duplicates
- Hash calculation excludes `readable_date` field (timezone-dependent) and `contact_name` (unreliable field that may change over time)
- Build deduplication index from existing repository
- Check all new entries against this index before accepting them

### Implementation Options
- **Option A**: Keep everything in memory (simpler but requires more RAM)
- **Option B**: Use temporary staging area (more complex but handles larger datasets)
- The key requirement is that the repository is only written/modified once

### Validation Criteria
SMS are validated using criteria from FEAT-003:
- Required fields:
  - `date`: Valid timestamp for year partitioning
  - `address`: Phone number or contact identifier
  - `type`: Message type (sent/received/draft)
  - `body`: Message content (can be empty for MMS)
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
- It will follow the naming pattern of `sms-$originalFileHash-$timestamp-rejects.xml` where:
  - `$originalFileHash` is the SHA-256 of the file being imported
  - `$timestamp` is the import timestamp in format YYYYMMDD-HHMMSS
- It will be structured the same as a valid "SMS Backup", except each row will be the rejected SMS.
  - The idea behind this is that it should be easy to import the rejected SMS once they have been corrected.
Next to the rejects file, will be a file called `sms-$originalFileHash-$timestamp-violations.yaml` which will contain the violations of each SMS.  It will look similar to 
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

## Required Refactoring

### Core Issues Identified
The existing SMS implementation has fundamental type inconsistencies that prevent it from working with the import architecture:

1. **Type Mismatches**: 
   - XML parsing converts timestamps to `time.Time` but structs expect `int64`
   - Boolean fields ("0"/"1" in XML) are parsed as `bool` but structs define them as `int`
   - These mismatches cause compilation errors

2. **Missing MMS Fields**:
   - Test data shows MMS messages have 59+ attributes
   - Current implementation only handles ~30 attributes
   - Missing fields like `sim_imsi`, `sub_id`, `creator`, `spam_report`, etc.

3. **Interface Incompatibility**:
   - Methods return wrong types for coalescer integration
   - Hash calculation references fields with incorrect types

### Refactoring Plan

#### Phase 1: Type System Alignment
Align SMS types with the pattern established in calls implementation:
- Store all timestamps as `int64` (epoch milliseconds)
- Store all boolean XML attributes as `int` (0 or 1)
- Only convert to `time.Time` when needed for display/operations

#### Phase 2: Fix XML Parsing
- Remove `time.Unix()` conversions during parsing
- Parse "0"/"1" strings to `int` not `bool`
- Add parsing for all MMS attributes found in test data

#### Phase 3: Update Methods
- Fix `GetDate()` to return `int64` directly
- Update hash calculation to use correct field types
- Fix writer to handle `int64` timestamps

#### Phase 4: Add Missing Functionality
- Implement `ValidateSMSFile()` method
- Add comprehensive validation rules
- Ensure feature parity with calls implementation

### Implementation Order
1. Update `types.go` - Fix all struct definitions (breaking change)
2. Update `xml_reader.go` - Remove type conversions
3. Fix interface methods and coalescer
4. Update writer and tests
5. Add validation methods
6. Integration testing with real data

## Tasks

### Refactoring Tasks (Prerequisites)
- [x] Fix type definitions in `types.go` to use `int64` for timestamps and `int` for booleans
- [x] Update XML parser to remove type conversions
- [x] Add missing MMS attributes (~30 fields)
- [x] Fix interface methods to return correct types
- [x] Update existing tests for new types

### Implementation Tasks
- [x] Design accumulator structure for new SMS/MMS (in-memory or staging)
- [x] Implement repository loading for deduplication index
- [x] Extend coalescer to handle SMS entries
- [x] Implement SMS-specific validation rules (reuse FEAT-003 logic)
- [x] Add SMS hash calculation (exclude `readable_date` and `contact_name`)
- [ ] Create rejection file writer for invalid SMS with timestamp in filename (TODO left for future)
- [x] Implement progress reporting for large imports
- [x] Create single-write repository update mechanism:
  - [x] Merge existing and new entries
  - [x] Sort by timestamp
  - [x] Partition by year
  - [x] Write to repository atomically
- [x] Add SMS import to main command flow (functionality only, CLI in FEAT-010)
- [x] Write unit tests for accumulator operations
- [x] Write unit tests for SMS validation logic
- [x] Write unit tests for hash calculation with contact_name exclusion
- [x] Write integration test: Import SMS into empty repository
- [ ] Write integration test: Import SMS with existing repository (duplicate detection)
- [ ] Write integration test: Import mixed SMS/MMS messages
- [ ] Write integration test: Import SMS with invalid entries (rejection handling)
- [ ] Write integration test: Import large dataset (performance test with 1000+ messages)
- [ ] Write integration test: Import SMS with same timestamp (order preservation)
- [ ] Write integration test: Verify repository is updated only once
- [ ] Update summary output to include SMS statistics

## References
- Related: FEAT-001-repository-validation
- Related: FEAT-003-sms-from-repository
- Related: FEAT-010-add-import-subcommand
- Related: FEAT-012-extract-attachments
