package validation

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ChecksumValidator validates SHA-256 checksums for repository files
type ChecksumValidator interface {
	// CalculateFileChecksum calculates SHA-256 for a file
	CalculateFileChecksum(filePath string) (string, error)

	// VerifyFileChecksum verifies a file matches expected SHA-256
	VerifyFileChecksum(filePath string, expectedChecksum string) error

	// ValidateManifestChecksums verifies all files in manifest have correct checksums
	ValidateManifestChecksums(manifest *FileManifest) []Violation
}

// ChecksumValidatorImpl implements ChecksumValidator interface
type ChecksumValidatorImpl struct {
	repositoryRoot string
}

// NewChecksumValidator creates a new checksum validator
func NewChecksumValidator(repositoryRoot string) ChecksumValidator {
	return &ChecksumValidatorImpl{
		repositoryRoot: repositoryRoot,
	}
}

// CalculateFileChecksum calculates SHA-256 for a file
func (v *ChecksumValidatorImpl) CalculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath) // nolint:gosec // Validation requires file access
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to calculate SHA-256 for %s: %w", filePath, err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// VerifyFileChecksum verifies a file matches expected SHA-256
func (v *ChecksumValidatorImpl) VerifyFileChecksum(filePath string, expectedChecksum string) error {
	actualChecksum, err := v.CalculateFileChecksum(filePath)
	if err != nil {
		return err
	}

	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s",
			filePath, expectedChecksum, actualChecksum)
	}

	return nil
}

// ValidateManifestChecksums verifies all files in manifest have correct checksums
func (v *ChecksumValidatorImpl) ValidateManifestChecksums(manifest *FileManifest) []Violation {
	var violations []Violation

	for _, entry := range manifest.Files {
		fullPath := filepath.Join(v.repositoryRoot, entry.Name)

		// Check if file exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			violations = append(violations, Violation{
				Type:     MissingFile,
				Severity: SeverityError,
				File:     entry.Name,
				Message:  fmt.Sprintf("File not found: %s", entry.Name),
			})
			continue
		}

		// Calculate actual checksum
		actualChecksum, err := v.CalculateFileChecksum(fullPath)
		if err != nil {
			violations = append(violations, Violation{
				Type:     ChecksumMismatch,
				Severity: SeverityError,
				File:     entry.Name,
				Message:  fmt.Sprintf("Failed to calculate checksum for %s: %v", entry.Name, err),
			})
			continue
		}

		// Extract hex checksum from sha256:xxx format
		expectedChecksum := strings.TrimPrefix(entry.Checksum, "sha256:")

		// Verify checksum matches
		if actualChecksum != expectedChecksum {
			violations = append(violations, Violation{
				Type:     ChecksumMismatch,
				Severity: SeverityError,
				File:     entry.Name,
				Message:  fmt.Sprintf("Checksum mismatch for %s", entry.Name),
				Expected: expectedChecksum,
				Actual:   actualChecksum,
			})
		}

		// Verify file size matches (if we can get it)
		if fileInfo, err := os.Stat(fullPath); err == nil {
			actualSize := fileInfo.Size()
			if actualSize != entry.Size {
				violations = append(violations, Violation{
					Type:     SizeMismatch,
					Severity: SeverityError,
					File:     entry.Name,
					Message:  fmt.Sprintf("Size mismatch for %s", entry.Name),
					Expected: fmt.Sprintf("%d", entry.Size),
					Actual:   fmt.Sprintf("%d", actualSize),
				})
			}
		}
	}

	return violations
}
