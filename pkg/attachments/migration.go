package attachments

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MigrationManager handles migration from legacy to new attachment format
type MigrationManager struct {
	repoPath  string
	dryRun    bool
	logOutput bool
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(repoPath string) *MigrationManager {
	return &MigrationManager{
		repoPath:  repoPath,
		dryRun:    false,
		logOutput: true,
	}
}

// SetDryRun configures dry run mode
func (mm *MigrationManager) SetDryRun(dryRun bool) {
	mm.dryRun = dryRun
}

// SetLogOutput configures logging output
func (mm *MigrationManager) SetLogOutput(enabled bool) {
	mm.logOutput = enabled
}

// MigrationResult represents the result of migrating a single attachment
type MigrationResult struct {
	Hash          string
	LegacyPath    string
	NewPath       string
	Size          int64
	Success       bool
	Error         string
	DetectedType  string
	GeneratedName string
}

// MigrationSummary provides overall migration statistics
type MigrationSummary struct {
	TotalFound      int
	Migrated        int
	Failed          int
	Skipped         int
	AlreadyMigrated int
	Results         []MigrationResult
}

// MigrateAllAttachments migrates all legacy attachments to new format
func (mm *MigrationManager) MigrateAllAttachments() (*MigrationSummary, error) {
	summary := &MigrationSummary{
		Results: make([]MigrationResult, 0),
	}

	attachmentManager := NewAttachmentManager(mm.repoPath)
	storage := NewDirectoryAttachmentStorage(mm.repoPath)

	// Find all legacy attachments
	err := attachmentManager.StreamAttachments(func(attachment *Attachment) error {
		summary.TotalFound++

		// Check if this is already migrated (skip if new format)
		if attachmentManager.IsNewFormat(attachment.Hash) {
			if mm.logOutput {
				log.Printf("[MIGRATION] Skipping already migrated attachment: %s", attachment.Hash)
			}
			summary.AlreadyMigrated++
			return nil
		}

		// Only migrate legacy format attachments
		if !attachmentManager.IsLegacyFormat(attachment.Hash) {
			if mm.logOutput {
				log.Printf("[MIGRATION] Skipping non-legacy attachment: %s", attachment.Hash)
			}
			summary.Skipped++
			return nil
		}

		result := mm.migrateAttachment(attachment, storage)
		summary.Results = append(summary.Results, result)

		if result.Success {
			summary.Migrated++
		} else {
			summary.Failed++
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to stream attachments for migration: %w", err)
	}

	if mm.logOutput {
		log.Printf("[MIGRATION] Migration complete: %d found, %d migrated, %d failed, %d skipped, %d already migrated",
			summary.TotalFound, summary.Migrated, summary.Failed, summary.Skipped, summary.AlreadyMigrated)
	}

	return summary, nil
}

// migrateAttachment migrates a single attachment from legacy to new format
func (mm *MigrationManager) migrateAttachment(
	attachment *Attachment,
	storage *DirectoryAttachmentStorage,
) MigrationResult {
	result := mm.initializeMigrationResult(attachment)

	if mm.logOutput {
		log.Printf("[MIGRATION] Migrating attachment: %s", attachment.Hash)
	}

	data, metadata, err := mm.prepareMigrationData(attachment, &result)
	if err != nil {
		return result
	}

	if mm.dryRun {
		return mm.handleDryRunMigration(attachment, storage, result)
	}

	return mm.executeMigration(attachment, storage, data, metadata, result)
}

// initializeMigrationResult creates an initial migration result
func (mm *MigrationManager) initializeMigrationResult(attachment *Attachment) MigrationResult {
	return MigrationResult{
		Hash:       attachment.Hash,
		LegacyPath: attachment.Path,
		Size:       attachment.Size,
	}
}

// prepareMigrationData reads legacy file and prepares metadata
func (mm *MigrationManager) prepareMigrationData(
	attachment *Attachment,
	result *MigrationResult,
) ([]byte, AttachmentInfo, error) {
	// Read the legacy attachment content
	legacyFullPath := filepath.Join(mm.repoPath, attachment.Path)
	data, err := os.ReadFile(legacyFullPath) // #nosec G304
	if err != nil {
		result.Error = fmt.Sprintf("failed to read legacy file: %v", err)
		return nil, AttachmentInfo{}, err
	}

	// Detect MIME type from content (simple detection)
	mimeType := detectMimeTypeFromContent(data, attachment.Hash)
	result.DetectedType = mimeType

	// Generate filename
	filename := GenerateFilename("", mimeType)
	result.GeneratedName = filename

	// Create metadata
	metadata := AttachmentInfo{
		Hash:         attachment.Hash,
		OriginalName: "", // No original name in legacy format
		MimeType:     mimeType,
		Size:         int64(len(data)),
		CreatedAt:    time.Now().UTC(),
		SourceMMS:    "", // Unknown for migrated attachments
	}

	return data, metadata, nil
}

// handleDryRunMigration handles migration in dry run mode
func (mm *MigrationManager) handleDryRunMigration(
	attachment *Attachment, storage *DirectoryAttachmentStorage, result MigrationResult,
) MigrationResult {
	// In dry run mode, just simulate the migration
	filename := result.GeneratedName
	result.NewPath = filepath.Join(storage.getAttachmentDirPath(attachment.Hash), filename)
	result.Success = true
	if mm.logOutput {
		log.Printf("[MIGRATION] [DRY RUN] Would migrate %s -> %s", result.LegacyPath, result.NewPath)
	}
	return result
}

// executeMigration performs the actual migration
func (mm *MigrationManager) executeMigration(
	attachment *Attachment, storage *DirectoryAttachmentStorage,
	data []byte, metadata AttachmentInfo, result MigrationResult,
) MigrationResult {
	// Store with new format
	if err := storage.Store(attachment.Hash, data, metadata); err != nil {
		result.Error = fmt.Sprintf("failed to store in new format: %v", err)
		return result
	}

	// Get the new path
	newPath, err := storage.GetPath(attachment.Hash)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get new path: %v", err)
		return result
	}
	result.NewPath = newPath

	// Remove the legacy file
	legacyFullPath := filepath.Join(mm.repoPath, attachment.Path)
	if err := os.Remove(legacyFullPath); err != nil {
		result.Error = fmt.Sprintf("failed to remove legacy file: %v", err)
		return result
	}

	result.Success = true
	if mm.logOutput {
		log.Printf("[MIGRATION] Successfully migrated %s -> %s", result.LegacyPath, result.NewPath)
	}

	return result
}

// detectMimeTypeFromContent performs basic MIME type detection
func detectMimeTypeFromContent(data []byte, _ string) string {
	if len(data) == 0 {
		return "application/octet-stream"
	}

	// Check for common file signatures
	if len(data) >= 8 {
		// PNG signature
		if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
			return "image/png"
		}
		// JPEG signature
		if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
			return "image/jpeg"
		}
		// GIF signature
		if strings.HasPrefix(string(data[:6]), "GIF87a") || strings.HasPrefix(string(data[:6]), "GIF89a") {
			return "image/gif"
		}
	}

	if len(data) >= 4 {
		// PDF signature
		if strings.HasPrefix(string(data[:4]), "%PDF") {
			return "application/pdf"
		}
		// ZIP signature (also used by other formats)
		if data[0] == 0x50 && data[1] == 0x4B && data[2] == 0x03 && data[3] == 0x04 {
			return "application/zip"
		}
	}

	// Check for text content
	if isTextContent(data) {
		return "text/plain"
	}

	// Default to binary
	return "application/octet-stream"
}

