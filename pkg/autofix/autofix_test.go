package autofix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillipgreen/mobilecombackup/pkg/validation"
)

// TestAutofixer_CanFix tests which violation types can be fixed
func TestAutofixer_CanFix(t *testing.T) {
	autofixer := &AutofixerImpl{}

	tests := []struct {
		name           string
		violationType  validation.ViolationType
		expectedCanFix bool
	}{
		{"MissingFile", validation.MissingFile, true},
		{"MissingMarkerFile", validation.MissingMarkerFile, true},
		{"CountMismatch", validation.CountMismatch, true},
		{"SizeMismatch", validation.SizeMismatch, true},
		{"StructureViolation", validation.StructureViolation, true},
		{"ChecksumMismatch", validation.ChecksumMismatch, false},
		{"OrphanedAttachment", validation.OrphanedAttachment, false},
		{"InvalidFormat", validation.InvalidFormat, false},
		{"ExtraFile", validation.ExtraFile, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := autofixer.CanFix(test.violationType)
			if result != test.expectedCanFix {
				t.Errorf("CanFix(%s) = %v, want %v", test.violationType, result, test.expectedCanFix)
			}
		})
	}
}

// TestAutofixer_DryRunMode tests dry-run functionality
func TestAutofixer_DryRunMode(t *testing.T) {
	tempDir := t.TempDir()
	autofixer := NewAutofixer(tempDir, &NullProgressReporter{})

	violations := []validation.ValidationViolation{
		{
			Type:     validation.MissingFile,
			File:     ".mobilecombackup.yaml",
			Message:  "Missing marker file",
			Severity: validation.SeverityError,
		},
		{
			Type:     validation.StructureViolation,
			File:     "calls/",
			Message:  "Missing directory",
			Severity: validation.SeverityError,
		},
		{
			Type:     validation.CountMismatch,
			File:     "calls/calls-2024.xml",
			Message:  "Count mismatch",
			Expected: "12",
			Actual:   "56",
			Severity: validation.SeverityError,
		},
	}

	options := AutofixOptions{
		DryRun:  true,
		Verbose: false,
	}

	report, err := autofixer.FixViolations(violations, options)
	if err != nil {
		t.Fatalf("FixViolations failed: %v", err)
	}

	// Verify no actual changes were made
	if _, err := os.Stat(filepath.Join(tempDir, ".mobilecombackup.yaml")); !os.IsNotExist(err) {
		t.Error("Expected marker file to not exist in dry-run mode")
	}

	if _, err := os.Stat(filepath.Join(tempDir, "calls")); !os.IsNotExist(err) {
		t.Error("Expected calls directory to not exist in dry-run mode")
	}

	// Verify report shows what would be fixed
	if len(report.FixedViolations) == 0 {
		t.Error("Expected fixed violations to be reported in dry-run mode")
	}

	// Check for detailed dry-run messages
	foundCountMismatchDetail := false
	for _, fixed := range report.FixedViolations {
		if fixed.OriginalViolation.Type == validation.CountMismatch {
			if strings.Contains(fixed.Details, "from 56 to 12") {
				foundCountMismatchDetail = true
			}
		}
	}

	if !foundCountMismatchDetail {
		t.Error("Expected detailed count mismatch information in dry-run mode")
	}
}

// TestAutofixer_CreateMissingDirectories tests directory creation
func TestAutofixer_CreateMissingDirectories(t *testing.T) {
	tempDir := t.TempDir()
	autofixer := NewAutofixer(tempDir, &NullProgressReporter{})

	violations := []validation.ValidationViolation{
		{
			Type:     validation.StructureViolation,
			File:     "calls/",
			Message:  "Missing directory: calls/",
			Severity: validation.SeverityError,
		},
		{
			Type:     validation.StructureViolation,
			File:     "sms/",
			Message:  "Missing directory: sms/",
			Severity: validation.SeverityError,
		},
		{
			Type:     validation.StructureViolation,
			File:     "attachments/",
			Message:  "Missing directory: attachments/",
			Severity: validation.SeverityError,
		},
	}

	options := AutofixOptions{
		DryRun:  false,
		Verbose: false,
	}

	report, err := autofixer.FixViolations(violations, options)
	if err != nil {
		t.Fatalf("FixViolations failed: %v", err)
	}

	// Verify directories were created
	for _, dir := range []string{"calls", "sms", "attachments"} {
		dirPath := filepath.Join(tempDir, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("Expected directory %s to be created", dir)
		}
	}

	// Verify report
	if report.Summary.FixedCount != 3 {
		t.Errorf("Expected 3 fixed violations, got %d", report.Summary.FixedCount)
	}

	if report.Summary.ErrorCount != 0 {
		t.Errorf("Expected 0 errors, got %d", report.Summary.ErrorCount)
	}
}

