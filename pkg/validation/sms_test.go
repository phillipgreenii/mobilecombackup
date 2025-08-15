package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/sms"
)

// mockSMSReader implements SMSReader for testing
type mockSMSReader struct {
	availableYears    []int
	messages          map[int][]sms.Message
	counts            map[int]int
	attachmentRefs    map[int][]string
	allAttachmentRefs map[string]bool
	validateError     map[int]error
}

func (m *mockSMSReader) ReadMessages(year int) ([]sms.Message, error) {
	if messageList, exists := m.messages[year]; exists {
		return messageList, nil
	}
	return nil, fmt.Errorf("no messages for year %d", year)
}

func (m *mockSMSReader) StreamMessagesForYear(year int, callback func(sms.Message) error) error {
	if messageList, exists := m.messages[year]; exists {
		for _, message := range messageList {
			if err := callback(message); err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("no messages for year %d", year)
}

func (m *mockSMSReader) GetAttachmentRefs(year int) ([]string, error) {
	if refs, exists := m.attachmentRefs[year]; exists {
		return refs, nil
	}
	return []string{}, nil
}

func (m *mockSMSReader) GetAllAttachmentRefs() (map[string]bool, error) {
	if m.allAttachmentRefs == nil {
		return nil, fmt.Errorf("mock error: allAttachmentRefs is nil")
	}
	return m.allAttachmentRefs, nil
}

func (m *mockSMSReader) GetAvailableYears() ([]int, error) {
	return m.availableYears, nil
}

func (m *mockSMSReader) GetMessageCount(year int) (int, error) {
	if count, exists := m.counts[year]; exists {
		return count, nil
	}
	return 0, fmt.Errorf("no count for year %d", year)
}

func (m *mockSMSReader) ValidateSMSFile(year int) error {
	if err, exists := m.validateError[year]; exists {
		return err
	}
	return nil
}

// mockSMS implements sms.Message interface for testing
type mockSMS struct {
	date         int64
	address      string
	messageType  sms.MessageType
	readableDate string
	contactName  string
}

func (m mockSMS) GetDate() int64           { return m.date }
func (m mockSMS) GetAddress() string       { return m.address }
func (m mockSMS) GetType() sms.MessageType { return m.messageType }
func (m mockSMS) GetReadableDate() string  { return m.readableDate }
func (m mockSMS) GetContactName() string   { return m.contactName }

func TestSMSValidatorImpl_ValidateSMSStructure(t *testing.T) {
	tempDir := t.TempDir()

	mockReader := &mockSMSReader{
		availableYears: []int{2015, 2016},
	}

	validator := NewSMSValidator(tempDir, mockReader)

	// Test missing SMS directory
	violations := validator.ValidateSMSStructure()
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation for missing SMS directory, got %d", len(violations))
	}

	// Create SMS directory
	smsDir := filepath.Join(tempDir, "sms")
	err := os.MkdirAll(smsDir, 0750)
	if err != nil {
		t.Fatalf("Failed to create SMS directory: %v", err)
	}

	// Test missing SMS files
	violations = validator.ValidateSMSStructure()
	if len(violations) != 2 { // Missing both year files
		t.Errorf("Expected 2 violations for missing SMS files, got %d", len(violations))
	}

	// Create SMS files
	for _, year := range mockReader.availableYears {
		fileName := fmt.Sprintf("sms-%d.xml", year)
		filePath := filepath.Join(smsDir, fileName)
		err := os.WriteFile(filePath, []byte("<smses></smses>"), 0600)
		if err != nil {
			t.Fatalf("Failed to create SMS file: %v", err)
		}
	}

	// Test with all files present
	violations = validator.ValidateSMSStructure()
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations with all files present, got %d: %v", len(violations), violations)
	}
}

func TestSMSValidatorImpl_ValidateSMSStructure_EmptyRepository(t *testing.T) {
	tempDir := t.TempDir()

	// Test with no available years
	mockReader := &mockSMSReader{
		availableYears: []int{},
	}

	validator := NewSMSValidator(tempDir, mockReader)

	// Should not require SMS directory if no years available
	violations := validator.ValidateSMSStructure()
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations with no years available, got %d: %v", len(violations), violations)
	}
}