// isTextContent checks if data appears to be text
func isTextContent(data []byte) bool {
	if len(data) == 0 {
		return true
	}

	// Sample the first part of the data
	sampleSize := 512
	if len(data) < sampleSize {
		sampleSize = len(data)
	}

	sample := data[:sampleSize]
	textChars := 0

	for _, b := range sample {
		// Count printable ASCII and common whitespace
		if (b >= 32 && b <= 126) || b == 9 || b == 10 || b == 13 {
			textChars++
		}
	}

	// If more than 95% are text characters, consider it text
	return float64(textChars)/float64(sampleSize) > 0.95
}

// ValidateMigration validates that migration was successful
func (mm *MigrationManager) ValidateMigration() error {
	attachmentManager := NewAttachmentManager(mm.repoPath)
	storage := NewDirectoryAttachmentStorage(mm.repoPath)

	hasLegacyAttachments := false
	hasErrors := false

	err := attachmentManager.StreamAttachments(func(attachment *Attachment) error {
		if attachmentManager.IsLegacyFormat(attachment.Hash) {
			hasLegacyAttachments = true
			if mm.logOutput {
				log.Printf("[VALIDATION] Found remaining legacy attachment: %s", attachment.Hash)
			}
		}

		if attachmentManager.IsNewFormat(attachment.Hash) {
			// Validate new format attachment
			metadata, err := storage.GetMetadata(attachment.Hash)
			if err != nil {
				hasErrors = true
				if mm.logOutput {
					log.Printf("[VALIDATION] Failed to read metadata for %s: %v", attachment.Hash, err)
				}
				return nil
			}

			// Verify the attachment file exists
			attachmentPath, err := storage.GetAttachmentFilePath(attachment.Hash)
			if err != nil {
				hasErrors = true
				if mm.logOutput {
					log.Printf("[VALIDATION] Failed to get file path for %s: %v", attachment.Hash, err)
				}
				return nil
			}

			if _, err := os.Stat(attachmentPath); os.IsNotExist(err) {
				hasErrors = true
				if mm.logOutput {
					log.Printf("[VALIDATION] Attachment file missing: %s", attachmentPath)
				}
			}

			// Verify metadata consistency
			if metadata.Hash != attachment.Hash {
				hasErrors = true
				if mm.logOutput {
					log.Printf("[VALIDATION] Metadata hash mismatch for %s", attachment.Hash)
				}
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if hasLegacyAttachments {
		return fmt.Errorf("migration incomplete: legacy attachments still exist")
	}

	if hasErrors {
		return fmt.Errorf("migration validation failed: errors found in migrated attachments")
	}

	if mm.logOutput {
		log.Printf("[VALIDATION] Migration validation successful")
	}

	return nil
}

// GetMigrationStatus returns the current migration status
func (mm *MigrationManager) GetMigrationStatus() (map[string]interface{}, error) {
	attachmentManager := NewAttachmentManager(mm.repoPath)

	status := map[string]interface{}{
		"legacy_count": 0,
		"new_count":    0,
		"total_count":  0,
		"migrated":     false,
	}

	err := attachmentManager.StreamAttachments(func(attachment *Attachment) error {
		status["total_count"] = status["total_count"].(int) + 1

		if attachmentManager.IsLegacyFormat(attachment.Hash) {
			status["legacy_count"] = status["legacy_count"].(int) + 1
		} else if attachmentManager.IsNewFormat(attachment.Hash) {
			status["new_count"] = status["new_count"].(int) + 1
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get migration status: %w", err)
	}

	status["migrated"] = status["legacy_count"].(int) == 0 && status["total_count"].(int) > 0

	return status, nil
}
