package autofix

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillipgreen/mobilecombackup/pkg/validation"
)

// TestAutofixer_Integration_FullRepository tests autofix on a complete repository structure
func TestAutofixer_Integration_FullRepository(t *testing.T) {
	tempDir := t.TempDir()

	// Create a repository with various violations
	setupRepositoryWithViolations(t, tempDir)

	autofixer := NewAutofixer(tempDir, &NullProgressReporter{})

	// Simulate violations that would be found by validation
	violations := []validation.Violation{
		{
			Type:     validation.StructureViolation,
			File:     "calls/",
			Message:  "Missing directory: calls/",
			Severity: validation.SeverityError,
		},
		{
			Type:     validation.StructureViolation,
			File:     "attachments/",
			Message:  "Missing directory: attachments/",
			Severity: validation.SeverityError,
		},
		{
			Type:     validation.MissingFile,
			File:     ".mobilecombackup.yaml",
			Message:  "Missing marker file",
			Severity: validation.SeverityError,
		},
		{
			Type:     validation.CountMismatch,
			File:     "sms/sms-2024.xml",
			Message:  "Count mismatch in SMS file",
			Expected: "2",
			Actual:   "10",
			Severity: validation.SeverityError,
		},
		{
			Type:     validation.MissingFile,
			File:     "files.yaml",
			Message:  "Missing files manifest",
			Severity: validation.SeverityError,
		},
	}

	options := Options{
		DryRun:  false,
		Verbose: true,
	}

	report, err := autofixer.FixViolations(violations, options)
	if err != nil {
		t.Fatalf("FixViolations failed: %v", err)
	}

	// Verify all fixable violations were resolved
	expectedFixed := 5 // calls directory, attachments directory, marker file, XML count, files.yaml
	if report.Summary.FixedCount != expectedFixed {
		t.Errorf("Expected %d fixed violations, got %d", expectedFixed, report.Summary.FixedCount)

		// Debug: print what was actually fixed
		for _, fixed := range report.FixedViolations {
			t.Logf("Fixed: %s - %s", fixed.FixAction, fixed.Details)
		}
	}

	// Verify repository structure was created
	expectedDirs := []string{"calls", "sms", "attachments"}
	for _, dir := range expectedDirs {
		dirPath := filepath.Join(tempDir, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("Expected directory %s to exist after autofix", dir)
		}
	}

	// Verify marker file was created
	markerPath := filepath.Join(tempDir, ".mobilecombackup.yaml")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Error("Expected marker file to be created")
	}

	// Verify XML count was fixed
	smsPath := filepath.Join(tempDir, "sms", "sms-2024.xml")
	if content, err := os.ReadFile(smsPath); err == nil { // nolint:gosec // Test-controlled path
		if !strings.Contains(string(content), `count="2"`) {
			t.Error("Expected SMS count to be fixed to 2")
		}
	}

	// Verify files.yaml was created
	filesPath := filepath.Join(tempDir, "files.yaml")
	if _, err := os.Stat(filesPath); os.IsNotExist(err) {
		t.Error("Expected files.yaml to be created")
	}
}

// TestAutofixer_Integration_DryRunVsRealRun compares dry-run and real execution
func TestAutofixer_Integration_DryRunVsRealRun(t *testing.T) {
	// Set up two identical test directories
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	setupRepositoryWithViolations(t, tempDir1)
	setupRepositoryWithViolations(t, tempDir2)

	violations := []validation.Violation{
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
	}

	// Run dry-run mode
	autofixer1 := NewAutofixer(tempDir1, &NullProgressReporter{})
	dryRunReport, err := autofixer1.FixViolations(violations, Options{DryRun: true})
	if err != nil {
		t.Fatalf("Dry-run failed: %v", err)
	}

	// Run real mode
	autofixer2 := NewAutofixer(tempDir2, &NullProgressReporter{})
	realReport, err := autofixer2.FixViolations(violations, Options{DryRun: false})
	if err != nil {
		t.Fatalf("Real run failed: %v", err)
	}

	// Compare reports - should have same number of planned fixes
	if len(dryRunReport.FixedViolations) != len(realReport.FixedViolations) {
		t.Errorf("Dry-run and real run should plan same number of fixes: dry-run=%d, real=%d",
			len(dryRunReport.FixedViolations), len(realReport.FixedViolations))
	}

	// Verify dry-run didn't make changes
	if _, err := os.Stat(filepath.Join(tempDir1, ".mobilecombackup.yaml")); !os.IsNotExist(err) {
		t.Error("Dry-run should not create files")
	}

	if _, err := os.Stat(filepath.Join(tempDir1, "calls")); !os.IsNotExist(err) {
		t.Error("Dry-run should not create directories")
	}

	// Verify real run made changes
	if _, err := os.Stat(filepath.Join(tempDir2, ".mobilecombackup.yaml")); os.IsNotExist(err) {
		t.Error("Real run should create marker file")
	}

	if _, err := os.Stat(filepath.Join(tempDir2, "calls")); os.IsNotExist(err) {
		t.Error("Real run should create calls directory")
	}
}