func TestSMSValidatorImpl_ValidateSMSContent(t *testing.T) {
	tempDir := t.TempDir()

	mockReader := &mockSMSReader{
		availableYears: []int{2015, 2016},
		messages: map[int][]sms.Message{
			2015: {
				mockSMS{
					date:         time.Date(2015, 6, 15, 14, 30, 0, 0, time.UTC).UnixMilli(),
					address:      "+15551234567",
					messageType:  sms.ReceivedMessage,
					readableDate: "2015-06-15 14:30:00",
					contactName:  "John Doe",
				},
				mockSMS{
					date:         time.Date(2016, 2, 10, 9, 15, 0, 0, time.UTC).UnixMilli(), // Wrong year!
					address:      "+15559876543",
					messageType:  sms.SentMessage,
					readableDate: "2016-02-10 09:15:00",
					contactName:  "Jane Smith",
				},
			},
			2016: {
				mockSMS{
					date:         time.Date(2016, 12, 25, 18, 0, 0, 0, time.UTC).UnixMilli(),
					address:      "+15555555555",
					messageType:  sms.ReceivedMessage,
					readableDate: "2016-12-25 18:00:00",
					contactName:  "Bob Johnson",
				},
			},
		},
		attachmentRefs: map[int][]string{
			2015: {"attachments/ab/ab1234567890abcdef", ""},                        // One valid, one empty
			2016: {"invalid_ref", "attachments/cd/cd1234567890abcdef1234567890ab"}, // One invalid, one valid
		},
		validateError: map[int]error{
			2016: fmt.Errorf("validation failed for 2016"),
		},
	}

	validator := NewSMSValidator(tempDir, mockReader)

	violations := validator.ValidateSMSContent()

	// Should have violations for:
	// 1. Year consistency issue in 2015 file (message from 2016)
	// 2. Validation error for 2016 file
	// 3. Empty attachment reference in 2015
	// 4. Invalid attachment reference in 2016
	if len(violations) < 4 {
		t.Errorf("Expected at least 4 violations, got %d: %v", len(violations), violations)
	}

	// Check for specific violation types
	foundYearViolation := false
	foundValidationViolation := false
	foundEmptyAttachmentViolation := false
	foundInvalidAttachmentViolation := false

	for _, violation := range violations {
		switch {
		case violation.File == "sms/sms-2015.xml" && violation.Type == InvalidFormat &&
			violation.Message != "" && violation.Message[:7] == "Message":
			foundYearViolation = true
		case violation.File == "sms/sms-2016.xml" && violation.Type == InvalidFormat &&
			violation.Message == "SMS file validation failed: validation failed for 2016":
			foundValidationViolation = true
		case violation.File == "sms/sms-2015.xml" && violation.Severity == SeverityWarning &&
			violation.Message == "Empty attachment reference found in MMS message":
			foundEmptyAttachmentViolation = true
		case violation.File == "sms/sms-2016.xml" && violation.Severity == SeverityWarning &&
			violation.Message == "Invalid attachment reference format: invalid_ref":
			foundInvalidAttachmentViolation = true
		}
	}

	if !foundYearViolation {
		t.Error("Expected year consistency violation for 2015 file")
	}

	if !foundValidationViolation {
		t.Error("Expected validation error violation for 2016 file")
	}

	if !foundEmptyAttachmentViolation {
		t.Error("Expected empty attachment reference violation for 2015 file")
	}

	if !foundInvalidAttachmentViolation {
		t.Error("Expected invalid attachment reference violation for 2016 file")
	}
}

func TestSMSValidatorImpl_ValidateSMSCounts(t *testing.T) {
	tempDir := t.TempDir()

	mockReader := &mockSMSReader{
		availableYears: []int{2015, 2016},
		counts: map[int]int{
			2015: 150,
			2016: 200,
		},
	}

	validator := NewSMSValidator(tempDir, mockReader)

	// Test with matching counts
	expectedCounts := map[int]int{
		2015: 150,
		2016: 200,
	}

	violations := validator.ValidateSMSCounts(expectedCounts)
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations with matching counts, got %d: %v", len(violations), violations)
	}

	// Test with mismatched counts
	expectedCountsMismatch := map[int]int{
		2015: 100, // Wrong count
		2016: 200,
	}

	violations = validator.ValidateSMSCounts(expectedCountsMismatch)
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation with mismatched count, got %d: %v", len(violations), violations)
	}

	if len(violations) > 0 && violations[0].Type != CountMismatch {
		t.Errorf("Expected CountMismatch violation, got %s", violations[0].Type)
	}

	// Test with missing expected year
	expectedCountsExtra := map[int]int{
		2015: 150,
		2016: 200,
		2017: 50, // Year that doesn't exist
	}

	violations = validator.ValidateSMSCounts(expectedCountsExtra)
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation for missing year, got %d: %v", len(violations), violations)
	}

	if len(violations) > 0 && violations[0].Type != MissingFile {
		t.Errorf("Expected MissingFile violation, got %s", violations[0].Type)
	}
}

