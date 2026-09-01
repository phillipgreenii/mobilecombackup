package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phillipgreenii/mobilecombackup/pkg/attachments"
	"github.com/phillipgreenii/mobilecombackup/pkg/validation"
)

// ValidatePathExists checks if a path exists and returns an appropriate error
func ValidatePathExists(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", path)
		}
		return fmt.Errorf("failed to access path: %w", err)
	}
	return nil
}

// ValidateRepositoryPath checks if a path is a valid repository
func ValidateRepositoryPath(path string) error {
	markerFile := filepath.Join(path, ".mobilecombackup.yaml")
	if _, err := os.Stat(markerFile); os.IsNotExist(err) {
		return fmt.Errorf("not a mobilecombackup repository (missing %s)", markerFile)
	}
	return nil
}

// ValidateDirectoryIsEmpty checks if a directory is empty
func ValidateDirectoryIsEmpty(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	if len(entries) > 0 {
		return fmt.Errorf("directory is not empty: contains %d items", len(entries))
	}

	return nil
}

// ClassifyRejectionFile determines the type of rejection file based on its name
func ClassifyRejectionFile(filename string) string {
	if !strings.HasSuffix(filename, ".xml") {
		return ""
	}

	if strings.Contains(filename, "calls") {
		return "calls"
	}
	if strings.Contains(filename, "sms") {
		return "sms"
	}

	return "unknown"
}

// CountRejectionsByType counts rejection files in a directory by type
func CountRejectionsByType(rejectedDir string) (map[string]int, error) {
	counts := make(map[string]int)

	// Check if rejected directory exists
	if _, err := os.Stat(rejectedDir); os.IsNotExist(err) {
		return counts, nil // Return empty map if directory doesn't exist
	}

	// Count rejection files by type
	entries, err := os.ReadDir(rejectedDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read rejected directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileType := ClassifyRejectionFile(entry.Name())
		if fileType != "" && fileType != "unknown" {
			counts[fileType]++
		}
	}

	return counts, nil
}

// CalculateTotalBytesForOrphans calculates the total size of orphaned attachments
func CalculateTotalBytesForOrphans(orphanedAttachments []*attachments.Attachment) int64 {
	var totalBytes int64
	for _, attachment := range orphanedAttachments {
		totalBytes += attachment.Size
	}
	return totalBytes
}

// GroupViolationsByType groups violations by their type for display
func GroupViolationsByType(violations []validation.Violation) map[string][]validation.Violation {
	grouped := make(map[string][]validation.Violation)
	for _, v := range violations {
		grouped[string(v.Type)] = append(grouped[string(v.Type)], v)
	}
	return grouped
}

// IsEmptyDirectory checks if a directory path is empty
func IsEmptyDirectory(dirPath string) (bool, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

// ValidatePathWithinRepository ensures a path is within repository bounds
func ValidatePathWithinRepository(targetPath, repoPath string) error {
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to resolve repository path: %w", err)
	}

	absTargetPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("failed to resolve target path: %w", err)
	}

	// Ensure the path is within repository boundaries
	// Add separator to prevent partial directory name matches
	if !strings.HasPrefix(absTargetPath+string(filepath.Separator), absRepoPath+string(filepath.Separator)) {
		return fmt.Errorf("path %s is outside repository %s", absTargetPath, absRepoPath)
	}

	return nil
}
