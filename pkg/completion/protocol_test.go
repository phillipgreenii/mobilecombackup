package completion

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestNewCompletionProtocol(t *testing.T) {
	cp := NewCompletionProtocol()

	if cp == nil {
		t.Fatal("NewCompletionProtocol returned nil")
	}

	if len(cp.VerifyCommands) == 0 {
		t.Error("Expected verify commands to be set")
	}

	if len(cp.TempDirs) == 0 {
		t.Error("Expected temp dirs to be set")
	}

	if !cp.LogActions {
		t.Error("Expected LogActions to be true by default")
	}
}

func TestCompletionStatus_String(t *testing.T) {
	tests := map[CompletionStatus]string{
		StatusPending:         "PENDING",
		StatusInProgress:      "IN_PROGRESS",
		StatusComplete:        "COMPLETE",
		StatusBlocked:         "BLOCKED",
		CompletionStatus(999): "UNKNOWN",
	}

	for status, expected := range tests {
		if got := status.String(); got != expected {
			t.Errorf("Status %d: got %q, want %q", status, got, expected)
		}
	}
}

func TestCompletionProtocol_AnalyzeWorkspace_CleanRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary git repository
	tempDir := t.TempDir()
	setupCleanGitRepo(t, tempDir)

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	cp := NewCompletionProtocol()
	cp.LogActions = false

	ws, err := cp.AnalyzeWorkspace()
	if err != nil {
		t.Fatalf("AnalyzeWorkspace failed: %v", err)
	}

	if !ws.IsClean {
		t.Error("Expected clean workspace")
	}

	if ws.HasUncommittedChanges {
		t.Error("Expected no uncommitted changes")
	}

	if len(ws.ModifiedFiles) > 0 {
		t.Errorf("Expected no modified files, got %v", ws.ModifiedFiles)
	}

	if len(ws.UntrackedFiles) > 0 {
		t.Errorf("Expected no untracked files, got %v", ws.UntrackedFiles)
	}
}

func TestCompletionProtocol_AnalyzeWorkspace_WithChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary git repository
	tempDir := t.TempDir()
	setupCleanGitRepo(t, tempDir)

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	// Create some changes
	if err := os.WriteFile("test.txt", []byte("modified content"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile("new.txt", []byte("new file"), 0644); err != nil {
		t.Fatal(err)
	}

	cp := NewCompletionProtocol()
	cp.LogActions = false

	ws, err := cp.AnalyzeWorkspace()
	if err != nil {
		t.Fatalf("AnalyzeWorkspace failed: %v", err)
	}

	if ws.IsClean {
		t.Error("Expected dirty workspace")
	}

	if !ws.HasUncommittedChanges {
		t.Error("Expected uncommitted changes")
	}

	if len(ws.ModifiedFiles) == 0 {
		t.Error("Expected modified files")
	}

	if len(ws.UntrackedFiles) == 0 {
		t.Error("Expected untracked files")
	}
}

func TestCompletionProtocol_AnalyzeWorkspace_WithTempFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary git repository
	tempDir := t.TempDir()
	setupCleanGitRepo(t, tempDir)

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	// Create temp directory and file
	tempSubDir := filepath.Join(tempDir, "tmp")
	if err := os.MkdirAll(tempSubDir, 0755); err != nil {
		t.Fatal(err)
	}

	tempFile := filepath.Join(tempSubDir, "temp.txt")
	if err := os.WriteFile(tempFile, []byte("temporary content"), 0644); err != nil {
		t.Fatal(err)
	}

	cp := NewCompletionProtocol()
	cp.LogActions = false

	ws, err := cp.AnalyzeWorkspace()
	if err != nil {
		t.Fatalf("AnalyzeWorkspace failed: %v", err)
	}

	if ws.IsClean {
		t.Error("Expected dirty workspace due to temp files")
	}

	if len(ws.TemporaryFiles) == 0 {
		t.Error("Expected temporary files")
	}

	found := false
	for _, f := range ws.TemporaryFiles {
		if f == tempFile {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected to find temp file %s in list %v", tempFile, ws.TemporaryFiles)
	}
}

func TestCompletionProtocol_CleanupTemporaryFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary git repository
	tempDir := t.TempDir()
	setupCleanGitRepo(t, tempDir)

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	// Create temp directory and file
	tempSubDir := filepath.Join(tempDir, "tmp")
	if err := os.MkdirAll(tempSubDir, 0755); err != nil {
		t.Fatal(err)
	}

	tempFile := filepath.Join(tempSubDir, "temp.txt")
	if err := os.WriteFile(tempFile, []byte("temporary content"), 0644); err != nil {
		t.Fatal(err)
	}

	cp := NewCompletionProtocol()
	cp.LogActions = false

	// Verify temp file exists
	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Fatal("Temp file should exist before cleanup")
	}

	// Run cleanup
	if err := cp.CleanupTemporaryFiles(); err != nil {
		t.Fatalf("CleanupTemporaryFiles failed: %v", err)
	}

	// Verify temp file is gone
	if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
		t.Error("Temp file should be removed after cleanup")
	}
}

