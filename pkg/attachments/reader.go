package attachments

import (
	"crypto/sha256"
	"fmt"
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

// GetAttachmentPath returns the expected path for a hash
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

// isValidHash checks if a hash string is valid (64-char lowercase hex)
func isValidHash(hash string) bool {
	if len(hash) != 64 {
		return false
	}

	// Check if all characters are lowercase hex
	matched, _ := regexp.MatchString("^[0-9a-f]+$", hash)
	return matched
}

// GetAttachment retrieves attachment info by hash
func (am *AttachmentManager) GetAttachment(hash string) (*Attachment, error) {
	if !isValidHash(hash) {
		return nil, fmt.Errorf("invalid hash format: %s", hash)
	}

	relPath := am.GetAttachmentPath(hash)
	fullPath := filepath.Join(am.repoPath, relPath)

	attachment := &Attachment{
		Hash: hash,
		Path: relPath,
	}

	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		attachment.Exists = false
		return attachment, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to stat attachment %s: %w", hash, err)
	}

	attachment.Exists = true
	attachment.Size = info.Size()

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

// ReadAttachment reads the actual file content
func (am *AttachmentManager) ReadAttachment(hash string) ([]byte, error) {
	if !isValidHash(hash) {
		return nil, fmt.Errorf("invalid hash format: %s", hash)
	}

	relPath := am.GetAttachmentPath(hash)
	fullPath := filepath.Join(am.repoPath, relPath)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read attachment %s: %w", hash, err)
	}

	return data, nil
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

// StreamAttachments streams attachment info for memory efficiency
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
			if prefixEntry.IsDir() {
				continue // Skip subdirectories
			}

			hash := prefixEntry.Name()
			if !isValidHash(hash) {
				continue // Skip invalid hash files
			}

			// Verify hash starts with the correct prefix
			if !strings.HasPrefix(hash, prefix) {
				continue // Skip misplaced files
			}

			info, err := prefixEntry.Info()
			if err != nil {
				continue // Skip files we can't read
			}

			attachment := &Attachment{
				Hash:   hash,
				Path:   am.GetAttachmentPath(hash),
				Size:   info.Size(),
				Exists: true,
			}

			if err := callback(attachment); err != nil {
				return err
			}
		}
	}

	return nil
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
			if subEntry.IsDir() {
				errors = append(errors, fmt.Sprintf("unexpected subdirectory: %s/%s", name, subEntry.Name()))
				continue
			}

			fileName := subEntry.Name()
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
