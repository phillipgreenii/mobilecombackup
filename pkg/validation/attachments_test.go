package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/phillipgreen/mobilecombackup/pkg/attachments"
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

func (m *mockAttachmentReader) FindOrphanedAttachments(referencedHashes map[string]bool) ([]*attachments.Attachment, error) {
	if m.orphanedError != nil {
		return nil, m.orphanedError
	}
	return m.orphanedAttachments, nil
}

func (m *mockAttachmentReader) ValidateAttachmentStructure() error {
	return m.structureError
}

func TestAttachmentsValidatorImpl_ValidateAttachmentsStructure(t *testing.T) {
	tempDir := t.TempDir()

	// Test with no attachments (empty repository)
	mockReader := &mockAttachmentReader{
		attachments: []*attachments.Attachment{},
	}

	validator := NewAttachmentsValidator(tempDir, mockReader)
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

	validator := NewAttachmentsValidator(tempDir, mockReader)
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
	err := os.MkdirAll(attachmentsDir, 0755)
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

	validator := NewAttachmentsValidator(tempDir, mockReader)
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

	validator := NewAttachmentsValidator(tempDir, mockReader)
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

	validator := NewAttachmentsValidator(tempDir, mockReader)
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

	validator := NewAttachmentsValidator(tempDir, mockReader)
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

	validator := NewAttachmentsValidator(tempDir, mockReader)

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
