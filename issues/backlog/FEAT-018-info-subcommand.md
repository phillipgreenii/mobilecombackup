# FEAT-018: Add info subcommand

## Status
- **Completed**: 
- **Priority**: medium

## Overview
Add an `info` subcommand that loads a specified repository and prints a comprehensive summary of its contents, including counts of calls, SMS messages, attachments, and contacts.

## Background
Users need a quick way to understand what's in their repository without having to manually check each directory or run validation. The info command would provide a high-level overview of the repository contents.

## Requirements
### Functional Requirements
- [ ] Accept repository path as argument (or use MB_REPO_ROOT environment variable)
- [ ] Display counts of calls by year
- [ ] Display counts of SMS/MMS messages by year
- [ ] Display total attachment count and size
- [ ] Display contact count
- [ ] Show repository metadata (version, creation date if available)
- [ ] Support JSON output format with --json flag

### Non-Functional Requirements
- [ ] Fast execution - should complete in seconds even for large repositories
- [ ] Clear, readable output format
- [ ] Graceful handling of missing components (e.g., no contacts file)

## Design
### Approach
Reuse existing repository readers to gather statistics and present them in a user-friendly format.

### API/Interface
```go
// Command usage
mobilecombackup info [--repo-root <path>] [--json]
```

### Data Structures
```go
type RepositoryInfo struct {
    Version      string                 `json:"version"`
    CreatedAt    time.Time             `json:"created_at,omitempty"`
    Calls        map[string]int        `json:"calls"`        // year -> count
    SMS          map[string]int        `json:"sms"`          // year -> count  
    Attachments  AttachmentInfo        `json:"attachments"`
    Contacts     int                   `json:"contacts"`
}

type AttachmentInfo struct {
    Count      int    `json:"count"`
    TotalSize  int64  `json:"total_size"`
}
```

### Implementation Notes
- Leverage existing readers from calls, sms, attachments, and contacts packages
- Use streaming APIs to avoid loading everything into memory
- Display human-readable sizes (KB, MB, GB) for attachment sizes

## Tasks
- [ ] Create info.go in cmd/mobilecombackup/cmd/
- [ ] Implement repository statistics gathering
- [ ] Add human-readable output formatting
- [ ] Add JSON output support
- [ ] Write unit tests
- [ ] Write integration tests
- [ ] Update README with usage examples

## Testing
### Unit Tests
- Test statistics gathering with mock readers
- Test output formatting (both text and JSON)
- Test with partial repositories (missing components)

### Integration Tests
- Test with real repository structure
- Verify correct counts against known test data
- Test with empty repository

### Edge Cases
- Repository with no data
- Repository with only some components
- Invalid repository path
- Large repositories (performance test)

## Risks and Mitigations
- **Risk**: Performance issues with very large repositories
  - **Mitigation**: Use streaming APIs and show progress if operation takes >2 seconds

## References
- Related features: FEAT-007 (validate subcommand has similar repository reading logic)
- Code locations: cmd/mobilecombackup/cmd/validate.go (reference implementation)
- Uses readers from: pkg/calls, pkg/sms, pkg/attachments, pkg/contacts

## Notes
This command provides read-only information and makes no modifications to the repository.