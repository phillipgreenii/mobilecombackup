package validation

import (
	"fmt"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/attachments"
	"github.com/phillipgreen/mobilecombackup/pkg/calls"
	"github.com/phillipgreen/mobilecombackup/pkg/contacts"
	"github.com/phillipgreen/mobilecombackup/pkg/sms"
)

// RepositoryValidator performs comprehensive repository validation using all reader APIs
type RepositoryValidator interface {
	// ValidateRepository performs complete repository validation
	ValidateRepository() (*Report, error)

	// ValidateStructure validates overall repository structure
	ValidateStructure() []Violation

	// ValidateManifest validates files.yaml and checksums
	ValidateManifest() []Violation

	// ValidateContent validates all content files
	ValidateContent() []Violation

	// ValidateConsistency performs cross-file consistency validation
	ValidateConsistency() []Violation
}

// RepositoryValidatorImpl implements comprehensive repository validation
type RepositoryValidatorImpl struct {
	repositoryRoot       string
	markerFileValidator  MarkerFileValidator
	manifestValidator    ManifestValidator
	checksumValidator    ChecksumValidator
	callsValidator       CallsValidator
	smsValidator         SMSValidator
	attachmentsValidator AttachmentsValidator
	contactsValidator    ContactsValidator
}

// NewRepositoryValidator creates a comprehensive repository validator
func NewRepositoryValidator(
	repositoryRoot string,
	callsReader calls.CallsReader,
	smsReader sms.SMSReader,
	attachmentReader attachments.AttachmentReader,
	contactsReader contacts.Reader,
) RepositoryValidator {
	return &RepositoryValidatorImpl{
		repositoryRoot:       repositoryRoot,
		markerFileValidator:  NewMarkerFileValidator(repositoryRoot),
		manifestValidator:    NewManifestValidator(repositoryRoot),
		checksumValidator:    NewChecksumValidator(repositoryRoot),
		callsValidator:       NewCallsValidator(repositoryRoot, callsReader),
		smsValidator:         NewSMSValidator(repositoryRoot, smsReader),
		attachmentsValidator: NewAttachmentsValidator(repositoryRoot, attachmentReader, smsReader),
		contactsValidator:    NewContactsValidator(repositoryRoot, contactsReader),
	}
}

// ValidateRepository performs complete repository validation
func (v *RepositoryValidatorImpl) ValidateRepository() (*Report, error) {
	report := &Report{
		Timestamp:      time.Now().UTC(),
		RepositoryPath: v.repositoryRoot,
		Status:         Valid,
		Violations:     []Violation{},
	}

	// Validate marker file first
	markerViolations, versionSupported, err := v.markerFileValidator.ValidateMarkerFile()
	if err != nil {
		return nil, fmt.Errorf("marker file validation error: %w", err)
	}

	// Handle fixable violations for missing marker file
	for _, violation := range markerViolations {
		if violation.Type == MissingMarkerFile {
			// Create a fixable violation with suggested content
			fixable := FixableViolation{
				Violation: violation,
				SuggestedFix:        v.markerFileValidator.GetSuggestedFix(),
			}
			report.Violations = append(report.Violations, fixable.Violation)
		} else {
			report.Violations = append(report.Violations, violation)
		}
	}

	// If version is not supported, stop further validation
	if !versionSupported {
		report.Status = Invalid
		return report, nil
	}

	// Validate in logical order: structure -> manifest -> content -> consistency
	structureViolations := v.ValidateStructure()
	manifestViolations := v.ValidateManifest()
	contentViolations := v.ValidateContent()
	consistencyViolations := v.ValidateConsistency()

	// Combine all violations
	report.Violations = append(report.Violations, structureViolations...)
	report.Violations = append(report.Violations, manifestViolations...)
	report.Violations = append(report.Violations, contentViolations...)
	report.Violations = append(report.Violations, consistencyViolations...)

	// Determine overall status
	hasErrors := false
	for _, violation := range report.Violations {
		if violation.Severity == SeverityError {
			hasErrors = true
			break
		}
	}

	if hasErrors {
		report.Status = Invalid
	} else if len(report.Violations) > 0 {
		// Only warnings
		report.Status = Valid // Valid with warnings
	}

	return report, nil
}

