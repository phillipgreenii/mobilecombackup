package attachments

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	customerrors "github.com/phillipgreen/mobilecombackup/pkg/errors"
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
	// Validate hash first to prevent null byte injection before path construction
	if err := das.validateHash(hash); err != nil {
		return customerrors.WrapWithValidation("validate attachment hash", err)
	}

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

// StoreFromReader stores an attachment by streaming from io.Reader
// This is memory-efficient for large attachments as it doesn't load all data into memory
func (das *DirectoryAttachmentStorage) StoreFromReader(hash string, data io.Reader, metadata AttachmentInfo) error {
	// Validate hash first to prevent null byte injection before path construction
	if err := das.validateHash(hash); err != nil {
		return customerrors.WrapWithValidation("validate attachment hash", err)
	}

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
		// Check if it's a disk space issue
		if strings.Contains(err.Error(), "no space left") {
			return fmt.Errorf("%w: %v", ErrDiskFull, err)
		}
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

	// Use temp file for atomic write operation
	tempFile := attachmentPath + ".tmp"

	// Create temp file
	file, err := os.Create(tempFile)
	if err != nil {
		if strings.Contains(err.Error(), "no space left") {
			return fmt.Errorf("%w: %v", ErrDiskFull, err)
		}
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	// Ensure cleanup on any error
	defer func() {
		_ = file.Close()
		_ = os.Remove(tempFile) // Clean up temp file if something goes wrong
	}()

	// Create hash writer for verification
	hasher := sha256.New()
	multiWriter := io.MultiWriter(file, hasher)

	// Stream with fixed buffer size for memory efficiency (32KB)
	written, err := io.CopyBuffer(multiWriter, data, make([]byte, 32*1024))
	if err != nil {
		if strings.Contains(err.Error(), "no space left") {
			return fmt.Errorf("%w: %v", ErrDiskFull, err)
		}
		return fmt.Errorf("failed to write attachment data: %w", err)
	}

	// Verify hash matches expected
	calculatedHash := hex.EncodeToString(hasher.Sum(nil))
	if calculatedHash != hash {
		return fmt.Errorf("%w: expected %s, got %s", ErrHashMismatch, hash, calculatedHash)
	}

	// Update metadata with actual size
	metadata.Size = written

	// Close temp file before renaming
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename to final location
	if err := os.Rename(tempFile, attachmentPath); err != nil {
		return fmt.Errorf("failed to move temp file to final location: %w", err)
	}

	// Write metadata file with validation (same as before)
	metadataRelPath, err := das.pathValidator.JoinAndValidate(validatedDirPath, "metadata.yaml")
	if err != nil {
		// Clean up attachment file if metadata fails
		_ = os.Remove(attachmentPath)
		return fmt.Errorf("invalid metadata file path: %w", err)
	}

	metadataPath, err := das.pathValidator.GetSafePath(metadataRelPath)
	if err != nil {
		_ = os.Remove(attachmentPath)
		return fmt.Errorf("failed to get safe metadata path: %w", err)
	}

	metadataData, err := yaml.Marshal(metadata)
	if err != nil {
		_ = os.Remove(attachmentPath) // Cleanup attempt - ignore error
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataData, 0644); err != nil {
		_ = os.Remove(attachmentPath) // Cleanup attempt - ignore error
		if strings.Contains(err.Error(), "no space left") {
			return fmt.Errorf("%w: %v", ErrDiskFull, err)
		}
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// GetPath returns the path to the attachment file
func (das *DirectoryAttachmentStorage) GetPath(hash string) (string, error) {
	// Validate hash first
	if err := das.validateHash(hash); err != nil {
		return "", fmt.Errorf("invalid hash: %w", err)
	}

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
	// Validate hash first
	if err := das.validateHash(hash); err != nil {
		return AttachmentInfo{}, fmt.Errorf("invalid hash: %w", err)
	}

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
	// Validate hash first - invalid hashes don't exist by definition
	if err := das.validateHash(hash); err != nil {
		return false
	}

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

// validateHash validates that a hash is safe to use in file paths
func (das *DirectoryAttachmentStorage) validateHash(hash string) error {
	// Check for empty hash
	if hash == "" {
		return fmt.Errorf("hash cannot be empty")
	}

	// Check for null bytes (must be done before any path operations)
	if strings.Contains(hash, "\x00") {
		return fmt.Errorf("hash contains null byte")
	}

	// Check for path traversal characters
	if strings.Contains(hash, "..") {
		return fmt.Errorf("hash contains path traversal sequence")
	}

	// Check for absolute path characters
	if strings.Contains(hash, "/") || strings.Contains(hash, "\\") {
		return fmt.Errorf("hash contains path separator")
	}

	// Check hash length (reasonable bounds)
	if len(hash) < 2 {
		return fmt.Errorf("hash too short (minimum 2 characters)")
	}
	if len(hash) > 256 {
		return fmt.Errorf("hash too long (maximum 256 characters)")
	}

	return nil
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
