# FEAT-006: Enable CLI

## Status
- **Completed**: -
- **Priority**: medium

## Overview
Implement the command-line interface for the mobilecombackup tool using the Cobra CLI framework.

## Background
The primary way to use this project will be through a CLI. This feature establishes the CLI foundation using Cobra, which will support future subcommands. Initially, the CLI will only display usage/help and version information.

## Design
### CLI Framework
Use [Cobra](https://github.com/spf13/cobra) for the CLI implementation to:
- Support subcommands structure for future features (validate, import, etc.)
- Provide consistent help generation
- Follow POSIX conventions (single dash for short options, double dash for long)

## Requirements
### Example Usage
```bash
# Show help (no arguments)
$ mobilecombackup
A tool for processing mobile phone backup files

Usage:
  mobilecombackup [command]

Available Commands:
  help        Help about any command

Flags:
  -h, --help      help for mobilecombackup
  -v, --version   version for mobilecombackup

Use "mobilecombackup [command] --help" for more information about a command.

# Show help explicitly
$ mobilecombackup --help
[same output as above]

# Show help subcommand
$ mobilecombackup help
[same output as above]

# Show version
$ mobilecombackup --version
mobilecombackup version 0.1.0
```

### Functional Requirements
- [ ] Running with no arguments displays help and returns exit code 0
- [ ] Running with `--help` or `-h` displays help and returns exit code 0
- [ ] Running with `--version` or `-v` displays version and returns exit code 0
- [ ] Implement `help` subcommand that displays help and returns exit code 0
- [ ] Running with unknown flags or subcommands displays error message, then help, and returns exit code 1
- [ ] Set up Cobra root command with proper metadata (description, version)

### Non-Functional Requirements
- [ ] Clear error messages that distinguish between unknown flags vs unknown subcommands
- [ ] Help output includes:
  - Tool description
  - Usage syntax
  - Available commands (currently just "help")
  - Available flags
  - Instructions for getting command-specific help
- [ ] Version string format: "mobilecombackup version X.Y.Z"

## Implementation Notes
- Version should be set as a variable that can be overridden at build time
- Root command should be structured to easily add subcommands in future features
- Error messages should be written to stderr, help/version to stdout

## References
- [Cobra Documentation](https://cobra.dev/)
- Future features that will add subcommands: FEAT-007 (validate), FEAT-010 (import)
