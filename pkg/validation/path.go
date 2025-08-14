// Package validation provides path validation functionality to prevent path traversal attacks.
package validation

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	customerrors "github.com/phillipgreen/mobilecombackup/pkg/errors"
)

// Maximum path length to prevent resource exhaustion attacks
const MaxPathLength = 4096

// PathValidator provides secure path validation to prevent directory traversal attacks.
type PathValidator struct {
	BaseDir string // The base directory that all paths must remain within
}

// Common path validation errors
var (
	ErrPathOutsideRepository = errors.New("path would access files outside repository")
	ErrInvalidPath           = errors.New("path contains invalid characters or sequences")
	ErrPathTooLong           = errors.New("path exceeds maximum length")
	ErrEmptyPath             = errors.New("path cannot be empty")
	ErrInvalidUnicode        = errors.New("path contains invalid unicode characters")
)

// NewPathValidator creates a new PathValidator for the given base directory.
// The baseDir must be an absolute path.
func NewPathValidator(baseDir string) (*PathValidator, error) {
	if baseDir == "" {
		return nil, customerrors.NewConfigurationError("baseDir", "", errors.New("base directory cannot be empty"))
	}

	// Convert to absolute path and clean it
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, customerrors.WrapWithFile(baseDir, "resolve absolute path", err)
	}

	// Clean the path to normalize it
	absBase = filepath.Clean(absBase)

	return &PathValidator{
		BaseDir: absBase,
	}, nil
}

// ValidatePath validates that the given path is safe and returns the cleaned path
// relative to the base directory. This function prevents directory traversal attacks.
func (v *PathValidator) ValidatePath(userPath string) (string, error) {
	// Check for empty path
	if userPath == "" {
		return "", ErrEmptyPath
	}

	// Check path length to prevent resource exhaustion
	if len(userPath) > MaxPathLength {
		return "", ErrPathTooLong
	}

	// Check for null bytes first (legacy attack but still worth defending against)
	if strings.Contains(userPath, "\x00") {
		return "", fmt.Errorf("%w: contains null byte", ErrInvalidPath)
	}

	// Validate unicode - reject paths with invalid unicode or control characters
	if err := v.validateUnicode(userPath); err != nil {
		return "", err
	}

	// Check for Windows-style absolute paths on non-Windows systems
	if err := v.validateCrossPlatformPaths(userPath); err != nil {
		return "", err
	}

	// Clean the path (removes . and .. elements, multiple slashes, etc.)
	cleaned := filepath.Clean(userPath)

	// If path is relative, join it with base directory
	var fullPath string
	if !filepath.IsAbs(cleaned) {
		fullPath = filepath.Join(v.BaseDir, cleaned)
	} else {
		fullPath = cleaned
	}

	// Resolve symlinks to prevent symlink attacks
	resolved, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		// If EvalSymlinks fails (e.g., file doesn't exist), we need to carefully
		// resolve any symlinks in the path components that do exist.
		resolved = v.resolveExistingSymlinks(fullPath)
	}

	// Ensure the resolved path is still within the base directory
	resolved = filepath.Clean(resolved)
	baseClean := filepath.Clean(v.BaseDir)

	// Check if resolved path is within base directory
	if !v.isWithinBaseDir(resolved, baseClean) {
		return "", ErrPathOutsideRepository
	}

	// Return path relative to base directory
	relPath, err := filepath.Rel(baseClean, resolved)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	// Additional check to ensure we don't return paths that start with ..
	if strings.HasPrefix(relPath, "..") {
		return "", ErrPathOutsideRepository
	}

	return relPath, nil
}

// JoinAndValidate joins the given path elements and validates the result.
// This is a safe alternative to filepath.Join when working with user input.
func (v *PathValidator) JoinAndValidate(elem ...string) (string, error) {
	if len(elem) == 0 {
		return "", ErrEmptyPath
	}

	// Join all elements
	joined := filepath.Join(elem...)

	// Validate the resulting path
	return v.ValidatePath(joined)
}

// validateUnicode checks for invalid unicode characters and control characters
// that might be used in path traversal attacks.
func (v *PathValidator) validateUnicode(path string) error {
	if !utf8.ValidString(path) {
		return fmt.Errorf("%w: invalid UTF-8 encoding", ErrInvalidUnicode)
	}

	// Check for control characters and other dangerous unicode
	for _, r := range path {
		if unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' {
			return fmt.Errorf("%w: contains control character", ErrInvalidPath)
		}
	}

	return nil
}

// validateCrossPlatformPaths checks for platform-specific absolute paths that
// should be rejected regardless of the current operating system.
func (v *PathValidator) validateCrossPlatformPaths(path string) error {
	// Check for Windows-style absolute paths (C:, D:, etc.)
	if len(path) >= 2 && path[1] == ':' {
		// Check if it looks like a Windows drive letter
		if (path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z') {
			return ErrPathOutsideRepository
		}
	}

	// Check for Windows UNC paths (\\server\share)
	if strings.HasPrefix(path, "\\\\") {
		return ErrPathOutsideRepository
	}

	return nil
}

// resolveExistingSymlinks resolves symlinks in the path components that exist,
// even if the final path doesn't exist. This prevents symlink attacks.
func (v *PathValidator) resolveExistingSymlinks(fullPath string) string {
	// Start from the full path and work backwards to find the longest
	// existing prefix, then resolve symlinks in that prefix.
	dir := fullPath
	var nonExistentSuffix string

	for {
		if _, err := os.Stat(dir); err == nil {
			// This path exists, try to resolve symlinks
			resolved, err := filepath.EvalSymlinks(dir)
			if err == nil {
				// Successfully resolved, reconstruct the full path
				if nonExistentSuffix != "" {
					return filepath.Join(resolved, nonExistentSuffix)
				}
				return resolved
			}
		}

		// Path doesn't exist or symlink resolution failed
		// Move up one directory level
		parent := filepath.Dir(dir)
		if parent == dir {
			// We've reached the root, can't go further
			break
		}

		// Add this component to the non-existent suffix
		base := filepath.Base(dir)
		if nonExistentSuffix == "" {
			nonExistentSuffix = base
		} else {
			nonExistentSuffix = filepath.Join(base, nonExistentSuffix)
		}
		dir = parent
	}

	// If we couldn't resolve anything, return the original path
	return fullPath
}

// isWithinBaseDir checks if the given path is within the base directory.
// Both paths should be absolute and cleaned.
func (v *PathValidator) isWithinBaseDir(path, baseDir string) bool {
	// Clean both paths to ensure consistency
	path = filepath.Clean(path)
	baseDir = filepath.Clean(baseDir)

	// If the paths are exactly equal, it's within the base
	if path == baseDir {
		return true
	}

	// Add separator to baseDir to avoid false positives
	// (e.g., /base should not match /basement)
	baseDirWithSep := baseDir + string(filepath.Separator)

	return strings.HasPrefix(path, baseDirWithSep)
}

// SafeFilePath is a convenience function that creates a PathValidator and validates a single path.
// This is useful for one-off validations.
func SafeFilePath(baseDir, userPath string) (string, error) {
	validator, err := NewPathValidator(baseDir)
	if err != nil {
		return "", err
	}

	return validator.ValidatePath(userPath)
}
