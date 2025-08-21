# Session Learnings

This document captures implementation-specific learnings from various feature development sessions.

## FEAT-005: Contacts Implementation

### YAML Dependency Management
- **Adding Go dependencies**: Use `devbox run go get package` or enter `devbox shell` and run `go get package`
- **Dependency visibility**: Added `gopkg.in/yaml.v3 v3.0.1` to `go.mod` for contacts YAML parsing
- **Build verification**: Always test compilation after adding dependencies

### ContactsReader Implementation Insights
- **O(1) Lookup Performance**: Built efficient hash maps for phone number→name and name→contact lookups
- **Phone Number Normalization**: Critical for handling various input formats (+1XXXXXXXXXX, (XXX) XXX-XXXX, etc.)
  - Remove all non-digits with regex `\D`
  - Strip leading "1" for 11-digit US numbers
  - Provides consistent lookups regardless of input format
- **Duplicate Detection**: Validate during load that no phone number appears in multiple contacts
- **Error Handling**: Graceful handling of missing `contacts.yaml` (not an error condition)

### Testing Excellence Patterns
- **High Coverage Achievement**: FEAT-005 achieved 97.3% test coverage through comprehensive unit tests
- **Integration Test Strategies**:
  - Large dataset testing (100 contacts) for performance validation
  - Phone number format variation testing
  - Reload functionality testing
  - Empty repository handling
- **Example Documentation**: Provide extensive `example_test.go` with real-world usage patterns

### Non-XML Data Handling
- **YAML Parsing**: Used `gopkg.in/yaml.v3` for structured contact data
- **Data Validation**: Check for empty contact names and duplicate phone numbers during load
- **Memory Efficiency**: Use maps for O(1) lookups while providing array access via `GetAllContacts()`

### Architecture Pattern Consistency
- **Interface-First Design**: Defined `ContactsReader` interface before implementation
- **Separation of Concerns**: Clear split between types, implementation, tests, and examples
- **Error Resilience**: Methods work safely on unloaded managers (return empty/false results)
- **Copy Semantics**: Return copies from `GetAllContacts()` to prevent external modification

### Specification Maintenance
- **Living Documentation**: Update `issues/specification.md` with each completed feature
- **API Documentation**: Include interface definitions and key features in specification
- **Cross-Referencing**: Maintain links between completed features and specification sections

## FEAT-004: Attachments Implementation

### Hash-Based Storage Implementation
- **Content-Addressed Storage**: Use SHA-256 hashes for immutable content addressing
- **Directory Sharding**: First 2 characters of hash as subdirectory prevents filesystem bottlenecks
  - Structure: `attachments/ab/ab123456...` for hash starting with "ab"
  - Prevents too many files in single directory (performance optimization)
- **Hash Validation**: Implement strict 64-character lowercase hex validation
  - Use regex `^[0-9a-f]{64}$` for validation
  - Normalize input to lowercase for consistent lookup
- **Path Generation**: Simple algorithm: `attachments/[hash[:2]]/[hash]`

### Binary File Handling Patterns
- **Content Verification**: Always verify file content matches expected hash
- **Binary Data Processing**: Handle arbitrary binary content (images, videos, etc.)
- **File Size Tracking**: Store file sizes for statistics and validation
- **Existence Checking**: Distinguish between "file doesn't exist" vs "file exists but corrupted"

### Test Data Management for Binary Content
- **Generate Real Test Data**: Use crypto/sha256 to create proper 64-character hashes
- **Temporary Hash Generation**: Create temporary files for hash calculation when needed
  ```go
  hasher := sha256.New()
  hasher.Write(content)
  hash := fmt.Sprintf("%x", hasher.Sum(nil))
  ```
- **Test with Various Content**: Test with different content sizes and types
- **Handle Missing Test Data**: Gracefully skip integration tests when test data unavailable

### Directory Structure Validation Patterns
- **Comprehensive Validation**: Check directory names, file placement, and structure integrity
- **Error Accumulation**: Collect all validation errors, don't stop at first error
- **Context-Rich Errors**: Include specific paths and violation details in error messages
- **Validation Categories**:
  - Directory name format (2 lowercase hex characters)
  - File placement (hash must start with directory name)
  - No files in root attachments directory
  - No unexpected subdirectories

### Streaming and Memory Efficiency
- **Callback-Based Streaming**: Use `func(item) error` pattern for memory-efficient processing
- **Directory Walking**: Use `os.ReadDir` for efficient directory traversal
- **Error Handling in Streams**: Continue processing on individual failures, log errors
- **Performance Testing**: Test with 100+ items to verify streaming efficiency

### Cross-Package Integration Patterns
- **Referenced Hash Maps**: Use `map[string]bool` for efficient lookup of referenced items
- **Orphan Detection**: Cross-reference with other packages (SMS reader) to find unreferenced items
- **Statistics Collection**: Aggregate data across multiple operations for repository analysis
- **Interface Consistency**: Follow same patterns as other readers (List, Stream, Validate, etc.)

### File System Operation Best Practices
- **Graceful Missing Directories**: Empty repository (no attachments/) is valid, not an error
- **Atomic Operations**: Read file metadata and content separately for better error handling
- **Path Manipulation**: Use `filepath.Join` for cross-platform compatibility
- **Temporary Directory Testing**: Use `t.TempDir()` for isolated test environments