// TestAutofixer_CreateMissingFiles tests file creation
func TestAutofixer_CreateMissingFiles(t *testing.T) {
	tempDir := t.TempDir()
	autofixer := NewAutofixer(tempDir, &NullProgressReporter{})

	violations := []validation.ValidationViolation{
		{
			Type:     validation.MissingMarkerFile,
			File:     ".mobilecombackup.yaml",
			Message:  "Missing marker file",
			Severity: validation.SeverityError,
		},
		{
			Type:     validation.MissingFile,
			File:     "contacts.yaml",
			Message:  "Missing contacts file",
			Severity: validation.SeverityError,
		},
		{
			Type:     validation.MissingFile,
			File:     "summary.yaml",
			Message:  "Missing summary file",
			Severity: validation.SeverityError,
		},
		{
			Type:     validation.MissingFile,
			File:     "files.yaml",
			Message:  "Missing files manifest",
			Severity: validation.SeverityError,
		},
		{
			Type:     validation.MissingFile,
			File:     "files.yaml.sha256",
			Message:  "Missing files manifest checksum",
			Severity: validation.SeverityError,
		},
	}

	options := AutofixOptions{
		DryRun:  false,
		Verbose: false,
	}

	report, err := autofixer.FixViolations(violations, options)
	if err != nil {
		t.Fatalf("FixViolations failed: %v", err)
	}

	// Verify files were created
	expectedFiles := []string{
		".mobilecombackup.yaml",
		"contacts.yaml",
		"summary.yaml",
		"files.yaml",
		"files.yaml.sha256",
	}

	for _, file := range expectedFiles {
		filePath := filepath.Join(tempDir, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file %s to be created", file)
		}
	}

	// Verify report
	if report.Summary.FixedCount != 5 {
		t.Errorf("Expected 5 fixed violations, got %d", report.Summary.FixedCount)
	}

	if report.Summary.ErrorCount != 0 {
		t.Errorf("Expected 0 errors, got %d", report.Summary.ErrorCount)
	}
}

// TestAutofixer_SkipUnsafeViolations tests that unsafe violations are skipped
func TestAutofixer_SkipUnsafeViolations(t *testing.T) {
	tempDir := t.TempDir()
	autofixer := NewAutofixer(tempDir, &NullProgressReporter{})

	violations := []validation.ValidationViolation{
		{
			Type:     validation.ChecksumMismatch,
			File:     "calls/calls-2024.xml",
			Message:  "SHA-256 mismatch",
			Severity: validation.SeverityError,
		},
		{
			Type:     validation.OrphanedAttachment,
			File:     "attachments/ab/abc123",
			Message:  "Orphaned attachment",
			Severity: validation.SeverityWarning,
		},
	}

	options := AutofixOptions{
		DryRun:  false,
		Verbose: false,
	}

	report, err := autofixer.FixViolations(violations, options)
	if err != nil {
		t.Fatalf("FixViolations failed: %v", err)
	}

	// Verify violations were skipped
	if report.Summary.SkippedCount != 2 {
		t.Errorf("Expected 2 skipped violations, got %d", report.Summary.SkippedCount)
	}

	// Verify skip reasons
	expectedReasons := map[validation.ViolationType]string{
		validation.ChecksumMismatch:   "Autofix preserves existing checksums to detect corruption",
		validation.OrphanedAttachment: "Use --remove-orphan-attachments flag",
	}

	for _, skipped := range report.SkippedViolations {
		expectedReason, exists := expectedReasons[skipped.OriginalViolation.Type]
		if !exists {
			t.Errorf("Unexpected skipped violation type: %s", skipped.OriginalViolation.Type)
			continue
		}

		if skipped.SkipReason != expectedReason {
			t.Errorf("Wrong skip reason for %s: got %q, want %q",
				skipped.OriginalViolation.Type, skipped.SkipReason, expectedReason)
		}
	}
}

