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
