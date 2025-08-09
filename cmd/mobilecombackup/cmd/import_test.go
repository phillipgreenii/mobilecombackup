package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/importer"
	"github.com/spf13/cobra"
)

func TestImportCommand(t *testing.T) {
	// Save original values
	origArgs := os.Args
	origRepoRoot := repoRoot
	defer func() {
		os.Args = origArgs
		repoRoot = origRepoRoot
	}()

	tests := []struct {
		name        string
		args        []string
		env         map[string]string
		wantErr     bool
		errContains string
	}{
		{
			name: "no repository and no args",
			args: []string{"mobilecombackup", "import"},
			wantErr: true,
			errContains: "no repository specified",
		},
		{
			name: "valid repository from flag",
			args: []string{"mobilecombackup", "import", "--repo-root", "test-repo"},
			wantErr: true, // Will fail on non-existent repo
			errContains: "invalid repository",
		},
		{
			name: "valid repository from env",
			args: []string{"mobilecombackup", "import"},
			env: map[string]string{"MB_REPO_ROOT": "test-repo"},
			wantErr: true, // Will fail on non-existent repo
			errContains: "invalid repository",
		},
		{
			name: "invalid filter value",
			args: []string{"mobilecombackup", "import", "--repo-root", ".", "--filter", "invalid"},
			wantErr: true,
			errContains: "invalid filter value",
		},
		{
			name: "help flag",
			args: []string{"mobilecombackup", "import", "--help"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset cobra command
			rootCmd = &cobra.Command{
				Use:   "mobilecombackup",
				Short: "A tool for processing mobile phone backup files",
			}
			// Re-register commands manually
			rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Suppress non-error output")
			rootCmd.PersistentFlags().StringVar(&repoRoot, "repo-root", ".", "Path to repository root")
			rootCmd.AddCommand(importCmd)

			// Set environment variables
			for k, v := range tt.env {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			// Set command line arguments
			os.Args = tt.args

			// Capture output
			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)

			// Execute command
			err := rootCmd.Execute()

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error = %v, want error containing %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestResolveImportRepoRoot(t *testing.T) {
	// Save original values
	origRepoRoot := repoRoot
	defer func() {
		repoRoot = origRepoRoot
	}()

	tests := []struct {
		name     string
		flagVal  string
		envVal   string
		expected string
	}{
		{
			name:     "flag takes precedence",
			flagVal:  "/flag/path",
			envVal:   "/env/path",
			expected: "/flag/path",
		},
		{
			name:     "env when no flag",
			flagVal:  ".",
			envVal:   "/env/path",
			expected: "/env/path",
		},
		{
			name:     "default when neither",
			flagVal:  ".",
			envVal:   "",
			expected: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoRoot = tt.flagVal
			if tt.envVal != "" {
				os.Setenv("MB_REPO_ROOT", tt.envVal)
				defer os.Unsetenv("MB_REPO_ROOT")
			}

			result := resolveImportRepoRoot()
			if result != tt.expected {
				t.Errorf("resolveImportRepoRoot() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConsoleProgressReporter(t *testing.T) {
	reporter := &consoleProgressReporter{}

	// Test StartFile
	reporter.StartFile("/path/to/backup.xml", 5, 1)
	if reporter.currentFile != "backup.xml" {
		t.Errorf("Expected currentFile to be 'backup.xml', got %s", reporter.currentFile)
	}
	if reporter.fileCount != 5 {
		t.Errorf("Expected fileCount to be 5, got %d", reporter.fileCount)
	}
	if reporter.fileIndex != 1 {
		t.Errorf("Expected fileIndex to be 1, got %d", reporter.fileIndex)
	}

	// Test UpdateProgress - should output on multiples of 100
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	reporter.UpdateProgress(100, 0)
	reporter.UpdateProgress(50, 0)  // Should not output
	reporter.UpdateProgress(200, 0)

	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "(100 records)") {
		t.Error("Expected output to contain '(100 records)'")
	}
	if !strings.Contains(output, "(200 records)") {
		t.Error("Expected output to contain '(200 records)'")
	}
	if strings.Contains(output, "(50 records)") {
		t.Error("Should not output for non-100 multiples")
	}
}

func TestImportCommandFlags(t *testing.T) {
	// Test that all expected flags are registered

	// Find the import command
	var importSubCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "import" {
			importSubCmd = cmd
			break
		}
	}

	if importSubCmd == nil {
		t.Fatal("Import command not found")
	}

	// Check flags exist
	expectedFlags := []string{
		"dry-run",
		"verbose", 
		"json",
		"filter",
		"no-error-on-rejects",
	}

	for _, flagName := range expectedFlags {
		flag := importSubCmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag %q not found", flagName)
		}
	}

	// Check global flags are available
	globalFlags := []string{
		"repo-root",
		"quiet",
	}

	for _, flagName := range globalFlags {
		// Check in persistent flags (inherited from root)
		flag := rootCmd.PersistentFlags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected global flag %q not found", flagName)
		}
	}
}

func TestDisplaySummary(t *testing.T) {
	// Create a test summary with all entity types
	summary := &importer.ImportSummary{
		Calls: &importer.EntityStats{
			YearStats: map[int]*importer.YearStat{
				2014: {Final: 10, Added: 8, Duplicates: 2},
				2015: {Final: 15, Added: 15, Duplicates: 0},
			},
			Total: &importer.YearStat{
				Initial:    0,
				Final:      25,
				Added:      23,
				Duplicates: 2,
			},
		},
		SMS: &importer.EntityStats{
			YearStats: map[int]*importer.YearStat{
				2014: {Final: 25, Added: 20, Duplicates: 5},
				2015: {Final: 30, Added: 25, Duplicates: 5},
			},
			Total: &importer.YearStat{
				Initial:    0,
				Final:      55,
				Added:      45,
				Duplicates: 10,
			},
		},
		Attachments: &importer.AttachmentStats{
			Total: &importer.AttachmentStat{
				Total:      12,
				New:        10,
				Duplicates: 2,
			},
		},
		Rejections: map[string]*importer.RejectionStats{
			"missing-timestamp": {Count: 1, Reason: "missing-timestamp"},
			"malformed-attachment": {Count: 2, Reason: "malformed-attachment"},
		},
		FilesProcessed: 5,
		Duration:       2 * time.Second,
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Save importFilter
	oldFilter := importFilter
	importFilter = ""
	defer func() { importFilter = oldFilter }()

	// Call displaySummary
	displaySummary(summary, false)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify all entity types are displayed
	expectedStrings := []string{
		"Import Summary:",
		"Calls:",
		"2014: 10 entries (8 new, 2 duplicates)",
		"2015: 15 entries (15 new, 0 duplicates)",
		"Total: 25 entries (23 new, 2 duplicates)",
		"SMS:",
		"2014: 25 entries (20 new, 5 duplicates)",
		"2015: 30 entries (25 new, 5 duplicates)",
		"Total: 55 entries (45 new, 10 duplicates)",
		"Attachments:",
		"Total: 12 files (10 new, 2 duplicates)",
		"Rejections:",
		"Files processed: 5",
		"Time taken: 2.0s",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, but it didn't.\nOutput:\n%s", expected, output)
		}
	}
}

func TestDisplayJSONSummary(t *testing.T) {
	// Create a test summary
	summary := &importer.ImportSummary{
		Calls: &importer.EntityStats{
			YearStats: map[int]*importer.YearStat{
				2014: {Final: 10, Added: 8, Duplicates: 2},
			},
			Total: &importer.YearStat{
				Initial:    0,
				Final:      10,
				Added:      8,
				Duplicates: 2,
			},
		},
		SMS: &importer.EntityStats{
			YearStats: map[int]*importer.YearStat{
				2014: {Final: 25, Added: 20, Duplicates: 5},
			},
			Total: &importer.YearStat{
				Initial:    0,
				Final:      25,
				Added:      20,
				Duplicates: 5,
			},
		},
		Attachments: &importer.AttachmentStats{
			Total: &importer.AttachmentStat{
				Total:      5,
				New:        4,
				Duplicates: 1,
			},
		},
		Rejections: map[string]*importer.RejectionStats{
			"missing-timestamp": {Count: 1, Reason: "missing-timestamp"},
		},
		FilesProcessed: 2,
		Duration:       1 * time.Second,
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call displayJSONSummary
	displayJSONSummary(summary)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify JSON structure contains expected fields
	expectedFields := []string{
		`"files_processed": 2`,
		`"calls"`,
		`"sms"`,
		`"attachments"`,
		`"rejections"`,
		`"total": 5`,
		`"new": 4`,
		`"duplicates": 1`,
	}

	for _, expected := range expectedFields {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected JSON output to contain %q, but it didn't.\nOutput:\n%s", expected, output)
		}
	}
}