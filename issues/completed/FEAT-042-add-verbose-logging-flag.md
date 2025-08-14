# FEAT-042: Add Verbose Logging Flag to Control Output Noise

## Status
- **Completed**: 2025-08-14 (Already implemented - complete logging infrastructure)
- **Priority**: high

## Overview
Add a command-line flag to control logging verbosity. By default, the output should only include results with minimal noise. When the verbose flag is enabled, entry-level logging should be displayed to help with debugging and monitoring the processing flow.

## Background
Currently, the CLI output is noisy with debugging information that makes it hard to see the actual results. Users need a clean output by default, but developers and power users should have the option to enable verbose logging when troubleshooting.

## Requirements
### Functional Requirements
- [ ] Add a global `--verbose` or `-v` flag to all commands
- [ ] By default, only show essential results and error messages
- [ ] When verbose flag is set, show entry-level logging for processing steps
- [ ] Maintain consistent logging levels across all subcommands
- [ ] Preserve current functionality while cleaning up default output

### Non-Functional Requirements
- [ ] No performance impact when verbose logging is disabled
- [ ] Logging should be consistent across all CLI commands
- [ ] Follow Go standard library logging conventions

## Design
### Approach
1. Implement a global verbose flag using Cobra's persistent flags
2. Create a logging utility that respects the verbose setting
3. Update all existing log statements to use appropriate log levels
4. Ensure clean output by default with optional verbose details

### API/Interface
```go
// Add global flag to root command
var verbose bool
rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")

// Logging utility interface
type Logger interface {
    Info(msg string)
    Error(msg string)
    Debug(msg string) // only shown in verbose mode
}
```

### Implementation Notes
- Use a global logger instance that can be configured based on the verbose flag
- Review all current fmt.Print* statements and categorize them as essential output vs debugging info
- Essential output (results, summaries) should always be shown
- Process details, step-by-step logging should only appear in verbose mode

## Tasks
- [ ] Add global verbose flag to root command using Cobra persistent flags
- [ ] Create logging utility that respects verbose setting
- [ ] Audit all existing output statements and categorize them
- [ ] Update import, validate, info, and other commands to use proper logging levels
- [ ] Write tests for verbose and non-verbose output modes
- [ ] Update CLI help documentation

## Testing
### Unit Tests
- Test that verbose flag is properly parsed and accessible to subcommands
- Test logger behavior in both verbose and non-verbose modes
- Test that essential output is always shown regardless of verbose setting

### Integration Tests
- Run each command with and without verbose flag to verify output cleanliness
- Verify that error messages are always shown regardless of verbose setting

### Edge Cases
- Ensure error handling doesn't change between verbose modes
- Test that piped output works correctly in both modes

## Risks and Mitigations
- **Risk**: Breaking existing scripts that depend on current output format
  - **Mitigation**: Carefully preserve essential output, only remove debugging noise
- **Risk**: Performance impact from logging infrastructure
  - **Mitigation**: Use conditional logging to avoid string formatting when not needed

## References
- Related features: All CLI commands (import, validate, info, etc.)
- Code locations: cmd/mobilecombackup/root.go, all subcommand files
- See NEW_ISSUES item #1 for original requirement

## Notes
This addresses the first issue in NEW_ISSUES where output is described as "noisy". The goal is to make the default experience clean while preserving debugging capabilities for when they're needed.