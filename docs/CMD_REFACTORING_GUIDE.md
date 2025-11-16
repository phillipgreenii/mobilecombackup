# CMD Package Refactoring Guide

## Current Testability Issues

### 1. Direct `os.Exit()` Calls
**Problem:** Functions call `os.Exit()` directly, terminating the test process.

```go
// ❌ CURRENT - Untestable
func handleImportError(err error) {
    if !quiet {
        PrintError("Import failed: %v", err)
    }
    os.Exit(2)  // Kills test process!
}

func createImporter(options *ImportOptions) (*Importer, error) {
    imp, err := importer.NewImporter(options, GetLogger())
    if err != nil {
        if !testing.Testing() {  // Code smell!
            os.Exit(2)
        }
        return nil, err
    }
    return imp, nil
}
```

### 2. Direct Filesystem Operations
**Problem:** Uses `os.ReadFile()` instead of `afero.Fs` interface.

```go
// ❌ CURRENT - Requires real filesystem
func readRepositoryMetadata(repoPath string, info *RepositoryInfo) error {
    markerPath := filepath.Join(repoPath, ".mobilecombackup.yaml")
    data, err := os.ReadFile(markerPath)  // Direct OS call!
    // ...
}

func gatherRepositoryInfo(repoPath string) (*RepositoryInfo, error) {
    // ...
    attachmentReader := attachments.NewAttachmentManager(repoPath, afero.NewOsFs())  // Hardcoded!
    // ...
}
```

### 3. Mixed Concerns
**Problem:** Business logic mixed with I/O, presentation, and error handling.

```go
// ❌ CURRENT - Does too much
func gatherRepositoryInfo(repoPath string) (*RepositoryInfo, error) {
    // 1. Creates objects
    callsReader := calls.NewXMLCallsReader(repoPath)

    // 2. Does I/O
    if err := readRepositoryMetadata(repoPath, info); err != nil {
        // ...
    }

    // 3. Business logic
    if err := gatherCallsStats(callsReader, info); err != nil {
        info.Errors["calls"] = 1
    }

    // 4. Validation
    info.ValidationOK = len(info.Errors) == 0

    return info, nil
}
```

### 4. Global State
**Problem:** Functions depend on package-level variables.

```go
var (
    quiet       bool
    verbose     bool
    importJSON  bool
    // ... many more globals
)

func displayResults(summary *ImportSummary) {
    effectiveQuiet := quiet || importJSON  // Reads globals!
    // ...
}
```

---

## Refactoring Strategy

### Pattern 1: Dependency Injection with Context Objects

**Create a testable context that holds dependencies:**

```go
// ✅ REFACTORED - Testable
type InfoContext struct {
    RepoPath string
    Fs       afero.Fs
    Output   io.Writer

    // Readers (injected for testing)
    CallsReader      calls.Reader
    SMSReader        sms.Reader
    AttachmentReader attachments.Manager
    ContactsReader   contacts.Manager
}

// Constructor for production use
func NewInfoContext(repoPath string) *InfoContext {
    fs := afero.NewOsFs()
    return &InfoContext{
        RepoPath:         repoPath,
        Fs:               fs,
        Output:           os.Stdout,
        CallsReader:      calls.NewXMLCallsReader(repoPath),
        SMSReader:        sms.NewXMLSMSReader(repoPath),
        AttachmentReader: attachments.NewAttachmentManager(repoPath, fs),
        ContactsReader:   contacts.NewContactsManager(repoPath),
    }
}

// Testable version - inject mocks
func NewInfoContextWithDeps(
    repoPath string,
    fs afero.Fs,
    callsReader calls.Reader,
    smsReader sms.Reader,
) *InfoContext {
    return &InfoContext{
        RepoPath:    repoPath,
        Fs:          fs,
        CallsReader: callsReader,
        SMSReader:   smsReader,
        Output:      io.Discard,
    }
}
```

**Refactor functions to use context:**

