package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewPathValidator(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewPathValidator(tempDir)

	// Should store absolute path
	if !filepath.IsAbs(validator.BaseDir) {
		t.Errorf("Expected absolute base directory, got %q", validator.BaseDir)
	}

	// Should clean the path
	validator2 := NewPathValidator(tempDir + "/./subdir/../")
	expectedClean := filepath.Join(tempDir, "subdir", "..")
	expectedAbs, _ := filepath.Abs(expectedClean)
	if validator2.BaseDir != expectedAbs {
		t.Errorf("Expected cleaned base directory %q, got %q", expectedAbs, validator2.BaseDir)
	}
}

func TestPathValidator_ValidatePath_ValidPaths(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewPathValidator(tempDir)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple relative path", "file.txt", "file.txt"},
		{"nested relative path", "dir/file.txt", filepath.Join("dir", "file.txt")},
		{"current directory", ".", "."},
		{"nested with current dir", "./dir/file.txt", filepath.Join("dir", "file.txt")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidatePath(tt.input)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestPathValidator_ValidatePath_AttackVectors(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewPathValidator(tempDir)

	tests := []struct {
		name          string
		input         string
		expectedError error
		skipOnUnix    bool
	}{
		{"directory traversal simple", "../etc/passwd", ErrPathOutsideRepository, false},
		{"directory traversal nested", "dir/../../etc/passwd", ErrPathOutsideRepository, false},
		{"directory traversal multiple", "../../../etc/passwd", ErrPathOutsideRepository, false},
		{"absolute path outside", "/etc/passwd", ErrPathOutsideRepository, false},
		{"windows absolute path", "C:\\Windows\\System32", ErrPathOutsideRepository, true}, // Skip on Unix
		{"null byte injection", "file.txt\x00malicious", ErrInvalidPath, false},
		{"empty path", "", ErrInvalidPath, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnUnix && filepath.Separator == '/' {
				t.Skip("Skipping Windows path test on Unix system")
			}

			_, err := validator.ValidatePath(tt.input)
			if err == nil {
				t.Fatalf("Expected error for input %q, got none", tt.input)
			}

			if !strings.Contains(err.Error(), tt.expectedError.Error()) {
				t.Errorf("Expected error containing %q, got %v", tt.expectedError, err)
			}
		})
	}
}

func TestPathValidator_ValidatePath_PathLength(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewPathValidator(tempDir)

	// Create a path that's too long
	longPath := strings.Repeat("a", MaxPathLength+1)

	_, err := validator.ValidatePath(longPath)
	if err == nil {
		t.Fatal("Expected error for overly long path")
	}

	if !strings.Contains(err.Error(), ErrPathTooLong.Error()) {
		t.Errorf("Expected path too long error, got %v", err)
	}

	// Path at exactly the limit should work
	limitPath := strings.Repeat("a", MaxPathLength)
	_, err = validator.ValidatePath(limitPath)
	if err != nil {
		t.Errorf("Path at limit should be valid, got error: %v", err)
	}
}

func TestPathValidator_ValidatePath_SymlinkAttack(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewPathValidator(tempDir)

	// Create a directory outside the base
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "secret.txt")
	err := os.WriteFile(outsideFile, []byte("secret"), 0600)
	if err != nil {
		t.Skipf("Cannot create test file: %v", err)
	}

	// Create a symlink inside the base directory pointing outside
	symlinkPath := filepath.Join(tempDir, "evil_link")
	err = os.Symlink(outsideFile, symlinkPath)
	if err != nil {
		t.Skipf("Cannot create symlink (may not be supported): %v", err)
	}

	// Try to access the symlink - should be rejected
	_, err = validator.ValidatePath("evil_link")
	if err == nil {
		t.Fatal("Expected error for symlink pointing outside base directory")
	}

	if !strings.Contains(err.Error(), ErrPathOutsideRepository.Error()) {
		t.Errorf("Expected path outside repository error, got %v", err)
	}
}

func TestPathValidator_JoinAndValidate(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewPathValidator(tempDir)

	tests := []struct {
		name        string
		elements    []string
		expectError bool
	}{
		{"valid join", []string{"dir", "subdir", "file.txt"}, false},
		{"join with traversal", []string{"dir", "..", "..", "etc", "passwd"}, true},
		{"empty elements", []string{}, true},
		{"single element", []string{"file.txt"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.JoinAndValidate(tt.elements...)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for elements %v, got none", tt.elements)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for elements %v, got %v", tt.elements, err)
				}

				// Result should be equivalent to filepath.Join of elements (when valid)
				if len(tt.elements) > 0 {
					expected := filepath.Join(tt.elements...)
					if result != expected {
						t.Errorf("Expected %q, got %q", expected, result)
					}
				}
			}
		})
	}
}

func TestPathValidator_ValidateAbsolutePath(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewPathValidator(tempDir)

	// Valid absolute path within base
	validAbs := filepath.Join(tempDir, "file.txt")
	result, err := validator.ValidateAbsolutePath(validAbs)
	if err != nil {
		t.Errorf("Expected no error for valid absolute path, got %v", err)
	}
	if result != "file.txt" {
		t.Errorf("Expected relative path 'file.txt', got %q", result)
	}

	// Relative path should be rejected
	_, err = validator.ValidateAbsolutePath("relative/path.txt")
	if err == nil {
		t.Error("Expected error for relative path in ValidateAbsolutePath")
	}

	// Absolute path outside base should be rejected
	_, err = validator.ValidateAbsolutePath("/etc/passwd")
	if err == nil {
		t.Error("Expected error for absolute path outside base")
	}
}

func TestPathValidator_GetSafePath(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewPathValidator(tempDir)

	// Valid relative path
	result, err := validator.GetSafePath("subdir/file.txt")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expected := filepath.Join(tempDir, "subdir", "file.txt")
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	// Invalid path with traversal
	_, err = validator.GetSafePath("../etc/passwd")
	if err == nil {
		t.Error("Expected error for path with traversal")
	}
}

// Benchmark tests for performance requirements
func BenchmarkPathValidator_ValidatePath(b *testing.B) {
	tempDir := b.TempDir()
	validator := NewPathValidator(tempDir)
	testPath := "subdir/file.txt"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.ValidatePath(testPath)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkPathValidator_ValidatePathWithTraversal(b *testing.B) {
	tempDir := b.TempDir()
	validator := NewPathValidator(tempDir)
	testPath := "../etc/passwd"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.ValidatePath(testPath)
		if err == nil {
			b.Fatal("Expected error for traversal path")
		}
	}
}

// Fuzzing test for path validation
func FuzzPathValidator_ValidatePath(f *testing.F) {
	// Seed corpus with interesting test cases
	f.Add("file.txt")
	f.Add("../etc/passwd")
	f.Add("dir/subdir/file.txt")
	f.Add("/absolute/path")
	f.Add("file\x00name")
	f.Add(strings.Repeat("a", 100))

	tempDir := f.TempDir()
	validator := NewPathValidator(tempDir)

	f.Fuzz(func(t *testing.T, input string) {
		// Should not panic regardless of input
		result, err := validator.ValidatePath(input)

		// If no error, result should be a valid relative path
		if err == nil {
			if filepath.IsAbs(result) {
				t.Errorf("ValidatePath returned absolute path: %q", result)
			}

			// Result should not contain .. elements after cleaning
			if strings.Contains(filepath.Clean(result), "..") {
				t.Errorf("ValidatePath returned path with .. elements: %q", result)
			}
		}
	})
}
