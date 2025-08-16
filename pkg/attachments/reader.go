package attachments

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// Compile regex once at package initialization
	dirNameRegex = regexp.MustCompile("^[0-9a-f]{2}$")
)

// AttachmentManager provides attachment management functionality
type AttachmentManager struct {
	repoPath string
}

// NewAttachmentManager creates a new AttachmentManager for the given repository path
func NewAttachmentManager(repoPath string) *AttachmentManager {
	return &AttachmentManager{
		repoPath: repoPath,
	}
}

// GetRepoPath returns the repository path
func (am *AttachmentManager) GetRepoPath() string {
	return am.repoPath
}

// GetAttachmentPath returns the expected path for a hash (legacy format)
func (am *AttachmentManager) GetAttachmentPath(hash string) string {
	if len(hash) < 2 {
		return ""
	}

	// Normalize hash to lowercase
	hash = strings.ToLower(hash)

	// Use first 2 characters as subdirectory
	prefix := hash[:2]
	return filepath.Join("attachments", prefix, hash)
}

// GetAttachmentDirPath returns the directory path for new directory-based structure
func (am *AttachmentManager) GetAttachmentDirPath(hash string) string {
	if len(hash) < 2 {
		return ""
	}

	// Normalize hash to lowercase
	hash = strings.ToLower(hash)

	// Use first 2 characters as subdirectory, hash as directory name
	prefix := hash[:2]
	return filepath.Join("attachments", prefix, hash)
}

// IsNewFormat checks if an attachment uses the new directory format
func (am *AttachmentManager) IsNewFormat(hash string) bool {
	dirPath := am.GetAttachmentDirPath(hash)
	fullDirPath := filepath.Join(am.repoPath, dirPath)

	// Check if directory exists
	if info, err := os.Stat(fullDirPath); err == nil && info.IsDir() {
		// Check if metadata.yaml exists in the directory
		metadataPath := filepath.Join(fullDirPath, "metadata.yaml")
		if _, err := os.Stat(metadataPath); err == nil {
			return true
		}
	}

	return false
}

// IsLegacyFormat checks if an attachment uses the legacy file format
func (am *AttachmentManager) IsLegacyFormat(hash string) bool {
	legacyPath := am.GetAttachmentPath(hash)
	fullPath := filepath.Join(am.repoPath, legacyPath)

	// Check if file exists and is not a directory
	if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
		return true
	}

	return false
}

// isValidHash checks if a hash string is valid (64-char lowercase hex)
func isValidHash(hash string) bool {
	if len(hash) != 64 {
		return false
	}

	// Check if all characters are lowercase hex
	matched, _ := regexp.MatchString("^[0-9a-f]+$", hash)
	return matched
}

// GetAttachment retrieves attachment info by hash (supports both formats)
func (am *AttachmentManager) GetAttachment(hash string) (*Attachment, error) {
	if !isValidHash(hash) {
		return nil, fmt.Errorf("invalid hash format: %s", hash)
	}

	attachment := &Attachment{
		Hash: hash,
	}

	// Check new format first
	if am.IsNewFormat(hash) {
		storage := NewDirectoryAttachmentStorage(am.repoPath)
		metadata, err := storage.GetMetadata(hash)
		if err != nil {
			return nil, fmt.Errorf("failed to get metadata for new format attachment: %w", err)
		}

		relPath, err := storage.GetPath(hash)
		if err != nil {
			return nil, fmt.Errorf("failed to get path for new format attachment: %w", err)
		}

		attachment.Path = relPath
		attachment.Size = metadata.Size
		attachment.Exists = true
		return attachment, nil
	}

	// Check legacy format
	if am.IsLegacyFormat(hash) {
		relPath := am.GetAttachmentPath(hash)
		fullPath := filepath.Join(am.repoPath, relPath)

		info, err := os.Stat(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to stat legacy attachment %s: %w", hash, err)
		}

		attachment.Path = relPath
		attachment.Size = info.Size()
		attachment.Exists = true
		return attachment, nil
	}

	// Not found in either format
	attachment.Exists = false
	attachment.Path = am.GetAttachmentPath(hash) // Default to legacy path format
	return attachment, nil
}

