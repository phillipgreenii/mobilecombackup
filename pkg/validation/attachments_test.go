package validation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/phillipgreenii/mobilecombackup/pkg/attachments"
	"github.com/phillipgreenii/mobilecombackup/pkg/sms"
)

// mockAttachmentReader implements AttachmentReader for testing
type mockAttachmentReader struct {
	attachments         []*attachments.Attachment
	listError           error
	structureError      error
	verificationResults map[string]bool
	verificationErrors  map[string]error
	existsResults       map[string]bool
	existsErrors        map[string]error
	orphanedAttachments []*attachments.Attachment
	orphanedError       error
}

func (m *mockAttachmentReader) GetAttachment(hash string) (*attachments.Attachment, error) {
	for _, attachment := range m.attachments {
		if attachment.Hash == hash {
			return attachment, nil
		}
	}
	return nil, fmt.Errorf("attachment not found: %s", hash)
}

func (m *mockAttachmentReader) ReadAttachment(hash string) ([]byte, error) {
	return []byte(fmt.Sprintf("content for %s", hash)), nil
}

func (m *mockAttachmentReader) AttachmentExists(hash string) (bool, error) {
	if err, exists := m.existsErrors[hash]; exists {
		return false, err
	}
	if result, exists := m.existsResults[hash]; exists {
		return result, nil
	}
	return false, nil
}

func (m *mockAttachmentReader) ListAttachments() ([]*attachments.Attachment, error) {
	if m.listError != nil {
		return nil, m.listError
	}
	return m.attachments, nil
}

func (m *mockAttachmentReader) StreamAttachments(callback func(*attachments.Attachment) error) error {
	if m.listError != nil {
		return m.listError
	}
	for _, attachment := range m.attachments {
		if err := callback(attachment); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockAttachmentReader) VerifyAttachment(hash string) (bool, error) {
	if err, exists := m.verificationErrors[hash]; exists {
		return false, err
	}
	if result, exists := m.verificationResults[hash]; exists {
		return result, nil
	}
	return true, nil
}

func (m *mockAttachmentReader) GetAttachmentPath(hash string) string {
	return fmt.Sprintf("attachments/%s/%s", hash[:2], hash)
}

func (m *mockAttachmentReader) FindOrphanedAttachments(_ map[string]bool) ([]*attachments.Attachment, error) {
	if m.orphanedError != nil {
		return nil, m.orphanedError
	}
	return m.orphanedAttachments, nil
}

func (m *mockAttachmentReader) ValidateAttachmentStructure() error {
	return m.structureError
}

// Context-aware method implementations for mock

func (m *mockAttachmentReader) GetAttachmentContext(ctx context.Context, hash string) (*attachments.Attachment, error) {
	return m.GetAttachment(hash)
}

func (m *mockAttachmentReader) ReadAttachmentContext(ctx context.Context, hash string) ([]byte, error) {
	return m.ReadAttachment(hash)
}

func (m *mockAttachmentReader) AttachmentExistsContext(ctx context.Context, hash string) (bool, error) {
	return m.AttachmentExists(hash)
}

func (m *mockAttachmentReader) ListAttachmentsContext(ctx context.Context) ([]*attachments.Attachment, error) {
	return m.ListAttachments()
}

func (m *mockAttachmentReader) StreamAttachmentsContext(ctx context.Context, callback func(*attachments.Attachment) error) error {
	return m.StreamAttachments(callback)
}

func (m *mockAttachmentReader) VerifyAttachmentContext(ctx context.Context, hash string) (bool, error) {
	return m.VerifyAttachment(hash)
}

func (m *mockAttachmentReader) FindOrphanedAttachmentsContext(ctx context.Context, referencedHashes map[string]bool) ([]*attachments.Attachment, error) {
	return m.FindOrphanedAttachments(referencedHashes)
}

func (m *mockAttachmentReader) ValidateAttachmentStructureContext(ctx context.Context) error {
	return m.ValidateAttachmentStructure()
}

func TestAttachmentsValidatorImpl_ValidateAttachmentsStructure(t *testing.T) {
	tempDir := t.TempDir()

	// Test with no attachments (empty repository)
	mockReader := &mockAttachmentReader{
		attachments: []*attachments.Attachment{},
	}

	mockSMSReader := &mockSMSReader{
		messages:          make(map[int][]sms.Message),
		allAttachmentRefs: make(map[string]bool),
	}
	validator := NewAttachmentsValidator(tempDir, mockReader, mockSMSReader)
	violations := validator.ValidateAttachmentsStructure()

	// Empty repository should have no violations
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations for empty attachments repository, got %d: %v", len(violations), violations)
	}

	// Test with attachments present and valid structure
	mockReader.attachments = []*attachments.Attachment{
		{
			Hash:   "abc123",
			Path:   "attachments/ab/abc123",
			Size:   1024,
			Exists: true,
		},
	}

	violations = validator.ValidateAttachmentsStructure()

	// Should have no violations with valid structure
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations with valid structure, got %d: %v", len(violations), violations)
	}

	// Test with structure validation error
	mockReader.structureError = fmt.Errorf("invalid directory structure")

	violations = validator.ValidateAttachmentsStructure()

	// Should have structure violation
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation with structure error, got %d", len(violations))
	}

	if len(violations) > 0 && violations[0].Type != StructureViolation {
		t.Errorf("Expected StructureViolation, got %s", violations[0].Type)
	}
}