func TestCompletionProtocol_EnsureCleanCompletion_AlreadyClean(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary git repository
	tempDir := t.TempDir()
	setupCleanGitRepo(t, tempDir)

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	cp := NewCompletionProtocol()
	cp.LogActions = false

	result := cp.EnsureCleanCompletion("TEST-001", "test task")

	if result.Status != StatusComplete {
		t.Errorf("Expected COMPLETE status, got %s", result.Status)
	}

	if result.Message != "Workspace is already clean" {
		t.Errorf("Unexpected message: %s", result.Message)
	}

	if result.Duration <= 0 {
		t.Error("Expected positive duration")
	}
}

func TestCompletionResult_Fields(t *testing.T) {
	result := &CompletionResult{
		Status:   StatusComplete,
		Message:  "Test message",
		Details:  []string{"detail1", "detail2"},
		Duration: time.Second,
	}

	if result.Status != StatusComplete {
		t.Error("Status not set correctly")
	}

	if result.Message != "Test message" {
		t.Error("Message not set correctly")
	}

	if len(result.Details) != 2 {
		t.Error("Details not set correctly")
	}

	if result.Duration != time.Second {
		t.Error("Duration not set correctly")
	}
}

// Helper function to set up a clean git repository
func setupCleanGitRepo(t *testing.T, dir string) {
	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to config git email: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to config git name: %v", err)
	}

	// Create initial file and commit
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add test file: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
}

func TestNewWorkspaceCleanup(t *testing.T) {
	t.Parallel()

	verifyCommands := []string{"devbox run formatter", "devbox run test"}
	tempDirs := []string{"tmp", "temp"}

	wc := NewWorkspaceCleanup(verifyCommands, tempDirs)

	if wc == nil {
		t.Fatal("NewWorkspaceCleanup returned nil")
	}

	if wc.CompletionProtocol == nil {
		t.Fatal("CompletionProtocol is nil")
	}

	if !wc.SafetyChecks {
		t.Error("Expected SafetyChecks to be true by default")
	}

	if wc.DryRun {
		t.Error("Expected DryRun to be false by default")
	}
}