// TestFixXMLCountAttribute tests XML count fixing functionality
func TestFixXMLCountAttribute(t *testing.T) {
	tests := []struct {
		name           string
		inputXML       string
		expectedCount  int
		expectedOutput string
	}{
		{
			name: "calls_count_mismatch",
			inputXML: `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="56">
  <call number="+15555550013" duration="33" date="1415054053956" type="2" />
  <call number="+15555550015" duration="20" date="1415057583749" type="2" />
  <call number="+15555550015" duration="22" date="1415057619663" type="2" />
</calls>`,
			expectedCount: 3,
			expectedOutput: `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="3">
  <call number="+15555550013" duration="33" date="1415054053956" type="2" />
  <call number="+15555550015" duration="20" date="1415057583749" type="2" />
  <call number="+15555550015" duration="22" date="1415057619663" type="2" />
</calls>`,
		},
		{
			name: "sms_count_mismatch",
			inputXML: `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="15">
  <sms protocol="0" address="7535" date="1373929642000" type="1" />
  <mms callback_set="0" text_only="1" sub="" date="1414697344000" />
</smses>`,
			expectedCount: 2,
			expectedOutput: `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="2">
  <sms protocol="0" address="7535" date="1373929642000" type="1" />
  <mms callback_set="0" text_only="1" sub="" date="1414697344000" />
</smses>`,
		},
		{
			name: "empty_document",
			inputXML: `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="5">
</calls>`,
			expectedCount: 0,
			expectedOutput: `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="0">
</calls>`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixedContent, actualCount, err := fixXMLCountAttribute([]byte(test.inputXML))
			if err != nil {
				t.Fatalf("fixXMLCountAttribute failed: %v", err)
			}

			if actualCount != test.expectedCount {
				t.Errorf("Expected count %d, got %d", test.expectedCount, actualCount)
			}

			if string(fixedContent) != test.expectedOutput {
				t.Errorf("Output mismatch.\nExpected:\n%s\nGot:\n%s", test.expectedOutput, string(fixedContent))
			}
		})
	}
}

// TestAutofixer_PermissionChecking tests permission checking in dry-run mode
func TestAutofixer_PermissionChecking(t *testing.T) {
	tempDir := t.TempDir()

	// Create a read-only subdirectory
	readOnlyDir := filepath.Join(tempDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0555); err != nil {
		t.Fatalf("Failed to create read-only directory: %v", err)
	}
	defer func() {
		// Restore permissions for cleanup
		_ = os.Chmod(readOnlyDir, 0750) // nolint:gosec // Cleanup permissions
	}()

	autofixer := NewAutofixer(readOnlyDir, &NullProgressReporter{})

	violations := []validation.ValidationViolation{
		{
			Type:     validation.MissingFile,
			File:     ".mobilecombackup.yaml",
			Message:  "Missing marker file",
			Severity: validation.SeverityError,
		},
	}

	options := AutofixOptions{
		DryRun:  true,
		Verbose: false,
	}

	report, err := autofixer.FixViolations(violations, options)
	if err != nil {
		t.Fatalf("FixViolations failed: %v", err)
	}

	// Check for permission errors in the report
	foundPermissionError := false
	for _, autofixErr := range report.Errors {
		if autofixErr.Operation == "permission_check" {
			foundPermissionError = true
			if !strings.Contains(autofixErr.Error, "not writable") {
				t.Errorf("Expected permission error message to contain 'not writable', got: %s", autofixErr.Error)
			}
			break
		}
	}

	if !foundPermissionError {
		t.Error("Expected permission error to be detected in dry-run mode")
	}
}

