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
git clone https://github.com/phillipgreen/mobilecombackup.git
cd mobilecombackup

# Build with automatic version injection (recommended)
devbox run build-cli

# Or build manually with version information
VERSION=$(bash scripts/build-version.sh)
go build -ldflags "-X main.Version=$VERSION" -o mobilecombackup github.com/phillipgreen/mobilecombackup/cmd/mobilecombackup

# Basic build without version (for development only)
go build -o mobilecombackup github.com/phillipgreen/mobilecombackup/cmd/mobilecombackup
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