func TestWorkspaceCleanup_categorizeFile(t *testing.T) {
	t.Parallel()

	wc := NewWorkspaceCleanup([]string{}, []string{})

	tests := []struct {
		filename string
		expected ChangeCategory
	}{
		// Test files
		{"pkg/test/example_test.go", CategoryTest},
		{"main_test.go", CategoryTest},
		{"testdata/example.xml", CategoryTest},
		{"pkg/testdata/calls.xml", CategoryTest},

		// Documentation
		{"README.md", CategoryDoc},
		{"CHANGELOG.md", CategoryDoc},
		{"docs/INSTALLATION.md", CategoryDoc},
		{"docs/development/setup.md", CategoryDoc},

		// Configuration files
		{"devbox.json", CategoryConfig},
		{"devbox.lock", CategoryConfig},
		{".gitignore", CategoryConfig},
		{".golangci.yml", CategoryConfig},
		{"config.yaml", CategoryConfig},
		{"settings.toml", CategoryConfig},
		{"Dockerfile", CategoryConfig},

		// Code files
		{"main.go", CategoryCode},
		{"pkg/calls/reader.go", CategoryCode},
		{"cmd/app/main.js", CategoryCode},
		{"src/utils.ts", CategoryCode},
		{"lib/helper.py", CategoryCode},
		{"include/header.h", CategoryCode},

		// Other files
		{"LICENSE", CategoryOther},
		{"data.txt", CategoryOther},
		{"image.png", CategoryOther},
		{"script.sh", CategoryOther},
	}

	for _, test := range tests {
		t.Run(test.filename, func(t *testing.T) {
			result := wc.categorizeFile(test.filename)
			if result != test.expected {
				t.Errorf("categorizeFile(%q) = %v, want %v", test.filename, result, test.expected)
			}
		})
	}
}

func TestWorkspaceCleanup_determineVerificationNeeds(t *testing.T) {
	t.Parallel()

	wc := NewWorkspaceCleanup([]string{}, []string{})

	tests := []struct {
		name     string
		changes  []CategorizedChange
		expected []ChangeCategory
	}{
		{
			name: "code changes need verification",
			changes: []CategorizedChange{
				{Filename: "main.go", Category: CategoryCode, Status: "M"},
			},
			expected: []ChangeCategory{CategoryCode},
		},
		{
			name: "test changes need verification",
			changes: []CategorizedChange{
				{Filename: "main_test.go", Category: CategoryTest, Status: "M"},
			},
			expected: []ChangeCategory{CategoryTest},
		},
		{
			name: "config changes need verification",
			changes: []CategorizedChange{
				{Filename: "devbox.json", Category: CategoryConfig, Status: "M"},
			},
			expected: []ChangeCategory{CategoryConfig},
		},
		{
			name: "doc changes don't need verification",
			changes: []CategorizedChange{
				{Filename: "README.md", Category: CategoryDoc, Status: "M"},
			},
			expected: []ChangeCategory{},
		},
		{
			name: "mixed changes",
			changes: []CategorizedChange{
				{Filename: "main.go", Category: CategoryCode, Status: "M"},
				{Filename: "README.md", Category: CategoryDoc, Status: "M"},
				{Filename: "main_test.go", Category: CategoryTest, Status: "M"},
			},
			expected: []ChangeCategory{CategoryCode, CategoryTest},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := wc.determineVerificationNeeds(test.changes)

			// Sort both slices for comparison
			if len(result) != len(test.expected) {
				t.Errorf("determineVerificationNeeds() returned %d categories, want %d", len(result), len(test.expected))
				return
			}

			// Create maps for easy comparison
			resultMap := make(map[ChangeCategory]bool)
			expectedMap := make(map[ChangeCategory]bool)

			for _, cat := range result {
				resultMap[cat] = true
			}
			for _, cat := range test.expected {
				expectedMap[cat] = true
			}

			for cat := range expectedMap {
				if !resultMap[cat] {
					t.Errorf("Expected category %v not found in result", cat)
				}
			}

			for cat := range resultMap {
				if !expectedMap[cat] {
					t.Errorf("Unexpected category %v found in result", cat)
				}
			}
		})
	}
}

func TestWorkspaceCleanup_canAutoCommit(t *testing.T) {
	t.Parallel()

	wc := NewWorkspaceCleanup([]string{}, []string{})

	tests := []struct {
		name     string
		state    *EnhancedWorkspaceState
		expected bool
	}{
		{
			name: "clean git state with no verification needed",
			state: &EnhancedWorkspaceState{
				GitState:          GitState{IsClean: true},
				NeedsVerification: []ChangeCategory{},
			},
			expected: true,
		},
		{
			name: "dirty git state",
			state: &EnhancedWorkspaceState{
				GitState:          GitState{IsClean: false, IsMerging: true},
				NeedsVerification: []ChangeCategory{},
			},
			expected: false,
		},
		{
			name: "conflict files present",
			state: &EnhancedWorkspaceState{
				GitState: GitState{
					IsClean:       false,
					ConflictFiles: []string{"main.go", "test.go"},
				},
				NeedsVerification: []ChangeCategory{},
			},
			expected: false,
		},
		{
			name: "clean state with verification needed",
			state: &EnhancedWorkspaceState{
				GitState:          GitState{IsClean: true},
				NeedsVerification: []ChangeCategory{CategoryCode},
			},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := wc.canAutoCommit(test.state)
			if result != test.expected {
				t.Errorf("canAutoCommit() = %v, want %v", result, test.expected)
			}
		})
	}
}

