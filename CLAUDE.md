# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go command-line tool for processing mobile phone backup files (Call and SMS logs in XML format). It coalesces multiple backup files, removes duplicates, extracts attachments, and organizes data by year.

## Development Commands

### Testing
```bash
# Run all tests with coverage
go test -v -covermode=set ./...

# Run a specific test
go test -v -run TestName ./pkg/packagename

# In Nix environment
run-tests
```

### Linting and Formatting
```bash
# Run golangci-lint (ensure it's installed)
golangci-lint run

# In Nix environment - format all files
treefmt

# Pre-commit checks (in Nix environment)
# Includes: check-go, markdownlint, typos, nil
```

### Development Environment
```bash
# Enter Nix development shell with all dependencies
nix develop

# The environment includes: go 1.16, golangci-lint, pre-commit hooks, treefmt

# If nix develop fails due to devenv issues, use nix-shell as fallback:
nix-shell -p go_1_23 --run "command here"

# Example: Build and test
nix-shell -p go_1_23 --run "go build ./pkg/calls"
nix-shell -p go_1_23 --run "go test -v -covermode=set ./pkg/calls"
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

## Feature Development Workflow

### Feature Documentation Structure
All features are documented in the `features/` directory:
```
features/
├── template.md        # Template for new features
├── next_steps.md      # Notes about current work and what to start next
├── active/            # Features actively being implemented
│   └── FEAT-XXX-name.md
├── ready/             # Fully planned features; ready to be implemented
│   └── FEAT-XXX-name.md
├── backlog/           # Features being planned; not yet ready to be implemented
│   └── FEAT-XXX-name.md
├── completed/         # Implemented features (reference)
│   └── FEAT-XXX-name.md
└── README.md          # Overview
```

### Feature Workflow
1. **Create Feature Document**
   - Copy `features/template.md` to `features/backlog/FEAT-XXX-descriptive-name.md`
   - Use sequential numbering (FEAT-001, FEAT-002, etc.)
   - Start with minimal details, filling in as you plan

2. **Planning Phase**
   - Define requirements (functional and non-functional)
   - Document design approach and key decisions
   - Break down into specific tasks
   - Identify risks and dependencies
   - Once planning is complete, move the feature to `features/ready/`.

3. **Implementation Phase**
   - When implementation starts, move the feature to `features/active/`.
   - Check off tasks as completed
   - Update implementation notes with decisions made
   - Keep design sections current with actual implementation

4. **Completion**
   - Update completed status to today's date.
   - Move feature to `completed/`
   - Update with final implementation details
   - Document any deviations from original plan

### Task Management
When planning features:
- Not all sections are required, some may be removed; new sections may also be added.
- Cross-reference related features in the References section

When implementing features:
- Work on one task at a time
- Update task checkboxes in the feature document
- Reference the feature document in commits (e.g., "FEAT-001: Implement generic XML parser")
- Cross-reference related features in the References section

## Feature Development Best Practices

Based on analysis of existing features, the following patterns and best practices should be followed:

### Feature Structure
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

## Implementing Features: Practical Workflow

### Initial Setup
1. **Move feature to active**: `git mv features/ready/FEAT-XXX.md features/active/FEAT-XXX.md`
2. **Commit the move**: Use format "Starting to implement FEAT-XXX"
3. **Use TodoWrite tool**: Create todo list from feature tasks for progress tracking

### Development Cycle
1. **Work on one task at a time**: Focus on single task until completion
2. **Ensure compilability**: Code must compile and tests pass before marking task complete
3. **Remove conflicting code**: If feature conflicts with legacy code, remove the old implementation
4. **Commit individual tasks**: Never use `git add .` or `git commit -a` - explicitly add files
5. **Reference feature in commits**: Include "FEAT-XXX:" prefix in commit messages
6. **Temporary files**: Create temp files in `tmp/` within the repository (not `/tmp`), clean up after use, never commit them

### Testing Best Practices
1. **Test early and often**: Build and test after each significant change
2. **Use existing test data**: Leverage files in `testdata/` for integration tests
3. **Handle test data quirks**: Some test files have count mismatches (e.g., count="56" but 12 entries)
4. **Aim for high coverage**: Target 80%+ test coverage with `go test -covermode=set`
   - FEAT-002 (calls): Achieved 85.5% coverage
   - FEAT-003 (sms): Achieved 81.2% coverage
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

### Feature Completion
1. **Update task checkboxes**: Mark all tasks as `[x]` in feature document
2. **Set completion date**: Update Status section with completion date
3. **Move to completed**: `git mv features/active/FEAT-XXX.md features/completed/FEAT-XXX.md`
4. **Final commit**: Commit the completed feature document

### Common Issues and Solutions
- **Nix environment problems**: Use `nix-shell -p go_1_23 --run "command"` as fallback
- **Test data count mismatches**: These are expected in some files; use for validation testing
- **Legacy code conflicts**: Remove old implementations when starting fresh features
- **Import path issues**: Always use full module path `github.com/phillipgreen/mobilecombackup/pkg/...`
- **Date conversion**: Timestamps are in milliseconds, not seconds - divide by 1000 for Unix time
- **Empty vs null**: XML attributes with value "null" should be treated as empty/zero values
- **Unused variables in tests**: Remove unused test fixtures to avoid compilation errors
- **Year validation**: Test data often contains mixed years - adjust tests accordingly
- **MMS type field**: Use `msg_box` (1=received, 2=sent) not `m_type` for message direction

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

## Slash Commands for Feature Development

The repository includes custom slash commands in `.claude/commands/` to streamline feature development:

### Available Commands
- **`/implement-feature FEAT-XXX`**: Start implementing a feature following the established workflow
  - Moves feature from `ready/` to `active/`
  - Creates TodoWrite list from feature tasks
  - Ensures code compilation and test passing before task completion
  - Commits only modified files
  - Updates both feature document and `specification.md` on completion

- **`/ready-feature FEAT-XXX`**: Validate if a feature has enough detail for implementation
  - Reviews feature document completeness
  - Moves from `backlog/` to `ready/` if sufficiently detailed

- **`/review-feature FEAT-XXX`**: Review a feature specification
  - Provides feedback and suggestions for improvements
  - Asks clarifying questions about requirements

- **`/remember-anything-learned-this-session`**: Update CLAUDE.md with session learnings
  - Captures development workflow improvements
  - Documents new patterns and best practices

### Using Slash Commands
These commands are invoked by the user and provide structured prompts for common feature development tasks. They help ensure consistency across feature implementations and reduce the cognitive load of remembering all workflow steps.

## Session Learnings: FEAT-005 Implementation

### YAML Dependency Management
- **Adding Go dependencies**: Use `nix-shell -p go_1_23 --run "go get package"` when Nix develop environment isn't available
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
- **Living Documentation**: Update `features/specification.md` with each completed feature
- **API Documentation**: Include interface definitions and key features in specification
- **Cross-Referencing**: Maintain links between completed features and specification sections

## Next Steps
See `features/next_steps.md`

