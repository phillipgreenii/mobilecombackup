package validation

import (
	"fmt"
	"path/filepath"

	"github.com/phillipgreen/mobilecombackup/pkg/attachments"
)

// AttachmentsValidator validates attachments directory and files using AttachmentReader
type AttachmentsValidator interface {
	// ValidateAttachmentsStructure validates attachments directory structure
	ValidateAttachmentsStructure() []ValidationViolation

	// ValidateAttachmentIntegrity verifies attachment content matches hashes
	ValidateAttachmentIntegrity() []ValidationViolation

	// ValidateAttachmentReferences checks attachment references against SMS data
	ValidateAttachmentReferences(referencedHashes map[string]bool) []ValidationViolation

	// GetOrphanedAttachments returns attachments not referenced by any messages
	GetOrphanedAttachments(referencedHashes map[string]bool) ([]*attachments.Attachment, error)
}

// AttachmentsValidatorImpl implements AttachmentsValidator interface
type AttachmentsValidatorImpl struct {
	repositoryRoot   string
	attachmentReader attachments.AttachmentReader
}

// NewAttachmentsValidator creates a new attachments validator
func NewAttachmentsValidator(repositoryRoot string, attachmentReader attachments.AttachmentReader) AttachmentsValidator {
	return &AttachmentsValidatorImpl{
		repositoryRoot:   repositoryRoot,
		attachmentReader: attachmentReader,
	}
}

// ValidateAttachmentsStructure validates attachments directory structure
func (v *AttachmentsValidatorImpl) ValidateAttachmentsStructure() []ValidationViolation {
	var violations []ValidationViolation

	// Check if attachments directory exists
	attachmentsDir := filepath.Join(v.repositoryRoot, "attachments")

	// Try to get attachments list to see if directory structure is valid
	attachmentsList, err := v.attachmentReader.ListAttachments()
	if err != nil {
		// If we can't list attachments, check if directory exists at all
		if !dirExists(attachmentsDir) {
			// Missing attachments directory is only an error if we have referenced attachments
			// For now, treat as warning since attachments are optional in empty repositories
			violations = append(violations, ValidationViolation{
				Type:     StructureViolation,
				Severity: SeverityWarning,
				File:     "attachments/",
				Message:  "Attachments directory not found (may be OK for repository without MMS)",
			})
		} else {
			violations = append(violations, ValidationViolation{
				Type:     StructureViolation,
				Severity: SeverityError,
				File:     "attachments/",
				Message:  fmt.Sprintf("Failed to list attachments: %v", err),
			})
		}
		return violations
	}

	// If we have no attachments, directory structure is optional
	if len(attachmentsList) == 0 {
		return violations
	}

	// Validate the attachment directory structure using the reader's validator
	if err := v.attachmentReader.ValidateAttachmentStructure(); err != nil {
		violations = append(violations, ValidationViolation{
			Type:     StructureViolation,
			Severity: SeverityError,
			File:     "attachments/",
			Message:  fmt.Sprintf("Attachment directory structure validation failed: %v", err),
		})
	}

	return violations
}

// ValidateAttachmentIntegrity verifies attachment content matches hashes
func (v *AttachmentsValidatorImpl) ValidateAttachmentIntegrity() []ValidationViolation {
	var violations []ValidationViolation

	// Get all attachments
	attachmentsList, err := v.attachmentReader.ListAttachments()
	if err != nil {
		violations = append(violations, ValidationViolation{
			Type:     StructureViolation,
			Severity: SeverityError,
			File:     "attachments/",
			Message:  fmt.Sprintf("Failed to list attachments for integrity check: %v", err),
		})
		return violations
	}

	// Verify each attachment's integrity
	for _, attachment := range attachmentsList {
		// Check if attachment exists
		if !attachment.Exists {
			violations = append(violations, ValidationViolation{
				Type:     MissingFile,
				Severity: SeverityError,
				File:     attachment.Path,
				Message:  fmt.Sprintf("Attachment file not found: %s", attachment.Hash),
			})
			continue
		}

		// Verify content matches hash
		verified, err := v.attachmentReader.VerifyAttachment(attachment.Hash)
		if err != nil {
			violations = append(violations, ValidationViolation{
				Type:     ChecksumMismatch,
				Severity: SeverityError,
				File:     attachment.Path,
				Message:  fmt.Sprintf("Failed to verify attachment %s: %v", attachment.Hash, err),
			})
			continue
		}

		if !verified {
			violations = append(violations, ValidationViolation{
				Type:     ChecksumMismatch,
				Severity: SeverityError,
				File:     attachment.Path,
				Message:  fmt.Sprintf("Attachment content doesn't match hash: %s", attachment.Hash),
				Expected: attachment.Hash,
				Actual:   "content hash mismatch",
			})
		}
	}

	return violations
}

// ValidateAttachmentReferences checks attachment references against SMS data
func (v *AttachmentsValidatorImpl) ValidateAttachmentReferences(referencedHashes map[string]bool) []ValidationViolation {
	var violations []ValidationViolation

	// Check for referenced attachments that don't exist
	for referencedHash := range referencedHashes {
		// Generate expected path for the attachment
		var expectedPath string
		if len(referencedHash) >= 2 {
			expectedPath = fmt.Sprintf("attachments/%s/%s", referencedHash[:2], referencedHash)
		} else {
			expectedPath = fmt.Sprintf("attachments/%s", referencedHash)
		}

		exists, err := v.attachmentReader.AttachmentExists(referencedHash)
		if err != nil {
			violations = append(violations, ValidationViolation{
				Type:     MissingFile,
				Severity: SeverityError,
				File:     expectedPath,
				Message:  fmt.Sprintf("Failed to check existence of referenced attachment %s: %v", referencedHash, err),
			})
			continue
		}

		if !exists {
			violations = append(violations, ValidationViolation{
				Type:     MissingFile,
				Severity: SeverityError,
				File:     expectedPath,
				Message:  fmt.Sprintf("Referenced attachment not found: %s", referencedHash),
			})
		}
	}

	// Find orphaned attachments (exist but not referenced)
	orphanedAttachments, err := v.attachmentReader.FindOrphanedAttachments(referencedHashes)
	if err != nil {
		violations = append(violations, ValidationViolation{
			Type:     StructureViolation,
			Severity: SeverityError,
			File:     "attachments/",
			Message:  fmt.Sprintf("Failed to find orphaned attachments: %v", err),
		})
		return violations
	}

	// Report orphaned attachments as warnings (not errors)
	for _, orphaned := range orphanedAttachments {
		violations = append(violations, ValidationViolation{
			Type:     OrphanedAttachment,
			Severity: SeverityWarning,
			File:     orphaned.Path,
			Message: fmt.Sprintf("Orphaned attachment not referenced by any message: %s (%d bytes)",
				orphaned.Hash, orphaned.Size),
		})
	}

	return violations
}

// GetOrphanedAttachments returns attachments not referenced by any messages
func (v *AttachmentsValidatorImpl) GetOrphanedAttachments(referencedHashes map[string]bool) ([]*attachments.Attachment, error) {
	return v.attachmentReader.FindOrphanedAttachments(referencedHashes)
}
