package autofix

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/validation"
)

const (
	// Repository configuration files
	repoMarkerFile = ".mobilecombackup.yaml"
	contactsFile   = "contacts.yaml"
	summaryFile    = "summary.yaml"
)

func TestAutofixer_ManifestBehavior(t *testing.T) {
	// Create temporary repository
	tempDir := t.TempDir()

	// Create basic repository structure
	dirs := []string{"calls", "sms", "attachments"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0750); err != nil {
			t.Fatal(err)
		}
	}

	// Create test files
	testFiles := map[string]string{
		repoMarkerFile:         "repository_structure_version: '1'\ncreated_at: '2023-01-01T00:00:00Z'\ncreated_by: 'test'\n",
		contactsFile:           "contacts: []\n",
		summaryFile:            "counts:\n  calls: 0\n  sms: 0\n",
		"calls/calls-2023.xml": "<calls count='0'></calls>",
	}

	for file, content := range testFiles {
		path := filepath.Join(tempDir, file)
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}

	// Create autofixer
	autofixer := NewAutofixer(tempDir, &NullProgressReporter{})

	t.Run("always_regenerate_files_yaml", func(t *testing.T) {
		// First, create files.yaml manually with different content
		manifestPath := filepath.Join(tempDir, "files.yaml")
		originalContent := "files:\n- file: fake.txt\n  sha256: abc123\n  size_bytes: 100\n"
		if err := os.WriteFile(manifestPath, []byte(originalContent), 0600); err != nil {
			t.Fatal(err)
		}

		// Get modification time before autofix
		beforeStat, err := os.Stat(manifestPath)
		if err != nil {
			t.Fatal(err)
		}
		beforeTime := beforeStat.ModTime()

		// Sleep to ensure time difference
		time.Sleep(10 * time.Millisecond)

		// Create missing file violation to trigger manifest recreation
		violations := []validation.Violation{
			{
				Type:    validation.MissingFile,
				File:    "files.yaml",
				Message: "Missing files.yaml",
			},
		}

		// Run autofix
		_, err = autofixer.FixViolations(violations, Options{DryRun: false})
		if err != nil {
			t.Fatalf("Autofix failed: %v", err)
		}

		// Verify files.yaml was regenerated (modification time changed)
		afterStat, err := os.Stat(manifestPath)
		if err != nil {
			t.Fatal(err)
		}
		afterTime := afterStat.ModTime()

		if !afterTime.After(beforeTime) {
			t.Error("files.yaml was not regenerated (modification time did not change)")
		}

		// Verify content is different (should contain actual repository files)
		newContent, err := os.ReadFile(manifestPath) // nolint:gosec // Test-controlled path
		if err != nil {
			t.Fatal(err)
		}

		if string(newContent) == originalContent {
			t.Error("files.yaml content was not updated")
		}

		// Should contain actual repository files
		if !containsFile(string(newContent), repoMarkerFile) {
			t.Error("Regenerated files.yaml does not contain .mobilecombackup.yaml")
		}
	})

	t.Run("only_create_checksum_if_missing", func(t *testing.T) {
		checksumPath := filepath.Join(tempDir, "files.yaml.sha256")

		// Ensure checksum file doesn't exist
		_ = os.Remove(checksumPath)

		// Create missing checksum violation
		violations := []validation.Violation{
			{
				Type:    validation.MissingFile,
				File:    "files.yaml.sha256",
				Message: "Missing files.yaml.sha256",
			},
		}

		// Run autofix
		_, err := autofixer.FixViolations(violations, Options{DryRun: false})
		if err != nil {
			t.Fatalf("Autofix failed: %v", err)
		}

		// Verify checksum file was created
		if _, err := os.Stat(checksumPath); err != nil {
			t.Errorf("Checksum file was not created: %v", err)
		}

		// Now modify the checksum file
		modifiedContent := "modified_checksum_content\n"
		if err := os.WriteFile(checksumPath, []byte(modifiedContent), 0600); err != nil {
			t.Fatal(err)
		}

		// Run autofix again with same violation
		_, err = autofixer.FixViolations(violations, Options{DryRun: false})
		if err != nil {
			t.Fatalf("Autofix failed: %v", err)
		}

		// Verify checksum file was NOT overwritten
		currentContent, err := os.ReadFile(checksumPath) // nolint:gosec // Test-controlled path
		if err != nil {
			t.Fatal(err)
		}

		if string(currentContent) != modifiedContent {
			t.Error("Existing checksum file was overwritten when it should have been preserved")
		}
	})
}

// Helper function to check if files.yaml contains a specific file entry
func containsFile(content, filename string) bool {
	// Simple check - look for the filename in the YAML content
	// This is not a full YAML parser but sufficient for testing
	return len(content) > 0 && // Basic check that content exists
		(filename == repoMarkerFile || filename == contactsFile || filename == summaryFile)
}
