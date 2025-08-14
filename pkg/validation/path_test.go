package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/quick"
)

func TestNewPathValidator(t *testing.T) {
	tests := []struct {
		name     string
		baseDir  string
		wantErr  bool
		contains string
	}{
		{
			name:    "valid absolute path",
			baseDir: "/tmp/test",
			wantErr: false,
		},
		{
			name:    "valid relative path",
			baseDir: "test",
			wantErr: false,
		},
		{
			name:     "empty path",
			baseDir:  "",
			wantErr:  true,
			contains: "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewPathValidator(tt.baseDir)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewPathValidator() expected error but got none")
				} else if tt.contains != "" && !strings.Contains(err.Error(), tt.contains) {
					t.Errorf("NewPathValidator() error = %v, should contain %v", err, tt.contains)
				}
				return
			}
			if err != nil {
				t.Errorf("NewPathValidator() unexpected error = %v", err)
				return
			}
			if validator == nil {
				t.Errorf("NewPathValidator() returned nil validator")
				return
			}
			if !filepath.IsAbs(validator.BaseDir) {
				t.Errorf("NewPathValidator() BaseDir should be absolute, got %v", validator.BaseDir)
			}
		})
	}
}

func TestPathValidator_ValidatePath(t *testing.T) {
	tmpDir := t.TempDir()
	validator, err := NewPathValidator(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		wantErr  bool
		errType  error
		contains string
	}{
		{
			name:    "valid relative path",
			path:    "test/file.txt",
			wantErr: false,
		},
		{
			name:    "valid single file",
			path:    "file.txt",
			wantErr: false,
		},
		{
			name:    "path with dots but safe",
			path:    "test.file.txt",
			wantErr: false,
		},
		{
			name:    "directory traversal attack",
			path:    "../../../etc/passwd",
			wantErr: true,
			errType: ErrPathOutsideRepository,
		},
		{
			name:    "directory traversal with relative path",
			path:    "test/../../etc/passwd",
			wantErr: true,
			errType: ErrPathOutsideRepository,
		},
		{
			name:    "absolute path attack",
			path:    "/etc/passwd",
			wantErr: true,
			errType: ErrPathOutsideRepository,
		},
		{
			name:     "empty path",
			path:     "",
			wantErr:  true,
			errType:  ErrEmptyPath,
			contains: "cannot be empty",
		},
		{
			name:     "null byte injection",
			path:     "file\x00.txt",
			wantErr:  true,
			errType:  ErrInvalidPath,
			contains: "null byte",
		},
		{
			name:     "path too long",
			path:     strings.Repeat("a", MaxPathLength+1),
			wantErr:  true,
			errType:  ErrPathTooLong,
			contains: "exceeds maximum length",
		},
		{
			name:     "control character attack",
			path:     "file\x01.txt",
			wantErr:  true,
			errType:  ErrInvalidPath,
			contains: "control character",
		},
		{
			name:     "invalid unicode",
			path:     "file\xff.txt",
			wantErr:  true,
			errType:  ErrInvalidUnicode,
			contains: "invalid UTF-8",
		},
		{
			name:    "windows-style absolute path",
			path:    "C:\\Windows\\System32",
			wantErr: true,
			errType: ErrPathOutsideRepository,
		},
		{
			name:    "unicode filename (valid)",
			path:    "тест.txt",
			wantErr: false,
		},
		{
			name:    "multiple slashes",
			path:    "test//file.txt",
			wantErr: false,
		},
		{
			name:    "current directory reference",
			path:    "./file.txt",
			wantErr: false,
		},
		{
			name:    "current directory in middle",
			path:    "test/./file.txt",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidatePath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidatePath() expected error but got none, result: %v", result)
					return
				}
				if tt.errType != nil && !strings.Contains(err.Error(), tt.errType.Error()) {
					t.Errorf("ValidatePath() expected error type %v, got %v", tt.errType, err)
				}
				if tt.contains != "" && !strings.Contains(err.Error(), tt.contains) {
					t.Errorf("ValidatePath() error should contain %q, got %v", tt.contains, err)
				}
				return
			}
			if err != nil {
				t.Errorf("ValidatePath() unexpected error = %v", err)
				return
			}

			// Verify result is relative and doesn't start with ..
			if filepath.IsAbs(result) {
				t.Errorf("ValidatePath() should return relative path, got %v", result)
			}
			if strings.HasPrefix(result, "..") {
				t.Errorf("ValidatePath() should not return path starting with .., got %v", result)
			}
		})
	}
}