func TestAttachmentsValidatorImpl_ValidateAttachmentsStructure_MissingDirectory(t *testing.T) {
	tempDir := t.TempDir()

	// Test with list error (directory missing)
	mockReader := &mockAttachmentReader{
		listError: fmt.Errorf("directory not found"),
	}

	mockSMSReader := &mockSMSReader{
		messages:          make(map[int][]sms.Message),
		allAttachmentRefs: make(map[string]bool),
	}
	validator := NewAttachmentsValidator(tempDir, mockReader, mockSMSReader)
	violations := validator.ValidateAttachmentsStructure()

	// Should have warning for missing directory
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation for missing directory, got %d", len(violations))
	}

	if len(violations) > 0 && violations[0].Severity != SeverityWarning {
		t.Errorf("Expected warning severity for missing directory, got %s", violations[0].Severity)
	}

	// Create attachments directory but keep list error
	attachmentsDir := filepath.Join(tempDir, "attachments")
	err := os.MkdirAll(attachmentsDir, 0750)
	if err != nil {
		t.Fatalf("Failed to create attachments directory: %v", err)
	}

	violations = validator.ValidateAttachmentsStructure()

	// Should have error for list failure with existing directory
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation for list error, got %d", len(violations))
	}

	if len(violations) > 0 && violations[0].Severity != SeverityError {
		t.Errorf("Expected error severity for list failure, got %s", violations[0].Severity)
	}
}

