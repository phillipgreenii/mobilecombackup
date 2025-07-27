# FEAT-010: Add Import subcommand

## Status
- **Completed**: -
- **Priority**: high

## Overview
The import featured added in FEAT-008 and FEAT-009, will be triggered via the `import` subcommand.

## Background
Users need a simple interface to process backup files and import them in the repository.

## Requirements
### Functional Requirements
- [ ] Parse command-line arguments (repo-root, file paths)
- [ ] Support processing individual files
- [ ] Support scanning directories for backup files
- [ ] Will validate the repo-root before starting the import
- [ ] Display processing summary with statistics (totals and per-year breakdown)
- [ ] Support `--dry-run` flag to preview without importing
- [ ] Support `--verbose` flag for detailed logging
- [ ] Use exit code 1 to represent the import completed, but rejects were found
- [ ] Support `--no-error-on-rejects` to return exit code 0 if the import completed, but rejects were found.
- [ ] Use exit code 2 to represent the import failed.

### Non-Functional Requirements
- [ ] Clear error messages
- [ ] Progress indication for long operations (emit log at start of each file and after fixed record intervals)
- [ ] Command-line argument `--repo-root` takes precedence over `MB_REPO_ROOT` environment variable

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
usage: mobilecombackup [--help] SUBCOMMAND
   Sub Commands:
      import --repo-root $PATH_TO_REPO $backupfile1 ... $backupfilesdir1 ...
```

### Example Output Format

```
Processing files...
  Processing: backup-2024-01-15.xml
  Processing: calls-2024-02-01.xml
  Processing: sms-archive.xml

Import Summary:
              Initial     Final     Delta     Duplicates    Rejected
Calls Total        10        45        35             12           3
  2023              5        15        10              3           1
  2024              5        30        25              9           2
SMS Total          23        78        55             20           5
  2023             13        38        25              8           2
  2024             10        40        30             12           3
```

## References
- Pre-req: `FEAT-001: Repository Validation`
- Pre-req: `FEAT-008: Import Calls`
- Pre-req: `FEAT-009: Import SMS`