func TestSMSValidatorImpl_ValidateSMSContent_ValidYear(t *testing.T) {
	tempDir := t.TempDir()

	// Test with all messages in correct years and valid attachment references
	mockReader := &mockSMSReader{
		availableYears: []int{2015},
		messages: map[int][]sms.Message{
			2015: {
				mockSMS{
					date:         time.Date(2015, 6, 15, 14, 30, 0, 0, time.UTC).UnixMilli(),
					address:      "+15551234567",
					messageType:  sms.ReceivedMessage,
					readableDate: "2015-06-15 14:30:00",
					contactName:  "John Doe",
				},
				mockSMS{
					date:         time.Date(2015, 12, 31, 23, 59, 0, 0, time.UTC).UnixMilli(),
					address:      "+15559876543",
					messageType:  sms.SentMessage,
					readableDate: "2015-12-31 23:59:00",
					contactName:  "Jane Smith",
				},
			},
		},
		attachmentRefs: map[int][]string{
			2015: {"attachments/ab/ab1234567890abcdef1234567890abcdef1234567890abcdef123456"},
		},
	}

	validator := NewSMSValidator(tempDir, mockReader)

	violations := validator.ValidateSMSContent()
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations with correct years and valid attachments, got %d: %v", len(violations), violations)
	}
}

func TestSMSValidatorImpl_GetAllAttachmentReferences(t *testing.T) {
	tempDir := t.TempDir()

	expectedRefs := map[string]bool{
		"attachments/ab/ab1234567890abcdef": true,
		"attachments/cd/cd9876543210fedcba": true,
	}

	mockReader := &mockSMSReader{
		allAttachmentRefs: expectedRefs,
	}

	validator := NewSMSValidator(tempDir, mockReader)

	refs, err := validator.GetAllAttachmentReferences()
	if err != nil {
		t.Fatalf("Failed to get attachment references: %v", err)
	}

	if len(refs) != len(expectedRefs) {
		t.Errorf("Expected %d attachment references, got %d", len(expectedRefs), len(refs))
	}

	for expectedRef := range expectedRefs {
		if !refs[expectedRef] {
			t.Errorf("Expected attachment reference %s not found", expectedRef)
		}
	}
}

func TestSMSValidatorImpl_ErrorHandling(t *testing.T) {
	tempDir := t.TempDir()

	// Test with reader that returns errors
	mockReader := &mockSMSReader{
		availableYears: []int{2015},
		counts:         map[int]int{},
	}

	// Test error in GetMessageCount
	validator := NewSMSValidator(tempDir, mockReader)
	violations := validator.ValidateSMSCounts(map[int]int{2015: 10})

	if len(violations) == 0 {
		t.Error("Expected violation when GetMessageCount returns error")
	}

	// Test with empty reader
	emptyReader := &mockSMSReader{
		availableYears: []int{},
	}

	validator = NewSMSValidator(tempDir, emptyReader)
	violations = validator.ValidateSMSStructure()

	// Empty years should not cause violations (empty repository is valid)
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations with empty years, got %d: %v", len(violations), violations)
	}
}

func TestSMSValidatorImpl_AttachmentReferenceValidation(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name        string
		attachments []string
		expectCount int
		expectType  Severity
	}{
		{
			name:        "valid attachments",
			attachments: []string{"attachments/ab/ab1234567890abcdef1234567890abcdef1234567890abcdef123456"},
			expectCount: 0,
		},
		{
			name:        "empty attachment reference",
			attachments: []string{""},
			expectCount: 1,
			expectType:  SeverityWarning,
		},
		{
			name:        "invalid format - wrong prefix",
			attachments: []string{"wrong/ab/ab1234567890abcdef"},
			expectCount: 1,
			expectType:  SeverityWarning,
		},
		{
			name:        "invalid format - too short",
			attachments: []string{"attachments/ab"},
			expectCount: 1,
			expectType:  SeverityWarning,
		},
		{
			name:        "mixed valid and invalid",
			attachments: []string{"attachments/ab/ab1234567890abcdef1234567890abcdef1234567890abcdef123456", "invalid", ""},
			expectCount: 2,
			expectType:  SeverityWarning,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockReader := &mockSMSReader{
				availableYears: []int{2015},
				messages: map[int][]sms.Message{
					2015: {
						mockSMS{
							date:         time.Date(2015, 6, 15, 14, 30, 0, 0, time.UTC).UnixMilli(),
							address:      "+15551234567",
							messageType:  sms.ReceivedMessage,
							readableDate: "2015-06-15 14:30:00",
							contactName:  "John Doe",
						},
					},
				},
				attachmentRefs: map[int][]string{
					2015: tc.attachments,
				},
			}

			validator := NewSMSValidator(tempDir, mockReader)
			violations := validator.ValidateSMSContent()

			warningCount := 0
			for _, violation := range violations {
				if violation.Severity == SeverityWarning {
					warningCount++
				}
			}

			if warningCount != tc.expectCount {
				t.Errorf("Expected %d warning violations, got %d: %v", tc.expectCount, warningCount, violations)
			}
		})
	}
}