func TestAttachmentsValidatorImpl_ValidateAttachmentIntegrity(t *testing.T) {
	tempDir := t.TempDir()

	// Create test attachment files
	validHashPath := filepath.Join(tempDir, "attachments", "va", "validhash123")
	corruptedHashPath := filepath.Join(tempDir, "attachments", "co", "corruptedhash789")

	// Create directory structure
	_ = os.MkdirAll(filepath.Dir(validHashPath), 0750)
	_ = os.MkdirAll(filepath.Dir(corruptedHashPath), 0750)

	// Create valid attachment file with PNG signature for format recognition
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	_ = os.WriteFile(validHashPath, pngData, 0600)

	// Create corrupted attachment file
	_ = os.WriteFile(corruptedHashPath, []byte("corrupted content"), 0600)

	mockReader := &mockAttachmentReader{
		attachments: []*attachments.Attachment{
			{
				Hash:   "validhash123",
				Path:   "attachments/va/validhash123",
				Size:   1024,
				Exists: true,
			},
			{
				Hash:   "missinghash456",
				Path:   "attachments/mi/missinghash456",
				Size:   2048,
				Exists: false, // Missing file
			},
			{
				Hash:   "corruptedhash789",
				Path:   "attachments/co/corruptedhash789",
				Size:   512,
				Exists: true,
			},
		},
		verificationResults: map[string]bool{
			"validhash123":     true,
			"corruptedhash789": false, // Corrupted
		},
		verificationErrors: map[string]error{
			"errorhash999": fmt.Errorf("verification failed"),
		},
	}

	mockSMSReader := &mockSMSReader{
		messages:          make(map[int][]sms.Message),
		allAttachmentRefs: make(map[string]bool),
	}
	validator := NewAttachmentsValidator(tempDir, mockReader, mockSMSReader)
	violations := validator.ValidateAttachmentIntegrity()

	// Should have violations for missing and corrupted files
	if len(violations) != 2 {
		t.Errorf("Expected 2 violations, got %d: %v", len(violations), violations)
	}

	// Check for specific violation types
	foundMissingFile := false
	foundChecksumMismatch := false

	for _, violation := range violations {
		switch violation.Type {
		case MissingFile:
			foundMissingFile = true
			if violation.File != "attachments/mi/missinghash456" {
				t.Errorf("Expected missing file violation for missinghash456, got %s", violation.File)
			}
		case ChecksumMismatch:
			foundChecksumMismatch = true
			if violation.File != "attachments/co/corruptedhash789" {
				t.Errorf("Expected checksum mismatch for corruptedhash789, got %s", violation.File)
			}
		}
	}

	if !foundMissingFile {
		t.Error("Expected MissingFile violation")
	}

	if !foundChecksumMismatch {
		t.Error("Expected ChecksumMismatch violation")
	}
}

func TestAttachmentsValidatorImpl_ValidateAttachmentIntegrity_VerificationError(t *testing.T) {
	tempDir := t.TempDir()

	mockReader := &mockAttachmentReader{
		attachments: []*attachments.Attachment{
			{
				Hash:   "errorhash999",
				Path:   "attachments/er/errorhash999",
				Size:   256,
				Exists: true,
			},
		},
		verificationErrors: map[string]error{
			"errorhash999": fmt.Errorf("verification failed"),
		},
	}

	mockSMSReader := &mockSMSReader{
		messages:          make(map[int][]sms.Message),
		allAttachmentRefs: make(map[string]bool),
	}
	validator := NewAttachmentsValidator(tempDir, mockReader, mockSMSReader)
	violations := validator.ValidateAttachmentIntegrity()

	// Should have violation for verification error
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation for verification error, got %d", len(violations))
	}

	if len(violations) > 0 && violations[0].Type != ChecksumMismatch {
		t.Errorf("Expected ChecksumMismatch violation type, got %s", violations[0].Type)
	}
}

