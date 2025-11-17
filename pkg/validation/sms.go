package validation

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/sms"
)

// SMSValidator validates SMS directory and files using SMSReader
type SMSValidator interface {
	// ValidateSMSStructure validates SMS directory structure
	ValidateSMSStructure() []Violation

	// ValidateSMSContent validates SMS file content and consistency
	ValidateSMSContent() []Violation

	// ValidateSMSCounts verifies SMS message counts match manifest/summary
	ValidateSMSCounts(expectedCounts map[int]int) []Violation

	// GetAllAttachmentReferences returns all attachment references for cross-validation
	GetAllAttachmentReferences() (map[string]bool, error)
}

// SMSValidatorImpl implements SMSValidator interface
type SMSValidatorImpl struct {
	repositoryRoot string
	smsReader      sms.Reader
}

// NewSMSValidator creates a new SMS validator
func NewSMSValidator(repositoryRoot string, smsReader sms.Reader) SMSValidator {
	return &SMSValidatorImpl{
		repositoryRoot: repositoryRoot,
		smsReader:      smsReader,
	}
}

// ValidateSMSStructure validates SMS directory structure
func (v *SMSValidatorImpl) ValidateSMSStructure() []Violation {
	adapter := NewSMSReaderAdapter(v.smsReader)
	config := StructureValidationConfig{
		DirectoryName: "sms",
		FilePrefix:    "sms",
		ContentType:   "SMS",
	}
	return ValidateDirectoryStructure(adapter, v.repositoryRoot, config)
}

// ValidateSMSContent validates SMS file content and consistency
func (v *SMSValidatorImpl) ValidateSMSContent() []Violation {
	var violations []Violation

	// Get available years
	years, err := v.smsReader.GetAvailableYears(context.Background())
	if err != nil {
		violations = append(violations, Violation{
			Type:     StructureViolation,
			Severity: SeverityError,
			File:     "sms/",
			Message:  fmt.Sprintf("Failed to read available SMS years: %v", err),
		})
		return violations
	}

	// Validate each year file
	for _, year := range years {
		fileName := fmt.Sprintf("sms-%d.xml", year)
		filePath := filepath.Join("sms", fileName)

		// Validate file structure and year consistency
		if err := v.smsReader.ValidateSMSFile(context.Background(), year); err != nil {
			violations = append(violations, Violation{
				Type:     InvalidFormat,
				Severity: SeverityError,
				File:     filePath,
				Message:  fmt.Sprintf("SMS file validation failed: %v", err),
			})
			// Continue with other validations even if file validation fails
		}

		// Validate that all messages in the file belong to the correct year
		violations = append(violations, v.validateSMSYearConsistency(year)...)

		// Validate attachment references in MMS messages
		violations = append(violations, v.validateAttachmentReferences(year)...)
	}

	return violations
}

// validateSMSYearConsistency checks that all messages in a year file belong to that year
func (v *SMSValidatorImpl) validateSMSYearConsistency(year int) []Violation {
	var violations []Violation
	fileName := fmt.Sprintf("sms-%d.xml", year)
	filePath := filepath.Join("sms", fileName)

	// Stream messages to check year consistency without loading all into memory
	err := v.smsReader.StreamMessagesForYear(context.Background(), year, func(message sms.Message) error {
		// Convert epoch milliseconds to time for year extraction
		messageTime := time.Unix(message.GetDate()/1000, 0).UTC()
		messageYear := messageTime.Year()
		if messageYear != year {
			violations = append(violations, Violation{
				Type:     InvalidFormat,
				Severity: SeverityError,
				File:     filePath,
				Message: fmt.Sprintf("Message dated %s belongs to year %d but found in %d file",
					messageTime.Format("2006-01-02"), messageYear, year),
				Expected: fmt.Sprintf("year %d", year),
				Actual:   fmt.Sprintf("year %d", messageYear),
			})
		}
		return nil
	})

	if err != nil {
		violations = append(violations, Violation{
			Type:     InvalidFormat,
			Severity: SeverityError,
			File:     filePath,
			Message:  fmt.Sprintf("Failed to stream messages for year consistency check: %v", err),
		})
	}

	return violations
}

// validateAttachmentReferences checks that attachment references are valid
func (v *SMSValidatorImpl) validateAttachmentReferences(year int) []Violation {
	var violations []Violation
	fileName := fmt.Sprintf("sms-%d.xml", year)
	filePath := filepath.Join("sms", fileName)

	// Get attachment references for this year
	attachmentRefs, err := v.smsReader.GetAttachmentRefs(context.Background(), year)
	if err != nil {
		violations = append(violations, Violation{
			Type:     InvalidFormat,
			Severity: SeverityError,
			File:     filePath,
			Message:  fmt.Sprintf("Failed to get attachment references: %v", err),
		})
		return violations
	}

	// Validate attachment reference format
	for _, ref := range attachmentRefs {
		if ref == "" {
			violations = append(violations, Violation{
				Type:     InvalidFormat,
				Severity: SeverityWarning,
				File:     filePath,
				Message:  "Empty attachment reference found in MMS message",
			})
			continue
		}

		// Check if reference follows expected format: attachments/xx/xxxx...
		if len(ref) < 16 || ref[:12] != "attachments/" {
			violations = append(violations, Violation{
				Type:     InvalidFormat,
				Severity: SeverityWarning,
				File:     filePath,
				Message:  fmt.Sprintf("Invalid attachment reference format: %s", ref),
				Expected: "attachments/[2-char]/[hash]",
				Actual:   ref,
			})
		}
	}

	return violations
}

// ValidateSMSCounts verifies message counts match manifest/summary
func (v *SMSValidatorImpl) ValidateSMSCounts(expectedCounts map[int]int) []Violation {
	adapter := NewSMSReaderAdapter(v.smsReader)
	config := StructureValidationConfig{
		DirectoryName: "sms",
		FilePrefix:    "sms",
		ContentType:   "SMS",
	}
	return ValidateDataCounts(adapter, expectedCounts, config)
}

// GetAllAttachmentReferences returns all attachment references for cross-validation
func (v *SMSValidatorImpl) GetAllAttachmentReferences() (map[string]bool, error) {
	return v.smsReader.GetAllAttachmentRefs(context.Background())
}
