package cmd_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestInitCommandIntegration tests the init command via the CLI
func TestInitCommandIntegration(t *testing.T) {
	// Build test binary
	testBin := filepath.Join(t.TempDir(), "mobilecombackup-test")
	buildCmd := exec.Command("go", "build", "-o", testBin, "../../../cmd/mobilecombackup")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build test binary: %v\nOutput: %s", err, output)
	}

	tests := []struct {
		name      string
		args      []string
		setup     func(t *testing.T) string
		wantError bool
		wantText  string
		validate  func(t *testing.T, repoRoot string)
	}{
		{
			name: "init in new directory",
			args: []string{"init"},
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "new-repo")
			},
			wantError: false,
			wantText:  "Initialized mobilecombackup repository",
			validate: func(t *testing.T, repoRoot string) {
				// Check directories
				for _, dir := range []string{"calls", "sms", "attachments"} {
					path := filepath.Join(repoRoot, dir)
					if info, err := os.Stat(path); err != nil {
						t.Errorf("Directory %s not created: %v", dir, err)
					} else if !info.IsDir() {
						t.Errorf("%s is not a directory", dir)
					}
				}

				// Check marker file
				markerPath := filepath.Join(repoRoot, ".mobilecombackup.yaml")
				data, err := os.ReadFile(markerPath)
				if err != nil {
					t.Errorf("Marker file not created: %v", err)
				} else {
					var marker map[string]interface{}
					if err := yaml.Unmarshal(data, &marker); err != nil {
						t.Errorf("Invalid marker file YAML: %v", err)
					} else {
						if v, ok := marker["repository_structure_version"].(string); !ok || v != "1" {
							t.Errorf("Invalid repository version: %v", marker["repository_structure_version"])
						}
					}
				}
			},
		},
		{
			name: "init with --dry-run",
			args: []string{"init", "--dry-run"},
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "dry-run-repo")
			},
			wantError: false,
			wantText:  "DRY RUN: No files or directories were created",
			validate: func(t *testing.T, repoRoot string) {
				// Should not create anything
				if _, err := os.Stat(repoRoot); !os.IsNotExist(err) {
					t.Error("Dry run created repository root")
				}
			},
		},
		{
			name: "init with --quiet",
			args: []string{"init", "--quiet"},
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "quiet-repo")
			},
			wantError: false,
			wantText:  "", // No output expected
			validate: func(t *testing.T, repoRoot string) {
				// Check that repository was created
				if _, err := os.Stat(repoRoot); err != nil {
					t.Errorf("Repository not created: %v", err)
				}
			},
		},
		{
			name: "init in existing repository",
			args: []string{"init"},
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				// Create marker file to simulate existing repo
				f, _ := os.Create(filepath.Join(dir, ".mobilecombackup.yaml"))
				_ = f.Close()
				return dir
			},
			wantError: true,
			wantText:  "already contains a mobilecombackup repository",
			validate:  func(t *testing.T, repoRoot string) {},
		},
		{
			name: "init in non-empty directory",
			args: []string{"init"},
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				// Create a file to make it non-empty
				f, _ := os.Create(filepath.Join(dir, "existing.txt"))
				_ = f.Close()
				return dir
			},
			wantError: true,
			wantText:  "directory is not empty",
			validate:  func(t *testing.T, repoRoot string) {},
		},
		{
			name: "init with custom repo-root",
			args: []string{"init"},
			setup: func(t *testing.T) string {
				// Create custom path
				baseDir := t.TempDir()
				customPath := filepath.Join(baseDir, "custom-repo")
				return customPath
			},
			wantError: false,
			wantText:  "Initialized mobilecombackup repository",
			validate: func(t *testing.T, repoRoot string) {
				// Check that repository was created at custom path
				if _, err := os.Stat(repoRoot); err != nil {
					t.Errorf("Repository not created at custom path: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoRoot := tt.setup(t)

			// Build command
			args := append([]string{}, tt.args...)
			// If no --repo-root specified, add it
			if !contains(args, "--repo-root") {
				args = append(args, "--repo-root", repoRoot)
			}

			cmd := exec.Command(testBin, args...)
			output, err := cmd.CombinedOutput()

			// Check error expectation
			if tt.wantError && err == nil {
				t.Fatalf("Expected error but command succeeded. Output: %s", output)
			}
			if !tt.wantError && err != nil {
				t.Fatalf("Command failed: %v\nOutput: %s", err, output)
			}

			// Check output contains expected text
			if tt.wantText != "" && !strings.Contains(string(output), tt.wantText) {
				t.Errorf("Output doesn't contain expected text.\nWant: %q\nGot: %s", tt.wantText, output)
			}

			// Additional validation
			tt.validate(t, repoRoot)
		})
	}
}

// TestInitCommandTreeOutput tests the tree-style output formatting
func TestInitCommandTreeOutput(t *testing.T) {
	// Save current directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldDir) }()

	// Build test binary
	testBin := filepath.Join(t.TempDir(), "mobilecombackup-test")
	buildCmd := exec.Command("go", "build", "-o", testBin, "../../../cmd/mobilecombackup")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build test binary: %v\nOutput: %s", err, output)
	}

	repoRoot := filepath.Join(t.TempDir(), "tree-test")
	cmd := exec.Command(testBin, "init", "--repo-root", repoRoot)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}

	// Check tree structure in output
	expectedLines := []string{
		"Created structure:",
		"tree-test",
		"├── calls",
		"├── sms",
		"├── attachments",
		"├── .mobilecombackup.yaml",
		"├── contacts.yaml",
		"└── summary.yaml",
	}

	outputStr := string(output)
	for _, line := range expectedLines {
		if !strings.Contains(outputStr, line) {
			t.Errorf("Output missing expected line: %q\nFull output:\n%s", line, outputStr)
		}
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
