# MobileComBackup - Tool used to coalesce Call and SMS backups

[![Built with Devbox](https://www.jetify.com/img/devbox/shield_galaxy.svg)](https://www.jetify.com/devbox/docs/contributor-quickstart/)

A command-line tool for processing mobile phone backup files (Call and SMS logs in XML format). It coalesces multiple backup files, removes duplicates, extracts attachments, and organizes data by year.

## Installation

### Building from Source

```bash
# Clone the repository
git clone https://github.com/phillipgreen/mobilecombackup.git
cd mobilecombackup

# Build the binary
go build -o mobilecombackup github.com/phillipgreen/mobilecombackup/cmd/mobilecombackup

# Build with version information
VERSION=$(git describe --tags --always --dirty)
go build -ldflags "-X main.Version=$VERSION" -o mobilecombackup github.com/phillipgreen/mobilecombackup/cmd/mobilecombackup

# Or use the devbox script
devbox run build-cli
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
  init        Initialize a new mobilecombackup repository

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
```

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -v -covermode=set ./...

# Run specific package tests
go test -v ./cmd/mobilecombackup/cmd/...
```

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

