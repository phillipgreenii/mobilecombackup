# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go command-line tool for processing mobile phone backup files (Call and SMS logs in XML format). It coalesces multiple backup files, removes duplicates, extracts attachments, and organizes data by year.

## Development Commands

### Building
```bash
# Build all packages
go build -v ./...
# or using devbox:
devbox run builder
```

### Testing
```bash
# Run all tests with coverage
go test -v -covermode=set ./...
# or using devbox:
devbox run tests

# Run a specific test
go test -v -run TestName ./pkg/packagename
```

### Linting and Formatting
```bash
# Format code (always run first)
go fmt ./...
# or using devbox:
devbox run formatter

# Run golangci-lint
golangci-lint run
# or using devbox:
devbox run linter
```

### Quality Workflow
The recommended quality workflow is: **format → test → lint**
```bash
# Complete quality workflow
devbox run formatter
devbox run tests
devbox run linter
```

### Development Environment
```bash
# Enter devbox development shell with all dependencies
devbox shell

# The environment includes: go 1.24, golangci-lint, claude-code

# Build all packages
devbox run builder

# Run tests directly
devbox run tests

# Run linter directly  
devbox run linter

# Alternative: Run one-off commands in devbox environment
devbox run go build ./pkg/calls
devbox run go test -v -covermode=set ./pkg/calls
```

## Architecture

### Package Structure
- **cmd/mobilecombackup**: Main CLI entry point with command parsing and orchestration
- **pkg/calls**: Call log processing
  - `types.go`: Call struct and CallType constants  
  - `reader.go`: CallsReader interface for reading from repository
  - `xml_reader.go`: XMLCallsReader implementation with streaming support
  - `*_test.go`: Comprehensive unit and integration tests
  - `example_test.go`: Usage documentation and examples
- **pkg/sms**: SMS/MMS message processing
  - `types.go`: Message interface, SMS and MMS structs, MessageType constants
  - `reader.go`: SMSReader interface for reading from repository
  - `xml_reader.go`: XMLSMSReader implementation with streaming support
  - `*_test.go`: Comprehensive unit and integration tests
  - `example_test.go`: Usage documentation and examples
- **pkg/contacts**: Contact information management
  - `types.go`: Contact struct and ContactsReader interface
  - `reader.go`: ContactsManager implementation with YAML parsing
  - `*_test.go`: Comprehensive unit and integration tests
  - `example_test.go`: Usage documentation and examples
- **pkg/attachments**: Attachment file management
  - `types.go`: Attachment struct and AttachmentReader interface
  - `reader.go`: AttachmentManager implementation with hash-based storage
  - `*_test.go`: Comprehensive unit and integration tests
  - `example_test.go`: Usage documentation and examples
