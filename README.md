# MobileComBackup - Tool used to coalesce Call and SMS backups

[![Built with Devbox](https://www.jetify.com/img/devbox/shield_galaxy.svg)](https://www.jetify.com/devbox/docs/contributor-quickstart/)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=phillipgreenii_mobilecombackup&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=phillipgreenii_mobilecombackup)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=phillipgreenii_mobilecombackup&metric=coverage)](https://sonarcloud.io/summary/new_code?id=phillipgreenii_mobilecombackup)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=phillipgreenii_mobilecombackup&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=phillipgreenii_mobilecombackup)

A command-line tool for processing mobile phone backup files (Call and SMS logs in XML format). It coalesces multiple backup files, removes duplicates, extracts attachments, and organizes data by year.

## Installation

### Building from Source

```bash
# Clone the repository
git clone https://github.com/phillipgreenii/mobilecombackup.git
cd mobilecombackup

# Build with automatic version injection (recommended)
devbox run build-cli

# Or build manually with version information
VERSION=$(bash scripts/build-version.sh)
go build -ldflags "-X main.Version=$VERSION" -o mobilecombackup github.com/phillipgreenii/mobilecombackup/cmd/mobilecombackup

# Basic build without version (for development only)
go build -o mobilecombackup github.com/phillipgreenii/mobilecombackup/cmd/mobilecombackup
```

## Usage

### Basic CLI Usage

```bash
# Show help (no arguments)
$ mobilecombackup
A tool for processing mobile phone backup files

Usage:
  mobilecombackup [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  import      Import mobile backup files into the repository
  info        Show repository information and statistics
  init        Initialize a new mobilecombackup repository
  validate    Validate a mobilecombackup repository

Flags:
  -h, --help              help for mobilecombackup
      --quiet             Suppress non-error output
      --repo-root string  Path to repository root (default ".")
  -v, --version           version for mobilecombackup

Use "mobilecombackup [command] --help" for more information about a command.

# Show version
$ mobilecombackup --version
mobilecombackup version 0.1.0

# Global flags example
$ mobilecombackup --repo-root /path/to/repo --quiet [command]
```

### Init Command

Initialize a new mobilecombackup repository with the required directory structure.

```bash
# Initialize in current directory
$ mobilecombackup init

# Initialize in specific directory
$ mobilecombackup init --repo-root /path/to/new/repo

# Preview without creating (dry run)
$ mobilecombackup init --dry-run

# Initialize quietly (suppress output)
$ mobilecombackup init --quiet
```

The init command creates:
- `calls/` - Directory for call log XML files
- `sms/` - Directory for SMS/MMS XML files
- `attachments/` - Directory for extracted attachment files
- `.mobilecombackup.yaml` - Repository marker file with version metadata
- `contacts.yaml` - Empty contacts file for future use
- `summary.yaml` - Initial summary with zero counts

Example output:
```
Initialized mobilecombackup repository in: /path/to/repo

Created structure:
repo
├── calls
├── sms
├── attachments
├── .mobilecombackup.yaml
├── contacts.yaml
└── summary.yaml
```

### Validate Command

Validate a mobilecombackup repository for structure, content, and consistency.

```bash
# Validate current directory
$ mobilecombackup validate

# Validate specific repository
$ mobilecombackup validate --repo-root /path/to/repo

# Use environment variable for repository
$ MB_REPO_ROOT=/path/to/repo mobilecombackup validate

# Quiet mode - only show violations
$ mobilecombackup validate --quiet

# Verbose mode - show detailed progress
$ mobilecombackup validate --verbose

# JSON output for scripting/CI
$ mobilecombackup validate --output-json
```

The validate command performs comprehensive validation including:
- Repository structure verification
- Manifest file validation
- Checksum verification
- Content format validation
- Cross-reference consistency checks

Exit codes:
- `0`: Repository is valid (no violations found)
- `1`: Validation violations detected
- `2`: Runtime error (e.g., invalid repository path)

Example output:
```
Validating repository... done

Validation Report for: /path/to/repo
------------------------------------------------------------
✓ Repository is valid
```

Or with violations:
```
Validating repository... done

Validation Report for: /path/to/repo
------------------------------------------------------------
✗ Found 2 violation(s)

missing_file (1):
  files.yaml: Failed to load manifest

invalid_format (1):
  calls/calls-2015.xml: Call entry missing required 'date' field
```

JSON output format:
```json
{
  "valid": false,
  "violations": [
    {
      "Type": "missing_file",
      "Severity": "error",
      "File": "files.yaml",
      "Message": "Failed to load manifest"
    }
  ]
}
```

### Info Command

Show comprehensive information about a mobilecombackup repository including statistics, metadata, and validation status.

```bash
# Show information for current directory
$ mobilecombackup info

# Show information for specific repository
$ mobilecombackup info --repo-root /path/to/repo

# Use environment variable for repository
$ MB_REPO_ROOT=/path/to/repo mobilecombackup info

# JSON output for scripting
$ mobilecombackup info --json

# Quiet mode (no output for valid empty repository)
$ mobilecombackup info --quiet
```

The info command displays:
- Repository metadata (version, creation date)
- Call statistics by year with date ranges
- SMS/MMS statistics by year with type breakdown
- Attachment statistics with type distribution and orphan detection
- Contact count
- Rejection and error counts
- Basic validation status

Example output:
```
Repository: /path/to/repo
Version: 1
Created: 2024-01-15T10:30:00Z

Calls:
  2023: 1,234 calls (Jan 5 - Dec 28)
  2024: 567 calls (Jan 2 - Jun 15)
  Total: 1,801 calls

Messages:
  2023: 5,432 messages (4,321 SMS, 1,111 MMS) (Jan 1 - Dec 31)
  2024: 2,345 messages (2,000 SMS, 345 MMS) (Jan 1 - Jun 20)
  Total: 7,777 messages

Attachments:
  Count: 1,456
  Total Size: 245.3 MB
  Types:
    image/jpeg: 1,200
    image/png: 200
    video/mp4: 56
  Orphaned: 12

Contacts: 123

Validation: OK
```

JSON output format:
```json
{
  "version": "1",
  "created_at": "2024-01-15T10:30:00Z",
  "calls": {
    "2023": {
      "count": 1234,
      "earliest": "2023-01-05T10:00:00Z",
      "latest": "2023-12-28T15:30:00Z"
    }
  },
  "sms": {
    "2023": {
      "total_count": 5432,
      "sms_count": 4321,
      "mms_count": 1111,
      "earliest": "2023-01-01T00:00:00Z",
      "latest": "2023-12-31T23:59:00Z"
    }
  },
  "attachments": {
    "count": 1456,
    "total_size": 245300000,
    "orphaned_count": 12,
    "by_type": {
      "image/jpeg": 1200,
      "image/png": 200,
      "video/mp4": 56
    }
  },
  "contacts": 123,
  "validation_ok": true
}
```

### Import Command

Import mobile backup files into the repository with deduplication, validation, and attachment extraction.

```bash
# Import specific files
$ mobilecombackup import backup1.xml backup2.xml

# Scan directory for backup files  
$ mobilecombackup import /path/to/backups/

# Import to specific repository
$ mobilecombackup import --repo-root /path/to/repo backups/

# Use environment variable for repository
$ MB_REPO_ROOT=/path/to/repo mobilecombackup import backups/

# Preview import without changes
$ mobilecombackup import --dry-run backup.xml

# Import only call logs
$ mobilecombackup import --filter calls backups/

# Import only SMS/MMS
$ mobilecombackup import --filter sms backups/

# Verbose output with progress details
$ mobilecombackup import --verbose backups/

# JSON output for scripting
$ mobilecombackup import --json backups/

# Continue even if rejected entries found
$ mobilecombackup import --no-error-on-rejects backups/

# Quiet mode (suppress progress output)
$ mobilecombackup import --quiet backups/
```

The import command performs:
1. **Repository Validation**: Comprehensive validation before any import operations
2. **File Discovery**: Scans for `calls*.xml` and `sms*.xml` files in specified paths
3. **Deduplication**: Detects and skips duplicate entries using SHA-256 hashes
4. **Attachment Extraction**: Extracts images, videos, and documents from MMS messages
5. **Contact Processing**: Extracts contact names with structured multi-address support
6. **Year-Based Organization**: Partitions data by year with accurate statistics tracking
7. **Atomic Write**: Single repository update operation with rollback on failure

**File Discovery Rules:**
- Follows symbolic links
- Skips hidden directories (starting with `.`)
- Excludes files already in repository structure
- Processes files matching patterns: `calls*.xml`, `sms*.xml`

**Import Process:**
```
Processing files...
  Processing: backup-2024-01-15.xml (100 records)... (200 records)... done
  Processing: calls-2024-02-01.xml (100 records)... done

Import Summary:
              Initial     Final     Delta     Duplicates    Rejected
Calls Total        10        45        35             12           3
  2023              5        15        10              3           1
  2024              5        30        25              9           2
SMS Total          23        78        55             20           5
  2023             13        38        25              8           2
  2024             10        40        30             12           3

Files processed: 2
Time taken: 2.3s
```

**JSON Output:**
```json
{
  "files_processed": 2,
  "duration_seconds": 2.3,
  "total": {
    "initial": 33,
    "final": 123,
    "added": 90,
    "duplicates": 32,
    "rejected": 8,
    "errors": 0
  },
  "years": {
    "2023": {"final": 53, "added": 35, "duplicates": 11, "rejected": 3},
    "2024": {"final": 70, "added": 55, "duplicates": 21, "rejected": 5}
  },
  "rejection_files": ["rejected/calls/calls-abc12345-20240115.xml"]
}
```

**Exit Codes:**
- `0`: Import completed successfully
- `1`: Import completed with rejected entries (unless `--no-error-on-rejects`)
- `2`: Import failed (validation error, repository error, I/O error)

**Features:**
- **Repository Validation**: Fast-fail validation ensures repository integrity before import
- **Progress Reporting**: Real-time progress every 100 records for large files
- **Attachment Extraction**: Automatic extraction and deduplication of MMS attachments
- **Contact Processing**: Structured extraction of contact names with multi-address support
- **Year-Based Statistics**: Accurate tracking of initial, added, and duplicate counts per year
- **Error Resilience**: Continue processing on individual entry failures, collect errors for reporting
- **Dry-Run Support**: Preview import operations without modifying repository
- **Filter Support**: Process only specific data types (calls or SMS)

### Exit Codes

- `0`: Success
- `1`: Error (invalid command, flag, or execution error)
- `2`: Runtime error (future subcommands may use this)

## Development

### Development Sandbox

```bash
# start new shell with required dependencies
$ devbox shell

# The environment includes: go 1.24, golangci-lint, claude-code
```

### Helpful Commands

```bash
# build all packages
devbox run builder

# run tests
devbox run tests

# lint code
devbox run linter

# build CLI with version information
devbox run build-cli

# run complete CI pipeline (formatting, tests, linting, build)
devbox run ci
```

### Continuous Integration

The project uses a comprehensive CI pipeline that runs formatting, testing, linting, and building:

```bash
# Run the complete CI pipeline locally
devbox run ci

# This executes:
# 1. devbox run formatter (go fmt ./...)
# 2. devbox run tests (go test -v -covermode=set ./...)  
# 3. devbox run linter (golangci-lint run)
# 4. devbox run build-cli (versioned binary build)
```

The same CI pipeline runs automatically on:
- Pull requests to main branch
- Pushes to main branch  
- Manual workflow dispatch
- Release builds (tags)

All CI workflows use devbox to ensure consistency between local development and CI environments, with Go 1.24 and the same tool versions.

### Code Quality Analysis

The project integrates with SonarQube Cloud for automated code quality analysis:

- **Quality Gate**: Ensures code meets maintainability and reliability standards
- **Coverage Tracking**: Monitors test coverage trends and identifies untested code  
- **Security Analysis**: Scans for potential security vulnerabilities
- **Code Smells Detection**: Identifies maintainability issues and technical debt
- **Duplication Analysis**: Tracks code duplication across the codebase

Quality metrics are automatically updated on every push and pull request. View the [SonarCloud dashboard](https://sonarcloud.io/project/overview?id=phillipgreenii_mobilecombackup) for detailed analysis reports.

**Setup Complete:**
- ✅ SonarCloud project configured with organization key `phillipgreenii`
- ✅ GitHub repository secret `SONAR_TOKEN` configured for authentication
- ✅ Automatic Analysis disabled in SonarCloud project settings (to avoid conflicts with CI analysis)

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -v -covermode=set ./...

# Run specific package tests
go test -v ./cmd/mobilecombackup/cmd/...
```

## Versioning

This project uses a git tag-based versioning system with fallback to a VERSION file. Version information is automatically injected during builds.

### Version Formats

- **Development builds**: `2.0.0-dev-g1234567` (base version + git hash)
- **Release builds**: `2.0.0` (clean semantic version from git tags)

### Check Version

```bash
# Check the version of built binary
$ mobilecombackup --version
mobilecombackup version 2.0.0-dev-g1234567

# Validate version file format
$ devbox run validate-version
```

### Version Sources (Priority Order)

1. **Git tags**: For release builds (e.g., `v2.0.0` → `2.0.0`)
2. **VERSION file + git hash**: For development builds
3. **VERSION file only**: When git is unavailable
4. **Fallback**: `dev` when no version source available

The version extraction automatically handles all scenarios, ensuring builds always have meaningful version information.

## Architecture

The CLI is built using the [Cobra](https://github.com/spf13/cobra) framework, providing:
- POSIX-compliant command-line interface
- Automatic help generation
- Global and command-specific flags
- Subcommand support (for future features)

### Package Structure

- `cmd/mobilecombackup/`: Main CLI entry point
  - `main.go`: Binary entry point with version injection support
  - `cmd/`: Command definitions
    - `root.go`: Root command and global flags