// ValidateStructure validates overall repository structure
func (v *RepositoryValidatorImpl) ValidateStructure() []Violation {
	var violations []Violation

	// Note: Individual validators handle directory and file structure checks
	// This method coordinates structure validation across all components

	// Validate each component's structure
	violations = append(violations, v.callsValidator.ValidateCallsStructure()...)
	violations = append(violations, v.smsValidator.ValidateSMSStructure()...)
	violations = append(violations, v.attachmentsValidator.ValidateAttachmentsStructure()...)
	violations = append(violations, v.contactsValidator.ValidateContactsStructure()...)

	return violations
}

// ValidateManifest validates files.yaml and checksums
func (v *RepositoryValidatorImpl) ValidateManifest() []Violation {
	var violations []Violation

	// Load and validate manifest format
	manifest, err := v.manifestValidator.LoadManifest()
	if err != nil {
		violations = append(violations, Violation{
			Type:     MissingFile,
			Severity: SeverityError,
			File:     "files.yaml",
			Message:  fmt.Sprintf("Failed to load manifest: %v", err),
		})
		return violations
	}

	// Validate manifest format
	violations = append(violations, v.manifestValidator.ValidateManifestFormat(manifest)...)

	// Check manifest completeness
	violations = append(violations, v.manifestValidator.CheckManifestCompleteness(manifest)...)

	// Verify manifest checksum
	if err := v.manifestValidator.VerifyManifestChecksum(); err != nil {
		violations = append(violations, Violation{
			Type:     ChecksumMismatch,
			Severity: SeverityError,
			File:     "files.yaml.sha256",
			Message:  fmt.Sprintf("Manifest checksum verification failed: %v", err),
		})
	}

	// Validate file checksums
	violations = append(violations, v.checksumValidator.ValidateManifestChecksums(manifest)...)

	return violations
}

// ValidateContent validates all content files
func (v *RepositoryValidatorImpl) ValidateContent() []Violation {
	var violations []Violation

	// Validate calls content
	violations = append(violations, v.callsValidator.ValidateCallsContent()...)

	// Validate SMS content
	violations = append(violations, v.smsValidator.ValidateSMSContent()...)

	// Validate attachment integrity
	violations = append(violations, v.attachmentsValidator.ValidateAttachmentIntegrity()...)

	// Validate contacts data
	violations = append(violations, v.contactsValidator.ValidateContactsData()...)

	return violations
}

// ValidateConsistency performs cross-file consistency validation
func (v *RepositoryValidatorImpl) ValidateConsistency() []Violation {
	var violations []Violation

	// Get attachment references from SMS
	referencedAttachments, err := v.smsValidator.GetAllAttachmentReferences()
	if err != nil {
		violations = append(violations, Violation{
			Type:     StructureViolation,
			Severity: SeverityError,
			File:     "sms/",
			Message:  fmt.Sprintf("Failed to get attachment references for consistency check: %v", err),
		})
	} else {
		// Validate attachment references
		violations = append(violations, v.attachmentsValidator.ValidateAttachmentReferences(referencedAttachments)...)
	}

	// Collect contact references from calls and SMS
	callContactRefs, smsContactRefs := v.collectContactReferences()

	// Validate contact references
	violations = append(violations, v.contactsValidator.ValidateContactReferences(callContactRefs, smsContactRefs)...)

	// TODO: Add summary.yaml validation when available
	// This would validate counts and statistics against actual data

	return violations
}

// collectContactReferences gathers contact names referenced in calls and SMS
func (v *RepositoryValidatorImpl) collectContactReferences() (map[string]bool, map[string]bool) {
	callContacts := make(map[string]bool)
	smsContacts := make(map[string]bool)

	// This is a simplified implementation - in a real scenario, we would
	// need to add methods to the readers to extract contact names efficiently
	// For now, we return empty maps to avoid errors

	// TODO: Implement contact extraction from calls and SMS
	// This would require extending the reader interfaces or implementing
	// streaming methods to extract contact names without loading all data

	return callContacts, smsContacts
}
