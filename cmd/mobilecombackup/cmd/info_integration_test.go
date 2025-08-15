package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInfoCommandIntegration(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want int // exit code
	}{
		{
			name: "info command with empty repository",
			args: []string{"info"},
			want: 0,
		},
		{
			name: "info command with JSON output",
			args: []string{"info", "--json"},
			want: 0,
		},
		{
			name: "info command with repository from flag",
			args: []string{"info", "--repo-root", "test-repo"},
			want: 0,
		},
		{
			name: "info command with non-existent repository",
			args: []string{"info", "--repo-root", "nonexistent"},
			want: 2,
		},
		{
			name: "info command with quiet flag",
			args: []string{"info", "--quiet"},
			want: 0,
		},
	}

	// Build test binary once for all tests
	testBin := buildTestBinary(t)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create temporary directory for this test
			tempDir := t.TempDir()

			// Set up test repository if needed
			if !strings.Contains(test.name, "non-existent") {
				repoPath := filepath.Join(tempDir, "repo")
				setupTestRepository(t, repoPath)

				// Update args to use the test repository
				for i, arg := range test.args {
					if arg == "test-repo" {
						test.args[i] = repoPath
					}
				}

				// If no repo-root specified, change to the repo directory
				if !containsRepoRootFlag(test.args) {
					oldWd, err := os.Getwd()
					if err != nil {
						t.Fatalf("Failed to get working directory: %v", err)
					}
					defer func() { _ = os.Chdir(oldWd) }()

					if err := os.Chdir(repoPath); err != nil {
						t.Fatalf("Failed to change to test directory: %v", err)
					}
				}
			}

			// Execute command
			cmd := exec.Command(testBin, test.args...) // #nosec G204
			output, err := cmd.CombinedOutput()

			// Check exit code
			exitCode := 0
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else if err != nil {
				t.Fatalf("Failed to execute command: %v", err)
			}

			if exitCode != test.want {
				t.Errorf("Exit code = %d, want %d\nOutput: %s", exitCode, test.want, output)
			}

			// Additional checks based on test case
			outputStr := string(output)
			switch test.name {
			case "info command with empty repository":
				if !strings.Contains(outputStr, "Repository:") {
					t.Error("Expected repository information in output")
				}
				if !strings.Contains(outputStr, "Validation: OK") {
					t.Error("Expected validation status in output")
				}
			case "info command with JSON output":
				// Verify it's valid JSON
				var info RepositoryInfo
				if err := json.Unmarshal(output, &info); err != nil {
					t.Errorf("Expected valid JSON output, got error: %v\nOutput: %s", err, output)
				}
			case "info command with quiet flag":
				if len(strings.TrimSpace(outputStr)) > 0 {
					t.Error("Expected no output in quiet mode for empty repository")
				}
			case "info command with non-existent repository":
				if !strings.Contains(outputStr, "Repository not found") {
					t.Error("Expected 'Repository not found' error message")
				}
			}
		})
	}
}

func TestInfoCommandWithData(t *testing.T) {
	// Build test binary
	testBin := buildTestBinary(t)

	// Create test repository with actual data
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	setupTestRepositoryWithData(t, repoPath)

	// Test text output
	cmd := exec.Command(testBin, "info", "--repo-root", repoPath) // #nosec G204
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Verify key elements are present
	checks := []string{
		"Repository:",
		"Version:",
		"Created:",
		"Calls:",
		"2014:",
		"Total:",
		"Validation: OK",
	}

	for _, check := range checks {
		if !strings.Contains(outputStr, check) {
			t.Errorf("Expected '%s' in output, got:\n%s", check, outputStr)
		}
	}

	// Test JSON output
	cmd = exec.Command(testBin, "info", "--repo-root", repoPath, "--json") // #nosec G204
	output, err = cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("JSON command failed: %v\nOutput: %s", err, output)
	}

	var info RepositoryInfo
	if err := json.Unmarshal(output, &info); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	// Verify JSON structure
	if info.Version == "" {
		t.Error("Expected version in JSON output")
	}

	if len(info.Calls) == 0 {
		t.Error("Expected calls data in JSON output")
	}

	if !info.ValidationOK {
		t.Error("Expected validation OK in JSON output")
	}
}

