package validation

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// Manifest file names
	manifestFileName         = "files.yaml"
	manifestChecksumFileName = "files.yaml.sha256"
)

// ManifestValidatorImpl implements ManifestValidator interface
type ManifestValidatorImpl struct {
	repositoryRoot string
}

// NewManifestValidator creates a new manifest validator
func NewManifestValidator(repositoryRoot string) ManifestValidator {
	return &ManifestValidatorImpl{
		repositoryRoot: repositoryRoot,
	}
}

// LoadManifest reads and parses files.yaml
func (v *ManifestValidatorImpl) LoadManifest() (*FileManifest, error) {
	manifestPath := filepath.Join(v.repositoryRoot, manifestFileName)

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read files.yaml: %w", err)
	}

	var manifest FileManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse files.yaml: %w", err)
	}

	return &manifest, nil
}

// ValidateManifestFormat checks files.yaml structure and format
func (v *ManifestValidatorImpl) ValidateManifestFormat(manifest *FileManifest) []ValidationViolation {
	var violations []ValidationViolation

	// Check for duplicate entries
	seenFiles := make(map[string]bool)

	// SHA-256 regex pattern for checksum format (sha256:64hexchars)
	sha256Pattern := regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

	for i, entry := range manifest.Files {
		entryContext := fmt.Sprintf("files.yaml entry %d", i+1)

		// Check for duplicate file paths
		if seenFiles[entry.Name] {
			violations = append(violations, ValidationViolation{
				Type:     InvalidFormat,
				Severity: SeverityError,
				File:     manifestFileName,
				Message:  fmt.Sprintf("Duplicate file entry: %s", entry.Name),
			})
		}
		seenFiles[entry.Name] = true

		// Validate SHA-256 format
		if !sha256Pattern.MatchString(entry.Checksum) {
			violations = append(violations, ValidationViolation{
				Type:     InvalidFormat,
				Severity: SeverityError,
				File:     entryContext,
				Message:  fmt.Sprintf("Invalid SHA-256 format for file %s: %s", entry.Name, entry.Checksum),
			})
		}

		// Validate positive size
		if entry.Size <= 0 {
			violations = append(violations, ValidationViolation{
				Type:     InvalidFormat,
				Severity: SeverityError,
				File:     entryContext,
				Message:  fmt.Sprintf("Invalid size_bytes for file %s: %d (must be positive)", entry.Name, entry.Size),
			})
		}

		// Validate relative path (no ".." or absolute paths)
		if filepath.IsAbs(entry.Name) {
			violations = append(violations, ValidationViolation{
				Type:     InvalidFormat,
				Severity: SeverityError,
				File:     entryContext,
				Message:  fmt.Sprintf("File path must be relative: %s", entry.Name),
			})
		}

		if strings.Contains(entry.Name, "..") {
			violations = append(violations, ValidationViolation{
				Type:     InvalidFormat,
				Severity: SeverityError,
				File:     entryContext,
				Message:  fmt.Sprintf("File path contains '..': %s", entry.Name),
			})
		}

		// Validate file path doesn't include files.yaml or files.yaml.sha256
		if entry.Name == manifestFileName || entry.Name == manifestChecksumFileName {
			violations = append(violations, ValidationViolation{
				Type:     InvalidFormat,
				Severity: SeverityError,
				File:     entryContext,
				Message:  fmt.Sprintf("files.yaml should not include itself or its checksum: %s", entry.Name),
			})
		}
	}

	return violations
}

// CheckManifestCompleteness verifies all files are listed
func (v *ManifestValidatorImpl) CheckManifestCompleteness(manifest *FileManifest) []ValidationViolation {
	var violations []ValidationViolation

	// Build set of files in manifest
	manifestFiles := make(map[string]bool)
	for _, entry := range manifest.Files {
		manifestFiles[entry.Name] = true
	}

	// Walk repository and find all files (excluding files.yaml and files.yaml.sha256)
	var actualFiles []string
	err := filepath.Walk(v.repositoryRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(v.repositoryRoot, path)
		if err != nil {
			return err
		}

		// Skip files.yaml and files.yaml.sha256
		if relPath == manifestFileName || relPath == manifestChecksumFileName {
			return nil
		}

		actualFiles = append(actualFiles, relPath)
		return nil
	})

	if err != nil {
		violations = append(violations, ValidationViolation{
			Type:     StructureViolation,
			Severity: SeverityError,
			File:     v.repositoryRoot,
			Message:  fmt.Sprintf("Failed to walk repository directory: %v", err),
		})
		return violations
	}

	// Check for files in repository not in manifest
	for _, file := range actualFiles {
		if !manifestFiles[file] {
			violations = append(violations, ValidationViolation{
				Type:     ExtraFile,
				Severity: SeverityError,
				File:     file,
				Message:  fmt.Sprintf("File exists in repository but not listed in files.yaml: %s", file),
			})
		}
	}

	// Check for files in manifest not in repository
	actualFileSet := make(map[string]bool)
	for _, file := range actualFiles {
		actualFileSet[file] = true
	}

	for _, entry := range manifest.Files {
		if !actualFileSet[entry.Name] {
			violations = append(violations, ValidationViolation{
				Type:     MissingFile,
				Severity: SeverityError,
				File:     entry.Name,
				Message:  fmt.Sprintf("File listed in files.yaml but not found in repository: %s", entry.Name),
			})
		}
	}

	return violations
}

// VerifyManifestChecksum validates files.yaml.sha256
func (v *ManifestValidatorImpl) VerifyManifestChecksum() error {
	manifestPath := filepath.Join(v.repositoryRoot, manifestFileName)
	checksumPath := filepath.Join(v.repositoryRoot, manifestChecksumFileName)

	// Check if checksum file exists
	if _, err := os.Stat(checksumPath); os.IsNotExist(err) {
		return fmt.Errorf("files.yaml.sha256 not found")
	}

	// Read expected checksum
	expectedData, err := os.ReadFile(checksumPath)
	if err != nil {
		return fmt.Errorf("failed to read files.yaml.sha256: %w", err)
	}

	expectedLine := strings.TrimSpace(string(expectedData))

	// Parse SHA256 checksum format: "hash  filename"
	parts := strings.Fields(expectedLine)
	if len(parts) != 2 {
		return fmt.Errorf("files.yaml.sha256 contains invalid format: %s", expectedLine)
	}

	expected := parts[0]
	expectedFilename := parts[1]

	// Validate that it's checking the correct file
	if expectedFilename != manifestFileName {
		return fmt.Errorf("files.yaml.sha256 should reference files.yaml, but references: %s", expectedFilename)
	}

	// Validate checksum format
	sha256Pattern := regexp.MustCompile(`^[0-9a-f]{64}$`)
	if !sha256Pattern.MatchString(expected) {
		return fmt.Errorf("files.yaml.sha256 contains invalid SHA-256 hash format: %s", expected)
	}

	// Calculate actual checksum
	file, err := os.Open(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to open files.yaml: %w", err)
	}
	defer func() { _ = file.Close() }()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("failed to calculate SHA-256 of files.yaml: %w", err)
	}

	actual := fmt.Sprintf("%x", hasher.Sum(nil))

	// Compare checksums
	if expected != actual {
		return fmt.Errorf("files.yaml checksum mismatch: expected %s, got %s", expected, actual)
	}

	return nil
}
