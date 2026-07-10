package repository

import (
	"fmt"
	"os"

	"github.com/spf13/afero"
)

// Validator handles validation of repository paths and contents
type Validator struct {
	fs afero.Fs
}

// NewValidator creates a new repository validator
func NewValidator(fs afero.Fs) *Validator {
	return &Validator{fs: fs}
}

// ValidateTargetDirectory validates that a directory is suitable for repository initialization
func (v *Validator) ValidateTargetDirectory(path string) error {
	info, err := v.fs.Stat(path)
	if err != nil {
		return v.handleStatError(err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path exists but is not a directory: %s", path)
	}

	return v.validateDirectoryContents(path)
}

// handleStatError handles errors from fs.Stat
func (v *Validator) handleStatError(err error) error {
	if os.IsNotExist(err) {
		// Directory doesn't exist - this is OK, we'll create it
		return nil
	}
	return fmt.Errorf("failed to check directory: %w", err)
}

// validateDirectoryContents validates that the directory is suitable for initialization
func (v *Validator) validateDirectoryContents(path string) error {
	entries, err := afero.ReadDir(v.fs, path)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	if err := v.checkForExistingRepository(entries); err != nil {
		return err
	}

	// Warn if directory is not empty
	if len(entries) > 0 {
		return fmt.Errorf("directory is not empty")
	}

	return nil
}

// checkForExistingRepository checks if directory already contains a repository
func (v *Validator) checkForExistingRepository(entries []os.FileInfo) error {
	for _, entry := range entries {
		if entry.Name() == ".mobilecombackup.yaml" {
			return fmt.Errorf("directory already contains a mobilecombackup repository")
		}
		if IsRepositoryDirectory(entry.Name()) {
			return fmt.Errorf("directory appears to be a repository (found %s/ directory)", entry.Name())
		}
	}
	return nil
}

// IsRepositoryDirectory checks if a directory name indicates a repository structure
func IsRepositoryDirectory(name string) bool {
	return name == CallsDir || name == SMSDir || name == AttachmentsDir
}