func TestInfoCommandEnvironmentVariable(t *testing.T) {
	// Build test binary
	testBin := buildTestBinary(t)

	// Create test repository
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	setupTestRepository(t, repoPath)

	// Set environment variable
	cmd := exec.Command(testBin, "info") // #nosec G204
	cmd.Env = append(os.Environ(), "MB_REPO_ROOT="+repoPath)

	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Command with env var failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Repository:") {
		t.Error("Expected repository information when using environment variable")
	}
}

func setupTestRepository(t *testing.T, repoPath string) {
	t.Helper()

	// Create directory structure
	dirs := []string{
		repoPath,
		filepath.Join(repoPath, "calls"),
		filepath.Join(repoPath, "sms"),
		filepath.Join(repoPath, "attachments"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create marker file
	markerContent := `repository_structure_version: "1"
created_at: "2024-01-15T10:30:00Z"
created_by: "mobilecombackup v1.0.0"
`
	markerPath := filepath.Join(repoPath, ".mobilecombackup.yaml")
	if err := os.WriteFile(markerPath, []byte(markerContent), 0644); err != nil {
		t.Fatalf("Failed to create marker file: %v", err)
	}

	// Create empty contacts file
	contactsPath := filepath.Join(repoPath, "contacts.yaml")
	contactsContent := "contacts: []\n"
	if err := os.WriteFile(contactsPath, []byte(contactsContent), 0644); err != nil {
		t.Fatalf("Failed to create contacts file: %v", err)
	}
}

func setupTestRepositoryWithData(t *testing.T, repoPath string) {
	t.Helper()

	// First create the basic structure
	setupTestRepository(t, repoPath)

	// Create test calls data
	callsContent := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="2">
  <call number="555-1234" duration="120" date="1388534400000" type="2" ` +
		`readable_date="Jan 1, 2014 12:00:00 AM" contact_name="John Doe" />
  <call number="555-5678" duration="60" date="1420070400000" type="1" ` +
		`readable_date="Jan 1, 2015 12:00:00 AM" contact_name="Jane Smith" />
</calls>`

	callsPath := filepath.Join(repoPath, "calls", "calls-2014.xml")
	if err := os.WriteFile(callsPath, []byte(callsContent), 0644); err != nil {
		t.Fatalf("Failed to create calls file: %v", err)
	}

	// Create another calls file for 2015
	calls2015Content := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="1">
  <call number="555-9999" duration="90" date="1451606400000" type="1" ` +
		`readable_date="Jan 1, 2016 12:00:00 AM" contact_name="Bob Wilson" />
</calls>`

	calls2015Path := filepath.Join(repoPath, "calls", "calls-2015.xml")
	if err := os.WriteFile(calls2015Path, []byte(calls2015Content), 0644); err != nil {
		t.Fatalf("Failed to create 2015 calls file: %v", err)
	}

	// Create test SMS data
	smsContent := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="2">
  <sms protocol="0" address="555-1111" date="1388534400000" type="1" subject="null" body="Hello World" ` +
		`read="1" status="-1" locked="0" date_sent="1388534400000" ` +
		`readable_date="Jan 1, 2014 12:00:00 AM" contact_name="Test Contact" />
  <mms date="1420070400000" msg_box="1" address="555-2222" m_type="132" thread_id="1" text_only="0" ` +
		`read="1" readable_date="Jan 1, 2015 12:00:00 AM" contact_name="MMS Contact">
    <parts>
      <part seq="0" ct="text/plain" text="Hello MMS World" />
    </parts>
  </mms>
</smses>`

	smsPath := filepath.Join(repoPath, "sms", "sms-2014.xml")
	if err := os.WriteFile(smsPath, []byte(smsContent), 0644); err != nil {
		t.Fatalf("Failed to create SMS file: %v", err)
	}

	// Create test contacts data
	contactsContent := `contacts:
  - name: "John Doe"
    numbers: ["555-1234", "555-4321"]
  - name: "Jane Smith"
    numbers: ["555-5678"]
`
	contactsPath := filepath.Join(repoPath, "contacts.yaml")
	if err := os.WriteFile(contactsPath, []byte(contactsContent), 0644); err != nil {
		t.Fatalf("Failed to create contacts file: %v", err)
	}
}

func buildTestBinary(t *testing.T) string {
	t.Helper()

	testBin := filepath.Join(t.TempDir(), "mobilecombackup-test")

	buildCmd := exec.Command("go", "build", "-o", testBin, "../../../cmd/mobilecombackup") // #nosec G204
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build test binary: %v", err)
	}

	return testBin
}

func containsRepoRootFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--repo-root" {
			return true
		}
	}
	return false
}