- **pkg/coalescer**: Core deduplication logic using hash-based comparison
- **pkg/mobilecombackup**: Main processing logic, interfaces (Processor, BackupReader, BackupWriter), and CLI implementation
- **internal/**: Test utilities and integration tests

### Key Interfaces
- `BackupEntry`: Base interface with `Timestamp()` and `Year()` methods
- `BackupReader[B BackupEntry]`: Reads XML files and returns `[]BackupEntryMetadata`
- `BackupWriter[V any]`: Writes processed entries to XML files
- `Coalescer[V any]`: Manages deduplication with `Add()` and `Dump()` methods
- `CallsReader`: Reads call records from repository with methods for streaming, validation, and metadata
- `SMSReader`: Reads SMS/MMS records from repository with attachment tracking capabilities
- `Message`: Base interface for SMS and MMS messages with common accessors
- `ContactsReader`: Reads contact information from repository with phone number normalization and lookup capabilities
- `AttachmentReader`: Reads attachment files from repository with hash-based storage and verification capabilities

### Key Types
- `BackupEntryMetadata[B BackupEntry]`: Wraps entries with hash, year, and error info
- `CoalImpl[V any]`: Implementation with memory store (map[string][V])
- `CoalesceSummary`: Tracks added, duplicate, and error counts

### Processing Flow
1. Parse CLI arguments for repo root and paths to process
2. Initialize coalescers and read existing repository
3. Walk additional paths, adding files to appropriate coalescers
4. For each entry:
   - Remove `readable_date` field (timezone-dependent)
   - Extract and write attachments to disk (hash-based naming)
   - Calculate hash on all fields except `readable_date`
   - Check for duplicates by hash
5. Sort all entries by timestamp
6. Partition by year (using `date` field)
7. Write to repository structure with recalculated `readable_date` in EST

### Repository Output Structure
```
repository/
├── calls/
│   ├── calls-2015.xml
│   └── calls-2016.xml
├── sms/
│   ├── sms-2015.xml
│   └── sms-2016.xml
└── attachments/
    └── [hash-based subdirectories]
```

## Important Implementation Details

### Generic XML Parsing
- Parse XML generically into map of maps to handle schema changes
- Top-level element contains count (verify against actual entries)
- Child elements processed with:
  - `readable_date` removed before hashing
  - Attachments extracted and stored by hash
  - Hash calculated on all fields except `readable_date`

### SMS/MMS Structure
- **SMS messages**: Simple structure with attributes like address, body, date, type
- **MMS messages**: Complex structure with multiple parts and addresses:
  - Parts array containing text, images, and SMIL presentation data
  - Each part has content type, charset, and either text or data field
  - SMIL parts (application/smil) define visual presentation layout
  - Text parts (text/plain) contain the actual message content
  - Image parts contain base64 data that needs extraction
  - Address field is primary recipient/sender
  - Additional addresses in separate addresses array for group messages
- **Message types**: Determined by `type` field for SMS (1=received, 2=sent) and `msg_box` field for MMS

### Attachment Storage
- Convert base64 data to bytes and hash
- Store in `attachments/[first-2-chars]/[full-hash]` structure
- Replace attachment field with relative path to file
- Example: `attachments/ac/ac78543342e3`
- MMS attachments are found in the `data` field of parts with image/video content types

### Deduplication
- Memory store using map[string][V] keyed by hash
- Process existing repository first, then new files
- Track added, duplicate, and error counts
- `readable_date` recalculated using EST timezone on write

### Error Handling
- Continue processing despite errors (track but don't fail)
- Missing/empty repository is warning, not failure
- Failed repository read causes app failure
- Per-entry errors collected and reported at end

### CLI Usage
```bash
mobilecombackup --repo-root path/to/reporoot path/to/file_to_import path/to/directory/to/scan
```

## Design Decisions

### Core Principles
- **Append-only system**: No updates to existing data, only new additions
- **Per-record resilience**: Each record processed independently; one failure doesn't stop others
- **No data removal**: Attachments are never deleted, so reference counting unnecessary
- **Manual conflict resolution**: Rare conflicts handled manually, not automatically

### Attachment Handling
- **Deduplication**: Multiple SMS messages can reference the same attachment file
- **Hash algorithm**: SHA-256 for content addressing
- **Storage**: `attachments/[first-2-chars]/[full-hash]` structure remains permanent

### Error Handling Strategy
- **Rejected records**: Written to `rejected/` directory with metadata
- **Rejection reasons**:
  - `missing-timestamp`: No valid date field
  - `malformed-attachment`: Base64 decode failure
  - `parse-error`: XML parsing failure
- **Rejected file format**:
  ```xml
  <rejected-entries>
    <entry rejection-reason="missing-timestamp" source-file="backup-2024-01-15.xml">
      <!-- original XML content -->
    </entry>
  </rejected-entries>
  ```

### Timestamp and Ordering
- **UTC-based**: All timestamp storage and year partitioning uses UTC
- **Same timestamp handling**: Records with identical timestamps both kept
- **Secondary sort**: Preserve insertion order for identical timestamps
- **Year extraction**: Based on UTC interpretation of `date` field

### Summary Output Enhancement
Should include rejected record counts:
```
              Initial     Final     Delta     Duplicates    Rejected    Errors
Calls              10        12         2              5           1         0
SMS                23        33        10             17           2         0
```

## Issue Development Workflow

### Issue Documentation Structure
All issues (features and bugs) are documented in the `issues/` directory:
```
issues/
├── feature_template.md # Template for new features
├── bug_template.md     # Template for bug reports
├── next_steps.md       # Notes about current work and what to start next
├── active/             # Issues actively being implemented/fixed
│   ├── FEAT-XXX-name.md
│   └── BUG-XXX-name.md
├── ready/              # Fully planned issues ready for implementation
│   ├── FEAT-XXX-name.md
│   └── BUG-XXX-name.md
├── backlog/            # Issues being planned; not yet ready
│   ├── FEAT-XXX-name.md
│   └── BUG-XXX-name.md
├── completed/          # Resolved issues (reference)
│   ├── FEAT-XXX-name.md
│   └── BUG-XXX-name.md
└── README.md           # Overview
```

### Issue Numbering
- All issues (features and bugs) share a single sequential numbering system
- Numbers are unique across all issue types (e.g., FEAT-001, BUG-002, FEAT-003)
- When creating a new issue, use the next number after the highest existing FEAT or BUG number

### Feature Workflow
1. **Create Feature Document**
   - Find the highest issue number across all FEAT-XXX and BUG-XXX files
   - Copy `issues/feature_template.md` to `issues/backlog/FEAT-XXX-descriptive-name.md`
   - Use the next sequential number
   - Start with minimal details, filling in as you plan

2. **Planning Phase**
   - Define requirements (functional and non-functional)
   - Document design approach and key decisions
   - Break down into specific tasks
   - Identify risks and dependencies
   - Once planning is complete, move the feature to `issues/ready/`.

3. **Implementation Phase**
   - When implementation starts, move the feature to `issues/active/`.
   - Check off tasks as completed
   - Update implementation notes with decisions made
   - Keep design sections current with actual implementation

4. **Completion**
   - Update completed status to today's date.
   - Move feature to `completed/`
   - Update with final implementation details
   - Document any deviations from original plan

### Bug Workflow
1. **Create Bug Report**
   - Find the highest issue number across all FEAT-XXX and BUG-XXX files
   - Copy `issues/bug_template.md` to `issues/backlog/BUG-XXX-descriptive-name.md`
   - Use the next sequential number
   - Include reproduction steps, expected vs actual behavior

2. **Investigation Phase**
   - Reproduce the issue
   - Identify root cause
   - Document findings in Root Cause Analysis section
   - Define fix approach
   - Once investigation is complete, move the bug to `issues/ready/`.

3. **Fix Implementation**
   - When fix starts, move the bug to `issues/active/`.
   - Implement the fix
   - Add regression tests
   - Verify fix resolves the issue

4. **Completion**
   - Update fixed date to today's date.
   - Move bug to `completed/`
   - Document verification steps

### Task Management
When planning issues:
- Not all sections are required, some may be removed; new sections may also be added.
- Cross-reference related issues in the References section

When implementing issues:
- Work on one task at a time
- Update task checkboxes in the feature document
- Reference the feature document in commits (e.g., "FEAT-001: Implement generic XML parser")
- Cross-reference related features in the References section

## Issue Development Best Practices

Based on analysis of existing issues, the following patterns and best practices should be followed:

### Issue Structure
- **Consistent sections**: Status, Overview, Background, Requirements, Design, Tasks, Testing, References
- **Clear priority levels**: high/medium/low based on user impact and dependencies
- **Explicit dependencies**: Use "Pre-req:", "Depends on:", "Related:" prefixes
- **Reference specifications**: Link to relevant sections in specification.md

### Design Principles
- **Interface-first design**: Define clear APIs before implementation
- **Memory efficiency**: Use streaming APIs for large data processing
- **Error resilience**: Continue processing on individual failures, collecting errors for reporting
- **Graceful degradation**: Handle missing/optional fields without failing
- **Conservative auto-fixes**: Only fix what's safe to fix automatically
- **Explicit dangerous operations**: Require specific flags for destructive actions

### Implementation Patterns
- **Streaming for large files**: Process data in chunks to avoid memory issues
- **Hash-based operations**: Use SHA-256 consistently for deduplication and content addressing
- **Progress reporting**: Report progress every 100 entries for long operations
- **Atomic operations**: Ensure related changes succeed or fail together
- **Performance targets**: Set specific throughput goals (e.g., 10,000 records/second for calls, 5,000 messages/second for SMS)
- **Null value handling**: Check for "null" string values in XML attributes and convert to empty strings
- **Interface-based polymorphism**: Use base interfaces (like Message) to handle different types uniformly
- **Type determination**: Use numeric fields (type, msg_box) to determine message direction
- **Complex nested parsing**: Handle nested XML structures (MMS parts/addresses) with separate parser functions

### Testing Strategy
- **Three-tier testing**: Always include Unit Tests, Integration Tests, and Edge Cases
- **Edge case focus**: Empty data, corrupted inputs, large datasets, invalid formats, special characters
- **Performance testing**: Include benchmarks for data-intensive operations
- **Validation testing**: Verify all validation rules with positive and negative test cases

### Task Breakdown Pattern
1. Define data structures/interfaces
2. Implement core logic
3. Add validation/error handling
4. Write tests (unit, then integration)
5. Add documentation/examples
6. Performance optimization (if needed)

### Error Handling
- **Context-rich errors**: Include file paths, line numbers, and specific violation types
- **Structured rejection format**: Use consistent XML format for rejected entries
- **Exit codes**: 0=success, 1=violations found, 2=errors occurred
- **Violation tracking**: Track and report all issues found during processing

### Documentation Requirements
- **Usage examples**: Provide multiple scenarios showing different use cases
- **Output format examples**: Show expected outputs clearly
- **API documentation**: Include Go code examples in Design sections
- **Command-line examples**: Demonstrate various flag combinations

## Implementing Issues: Practical Workflow

### Initial Setup
1. **Move issue to active**: `git mv issues/ready/FEAT-XXX.md issues/active/FEAT-XXX.md`
2. **Commit the move**: Use format "Starting to implement FEAT-XXX"
3. **Use TodoWrite tool**: Create todo list from feature tasks for progress tracking

### Development Cycle
1. **Work on one task at a time**: Focus on single task until completion
2. **Format code before testing**: Always run `devbox run formatter` before testing or committing
3. **Task completion verification**: Before marking any task complete, MUST verify clean state:
   - Run `devbox run tests` - all tests must pass
   - Run `devbox run linter` - zero lint violations allowed
   - Run `devbox run build-cli` - build must succeed
   - Fix any failures found before marking task complete
4. **Remove conflicting code**: If feature conflicts with legacy code, remove the old implementation
5. **Commit individual tasks**: Never use `git add .` or `git commit -a` - explicitly add files
6. **Reference feature in commits**: Include "FEAT-XXX:" prefix in commit messages
7. **Temporary files**: Create temp files in `tmp/` within the repository (not `/tmp`), clean up after use, never commit them

### Testing Best Practices
1. **Test early and often**: Build and test after each significant change
2. **Use existing test data**: Leverage files in `testdata/` for integration tests
3. **Handle test data quirks**: Some test files have count mismatches (e.g., count="56" but 12 entries)
4. **Aim for high coverage**: Target 80%+ test coverage with `go test -covermode=set`
   - FEAT-002 (calls): Achieved 85.5% coverage
   - FEAT-003 (sms): Achieved 81.2% coverage
   - FEAT-004 (attachments): Achieved 84.8% coverage
   - FEAT-005 (contacts): Achieved 97.3% coverage
5. **Test realistic scenarios**: Use actual test data from `testdata/archive/` and `testdata/it/`
6. **Integration test pattern**: Copy test files to temp directory to simulate repository structure
7. **Test both success and failure paths**: Validate that validation functions properly detect errors

### File Organization Patterns
1. **Separate concerns**: Split types, interfaces, implementations, tests, and examples
2. **Use descriptive names**: `types.go`, `reader.go`, `xml_reader.go`, `*_test.go`, `example_test.go`
3. **Group related functionality**: Keep interfaces and implementations in same package
4. **Add usage examples**: Create `example_test.go` with runnable code examples

### Temporary File Management
1. **Location**: Always use `tmp/` directory in the repository root, never `/tmp` or other system directories
2. **Cleanup**: Delete temporary files immediately after use
3. **Git ignore**: The `tmp/` directory should be in `.gitignore` to prevent accidental commits
4. **Naming**: Use descriptive names that indicate purpose (e.g., `tmp/test-date-conversion.go`)
5. **Example**:
   ```bash
   # Create temp file for testing
   mkdir -p tmp
   cat > tmp/test-snippet.go << 'EOF'
   // temporary test code
   EOF
   
   # Use the file
   go run tmp/test-snippet.go
   
   # Clean up immediately
   rm tmp/test-snippet.go
   ```

### Git Workflow Details
- Only commit files in which you modified as part of the work.

```bash
# Safe file staging (NEVER use git add . or git commit -a)
git add pkg/specific/file.go
git add pkg/specific/file_test.go

# Commit with feature reference and description
git commit -m "$(cat <<'EOF'
FEAT-XXX: Brief description of what was implemented

Detailed explanation of changes:
- What was added/changed
- Why it was needed
- Any important implementation notes

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

### Task Completion Verification

Before marking any TodoWrite task as complete, agents MUST verify clean state by running all verification commands and fixing any failures found:

#### Required Verification Commands
1. **Full Test Suite**: `devbox run tests`
   - All tests must pass (no failures, no compilation errors)
   - Fix any test failures before proceeding
   
2. **Full Linter**: `devbox run linter`  
   - Zero lint violations allowed
   - Fix all lint issues before proceeding
   
3. **CLI Build**: `devbox run build-cli`
   - Build must succeed without errors
   - Fix any compilation issues before proceeding

#### Development Process
- **Incremental Testing**: Agents MAY use targeted commands during development for efficiency:
  - `go test ./pkg/specific` for individual package testing
  - `golangci-lint run ./pkg/specific` for targeted linting
  - Quick builds with `go build ./pkg/specific`
  
- **Final Verification**: MUST run complete verification before task completion:
  - No exceptions - all three verification commands must succeed
  - If any command fails, task remains incomplete until fixed

#### Common Auto-Fix Patterns
- **Test Failures**:
  - `undefined: functionName` → Add missing import or fix typo
  - `cannot use x (type A) as type B` → Add type conversion  
  - `declared but not used` → Remove unused variable or add usage
  - Missing test data files → Create required files in testdata/
  
- **Lint Violations**:
  - `declared but not used` → Remove unused variables/imports/functions
  - `Error return value is not checked` → Add proper error handling
  - `should have comment or be unexported` → Add documentation comments
  - Formatting issues → Run `gofmt` or use `devbox run formatter`
  
- **Build Failures**:
  - Missing imports → Add required imports
  - Syntax errors → Fix code syntax
  - Missing dependencies → Run `go mod tidy` or add dependencies

#### When to Ask User
- Test logic appears incorrect (wrong expected values)
- Multiple valid approaches to fix a lint violation  
- Fix would significantly change program behavior
- Unfamiliar error patterns not covered by common fixes
- Repeated failures after multiple fix attempts

## TodoWrite Integration and Completion Verification

### Mandatory TodoWrite Workflow
When using the TodoWrite tool for task management, agents MUST follow the completion verification workflow:

1. **Task Status Management**:
   - Tasks start as `"status": "pending"`
   - Change to `"status": "in_progress"` when beginning work
   - Only change to `"status": "completed"` after FULL verification passes

2. **Completion Verification Requirements**:
   Before marking any TodoWrite task as `"completed"`, agents MUST:
   ```bash
   # All three commands must succeed:
   devbox run tests    # Must pass with zero failures
   devbox run linter   # Must pass with zero violations  
   devbox run build-cli # Must build successfully
   ```

3. **No Exceptions Policy**:
   - If ANY verification command fails, task remains `"in_progress"`
   - Fix all issues before attempting completion again
   - Document any fixes made in task comments

4. **Auto-Fix Integration**:
   When verification fails, agents should:
   - Apply common auto-fixes for known patterns
   - Re-run verification after each fix
   - Ask user for guidance on complex issues
   - Continue until all verification passes

### TodoWrite Best Practices with Verification

1. **Task Granularity**: Break down tasks small enough that verification can pass for each individual task
2. **Progress Updates**: Update TodoWrite status immediately when starting/completing verification
3. **Error Documentation**: If verification fails, update task with details of what was fixed
4. **Incremental Development**: Use targeted testing during development, full verification at completion

### Example TodoWrite Workflow

```javascript
// Starting a task
{"id": "task-1", "content": "Add user authentication", "status": "in_progress", "priority": "high"}

// During development - incremental testing allowed
go test ./pkg/auth

// Before marking complete - MANDATORY full verification
devbox run tests   // All tests must pass
devbox run linter  // Zero violations required
devbox run build-cli // Build must succeed

// Only after all verification passes:
{"id": "task-1", "content": "Add user authentication", "status": "completed", "priority": "high"}
```

### Integration with Agent Templates

All code-implementation agents must:
- Check TodoWrite task status before beginning work
- Update status to `"in_progress"` when starting
- Run full verification before marking `"completed"`
- Document any issues found and fixed during verification

### Issue Completion
1. **Update task checkboxes**: Mark all tasks as `[x]` in issue document
2. **Set completion date**: Update Status section with completion date
3. **Move to completed**: `git mv issues/active/FEAT-XXX.md issues/completed/FEAT-XXX.md` (or BUG-XXX.md)
4. **Final commit**: Commit the completed issue document

### Common Issues and Solutions
- **Devbox environment problems**: Use `devbox run command` to run commands without entering the shell
- **Test data count mismatches**: These are expected in some files; use for validation testing
- **Legacy code conflicts**: Remove old implementations when starting fresh features
- **Import path issues**: Always use full module path `github.com/phillipgreen/mobilecombackup/pkg/...`
- **Date conversion**: Timestamps are in milliseconds, not seconds - divide by 1000 for Unix time
- **Empty vs null**: XML attributes with value "null" should be treated as empty/zero values
- **Unused variables in tests**: Remove unused test fixtures to avoid compilation errors
- **Year validation**: Test data often contains mixed years - adjust tests accordingly
- **MMS type field**: Use `msg_box` (1=received, 2=sent) not `m_type` for message direction

## Common Test Failure Patterns and Fixes

### Compilation Errors in Tests
- **`undefined: functionName`**:
  - **Cause**: Missing import or typo in function name
  - **Fix**: Add missing import (`import "package/path"`) or correct function name
  - **Example**: `undefined: filepath` → add `import "path/filepath"`

- **`cannot use x (type A) as type B`**:
  - **Cause**: Type mismatch in function arguments or return values
  - **Fix**: Add explicit type conversion or fix function signature
  - **Example**: `string` to `[]byte` → use `[]byte(stringVar)`

- **`declared but not used`**:
  - **Cause**: Variable declared but never referenced
  - **Fix**: Remove unused variable or add usage (use `_` for intentionally unused)
  - **Example**: `result := someFunc()` not used → `_ = someFunc()` or use result

### Test Logic Errors
- **Wrong expected values**:
  - **Cause**: Test expects incorrect data or counts
  - **Fix**: Verify actual test data content and adjust expectations
  - **Example**: Test expects 56 entries but file has 12 → update expected count

- **Path resolution issues**:
  - **Cause**: Relative paths incorrect from test directory
  - **Fix**: Use correct relative paths or absolute paths with `filepath.Abs()`
  - **Example**: `../../../../testdata/` → `../../../testdata/` from cmd/mobilecombackup/cmd/

- **Exit code mismatches**:
  - **Cause**: Unit tests vs integration tests expect different error handling
  - **Fix**: Use `os.Exit()` for integration tests, return errors for unit tests
  - **Example**: Use `testing.Testing()` to detect test mode

### Test Data and File Issues
- **Missing test data files**:
  - **Cause**: Tests reference non-existent files
  - **Fix**: Create required files in `testdata/` or update test paths
  - **Example**: Create `testdata/to_process/00/calls-test.xml`

- **Permission errors**:
  - **Cause**: Test files have wrong permissions
  - **Fix**: Fix file/directory permissions using `chmod` or `os.Chmod()`
  - **Example**: `os.Chmod(dir, 0755)` for directories

- **Empty XML causing rejections**:
  - **Cause**: Test uses empty XML files that get rejected during import
  - **Fix**: Use `--no-error-on-rejects` flag or provide realistic test data
  - **Example**: Add flag to test command or use actual call/SMS records

### Integration vs Unit Test Patterns
- **Unit Tests**: Test command functions directly via `rootCmd.Execute()`
  - Expect errors to be returned, not `os.Exit()` calls
  - Mock external dependencies
  - Focus on logic validation

- **Integration Tests**: Test binary execution via `exec.Command`
  - Expect specific exit codes via `os.Exit()` calls
  - Use real file system and external dependencies
  - Focus on end-to-end behavior

## Common Lint Violation Patterns and Fixes

### Error Handling Violations
- **`Error return value is not checked (errcheck)`**:
  - **Cause**: Function returning error not handled
  - **Fix**: Add proper error handling or use `_` to explicitly ignore
  - **Examples**:
    - `file.Close()` → `_ = file.Close()` or `defer func() { _ = file.Close() }()`
    - `os.Setenv(k, v)` → `_ = os.Setenv(k, v)` in tests

### Unused Code Violations
- **`declared but not used (unused)`**:
  - **Cause**: Variables, functions, or imports not referenced
  - **Fix**: Remove unused code or add usage
  - **Examples**:
    - Unused import → Remove from import statement
    - Unused variable → Remove declaration or add usage
    - Unused function → Remove or mark as used in tests

### Documentation Violations
- **`should have comment or be unexported (golint)`**:
  - **Cause**: Exported functions/types missing documentation
  - **Fix**: Add proper documentation comments
  - **Example**: `// ProcessCalls processes call records from XML files`

### Static Analysis Violations
- **`empty branch (staticcheck)`**:
  - **Cause**: Empty if/else branches that do nothing
  - **Fix**: Add meaningful code or remove empty branch
  - **Example**: Replace empty `if` with comment explaining why no action needed

- **`could use tagged switch (staticcheck)`**:
  - **Cause**: Complex if/else chain that could be a switch
  - **Fix**: Refactor to use switch statement for clarity
  - **Example**: Convert `if result.Action == "extracted"` chain to switch

### Import and Formatting Issues
- **Import ordering**:
  - **Cause**: Imports not in standard Go order
  - **Fix**: Use `goimports` or `devbox run formatter` to fix
  - **Standard order**: stdlib, third-party, local packages

- **Formatting inconsistencies**:
  - **Cause**: Code not formatted according to `gofmt` standards
  - **Fix**: Run `gofmt` or `devbox run formatter`

## Test Data Structure and Usage

### Test Data Locations
- `testdata/archive/`: Original backup files (calls.xml with 16 entries, realistic data)
- `testdata/to_process/`: Files to be processed (calls-test.xml with count issues)
- `testdata/it/scenerio-00/`: Integration test scenario with before/after states
  - `original_repo_root/`: Existing repository state
  - `to_process/`: New files to import  
  - `expected_repo_root/`: Expected result after processing

### Test Data Characteristics
- **Count mismatches**: Some files have `count="56"` but only 12 actual entries
- **Mixed years**: Test data may span multiple years (2013-2015 in SMS, 2014-2015 in calls)
- **Realistic content**: Uses 555 phone numbers, anonymized contacts
- **Binary attachments**: MMS files contain real PNG data in base64
- **Validation opportunities**: Count mismatches are useful for testing validation logic
- **SMS test data**: `sms-test.xml` contains mix of SMS (6) and MMS (9) messages
- **MMS complexity**: Test MMS messages include SMIL parts, text parts, and group messages
- **Character encoding**: MMS parts use various charsets (106, 3, null)
- **Special characters**: SMS bodies contain escaped XML entities (&amp; for &)

### Using Test Data Effectively
```go
// Copy test data to temporary location for integration tests
func copyFile(src, dst string) error {
    // Create destination directory
    if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
        return err
    }
    // Copy file content...
}

// Test with realistic data from testdata/
err := copyFile("../../testdata/archive/calls.xml", 
                filepath.Join(tempDir, "calls", "calls-2014.xml"))
```

## Slash Commands for Issue Development

The repository includes custom slash commands in `.claude/commands/` to streamline issue development:

### Available Commands
- **`/implement-issue FEAT-XXX or BUG-XXX`**: Start implementing an issue following the established workflow
  - Moves issue from `ready/` to `active/`
  - Creates TodoWrite list from issue tasks
  - Ensures code compilation and test passing before task completion
  - Commits only modified files
  - Updates both issue document and `specification.md` on completion

- **`/ready-issue FEAT-XXX or BUG-XXX`**: Validate if an issue has enough detail for implementation
  - Reviews issue document completeness
  - Moves from `backlog/` to `ready/` if sufficiently detailed

- **`/review-issue FEAT-XXX or BUG-XXX`**: Review an issue specification
  - Provides feedback and suggestions for improvements
  - Asks clarifying questions about requirements

- **`/create-feature <description>`**: Create a new feature issue
  - Finds next sequential issue number
  - Creates FEAT-XXX document from template
  - Places in `backlog/` for planning

- **`/create-bug <description>`**: Create a new bug report
  - Finds next sequential issue number
  - Creates BUG-XXX document from template
  - Places in `backlog/` for investigation

- **`/remember-anything-learned-this-session`**: Update CLAUDE.md with session learnings
  - Captures development workflow improvements
  - Documents new patterns and best practices

### Using Slash Commands
These commands are invoked by the user and provide structured prompts for common issue development tasks. They help ensure consistency across issue implementations and reduce the cognitive load of remembering all workflow steps.

## Session Learnings: FEAT-005 Implementation

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

## Session Learnings: FEAT-004 Implementation

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

## Session Learnings: CLI Subcommand Implementation (FEAT-006, FEAT-014, FEAT-007)

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

## Next Steps
See `issues/next_steps.md`

