package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillipgreenii/mobilecombackup/pkg/attachments"
	"github.com/phillipgreenii/mobilecombackup/pkg/sms"
	"github.com/phillipgreenii/mobilecombackup/pkg/validation"

	"github.com/spf13/afero"
)

const (
	// Test hash constants
	testHash1 = "a123456789abcdef123456789abcdef123456789abcdef123456789abcdef123"
	testHash2 = "b223456789abcdef123456789abcdef123456789abcdef123456789abcdef123"
)

func TestOrphanRemovalIntegration(t *testing.T) {
	tests := []struct {
		name           string
		setupRepo      func(t *testing.T, repoPath string) (map[string]bool, []string) // returns (referencedHashes, orphanPaths)
		dryRun         bool
		expectOrphans  int
		expectRemoved  int
		expectFailures int
	}{
		{
			name: "no orphans found",
			setupRepo: func(t *testing.T, repoPath string) (map[string]bool, []string) {
				// Create some attachments
				hash1 := testHash1
				hash2 := testHash2

				createTestAttachment(t, repoPath, hash1, "test data 1")
				createTestAttachment(t, repoPath, hash2, "test data 2")

				// All attachments are referenced
				referenced := map[string]bool{
					hash1: true,
					hash2: true,
				}

				return referenced, []string{}
			},
			expectOrphans:  0,
			expectRemoved:  0,
			expectFailures: 0,
		},
		{
			name: "some orphans found and removed",
			setupRepo: func(t *testing.T, repoPath string) (map[string]bool, []string) {
				// Create some attachments
				hash1 := testHash1
				hash2 := testHash2
				hash3 := "c323456789abcdef123456789abcdef123456789abcdef123456789abcdef123"

				createTestAttachment(t, repoPath, hash1, "test data 1")
				createTestAttachment(t, repoPath, hash2, "test data 2")
				createTestAttachment(t, repoPath, hash3, "test data 3")

				// Only hash1 is referenced, hash2 and hash3 are orphans
				referenced := map[string]bool{
					hash1: true,
				}

				return referenced, []string{
					fmt.Sprintf("attachments/b2/%s", hash2),
					fmt.Sprintf("attachments/c3/%s", hash3),
				}
			},
			expectOrphans:  2,
			expectRemoved:  2,
			expectFailures: 0,
		},
		{
			name:   "dry run mode",
			dryRun: true,
			setupRepo: func(t *testing.T, repoPath string) (map[string]bool, []string) {
				// Create some attachments
				hash1 := testHash1
				hash2 := testHash2

				createTestAttachment(t, repoPath, hash1, "test data 1")
				createTestAttachment(t, repoPath, hash2, "test data 2")

				// Only hash1 is referenced, hash2 is orphan
				referenced := map[string]bool{
					hash1: true,
				}

				return referenced, []string{
					fmt.Sprintf("attachments/b2/%s", hash2),
				}
			},
			expectOrphans:  1,
			expectRemoved:  1, // Would be removed in dry run
			expectFailures: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary repository
			tempDir := t.TempDir()

			// Setup repository with attachments
			referencedHashes, expectedOrphanPaths := tt.setupRepo(t, tempDir)

			// Create mock SMS reader
			mockSMSReader := &mockSMSReader{
				referencedHashes: referencedHashes,
			}

			// Create attachment manager
			attachmentManager := attachments.NewAttachmentManager(tempDir, afero.NewOsFs())

			// Create mock progress reporter
			reporter := &NullProgressReporter{}

			// Set validateDryRun global variable
			originalDryRun := validateDryRun
			validateDryRun = tt.dryRun
			defer func() { validateDryRun = originalDryRun }()

			// Call orphan removal function
			result, err := removeOrphanAttachmentsWithProgress(mockSMSReader, attachmentManager, reporter)

			// Check no error occurred
			if err != nil {
				t.Fatalf("removeOrphanAttachmentsWithProgress returned error: %v", err)
			}

			// Verify results
			if result.OrphansFound != tt.expectOrphans {
				t.Errorf("Expected %d orphans found, got %d", tt.expectOrphans, result.OrphansFound)
			}

			if result.OrphansRemoved != tt.expectRemoved {
				t.Errorf("Expected %d orphans removed, got %d", tt.expectRemoved, result.OrphansRemoved)
			}

			if result.RemovalFailures != tt.expectFailures {
				t.Errorf("Expected %d removal failures, got %d", tt.expectFailures, result.RemovalFailures)
			}

			// In non-dry-run mode, verify files were actually removed
			if !tt.dryRun && tt.expectRemoved > 0 {
				for _, orphanPath := range expectedOrphanPaths {
					fullPath := filepath.Join(tempDir, orphanPath)
					if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
						t.Errorf("Expected orphan file %s to be removed, but it still exists", orphanPath)
					}
				}
			}

			// In dry-run mode, verify files were NOT removed
			if tt.dryRun && tt.expectOrphans > 0 {
				for _, orphanPath := range expectedOrphanPaths {
					fullPath := filepath.Join(tempDir, orphanPath)
					if _, err := os.Stat(fullPath); os.IsNotExist(err) {
						t.Errorf("Expected orphan file %s to remain in dry-run mode, but it was removed", orphanPath)
					}
				}
			}
		})
	}
}

