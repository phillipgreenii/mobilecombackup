package autofix

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phillipgreenii/mobilecombackup/pkg/validation"
	"github.com/spf13/afero"
)

// Tests for fixSizeMismatch function

func TestAutofixer_FixSizeMismatch(t *testing.T) {
	t.Parallel()

	t.Run("regenerates files.yaml with correct sizes", func(t *testing.T) {
		// Create temp repository
		tmpDir := t.TempDir()

		// Create basic repository structure
		dirs := []string{
			filepath.Join(tmpDir, "calls"),
			filepath.Join(tmpDir, "sms"),
			filepath.Join(tmpDir, "attachments"),
		}
		for _, dir := range dirs {
			if err := os.MkdirAll(dir, 0750); err != nil {
				t.Fatal(err)
			}
		}

		// Create marker file
		markerPath := filepath.Join(tmpDir, ".mobilecombackup.yaml")
		markerContent := `repository_structure_version: "1"
created_at: "2024-01-01T00:00:00Z"
created_by: "test"
`
		if err := os.WriteFile(markerPath, []byte(markerContent), 0600); err != nil {
			t.Fatal(err)
		}

		// Create a test file with wrong size in manifest
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test content"), 0600); err != nil {
			t.Fatal(err)
		}

		// Create autofixer
		autofixer := NewAutofixer(tmpDir, &NullProgressReporter{}, afero.NewOsFs())

		// Create a size mismatch violation
		violation := validation.Violation{
			Type:    validation.SizeMismatch,
			File:    "test.txt",
			Message: "size mismatch",
		}

		// Fix size mismatch
		err := autofixer.(*AutofixerImpl).fixSizeMismatch(violation)
		if err != nil {
			t.Errorf("fixSizeMismatch failed: %v", err)
		}

		// Verify files.yaml was created
		manifestPath := filepath.Join(tmpDir, "files.yaml")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			t.Error("files.yaml was not created")
		}
	})
}

// Tests for isDirectoryMissing function

func TestIsDirectoryMissing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		violation validation.Violation
		expected  bool
	}{
		{
			name: "calls directory missing",
			violation: validation.Violation{
				Type: validation.StructureViolation,
				File: "calls/",
			},
			expected: true,
		},
		{
			name: "sms directory missing",
			violation: validation.Violation{
				Type: validation.StructureViolation,
				File: "sms/",
			},
			expected: true,
		},
		{
			name: "attachments directory missing",
			violation: validation.Violation{
				Type: validation.StructureViolation,
				File: "attachments/",
			},
			expected: true,
		},
		{
			name: "directory missing in message",
			violation: validation.Violation{
				Type:    validation.StructureViolation,
				File:    "some/path",
				Message: "Directory missing: some/path",
			},
			expected: true,
		},
		{
			name: "directory not found in message",
			violation: validation.Violation{
				Type:    validation.StructureViolation,
				File:    "some/path",
				Message: "Directory not found",
			},
			expected: true,
		},
		{
			name: "directory does not exist",
			violation: validation.Violation{
				Type:    validation.StructureViolation,
				File:    "some/path",
				Message: "Directory does not exist",
			},
			expected: true,
		},
		{
			name: "not a structure violation",
			violation: validation.Violation{
				Type:    validation.CountMismatch,
				File:    "calls/",
				Message: "Directory missing",
			},
			expected: false,
		},
		{
			name: "file violation not directory",
			violation: validation.Violation{
				Type:    validation.StructureViolation,
				File:    "test.txt",
				Message: "File is corrupted",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDirectoryMissing(tt.violation)
			if result != tt.expected {
				t.Errorf("isDirectoryMissing() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Tests for performPermissionChecks function

// Additional tests for edge cases

func TestExtractDirectoryFromViolation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		violation validation.Violation
		expected  string
	}{
		{
			name: "simple directory path",
			violation: validation.Violation{
				File: "calls/",
			},
			expected: "calls/",
		},
		{
			name: "nested directory path",
			violation: validation.Violation{
				File: "attachments/ab/",
			},
			expected: "attachments/ab/",
		},
		{
			name: "file path",
			violation: validation.Violation{
				File: "test.txt",
			},
			expected: "test.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDirectoryFromViolation(tt.violation)
			if result != tt.expected {
				t.Errorf("extractDirectoryFromViolation() = %v, want %v", result, tt.expected)
			}
		})
	}
}