```go
// ✅ REFACTORED - Testable with mocks
func (ctx *InfoContext) GatherRepositoryInfo() (*RepositoryInfo, error) {
    info := &RepositoryInfo{
        Calls:      make(map[string]YearInfo),
        SMS:        make(map[string]MessageInfo),
        Rejections: make(map[string]int),
        Errors:     make(map[string]int),
    }

    // Read metadata using injected filesystem
    if err := ctx.readRepositoryMetadata(info); err != nil {
        if !os.IsNotExist(err) {
            return nil, fmt.Errorf("failed to read metadata: %w", err)
        }
    }

    // Use injected readers (can be mocked in tests!)
    if err := gatherCallsStats(ctx.CallsReader, info); err != nil {
        info.Errors["calls"] = 1
    }

    if err := gatherSMSStats(ctx.SMSReader, info); err != nil {
        info.Errors["sms"] = 1
    }

    info.ValidationOK = len(info.Errors) == 0
    return info, nil
}

func (ctx *InfoContext) readRepositoryMetadata(info *RepositoryInfo) error {
    markerPath := filepath.Join(ctx.RepoPath, ".mobilecombackup.yaml")

    // Uses injected filesystem - can be memory FS in tests!
    data, err := afero.ReadFile(ctx.Fs, markerPath)
    if err != nil {
        return err
    }

    var marker InfoMarkerFileContent
    if err := yaml.Unmarshal(data, &marker); err != nil {
        return fmt.Errorf("failed to parse marker file: %w", err)
    }

    info.Version = marker.RepositoryStructureVersion
    if marker.CreatedAt != "" {
        if createdAt, err := time.Parse(time.RFC3339, marker.CreatedAt); err == nil {
            info.CreatedAt = createdAt
        }
    }

    return nil
}
```

### Pattern 2: Exit Handler Interface

**Problem:** Direct `os.Exit()` calls can't be tested.

```go
// ✅ REFACTORED - Testable exit handling
type ExitHandler interface {
    Exit(code int)
}

type OSExitHandler struct{}

func (h OSExitHandler) Exit(code int) {
    os.Exit(code)
}

type TestExitHandler struct {
    Code int
    Called bool
}

func (h *TestExitHandler) Exit(code int) {
    h.Code = code
    h.Called = true
}

// Command context with exit handler
type ImportContext struct {
    Options     *importer.ImportOptions
    Quiet       bool
    JSON        bool
    ExitHandler ExitHandler
    Output      io.Writer
}

// ✅ Now testable!
func (ctx *ImportContext) HandleImportError(err error) {
    if !ctx.Quiet {
        fmt.Fprintf(ctx.Output, "Import failed: %v\n", err)
    }
    ctx.ExitHandler.Exit(2)  // Can capture in tests!
}

func (ctx *ImportContext) HandleExitCode(summary *importer.ImportSummary) {
    totalRejected := summary.Calls.Total.Rejected + summary.SMS.Total.Rejected
    if totalRejected > 0 {
        ctx.ExitHandler.Exit(1)
    }
}
```

**Test example:**

```go
func TestHandleImportError(t *testing.T) {
    exitHandler := &TestExitHandler{}
    var buf bytes.Buffer

    ctx := &ImportContext{
        Quiet:       false,
        ExitHandler: exitHandler,
        Output:      &buf,
    }

    ctx.HandleImportError(errors.New("test error"))

    if !exitHandler.Called {
        t.Error("expected exit handler to be called")
    }
    if exitHandler.Code != 2 {
        t.Errorf("expected exit code 2, got %d", exitHandler.Code)
    }
    if !strings.Contains(buf.String(), "test error") {
        t.Error("expected error message in output")
    }
}
```

### Pattern 3: Separate Pure Logic from I/O

**Extract pure business logic:**

```go
// ✅ Pure function - 100% testable
func CalculateRepositoryHealth(info *RepositoryInfo) HealthStatus {
    if len(info.Errors) > 0 {
        return HealthUnhealthy
    }
    if info.Calls.Total == 0 && info.SMS.Total == 0 {
        return HealthEmpty
    }
    return HealthOK
}

// ✅ Pure function - easy to test edge cases
func DetermineExitCode(summary *importer.ImportSummary, allowRejects bool) int {
    totalRejected := summary.Calls.Total.Rejected + summary.SMS.Total.Rejected
    if totalRejected > 0 && !allowRejects {
        return 1
    }
    return 0
}

// ✅ Pure function - presentation logic only
func FormatImportSummary(summary *importer.ImportSummary, dryRun bool) string {
    var buf bytes.Buffer
    // ... formatting logic
    return buf.String()
}
```

### Pattern 4: Builder Pattern for Commands

**Make command configuration explicit and testable:**

