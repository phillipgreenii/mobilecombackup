package validation

import (
	"context"
	"fmt"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/attachments"
	"github.com/phillipgreen/mobilecombackup/pkg/calls"
	"github.com/phillipgreen/mobilecombackup/pkg/contacts"
	"github.com/phillipgreen/mobilecombackup/pkg/sms"
)

// RepositoryValidator performs comprehensive repository validation using all reader APIs
type RepositoryValidator interface {
	// Legacy methods (deprecated but maintained for backward compatibility)
	// These delegate to context versions with context.Background()

	// Deprecated: ValidateRepository is deprecated. Use ValidateRepositoryContext instead.
	// This method will be removed in v2.0.0 (estimated 6 months).
	ValidateRepository() (*Report, error)

	// Deprecated: ValidateStructure is deprecated. Use ValidateStructureContext instead.
	// This method will be removed in v2.0.0 (estimated 6 months).
	ValidateStructure() []Violation

	// Deprecated: ValidateManifest is deprecated. Use ValidateManifestContext instead.
	// This method will be removed in v2.0.0 (estimated 6 months).
	ValidateManifest() []Violation

	// Deprecated: ValidateContent is deprecated. Use ValidateContentContext instead.
	// This method will be removed in v2.0.0 (estimated 6 months).
	ValidateContent() []Violation

	// Deprecated: ValidateConsistency is deprecated. Use ValidateConsistencyContext instead.
	// This method will be removed in v2.0.0 (estimated 6 months).
	ValidateConsistency() []Violation

	// Context-aware methods
	// These are the preferred methods for new code
	ValidateRepositoryContext(ctx context.Context) (*Report, error)
	ValidateStructureContext(ctx context.Context) []Violation
	ValidateManifestContext(ctx context.Context) []Violation
	ValidateContentContext(ctx context.Context) []Violation
	ValidateConsistencyContext(ctx context.Context) []Violation
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
	callsReader calls.Reader,
	smsReader sms.Reader,
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
// Deprecated: Use ValidateRepositoryContext instead. This method will be removed in v2.0.0.
func (v *RepositoryValidatorImpl) ValidateRepository() (*Report, error) {
	return v.ValidateRepositoryContext(context.Background())
}

// ValidateStructure validates overall repository structure
// Deprecated: Use ValidateStructureContext instead. This method will be removed in v2.0.0.
func (v *RepositoryValidatorImpl) ValidateStructure() []Violation {
	return v.ValidateStructureContext(context.Background())
}

// ValidateManifest validates files.yaml and checksums
// Deprecated: Use ValidateManifestContext instead. This method will be removed in v2.0.0.
func (v *RepositoryValidatorImpl) ValidateManifest() []Violation {
	return v.ValidateManifestContext(context.Background())
}

// ValidateContent validates all content files
// Deprecated: Use ValidateContentContext instead. This method will be removed in v2.0.0.
func (v *RepositoryValidatorImpl) ValidateContent() []Violation {
	return v.ValidateContentContext(context.Background())
}

// ValidateConsistency performs cross-file consistency validation
// Deprecated: Use ValidateConsistencyContext instead. This method will be removed in v2.0.0.
func (v *RepositoryValidatorImpl) ValidateConsistency() []Violation {
	return v.ValidateConsistencyContext(context.Background())
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

// Context-aware method implementations

// ValidateRepositoryContext performs complete repository validation with context support
func (v *RepositoryValidatorImpl) ValidateRepositoryContext(ctx context.Context) (*Report, error) {
	// Check context before starting
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

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
				Violation:    violation,
				SuggestedFix: v.markerFileValidator.GetSuggestedFix(),
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

	// Check context before each validation step
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Validate in logical order: structure -> manifest -> content -> consistency
	structureViolations := v.ValidateStructureContext(ctx)
	manifestViolations := v.ValidateManifestContext(ctx)
	contentViolations := v.ValidateContentContext(ctx)
	consistencyViolations := v.ValidateConsistencyContext(ctx)

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

// ValidateStructureContext validates overall repository structure with context support
func (v *RepositoryValidatorImpl) ValidateStructureContext(ctx context.Context) []Violation {
	// Check context before starting
	select {
	case <-ctx.Done():
		return nil
	default:
	}

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

// ValidateManifestContext validates files.yaml and checksums with context support
func (v *RepositoryValidatorImpl) ValidateManifestContext(ctx context.Context) []Violation {
	// Check context before starting
	select {
	case <-ctx.Done():
		return nil
	default:
	}

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

// ValidateContentContext validates all content files with context support
func (v *RepositoryValidatorImpl) ValidateContentContext(ctx context.Context) []Violation {
	// Check context before starting
	select {
	case <-ctx.Done():
		return nil
	default:
	}

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

// ValidateConsistencyContext performs cross-file consistency validation with context support
func (v *RepositoryValidatorImpl) ValidateConsistencyContext(ctx context.Context) []Violation {
	// Check context before starting
	select {
	case <-ctx.Done():
		return nil
	default:
	}

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
