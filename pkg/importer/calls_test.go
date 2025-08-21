package importer

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/phillipgreenii/mobilecombackup/pkg/calls"
	"github.com/phillipgreenii/mobilecombackup/pkg/logging"
	"github.com/phillipgreenii/mobilecombackup/pkg/manifest"
	"github.com/phillipgreenii/mobilecombackup/pkg/sms"
)

func TestCallValidator_Validate(t *testing.T) {
	validator := NewCallValidator()

	tests := []struct {
		name       string
		call       *calls.Call
		violations []string
	}{
		{
			name: "valid call",
			call: &calls.Call{
				Number:   "+1234567890",
				Duration: 120,
				Date:     1609459200000,
				Type:     calls.Incoming,
			},
			violations: nil,
		},
		{
			name: "missing timestamp",
			call: &calls.Call{
				Number:   "+1234567890",
				Duration: 120,
				Date:     0,
				Type:     calls.Incoming,
			},
			violations: []string{"missing-timestamp"},
		},
		{
			name: "negative timestamp",
			call: &calls.Call{
				Number:   "+1234567890",
				Duration: 120,
				Date:     -1,
				Type:     calls.Incoming,
			},
			violations: []string{"missing-timestamp"},
		},
		{
			name: "missing number",
			call: &calls.Call{
				Number:   "",
				Duration: 120,
				Date:     1609459200000,
				Type:     calls.Incoming,
			},
			violations: []string{"missing-number"},
		},
		{
			name: "whitespace only number",
			call: &calls.Call{
				Number:   "   ",
				Duration: 120,
				Date:     1609459200000,
				Type:     calls.Incoming,
			},
			violations: []string{"missing-number"},
		},
		{
			name: "invalid call type",
			call: &calls.Call{
				Number:   "+1234567890",
				Duration: 120,
				Date:     1609459200000,
				Type:     99, // Invalid type
			},
			violations: []string{"invalid-type: 99"},
		},
		{
			name: "negative duration",
			call: &calls.Call{
				Number:   "+1234567890",
				Duration: -10,
				Date:     1609459200000,
				Type:     calls.Incoming,
			},
			violations: []string{"negative-duration"},
		},
		{
			name: "multiple violations",
			call: &calls.Call{
				Number:   "",
				Duration: -10,
				Date:     0,
				Type:     99,
			},
			violations: []string{
				"missing-timestamp",
				"missing-number",
				"invalid-type: 99",
				"negative-duration",
			},
		},
		{
			name: "valid incoming call",
			call: &calls.Call{
				Number:   "5551234567",
				Duration: 0, // Missed call can have 0 duration
				Date:     1609459200000,
				Type:     calls.Incoming,
			},
			violations: nil,
		},
		{
			name: "valid outgoing call",
			call: &calls.Call{
				Number:   "+15551234567",
				Duration: 300,
				Date:     1609459200000,
				Type:     calls.Outgoing,
			},
			violations: nil,
		},
		{
			name: "valid missed call",
			call: &calls.Call{
				Number:   "1234567",
				Duration: 0,
				Date:     1609459200000,
				Type:     calls.Missed,
			},
			violations: nil,
		},
		{
			name: "valid voicemail",
			call: &calls.Call{
				Number:   "+15551234567",
				Duration: 45,
				Date:     1609459200000,
				Type:     calls.Voicemail,
			},
			violations: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := validator.Validate(tt.call)

			if !reflect.DeepEqual(violations, tt.violations) {
				t.Errorf("Expected violations %v, got %v", tt.violations, violations)
			}
		})
	}
}

func TestCallValidator_PhoneNumberFormats(t *testing.T) {
	validator := NewCallValidator()

	// Test various phone number formats that should be valid
	validNumbers := []string{
		"1234567",         // Short number
		"5551234567",      // 10-digit US
		"15551234567",     // 11-digit US
		"+15551234567",    // International format
		"+441234567890",   // UK number
		"911",             // Emergency
		"*67",             // Special code
		"#31#",            // Special code with hash
		"+86138000138000", // China mobile
	}

	for _, number := range validNumbers {
		call := &calls.Call{
			Number:   number,
			Duration: 60,
			Date:     1609459200000,
			Type:     calls.Incoming,
		}

		violations := validator.Validate(call)
		if len(violations) > 0 {
			t.Errorf("Phone number %q should be valid, got violations: %v", number, violations)
		}
	}
}