```go
// ✅ REFACTORED - Builder pattern
type ImportCommandBuilder struct {
    repoPath       string
    paths          []string
    fs             afero.Fs
    output         io.Writer
    exitHandler    ExitHandler
    quiet          bool
    verbose        bool
    json           bool
    dryRun         bool
    filter         string
    maxXMLSize     string
    maxMessageSize string
}

func NewImportCommand() *ImportCommandBuilder {
    return &ImportCommandBuilder{
        fs:             afero.NewOsFs(),
        output:         os.Stdout,
        exitHandler:    OSExitHandler{},
        maxXMLSize:     "500MB",
        maxMessageSize: "10MB",
    }
}

// Chainable configuration
func (b *ImportCommandBuilder) WithRepoPath(path string) *ImportCommandBuilder {
    b.repoPath = path
    return b
}

func (b *ImportCommandBuilder) WithFilesystem(fs afero.Fs) *ImportCommandBuilder {
    b.fs = fs
    return b
}

func (b *ImportCommandBuilder) WithOutput(w io.Writer) *ImportCommandBuilder {
    b.output = w
    return b
}

func (b *ImportCommandBuilder) WithExitHandler(h ExitHandler) *ImportCommandBuilder {
    b.exitHandler = h
    return b
}

// Build creates the executable command
func (b *ImportCommandBuilder) Build() (*ImportCommand, error) {
    if b.repoPath == "" {
        return nil, fmt.Errorf("repository path required")
    }

    options, err := b.createImportOptions()
    if err != nil {
        return nil, err
    }

    return &ImportCommand{
        options:     options,
        output:      b.output,
        exitHandler: b.exitHandler,
        quiet:       b.quiet,
        json:        b.json,
        dryRun:      b.dryRun,
    }, nil
}

// Testable command execution
type ImportCommand struct {
    options     *importer.ImportOptions
    output      io.Writer
    exitHandler ExitHandler
    quiet       bool
    json        bool
    dryRun      bool
}

func (c *ImportCommand) Execute(ctx context.Context) error {
    // All dependencies injected - fully testable!
    imp, err := importer.NewImporter(c.options, logging.NewLogger())
    if err != nil {
        if !c.quiet {
            fmt.Fprintf(c.output, "Failed to initialize: %v\n", err)
        }
        c.exitHandler.Exit(2)
        return err
    }

    summary, err := imp.Import(ctx)
    if err != nil {
        if !c.quiet {
            fmt.Fprintf(c.output, "Import failed: %v\n", err)
        }
        c.exitHandler.Exit(2)
        return err
    }

    c.displayResults(summary)

    exitCode := DetermineExitCode(summary, false)
    if exitCode != 0 {
        c.exitHandler.Exit(exitCode)
    }

    return nil
}
```

**Test example:**

```go
func TestImportCommand_Execute(t *testing.T) {
    fs := afero.NewMemMapFs()
    exitHandler := &TestExitHandler{}
    var buf bytes.Buffer

    // Create test repository
    setupTestRepo(t, fs, "/test/repo")

    cmd, err := NewImportCommand().
        WithRepoPath("/test/repo").
        WithFilesystem(fs).
        WithOutput(&buf).
        WithExitHandler(exitHandler).
        WithQuiet(true).
        Build()

    if err != nil {
        t.Fatalf("failed to build command: %v", err)
    }

    err = cmd.Execute(context.Background())

    // Now we can test exit codes, output, etc.!
    if exitHandler.Called {
        t.Errorf("unexpected exit call with code %d", exitHandler.Code)
    }
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
}
```

---

## Migration Path

### Phase 1: Extract Pure Functions (Low Risk)
1. ✅ Move pure logic to separate functions
2. ✅ Add comprehensive unit tests
3. ✅ No API changes needed

**Files to refactor:**
- Extract `DetermineExitCode()` from `handleExitCode()`
- Extract `FormatImportSummary()` from `displaySummary()`
- Extract `CalculateRepositoryHealth()` from `gatherRepositoryInfo()`

### Phase 2: Add Interfaces (Medium Risk)
1. ✅ Create `ExitHandler` interface
2. ✅ Create context objects with injected dependencies
3. ✅ Keep existing functions as wrappers (backward compatible)

**Files to refactor:**
- Create `InfoContext` for info.go
- Create `ImportContext` for import.go
- Create `ExitHandler` interface

### Phase 3: Refactor Cobra Integration (Low Risk)
1. ✅ Update Cobra command handlers to use new context objects
2. ✅ Remove old wrapper functions
3. ✅ Update tests

**Example:**