func TestAttachmentsValidatorImpl_ValidateAttachmentReferences(t *testing.T) {
	tempDir := t.TempDir()

	referencedHashes := map[string]bool{
		"existinghash123":   true,
		"missinghash456":    true,
		"errorcheckhash789": true,
	}

	mockReader := &mockAttachmentReader{
		attachments: []*attachments.Attachment{
			{
				Hash:   "orphanedhash999",
				Path:   "attachments/or/orphanedhash999",
				Size:   1024,
				Exists: true,
			},
		},
		existsResults: map[string]bool{
			"existinghash123": true,
			"missinghash456":  false,
		},
		existsErrors: map[string]error{
			"errorcheckhash789": fmt.Errorf("check failed"),
		},
		orphanedAttachments: []*attachments.Attachment{
			{
				Hash:   "orphanedhash999",
				Path:   "attachments/or/orphanedhash999",
				Size:   1024,
				Exists: true,
			},
		},
	}

	mockSMSReader := &mockSMSReader{
		messages:          make(map[int][]sms.Message),
		allAttachmentRefs: make(map[string]bool),
	}
	validator := NewAttachmentsValidator(tempDir, mockReader, mockSMSReader)
	violations := validator.ValidateAttachmentReferences(referencedHashes)

	// Should have violations for:
	// 1. Missing referenced attachment
	// 2. Error checking attachment
	// 3. Orphaned attachment (warning)
	if len(violations) != 3 {
		t.Errorf("Expected 3 violations, got %d: %v", len(violations), violations)
	}

	// Check for specific violation types
	foundMissingRef := false
	foundErrorCheck := false
	foundOrphaned := false

	for _, violation := range violations {
		switch violation.Type {
		case MissingFile:
			if len(violation.Message) >= 31 && violation.Message[:31] == "Referenced attachment not found" {
				foundMissingRef = true
			} else if len(violation.Message) >= 25 && violation.Message[:25] == "Failed to check existence" {
				foundErrorCheck = true
			}
		case OrphanedAttachment:
			foundOrphaned = true
			if violation.Severity != SeverityWarning {
				t.Errorf("Expected warning severity for orphaned attachment, got %s", violation.Severity)
			}
		}
	}

	if !foundMissingRef {
		t.Error("Expected missing referenced attachment violation")
	}

	if !foundErrorCheck {
		t.Error("Expected error checking attachment violation")
	}

	if !foundOrphaned {
		t.Error("Expected orphaned attachment violation")
	}
}

func TestAttachmentsValidatorImpl_GetOrphanedAttachments(t *testing.T) {
	tempDir := t.TempDir()

	referencedHashes := map[string]bool{
		"referenced123": true,
	}

	expectedOrphaned := []*attachments.Attachment{
		{
			Hash:   "orphaned456",
			Path:   "attachments/or/orphaned456",
			Size:   2048,
			Exists: true,
		},
	}

	mockReader := &mockAttachmentReader{
		orphanedAttachments: expectedOrphaned,
	}

	mockSMSReader := &mockSMSReader{
		messages:          make(map[int][]sms.Message),
		allAttachmentRefs: make(map[string]bool),
	}
	validator := NewAttachmentsValidator(tempDir, mockReader, mockSMSReader)
	orphaned, err := validator.GetOrphanedAttachments(referencedHashes)

	if err != nil {
		t.Fatalf("Failed to get orphaned attachments: %v", err)
	}

	if len(orphaned) != 1 {
		t.Errorf("Expected 1 orphaned attachment, got %d", len(orphaned))
	}

	if len(orphaned) > 0 && orphaned[0].Hash != "orphaned456" {
		t.Errorf("Expected orphaned hash 'orphaned456', got %s", orphaned[0].Hash)
	}
}

func TestAttachmentsValidatorImpl_ErrorHandling(t *testing.T) {
	tempDir := t.TempDir()

	// Test with list error for integrity check
	mockReader := &mockAttachmentReader{
		listError: fmt.Errorf("failed to list attachments"),
	}

	mockSMSReader := &mockSMSReader{
		messages:          make(map[int][]sms.Message),
		allAttachmentRefs: make(map[string]bool),
	}
	validator := NewAttachmentsValidator(tempDir, mockReader, mockSMSReader)

	// Test integrity validation with list error
	violations := validator.ValidateAttachmentIntegrity()
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation for list error in integrity check, got %d", len(violations))
	}

	// Reference validation doesn't use ListAttachments, so no error expected with empty references
	violations = validator.ValidateAttachmentReferences(make(map[string]bool))
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations for empty reference check, got %d", len(violations))
	}

	// Test with orphaned attachments error
	mockReader.listError = nil
	mockReader.orphanedError = fmt.Errorf("failed to find orphaned attachments")

	violations = validator.ValidateAttachmentReferences(make(map[string]bool))
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation for orphaned attachments error, got %d", len(violations))
	}

	// Test GetOrphanedAttachments with error
	_, err := validator.GetOrphanedAttachments(make(map[string]bool))
	if err == nil {
		t.Error("Expected error from GetOrphanedAttachments")
	}
}