// TestImporter_ErrorContextPreservation tests that errors maintain proper context through the call chain
func TestImporter_ErrorContextPreservation(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for the test repository
	tempDir := t.TempDir()

	// Create options with an invalid file to trigger file processing errors
	invalidFile := "/nonexistent/path/that/does/not/exist.xml"

	options := &ImportOptions{
		RepoRoot: tempDir,
		Paths:    []string{invalidFile},
		Quiet:    true,
	}

	// Try to create importer - this will fail due to invalid repository
	_, err := NewImporter(options, logging.NewNullLogger())

	// Verify error contains proper context
	if err == nil {
		t.Fatal("Expected error but got none")
	}

	errorMsg := err.Error()

	// Check that error message contains context from validation
	if !strings.Contains(errorMsg, "invalid repository") {
		t.Errorf("Error message missing repository context. Got: %s", errorMsg)
	}

	// Verify this is proper error wrapping by checking the error chain
	if !strings.Contains(errorMsg, "validation failed") {
		t.Errorf("Error message missing validation context. Got: %s", errorMsg)
	}

	// Test with valid repository but nonexistent file to trigger file processing error
	repoRoot := tempDir + "/valid_repo"
	setupValidRepository(t, repoRoot)

	validOptions := &ImportOptions{
		RepoRoot: repoRoot,
		Paths:    []string{invalidFile}, // File that doesn't exist
		Quiet:    true,
	}

	importer, err := NewImporter(validOptions, logging.NewNullLogger())
	if err != nil {
		t.Fatalf("Failed to create importer with valid repository: %v", err)
	}

	// Attempt import, which should fail when trying to process the nonexistent file
	_, err = importer.Import()

	if err == nil {
		t.Fatal("Expected error for nonexistent file but got none")
	}

	errorMsg = err.Error()

	// Verify error context is preserved through the call chain
	expectedContexts := []string{
		"failed to process import files", // From main Import method
		"stat",                           // From file stat operation
	}

	for _, expectedContext := range expectedContexts {
		if !strings.Contains(errorMsg, expectedContext) {
			t.Errorf("Error message missing expected context '%s'. Got: %s", expectedContext, errorMsg)
		}
	}

	// Verify error chain is preserved (can unwrap to original error)
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) {
		t.Errorf("Expected to unwrap to os.PathError, but couldn't. Error chain: %v", err)
	}
}

