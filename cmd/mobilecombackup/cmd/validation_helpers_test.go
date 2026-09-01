package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phillipgreenii/mobilecombackup/pkg/attachments"
	"github.com/phillipgreenii/mobilecombackup/pkg/validation"
)

func TestValidatePathExists(t *testing.T) {
	t.Run("existing path", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := ValidatePathExists(tmpDir)
		if err != nil {
			t.Errorf("Expected no error for existing path, got: %v", err)
		}
	})

	t.Run("non-existent path", func(t *testing.T) {
		err := ValidatePathExists("/nonexistent/path/that/should/not/exist")
		if err == nil {
			t.Error("Expected error for non-existent path")
		}
		if !os.IsNotExist(err) {
			// Should contain "does not exist"
			if err.Error() != "path does not exist: /nonexistent/path/that/should/not/exist" {
				t.Errorf("Expected 'does not exist' error, got: %v", err)
			}
		}
	})
}

func TestValidateRepositoryPath(t *testing.T) {
	t.Run("valid repository", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create marker file
		markerFile := filepath.Join(tmpDir, ".mobilecombackup.yaml")
		if err := os.WriteFile(markerFile, []byte("version: 1.0\n"), 0600); err != nil {
			t.Fatal(err)
		}

		err := ValidateRepositoryPath(tmpDir)
		if err != nil {
			t.Errorf("Expected no error for valid repository, got: %v", err)
		}
	})

	t.Run("invalid repository - missing marker", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := ValidateRepositoryPath(tmpDir)
		if err == nil {
			t.Error("Expected error for invalid repository")
		}
	})
}

func TestValidateDirectoryIsEmpty(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := ValidateDirectoryIsEmpty(tmpDir)
		if err != nil {
			t.Errorf("Expected no error for empty directory, got: %v", err)
		}
	})

	t.Run("non-empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a file
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}

		err := ValidateDirectoryIsEmpty(tmpDir)
		if err == nil {
			t.Error("Expected error for non-empty directory")
		}
	})

	t.Run("non-existent directory", func(t *testing.T) {
		err := ValidateDirectoryIsEmpty("/nonexistent/directory")
		if err == nil {
			t.Error("Expected error for non-existent directory")
		}
	})
}

func TestClassifyRejectionFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"calls file", "rejected-calls-2024.xml", "calls"},
		{"sms file", "rejected-sms-2024.xml", "sms"},
		{"calls with timestamp", "calls-rejected-20240101.xml", "calls"},
		{"sms with timestamp", "sms-rejected-20240101.xml", "sms"},
		{"non-xml file", "rejected-calls.txt", ""},
		{"no type indicator", "rejected-2024.xml", "unknown"},
		{"empty filename", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyRejectionFile(tt.filename)
			if result != tt.expected {
				t.Errorf("ClassifyRejectionFile(%q) = %q; want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestCountRejectionsByType(t *testing.T) {
	t.Run("no rejected directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		rejectedDir := filepath.Join(tmpDir, "rejected")

		counts, err := CountRejectionsByType(rejectedDir)
		if err != nil {
			t.Errorf("Expected no error when directory doesn't exist, got: %v", err)
		}
		if len(counts) != 0 {
			t.Errorf("Expected empty map, got: %v", counts)
		}
	})

	t.Run("with rejection files", func(t *testing.T) {
		tmpDir := t.TempDir()
		rejectedDir := filepath.Join(tmpDir, "rejected")
		if err := os.Mkdir(rejectedDir, 0750); err != nil {
			t.Fatal(err)
		}

		// Create test files
		files := []string{
			"rejected-calls-2024-01.xml",
			"rejected-calls-2024-02.xml",
			"rejected-sms-2024-01.xml",
			"rejected-other.txt", // Should be ignored
		}

		for _, file := range files {
			path := filepath.Join(rejectedDir, file)
			if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
				t.Fatal(err)
			}
		}

		counts, err := CountRejectionsByType(rejectedDir)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if counts["calls"] != 2 {
			t.Errorf("Expected 2 calls rejections, got: %d", counts["calls"])
		}
		if counts["sms"] != 1 {
			t.Errorf("Expected 1 sms rejection, got: %d", counts["sms"])
		}
	})

	t.Run("with subdirectories", func(t *testing.T) {
		tmpDir := t.TempDir()
		rejectedDir := filepath.Join(tmpDir, "rejected")
		if err := os.Mkdir(rejectedDir, 0750); err != nil {
			t.Fatal(err)
		}

		// Create a subdirectory (should be ignored)
		subDir := filepath.Join(rejectedDir, "subdir")
		if err := os.Mkdir(subDir, 0750); err != nil {
			t.Fatal(err)
		}

		// Create a file
		file := filepath.Join(rejectedDir, "rejected-calls-2024.xml")
		if err := os.WriteFile(file, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}

		counts, err := CountRejectionsByType(rejectedDir)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if counts["calls"] != 1 {
			t.Errorf("Expected 1 calls rejection, got: %d", counts["calls"])
		}
	})
}

