package cmd_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/validation"
)

// TestValidateCommandIntegration tests the validate command via the CLI
func TestValidateCommandIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Build test binary
	testBin := filepath.Join(t.TempDir(), "mobilecombackup-test")
	buildCmd := exec.Command("go", "build", "-o", testBin, "../../../cmd/mobilecombackup") // #nosec G204
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build test binary: %v\nOutput: %s", err, output)
	}

	tests := []struct {
		name         string
		args         []string
		envVars      map[string]string
		setup        func(t *testing.T) string
		wantExitCode int
		wantOutput   []string
		notWant      []string
	}{
		{
			name: "repository with violations",
			args: []string{"validate"},
			setup: func(t *testing.T) string {
				dir := createValidRepository(t)
				return dir
			},
			wantExitCode: 1,
			wantOutput:   []string{"✗ Found", "violation(s)"},
			notWant:      []string{"✓ Repository is valid"},
		},
		{
			name: "non-existent repository",
			args: []string{"validate", "--repo-root", "/path/that/does/not/exist"},
			setup: func(_ *testing.T) string {
				return ""
			},
			wantExitCode: 2,
			wantOutput:   []string{"Repository not found: /path/that/does/not/exist"},
			notWant:      []string{"✓"},
		},
		{
			name: "repository with missing marker file",
			args: []string{"validate"},
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				// Create directories but no marker file
				_ = os.MkdirAll(filepath.Join(dir, "calls"), 0750)
				_ = os.MkdirAll(filepath.Join(dir, "sms"), 0750)
				_ = os.MkdirAll(filepath.Join(dir, "attachments"), 0750)
				return dir
			},
			wantExitCode: 1,
			wantOutput:   []string{"✗ Found", "violation(s)", "missing_marker_file"},
			notWant:      []string{"✓ Repository is valid"},
		},
		{
			name: "quiet mode with repository violations",
			args: []string{"validate", "--quiet"},
			setup: func(t *testing.T) string {
				dir := createValidRepository(t)
				return dir
			},
			wantExitCode: 1,
			wantOutput:   []string{"files.yaml"}, // Shows violations even in quiet mode
			notWant:      []string{"Validation Report", "✗ Found"},
		},
		{
			name: "quiet mode with invalid repository",
			args: []string{"validate", "--quiet"},
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				_ = os.MkdirAll(filepath.Join(dir, "calls"), 0750)
				return dir
			},
			wantExitCode: 1,
			wantOutput:   []string{".mobilecombackup.yaml", "Repository marker file is missing"}, // Only violations in quiet mode
			notWant:      []string{"Validation Report", "✗ Found"},
		},
		{
			name: "verbose mode",
			args: []string{"validate", "--verbose"},
			setup: func(t *testing.T) string {
				dir := createValidRepository(t)
				return dir
			},
			wantExitCode: 1,
			wantOutput:   []string{"Validating repository...", "Completed repository validation", "✗ Found", "violation(s)"},
			notWant:      []string{},
		},
		{
			name: "JSON output for repository with violations",
			args: []string{"validate", "--output-json"},
			setup: func(t *testing.T) string {
				dir := createValidRepository(t)
				return dir
			},
			wantExitCode: 1,
			wantOutput:   []string{`"valid": false`, `"Type": "missing_file"`},
			notWant:      []string{"Validation Report"},
		},
		{
			name: "JSON output for invalid repository",
			args: []string{"validate", "--output-json"},
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				_ = os.MkdirAll(filepath.Join(dir, "calls"), 0750)
				return dir
			},
			wantExitCode: 1,
			wantOutput:   []string{`"valid": false`, `"Type": "missing_marker_file"`},
			notWant:      []string{"Validation Report"},
		},
		{
			name: "environment variable for repo root",
			envVars: map[string]string{
				"MB_REPO_ROOT": "",
			},
			args: []string{"validate"},
			setup: func(t *testing.T) string {
				dir := createValidRepository(t)
				return dir
			},
			wantExitCode: 1,
			wantOutput:   []string{"✗ Found", "violation(s)"},
		},
		{
			name: "CLI flag overrides environment variable",
			envVars: map[string]string{
				"MB_REPO_ROOT": "/some/other/path",
			},
			args: []string{"validate", "--repo-root", ""},
			setup: func(t *testing.T) string {
				dir := createValidRepository(t)
				return dir
			},
			wantExitCode: 1,
			wantOutput:   []string{"✗ Found", "violation(s)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			repoRoot := tt.setup(t)

			// Build command
			args := make([]string, len(tt.args))
			copy(args, tt.args)

			// Replace empty repo-root with actual path
			for i, arg := range args {
				if arg == "--repo-root" && i+1 < len(args) && args[i+1] == "" {
					args[i+1] = repoRoot
				}
			}

			// Set working directory if no repo-root specified
			var cmd *exec.Cmd
			if !contains(args, "--repo-root") && repoRoot != "" {
				cmd = exec.Command(testBin, args...) // #nosec G204
				cmd.Dir = repoRoot
			} else {
				cmd = exec.Command(testBin, args...) // #nosec G204
			}

			// Set environment variables
			if tt.envVars != nil {
				cmd.Env = os.Environ()
				for k, v := range tt.envVars {
					if v == "" && repoRoot != "" {
						// Replace empty value with actual repo root
						cmd.Env = append(cmd.Env, k+"="+repoRoot)
					} else {
						cmd.Env = append(cmd.Env, k+"="+v)
					}
				}
			}

			// Run command
			output, err := cmd.CombinedOutput()

			// Check exit code
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					t.Fatalf("Command failed with non-exit error: %v", err)
				}
			}

			if exitCode != tt.wantExitCode {
				t.Errorf("Exit code = %d, want %d\nOutput: %s", exitCode, tt.wantExitCode, output)
			}

			// Check output
			outputStr := string(output)
			for _, want := range tt.wantOutput {
				if !strings.Contains(outputStr, want) {
					t.Errorf("Output missing expected text %q\nGot: %s", want, outputStr)
				}
			}

			for _, notWant := range tt.notWant {
				if strings.Contains(outputStr, notWant) {
					t.Errorf("Output contains unexpected text %q\nGot: %s", notWant, outputStr)
				}
			}
		})
	}
}