func TestDetectFileFormat(t *testing.T) {
	tempDir := t.TempDir()

	// Test PNG detection
	pngFile := filepath.Join(tempDir, "test.png")
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52} // PNG header
	if err := os.WriteFile(pngFile, pngData, 0600); err != nil {
		t.Fatalf("Failed to write PNG test file: %v", err)
	}

	format, err := DetectFileFormat(pngFile)
	if err != nil {
		t.Errorf("Expected no error for PNG file, got: %v", err)
	}
	if format != "image/png" {
		t.Errorf("Expected 'image/png', got: %s", format)
	}

	// Test JPEG detection
	jpegFile := filepath.Join(tempDir, "test.jpg")
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46} // JPEG header
	if err := os.WriteFile(jpegFile, jpegData, 0600); err != nil {
		t.Fatalf("Failed to write JPEG test file: %v", err)
	}

	format, err = DetectFileFormat(jpegFile)
	if err != nil {
		t.Errorf("Expected no error for JPEG file, got: %v", err)
	}
	if format != "image/jpeg" {
		t.Errorf("Expected 'image/jpeg', got: %s", format)
	}

	// Test GIF detection
	gifFile := filepath.Join(tempDir, "test.gif")
	gifData := []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61} // GIF89a header
	if err := os.WriteFile(gifFile, gifData, 0600); err != nil {
		t.Fatalf("Failed to write GIF test file: %v", err)
	}

	format, err = DetectFileFormat(gifFile)
	if err != nil {
		t.Errorf("Expected no error for GIF file, got: %v", err)
	}
	if format != "image/gif" {
		t.Errorf("Expected 'image/gif', got: %s", format)
	}

	// Test MP4 detection
	mp4File := filepath.Join(tempDir, "test.mp4")
	mp4Data := []byte{0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70, 0x6D, 0x70, 0x34, 0x31} // MP4 header with ftyp at offset 4
	if err := os.WriteFile(mp4File, mp4Data, 0600); err != nil {
		t.Fatalf("Failed to write MP4 test file: %v", err)
	}

	format, err = DetectFileFormat(mp4File)
	if err != nil {
		t.Errorf("Expected no error for MP4 file, got: %v", err)
	}
	if format != "video/mp4" {
		t.Errorf("Expected 'video/mp4', got: %s", format)
	}

	// Test PDF detection
	pdfFile := filepath.Join(tempDir, "test.pdf")
	pdfData := []byte{0x25, 0x50, 0x44, 0x46, 0x2D, 0x31, 0x2E, 0x34} // %PDF-1.4 header
	if err := os.WriteFile(pdfFile, pdfData, 0600); err != nil {
		t.Fatalf("Failed to write PDF test file: %v", err)
	}

	format, err = DetectFileFormat(pdfFile)
	if err != nil {
		t.Errorf("Expected no error for PDF file, got: %v", err)
	}
	if format != "application/pdf" {
		t.Errorf("Expected 'application/pdf', got: %s", format)
	}

	// Test unknown format
	unknownFile := filepath.Join(tempDir, "test.unknown")
	unknownData := []byte{0x01, 0x02, 0x03, 0x04, 0x05} // Unknown format
	if err := os.WriteFile(unknownFile, unknownData, 0600); err != nil {
		t.Fatalf("Failed to write unknown test file: %v", err)
	}

	_, err = DetectFileFormat(unknownFile)
	if err == nil {
		t.Error("Expected error for unknown file format, got nil")
	}
	if err != nil && err.Error() != "unknown file format" {
		t.Errorf("Expected 'unknown file format' error, got: %v", err)
	}

	// Test file not found
	nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
	_, err = DetectFileFormat(nonExistentFile)
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}

	// Test empty file
	emptyFile := filepath.Join(tempDir, "empty.txt")
	if err := os.WriteFile(emptyFile, []byte{}, 0600); err != nil {
		t.Fatalf("Failed to write empty test file: %v", err)
	}

	_, err = DetectFileFormat(emptyFile)
	if err == nil {
		t.Error("Expected error for empty file, got nil")
	}

	// Test file too small for signature
	smallFile := filepath.Join(tempDir, "small.txt")
	smallData := []byte{0x89} // Too small for PNG signature
	if err := os.WriteFile(smallFile, smallData, 0600); err != nil {
		t.Fatalf("Failed to write small test file: %v", err)
	}

	_, err = DetectFileFormat(smallFile)
	if err == nil {
		t.Error("Expected error for file too small, got nil")
	}
}