// TestAutofixer_Integration_LargeRepository tests performance with larger datasets
func TestAutofixer_Integration_LargeRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large repository test in short mode")
	}

	tempDir := t.TempDir()
	autofixer := NewAutofixer(tempDir, &NullProgressReporter{})

	// Create a large number of violations to test performance
	violations := make([]validation.Violation, 0)

	// Add structure violations
	for _, dir := range []string{"calls", "sms", "attachments"} {
		violations = append(violations, validation.Violation{
			Type:     validation.StructureViolation,
			File:     dir + "/",
			Message:  "Missing directory: " + dir,
			Severity: validation.SeverityError,
		})
	}

	// Add many file violations
	fileTypes := []string{".mobilecombackup.yaml", "contacts.yaml", "summary.yaml", "files.yaml", "files.yaml.sha256"}
	for _, file := range fileTypes {
		violations = append(violations, validation.Violation{
			Type:     validation.MissingFile,
			File:     file,
			Message:  "Missing file: " + file,
			Severity: validation.SeverityError,
		})
	}

	// Create test files to scan for files.yaml generation
	createTestFilesForManifest(t, tempDir, 100)

	options := Options{
		DryRun:  false,
		Verbose: false,
	}

	report, err := autofixer.FixViolations(violations, options)
	if err != nil {
		t.Fatalf("FixViolations failed: %v", err)
	}

	// Verify all violations were processed
	totalViolations := len(violations)
	totalProcessed := report.Summary.FixedCount + report.Summary.SkippedCount + report.Summary.ErrorCount

	if totalProcessed != totalViolations {
		t.Errorf("Expected to process %d violations, processed %d", totalViolations, totalProcessed)
	}

	// Verify files.yaml was created and contains all test files
	filesPath := filepath.Join(tempDir, "files.yaml")
	if content, err := os.ReadFile(filesPath); err == nil { // nolint:gosec // Test-controlled path
		// Should contain references to the test files
		if !strings.Contains(string(content), "test-file-") {
			t.Error("Expected files.yaml to contain test files")
		}
	} else {
		t.Errorf("Failed to read files.yaml: %v", err)
	}
}

// TestAutofixer_Integration_PermissionErrors tests handling of permission errors
func TestAutofixer_Integration_PermissionErrors(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tempDir := t.TempDir()

	// Create a read-only directory
	readOnlyDir := filepath.Join(tempDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0500); err != nil {
		t.Fatalf("Failed to create read-only directory: %v", err)
	}
	defer func() {
		// Restore permissions for cleanup
		_ = os.Chmod(readOnlyDir, 0750) // nolint:gosec // Cleanup permissions
	}()

	autofixer := NewAutofixer(readOnlyDir, &NullProgressReporter{})

	violations := []validation.Violation{
		{
			Type:     validation.MissingFile,
			File:     ".mobilecombackup.yaml",
			Message:  "Missing marker file",
			Severity: validation.SeverityError,
		},
	}

	options := Options{
		DryRun:  false,
		Verbose: false,
	}

	report, err := autofixer.FixViolations(violations, options)
	if err != nil {
		t.Fatalf("FixViolations failed: %v", err)
	}

	// Should have errors due to permission issues
	if report.Summary.ErrorCount == 0 {
		t.Error("Expected permission errors")
	}

	// Verify error details include permission information
	foundPermissionError := false
	for _, autofixErr := range report.Errors {
		if strings.Contains(strings.ToLower(autofixErr.Error), "permission") {
			foundPermissionError = true
			break
		}
	}

	if !foundPermissionError {
		t.Error("Expected permission-related error message")
	}
}