### Error Handling for File Operations
- **Distinguish Error Types**: File not found vs permission errors vs validation failures
- **Context Preservation**: Wrap errors with additional context about what operation failed
- **Validation vs Runtime Errors**: Separate validation failures from system errors
- **Robust Directory Walking**: Continue processing other files when individual files fail

### Performance Optimization Strategies
- **Lazy Loading**: Only read file content when specifically requested
- **Efficient Metadata**: Use `os.Stat` for size/existence without reading content
- **Directory Caching**: Could cache directory listings for repeated operations
- **Hash Calculation**: Only calculate hashes during verification, not regular operations
- **Streaming APIs**: Mandatory for large repositories to prevent memory issues

## CLI Subcommand Implementation (FEAT-006, FEAT-014, FEAT-007)

### Cobra CLI Framework Integration
- **Package Structure**: Create `cmd/mobilecombackup/cmd/` directory for command implementations
- **Root Command Setup**: Define global flags in `root.go` (--quiet, --repo-root, --version, --help)
- **Subcommand Pattern**: Each subcommand gets its own file (init.go, validate.go, etc.)
- **Version Injection**: Use ldflags during build for version information
  ```bash
  go build -ldflags "-X main.Version=1.2.3" -o mobilecombackup
  ```

### CLI Testing Patterns
- **Unit Tests**: Test individual command functions and flag parsing
- **Integration Tests**: Build binary and test via exec.Command for realistic behavior
- **Test Binary Creation**: Build once per test suite to avoid repeated compilation
  ```go
  testBin := filepath.Join(t.TempDir(), "mobilecombackup-test")
  buildCmd := exec.Command("go", "build", "-o", testBin, "../../../cmd/mobilecombackup")
  ```
- **Exit Code Testing**: Use exitErr.ExitCode() to verify proper exit codes
- **Output Verification**: Capture stdout/stderr and verify expected content

### Integration Test Best Practices
- **Avoid Duplicate Functions**: Package-level test functions can conflict across files
- **Test Repository Creation**: Create minimal valid repositories for testing
- **Handle Working Directory**: Some tests change directories - save and restore
- **Environment Variable Testing**: Test MB_REPO_ROOT and other env vars properly
- **Performance Testing**: Include tests with larger datasets to ensure scalability

### Validation Command Specifics
- **Repository Reader Creation**: Must create all readers (calls, SMS, attachments, contacts)
- **JSON Output Formatting**: ValidationViolation struct needs proper JSON tags for lowercase fields
- **Progress Reporting**: Interface pattern for future enhancement without breaking changes
- **Output Modes**: Implement quiet (violations only), verbose (detailed progress), and JSON
- **Exit Codes**: 0=valid, 1=violations found, 2=runtime error

### Init Command Implementation Details
- **Directory Validation**: Check for existing repository markers (.mobilecombackup.yaml, calls/, etc.)
- **Tree Output**: Implement custom tree rendering for created structure display
- **Dry Run Support**: Preview actions without creating files/directories
- **Marker File Creation**: Include version, timestamp (RFC3339), and creator info
- **File Permissions**: Use 0750 for directories, 0644 for files

### Common Pitfalls and Solutions
- **Tree Rendering Issues**: Root node needs special handling (isRoot parameter)
- **Test Data Requirements**: Validation tests need complete repository structure
- **JSON Field Names**: Go structs need json tags for proper JSON marshaling
- **Integration Test Isolation**: Each test should create its own temp directory
- **Help Command**: Cobra automatically provides help command - update tests accordingly

### Documentation Requirements for CLI Commands
- **README Updates**: Add command usage examples and descriptions
- **Exit Code Documentation**: Clearly document what each exit code means
- **Flag Documentation**: Include all flags with examples
- **Output Examples**: Show both success and failure output examples
- **Specification Updates**: Add command details to issues/specification.md

### Test Coverage Expectations
- **Unit Tests**: Cover all command functions, flag parsing, output formatting
- **Integration Tests**: Test real command execution with various scenarios
- **Edge Cases**: Non-existent paths, permission issues, malformed input
- **Performance Tests**: Ensure commands scale with repository size

## Common Implementation Patterns

### High Test Coverage Achievement
Projects achieved excellent test coverage through systematic testing:
- FEAT-002 (calls): 85.5% coverage
- FEAT-003 (sms): 81.2% coverage
- FEAT-004 (attachments): 84.8% coverage
- FEAT-005 (contacts): 97.3% coverage

### Temporary File Management
- **Location**: Always use `tmp/` directory in the repository root, never `/tmp` or other system directories
- **Cleanup**: Delete temporary files immediately after use
- **Git ignore**: The `tmp/` directory should be in `.gitignore` to prevent accidental commits
- **Naming**: Use descriptive names that indicate purpose (e.g., `tmp/test-date-conversion.go`)

### Common Issues and Solutions
- **Devbox environment problems**: Use `devbox run command` to run commands without entering the shell
- **Legacy code conflicts**: Remove old implementations when starting fresh features
- **Import path issues**: Always use full module path `github.com/phillipgreenii/mobilecombackup/pkg/...`
- **Date conversion**: Timestamps are in milliseconds, not seconds - divide by 1000 for Unix time
- **Empty vs null**: XML attributes with value "null" should be treated as empty/zero values
- **MMS type field**: Use `msg_box` (1=received, 2=sent) not `m_type` for message direction