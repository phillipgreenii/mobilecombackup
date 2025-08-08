# FEAT-006: Enable CLI

## Status
- **Completed**: 2025-08-07
- **Priority**: medium

## Overview
Implement the command-line interface for the mobilecombackup tool using the Cobra CLI framework.

## Background
The primary way to use this project will be through a CLI. This feature establishes the CLI foundation using Cobra, which will support future subcommands. Initially, the CLI will only display usage/help and version information.

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
  -h, --help              help for mobilecombackup
      --quiet             Suppress non-error output
      --repo-root string  Path to repository root (default ".")
  -v, --version           version for mobilecombackup

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

# Unknown command error
$ mobilecombackup unknown
Error: unknown command "unknown" for "mobilecombackup"
Run 'mobilecombackup --help' for usage.

# Unknown flag error
$ mobilecombackup --unknown
Error: unknown flag: --unknown
Run 'mobilecombackup --help' for usage.

# Global flags example (will be used by future subcommands)
$ mobilecombackup --repo-root /path/to/repo --quiet help
[help output with global flags set]
```

### Functional Requirements
- [ ] Running with no arguments displays help and returns exit code 0
- [ ] Running with `--help` or `-h` displays help and returns exit code 0
- [ ] Running with `--version` or `-v` displays version and returns exit code 0
- [ ] Implement `help` subcommand that displays help and returns exit code 0
- [ ] Running with unknown flags or subcommands displays error message, then help, and returns exit code 1
- [ ] Set up Cobra root command with proper metadata (description, version)
- [ ] Implement global `--quiet` flag to suppress non-error output
- [ ] Implement global `--repo-root` flag for repository path (inherited by subcommands)
- [ ] Follow semantic versioning (X.Y.Z format)

### Non-Functional Requirements
- [ ] Clear error messages that distinguish between unknown flags vs unknown subcommands
- [ ] Help output includes:
  - Tool description
  - Usage syntax
  - Available commands (currently just "help")
  - Available flags
  - Instructions for getting command-specific help
- [ ] Version string format: "mobilecombackup version X.Y.Z"
- [ ] Error messages prefixed with "Error: " and written to stderr
- [ ] Successful output written to stdout
- [ ] Include suggestions for common mistakes in error messages

## Design
### CLI Framework
Use [Cobra](https://github.com/spf13/cobra) for the CLI implementation to:
- Support subcommands structure for future features (validate, import, etc.)
- Provide consistent help generation
- Follow POSIX conventions (single dash for short options, double dash for long)

### API/Interface
```go
// cmd/mobilecombackup/main.go
package main

import (
    "os"
    "github.com/phillipgreenii/mobilecombackup/cmd/mobilecombackup/cmd"
)

var (
    // Version is set via ldflags during build
    Version = "dev"
    BuildTime = "unknown"
)

func main() {
    cmd.SetVersion(Version)
    if err := cmd.Execute(); err != nil {
        os.Exit(1)
    }
}

// cmd/mobilecombackup/cmd/root.go
package cmd

import (
    "fmt"
    "os"
    
    "github.com/spf13/cobra"
)

var (
    version string
    quiet bool
    repoRoot string
)