// AttachmentExists checks if attachment exists
func (am *AttachmentManager) AttachmentExists(hash string) (bool, error) {
	attachment, err := am.GetAttachment(hash)
	if err != nil {
		return false, err
	}
	return attachment.Exists, nil
}

// ReadAttachment reads the actual file content (supports both formats)
func (am *AttachmentManager) ReadAttachment(hash string) ([]byte, error) {
	if !isValidHash(hash) {
		return nil, fmt.Errorf("invalid hash format: %s", hash)
	}

	// Check new format first
	if am.IsNewFormat(hash) {
		storage := NewDirectoryAttachmentStorage(am.repoPath)
		return storage.ReadAttachment(hash)
	}

	// Check legacy format
	if am.IsLegacyFormat(hash) {
		relPath := am.GetAttachmentPath(hash)
		fullPath := filepath.Join(am.repoPath, relPath)

		data, err := os.ReadFile(fullPath) // #nosec G304
		if err != nil {
			return nil, fmt.Errorf("failed to read legacy attachment %s: %w", hash, err)
		}

		return data, nil
	}

	return nil, fmt.Errorf("attachment not found: %s", hash)
}

// VerifyAttachment checks if file content matches its hash
func (am *AttachmentManager) VerifyAttachment(hash string) (bool, error) {
	data, err := am.ReadAttachment(hash)
	if err != nil {
		return false, err
	}

	// Calculate SHA-256 hash of content
	hasher := sha256.New()
	hasher.Write(data)
	actualHash := fmt.Sprintf("%x", hasher.Sum(nil))

	return strings.EqualFold(actualHash, hash), nil
}

// ListAttachments returns all attachments in repository
func (am *AttachmentManager) ListAttachments() ([]*Attachment, error) {
	var attachments []*Attachment

	err := am.StreamAttachments(func(attachment *Attachment) error {
		attachments = append(attachments, attachment)
		return nil
	})

	return attachments, err
}