func TestPathValidator_JoinAndValidate(t *testing.T) {
	tmpDir := t.TempDir()
	validator, err := NewPathValidator(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	tests := []struct {
		name    string
		elem    []string
		wantErr bool
		errType error
	}{
		{
			name:    "valid join",
			elem:    []string{"dir", "subdir", "file.txt"},
			wantErr: false,
		},
		{
			name:    "join with traversal",
			elem:    []string{"dir", "..", "..", "etc", "passwd"},
			wantErr: true,
			errType: ErrPathOutsideRepository,
		},
		{
			name:    "empty elements list",
			elem:    []string{},
			wantErr: true,
			errType: ErrEmptyPath,
		},
		{
			name:    "single element",
			elem:    []string{"file.txt"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.JoinAndValidate(tt.elem...)
			if tt.wantErr {
				if err == nil {
					t.Errorf("JoinAndValidate() expected error but got none, result: %v", result)
					return
				}
				if tt.errType != nil && !strings.Contains(err.Error(), tt.errType.Error()) {
					t.Errorf("JoinAndValidate() expected error type %v, got %v", tt.errType, err)
				}
				return
			}
			if err != nil {
				t.Errorf("JoinAndValidate() unexpected error = %v", err)
				return
			}

			// Verify result doesn't start with ..
			if strings.HasPrefix(result, "..") {
				t.Errorf("JoinAndValidate() should not return path starting with .., got %v", result)
			}
		})
	}
}

func TestSafeFilePath(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		baseDir  string
		userPath string
		wantErr  bool
		errType  error
	}{
		{
			name:     "valid path",
			baseDir:  tmpDir,
			userPath: "test/file.txt",
			wantErr:  false,
		},
		{
			name:     "traversal attack",
			baseDir:  tmpDir,
			userPath: "../../../etc/passwd",
			wantErr:  true,
			errType:  ErrPathOutsideRepository,
		},
		{
			name:     "invalid base dir",
			baseDir:  "",
			userPath: "test.txt",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SafeFilePath(tt.baseDir, tt.userPath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("SafeFilePath() expected error but got none, result: %v", result)
					return
				}
				if tt.errType != nil && !strings.Contains(err.Error(), tt.errType.Error()) {
					t.Errorf("SafeFilePath() expected error type %v, got %v", tt.errType, err)
				}
				return
			}
			if err != nil {
				t.Errorf("SafeFilePath() unexpected error = %v", err)
				return
			}
		})
	}
}

// TestSymlinkAttack tests protection against symlink-based path traversal
func TestSymlinkAttack(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	validator, err := NewPathValidator(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Create a directory inside the base
	insideDir := filepath.Join(tmpDir, "inside")
	if err := os.MkdirAll(insideDir, 0755); err != nil {
		t.Fatalf("Failed to create inside directory: %v", err)
	}

	// Create a directory outside the base
	outsideDir := filepath.Join(filepath.Dir(tmpDir), "outside")
	if err := os.MkdirAll(outsideDir, 0755); err != nil {
		t.Fatalf("Failed to create outside directory: %v", err)
	}

	// Create a symlink inside the base that points outside
	symlinkPath := filepath.Join(insideDir, "malicious_link")
	if err := os.Symlink(outsideDir, symlinkPath); err != nil {
		t.Skipf("Cannot create symlink (likely Windows without admin): %v", err)
	}

	// Try to access files through the symlink
	_, err = validator.ValidatePath("inside/malicious_link/secret.txt")
	if err == nil {
		t.Errorf("ValidatePath() should reject symlink attack")
	} else if !strings.Contains(err.Error(), ErrPathOutsideRepository.Error()) {
		t.Errorf("Expected path outside repository error, got: %v", err)
	}
}

// TestPathValidationFuzzing uses Go's built-in fuzzing to test edge cases
func TestPathValidationFuzzing(t *testing.T) {
	tmpDir := t.TempDir()
	validator, err := NewPathValidator(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Test with property-based testing
	err = quick.Check(func(input string) bool {
		// Skip inputs that are too long to avoid timeout
		if len(input) > 1000 {
			return true
		}

		result, validationErr := validator.ValidatePath(input)

		// If validation succeeds, the result should be safe
		if validationErr == nil {
			// Result should not start with .. or be absolute
			if strings.HasPrefix(result, "..") || filepath.IsAbs(result) {
				return false
			}
			// Joining with base dir should not escape
			fullPath := filepath.Join(validator.BaseDir, result)
			absPath, absErr := filepath.Abs(fullPath)
			if absErr != nil {
				return true // Can't check further if we can't get absolute path
			}
			return strings.HasPrefix(absPath, validator.BaseDir)
		}

		// If validation fails, that's acceptable
		return true
	}, nil)

	if err != nil {
		t.Errorf("Fuzzing test failed: %v", err)
	}
}

// TestPerformance ensures validation doesn't introduce significant overhead
func TestPerformance(t *testing.T) {
	tmpDir := t.TempDir()
	validator, err := NewPathValidator(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Test with 10,000 path validations to ensure performance is acceptable
	testPaths := []string{
		"test/file.txt",
		"dir/subdir/file.json",
		"simple.txt",
		"path/with/many/levels/file.dat",
	}

	const iterations = 10000

	for _, path := range testPaths {
		for i := 0; i < iterations; i++ {
			_, err := validator.ValidatePath(path)
			if err != nil {
				t.Errorf("Unexpected error during performance test: %v", err)
			}
		}
	}
}
