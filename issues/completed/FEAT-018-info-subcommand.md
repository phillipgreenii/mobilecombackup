# FEAT-018: Add info subcommand

## Status
- **Completed**: 2025-08-09
- **Priority**: medium

## Overview
Add an `info` subcommand that loads a specified repository and prints a comprehensive summary of its contents, including counts of calls, SMS messages, attachments, and contacts.

## Background
Users need a quick way to understand what's in their repository without having to manually check each directory or run validation. The info command would provide a high-level overview of the repository contents.

## Requirements
### Functional Requirements
- [ ] Accept repository path as argument (or use MB_REPO_ROOT environment variable)
- [ ] Display counts of calls by year with date ranges
- [ ] Display counts of SMS/MMS messages by year (separate counts) with date ranges
- [ ] Display total attachment count and size
- [ ] Display attachment distribution by file type (images, videos, etc.)
- [ ] Display orphaned attachments count (not referenced by any message)
- [ ] Display contact count
- [ ] Show repository metadata (version, creation date from .mobilecombackup.yaml)
- [ ] Support JSON output format with --json flag
- [ ] Include rejection/error information if present
- [ ] Include basic validation status

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
    Calls        map[string]YearInfo   `json:"calls"`        // year -> info
    SMS          map[string]MessageInfo `json:"sms"`          // year -> info  
    Attachments  AttachmentInfo        `json:"attachments"`
    Contacts     int                   `json:"contacts"`
    Rejections   map[string]int        `json:"rejections,omitempty"`    // component -> count
    Errors       map[string]int        `json:"errors,omitempty"`        // component -> count
    ValidationOK bool                  `json:"validation_ok"`
}

type YearInfo struct {
    Count     int        `json:"count"`
    Earliest  time.Time  `json:"earliest,omitempty"`
    Latest    time.Time  `json:"latest,omitempty"`
}

type MessageInfo struct {
    TotalCount int        `json:"total_count"`
    SMSCount   int        `json:"sms_count"`
    MMSCount   int        `json:"mms_count"`
    Earliest   time.Time  `json:"earliest,omitempty"`
    Latest     time.Time  `json:"latest,omitempty"`
}

type AttachmentInfo struct {
    Count         int              `json:"count"`
    TotalSize     int64            `json:"total_size"`
    OrphanedCount int              `json:"orphaned_count"`
    ByType        map[string]int   `json:"by_type"` // mime type -> count
}
```

### Implementation Notes
- Leverage existing readers from calls, sms, attachments, and contacts packages
- Use streaming APIs to avoid loading everything into memory
- Display human-readable sizes (KB, MB, GB) for attachment sizes
- Read repository metadata from .mobilecombackup.yaml marker file
- Determine MIME types from attachment file extensions or content inspection
- Track referenced attachments while processing SMS to identify orphans
- Run basic validation checks to determine validation status

### Output Format Example
```
Repository: /home/user/backup
Version: 1.0.0
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

## Tasks
- [x] Create info.go in cmd/mobilecombackup/cmd/
- [x] Implement repository metadata reading from .mobilecombackup.yaml
- [x] Implement calls statistics gathering with date ranges
- [x] Implement SMS/MMS statistics gathering with separate counts and date ranges
- [x] Implement attachment statistics with type distribution
- [x] Implement orphaned attachment detection
- [x] Implement rejection/error counting
- [x] Implement basic validation status check
- [x] Add human-readable output formatting
- [x] Add JSON output support with --json flag
- [x] Write unit tests
- [x] Write integration tests
- [x] Update README with usage examples

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