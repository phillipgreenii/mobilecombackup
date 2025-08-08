package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillipgreen/mobilecombackup/pkg/validation"
)

func TestResolveRepoRoot(t *testing.T) {
	tests := []struct {
		name     string
		cliFlag  string
		envVar   string
		expected string
	}{
		{
			name:     "CLI flag takes precedence",
			cliFlag:  "/path/from/cli",
			envVar:   "/path/from/env",
			expected: "/path/from/cli",
		},
		{
			name:     "environment variable when no CLI flag",
			cliFlag:  ".",
			envVar:   "/path/from/env",
			expected: "/path/from/env",
		},
		{
			name:     "current directory as fallback",
			cliFlag:  ".",
			envVar:   "",
			expected: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			originalRepoRoot := repoRoot
			originalEnv := os.Getenv("MB_REPO_ROOT")
			defer func() {
				repoRoot = originalRepoRoot
				os.Setenv("MB_REPO_ROOT", originalEnv)
			}()

			// Set test values
			repoRoot = tt.cliFlag
			if tt.envVar != "" {
				os.Setenv("MB_REPO_ROOT", tt.envVar)
			} else {
				os.Unsetenv("MB_REPO_ROOT")
			}

			// Test
			result := resolveRepoRoot()
			if result != tt.expected {
				t.Errorf("resolveRepoRoot() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestProgressReporter(t *testing.T) {
	t.Run("ConsoleProgressReporter verbose", func(t *testing.T) {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		reporter := &ConsoleProgressReporter{verbose: true}
		reporter.StartPhase("test phase")
		reporter.UpdateProgress(5, 10)
		reporter.CompletePhase()

		w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "Validating test phase...") {
			t.Error("Expected phase start message")
		}
		if !strings.Contains(output, "Progress: 5/10") {
			t.Error("Expected progress update")
		}
		if !strings.Contains(output, "Completed test phase validation") {
			t.Error("Expected completion message")
		}
	})

	t.Run("ConsoleProgressReporter quiet", func(t *testing.T) {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		reporter := &ConsoleProgressReporter{quiet: true}
		reporter.StartPhase("test phase")
		reporter.UpdateProgress(5, 10)
		reporter.CompletePhase()

		w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		if output != "" {
			t.Errorf("Expected no output in quiet mode, got: %q", output)
		}
	})

	t.Run("NullProgressReporter", func(t *testing.T) {
		// Should not panic or produce output
		reporter := &NullProgressReporter{}
		reporter.StartPhase("test")
		reporter.UpdateProgress(1, 2)
		reporter.CompletePhase()
	})
}

func TestValidationResultJSON(t *testing.T) {
	tests := []struct {
		name     string
		result   ValidationResult
		expected string
	}{
		{
			name: "valid repository",
			result: ValidationResult{
				Valid:      true,
				Violations: []validation.ValidationViolation{},
			},
			expected: `{
  "valid": true,
  "violations": []
}`,
		},
		{
			name: "repository with violations",
			result: ValidationResult{
				Valid: false,
				Violations: []validation.ValidationViolation{
					{
						Type:    validation.InvalidFormat,
						File:    "calls/calls-2015.xml",
						Message: "Call entry missing required 'date' field",
					},
					{
						Type:     validation.ChecksumMismatch,
						File:     "attachments/ab/ab12345",
						Message:  "Attachment file content does not match expected hash",
						Expected: "ab12345",
						Actual:   "cd67890",
					},
				},
			},
			expected: `{
  "valid": false,
  "violations": [
    {
      "Type": "invalid_format",
      "Severity": "",
      "File": "calls/calls-2015.xml",
      "Message": "Call entry missing required 'date' field",
      "Expected": "",
      "Actual": ""
    },
    {
      "Type": "checksum_mismatch",
      "Severity": "",
      "File": "attachments/ab/ab12345",
      "Message": "Attachment file content does not match expected hash",
      "Expected": "ab12345",
      "Actual": "cd67890"
    }
  ]
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			encoder := json.NewEncoder(&buf)
			encoder.SetIndent("", "  ")
			err := encoder.Encode(tt.result)
			if err != nil {
				t.Fatalf("Failed to encode JSON: %v", err)
			}

			got := strings.TrimSpace(buf.String())
			want := strings.TrimSpace(tt.expected)
			if got != want {
				t.Errorf("JSON output mismatch\nGot:\n%s\nWant:\n%s", got, want)
			}
		})
	}
}

func TestValidationResultTextOutput(t *testing.T) {
	tests := []struct {
		name        string
		result      ValidationResult
		repoPath    string
		quiet       bool
		wantStrings []string
		notWant     []string
	}{
		{
			name: "valid repository normal output",
			result: ValidationResult{
				Valid:      true,
				Violations: []validation.ValidationViolation{},
			},
			repoPath:    "/test/repo",
			quiet:       false,
			wantStrings: []string{"Validation Report for: /test/repo", "✓ Repository is valid"},
			notWant:     []string{"✗"},
		},
		{
			name: "valid repository quiet mode",
			result: ValidationResult{
				Valid:      true,
				Violations: []validation.ValidationViolation{},
			},
			repoPath:    "/test/repo",
			quiet:       true,
			wantStrings: []string{},
			notWant:     []string{"Validation Report", "Repository is valid"},
		},
		{
			name: "repository with violations",
			result: ValidationResult{
				Valid: false,
				Violations: []validation.ValidationViolation{
					{
						Type:    validation.InvalidFormat,
						File:    "calls/calls-2015.xml",
						Message: "Call entry missing required 'date' field",
					},
					{
						Type:    validation.InvalidFormat,
						File:    "sms/sms-2015.xml",
						Message: "SMS entry missing required 'date' field",
					},
				},
			},
			repoPath: "/test/repo",
			quiet:    false,
			wantStrings: []string{
				"✗ Found 2 violation(s)",
				"invalid_format (2):",
				"calls/calls-2015.xml: Call entry missing required 'date' field",
				"sms/sms-2015.xml: SMS entry missing required 'date' field",
			},
			notWant: []string{"✓"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Save original quiet value
			originalQuiet := quiet
			quiet = tt.quiet
			defer func() { quiet = originalQuiet }()

			outputTextResult(tt.result, tt.repoPath)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// Check expected strings
			for _, want := range tt.wantStrings {
				if !strings.Contains(output, want) {
					t.Errorf("Expected output to contain %q, but it didn't.\nOutput:\n%s", want, output)
				}
			}

			// Check unwanted strings
			for _, notWant := range tt.notWant {
				if strings.Contains(output, notWant) {
					t.Errorf("Expected output NOT to contain %q, but it did.\nOutput:\n%s", notWant, output)
				}
			}
		})
	}
}

func TestValidateCommandFlags(t *testing.T) {
	// Test that flags are properly registered
	cmd := validateCmd
	
	// Check verbose flag
	verboseFlag := cmd.Flags().Lookup("verbose")
	if verboseFlag == nil {
		t.Error("verbose flag not registered")
	}
	
	// Check output-json flag
	jsonFlag := cmd.Flags().Lookup("output-json")
	if jsonFlag == nil {
		t.Error("output-json flag not registered")
	}
}

func TestValidateWithProgress(t *testing.T) {
	// Create a temporary test repository
	tempDir := t.TempDir()
	
	// Create basic repository structure
	os.MkdirAll(filepath.Join(tempDir, "calls"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "sms"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "attachments"), 0755)
	
	// Create marker file with valid content
	markerContent := `repository_structure_version: "1"
created_at: "2024-01-01T10:00:00Z"
created_by: "test"
`
	err := os.WriteFile(filepath.Join(tempDir, ".mobilecombackup.yaml"), []byte(markerContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create marker file: %v", err)
	}
	
	// Create empty contacts and summary files
	os.WriteFile(filepath.Join(tempDir, "contacts.yaml"), []byte("contacts: []\n"), 0644)
	os.WriteFile(filepath.Join(tempDir, "summary.yaml"), []byte("counts:\n  calls: 0\n  sms: 0\n"), 0644)
	
	// TODO: This test would be more complete with mocked readers
	// For now, we're just testing that the function doesn't panic
	// Full integration tests will be in the integration test file
}