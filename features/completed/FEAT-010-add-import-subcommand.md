# FEAT-010: Add Import subcommand

## Status
- **Completed**: 2025-08-08
- **Priority**: high

## Overview
The import featured added in FEAT-008 and FEAT-009, will be triggered via the `import` subcommand.

## Background
Users need a simple interface to process backup files and import them in the repository.

## Requirements
### Functional Requirements
- [x] Parse command-line arguments (repo-root, file paths)
- [x] Support processing individual files
- [x] Support scanning directories for backup files (case-sensitive: calls*.xml, sms*.xml)
- [x] Skip files already in repository structure when scanning
- [x] Will validate the repo-root before starting the import
- [x] Display processing summary with statistics (totals and per-year breakdown)
- [x] Support `--dry-run` flag to preview without importing (processes files, deduplicates, generates statistics)
- [x] Support `--verbose` flag for detailed logging
- [x] Support `--quiet` flag to suppress progress, show only summary
- [x] Support `--json` flag for machine-readable output
- [x] Support `--filter` flag to process only calls or only SMS (values: calls, sms)
- [x] Use exit code 1 to represent the import completed, but rejects were found
- [x] Support `--no-error-on-rejects` to return exit code 0 if the import completed, but rejects were found
- [x] Use exit code 2 to represent the import failed

### Non-Functional Requirements
- [x] Clear error messages
- [x] Progress indication for long operations (emit log at start of each file and every 100 records)
- [x] Command-line argument `--repo-root` takes precedence over `MB_REPO_ROOT` environment variable
- [x] Directory scanning follows symlinks, skips hidden directories, no depth limit

### Usage Examples

#### `repo-root` specifies the repository, other arguments are sources of backup files

```
mobilecombackup import --repo-root /path/to/repo file1.xml dir/
```
- repository: `/path/to/repo`
- files to import:
  - `file1.xml`
  - any `calls*.xml` or `sms*.xml` found in `dir/` or any subdirectory, recursively searched

#### when repository is specified and no arguments, look for backup files in current directory

```
mobilecombackup import --repo-root /path/to/repo
```
- repository: `/path/to/repo`
- files to import:
  - any `calls*.xml` or `sms*.xml` found in `.` or any subdirectory, recursively searched

#### when repository is specified via environment and no arguments, look for backup files in current directory

```
MB_REPO_ROOT=/path/to/repo
mobilecombackup import file1.xml dir/
```
- repository: `/path/to/repo`
- files to import:
  - `file1.xml`
  - any `calls*.xml` or `sms*.xml` found in `dir/` or any subdirectory, recursively searched

#### when no repository is specified, use current directory as the repository, other arguments are sources of backup files

```
mobilecombackup import file1.xml dir/
```
- repository: `.`
- files to import:
  - `file1.xml`
  - any `calls*.xml` or `sms*.xml` found in `dir/` or any subdirectory, recursively searched (case-sensitive)

#### when repository is specified via environment and no arguments, look for backup files in current directory

```
MB_REPO_ROOT=/path/to/repo
mobilecombackup import 
```
- repository: `/path/to/repo`
- files to import:
  - any `calls*.xml` or `sms*.xml` found in `.` or any subdirectory, recursively searched

#### when repository is not specified and no arguments, error and display usage

```
mobilecombackup import 
```

```
Usage: mobilecombackup import [flags] [paths...]

Import mobile backup files into the repository

Arguments:
  paths    Files or directories to import (default: current directory)

Flags:
      --repo-root string        Repository root directory (env: MB_REPO_ROOT)
      --dry-run                 Preview import without making changes
      --verbose                 Enable verbose output
      --quiet                   Suppress progress output, show only summary
      --json                    Output summary in JSON format
      --filter string           Process only specific type: calls, sms
      --no-error-on-rejects     Don't exit with error code if rejects found
  -h, --help                    help for import

Examples:
  # Import specific files
  mobilecombackup import --repo-root /path/to/repo backup1.xml backup2.xml
  
  # Scan directory for backup files
  mobilecombackup import --repo-root /path/to/repo /path/to/backups/
  
  # Preview import without changes
  mobilecombackup import --repo-root /path/to/repo --dry-run backup.xml
  
  # Import only call logs
  mobilecombackup import --repo-root /path/to/repo --filter calls backups/

Exit codes:
  0 - Success
  1 - Import completed with rejected entries
  2 - Import failed
```

