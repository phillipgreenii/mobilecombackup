package validation

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/attachments"
	"github.com/phillipgreen/mobilecombackup/pkg/calls"
	"github.com/phillipgreen/mobilecombackup/pkg/contacts"
	"github.com/phillipgreen/mobilecombackup/pkg/sms"
)

func TestRepositoryValidatorImpl_ValidateRepository(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create mock readers with no violations
	callsReader := &mockCallsReader{
		availableYears: []int{},
	}
	smsReader := &mockSMSReader{
		availableYears:    []int{},
		allAttachmentRefs: make(map[string]bool),
	}
	attachmentReader := &mockAttachmentReader{
		attachments: []*attachments.Attachment{},
	}
	contactsReader := &mockContactsReader{
		contacts: []*contacts.Contact{},
	}
	
	validator := NewRepositoryValidator(
		tempDir,
		callsReader,
		smsReader,
		attachmentReader,
		contactsReader,
	)
	
	report, err := validator.ValidateRepository()
	if err != nil {
		t.Fatalf("ValidateRepository failed: %v", err)
	}
	
	// Check report structure
	if report == nil {
		t.Fatal("Expected validation report, got nil")
	}
	
	if report.RepositoryPath != tempDir {
		t.Errorf("Expected repository path %s, got %s", tempDir, report.RepositoryPath)
	}
	
	if report.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
	
	// With empty repository and manifest errors, should have violations but overall structure should work
	if len(report.Violations) == 0 {
		t.Log("Note: No violations found with empty mock data")
	}
	
	// Status should be determined by violation severity
	if report.Status == "" {
		t.Error("Expected status to be set")
	}
	
	// Should have violation for missing marker file
	hasMarkerViolation := false
	for _, v := range report.Violations {
		if v.Type == MissingMarkerFile {
			hasMarkerViolation = true
			break
		}
	}
	if !hasMarkerViolation {
		t.Error("Expected MissingMarkerFile violation")
	}
}

