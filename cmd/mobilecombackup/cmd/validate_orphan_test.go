package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/phillipgreenii/mobilecombackup/pkg/attachments"
	"github.com/phillipgreenii/mobilecombackup/pkg/validation"
)

// TestOrphanRemovalOutputFormats tests the various output formats for orphan removal results
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
				OrphanRemoval: &attachments.OrphanRemovalResult{
					AttachmentsScanned: 10,
					OrphansFound:       3,
					OrphansRemoved:     3,
					BytesFreed:         1024 * 1024, // 1 MB
					RemovalFailures:    0,
					FailedRemovals:     []attachments.FailedRemoval{},
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