// TestAutofixer_XMLCountFix_Integration tests XML count fixing with real files
func TestAutofixer_XMLCountFix_Integration(t *testing.T) {
	tempDir := t.TempDir()
	autofixer := NewAutofixer(tempDir, &NullProgressReporter{})

	// Create test XML file with count mismatch
	callsDir := filepath.Join(tempDir, "calls")
	if err := os.MkdirAll(callsDir, 0755); err != nil {
		t.Fatalf("Failed to create calls directory: %v", err)
	}

	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="100">
  <call number="+15555550013" duration="33" date="1415054053956" type="2" />
  <call number="+15555550015" duration="20" date="1415057583749" type="2" />
</calls>`

	xmlFile := filepath.Join(callsDir, "calls-2024.xml")
	if err := os.WriteFile(xmlFile, []byte(testXML), 0644); err != nil {
		t.Fatalf("Failed to write test XML file: %v", err)
	}

	violations := []validation.ValidationViolation{
		{
			Type:     validation.CountMismatch,
			File:     "calls/calls-2024.xml",
			Message:  "Count mismatch: expected 2, got 100",
			Expected: "2",
			Actual:   "100",
			Severity: validation.SeverityError,
		},
	}

	options := AutofixOptions{
		DryRun:  false,
		Verbose: false,
	}

	report, err := autofixer.FixViolations(violations, options)
	if err != nil {
		t.Fatalf("FixViolations failed: %v", err)
	}

	// Verify the fix was applied
	if report.Summary.FixedCount != 1 {
		t.Errorf("Expected 1 fixed violation, got %d", report.Summary.FixedCount)
	}

	// Read the fixed file and verify the count was updated
	fixedContent, err := os.ReadFile(xmlFile) // nolint:gosec // Test-controlled path
	if err != nil {
		t.Fatalf("Failed to read fixed XML file: %v", err)
	}

	if !strings.Contains(string(fixedContent), `count="2"`) {
		t.Errorf("Expected count to be updated to 2, got: %s", string(fixedContent))
	}

	if strings.Contains(string(fixedContent), `count="100"`) {
		t.Error("Old count value should have been replaced")
	}
}

// TestAutofixer_ErrorHandling tests error handling and reporting
func TestAutofixer_ErrorHandling(t *testing.T) {
	// Use a non-existent directory to trigger errors
	nonExistentDir := "/this/path/does/not/exist"
	autofixer := NewAutofixer(nonExistentDir, &NullProgressReporter{})

	violations := []validation.ValidationViolation{
		{
			Type:     validation.MissingFile,
			File:     ".mobilecombackup.yaml",
			Message:  "Missing marker file",
			Severity: validation.SeverityError,
		},
	}

	options := AutofixOptions{
		DryRun:  false,
		Verbose: false,
	}

	report, err := autofixer.FixViolations(violations, options)
	if err != nil {
		t.Fatalf("FixViolations failed: %v", err)
	}

	// Should have errors due to non-existent directory
	if report.Summary.ErrorCount == 0 {
		t.Error("Expected errors due to non-existent directory")
	}

	// Verify error details
	foundFileError := false
	for _, autofixErr := range report.Errors {
		if autofixErr.ViolationType == validation.MissingFile {
			foundFileError = true
			if autofixErr.File != ".mobilecombackup.yaml" {
				t.Errorf("Expected error file to be .mobilecombackup.yaml, got %s", autofixErr.File)
			}
			if autofixErr.Operation != OperationCreateFile {
				t.Errorf("Expected operation to be %s, got %s", OperationCreateFile, autofixErr.Operation)
			}
		}
	}

	if !foundFileError {
		t.Error("Expected file creation error to be reported")
	}
}

// TestNewAutofixer tests autofixer creation
func TestNewAutofixer(t *testing.T) {
	tempDir := t.TempDir()

	// Test with null reporter
	autofixer1 := NewAutofixer(tempDir, nil)
	if autofixer1 == nil {
		t.Error("Expected autofixer to be created with nil reporter")
	}

	// Test with real reporter
	reporter := &NullProgressReporter{}
	autofixer2 := NewAutofixer(tempDir, reporter)
	if autofixer2 == nil {
		t.Error("Expected autofixer to be created with reporter")
	}

	// Verify the repository root is set correctly
	impl := autofixer2.(*AutofixerImpl)
	if impl.repositoryRoot != tempDir {
		t.Errorf("Expected repository root %s, got %s", tempDir, impl.repositoryRoot)
	}
}

// Helper function to create test violations
func createTestViolations() []validation.ValidationViolation {
	return []validation.ValidationViolation{
		{
			Type:     validation.MissingFile,
			File:     ".mobilecombackup.yaml",
			Message:  "Missing marker file",
			Severity: validation.SeverityError,
		},
		{
			Type:     validation.StructureViolation,
			File:     "calls/",
			Message:  "Missing directory",
			Severity: validation.SeverityError,
		},
		{
			Type:     validation.CountMismatch,
			File:     "calls/calls-2024.xml",
			Message:  "Count mismatch",
			Expected: "3",
			Actual:   "56",
			Severity: validation.SeverityError,
		},
		{
			Type:     validation.ChecksumMismatch,
			File:     "files.yaml",
			Message:  "Checksum mismatch",
			Severity: validation.SeverityError,
		},
	}
}

// TestAutofixer_ComprehensiveWorkflow tests a complete autofix workflow
func TestAutofixer_ComprehensiveWorkflow(t *testing.T) {
	tempDir := t.TempDir()
	autofixer := NewAutofixer(tempDir, &NullProgressReporter{})

	violations := createTestViolations()

	options := AutofixOptions{
		DryRun:  false,
		Verbose: true, // Test verbose mode
	}

	report, err := autofixer.FixViolations(violations, options)
	if err != nil {
		t.Fatalf("FixViolations failed: %v", err)
	}

	// Verify the summary counts
	expectedFixed := 2   // marker file + directory
	expectedSkipped := 1 // checksum mismatch
	expectedErrors := 1  // count mismatch (no XML file exists)

	if report.Summary.FixedCount != expectedFixed {
		t.Errorf("Expected %d fixed violations, got %d", expectedFixed, report.Summary.FixedCount)
	}

	if report.Summary.SkippedCount != expectedSkipped {
		t.Errorf("Expected %d skipped violations, got %d", expectedSkipped, report.Summary.SkippedCount)
	}

	if report.Summary.ErrorCount != expectedErrors {
		t.Errorf("Expected %d errors, got %d", expectedErrors, report.Summary.ErrorCount)
	}

	// Verify timestamp and repository path
	if report.RepositoryPath != tempDir {
		t.Errorf("Expected repository path %s, got %s", tempDir, report.RepositoryPath)
	}

	if report.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

// TestCheckWritePermission_ResourceCleanup tests that file handles are properly closed
func TestCheckWritePermission_ResourceCleanup(t *testing.T) {
	tempDir := t.TempDir()

	// Test directory permission check with proper resource cleanup
	err := checkWritePermission(tempDir)
	if err != nil {
		t.Errorf("checkWritePermission on directory failed: %v", err)
	}

	// Verify that temporary test file is cleaned up
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read temp directory: %v", err)
	}

	for _, file := range files {
		if strings.Contains(file.Name(), ".autofix-permission-test") {
			t.Errorf("Temporary test file %s was not cleaned up", file.Name())
		}
	}

	// Test file permission check with proper resource cleanup
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = checkWritePermission(testFile)
	if err != nil {
		t.Errorf("checkWritePermission on file failed: %v", err)
	}

	// Verify file is still accessible (not corrupted by permission test)
	content, err := os.ReadFile(testFile) // nolint:gosec // Test-controlled path
	if err != nil {
		t.Errorf("Test file was corrupted after permission check: %v", err)
	}
	if string(content) != "test" {
		t.Errorf("Test file content was modified after permission check")
	}
}

// TestCheckWritePermission_ErrorScenarios tests resource cleanup during error conditions
func TestCheckWritePermission_ErrorScenarios(t *testing.T) {
	// Test with non-existent path (should check parent directory)
	tempDir := t.TempDir()
	nonExistentPath := filepath.Join(tempDir, "nonexistent")

	err := checkWritePermission(nonExistentPath)
	if err != nil {
		t.Errorf("checkWritePermission with non-existent path failed: %v", err)
	}

	// Verify no temporary files left behind in parent directory
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read temp directory: %v", err)
	}

	for _, file := range files {
		if strings.Contains(file.Name(), ".autofix-permission-test") {
			t.Errorf("Temporary test file %s was not cleaned up after error scenario", file.Name())
		}
	}
}

// BenchmarkFixXMLCountAttribute benchmarks the XML count fixing function
func BenchmarkFixXMLCountAttribute(b *testing.B) {
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="1000">
  <call number="+15555550013" duration="33" date="1415054053956" type="2" />
  <call number="+15555550015" duration="20" date="1415057583749" type="2" />
  <call number="+15555550016" duration="22" date="1415057619663" type="2" />
  <call number="+15555550017" duration="45" date="1415057719663" type="1" />
  <call number="+15555550018" duration="12" date="1415057819663" type="2" />
</calls>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := fixXMLCountAttribute([]byte(testXML))
		if err != nil {
			b.Fatalf("fixXMLCountAttribute failed: %v", err)
		}
	}
}
