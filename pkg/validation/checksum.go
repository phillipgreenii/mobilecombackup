package validation

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ChecksumValidator validates SHA-256 checksums for repository files
type ChecksumValidator interface {
	// CalculateFileChecksum calculates SHA-256 for a file
	CalculateFileChecksum(filePath string) (string, error)
	
	// VerifyFileChecksum verifies a file matches expected SHA-256
	VerifyFileChecksum(filePath string, expectedChecksum string) error
	
	// ValidateManifestChecksums verifies all files in manifest have correct checksums
	ValidateManifestChecksums(manifest *FileManifest) []ValidationViolation
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
	file, err := os.Open(filePath)
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
func (v *ChecksumValidatorImpl) ValidateManifestChecksums(manifest *FileManifest) []ValidationViolation {
	var violations []ValidationViolation
	
	for _, entry := range manifest.Files {
		fullPath := filepath.Join(v.repositoryRoot, entry.File)
		
		// Check if file exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			violations = append(violations, ValidationViolation{
				Type:     MissingFile,
				Severity: SeverityError,
				File:     entry.File,
				Message:  fmt.Sprintf("File not found: %s", entry.File),
			})
			continue
		}
		
		// Calculate actual checksum
		actualChecksum, err := v.CalculateFileChecksum(fullPath)
		if err != nil {
			violations = append(violations, ValidationViolation{
				Type:     ChecksumMismatch,
				Severity: SeverityError,
				File:     entry.File,
				Message:  fmt.Sprintf("Failed to calculate checksum for %s: %v", entry.File, err),
			})
			continue
		}
		
		// Verify checksum matches
		if actualChecksum != entry.SHA256 {
			violations = append(violations, ValidationViolation{
				Type:     ChecksumMismatch,
				Severity: SeverityError,
				File:     entry.File,
				Message:  fmt.Sprintf("Checksum mismatch for %s", entry.File),
				Expected: entry.SHA256,
				Actual:   actualChecksum,
			})
		}
		
		// Verify file size matches (if we can get it)
		if fileInfo, err := os.Stat(fullPath); err == nil {
			actualSize := fileInfo.Size()
			if actualSize != entry.SizeBytes {
				violations = append(violations, ValidationViolation{
					Type:     SizeMismatch,
					Severity: SeverityError,
					File:     entry.File,
					Message:  fmt.Sprintf("Size mismatch for %s", entry.File),
					Expected: fmt.Sprintf("%d", entry.SizeBytes),
					Actual:   fmt.Sprintf("%d", actualSize),
				})
			}
		}
	}
	
	return violations
}