// TestAutofixer_Integration_ErrorRecovery tests that errors don't stop processing
func TestAutofixer_Integration_ErrorRecovery(t *testing.T) {
	tempDir := t.TempDir()
	autofixer := NewAutofixer(tempDir, &NullProgressReporter{})

	// Create a mix of valid and problematic violations
	violations := []validation.Violation{
		{
			Type:     validation.StructureViolation,
			File:     "calls/",
			Message:  "Missing directory",
			Severity: validation.SeverityError,
		},
		{
			Type:     validation.CountMismatch,
			File:     "nonexistent/file.xml", // This will cause an error
			Message:  "Count mismatch",
			Severity: validation.SeverityError,
		},
		{
			Type:     validation.MissingFile,
			File:     ".mobilecombackup.yaml",
			Message:  "Missing marker file",
			Severity: validation.SeverityError,
		},
	}

	options := Options{
		DryRun:  false,
		Verbose: false,
	}

	report, err := autofixer.FixViolations(violations, options)
	if err != nil {
		t.Fatalf("FixViolations failed: %v", err)
	}

	// Should have some fixes and some errors
	if report.Summary.FixedCount == 0 {
		t.Error("Expected some violations to be fixed despite errors")
	}

	if report.Summary.ErrorCount == 0 {
		t.Error("Expected some errors due to problematic violations")
	}

	// Verify successful fixes were applied
	if _, err := os.Stat(filepath.Join(tempDir, "calls")); os.IsNotExist(err) {
		t.Error("Expected calls directory to be created despite other errors")
	}

	if _, err := os.Stat(filepath.Join(tempDir, ".mobilecombackup.yaml")); os.IsNotExist(err) {
		t.Error("Expected marker file to be created despite other errors")
	}
}

// setupRepositoryWithViolations creates a test repository with known violations
func setupRepositoryWithViolations(t *testing.T, repoDir string) {
	// Create SMS directory and file with count mismatch
	smsDir := filepath.Join(repoDir, "sms")
	if err := os.MkdirAll(smsDir, 0750); err != nil {
		t.Fatalf("Failed to create SMS directory: %v", err)
	}

	// Create SMS file with incorrect count
	smsContent := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="10">
  <sms protocol="0" address="12345" date="1373929642000" type="1" />
  <sms protocol="0" address="67890" date="1373929975000" type="2" />
</smses>`

	smsFile := filepath.Join(smsDir, "sms-2024.xml")
	if err := os.WriteFile(smsFile, []byte(smsContent), 0600); err != nil {
		t.Fatalf("Failed to write SMS file: %v", err)
	}

	// Note: calls/ directory is intentionally missing to test directory creation
	// Note: .mobilecombackup.yaml is intentionally missing to test file creation
}

// createTestFilesForManifest creates test files for files.yaml generation testing
func createTestFilesForManifest(t *testing.T, repoDir string, count int) {
	// Create test files in various directories
	for i := 0; i < count; i++ {
		var subdir string
		switch i % 3 {
		case 0:
			subdir = "calls"
		case 1:
			subdir = "sms"
		case 2:
			subdir = "attachments"
		}

		dir := filepath.Join(repoDir, subdir)
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		fileName := filepath.Join(dir, fmt.Sprintf("test-file-%d.txt", i))
		content := fmt.Sprintf("Test file content %d", i)

		if err := os.WriteFile(fileName, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to write test file %s: %v", fileName, err)
		}
	}
}

// TestAutofixer_Integration_Idempotent tests that running autofix multiple times is safe
func TestAutofixer_Integration_Idempotent(t *testing.T) {
	tempDir := t.TempDir()
	autofixer := NewAutofixer(tempDir, &NullProgressReporter{})

	violations := []validation.Violation{
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
	}

	options := Options{
		DryRun:  false,
		Verbose: false,
	}

	// Run autofix first time
	report1, err := autofixer.FixViolations(violations, options)
	if err != nil {
		t.Fatalf("First FixViolations failed: %v", err)
	}

	// Run autofix second time with same violations
	report2, err := autofixer.FixViolations(violations, options)
	if err != nil {
		t.Fatalf("Second FixViolations failed: %v", err)
	}

	// First run should fix violations
	if report1.Summary.FixedCount != 2 {
		t.Errorf("Expected 2 fixes in first run, got %d", report1.Summary.FixedCount)
	}

	// Second run should have no fixes (violations already resolved)
	// This depends on implementation - it might try to "fix" again but should be safe
	if report2.Summary.ErrorCount > 0 {
		t.Errorf("Second run should not generate errors: %d errors", report2.Summary.ErrorCount)
	}

	// Verify files still exist and are valid
	markerPath := filepath.Join(tempDir, ".mobilecombackup.yaml")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Error("Marker file should still exist after second run")
	}

	callsPath := filepath.Join(tempDir, "calls")
	if _, err := os.Stat(callsPath); os.IsNotExist(err) {
		t.Error("Calls directory should still exist after second run")
	}
}