// TestValidateJSONSchema tests that JSON output matches the expected schema
func TestValidateJSONSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Build test binary
	testBin := filepath.Join(t.TempDir(), "mobilecombackup-test")
	buildCmd := exec.Command("go", "build", "-o", testBin, "../../../cmd/mobilecombackup") // #nosec G204
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build test binary: %v\nOutput: %s", err, output)
	}

	// Create repository with violation
	repoRoot := t.TempDir()
	_ = os.MkdirAll(filepath.Join(repoRoot, "calls"), 0750)

	// Run validate with JSON output
	cmd := exec.Command(testBin, "validate", "--repo-root", repoRoot, "--output-json") // #nosec G204
	output, _ := cmd.CombinedOutput()

	// Parse JSON
	var result struct {
		Valid      bool                   `json:"valid"`
		Violations []validation.Violation `json:"violations"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	// Verify structure
	if result.Valid {
		t.Error("Expected valid to be false")
	}

	if len(result.Violations) == 0 {
		t.Error("Expected at least one violation")
	}

	// Check violation structure
	for _, v := range result.Violations {
		if v.Type == "" {
			t.Error("Violation missing type")
		}
		if v.Message == "" {
			t.Error("Violation missing message")
		}
	}
}

// TestValidatePerformance tests validation performance on larger repositories
func TestValidatePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Build test binary
	testBin := filepath.Join(t.TempDir(), "mobilecombackup-test")
	buildCmd := exec.Command("go", "build", "-o", testBin, "../../../cmd/mobilecombackup") // #nosec G204
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build test binary: %v\nOutput: %s", err, output)
	}

	// Create a larger repository
	repoRoot := createValidRepository(t)

	// Add multiple call files
	for year := 2020; year <= 2023; year++ {
		// Calculate timestamp for January 1st of each year
		timestamp := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

		content := `<?xml version="1.0" encoding="UTF-8"?>
<calls count="100">`
		for i := 0; i < 100; i++ {
			content += fmt.Sprintf(`
  <call number="5551234567" duration="60" date="%d" type="1" />`, timestamp+int64(i*3600000)) // Add 1 hour per call
		}
		content += "\n</calls>"

		filename := filepath.Join(repoRoot, "calls", fmt.Sprintf("calls-%d.xml", year))
		_ = os.WriteFile(filename, []byte(content), 0600)
	}

	// Time the validation
	start := time.Now()
	cmd := exec.Command(testBin, "validate", "--repo-root", repoRoot) // #nosec G204
	output, _ := cmd.CombinedOutput()
	duration := time.Since(start)

	// We expect violations but are testing performance
	if !strings.Contains(string(output), "violation(s)") {
		t.Errorf("Expected validation to find violations\nOutput: %s", output)
	}

	// Check performance (should complete within reasonable time)
	if duration > 10*time.Second {
		t.Errorf("Validation took too long: %v", duration)
	}

	t.Logf("Validation completed in %v", duration)
}

// Helper functions

func createValidRepository(t *testing.T) string {
	// For now, let's create a minimal repository that passes validation
	// The full validator expects many things, so we'll keep tests focused on command behavior
	dir := t.TempDir()

	// Create directories
	_ = os.MkdirAll(filepath.Join(dir, "calls"), 0750)
	_ = os.MkdirAll(filepath.Join(dir, "sms"), 0750)
	_ = os.MkdirAll(filepath.Join(dir, "attachments"), 0750)

	// Create marker file
	markerContent := `repository_structure_version: "1"
created_at: "2024-01-01T10:00:00Z"
created_by: "test"
`
	_ = os.WriteFile(filepath.Join(dir, ".mobilecombackup.yaml"), []byte(markerContent), 0600)

	// Note: In reality, a valid repository would need files.yaml with all files listed,
	// proper checksums, etc. For integration tests, we're focusing on command behavior
	// rather than full validation logic (which is tested elsewhere)

	return dir
}

// TestValidateOrphanRemovalIntegration tests the --remove-orphan-attachments flag via CLI
func TestValidateOrphanRemovalIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Build test binary
	testBin := filepath.Join(t.TempDir(), "mobilecombackup-test")
	buildCmd := exec.Command("go", "build", "-o", testBin, "../../../cmd/mobilecombackup") // #nosec G204
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build test binary: %v\nOutput: %s", err, output)
	}

	tests := []struct {
		name         string
		args         []string
		setup        func(t *testing.T) string
		wantExitCode int
		checkOutput  func(t *testing.T, output string)
		checkFiles   func(t *testing.T, repoPath string)
	}{
		{
			name:         "dry run orphan removal",
			args:         []string{"validate", "--remove-orphan-attachments", "--dry-run"},
			setup:        createRepoWithOrphans,
			wantExitCode: 1, // Validation violations exist
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Orphan attachment removal:") {
					t.Error("Expected output to contain orphan removal section")
				}
				if !strings.Contains(output, "Would remove:") {
					t.Error("Expected output to contain 'Would remove:' for dry run")
				}
			},
			checkFiles: func(t *testing.T, repoPath string) {
				// In dry run, orphan files should still exist
				orphanPath := filepath.Join(repoPath, "attachments", "ab", "ab123456789abcdef123456789abcdef123456789abcdef123456789abcdef12")
				if _, err := os.Stat(orphanPath); os.IsNotExist(err) {
					t.Error("Expected orphan file to remain in dry-run mode")
				}
			},
		},
		{
			name:         "actual orphan removal",
			args:         []string{"validate", "--remove-orphan-attachments"},
			setup:        createRepoWithOrphans,
			wantExitCode: 1, // Validation violations exist
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Orphan attachment removal:") {
					t.Error("Expected output to contain orphan removal section")
				}
				if !strings.Contains(output, "Orphans removed:") {
					t.Error("Expected output to contain 'Orphans removed:'")
				}
			},
			checkFiles: func(t *testing.T, repoPath string) {
				// In actual removal, orphan files should be gone
				orphanPath := filepath.Join(repoPath, "attachments", "ab", "ab123456789abcdef123456789abcdef123456789abcdef123456789abcdef12")
				if _, err := os.Stat(orphanPath); !os.IsNotExist(err) {
					t.Error("Expected orphan file to be removed")
				}
			},
		},
		{
			name:         "JSON output with orphan removal",
			args:         []string{"validate", "--remove-orphan-attachments", "--output-json"},
			setup:        createRepoWithOrphans,
			wantExitCode: 1, // Validation violations exist
			checkOutput: func(t *testing.T, output string) {
				// Filter out log messages to get pure JSON
				lines := strings.Split(output, "\n")
				var jsonLines []string
				inJSON := false
				for _, line := range lines {
					if strings.HasPrefix(line, "{") {
						inJSON = true
					}
					if inJSON {
						jsonLines = append(jsonLines, line)
					}
					if strings.HasPrefix(line, "}") && inJSON {
						break
					}
				}

				if len(jsonLines) == 0 {
					t.Error("No JSON found in output")
					return
				}

				jsonContent := strings.Join(jsonLines, "\n")

				// Validate JSON structure
				var result struct {
					Valid         bool `json:"valid"`
					OrphanRemoval *struct {
						AttachmentsScanned int   `json:"attachments_scanned"`
						OrphansFound       int   `json:"orphans_found"`
						OrphansRemoved     int   `json:"orphans_removed"`
						BytesFreed         int64 `json:"bytes_freed"`
					} `json:"orphan_removal"`
				}

				if err := json.Unmarshal([]byte(jsonContent), &result); err != nil {
					t.Errorf("Failed to parse JSON output: %v", err)
					return
				}

				if result.OrphanRemoval == nil {
					t.Error("Expected orphan_removal section in JSON output")
					return
				}

				if result.OrphanRemoval.OrphansFound == 0 {
					t.Error("Expected to find orphans in test setup")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoPath := tt.setup(t)

			// Run command
			cmd := exec.Command(testBin, tt.args...) // #nosec G204
			cmd.Args = append(cmd.Args, "--repo-root", repoPath)

			output, err := cmd.CombinedOutput()

			// Check exit code
			var exitCode int
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					t.Fatalf("Failed to run command: %v", err)
				}
			}

			if exitCode != tt.wantExitCode {
				t.Errorf("Expected exit code %d, got %d\nOutput: %s", tt.wantExitCode, exitCode, output)
			}

			// Check output
			if tt.checkOutput != nil {
				tt.checkOutput(t, string(output))
			}

			// Check files
			if tt.checkFiles != nil {
				tt.checkFiles(t, repoPath)
			}
		})
	}
}

// createRepoWithOrphans creates a test repository with orphaned attachments
func createRepoWithOrphans(t *testing.T) string {
	tempDir := t.TempDir()

	// Create repository structure
	for _, dir := range []string{"calls", "sms", "attachments", "contacts"} {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0750); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create marker file
	markerContent := `version: 1
created_at: "2024-01-01T10:00:00Z"
created_by: "test"
`
	if err := os.WriteFile(filepath.Join(tempDir, ".mobilecombackup.yaml"), []byte(markerContent), 0600); err != nil {
		t.Fatalf("Failed to create marker file: %v", err)
	}

	// Create SMS file that references one attachment
	smsContent := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="1">
  <sms address="555-1234" type="1" subject="null" body="Test message" date="1609459200000" readable_date="Jan 1, 2021 12:00:00 AM" />
</smses>`
	if err := os.WriteFile(filepath.Join(tempDir, "sms", "sms-2021.xml"), []byte(smsContent), 0600); err != nil {
		t.Fatalf("Failed to create SMS file: %v", err)
	}

	// Create two attachments - one referenced, one orphaned
	referencedHash := "a123456789abcdef123456789abcdef123456789abcdef123456789abcdef12"
	orphanHash := "ab123456789abcdef123456789abcdef123456789abcdef123456789abcdef12"

	// Create referenced attachment
	refDir := filepath.Join(tempDir, "attachments", "a1")
	if err := os.MkdirAll(refDir, 0750); err != nil {
		t.Fatalf("Failed to create referenced attachment directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(refDir, referencedHash), []byte("referenced content"), 0600); err != nil {
		t.Fatalf("Failed to create referenced attachment: %v", err)
	}

	// Create orphaned attachment
	orphanDir := filepath.Join(tempDir, "attachments", "ab")
	if err := os.MkdirAll(orphanDir, 0750); err != nil {
		t.Fatalf("Failed to create orphan attachment directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(orphanDir, orphanHash), []byte("orphaned content"), 0600); err != nil {
		t.Fatalf("Failed to create orphan attachment: %v", err)
	}

	return tempDir
}