func TestWorkspaceCleanup_generateCommitMessage(t *testing.T) {
	t.Parallel()

	wc := NewWorkspaceCleanup([]string{}, []string{})

	tests := []struct {
		name     string
		changes  []CategorizedChange
		expected string
	}{
		{
			name:     "no changes",
			changes:  []CategorizedChange{},
			expected: "workspace cleanup: misc changes",
		},
		{
			name: "single code file",
			changes: []CategorizedChange{
				{Filename: "main.go", Category: CategoryCode, Status: "M"},
			},
			expected: "workspace cleanup: code changes (1 files)",
		},
		{
			name: "single doc file",
			changes: []CategorizedChange{
				{Filename: "README.md", Category: CategoryDoc, Status: "M"},
			},
			expected: "workspace cleanup: documentation updates (1 files)",
		},
		{
			name: "multiple categories",
			changes: []CategorizedChange{
				{Filename: "main.go", Category: CategoryCode, Status: "M"},
				{Filename: "main_test.go", Category: CategoryTest, Status: "M"},
				{Filename: "README.md", Category: CategoryDoc, Status: "M"},
			},
			expected: "workspace cleanup: code changes (1 files), test updates (1 files), documentation updates (1 files)",
		},
		{
			name: "multiple files same category",
			changes: []CategorizedChange{
				{Filename: "main.go", Category: CategoryCode, Status: "M"},
				{Filename: "utils.go", Category: CategoryCode, Status: "A"},
				{Filename: "helper.go", Category: CategoryCode, Status: "M"},
			},
			expected: "workspace cleanup: code changes (3 files)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := wc.generateCommitMessage(test.changes)
			if result != test.expected {
				t.Errorf("generateCommitMessage() = %q, want %q", result, test.expected)
			}
		})
	}
}

func TestWorkspaceCleanup_buildChangesSummary(t *testing.T) {
	t.Parallel()

	wc := NewWorkspaceCleanup([]string{}, []string{})

	changes := []CategorizedChange{
		{Filename: "main.go", Category: CategoryCode, Status: "M"},
		{Filename: "utils.go", Category: CategoryCode, Status: "A"},
		{Filename: "README.md", Category: CategoryDoc, Status: "M"},
		{Filename: "main_test.go", Category: CategoryTest, Status: "M"},
	}

	result := wc.buildChangesSummary(changes)

	// Check that we have entries for each category
	hasCode := false
	hasDoc := false
	hasTest := false

	for _, line := range result {
		if line == "code changes:" {
			hasCode = true
		}
		if line == "doc changes:" {
			hasDoc = true
		}
		if line == "test changes:" {
			hasTest = true
		}
	}

	if !hasCode {
		t.Error("Expected 'code changes:' in summary")
	}
	if !hasDoc {
		t.Error("Expected 'doc changes:' in summary")
	}
	if !hasTest {
		t.Error("Expected 'test changes:' in summary")
	}
}

func TestChangeCategory_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		category ChangeCategory
		expected string
	}{
		{CategoryCode, "code"},
		{CategoryTest, "test"},
		{CategoryDoc, "doc"},
		{CategoryConfig, "config"},
		{CategoryOther, "other"},
	}

	for _, test := range tests {
		t.Run(string(test.category), func(t *testing.T) {
			result := test.category.String()
			if result != test.expected {
				t.Errorf("String() = %q, want %q", result, test.expected)
			}
		})
	}
}
