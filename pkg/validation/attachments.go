package validation

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/phillipgreen/mobilecombackup/pkg/attachments"
	"github.com/phillipgreen/mobilecombackup/pkg/sms"
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
	smsReader        sms.SMSReader
}

// NewAttachmentsValidator creates a new attachments validator
func NewAttachmentsValidator(repositoryRoot string, attachmentReader attachments.AttachmentReader, smsReader sms.SMSReader) AttachmentsValidator {
	return &AttachmentsValidatorImpl{
		repositoryRoot:   repositoryRoot,
		attachmentReader: attachmentReader,
		smsReader:        smsReader,
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

// ValidateAttachmentIntegrity verifies attachment content matches hashes and validates file formats
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

	// Get MIME type mappings from SMS/MMS data
	attachmentMimeTypes, err := v.getAttachmentMimeTypes()
	if err != nil {
		violations = append(violations, ValidationViolation{
			Type:     StructureViolation,
			Severity: SeverityError,
			File:     "attachments/",
			Message:  fmt.Sprintf("Failed to get attachment MIME types: %v", err),
		})
		// Continue with validation but without format checking
		attachmentMimeTypes = make(map[string]string)
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
			continue // Skip format validation for corrupted files
		}

		// Perform format validation - validate that file content matches expected MIME type
		attachmentPath := filepath.Join(v.repositoryRoot, attachment.Path)
		detectedFormat, err := DetectFileFormat(attachmentPath)
		if err != nil {
			// Unknown format - this is an error
			violations = append(violations, ValidationViolation{
				Type:     UnknownFormat,
				Severity: SeverityError,
				File:     attachment.Path,
				Message:  fmt.Sprintf("Unknown or unrecognized file format for attachment %s: %v", attachment.Hash, err),
			})
			continue
		}

		// Check if we have expected MIME type from SMS/MMS data
		if expectedMimeType, exists := attachmentMimeTypes[attachment.Hash]; exists {
			// Compare detected format with expected MIME type
			if detectedFormat != expectedMimeType {
				violations = append(violations, ValidationViolation{
					Type:     FormatMismatch,
					Severity: SeverityError,
					File:     attachment.Path,
					Message:  fmt.Sprintf("File format mismatch for attachment %s", attachment.Hash),
					Expected: expectedMimeType,
					Actual:   detectedFormat,
				})
			}
		}
		// Note: If no expected MIME type is available from SMS data, we only verify
		// that the format is recognized (not unknown), but don't enforce a specific type
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

// formatSignature represents a file format magic byte signature
type formatSignature struct {
	mimeType string
	magic    []byte
	offset   int
}

// formatSignatures contains known file format magic bytes
var formatSignatures = []formatSignature{
	{"image/png", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, 0},
	{"image/jpeg", []byte{0xFF, 0xD8, 0xFF}, 0},
	{"image/gif", []byte{0x47, 0x49, 0x46, 0x38}, 0},       // GIF87a or GIF89a (partial)
	{"video/mp4", []byte{0x66, 0x74, 0x79, 0x70}, 4},       // 'ftyp' at offset 4
	{"application/pdf", []byte{0x25, 0x50, 0x44, 0x46}, 0}, // '%PDF'
}

// DetectFileFormat reads the file header and detects the MIME type based on magic bytes
func DetectFileFormat(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read first 512 bytes for format detection
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read file header: %w", err)
	}

	// Truncate buffer to actual bytes read
	buffer = buffer[:n]

	// Check against known format signatures
	for _, sig := range formatSignatures {
		// Check if we have enough bytes
		if len(buffer) < sig.offset+len(sig.magic) {
			continue
		}

		// Check if magic bytes match at the expected offset
		match := true
		for i, b := range sig.magic {
			if buffer[sig.offset+i] != b {
				match = false
				break
			}
		}

		if match {
			return sig.mimeType, nil
		}
	}

	// No known format detected
	return "", fmt.Errorf("unknown file format")
}

// getAttachmentMimeTypes retrieves MIME type information from SMS/MMS messages
// Returns a map of attachment hash -> MIME type
func (v *AttachmentsValidatorImpl) getAttachmentMimeTypes() (map[string]string, error) {
	mimeTypes := make(map[string]string)

	// Skip if no SMS reader available
	if v.smsReader == nil {
		return mimeTypes, nil
	}

	// Get all available years with SMS data
	years, err := v.smsReader.GetAvailableYears()
	if err != nil {
		return nil, fmt.Errorf("failed to get available SMS years: %w", err)
	}

	// Process each year to extract attachment MIME types
	for _, year := range years {
		err := v.smsReader.StreamMessagesForYear(year, func(message sms.Message) error {
			// Only process MMS messages which can have attachments
			if mms, ok := message.(*sms.MMS); ok {
				// Process each part of the MMS
				for _, part := range mms.Parts {
					// Check if this part has an attachment reference (hash)
					if part.AttachmentRef != "" && part.ContentType != "" {
						// Store the MIME type for this attachment hash
						mimeTypes[part.AttachmentRef] = part.ContentType
					}
				}
			}
			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to stream messages for year %d: %w", year, err)
		}
	}

	return mimeTypes, nil
}