```go
// ✅ Updated Cobra handler
var infoCmd = &cobra.Command{
    Use:   "info [repository]",
    Short: "Display repository information",
    RunE: func(cmd *cobra.Command, args []string) error {
        repoPath := resolveRepoPath(args)

        // Create context with real dependencies for production
        ctx := NewInfoContext(repoPath)

        info, err := ctx.GatherRepositoryInfo()
        if err != nil {
            return err
        }

        // Display using context output writer
        return ctx.DisplayInfo(info)
    },
}
```

### Phase 4: Comprehensive Testing (High Value)
1. ✅ Add unit tests for all pure functions
2. ✅ Add integration tests with memory filesystem
3. ✅ Achieve >80% coverage

---

## Benefits

### Before Refactoring
- ❌ 51.4% coverage
- ❌ Can't test `os.Exit()` paths
- ❌ Requires real filesystem for all tests
- ❌ Hard to test error scenarios
- ❌ Global state makes tests interfere

### After Refactoring
- ✅ >80% coverage achievable
- ✅ Exit handling fully testable
- ✅ Tests run in memory (faster)
- ✅ Easy to test error paths with mocks
- ✅ Tests isolated and parallel

---

## Quick Wins (Implement First)

### 1. Extract Pure Functions (1-2 hours)
```go
// Pure logic - test immediately
func DetermineExitCode(summary *ImportSummary, allowRejects bool) int
func FormatImportSummary(summary *ImportSummary, dryRun bool) string
func CalculateTotalRejections(summary *ImportSummary) int
func ShouldSkipFile(path string) bool
```

### 2. Add ExitHandler Interface (2-3 hours)
```go
type ExitHandler interface { Exit(code int) }
type TestExitHandler struct { Code int; Called bool }
```

### 3. Create Context Objects (4-6 hours)
```go
type InfoContext struct { /* injectable dependencies */ }
type ImportContext struct { /* injectable dependencies */ }
```

---

## Example: Complete Refactoring of `gatherRepositoryInfo`

### Before (Untestable)
```go
func gatherRepositoryInfo(repoPath string) (*RepositoryInfo, error) {
    // Hardcoded dependencies
    callsReader := calls.NewXMLCallsReader(repoPath)
    smsReader := sms.NewXMLSMSReader(repoPath)
    attachmentReader := attachments.NewAttachmentManager(repoPath, afero.NewOsFs())

    // Direct OS call
    data, err := os.ReadFile(filepath.Join(repoPath, ".mobilecombackup.yaml"))
    // ...
}
```

### After (Fully Testable)
```go
// Context with injected dependencies
type InfoContext struct {
    RepoPath         string
    Fs               afero.Fs
    CallsReader      calls.Reader
    SMSReader        sms.Reader
    AttachmentReader attachments.Manager
}

func (ctx *InfoContext) GatherRepositoryInfo() (*RepositoryInfo, error) {
    info := &RepositoryInfo{
        Calls: make(map[string]YearInfo),
        SMS:   make(map[string]MessageInfo),
    }

    // Use injected filesystem
    if err := ctx.readMetadata(info); err != nil {
        if !os.IsNotExist(err) {
            return nil, err
        }
    }

    // Use injected readers (mockable!)
    if err := gatherCallsStats(ctx.CallsReader, info); err != nil {
        info.Errors["calls"] = 1
    }

    return info, nil
}

// Test with mocks
func TestGatherRepositoryInfo(t *testing.T) {
    fs := afero.NewMemMapFs()
    mockCalls := &MockCallsReader{/* test data */}
    mockSMS := &MockSMSReader{/* test data */}

    ctx := &InfoContext{
        RepoPath:    "/test",
        Fs:          fs,
        CallsReader: mockCalls,
        SMSReader:   mockSMS,
    }

    info, err := ctx.GatherRepositoryInfo()

    // Now we can test everything!
    assert.NoError(t, err)
    assert.Equal(t, 100, info.Calls["2023"].Count)
}
```

---

## Conclusion

The refactoring focuses on **Dependency Injection** and **Separation of Concerns**:

1. **Inject** filesystem, logger, exit handler
2. **Separate** pure logic from I/O
3. **Use** interfaces for all external dependencies
4. **Avoid** global state and side effects
5. **Return** errors instead of calling `os.Exit()`

This makes the code:
- **100% unit testable** (with mocks)
- **Fast** (in-memory tests)
- **Reliable** (isolated tests)
- **Maintainable** (clear dependencies)

The migration can be done incrementally without breaking existing functionality.
