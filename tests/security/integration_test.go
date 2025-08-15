package security_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillipgreen/mobilecombackup/pkg/attachments"
	"github.com/phillipgreen/mobilecombackup/pkg/autofix"
	"github.com/phillipgreen/mobilecombackup/pkg/security"
	"github.com/phillipgreen/mobilecombackup/pkg/validation"
)

// TestAttachmentStorage_SecurityIntegration tests that attachment storage is protected against directory traversal
func TestAttachmentStorage_SecurityIntegration(t *testing.T) {
	tempDir := t.TempDir()
	storage := attachments.NewDirectoryAttachmentStorage(tempDir)

	testData := []byte("test attachment data")
	metadata := attachments.AttachmentInfo{
		OriginalName: "test.txt",
		MimeType:     "text/plain",
		Size:         int64(len(testData)),
		Hash:         "test-hash",
	}

	// Test that directory traversal attack is blocked
	t.Run("directory_traversal_blocked", func(t *testing.T) {
		err := storage.Store("../../../etc/passwd", testData, metadata)
		if err == nil {
			t.Error("Expected directory traversal attack to be blocked")
		}
		if !strings.Contains(err.Error(), "invalid") {
			t.Errorf("Expected path validation error, got: %v", err)
		}
	})

	// Test that null byte injection is blocked
	t.Run("null_byte_blocked", func(t *testing.T) {
		err := storage.Store("test\x00../../../etc/passwd", testData, metadata)
		if err == nil {
			t.Error("Expected null byte injection attack to be blocked")
		}
		if !strings.Contains(err.Error(), "invalid") {
			t.Errorf("Expected path validation error, got: %v", err)
		}
	})

	// Test that legitimate operations still work
	t.Run("legitimate_operation_works", func(t *testing.T) {
		validHash := "abc123def456789012345678901234567890123456789012345678901234abcd"
		err := storage.Store(validHash, testData, metadata)
		if err != nil {
			t.Errorf("Expected legitimate operation to succeed, got error: %v", err)
		}

		// Verify the file was created in the expected location
		expectedPath := filepath.Join(tempDir, "attachments", "ab", validHash, "test.txt")
		if _, err := os.Stat(expectedPath); err != nil {
			t.Errorf("Expected attachment file to be created at %s, but got error: %v", expectedPath, err)
		}
	})
}

// TestAutofix_SecurityIntegration tests that autofix operations are protected against directory traversal
func TestAutofix_SecurityIntegration(t *testing.T) {
	tempDir := t.TempDir()
	reporter := &autofix.NullProgressReporter{}
	fixer := autofix.NewAutofixer(tempDir, reporter)

	// Create attack vectors via validation violations
	attackViolations := []validation.ValidationViolation{
		{
			Type:     validation.StructureViolation,
			Severity: validation.SeverityError,
			File:     "../../../tmp/malicious",
			Message:  "Missing directory with traversal attack",
		},
		{
			Type:     validation.MissingFile,
			Severity: validation.SeverityError,
			File:     "/etc/malicious.yaml",
			Message:  "Missing file with absolute path attack",
		},
		{
			Type:     validation.CountMismatch,
			Severity: validation.SeverityError,
			File:     "test\x00../../../etc/passwd",
			Message:  "Count mismatch with null byte injection",
		},
	}

	// Attempt to fix violations with malicious paths
	for _, violation := range attackViolations {
		t.Run(violation.File, func(t *testing.T) {
			report, err := fixer.FixViolations([]validation.ValidationViolation{violation}, autofix.AutofixOptions{})

			// Should either fail with validation error or successfully skip the violation
			if err == nil {
				// If no error, check that no fixes were actually applied
				if report.Summary.FixedCount > 0 {
					t.Errorf("Expected no fixes for malicious path, but %d fixes were applied", report.Summary.FixedCount)
				}
			} else {
				// If error, should be path validation error
				if !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "path") {
					t.Errorf("Expected path validation error, got: %v", err)
				}
			}
		})
	}
}

// TestPathValidator_RealisticAttackScenarios tests path validation with realistic attack scenarios
func TestPathValidator_RealisticAttackScenarios(t *testing.T) {
	tempDir := t.TempDir()
	validator := security.NewPathValidator(tempDir)

	// Create legitimate files and directories for the test
	legitDir := filepath.Join(tempDir, "attachments", "ab")
	err := os.MkdirAll(legitDir, 0750)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Realistic attack scenarios
	scenarios := []struct {
		name    string
		path    string
		wantErr bool
		desc    string
	}{
		{
			name:    "legitimate_attachment_path",
			path:    "attachments/ab/abc123.txt",
			wantErr: false,
			desc:    "Normal attachment path should work",
		},
		{
			name:    "config_file_traversal",
			path:    "../../.env",
			wantErr: true,
			desc:    "Attempt to access parent directory config files",
		},
		{
			name:    "system_file_traversal",
			path:    "../../../etc/shadow",
			wantErr: true,
			desc:    "Attempt to access system password file",
		},
		{
			name:    "home_directory_traversal",
			path:    "../../../home/user/.ssh/id_rsa",
			wantErr: true,
			desc:    "Attempt to access SSH private keys",
		},
		{
			name:    "windows_system_traversal",
			path:    "..\\..\\..\\Windows\\System32\\config\\SAM",
			wantErr: true,
			desc:    "Windows-style path traversal",
		},
		{
			name:    "encoded_traversal",
			path:    "%2e%2e%2f%2e%2e%2fetc%2fpasswd",
			wantErr: true,
			desc:    "URL-encoded directory traversal",
		},
		{
			name:    "unicode_traversal",
			path:    "..\u002f..\u002fetc\u002fpasswd",
			wantErr: true,
			desc:    "Unicode-encoded directory traversal",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			result, err := validator.ValidatePath(scenario.path)

			if scenario.wantErr {
				if err == nil {
					t.Errorf("%s: expected error for %s, but validation succeeded with result: %q", scenario.desc, scenario.path, result)
				}
			} else {
				if err != nil {
					t.Errorf("%s: expected success for %s, but got error: %v", scenario.desc, scenario.path, err)
				}
			}
		})
	}
}

// TestPathValidator_Performance tests that path validation meets performance requirements
func TestPathValidator_Performance(t *testing.T) {
	tempDir := t.TempDir()
	validator := security.NewPathValidator(tempDir)

	// Test path validation performance with 10,000 validations
	testPaths := []string{
		"attachments/ab/valid-file.txt",
		"calls/calls-2024.xml",
		"sms/sms-2024.xml",
		"contacts.yaml",
		"summary.yaml",
	}

	iterations := 10000

	// Warm up
	for i := 0; i < 100; i++ {
		path := testPaths[i%len(testPaths)]
		_, _ = validator.ValidatePath(path) // Ignore errors in warmup
	}

	// Performance test
	for i := 0; i < iterations; i++ {
		path := testPaths[i%len(testPaths)]
		_, err := validator.ValidatePath(path)
		if err != nil {
			t.Fatalf("Unexpected validation error on iteration %d: %v", i, err)
		}
	}

	// Test should complete in reasonable time (handled by test timeout)
	// Individual validation should be under 1ms (tested by benchmark)
}
