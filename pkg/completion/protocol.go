// Package completion provides types and functions for the agent completion protocol.
package completion

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CompletionStatus represents the completion state of an agent task
type CompletionStatus int

const (
	StatusPending CompletionStatus = iota
	StatusInProgress
	StatusComplete
	StatusBlocked
)

func (s CompletionStatus) String() string {
	switch s {
	case StatusPending:
		return "PENDING"
	case StatusInProgress:
		return "IN_PROGRESS"
	case StatusComplete:
		return "COMPLETE"
	case StatusBlocked:
		return "BLOCKED"
	default:
		return "UNKNOWN"
	}
}

// WorkspaceState represents the current state of the git workspace
type WorkspaceState struct {
	IsClean               bool     `json:"is_clean"`
	ModifiedFiles         []string `json:"modified_files"`
	UntrackedFiles        []string `json:"untracked_files"`
	StagedFiles           []string `json:"staged_files"`
	TemporaryFiles        []string `json:"temporary_files"`
	HasUncommittedChanges bool     `json:"has_uncommitted_changes"`
}

// CompletionResult represents the result of attempting task completion
type CompletionResult struct {
	Status    CompletionStatus `json:"status"`
	Message   string           `json:"message"`
	Details   []string         `json:"details,omitempty"`
	Workspace *WorkspaceState  `json:"workspace,omitempty"`
	Duration  time.Duration    `json:"duration"`
}

// CompletionProtocol provides methods for enforcing agent completion requirements
type CompletionProtocol struct {
	VerifyCommands []string `json:"verify_commands"`
	TempDirs       []string `json:"temp_dirs"`
	LogActions     bool     `json:"log_actions"`
}

// NewCompletionProtocol creates a new completion protocol with default settings
func NewCompletionProtocol() *CompletionProtocol {
	return &CompletionProtocol{
		VerifyCommands: []string{
			"devbox run formatter",
			"devbox run tests",
			"devbox run linter",
			"devbox run build-cli",
		},
		TempDirs: []string{
			"tmp/",
			"/tmp/",
		},
		LogActions: true,
	}
}

// AnalyzeWorkspace examines the current workspace state
func (cp *CompletionProtocol) AnalyzeWorkspace() (*WorkspaceState, error) {
	ws := &WorkspaceState{}

	// Check git status
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to check git status: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		// No output means clean workspace
		ws.IsClean = true
		ws.HasUncommittedChanges = false
		return ws, nil
	}

	// Parse git status output
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}

		status := line[:2]
		filename := line[3:]

		switch {
		case status[0] == 'M' || status[1] == 'M':
			ws.ModifiedFiles = append(ws.ModifiedFiles, filename)
		case status[0] == 'A' || status[1] == 'A':
			ws.StagedFiles = append(ws.StagedFiles, filename)
		case status == "??":
			ws.UntrackedFiles = append(ws.UntrackedFiles, filename)
		default:
			// Other states (renamed, deleted, etc.)
			ws.ModifiedFiles = append(ws.ModifiedFiles, filename)
		}
	}

	ws.HasUncommittedChanges = len(ws.ModifiedFiles) > 0 || len(ws.UntrackedFiles) > 0 || len(ws.StagedFiles) > 0
	ws.IsClean = !ws.HasUncommittedChanges

	// Check for temporary files
	for _, tempDir := range cp.TempDirs {
		cp.scanTempDirectory(tempDir, ws)
	}

	// Update clean status based on temp files
	if len(ws.TemporaryFiles) > 0 {
		ws.IsClean = false
	}

	return ws, nil
}

// scanTempDirectory scans a temporary directory for files
func (cp *CompletionProtocol) scanTempDirectory(tempDir string, ws *WorkspaceState) {
	if _, err := os.Stat(tempDir); err != nil {
		return // Directory doesn't exist, skip
	}

	err := filepath.WalkDir(tempDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !d.IsDir() {
			ws.TemporaryFiles = append(ws.TemporaryFiles, path)
		}
		return nil
	})
	if err != nil && cp.LogActions {
		fmt.Printf("Warning: failed to scan temp directory %s: %v\n", tempDir, err)
	}
}

// RunVerification executes the verification commands if there are uncommitted changes
func (cp *CompletionProtocol) RunVerification() error {
	ws, err := cp.AnalyzeWorkspace()
	if err != nil {
		return fmt.Errorf("failed to analyze workspace: %w", err)
	}

	// Skip verification if workspace is already clean
	if ws.IsClean {
		if cp.LogActions {
			fmt.Println("Workspace is clean, skipping verification")
		}
		return nil
	}

	if cp.LogActions {
		fmt.Println("Running verification before cleanup...")
	}

	// Run each verification command
	for _, cmdStr := range cp.VerifyCommands {
		if err := cp.runVerificationCommand(cmdStr); err != nil {
			return fmt.Errorf("verification failed on command '%s': %w", cmdStr, err)
		}
	}

	if cp.LogActions {
		fmt.Println("Verification completed successfully")
	}
	return nil
}

