# FEAT-014: Add Init Subcommand

## Status
- **Completed**: Not yet started
- **Priority**: high (will be worked on next)

## Overview
Add an `init` subcommand to create a new repository structure with the required directories (calls/, sms/, attachments/). This provides users with a clean way to initialize a new backup repository before importing data.

## Background
Currently, users must manually create the repository directory structure or rely on the import process to create directories as needed. An explicit initialization command would:
- Provide a clear starting point for new users
- Ensure consistent repository structure
- Validate environment before repository creation
- Prevent partial initialization failures

## Requirements
### Functional Requirements
- [ ] Create repository directory structure: calls/, sms/, attachments/
- [ ] Use current directory as default repo root if --repo-root not specified
- [ ] Validate target directory is either non-existent or empty
- [ ] Error if directory exists with files (distinguish between existing repo vs other files)
- [ ] Check write permissions before creating any directories
- [ ] Display created structure upon successful completion
- [ ] Support --dry-run mode to preview actions without creating directories

### Non-Functional Requirements
- [ ] Atomic operation - either create all directories or none
- [ ] Clear error messages distinguishing different failure scenarios
- [ ] Follow existing CLI patterns and conventions

## Design
### Approach
The init command will:
1. Parse command-line arguments (--repo-root, --dry-run)
2. Validate the target directory (permissions, emptiness)
3. Check for existing repository structure
4. Create all required directories atomically
5. Display the created structure

### API/Interface
```go
// InitCommand represents the init subcommand
type InitCommand struct {
    RepoRoot string
    DryRun   bool
}

// Execute runs the init command
func (c *InitCommand) Execute() error {
    // Validate and create repository
}

// CLI usage
// mobilecombackup init [--repo-root <path>] [--dry-run]
```

### Data Structures
```go
// RepositoryStructure defines the directories to create
var RepositoryStructure = []string{
    "calls",
    "sms", 
    "attachments",
}

// InitResult contains the result of initialization
type InitResult struct {
    RepoRoot string
    Created  []string // Directories created
    DryRun   bool
}
```

### Implementation Notes
- Use os.MkdirAll with 0755 permissions for directory creation
- Validate write permissions using os.OpenFile with O_CREATE|O_EXCL
- Check for existing repository by looking for characteristic directories
- Implement rollback on failure (remove partially created directories)
- Use filepath.Join for cross-platform path handling

## Tasks
- [ ] Add init subcommand to CLI parser
- [ ] Implement directory validation logic
  - [ ] Check if path exists and is empty
  - [ ] Detect existing repository structure
  - [ ] Validate write permissions
- [ ] Implement directory creation logic
  - [ ] Create directories atomically
  - [ ] Handle rollback on failure
- [ ] Add --dry-run support
- [ ] Implement output formatting (display created structure)
- [ ] Write unit tests
  - [ ] Test empty directory initialization
  - [ ] Test permission validation
  - [ ] Test existing repository detection
  - [ ] Test dry-run mode
- [ ] Write integration tests
  - [ ] End-to-end initialization test
  - [ ] Test with various directory states
- [ ] Update documentation
  - [ ] Add init command to README
  - [ ] Update CLI help text

## Testing
### Unit Tests
- Test directory validation with various states (non-existent, empty, with files)
- Test permission checking logic
- Test existing repository detection
- Test dry-run mode doesn't create directories
- Test error messages for different failure scenarios

### Integration Tests
- Initialize repository in temporary directory
- Verify all directories created with correct permissions
- Test initialization failure scenarios
- Test --repo-root with absolute and relative paths

### Edge Cases
- Directory with no write permissions
- Path that is a file, not a directory
- Symbolic links in path
- Very long path names
- Directory with hidden files only
- Concurrent initialization attempts

## Risks and Mitigations
- **Risk**: Partial directory creation on failure
  - **Mitigation**: Implement atomic creation with rollback
- **Risk**: Permission issues on different platforms
  - **Mitigation**: Use standard Go os package permissions
- **Risk**: Race conditions with concurrent access
  - **Mitigation**: Use file locking or clear documentation about single-user assumption

## References
- Related features: FEAT-006 (enable CLI)
- Code locations: cmd/mobilecombackup/main.go
- Similar patterns: Repository validation in FEAT-001

## Notes
- Consider adding a `.mobilecombackup` marker file to identify initialized repositories
- The attachments/ directory structure (with hash-based subdirectories) will be created on demand during import
