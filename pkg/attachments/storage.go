package attachments

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/phillipgreen/mobilecombackup/pkg/security"
	"gopkg.in/yaml.v3"
)

// DirectoryAttachmentStorage implements AttachmentStorage with directory-based organization
type DirectoryAttachmentStorage struct {
	repoPath      string
	pathValidator *security.PathValidator
}

// NewDirectoryAttachmentStorage creates a new directory-based attachment storage
func NewDirectoryAttachmentStorage(repoPath string) *DirectoryAttachmentStorage {
	return &DirectoryAttachmentStorage{
		repoPath:      repoPath,
		pathValidator: security.NewPathValidator(repoPath),
	}
}

// Store saves an attachment with its metadata in a directory structure
func (das *DirectoryAttachmentStorage) Store(hash string, data []byte, metadata AttachmentInfo) error {
	// Create directory path: attachments/ab/abc123.../
	dirPath := das.getAttachmentDirPath(hash)

	// Validate directory path
	validatedDirPath, err := das.pathValidator.ValidatePath(dirPath)
	if err != nil {
		return fmt.Errorf("invalid attachment directory path: %w", err)
	}

	fullDirPath, err := das.pathValidator.GetSafePath(validatedDirPath)
	if err != nil {
		return fmt.Errorf("failed to get safe directory path: %w", err)
	}

	// Create directory
	if err := os.MkdirAll(fullDirPath, 0750); err != nil {
		return fmt.Errorf("failed to create attachment directory: %w", err)
	}

	// Generate filename and validate
	filename := GenerateFilename(metadata.OriginalName, metadata.MimeType)
	attachmentRelPath, err := das.pathValidator.JoinAndValidate(validatedDirPath, filename)
	if err != nil {
		return fmt.Errorf("invalid attachment file path: %w", err)
	}

	attachmentPath, err := das.pathValidator.GetSafePath(attachmentRelPath)
	if err != nil {
		return fmt.Errorf("failed to get safe attachment path: %w", err)
	}

	// Write attachment file
	if err := os.WriteFile(attachmentPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write attachment file: %w", err)
	}

	// Write metadata file with validation
	metadataRelPath, err := das.pathValidator.JoinAndValidate(validatedDirPath, "metadata.yaml")
	if err != nil {
		return fmt.Errorf("invalid metadata file path: %w", err)
	}

	metadataPath, err := das.pathValidator.GetSafePath(metadataRelPath)
	if err != nil {
		return fmt.Errorf("failed to get safe metadata path: %w", err)
	}

	metadataData, err := yaml.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataData, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// Store with io.Reader interface
func (das *DirectoryAttachmentStorage) StoreFromReader(hash string, data io.Reader, metadata AttachmentInfo) error {
	// Read all data first
	dataBytes, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("failed to read attachment data: %w", err)
	}

	return das.Store(hash, dataBytes, metadata)
}

// GetPath returns the path to the attachment file
func (das *DirectoryAttachmentStorage) GetPath(hash string) (string, error) {
	dirPath := das.getAttachmentDirPath(hash)

	// Validate directory path
	validatedDirPath, err := das.pathValidator.ValidatePath(dirPath)
	if err != nil {
		return "", fmt.Errorf("invalid attachment directory path: %w", err)
	}

	fullDirPath, err := das.pathValidator.GetSafePath(validatedDirPath)
	if err != nil {
		return "", fmt.Errorf("failed to get safe directory path: %w", err)
	}

	// Check if directory exists
	if _, err := os.Stat(fullDirPath); os.IsNotExist(err) {
		return "", fmt.Errorf("attachment directory not found: %s", hash)
	}

	// Read metadata to get filename
	metadata, err := das.GetMetadata(hash)
	if err != nil {
		return "", fmt.Errorf("failed to get metadata for path resolution: %w", err)
	}

	filename := GenerateFilename(metadata.OriginalName, metadata.MimeType)

	// Validate and return the relative path
	attachmentRelPath, err := das.pathValidator.JoinAndValidate(validatedDirPath, filename)
	if err != nil {
		return "", fmt.Errorf("invalid attachment file path: %w", err)
	}

	return attachmentRelPath, nil
}

// GetMetadata reads the metadata.yaml file
func (das *DirectoryAttachmentStorage) GetMetadata(hash string) (AttachmentInfo, error) {
	dirPath := das.getAttachmentDirPath(hash)

	// Validate directory path
	validatedDirPath, err := das.pathValidator.ValidatePath(dirPath)
	if err != nil {
		return AttachmentInfo{}, fmt.Errorf("invalid attachment directory path: %w", err)
	}

	// Validate metadata file path
	metadataRelPath, err := das.pathValidator.JoinAndValidate(validatedDirPath, "metadata.yaml")
	if err != nil {
		return AttachmentInfo{}, fmt.Errorf("invalid metadata file path: %w", err)
	}

	metadataPath, err := das.pathValidator.GetSafePath(metadataRelPath)
	if err != nil {
		return AttachmentInfo{}, fmt.Errorf("failed to get safe metadata path: %w", err)
	}

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return AttachmentInfo{}, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata AttachmentInfo
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return AttachmentInfo{}, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return metadata, nil
}

// Exists checks if an attachment directory exists
func (das *DirectoryAttachmentStorage) Exists(hash string) bool {
	dirPath := das.getAttachmentDirPath(hash)

	// Validate directory path
	validatedDirPath, err := das.pathValidator.ValidatePath(dirPath)
	if err != nil {
		return false // Invalid path means attachment doesn't exist securely
	}

	fullDirPath, err := das.pathValidator.GetSafePath(validatedDirPath)
	if err != nil {
		return false
	}

	if _, err := os.Stat(fullDirPath); os.IsNotExist(err) {
		return false
	}

	// Check if metadata file exists with validation
	metadataRelPath, err := das.pathValidator.JoinAndValidate(validatedDirPath, "metadata.yaml")
	if err != nil {
		return false
	}

	metadataPath, err := das.pathValidator.GetSafePath(metadataRelPath)
	if err != nil {
		return false
	}

	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return false
	}

	return true
}

// getAttachmentDirPath returns the relative directory path for a hash
func (das *DirectoryAttachmentStorage) getAttachmentDirPath(hash string) string {
	if len(hash) < 2 {
		return ""
	}

	// Use first 2 characters as subdirectory
	prefix := hash[:2]
	return filepath.Join("attachments", prefix, hash)
}

// GetAttachmentFilePath returns the full path to the attachment file
func (das *DirectoryAttachmentStorage) GetAttachmentFilePath(hash string) (string, error) {
	metadata, err := das.GetMetadata(hash)
	if err != nil {
		return "", err
	}

	dirPath := das.getAttachmentDirPath(hash)

	// Validate directory path
	validatedDirPath, err := das.pathValidator.ValidatePath(dirPath)
	if err != nil {
		return "", fmt.Errorf("invalid attachment directory path: %w", err)
	}

	filename := GenerateFilename(metadata.OriginalName, metadata.MimeType)

	// Validate attachment file path
	attachmentRelPath, err := das.pathValidator.JoinAndValidate(validatedDirPath, filename)
	if err != nil {
		return "", fmt.Errorf("invalid attachment file path: %w", err)
	}

	return das.pathValidator.GetSafePath(attachmentRelPath)
}

// ReadAttachment reads the attachment file content
func (das *DirectoryAttachmentStorage) ReadAttachment(hash string) ([]byte, error) {
	filePath, err := das.GetAttachmentFilePath(hash)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read attachment file: %w", err)
	}

	return data, nil
}