func TestOrphanRemovalOutputFormats(t *testing.T) {
	tests := []struct {
		name     string
		dryRun   bool
		quiet    bool
		json     bool
		expected []string // Expected strings in output
	}{
		{
			name:   "normal text output",
			dryRun: false,
			quiet:  false,
			json:   false,
			expected: []string{
				"Orphan attachment removal:",
				"Attachments scanned:",
				"Orphans found:",
				"Orphans removed:",
				"MB freed",
			},
		},
		{
			name:   "dry run text output",
			dryRun: true,
			quiet:  false,
			json:   false,
			expected: []string{
				"Orphan attachment removal:",
				"Attachments scanned:",
				"Orphans found:",
				"Would remove:",
			},
		},
		{
			name:   "JSON output",
			dryRun: false,
			quiet:  false,
			json:   true,
			expected: []string{
				"\"orphan_removal\"",
				"\"attachments_scanned\"",
				"\"orphans_found\"",
				"\"orphans_removed\"",
				"\"bytes_freed\"",
			},
		},
		{
			name:     "quiet mode with orphan removal",
			dryRun:   false,
			quiet:    true,
			json:     false,
			expected: []string{"Orphans removed:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create result with orphan removal data
			result := ValidationResult{
				Valid:      true,
				Violations: []validation.Violation{},
				OrphanRemoval: &OrphanRemovalResult{
					AttachmentsScanned: 10,
					OrphansFound:       3,
					OrphansRemoved:     3,
					BytesFreed:         1024 * 1024, // 1 MB
					RemovalFailures:    0,
					FailedRemovals:     []FailedRemoval{},
				},
			}

			// Set global flags
			originalQuiet := quiet
			originalDryRun := validateDryRun
			originalJSON := outputJSON

			quiet = tt.quiet
			validateDryRun = tt.dryRun
			outputJSON = tt.json

			defer func() {
				quiet = originalQuiet
				validateDryRun = originalDryRun
				outputJSON = originalJSON
			}()

			// Capture output
			var buf bytes.Buffer
			originalStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Generate output
			if tt.json {
				err := outputJSONResult(result)
				if err != nil {
					t.Fatalf("outputJSONResult failed: %v", err)
				}
			} else {
				outputTextResult(result, "/test/repo")
			}

			// Restore stdout and capture output
			_ = w.Close()
			os.Stdout = originalStdout

			output, _ := io.ReadAll(r)
			buf.Write(output)
			outputStr := buf.String()

			// Verify expected strings are present
			for _, expected := range tt.expected {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected output to contain %q, but it didn't. Output: %s", expected, outputStr)
				}
			}

			// For JSON output, verify it's valid JSON
			if tt.json {
				var jsonResult ValidationResult
				if err := json.Unmarshal(output, &jsonResult); err != nil {
					t.Errorf("Output is not valid JSON: %v", err)
				}
			}
		})
	}
}