func TestValidateAttachmentIntegrityWithFormatValidation(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test PNG file
	pngHash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	pngPath := filepath.Join("attachments", "ab", pngHash)
	pngFullPath := filepath.Join(tempDir, pngPath)

	// Create directory structure
	if err := os.MkdirAll(filepath.Dir(pngFullPath), 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Write PNG signature (first 8 bytes are sufficient for format detection)
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if err := os.WriteFile(pngFullPath, pngData, 0600); err != nil {
		t.Fatalf("Failed to write PNG file: %v", err)
	}

	// Mock attachment reader that verifies successfully
	mockReader := &mockAttachmentReader{
		attachments: []*attachments.Attachment{
			{Hash: pngHash, Path: pngPath, Exists: true, Size: int64(len(pngData))},
		},
		verificationResults: map[string]bool{pngHash: true},
	}

	// Mock SMS reader with MMS that declares the attachment as JPEG (mismatch)
	mms := &sms.MMS{
		Parts: []sms.MMSPart{
			{
				ContentType:   "image/jpeg", // Declares as JPEG but file is PNG
				AttachmentRef: pngHash,
			},
		},
	}

	mockSMSReader := &mockSMSReader{
		availableYears: []int{2023}, // Must provide available years
		messages: map[int][]sms.Message{
			2023: {mms},
		},
		allAttachmentRefs: make(map[string]bool),
	}

	validator := NewAttachmentsValidator(tempDir, mockReader, mockSMSReader)
	violations := validator.ValidateAttachmentIntegrity()

	// Should have one format mismatch violation
	found := false
	for _, v := range violations {
		if v.Type == FormatMismatch {
			found = true
			if v.Expected != "image/jpeg" {
				t.Errorf("Expected 'image/jpeg', got: %s", v.Expected)
			}
			if v.Actual != "image/png" {
				t.Errorf("Expected 'image/png', got: %s", v.Actual)
			}
		}
	}

	if !found {
		t.Error("Expected format mismatch violation not found")
	}
}

func TestValidateAttachmentIntegrityWithUnknownFormat(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file with unknown format
	unknownHash := "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	unknownPath := filepath.Join("attachments", "fe", unknownHash)
	unknownFullPath := filepath.Join(tempDir, unknownPath)

	// Create directory structure
	if err := os.MkdirAll(filepath.Dir(unknownFullPath), 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Write unknown format data
	unknownData := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	if err := os.WriteFile(unknownFullPath, unknownData, 0600); err != nil {
		t.Fatalf("Failed to write unknown file: %v", err)
	}

	// Mock attachment reader that verifies successfully
	mockReader := &mockAttachmentReader{
		attachments: []*attachments.Attachment{
			{Hash: unknownHash, Path: unknownPath, Exists: true, Size: int64(len(unknownData))},
		},
		verificationResults: map[string]bool{unknownHash: true},
	}

	// Mock SMS reader with no MMS data
	mockSMSReader := &mockSMSReader{
		messages:          make(map[int][]sms.Message),
		allAttachmentRefs: make(map[string]bool),
	}

	validator := NewAttachmentsValidator(tempDir, mockReader, mockSMSReader)
	violations := validator.ValidateAttachmentIntegrity()

	// Should have one unknown format violation
	found := false
	for _, v := range violations {
		if v.Type == UnknownFormat {
			found = true
		}
	}

	if !found {
		t.Error("Expected unknown format violation not found")
	}
}