// StreamAttachments streams attachment info for memory efficiency (supports both formats)
func (am *AttachmentManager) StreamAttachments(callback func(*Attachment) error) error {
	attachmentsDir := filepath.Join(am.repoPath, "attachments")

	// Check if attachments directory exists
	if _, err := os.Stat(attachmentsDir); os.IsNotExist(err) {
		// No attachments directory is not an error
		return nil
	}

	// Read subdirectories (2-char prefixes)
	entries, err := os.ReadDir(attachmentsDir)
	if err != nil {
		return fmt.Errorf("failed to read attachments directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		prefix := entry.Name()
		if len(prefix) != 2 {
			continue // Skip invalid directory names
		}

		// Read files in this prefix directory
		prefixDir := filepath.Join(attachmentsDir, prefix)
		prefixEntries, err := os.ReadDir(prefixDir)
		if err != nil {
			// Log warning but continue with other directories
			continue
		}

		for _, prefixEntry := range prefixEntries {
			hash := prefixEntry.Name()
			if !isValidHash(hash) {
				continue // Skip invalid hash names
			}

			// Verify hash starts with the correct prefix
			if !strings.HasPrefix(hash, prefix) {
				continue // Skip misplaced files
			}

			if prefixEntry.IsDir() {
				if err := am.processNewFormatAttachment(prefixDir, hash, callback); err != nil {
					return err
				}
			} else {
				if err := am.processLegacyFormatAttachment(hash, prefixEntry, callback); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// processNewFormatAttachment processes a new format attachment directory
func (am *AttachmentManager) processNewFormatAttachment(prefixDir, hash string, callback func(*Attachment) error) error {
	metadataPath := filepath.Join(prefixDir, hash, "metadata.yaml")
	if _, err := os.Stat(metadataPath); err != nil {
		return nil // Skip if no metadata file
	}

	attachment, err := am.GetAttachment(hash)
	if err != nil {
		return nil // Skip attachments we can't process
	}

	return callback(attachment)
}

// processLegacyFormatAttachment processes a legacy format attachment file
func (am *AttachmentManager) processLegacyFormatAttachment(hash string, entry fs.DirEntry, callback func(*Attachment) error) error {
	info, err := entry.Info()
	if err != nil {
		return nil // Skip files we can't read
	}

	attachment := &Attachment{
		Hash:   hash,
		Path:   am.GetAttachmentPath(hash),
		Size:   info.Size(),
		Exists: true,
	}

	return callback(attachment)
}

// FindOrphanedAttachments returns attachments not referenced by any messages
func (am *AttachmentManager) FindOrphanedAttachments(referencedHashes map[string]bool) ([]*Attachment, error) {
	var orphaned []*Attachment

	err := am.StreamAttachments(func(attachment *Attachment) error {
		// Check if this attachment is referenced
		if !referencedHashes[attachment.Hash] {
			orphaned = append(orphaned, attachment)
		}
		return nil
	})

	return orphaned, err
}

// ValidateAttachmentStructure validates the directory structure
func (am *AttachmentManager) ValidateAttachmentStructure() error {
	attachmentsDir := filepath.Join(am.repoPath, "attachments")

	// Check if attachments directory exists
	if _, err := os.Stat(attachmentsDir); os.IsNotExist(err) {
		// No attachments directory is valid (empty repository)
		return nil
	}

	entries, err := os.ReadDir(attachmentsDir)
	if err != nil {
		return fmt.Errorf("failed to read attachments directory: %w", err)
	}

	var errors []string

	for _, entry := range entries {
		name := entry.Name()

		if !entry.IsDir() {
			errors = append(errors, fmt.Sprintf("file found in attachments root: %s", name))
			continue
		}

		// Validate directory name (must be 2 lowercase hex chars)
		if len(name) != 2 {
			errors = append(errors, fmt.Sprintf("invalid directory name length: %s", name))
			continue
		}

		if !dirNameRegex.MatchString(name) {
			errors = append(errors, fmt.Sprintf("invalid directory name format: %s", name))
			continue
		}

		// Validate files in subdirectory
		subDir := filepath.Join(attachmentsDir, name)
		subEntries, err := os.ReadDir(subDir)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to read directory %s: %v", name, err))
			continue
		}

		for _, subEntry := range subEntries {
			entryName := subEntry.Name()

			if subEntry.IsDir() {
				// This could be a new format hash directory
				if !isValidHash(entryName) {
					errors = append(errors, fmt.Sprintf("invalid hash directory name: %s/%s", name, entryName))
					continue
				}

				// Verify directory is in correct prefix (hash starts with directory name)
				if !strings.HasPrefix(entryName, name) {
					errors = append(errors, fmt.Sprintf("misplaced hash directory %s in prefix %s", entryName, name))
					continue
				}

				// Check if this is a valid new format directory (has metadata.yaml)
				metadataPath := filepath.Join(subDir, entryName, "metadata.yaml")
				if _, err := os.Stat(metadataPath); err == nil {
					// Valid new format directory - no further validation needed here
					continue
				}
				// Directory without metadata.yaml is invalid
				errors = append(errors, fmt.Sprintf("hash directory missing metadata.yaml: %s/%s", name, entryName))
				continue
			}

			// Legacy format: direct file in prefix directory
			fileName := entryName
			if !isValidHash(fileName) {
				errors = append(errors, fmt.Sprintf("invalid hash filename: %s/%s", name, fileName))
				continue
			}

			// Verify file is in correct directory (hash starts with directory name)
			if !strings.HasPrefix(fileName, name) {
				errors = append(errors, fmt.Sprintf("misplaced file %s in directory %s", fileName, name))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("attachment structure validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// GetAttachmentStats collects statistics about attachments in the repository
func (am *AttachmentManager) GetAttachmentStats(referencedHashes map[string]bool) (*AttachmentStats, error) {
	stats := &AttachmentStats{}

	err := am.StreamAttachments(func(attachment *Attachment) error {
		stats.TotalCount++
		stats.TotalSize += attachment.Size

		// Check if orphaned
		if !referencedHashes[attachment.Hash] {
			stats.OrphanedCount++
		}

		// Check if corrupted (optional verification)
		if verified, verifyErr := am.VerifyAttachment(attachment.Hash); verifyErr == nil && !verified {
			stats.CorruptedCount++
		}

		return nil
	})

	return stats, err
}