func TestCleanupEmptyDirectory(t *testing.T) {
	tests := []struct {
		name          string
		setupDir      func(t *testing.T, dirPath string)
		expectRemoved bool
	}{
		{
			name: "empty directory is removed",
			setupDir: func(t *testing.T, dirPath string) {
				err := os.MkdirAll(dirPath, 0750)
				if err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}
			},
			expectRemoved: true,
		},
		{
			name: "non-empty directory is not removed",
			setupDir: func(t *testing.T, dirPath string) {
				err := os.MkdirAll(dirPath, 0750)
				if err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}

				// Add a file to make it non-empty
				filePath := filepath.Join(dirPath, "test.txt")
				err = os.WriteFile(filePath, []byte("test"), 0600)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			},
			expectRemoved: false,
		},
		{
			name: "non-existent directory",
			setupDir: func(_ *testing.T, _ string) {
				// Don't create the directory
			},
			expectRemoved: false, // Can't remove what doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			testDir := filepath.Join(tempDir, "test-cleanup")

			// Setup test directory
			tt.setupDir(t, testDir)

			// Check if directory exists before cleanup
			_, existsBefore := os.Stat(testDir)

			// Call cleanup function
			cleanupEmptyDirectory(testDir)

			// Check if directory exists after cleanup
			_, existsAfter := os.Stat(testDir)

			if tt.expectRemoved {
				// Directory should have been removed
				if existsBefore == nil && existsAfter == nil {
					t.Error("Expected directory to be removed, but it still exists")
				}
			} else {
				// Directory should still exist (or never existed)
				if existsBefore == nil && existsAfter != nil {
					t.Error("Expected directory to remain, but it was removed")
				}
			}
		})
	}
}

// Helper functions and mocks

type mockSMSReader struct {
	referencedHashes map[string]bool
	shouldError      bool
}

func (m *mockSMSReader) ReadMessages(_ int) ([]sms.Message, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSMSReader) StreamMessagesForYear(_ int, _ func(sms.Message) error) error {
	return fmt.Errorf("not implemented")
}

func (m *mockSMSReader) GetAttachmentRefs(_ int) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSMSReader) GetAllAttachmentRefs() (map[string]bool, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}
	return m.referencedHashes, nil
}

func (m *mockSMSReader) GetAvailableYears() ([]int, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSMSReader) GetMessageCount(_ int) (int, error) {
	return 0, fmt.Errorf("not implemented")
}

func (m *mockSMSReader) ValidateSMSFile(_ int) error {
	return fmt.Errorf("not implemented")
}

// Context-aware method implementations for mock

func (m *mockSMSReader) ReadMessagesContext(ctx context.Context, year int) ([]sms.Message, error) {
	return m.ReadMessages(year)
}

func (m *mockSMSReader) StreamMessagesForYearContext(ctx context.Context, year int, callback func(sms.Message) error) error {
	return m.StreamMessagesForYear(year, callback)
}

func (m *mockSMSReader) GetAttachmentRefsContext(ctx context.Context, year int) ([]string, error) {
	return m.GetAttachmentRefs(year)
}

func (m *mockSMSReader) GetAllAttachmentRefsContext(ctx context.Context) (map[string]bool, error) {
	return m.GetAllAttachmentRefs()
}

func (m *mockSMSReader) GetAvailableYearsContext(ctx context.Context) ([]int, error) {
	return m.GetAvailableYears()
}

func (m *mockSMSReader) GetMessageCountContext(ctx context.Context, year int) (int, error) {
	return m.GetMessageCount(year)
}

func (m *mockSMSReader) ValidateSMSFileContext(ctx context.Context, year int) error {
	return m.ValidateSMSFile(year)
}

func createTestAttachment(t *testing.T, repoPath, hash, content string) {
	if len(hash) != 64 {
		t.Fatalf("Invalid hash length: %d", len(hash))
	}

	// Create directory structure
	prefix := hash[:2]
	attachmentDir := filepath.Join(repoPath, "attachments", prefix)
	err := os.MkdirAll(attachmentDir, 0750)
	if err != nil {
		t.Fatalf("Failed to create attachment directory: %v", err)
	}

	// Create attachment file
	attachmentPath := filepath.Join(attachmentDir, hash)
	err = os.WriteFile(attachmentPath, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create attachment file: %v", err)
	}
}
