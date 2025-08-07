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
- [ ] Create .mobilecombackup.yaml marker file with key-value pairs
  - [ ] Include "repository_structure_version" with value "1"
  - [ ] Include "created_at" with ISO timestamp
  - [ ] Include "created_by" with CLI version
  - [ ] Use YAML format for easy parsing and future extensibility
- [ ] Create empty contacts.yaml file
- [ ] Create summary.yaml file with zero counts for calls and sms
- [ ] Use current directory as default repo root if --repo-root not specified
- [ ] Validate target directory is either non-existent or empty
- [ ] Error if directory exists with files:
  - [ ] Provide specific error if it looks like an existing repository (has .mobilecombackup.yaml or expected directories)
  - [ ] Provide different error for non-empty directory with other files
- [ ] Check write permissions before creating any directories
- [ ] Display created structure in tree-style format upon successful completion
- [ ] Support --dry-run mode to preview actions without creating directories
- [ ] Support --quiet flag to suppress output

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
    Quiet    bool
}

// Execute runs the init command
func (c *InitCommand) Execute() error {
    // Validate and create repository
}

// CLI usage
// mobilecombackup init [--repo-root <path>] [--dry-run] [--quiet]
```

### Data Structures
```go
// RepositoryStructure defines the directories to create
var RepositoryStructure = []string{
    "calls",
    "sms", 
    "attachments",
}

// MarkerFileContent defines the content of .mobilecombackup.yaml
type MarkerFileContent struct {
    RepositoryStructureVersion string `yaml:"repository_structure_version"`
    CreatedAt                  string `yaml:"created_at"`
    CreatedBy                  string `yaml:"created_by"`
}

// DefaultMarkerContent returns the default marker file content
func DefaultMarkerContent(version string) MarkerFileContent {
    return MarkerFileContent{
        RepositoryStructureVersion: "1",
        CreatedAt:                  time.Now().UTC().Format(time.RFC3339),
        CreatedBy:                  fmt.Sprintf("mobilecombackup v%s", version),
    }
}

// SummaryContent defines the initial summary.yaml content
type SummaryContent struct {
    Counts struct {
        Calls int `yaml:"calls"`
        SMS   int `yaml:"sms"`
    } `yaml:"counts"`
}

// InitResult contains the result of initialization
type InitResult struct {
    RepoRoot string
    Created  []string // Directories and files created
    DryRun   bool
}
```

### Implementation Notes
- Use os.MkdirAll with 0750 permissions for directory creation
- Create .mobilecombackup.yaml file with YAML content using gopkg.in/yaml.v3
- Create empty contacts.yaml and summary.yaml with appropriate initial content
- Validate write permissions using os.OpenFile with O_CREATE|O_EXCL
- Check for existing repository by looking for .mobilecombackup.yaml marker file or directory structure
- Error messages:
  - "Directory already contains a mobilecombackup repository" (if .mobilecombackup.yaml exists)
  - "Directory appears to be a repository (found calls/, sms/, or attachments/ directories)" (if dirs exist)
  - "Directory is not empty" (for other non-empty directories)
- Use filepath.Join for cross-platform path handling
- The .mobilecombackup.yaml file should be included in files.yaml during future operations
- Tree-style output should show the created structure hierarchically

## Tasks
- [ ] Add init subcommand to CLI parser
- [ ] Implement directory validation logic
  - [ ] Check if path exists and is empty
  - [ ] Detect existing repository structure (.mobilecombackup.yaml file)
  - [ ] Validate write permissions
- [ ] Implement directory creation logic
  - [ ] Create directories with 0750 permissions
  - [ ] Create .mobilecombackup.yaml marker file with version, timestamp, and CLI version
  - [ ] Create empty contacts.yaml
  - [ ] Create summary.yaml with zero counts
- [ ] Add --dry-run support
- [ ] Add --quiet flag support
- [ ] Implement tree-style output formatting
- [ ] Write unit tests
  - [ ] Test empty directory initialization
  - [ ] Test .mobilecombackup.yaml file creation
  - [ ] Test permission validation
  - [ ] Test existing repository detection
  - [ ] Test dry-run mode
- [ ] Write integration tests
  - [ ] End-to-end initialization test
  - [ ] Test with various directory states
  - [ ] Verify marker file content
- [ ] Update documentation
  - [ ] Add init command to README
  - [ ] Update CLI help text
  - [ ] Document .mobilecombackup.yaml file format

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
- Directory names with spaces or special characters

## Risks and Mitigations
- **Risk**: Partial directory creation on failure
  - **Mitigation**: Since directory must be empty, user can simply delete and retry
- **Risk**: Permission issues on different platforms
  - **Mitigation**: Use standard Go os package permissions (0750)
- **Risk**: Race conditions with concurrent access
  - **Mitigation**: Use file locking or clear documentation about single-user assumption

## References
- Related features: FEAT-006 (enable CLI)
- Code locations: cmd/mobilecombackup/main.go
- Similar patterns: Repository validation in FEAT-001

## Notes
- The .mobilecombackup.yaml marker file enables repository version tracking for future migrations
- The attachments/ directory structure (with hash-based subdirectories) will be created on demand during import
- Future enhancements could extend the marker file with additional metadata (creation date, tool version, etc.)
- FEAT-001 (Repository Validation) will need to be updated to validate the presence of .mobilecombackup.yaml file
- The marker file being missing is a fixable validation violation (can be created with default version "1")