var rootCmd = &cobra.Command{
    Use:   "mobilecombackup",
    Short: "A tool for processing mobile phone backup files",
    Long: `mobilecombackup processes call logs and SMS/MMS messages from 
mobile phone backup files, removing duplicates and organizing by year.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Show help when no subcommand provided
        return cmd.Help()
    },
}

func Execute() error {
    return rootCmd.Execute()
}

func SetVersion(v string) {
    version = v
    rootCmd.Version = v
}

func init() {
    rootCmd.SetVersionTemplate("mobilecombackup version {{.Version}}\n")
    
    // Global flags
    rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Suppress non-error output")
    rootCmd.PersistentFlags().StringVar(&repoRoot, "repo-root", ".", "Path to repository root")
}

// Helper functions for error handling
func PrintError(format string, args ...interface{}) {
    fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}
```

### Project Structure
```
cmd/mobilecombackup/
├── main.go           # Entry point with version variables
└── cmd/
    └── root.go       # Root command definition
```

## Build Configuration
```bash
# Build with version injection
VERSION=$(git describe --tags --always --dirty)
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
go build -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" ./cmd/mobilecombackup

# For release builds
VERSION=0.1.0
go build -ldflags "-X main.Version=$VERSION" ./cmd/mobilecombackup

# Devbox script addition
devbox add --script "build-cli" -- "go build -ldflags \"-X main.Version=\$(git describe --tags --always --dirty)\" ./cmd/mobilecombackup"
```

## Tasks
- [x] Add Cobra dependency to go.mod
- [x] Create cmd/mobilecombackup/cmd/ directory structure
- [x] Implement root.go with root command definition
- [x] Add global flags (--quiet, --repo-root)
- [x] Update main.go to use version injection
- [x] Configure help behavior for no arguments
- [x] Implement version flag handling with semantic versioning
- [x] Set up error handling with "Error: " prefix
- [x] Write helper functions for consistent error output
- [x] Write unit tests for CLI behavior
- [x] Write integration tests for command execution
- [x] Update devbox.json with build script
- [x] Document CLI usage in README

## Testing
### Unit Tests
- Test root command initialization
- Test version flag output format matches "mobilecombackup version X.Y.Z"
- Test help flag behavior
- Test unknown command error handling with proper error prefix
- Test unknown flag error handling with proper error prefix
- Test global flag parsing (--quiet, --repo-root)
- Test error output goes to stderr
- Test successful output goes to stdout

### Integration Tests
- Test binary execution with no arguments (exit code 0, help displayed)
- Test binary execution with --help (exit code 0)
- Test binary execution with --version (exit code 0)
- Test binary execution with invalid commands (exit code 1)
- Test binary execution with invalid flags (exit code 1)
- Test exit codes for all scenarios
- Test global flag inheritance by subcommands (when added)

### Edge Cases
- Multiple flags (--help --version) - version should take precedence
- Abbreviated flags (-hv)
- Case sensitivity of commands
- Special characters in arguments
- Empty --repo-root value
- Invalid --repo-root path
- Combining --quiet with --help or --version

## Risks and Mitigations
- **Risk**: Version conflicts between Cobra and other dependencies
  - **Mitigation**: Pin to specific Cobra version in go.mod (use v1.8.0 or later)
- **Risk**: Breaking changes in future Cobra versions
  - **Mitigation**: Use stable API features, avoid experimental ones
- **Risk**: Inconsistent command structure as more subcommands added
  - **Mitigation**: Establish patterns now for future subcommands (cmd directory structure)
- **Risk**: Version injection failing in different build environments
  - **Mitigation**: Default to "dev" version when not injected
- **Risk**: Global flags conflicting with subcommand flags
  - **Mitigation**: Document reserved flag names for future subcommands

## Future Considerations
Additional global flags that might be needed by multiple subcommands:
- `--verbose`: Enable debug logging (mutually exclusive with --quiet)
- `--no-color`: Disable colored output
- `--config`: Path to configuration file

These could be added to the root command and inherited by all subcommands.

## Implementation Notes
- Version should follow semantic versioning (X.Y.Z)
- Version variable set via ldflags during build for accurate version tracking
- Root command should be structured to easily add subcommands in future features
- Error messages should be written to stderr with "Error: " prefix
- Help and version output to stdout
- Use default Cobra help template
- Project structure uses cmd/ directory pattern for commands

## References
- [Cobra Documentation](https://cobra.dev/)
- Future features that will add subcommands: FEAT-007 (validate), FEAT-010 (import), FEAT-014 (init)
- Semantic Versioning: https://semver.org/
