package repository

import (
	"testing"

	"github.com/spf13/afero"
)

func TestValidator_ValidateTargetDirectory(t *testing.T) {
	t.Run("accepts non-existent directory", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		validator := NewValidator(fs)

		err := validator.ValidateTargetDirectory("/test/repo")
		if err != nil {
			t.Errorf("Expected no error for non-existent directory, got: %v", err)
		}
	})

	t.Run("accepts empty directory", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		validator := NewValidator(fs)

		// Create empty directory
		_ = fs.MkdirAll("/test/repo", 0750)

		err := validator.ValidateTargetDirectory("/test/repo")
		if err != nil {
			t.Errorf("Expected no error for empty directory, got: %v", err)
		}
	})

	t.Run("rejects non-empty directory", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		validator := NewValidator(fs)

		// Create directory with file
		_ = fs.MkdirAll("/test/repo", 0750)
		_ = afero.WriteFile(fs, "/test/repo/file.txt", []byte("test"), 0600)

		err := validator.ValidateTargetDirectory("/test/repo")
		if err == nil {
			t.Error("Expected error for non-empty directory")
		}
		if err.Error() != "directory is not empty" {
			t.Errorf("Expected 'directory is not empty' error, got: %v", err)
		}
	})

	t.Run("rejects path that is a file", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		validator := NewValidator(fs)

		// Create file instead of directory
		_ = fs.MkdirAll("/test", 0750)
		_ = afero.WriteFile(fs, "/test/file.txt", []byte("test"), 0600)

		err := validator.ValidateTargetDirectory("/test/file.txt")
		if err == nil {
			t.Error("Expected error for file path")
		}
	})

	t.Run("rejects directory with .mobilecombackup.yaml", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		validator := NewValidator(fs)

		// Create directory with marker file
		_ = fs.MkdirAll("/test/repo", 0750)
		_ = afero.WriteFile(fs, "/test/repo/.mobilecombackup.yaml", []byte("test"), 0600)

		err := validator.ValidateTargetDirectory("/test/repo")
		if err == nil {
			t.Error("Expected error for existing repository")
		}
		if !contains(err.Error(), "already contains a mobilecombackup repository") {
			t.Errorf("Expected 'already contains repository' error, got: %v", err)
		}
	})

	t.Run("rejects directory with calls/ subdirectory", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		validator := NewValidator(fs)

		// Create directory with calls subdirectory
		_ = fs.MkdirAll("/test/repo/calls", 0750)

		err := validator.ValidateTargetDirectory("/test/repo")
		if err == nil {
			t.Error("Expected error for directory with calls/")
		}
		if !contains(err.Error(), "appears to be a repository") {
			t.Errorf("Expected 'appears to be a repository' error, got: %v", err)
		}
	})

	t.Run("rejects directory with sms/ subdirectory", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		validator := NewValidator(fs)

		// Create directory with sms subdirectory
		_ = fs.MkdirAll("/test/repo/sms", 0750)

		err := validator.ValidateTargetDirectory("/test/repo")
		if err == nil {
			t.Error("Expected error for directory with sms/")
		}
	})

	t.Run("rejects directory with attachments/ subdirectory", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		validator := NewValidator(fs)

		// Create directory with attachments subdirectory
		_ = fs.MkdirAll("/test/repo/attachments", 0750)

		err := validator.ValidateTargetDirectory("/test/repo")
		if err == nil {
			t.Error("Expected error for directory with attachments/")
		}
	})
}

func TestIsRepositoryDirectory(t *testing.T) {
	tests := []struct {
		name     string
		dirName  string
		expected bool
	}{
		{"calls directory", "calls", true},
		{"sms directory", "sms", true},
		{"attachments directory", "attachments", true},
		{"other directory", "other", false},
		{"data directory", "data", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRepositoryDirectory(tt.dirName)
			if result != tt.expected {
				t.Errorf("IsRepositoryDirectory(%q) = %v; want %v", tt.dirName, result, tt.expected)
			}
		})
	}
}