### Example Output Format

```
Processing files...
  Processing: backup-2024-01-15.xml (100 records)... (200 records)... done
  Processing: calls-2024-02-01.xml (100 records)... done
  Processing: sms-archive.xml (100 records)... (200 records)... (300 records)... done

Import Summary:
              Initial     Final     Delta     Duplicates    Rejected
Calls Total        10        45        35             12           3
  2023              5        15        10              3           1
  2024              5        30        25              9           2
SMS Total          23        78        55             20           5
  2023             13        38        25              8           2
  2024             10        40        30             12           3

Files processed: 3
Rejected files created:
  - rejected/calls-a1b2c3d4-20240115-143022-rejects.xml (3 entries)
  - rejected/sms-e5f6g7h8-20240115-143025-rejects.xml (5 entries)

Time taken: 2.3s
```

## Design/Implementation Approach

### Command Structure
- Integrate with Cobra framework established in FEAT-006
- Create `cmd/import.go` for the import subcommand
- Reuse repository validation from FEAT-007

### Processing Flow
1. Parse and validate command-line arguments
2. Determine repository location (flag > env > current directory)
3. Validate repository structure using FEAT-007 validation
4. If validation fails, exit with code 2
5. Load repository for deduplication:
   - Load existing calls using FEAT-002 reader
   - Load existing SMS/MMS using FEAT-003 reader
   - Build deduplication indices
6. Scan specified paths for backup files:
   - Follow symlinks
   - Skip hidden directories
   - Match patterns: `calls*.xml`, `sms*.xml` (case-sensitive)
   - Skip files already in repository structure
7. For each file found:
   - Log start of processing (unless --quiet)
   - Determine type (calls/SMS) from filename
   - Process using FEAT-008 (calls) or FEAT-009 (SMS) logic
   - Accumulate valid entries (not written to repository yet)
   - Report progress every 100 records
   - Continue to next file on errors
8. Write/update repository (single operation):
   - Merge existing and new entries
   - Sort and partition by year
   - Write all calls and SMS/MMS to repository
   - Only write if not --dry-run
9. Generate and display summary
10. Exit with appropriate code

### Output Formats
- **Default**: Human-readable progress and summary table
- **Quiet**: Summary table only
- **JSON**: Machine-readable JSON output with all statistics
- **Verbose**: Detailed logging including duplicate details

### Error Handling
- Repository validation failure: Exit immediately with code 2
- Repository load failure: Exit immediately with code 2
- File read errors: Log error, continue with next file
- Processing errors: Create rejection files, continue processing
- Repository write failure: Exit with code 2 after displaying what was processed
- Track all errors for final summary

## Tasks
- [x] Create import command file (`cmd/mobilecombackup/cmd/import.go`)
- [x] Define command flags and help text using Cobra
- [x] Implement repository location resolution (flag > env > cwd)
- [x] Integrate repository validation from FEAT-007
- [x] Implement repository loading for deduplication:
  - [x] Load calls for building deduplication index
  - [x] Load SMS/MMS for building deduplication index
- [x] Implement file scanning logic:
  - [x] Recursive directory walking with symlink support
  - [x] Hidden directory filtering
  - [x] Pattern matching for backup files
  - [x] Repository file exclusion
- [x] Create progress reporter interface for 100-record intervals
- [x] Integrate FEAT-008 call import functionality
- [x] Integrate FEAT-009 SMS import functionality
- [x] Implement single repository update:
  - [x] Coordinate final write of all accumulated data
  - [x] Ensure atomic update (no partial writes)
  - [x] Skip writing if --dry-run
- [x] Implement summary statistics collection and formatting
- [x] Add JSON output formatter
- [x] Implement dry-run mode (process without writing)
- [x] Add filter flag logic (calls-only, sms-only)
- [x] Calculate and display processing time
- [x] Write unit tests for command parsing
- [x] Write unit tests for file scanning logic
- [x] Write integration test: Import with various flag combinations
- [x] Write integration test: Import with missing repository
- [x] Write integration test: Import with file errors
- [x] Write integration test: Dry-run verification
- [x] Update main.go to register import subcommand

## References
- Pre-req: `FEAT-001: Repository Validation`
- Pre-req: `FEAT-008: Import Calls`
- Pre-req: `FEAT-009: Import SMS`
- Related: `FEAT-006: Init subcommand`
- Related: `FEAT-007: Validate subcommand`

