// Package security provides security-related functionality including path validation to prevent directory traversal attacks.
package security

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var (
	// ErrPathOutsideRepository indicates a path would access files outside the repository
	ErrPathOutsideRepository = errors.New("path would access files outside repository")
	// ErrInvalidPath indicates a path contains invalid characters or sequences
	ErrInvalidPath = errors.New("path contains invalid characters or sequences")
	// ErrPathTooLong indicates a path exceeds maximum length
	ErrPathTooLong = errors.New("path exceeds maximum length")
)

const (
	// MaxPathLength defines the maximum allowed path length
	MaxPathLength = 1024
)

// PathValidator provides secure path validation to prevent directory traversal attacks
type PathValidator struct {
	BaseDir string
}

// NewPathValidator creates a new PathValidator with the specified base directory
func NewPathValidator(baseDir string) *PathValidator {
	// Clean and make absolute to ensure consistent behavior
	absBase, _ := filepath.Abs(filepath.Clean(baseDir))
	return &PathValidator{
		BaseDir: absBase,
	}
}

// ValidatePath validates a user-provided path and returns a safe path relative to the base directory
func (v *PathValidator) ValidatePath(userPath string) (string, error) {
	// 1. Check for empty path
	if userPath == "" {
		return "", fmt.Errorf("%w: empty path", ErrInvalidPath)
	}

	// 2. Check path length
	if len(userPath) > MaxPathLength {
		return "", fmt.Errorf("%w: path length %d exceeds maximum %d", ErrPathTooLong, len(userPath), MaxPathLength)
	}

	// 3. Check for null bytes (defense against null byte injection)
	if strings.Contains(userPath, "\x00") {
		return "", fmt.Errorf("%w: null byte in path", ErrInvalidPath)
	}

	// 4. Check for Windows-style path traversal (backslashes)
	if strings.Contains(userPath, "\\") {
		return "", fmt.Errorf("%w: backslash in path", ErrInvalidPath)
	}

	// 5. Check for URL-encoded directory traversal sequences
	urlEncodedPatterns := []string{
		"%2e%2e%2f", // ../
		"%2e%2e/",   // ../
		"..%2f",     // ../
		"%2e%2e\\",  // ..\
		"%2e%2e%5c", // ..\
	}
	lowerPath := strings.ToLower(userPath)
	for _, pattern := range urlEncodedPatterns {
		if strings.Contains(lowerPath, pattern) {
			return "", fmt.Errorf("%w: URL-encoded traversal sequence in path", ErrInvalidPath)
		}
	}

	// 6. Clean the path (removes . and .. elements)
	cleaned := filepath.Clean(userPath)

	// 7. Make absolute if relative
	var absPath string
	if filepath.IsAbs(cleaned) {
		absPath = cleaned
	} else {
		absPath = filepath.Join(v.BaseDir, cleaned)
	}

	// 8. Resolve symlinks to real path (prevents symlink attacks)
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If EvalSymlinks fails, the path might not exist, which is often OK
		// Fall back to Abs for non-existing paths
		realPath, err = filepath.Abs(absPath)
		if err != nil {
			return "", fmt.Errorf("%w: failed to resolve path: %v", ErrInvalidPath, err)
		}
	}

	// 9. Verify the resolved path is within base directory
	absBase, err := filepath.Abs(v.BaseDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base directory: %w", err)
	}

	// Ensure both paths have trailing separators for accurate prefix checking
	realWithSep := realPath + string(filepath.Separator)
	baseWithSep := absBase + string(filepath.Separator)

	if !strings.HasPrefix(realWithSep, baseWithSep) {
		return "", fmt.Errorf("%w: resolved path %q is outside base directory %q", ErrPathOutsideRepository, realPath, absBase)
	}

	// 10. Return path relative to base directory
	relPath, err := filepath.Rel(absBase, realPath)
	if err != nil {
		return "", fmt.Errorf("failed to make path relative: %w", err)
	}

	return relPath, nil
}

// JoinAndValidate safely joins path elements and validates the result
func (v *PathValidator) JoinAndValidate(elem ...string) (string, error) {
	if len(elem) == 0 {
		return "", fmt.Errorf("%w: no path elements provided", ErrInvalidPath)
	}

	// Join all elements
	joined := filepath.Join(elem...)

	// Validate the joined path
	return v.ValidatePath(joined)
}

// ValidateAbsolutePath validates that a path is absolute and within the base directory
func (v *PathValidator) ValidateAbsolutePath(absPath string) (string, error) {
	if !filepath.IsAbs(absPath) {
		return "", fmt.Errorf("%w: path must be absolute", ErrInvalidPath)
	}

	return v.ValidatePath(absPath)
}

// GetSafePath returns a validated absolute path for a given relative path
func (v *PathValidator) GetSafePath(relativePath string) (string, error) {
	validRelPath, err := v.ValidatePath(relativePath)
	if err != nil {
		return "", err
	}

	return filepath.Join(v.BaseDir, validRelPath), nil
}