// setupValidRepository creates a minimal valid repository for testing
func setupValidRepository(t *testing.T, repoRoot string) {
	t.Helper()

	// Create directory structure
	dirs := []string{
		repoRoot,
		filepath.Join(repoRoot, "calls"),
		filepath.Join(repoRoot, "sms"),
		filepath.Join(repoRoot, "attachments"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create marker file
	markerContent := `repository_structure_version: "1"
created_at: "2024-01-15T10:30:00Z"
created_by: "mobilecombackup v1.0.0"
`
	markerPath := filepath.Join(repoRoot, ".mobilecombackup.yaml")
	if err := os.WriteFile(markerPath, []byte(markerContent), 0600); err != nil {
		t.Fatalf("Failed to create marker file: %v", err)
	}

	// Create empty contacts file
	contactsPath := filepath.Join(repoRoot, "contacts.yaml")
	contactsContent := "contacts: []\n"
	if err := os.WriteFile(contactsPath, []byte(contactsContent), 0600); err != nil {
		t.Fatalf("Failed to create contacts file: %v", err)
	}

	// Create summary file
	summaryPath := filepath.Join(repoRoot, "summary.yaml")
	summaryContent := `counts:
  calls: 0
  sms: 0
`
	if err := os.WriteFile(summaryPath, []byte(summaryContent), 0600); err != nil {
		t.Fatalf("Failed to create summary file: %v", err)
	}

	// Generate and write manifest files
	manifestGenerator := manifest.NewManifestGenerator(repoRoot)
	fileManifest, err := manifestGenerator.GenerateFileManifest()
	if err != nil {
		t.Fatalf("Failed to generate file manifest: %v", err)
	}

	if err := manifestGenerator.WriteManifestFiles(fileManifest); err != nil {
		t.Fatalf("Failed to write manifest files: %v", err)
	}
}

// TestDependencyInjection tests that the new dependency injection constructors work correctly
func TestDependencyInjection(t *testing.T) {
	tempDir := t.TempDir()
	setupValidRepository(t, tempDir)

	options := &ImportOptions{
		RepoRoot: tempDir,
		Paths:    []string{},
	}

	// Create mock dependencies
	mockContactsManager := NewMockContactsManager()
	mockContactsManager.SetContact("1234567890", "John Doe")

	mockAttachmentStorage := NewMockAttachmentStorage()

	// Create real readers for testing
	callsReader := calls.NewXMLCallsReader(tempDir)
	smsReader := sms.NewXMLSMSReader(tempDir)

	// Test NewImporterWithDependencies
	importer, err := NewImporterWithDependencies(
		options,
		mockContactsManager,
		callsReader,
		smsReader,
		mockAttachmentStorage,
		logging.NewNullLogger(),
	)

	if err != nil {
		t.Fatalf("NewImporterWithDependencies failed: %v", err)
	}

	if importer == nil {
		t.Fatal("NewImporterWithDependencies returned nil importer")
	}

	// Verify the importer has the injected dependencies
	if importer.contactsManager == nil {
		t.Error("contactsManager not set")
	}

	// Test that the mock contacts manager is working
	name, exists := importer.contactsManager.GetContactByNumber("1234567890")
	if !exists || name != "John Doe" {
		t.Errorf("Expected contact 'John Doe' for number '1234567890', got %s, exists: %v", name, exists)
	}
}

// TestCallsImporterDependencyInjection tests CallsImporter with dependency injection
func TestCallsImporterDependencyInjection(t *testing.T) {
	tempDir := t.TempDir()
	setupValidRepository(t, tempDir)

	options := &ImportOptions{
		RepoRoot: tempDir,
		Paths:    []string{},
	}

	// Create mock dependencies
	mockContactsManager := NewMockContactsManager()
	yearTracker := NewYearTracker()
	callsReader := calls.NewXMLCallsReader(tempDir)

	// Test NewCallsImporterWithDependencies
	callsImporter, err := NewCallsImporterWithDependencies(
		options,
		mockContactsManager,
		yearTracker,
		callsReader,
	)

	if err != nil {
		t.Fatalf("NewCallsImporterWithDependencies failed: %v", err)
	}

	if callsImporter == nil {
		t.Fatal("NewCallsImporterWithDependencies returned nil importer")
	}

	// Verify the importer has the injected dependencies
	if callsImporter.contactsManager == nil {
		t.Error("contactsManager not set in CallsImporter")
	}
}

// TestSMSImporterDependencyInjection tests SMSImporter with dependency injection
func TestSMSImporterDependencyInjection(t *testing.T) {
	tempDir := t.TempDir()
	setupValidRepository(t, tempDir)

	options := &ImportOptions{
		RepoRoot: tempDir,
		Paths:    []string{},
	}

	// Create mock dependencies
	mockContactsManager := NewMockContactsManager()
	mockAttachmentStorage := NewMockAttachmentStorage()
	yearTracker := NewYearTracker()
	smsReader := sms.NewXMLSMSReader(tempDir)

	// Test NewSMSImporter with interface (dependency injection)
	smsImporter := NewSMSImporter(options, mockContactsManager, yearTracker)

	if smsImporter == nil {
		t.Fatal("NewSMSImporter returned nil importer")
	}

	// Verify the importer has the injected dependencies
	if smsImporter.contactsManager == nil {
		t.Error("contactsManager not set in SMSImporter")
	}

	// Use the variables to prevent "declared and not used" errors
	_ = mockAttachmentStorage
	_ = smsReader
}
