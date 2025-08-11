package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// testBinPath is set once per test suite
var testBinPath string

func TestMain(m *testing.M) {
	// Build test binary once for all tests
	tmpDir, err := os.MkdirTemp("", "mobilecombackup-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	testBinPath = filepath.Join(tmpDir, "mobilecombackup-test")
	buildCmd := exec.Command("go", "build", "-o", testBinPath, "../../../cmd/mobilecombackup")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build test binary: %v\nOutput: %s\n", err, output)
		os.Exit(1)
	}

	// Run tests
	os.Exit(m.Run())
}

func TestImportIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name            string
		setupFunc       func(t *testing.T) (repoPath string, importPath string)
		args            []string
		wantExitCode    int
		wantInOutput    []string
		wantNotInOutput []string
		checkFunc       func(t *testing.T, repoPath string)
	}{
		{
			name: "import into empty repository",
			setupFunc: func(t *testing.T) (string, string) {
				// Create empty repository
				repoPath := t.TempDir()
				initRepo(t, repoPath)

				// Create import directory with test files
				importPath := t.TempDir()
				copyTestFile(t, "../../../../testdata/to_process/00/calls-test.xml",
					filepath.Join(importPath, "calls-test.xml"))
				copyTestFile(t, "../../../../testdata/to_process/sms-test.xml",
					filepath.Join(importPath, "sms-test.xml"))

				return repoPath, importPath
			},
			args:         []string{"--quiet"},
			wantExitCode: 0,
			checkFunc: func(t *testing.T, repoPath string) {
				// Check that files were created
				checkFileExists(t, filepath.Join(repoPath, "calls", "calls-2014.xml"))
				checkFileExists(t, filepath.Join(repoPath, "calls", "calls-2015.xml"))
				checkFileExists(t, filepath.Join(repoPath, "sms", "sms-2013.xml"))
				checkFileExists(t, filepath.Join(repoPath, "sms", "sms-2014.xml"))
			},
		},
		{
			name: "import with filter calls only",
			setupFunc: func(t *testing.T) (string, string) {
				// Create empty repository
				repoPath := t.TempDir()
				initRepo(t, repoPath)

				// Create import directory with test files
				importPath := t.TempDir()
				copyTestFile(t, "../../../../testdata/to_process/00/calls-test.xml",
					filepath.Join(importPath, "calls-test.xml"))
				copyTestFile(t, "../../../../testdata/to_process/sms-test.xml",
					filepath.Join(importPath, "sms-test.xml"))

				return repoPath, importPath
			},
			args:         []string{"--filter", "calls", "--quiet"},
			wantExitCode: 0,
			checkFunc: func(t *testing.T, repoPath string) {
				// Check that only calls were imported
				checkFileExists(t, filepath.Join(repoPath, "calls", "calls-2014.xml"))
				checkFileNotExists(t, filepath.Join(repoPath, "sms", "sms-2013.xml"))
			},
		},
		{
			name: "dry run mode",
			setupFunc: func(t *testing.T) (string, string) {
				// Create empty repository
				repoPath := t.TempDir()
				initRepo(t, repoPath)

				// Create import directory with test files
				importPath := t.TempDir()
				copyTestFile(t, "../../../../testdata/to_process/00/calls-test.xml",
					filepath.Join(importPath, "calls-test.xml"))

				return repoPath, importPath
			},
			args:         []string{"--dry-run"},
			wantExitCode: 0,
			wantInOutput: []string{"DRY RUN", "Files processed: 1"},
			checkFunc: func(t *testing.T, repoPath string) {
				// Check that no files were created
				checkFileNotExists(t, filepath.Join(repoPath, "calls", "calls-2014.xml"))
			},
		},
		{
			name: "json output mode",
			setupFunc: func(t *testing.T) (string, string) {
				// Create empty repository
				repoPath := t.TempDir()
				initRepo(t, repoPath)

				// Create import directory with test files
				importPath := t.TempDir()
				copyTestFile(t, "../../../../testdata/to_process/00/calls-test.xml",
					filepath.Join(importPath, "calls-test.xml"))

				return repoPath, importPath
			},
			args:         []string{"--json"},
			wantExitCode: 0,
			checkFunc: func(t *testing.T, repoPath string) {
				// JSON output is checked in test body
			},
		},
		{
			name: "invalid repository",
			setupFunc: func(t *testing.T) (string, string) {
				// Non-existent repository
				repoPath := filepath.Join(t.TempDir(), "nonexistent")
				importPath := t.TempDir()
				return repoPath, importPath
			},
			args:         []string{},
			wantExitCode: 2,
			wantInOutput: []string{"invalid repository"},
		},
		{
			name: "repository from environment",
			setupFunc: func(t *testing.T) (string, string) {
				// Create repository
				repoPath := t.TempDir()
				initRepo(t, repoPath)

				// Set environment variable
				os.Setenv("MB_REPO_ROOT", repoPath)
				t.Cleanup(func() { os.Unsetenv("MB_REPO_ROOT") })

				// Create import directory
				importPath := t.TempDir()
				copyTestFile(t, "../../../../testdata/to_process/00/calls-test.xml",
					filepath.Join(importPath, "calls-test.xml"))

				return "", importPath // Empty repo path to test env var
			},
			args:         []string{"--quiet"},
			wantExitCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoPath, importPath := tt.setupFunc(t)

			// Build command arguments
			args := []string{"import"}
			if repoPath != "" {
				args = append(args, "--repo-root", repoPath)
			}
			args = append(args, tt.args...)
			if importPath != "" {
				args = append(args, importPath)
			}

			// Execute command
			cmd := exec.Command(testBinPath, args...)
			output, err := cmd.CombinedOutput()

			// Check exit code
			exitCode := 0
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}

			if exitCode != tt.wantExitCode {
				t.Errorf("Exit code = %d, want %d\nOutput: %s", exitCode, tt.wantExitCode, output)
			}

			// Check output content
			outputStr := string(output)
			for _, want := range tt.wantInOutput {
				if !strings.Contains(outputStr, want) {
					t.Errorf("Output missing %q\nGot: %s", want, outputStr)
				}
			}

			for _, notWant := range tt.wantNotInOutput {
				if strings.Contains(outputStr, notWant) {
					t.Errorf("Output should not contain %q\nGot: %s", notWant, outputStr)
				}
			}

			// Check JSON output if applicable
			if tt.name == "json output mode" && exitCode == 0 {
				var result map[string]interface{}
				if err := json.Unmarshal(output, &result); err != nil {
					t.Errorf("Failed to parse JSON output: %v\nOutput: %s", err, output)
				}

				// Verify expected fields
				if _, ok := result["files_processed"]; !ok {
					t.Error("JSON missing 'files_processed' field")
				}
				if _, ok := result["total"]; !ok {
					t.Error("JSON missing 'total' field")
				}
			}

			// Run additional checks
			if tt.checkFunc != nil && repoPath != "" {
				tt.checkFunc(t, repoPath)
			}
		})
	}
}

