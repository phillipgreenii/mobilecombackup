package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMarkerFileValidator_ValidateMarkerFile(t *testing.T) {
	tests := []struct {
		name             string
		markerContent    string
		expectViolations bool
		violationTypes   []ViolationType
		versionSupported bool
		wantError        bool
	}{
		{
			name: "valid marker file",
			markerContent: `repository_structure_version: "1"
created_at: "2024-01-15T10:30:00Z"
created_by: "mobilecombackup v1.0.0"
`,
			expectViolations: false,
			versionSupported: true,
		},
		{
			name:             "missing marker file",
			markerContent:    "", // File won't be created
			expectViolations: true,
			violationTypes:   []ViolationType{MissingMarkerFile},
			versionSupported: true, // Missing file doesn't prevent version support
		},
		{
			name: "malformed YAML",
			markerContent: `repository_structure_version: "1
created_at: invalid yaml
`,
			expectViolations: true,
			violationTypes:   []ViolationType{InvalidFormat},
			versionSupported: false, // Can't determine version
		},
		{
			name: "missing repository_structure_version",
			markerContent: `created_at: "2024-01-15T10:30:00Z"
created_by: "mobilecombackup v1.0.0"
`,
			expectViolations: true,
			violationTypes:   []ViolationType{InvalidFormat},
			versionSupported: true, // Missing version field is supported
		},
		{
			name: "unsupported version",
			markerContent: `repository_structure_version: "2"
created_at: "2024-01-15T10:30:00Z"
created_by: "mobilecombackup v2.0.0"
`,
			expectViolations: true,
			violationTypes:   []ViolationType{UnsupportedVersion},
			versionSupported: false,
		},
		{
			name: "missing created_at",
			markerContent: `repository_structure_version: "1"
created_by: "mobilecombackup v1.0.0"
`,
			expectViolations: true,
			violationTypes:   []ViolationType{InvalidFormat},
			versionSupported: true,
		},
		{
			name: "invalid RFC3339 timestamp",
			markerContent: `repository_structure_version: "1"
created_at: "2024-01-15 10:30:00"
created_by: "mobilecombackup v1.0.0"
`,
			expectViolations: true,
			violationTypes:   []ViolationType{InvalidFormat},
			versionSupported: true,
		},
		{
			name: "missing created_by",
			markerContent: `repository_structure_version: "1"
created_at: "2024-01-15T10:30:00Z"
`,
			expectViolations: true,
			violationTypes:   []ViolationType{InvalidFormat},
			versionSupported: true,
		},
		{
			name: "extra fields logged as warning",
			markerContent: `repository_structure_version: "1"
created_at: "2024-01-15T10:30:00Z"
created_by: "mobilecombackup v1.0.0"
extra_field: "should log warning"
another_field: 123
`,
			expectViolations: false,
			versionSupported: true,
		},
		{
			name:             "empty YAML file",
			markerContent:    "---",
			expectViolations: true,
			violationTypes:   []ViolationType{InvalidFormat, InvalidFormat, InvalidFormat}, // All fields missing
			versionSupported: true,
		},
		{
			name: "null values",
			markerContent: `repository_structure_version: null
created_at: null
created_by: null
`,
			expectViolations: true,
			violationTypes:   []ViolationType{InvalidFormat, InvalidFormat, InvalidFormat},
			versionSupported: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tempDir := t.TempDir()

			// Create marker file if content provided
			if tt.markerContent != "" {
				markerPath := filepath.Join(tempDir, ".mobilecombackup.yaml")
				err := os.WriteFile(markerPath, []byte(tt.markerContent), 0600)
				if err != nil {
					t.Fatalf("Failed to create test marker file: %v", err)
				}
			}

			// Create validator
			validator := NewMarkerFileValidator(tempDir)

			// Validate
			violations, versionSupported, err := validator.ValidateMarkerFile()

			// Check error
			if tt.wantError && err == nil {
				t.Error("Expected error but got none")
			} else if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check violations
			if tt.expectViolations && len(violations) == 0 {
				t.Error("Expected violations but got none")
			} else if !tt.expectViolations && len(violations) > 0 {
				t.Errorf("Unexpected violations: %v", violations)
			}

			// Check violation types
			if tt.expectViolations {
				if len(violations) != len(tt.violationTypes) {
					t.Errorf("Expected %d violations, got %d", len(tt.violationTypes), len(violations))
				}

				for i, expectedType := range tt.violationTypes {
					if i >= len(violations) {
						break
					}
					if violations[i].Type != expectedType {
						t.Errorf("Violation %d: expected type %s, got %s", i, expectedType, violations[i].Type)
					}
				}
			}

			// Check version support
			if versionSupported != tt.versionSupported {
				t.Errorf("Version supported: expected %v, got %v", tt.versionSupported, versionSupported)
			}
		})
	}
}

func TestMarkerFileValidator_GetSuggestedFix(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewMarkerFileValidator(tempDir)

	suggestedFix := validator.GetSuggestedFix()

	// Check that suggested fix contains required fields
	if !strings.Contains(suggestedFix, `repository_structure_version: "1"`) {
		t.Error("Suggested fix missing repository_structure_version")
	}

	if !strings.Contains(suggestedFix, "created_at:") {
		t.Error("Suggested fix missing created_at")
	}

	if !strings.Contains(suggestedFix, "created_by:") {
		t.Error("Suggested fix missing created_by")
	}

	// Check that created_at is valid RFC3339
	lines := strings.Split(suggestedFix, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "created_at:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				// Remove quotes and whitespace
				timestamp := strings.TrimSpace(parts[1])
				timestamp = strings.Trim(timestamp, `"`)
				timestamp = strings.TrimSpace(timestamp)
				if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
					t.Errorf("Suggested fix has invalid RFC3339 timestamp: %q (error: %v)", timestamp, err)
				}
			}
		}
	}
}

func TestMarkerFileValidator_FixableViolation(t *testing.T) {
	// Create temp directory without marker file
	tempDir := t.TempDir()
	validator := NewMarkerFileValidator(tempDir)

	violations, _, err := validator.ValidateMarkerFile()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have one violation for missing file
	if len(violations) != 1 {
		t.Fatalf("Expected 1 violation, got %d", len(violations))
	}

	violation := violations[0]
	if violation.Type != MissingMarkerFile {
		t.Errorf("Expected MissingMarkerFile, got %s", violation.Type)
	}

	// Create fixable violation
	fixable := FixableViolation{
		ValidationViolation: violation,
		SuggestedFix:        validator.GetSuggestedFix(),
	}

	// Verify fixable violation has suggested fix
	if fixable.SuggestedFix == "" {
		t.Error("Fixable violation missing suggested fix")
	}
}

func TestMarkerFileValidator_ErrorHandling(t *testing.T) {
	// Test with non-existent directory
	// The validator will report missing marker file even if parent directory doesn't exist
	validator := NewMarkerFileValidator("/non/existent/path")

	violations, versionSupported, err := validator.ValidateMarkerFile()

	// Should not return error for non-existent directory (just missing file violation)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should have missing file violation
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation, got %d", len(violations))
	}

	if len(violations) > 0 && violations[0].Type != MissingMarkerFile {
		t.Errorf("Expected MissingMarkerFile violation, got %s", violations[0].Type)
	}

	// Version should be supported for missing file
	if !versionSupported {
		t.Error("Version should be supported for missing file")
	}
}
