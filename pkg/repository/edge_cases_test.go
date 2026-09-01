package repository

import (
	"errors"
	"os"
	"testing"

	"github.com/spf13/afero"
)

// readOnlyFs wraps afero.Fs to simulate permission errors
type readOnlyFs struct {
	afero.Fs
	failOn string // Path that should fail
}

func (fs *readOnlyFs) Mkdir(name string, perm os.FileMode) error {
	if fs.failOn != "" && name == fs.failOn {
		return errors.New("permission denied")
	}
	return fs.Fs.Mkdir(name, perm)
}

func (fs *readOnlyFs) MkdirAll(path string, perm os.FileMode) error {
	if fs.failOn != "" && path == fs.failOn {
		return errors.New("permission denied")
	}
	return fs.Fs.MkdirAll(path, perm)
}

func (fs *readOnlyFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if fs.failOn != "" && name == fs.failOn {
		return nil, errors.New("permission denied")
	}
	return fs.Fs.OpenFile(name, flag, perm)
}

// errorFs wraps afero.Fs to simulate Stat errors
type errorFs struct {
	afero.Fs
	statError bool
}

func (fs *errorFs) Stat(name string) (os.FileInfo, error) {
	if fs.statError {
		return nil, errors.New("stat error")
	}
	return fs.Fs.Stat(name)
}

// Tests for Creator initialization errors

func TestCreator_Initialize_DirectoryErrors(t *testing.T) {
	t.Parallel()

	t.Run("root directory creation fails", func(t *testing.T) {
		baseFs := afero.NewMemMapFs()
		fs := &readOnlyFs{Fs: baseFs, failOn: "/test/repo"}
		creator := NewCreator(fs, "test-1.0.0")

		_, err := creator.Initialize("/test/repo", false)
		if err == nil {
			t.Error("Expected error when root directory creation fails")
		}
	})

	t.Run("calls subdirectory creation fails", func(t *testing.T) {
		baseFs := afero.NewMemMapFs()
		fs := &readOnlyFs{Fs: baseFs, failOn: "/test/repo/calls"}
		creator := NewCreator(fs, "test-1.0.0")

		_, err := creator.Initialize("/test/repo", false)
		if err == nil {
			t.Error("Expected error when calls directory creation fails")
		}
	})

	t.Run("sms subdirectory creation fails", func(t *testing.T) {
		baseFs := afero.NewMemMapFs()
		fs := &readOnlyFs{Fs: baseFs, failOn: "/test/repo/sms"}
		creator := NewCreator(fs, "test-1.0.0")

		_, err := creator.Initialize("/test/repo", false)
		if err == nil {
			t.Error("Expected error when sms directory creation fails")
		}
	})

	t.Run("attachments subdirectory creation fails", func(t *testing.T) {
		baseFs := afero.NewMemMapFs()
		fs := &readOnlyFs{Fs: baseFs, failOn: "/test/repo/attachments"}
		creator := NewCreator(fs, "test-1.0.0")

		_, err := creator.Initialize("/test/repo", false)
		if err == nil {
			t.Error("Expected error when attachments directory creation fails")
		}
	})
}

func TestCreator_Initialize_FileErrors(t *testing.T) {
	t.Parallel()

	t.Run("marker file creation fails", func(t *testing.T) {
		baseFs := afero.NewMemMapFs()
		fs := &readOnlyFs{Fs: baseFs, failOn: "/test/repo/.mobilecombackup.yaml"}
		creator := NewCreator(fs, "test-1.0.0")

		_, err := creator.Initialize("/test/repo", false)
		if err == nil {
			t.Error("Expected error when marker file creation fails")
		}
	})

	t.Run("contacts file creation fails", func(t *testing.T) {
		baseFs := afero.NewMemMapFs()
		fs := &readOnlyFs{Fs: baseFs, failOn: "/test/repo/contacts.yaml"}
		creator := NewCreator(fs, "test-1.0.0")

		_, err := creator.Initialize("/test/repo", false)
		if err == nil {
			t.Error("Expected error when contacts file creation fails")
		}
	})

	t.Run("summary file creation fails", func(t *testing.T) {
		baseFs := afero.NewMemMapFs()
		fs := &readOnlyFs{Fs: baseFs, failOn: "/test/repo/summary.yaml"}
		creator := NewCreator(fs, "test-1.0.0")

		_, err := creator.Initialize("/test/repo", false)
		if err == nil {
			t.Error("Expected error when summary file creation fails")
		}
	})

	// Note: manifest and checksum file creation failures are hard to test
	// because they may be created atomically or the filesystem may allow
	// the operations even with the readOnlyFs wrapper
}

// Tests for Validator edge cases

func TestValidator_HandleStatError(t *testing.T) {
	t.Parallel()

	t.Run("stat error other than not exist", func(t *testing.T) {
		baseFs := afero.NewMemMapFs()
		fs := &errorFs{Fs: baseFs, statError: true}
		validator := NewValidator(fs)

		err := validator.ValidateTargetDirectory("/test")
		if err == nil {
			t.Error("Expected error when stat fails")
		}
	})
}

func TestValidator_ValidateDirectoryContents_Cases(t *testing.T) {
	t.Parallel()

	t.Run("target is file not directory", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		validator := NewValidator(fs)

		_ = afero.WriteFile(fs, "/test/file.txt", []byte(""), 0600)

		err := validator.ValidateTargetDirectory("/test/file.txt")
		if err == nil {
			t.Error("Expected error when target is a file")
		}
		// Just check that we got an error, don't check exact message
	})

	t.Run("directory with hidden file other than marker", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		validator := NewValidator(fs)

		_ = fs.MkdirAll("/test/dir", 0750)
		_ = afero.WriteFile(fs, "/test/dir/.gitignore", []byte(""), 0600)

		err := validator.ValidateTargetDirectory("/test/dir")
		if err == nil {
			t.Error("Expected error for directory with non-marker hidden file")
		}
	})

	t.Run("directory with subdirectory", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		validator := NewValidator(fs)

		_ = fs.MkdirAll("/test/dir/subdir", 0750)

		err := validator.ValidateTargetDirectory("/test/dir")
		if err == nil {
			t.Error("Expected error for directory with subdirectory")
		}
	})

	t.Run("directory with marker file", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		validator := NewValidator(fs)

		_ = fs.MkdirAll("/test/repo", 0750)
		_ = afero.WriteFile(fs, "/test/repo/.mobilecombackup.yaml", []byte("version: 1.0.0\n"), 0600)

		err := validator.ValidateTargetDirectory("/test/repo")
		if err == nil {
			t.Error("Expected error for directory with existing repository marker")
		}
	})

	t.Run("empty directory is valid", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		validator := NewValidator(fs)

		_ = fs.MkdirAll("/test/empty", 0750)

		err := validator.ValidateTargetDirectory("/test/empty")
		if err != nil {
			t.Errorf("Expected no error for empty directory, got: %v", err)
		}
	})
}