func TestCalculateTotalBytesForOrphans(t *testing.T) {
	t.Run("no orphans", func(t *testing.T) {
		total := CalculateTotalBytesForOrphans([]*attachments.Attachment{})
		if total != 0 {
			t.Errorf("Expected 0 bytes for empty list, got: %d", total)
		}
	})

	t.Run("multiple orphans", func(t *testing.T) {
		orphans := []*attachments.Attachment{
			{Path: "file1.jpg", Size: 1024},
			{Path: "file2.jpg", Size: 2048},
			{Path: "file3.jpg", Size: 512},
		}

		total := CalculateTotalBytesForOrphans(orphans)
		expected := int64(1024 + 2048 + 512)
		if total != expected {
			t.Errorf("Expected %d bytes, got: %d", expected, total)
		}
	})

	t.Run("with zero-size attachments", func(t *testing.T) {
		orphans := []*attachments.Attachment{
			{Path: "file1.jpg", Size: 1024},
			{Path: "file2.jpg", Size: 0},
			{Path: "file3.jpg", Size: 512},
		}

		total := CalculateTotalBytesForOrphans(orphans)
		expected := int64(1024 + 512)
		if total != expected {
			t.Errorf("Expected %d bytes, got: %d", expected, total)
		}
	})
}

func TestGroupViolationsByType(t *testing.T) {
	t.Run("no violations", func(t *testing.T) {
		grouped := GroupViolationsByType([]validation.Violation{})
		if len(grouped) != 0 {
			t.Errorf("Expected empty map for no violations, got: %v", grouped)
		}
	})

	t.Run("multiple violation types", func(t *testing.T) {
		violations := []validation.Violation{
			{Type: validation.MissingFile, File: "file1.xml", Message: "missing"},
			{Type: validation.MissingFile, File: "file2.xml", Message: "missing"},
			{Type: validation.ChecksumMismatch, File: "file3.xml", Message: "checksum"},
			{Type: validation.StructureViolation, File: "dir/", Message: "structure"},
		}

		grouped := GroupViolationsByType(violations)

		if len(grouped) != 3 {
			t.Errorf("Expected 3 groups, got: %d", len(grouped))
		}

		if len(grouped[string(validation.MissingFile)]) != 2 {
			t.Errorf("Expected 2 MissingFile violations, got: %d", len(grouped[string(validation.MissingFile)]))
		}

		if len(grouped[string(validation.ChecksumMismatch)]) != 1 {
			t.Errorf("Expected 1 ChecksumMismatch violation, got: %d", len(grouped[string(validation.ChecksumMismatch)]))
		}

		if len(grouped[string(validation.StructureViolation)]) != 1 {
			t.Errorf("Expected 1 StructureViolation violation, got: %d", len(grouped[string(validation.StructureViolation)]))
		}
	})
}

func TestIsEmptyDirectory(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		isEmpty, err := IsEmptyDirectory(tmpDir)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if !isEmpty {
			t.Error("Expected directory to be empty")
		}
	})

	t.Run("non-empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a file
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}

		isEmpty, err := IsEmptyDirectory(tmpDir)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if isEmpty {
			t.Error("Expected directory to be non-empty")
		}
	})

	t.Run("non-existent directory", func(t *testing.T) {
		_, err := IsEmptyDirectory("/nonexistent/directory")
		if err == nil {
			t.Error("Expected error for non-existent directory")
		}
	})
}

func TestValidatePathWithinRepository(t *testing.T) {
	t.Run("path within repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		subPath := filepath.Join(tmpDir, "sub", "path")
		if err := os.MkdirAll(subPath, 0750); err != nil {
			t.Fatal(err)
		}

		err := ValidatePathWithinRepository(subPath, tmpDir)
		if err != nil {
			t.Errorf("Expected no error for path within repository, got: %v", err)
		}
	})

	t.Run("path outside repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		outsidePath := filepath.Join(os.TempDir(), "outside")

		err := ValidatePathWithinRepository(outsidePath, tmpDir)
		if err == nil {
			t.Error("Expected error for path outside repository")
		}
	})

	t.Run("path is repository itself", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := ValidatePathWithinRepository(tmpDir, tmpDir)
		if err != nil {
			t.Errorf("Expected no error for repository path itself, got: %v", err)
		}
	})

	t.Run("relative paths", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create subdirectory
		subPath := filepath.Join(tmpDir, "sub")
		if err := os.MkdirAll(subPath, 0750); err != nil {
			t.Fatal(err)
		}

		// Test with relative path
		relPath := filepath.Join(tmpDir, "sub", "..", "sub")
		err := ValidatePathWithinRepository(relPath, tmpDir)
		if err != nil {
			t.Errorf("Expected no error for relative path within repository, got: %v", err)
		}
	})

	t.Run("path traversal attempt", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Attempt to escape using ../
		escapeAttempt := filepath.Join(tmpDir, "..", "..", "etc", "passwd")

		err := ValidatePathWithinRepository(escapeAttempt, tmpDir)
		if err == nil {
			t.Error("Expected error for path traversal attempt")
		}
	})
}