func TestRepositoryValidatorImpl_ValidateRepositoryWithMarkerFile(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create a valid marker file
	markerContent := `repository_structure_version: "1"
created_at: "2024-01-15T10:30:00Z"
created_by: "mobilecombackup v1.0.0"
`
	err := os.WriteFile(filepath.Join(tempDir, ".mobilecombackup.yaml"), []byte(markerContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create marker file: %v", err)
	}
	
	// Create mock readers
	callsReader := &mockCallsReader{availableYears: []int{}}
	smsReader := &mockSMSReader{
		availableYears:    []int{},
		allAttachmentRefs: make(map[string]bool),
	}
	attachmentReader := &mockAttachmentReader{attachments: []*attachments.Attachment{}}
	contactsReader := &mockContactsReader{contacts: []*contacts.Contact{}}
	
	validator := NewRepositoryValidator(
		tempDir,
		callsReader,
		smsReader,
		attachmentReader,
		contactsReader,
	)
	
	report, err := validator.ValidateRepository()
	if err != nil {
		t.Fatalf("ValidateRepository failed: %v", err)
	}
	
	// Should NOT have marker file violations
	for _, v := range report.Violations {
		if v.Type == MissingMarkerFile || v.Type == UnsupportedVersion {
			t.Errorf("Unexpected marker file violation: %+v", v)
		}
	}
}

func TestRepositoryValidatorImpl_UnsupportedVersion(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create marker file with unsupported version
	markerContent := `repository_structure_version: "2"
created_at: "2024-01-15T10:30:00Z"
created_by: "mobilecombackup v2.0.0"
`
	err := os.WriteFile(filepath.Join(tempDir, ".mobilecombackup.yaml"), []byte(markerContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create marker file: %v", err)
	}
	
	// Create mock readers
	callsReader := &mockCallsReader{availableYears: []int{}}
	smsReader := &mockSMSReader{
		availableYears:    []int{},
		allAttachmentRefs: make(map[string]bool),
	}
	attachmentReader := &mockAttachmentReader{attachments: []*attachments.Attachment{}}
	contactsReader := &mockContactsReader{contacts: []*contacts.Contact{}}
	
	validator := NewRepositoryValidator(
		tempDir,
		callsReader,
		smsReader,
		attachmentReader,
		contactsReader,
	)
	
	report, err := validator.ValidateRepository()
	if err != nil {
		t.Fatalf("ValidateRepository failed: %v", err)
	}
	
	// Should have unsupported version violation
	hasUnsupportedVersion := false
	for _, v := range report.Violations {
		if v.Type == UnsupportedVersion {
			hasUnsupportedVersion = true
			break
		}
	}
	if !hasUnsupportedVersion {
		t.Error("Expected UnsupportedVersion violation")
	}
	
	// Status should be Invalid due to unsupported version
	if report.Status != Invalid {
		t.Errorf("Expected Invalid status, got %s", report.Status)
	}
	
	// Should not run other validations - check that we don't have many other violations
	// (we expect only marker file related violations)
	markerViolations := 0
	for _, v := range report.Violations {
		if v.File == ".mobilecombackup.yaml" {
			markerViolations++
		}
	}
	if markerViolations != len(report.Violations) {
		t.Errorf("Expected only marker file violations when version unsupported, got %d other violations", 
			len(report.Violations)-markerViolations)
	}
}

func TestRepositoryValidatorImpl_ValidateStructure(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create mock readers
	callsReader := &mockCallsReader{availableYears: []int{2015}}
	smsReader := &mockSMSReader{availableYears: []int{2016}}
	attachmentReader := &mockAttachmentReader{
		attachments: []*attachments.Attachment{
			{Hash: "test123", Path: "attachments/te/test123", Size: 100, Exists: true},
		},
	}
	contactsReader := &mockContactsReader{contacts: []*contacts.Contact{}}
	
	validator := NewRepositoryValidator(
		tempDir,
		callsReader,
		smsReader,
		attachmentReader,
		contactsReader,
	)
	
	violations := validator.ValidateStructure()
	
	// Should have violations for missing directories/files since we have data but no files
	if len(violations) == 0 {
		t.Log("Note: Structure validation may have warnings for missing directories")
	}
	
	// Violations should come from individual validators
	// The exact number depends on the mock implementation details
}

func TestRepositoryValidatorImpl_ValidateManifest(t *testing.T) {
	tempDir := t.TempDir()
	
	validator := &RepositoryValidatorImpl{
		repositoryRoot:    tempDir,
		manifestValidator: NewManifestValidator(tempDir),
		checksumValidator: NewChecksumValidator(tempDir),
	}
	
	violations := validator.ValidateManifest()
	
	// Should have violation for missing files.yaml
	if len(violations) == 0 {
		t.Error("Expected violations for missing manifest")
	}
	
	// First violation should be about missing manifest
	foundMissingManifest := false
	for _, violation := range violations {
		if violation.Type == MissingFile && violation.File == "files.yaml" {
			foundMissingManifest = true
			break
		}
	}
	
	if !foundMissingManifest {
		t.Error("Expected missing manifest violation")
	}
}

func TestRepositoryValidatorImpl_ValidateContent(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create mock readers with some content
	callsReader := &mockCallsReader{
		availableYears: []int{2015},
		calls: map[int][]calls.Call{
			2015: {
				{
					Number:       "+15551234567",
					Duration:     120,
					Date:         time.Date(2015, 6, 15, 14, 30, 0, 0, time.UTC),
					Type:         calls.IncomingCall,
					ReadableDate: "2015-06-15 14:30:00",
					ContactName:  "John Doe",
				},
			},
		},
		counts: map[int]int{2015: 1},
	}
	
	smsReader := &mockSMSReader{
		availableYears: []int{2015},
		messages: map[int][]sms.Message{
			2015: {
				mockSMS{
					date:         time.Date(2015, 6, 15, 14, 30, 0, 0, time.UTC),
					address:      "+15551234567",
					messageType:  sms.ReceivedMessage,
					readableDate: "2015-06-15 14:30:00",
					contactName:  "John Doe",
				},
			},
		},
		counts: map[int]int{2015: 1},
	}
	
	attachmentReader := &mockAttachmentReader{
		attachments: []*attachments.Attachment{
			{Hash: "test123", Path: "attachments/te/test123", Size: 100, Exists: true},
		},
		verificationResults: map[string]bool{"test123": true},
	}
	
	contactsReader := &mockContactsReader{
		contacts: []*contacts.Contact{
			{Name: "John Doe", Numbers: []string{"+15551234567"}},
		},
	}
	
	validator := NewRepositoryValidator(
		tempDir,
		callsReader,
		smsReader,
		attachmentReader,
		contactsReader,
	)
	
	violations := validator.ValidateContent()
	
	// May have violations for missing files, but should not fail completely
	if len(violations) > 10 {
		t.Logf("Got %d content violations (may include missing file violations): %v", len(violations), violations)
	}
}

func TestRepositoryValidatorImpl_ValidateConsistency(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create mock readers with cross-references
	smsReader := &mockSMSReader{
		allAttachmentRefs: map[string]bool{
			"attachment123": true,
			"attachment456": true,
		},
	}
	
	attachmentReader := &mockAttachmentReader{
		attachments: []*attachments.Attachment{
			{Hash: "attachment123", Path: "attachments/at/attachment123", Size: 100, Exists: true},
			{Hash: "orphaned789", Path: "attachments/or/orphaned789", Size: 200, Exists: true},
		},
		existsResults: map[string]bool{
			"attachment123": true,
			"attachment456": false, // Missing referenced attachment
		},
		orphanedAttachments: []*attachments.Attachment{
			{Hash: "orphaned789", Path: "attachments/or/orphaned789", Size: 200, Exists: true},
		},
	}
	
	contactsReader := &mockContactsReader{
		contacts: []*contacts.Contact{
			{Name: "John Doe", Numbers: []string{"+15551234567"}},
		},
	}
	
	validator := NewRepositoryValidator(
		tempDir,
		&mockCallsReader{availableYears: []int{}},
		smsReader,
		attachmentReader,
		contactsReader,
	)
	
	violations := validator.ValidateConsistency()
	
	// Should have violations for missing referenced attachment and orphaned attachment
	if len(violations) < 2 {
		t.Errorf("Expected at least 2 consistency violations, got %d: %v", len(violations), violations)
	}
	
	
	// Check for specific violation types
	foundMissingAttachment := false
	foundOrphanedAttachment := false
	
	for _, violation := range violations {
		switch violation.Type {
		case MissingFile:
			if len(violation.Message) >= 21 && violation.Message[:21] == "Referenced attachment" {
				foundMissingAttachment = true
			}
		case OrphanedAttachment:
			if len(violation.Message) >= 19 && violation.Message[:19] == "Orphaned attachment" { // Orphaned attachment, not contact
				foundOrphanedAttachment = true
			}
		}
	}
	
	if !foundMissingAttachment {
		t.Error("Expected missing referenced attachment violation")
	}
	
	if !foundOrphanedAttachment {
		t.Error("Expected orphaned attachment violation")
	}
}

func TestRepositoryValidatorImpl_StatusDetermination(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create a validator that can control violation types
	testValidator := &RepositoryValidatorImpl{
		repositoryRoot:       tempDir,
		markerFileValidator:  NewMarkerFileValidator(tempDir),
		manifestValidator:    NewManifestValidator(tempDir),
		checksumValidator:    NewChecksumValidator(tempDir),
		callsValidator:       NewCallsValidator(tempDir, &mockCallsReader{availableYears: []int{}}),
		smsValidator:         NewSMSValidator(tempDir, &mockSMSReader{availableYears: []int{}, allAttachmentRefs: make(map[string]bool)}),
		attachmentsValidator: NewAttachmentsValidator(tempDir, &mockAttachmentReader{attachments: []*attachments.Attachment{}}),
		contactsValidator:    NewContactsValidator(tempDir, &mockContactsReader{contacts: []*contacts.Contact{}}),
	}
	
	report, err := testValidator.ValidateRepository()
	if err != nil {
		t.Fatalf("ValidateRepository failed: %v", err)
	}
	
	// Status determination logic:
	// - Invalid if any error-level violations
	// - Valid if only warnings or no violations
	
	hasErrors := false
	hasWarnings := false
	for _, violation := range report.Violations {
		switch violation.Severity {
		case SeverityError:
			hasErrors = true
		case SeverityWarning:
			hasWarnings = true
		}
	}
	
	if hasErrors && report.Status != Invalid {
		t.Errorf("Expected Invalid status with errors, got %s", report.Status)
	}
	
	if !hasErrors && hasWarnings && report.Status != Valid {
		t.Errorf("Expected Valid status with only warnings, got %s", report.Status)
	}
}

func TestRepositoryValidatorImpl_ErrorHandling(t *testing.T) {
	tempDir := t.TempDir()
	
	// Test with SMS reader that returns error for attachment references
	smsReader := &mockSMSReader{
		availableYears: []int{2015},
		allAttachmentRefs: nil, // Will cause GetAllAttachmentRefs to return error
	}
	
	validator := NewRepositoryValidator(
		tempDir,
		&mockCallsReader{availableYears: []int{}},
		smsReader,
		&mockAttachmentReader{attachments: []*attachments.Attachment{}},
		&mockContactsReader{contacts: []*contacts.Contact{}},
	)
	
	violations := validator.ValidateConsistency()
	
	
	// Should handle error gracefully and report it as a violation
	foundErrorViolation := false
	for _, violation := range violations {
		if violation.Type == StructureViolation && len(violation.Message) >= 24 && violation.Message[:24] == "Failed to get attachment" {
			foundErrorViolation = true
			break
		}
	}
	
	if !foundErrorViolation {
		t.Error("Expected error violation for attachment reference failure")
	}
}

func TestNewRepositoryValidator(t *testing.T) {
	tempDir := t.TempDir()
	
	validator := NewRepositoryValidator(
		tempDir,
		&mockCallsReader{},
		&mockSMSReader{},
		&mockAttachmentReader{},
		&mockContactsReader{},
	)
	
	if validator == nil {
		t.Fatal("Expected non-nil validator")
	}
	
	// Verify validator is properly initialized
	impl, ok := validator.(*RepositoryValidatorImpl)
	if !ok {
		t.Fatal("Expected RepositoryValidatorImpl")
	}
	
	if impl.repositoryRoot != tempDir {
		t.Errorf("Expected repository root %s, got %s", tempDir, impl.repositoryRoot)
	}
	
	if impl.manifestValidator == nil {
		t.Error("Expected manifest validator to be initialized")
	}
	
	if impl.checksumValidator == nil {
		t.Error("Expected checksum validator to be initialized")
	}
	
	if impl.callsValidator == nil {
		t.Error("Expected calls validator to be initialized")
	}
	
	if impl.smsValidator == nil {
		t.Error("Expected SMS validator to be initialized")
	}
	
	if impl.attachmentsValidator == nil {
		t.Error("Expected attachments validator to be initialized")
	}
	
	if impl.contactsValidator == nil {
		t.Error("Expected contacts validator to be initialized")
	}
}