func TestImportScanningLogic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create test directory structure
	testDir := t.TempDir()
	repoPath := filepath.Join(testDir, "repo")
	initRepo(t, repoPath)

	importDir := filepath.Join(testDir, "import")

	// Create directory structure with various files
	createTestFile(t, filepath.Join(importDir, "calls-1.xml"), "<calls></calls>")
	createTestFile(t, filepath.Join(importDir, "sms-1.xml"), "<sms></sms>")
	createTestFile(t, filepath.Join(importDir, "other.xml"), "<other></other>")              // Should be ignored
	createTestFile(t, filepath.Join(importDir, ".hidden", "calls-2.xml"), "<calls></calls>") // In hidden dir
	createTestFile(t, filepath.Join(importDir, "subdir", "calls-3.xml"), "<calls></calls>")
	createTestFile(t, filepath.Join(importDir, "subdir", "sms-2.xml"), "<sms></sms>")

	// Also create files that should be excluded (already in repo structure)
	os.MkdirAll(filepath.Join(importDir, "calls"), 0755)
	createTestFile(t, filepath.Join(importDir, "calls", "calls-2020.xml"), "<calls></calls>")

	// Run import with verbose to see what files are processed
	cmd := exec.Command(testBinPath, "import", "--repo-root", repoPath, "--verbose", importDir)
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Import failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Check that correct files were processed
	expectedFiles := []string{
		"calls-1.xml",
		"sms-1.xml",
		"calls-3.xml",
		"sms-2.xml",
	}

	for _, file := range expectedFiles {
		if !strings.Contains(outputStr, file) {
			t.Errorf("Expected file %s to be processed", file)
		}
	}

	// Check that these files were NOT processed
	unexpectedFiles := []string{
		"other.xml",            // Wrong pattern
		".hidden/calls-2.xml",  // Hidden directory
		"calls/calls-2020.xml", // In repository structure
	}

	for _, file := range unexpectedFiles {
		if strings.Contains(outputStr, file) {
			t.Errorf("File %s should not have been processed", file)
		}
	}
}

// Helper functions

func initRepo(t *testing.T, repoPath string) {
	cmd := exec.Command(testBinPath, "init", "--repo-root", repoPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init repository: %v\nOutput: %s", err, output)
	}
}

func copyTestFile(t *testing.T, src, dst string) {
	// Resolve src path relative to test file location
	srcAbs, err := filepath.Abs(src)
	if err != nil {
		t.Fatalf("Failed to resolve source path: %v", err)
	}

	data, err := os.ReadFile(srcAbs)
	if err != nil {
		t.Fatalf("Failed to read test file %s: %v", src, err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
}

func createTestFile(t *testing.T, path, content string) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
}

func checkFileExists(t *testing.T, path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist", path)
	}
}

func checkFileNotExists(t *testing.T, path string) {
	if _, err := os.Stat(path); err == nil {
		t.Errorf("Expected file %s to not exist", path)
	}
}