// runVerificationCommand safely runs a verification command
func (cp *CompletionProtocol) runVerificationCommand(cmdStr string) error {
	if cp.LogActions {
		fmt.Printf("Running: %s\n", cmdStr)
	}

	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return fmt.Errorf("empty command string")
	}

	// #nosec G204 - These are predefined verification commands from configuration
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// CleanupTemporaryFiles removes temporary files created during task execution
func (cp *CompletionProtocol) CleanupTemporaryFiles() error {
	ws, err := cp.AnalyzeWorkspace()
	if err != nil {
		return fmt.Errorf("failed to analyze workspace: %w", err)
	}

	if len(ws.TemporaryFiles) == 0 {
		if cp.LogActions {
			fmt.Println("No temporary files to clean up")
		}
		return nil
	}

	if cp.LogActions {
		fmt.Printf("Cleaning up %d temporary files...\n", len(ws.TemporaryFiles))
	}

	for _, file := range ws.TemporaryFiles {
		if err := os.Remove(file); err != nil {
			if cp.LogActions {
				fmt.Printf("Warning: failed to remove temporary file %s: %v\n", file, err)
			}
		} else if cp.LogActions {
			fmt.Printf("Removed temporary file: %s\n", file)
		}
	}

	return nil
}

// CommitChanges commits all valid changes with an appropriate message
func (cp *CompletionProtocol) CommitChanges(issueID, taskDescription string) error {
	ws, err := cp.AnalyzeWorkspace()
	if err != nil {
		return fmt.Errorf("failed to analyze workspace: %w", err)
	}

	if !ws.HasUncommittedChanges {
		if cp.LogActions {
			fmt.Println("No changes to commit")
		}
		return nil
	}

	// Stage all modified and untracked files
	filesToStage := make([]string, 0)
	filesToStage = append(filesToStage, ws.ModifiedFiles...)
	filesToStage = append(filesToStage, ws.UntrackedFiles...)

	if len(filesToStage) == 0 {
		if cp.LogActions {
			fmt.Println("No files to stage")
		}
		return nil
	}

	// Stage files
	if cp.LogActions {
		fmt.Printf("Staging %d files...\n", len(filesToStage))
	}

	for _, file := range filesToStage {
		// #nosec G204 - Git commands with validated filenames from git status
		cmd := exec.Command("git", "add", file)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to stage file %s: %w", file, err)
		}
	}

	// Create commit message
	commitMsg := fmt.Sprintf("%s: %s\n\n🤖 Generated with [Claude Code](https://claude.ai/code)\n\n"+
		"Co-Authored-By: Claude <noreply@anthropic.com>", issueID, taskDescription)

	// Commit changes
	// #nosec G204 - Git commit with sanitized message
	cmd := exec.Command("git", "commit", "-m", commitMsg)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	if cp.LogActions {
		fmt.Printf("Successfully committed changes for %s\n", issueID)
	}

	return nil
}

// EnsureCleanCompletion is the main method that enforces the completion protocol
func (cp *CompletionProtocol) EnsureCleanCompletion(issueID, taskDescription string) *CompletionResult {
	start := time.Now()
	result := &CompletionResult{
		Status:   StatusInProgress,
		Duration: 0,
	}

	// Step 1: Analyze current workspace
	ws, err := cp.AnalyzeWorkspace()
	if err != nil {
		return cp.createBlockedResult("Failed to analyze workspace", err, start)
	}

	result.Workspace = ws

	// Step 2: If already clean, report complete
	if ws.IsClean {
		result.Status = StatusComplete
		result.Message = "Workspace is already clean"
		result.Duration = time.Since(start)
		return result
	}

	// Step 3: Run verification if there are changes
	if ws.HasUncommittedChanges {
		if err := cp.RunVerification(); err != nil {
			result := cp.createBlockedResult("Verification failed", err, start)
			result.Details = append(result.Details, "Fix verification failures before completion")
			return result
		}
	}

	// Step 4: Commit changes
	if ws.HasUncommittedChanges {
		if err := cp.CommitChanges(issueID, taskDescription); err != nil {
			result := cp.createBlockedResult("Failed to commit changes", err, start)
			result.Details = append(result.Details, "Resolve commit conflicts manually")
			return result
		}
	}

	// Step 5: Clean up temporary files
	if err := cp.CleanupTemporaryFiles(); err != nil {
		result := cp.createBlockedResult("Failed to clean temporary files", err, start)
		result.Details = append(result.Details, "Manual cleanup required")
		return result
	}

	// Step 6: Final verification that workspace is clean
	return cp.performFinalVerification(start)
}

// createBlockedResult creates a blocked completion result with standardized format
func (cp *CompletionProtocol) createBlockedResult(message string, err error, start time.Time) *CompletionResult {
	return &CompletionResult{
		Status:   StatusBlocked,
		Message:  fmt.Sprintf("%s: %v", message, err),
		Duration: time.Since(start),
	}
}

// performFinalVerification performs the final workspace verification
func (cp *CompletionProtocol) performFinalVerification(start time.Time) *CompletionResult {
	finalWs, err := cp.AnalyzeWorkspace()
	if err != nil {
		return cp.createBlockedResult("Failed final workspace check", err, start)
	}

	if !finalWs.IsClean {
		return &CompletionResult{
			Status:  StatusBlocked,
			Message: "Workspace still not clean after cleanup",
			Details: []string{
				fmt.Sprintf("Modified files: %v", finalWs.ModifiedFiles),
				fmt.Sprintf("Untracked files: %v", finalWs.UntrackedFiles),
				fmt.Sprintf("Temporary files: %v", finalWs.TemporaryFiles),
			},
			Duration: time.Since(start),
		}
	}

	// Success!
	return &CompletionResult{
		Status:    StatusComplete,
		Message:   "Task completed successfully with clean workspace",
		Workspace: finalWs,
		Duration:  time.Since(start),
	}
}
