// Package manifest provides file manifest generation and verification capabilities.
//
// The manifest package creates and manages YAML-based file manifests (files.yaml)
// that track all files in a repository with their checksums, sizes, and modification
// times. This enables data integrity verification and change detection across
// repository operations.
//
// # File Manifest Structure
//
// The generated files.yaml contains:
//   - Version information for compatibility tracking
//   - Generation timestamp and tool information
//   - Complete file listing with SHA-256 checksums
//   - File sizes and modification timestamps
//   - Exclusion of manifest files themselves and rejected/ directories
//
// # Usage Example
//
// Basic manifest generation:
//
//	generator := NewManifestGenerator("/path/to/repository")
//
//	// Generate manifest from current repository state
//	manifest, err := generator.GenerateFileManifest()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Save manifest to files.yaml
//	err = generator.SaveFileManifest(manifest)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Generate checksum file for the manifest
//	err = generator.GenerateManifestChecksum()
//	if err != nil {
//		log.Fatal(err)
//	}
//
// # Security Features
//
//   - SHA-256 checksums for strong integrity verification
//   - Path traversal protection using security package
//   - Atomic file operations with temporary files
//   - Exclusion of sensitive or temporary files
//
// # File Exclusions
//
// The manifest automatically excludes:
//   - files.yaml (the manifest itself)
//   - files.yaml.sha256 (the manifest checksum)
//   - rejected/ directory contents
//   - Temporary and backup files
//
// # Performance Considerations
//
// Manifest generation performs full repository scans and calculates checksums
// for all files. For large repositories, this can be I/O intensive. The operation
// is designed to be memory-efficient, processing files individually rather than
// loading entire contents into memory.
package manifest

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ManifestGenerator handles generation and management of file manifests
type ManifestGenerator struct {
	repositoryRoot string
}

// NewManifestGenerator creates a new manifest generator
func NewManifestGenerator(repositoryRoot string) *ManifestGenerator {
	return &ManifestGenerator{
		repositoryRoot: repositoryRoot,
	}
}

// GenerateFileManifest scans the repository and creates a complete file manifest
// It excludes files.yaml itself, files.yaml.sha256, and anything in rejected/
func (g *ManifestGenerator) GenerateFileManifest() (*FileManifest, error) {
	now := time.Now().UTC()
	manifest := &FileManifest{
		Version:   "1.0",
		Generated: now.Format(time.RFC3339),
		Generator: "mobilecombackup dev",
		Files:     []FileEntry{},
	}

	err := filepath.Walk(g.repositoryRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path with forward slashes for cross-platform consistency
		relPath, err := filepath.Rel(g.repositoryRoot, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath) // Convert to forward slashes

		// Skip files that should not be included in manifest
		if g.shouldSkipFile(relPath) {
			return nil
		}

		// Calculate SHA-256 hash
		hash, err := calculateFileHash(path)
		if err != nil {
			return fmt.Errorf("failed to calculate hash for %s: %w", relPath, err)
		}

		// Add to manifest
		manifest.Files = append(manifest.Files, FileEntry{
			Name:     relPath,
			Size:     info.Size(),
			Checksum: fmt.Sprintf("sha256:%s", hash),
			Modified: info.ModTime().UTC().Format(time.RFC3339),
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan repository: %w", err)
	}

	return manifest, nil
}

// WriteManifestFiles writes the manifest and its checksum to the repository
func (g *ManifestGenerator) WriteManifestFiles(manifest *FileManifest) error {
	// Write files.yaml
	if err := g.writeManifest(manifest); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	// Write files.yaml.sha256
	if err := g.writeManifestChecksum(); err != nil {
		return fmt.Errorf("failed to write checksum: %w", err)
	}

	return nil
}

// WriteManifestOnly writes only files.yaml (not the checksum)
func (g *ManifestGenerator) WriteManifestOnly(manifest *FileManifest) error {
	return g.writeManifest(manifest)
}

// WriteChecksumOnly writes only files.yaml.sha256 (if it doesn't exist)
func (g *ManifestGenerator) WriteChecksumOnly() error {
	checksumPath := filepath.Join(g.repositoryRoot, "files.yaml.sha256")

	// Check if checksum file already exists
	if _, err := os.Stat(checksumPath); err == nil {
		// File exists, don't overwrite
		return nil
	}

	return g.writeManifestChecksum()
}

// shouldSkipFile determines if a file should be excluded from the manifest
func (g *ManifestGenerator) shouldSkipFile(relPath string) bool {
	// Skip files.yaml itself and its checksum
	if relPath == "files.yaml" || relPath == "files.yaml.sha256" {
		return true
	}

	// Skip temporary files
	if strings.HasSuffix(relPath, ".tmp") {
		return true
	}

	// Skip anything in rejected/ directory
	if strings.HasPrefix(relPath, "rejected/") {
		return true
	}

	// Skip hidden files (starting with .)
	baseName := filepath.Base(relPath)
	if strings.HasPrefix(baseName, ".") && baseName != ".mobilecombackup.yaml" {
		return true
	}

	return false
}

// writeManifest writes the manifest to files.yaml
func (g *ManifestGenerator) writeManifest(manifest *FileManifest) error {
	manifestPath := filepath.Join(g.repositoryRoot, "files.yaml")

	// Marshal to YAML
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal files manifest: %w", err)
	}

	// Write to file atomically
	tempPath := manifestPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temporary files manifest: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, manifestPath); err != nil {
		_ = os.Remove(tempPath) // Clean up temp file, ignore error
		return fmt.Errorf("failed to rename files manifest: %w", err)
	}

	return nil
}

// writeManifestChecksum writes the checksum file for files.yaml
func (g *ManifestGenerator) writeManifestChecksum() error {
	manifestPath := filepath.Join(g.repositoryRoot, "files.yaml")
	checksumPath := filepath.Join(g.repositoryRoot, "files.yaml.sha256")

	// Calculate hash of files.yaml
	hash, err := calculateFileHash(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to calculate files.yaml hash: %w", err)
	}

	// Create checksum content in standard format: "hash  filename"
	checksumContent := hash + "  files.yaml\n"

	// Write to file atomically
	tempPath := checksumPath + ".tmp"
	if err := os.WriteFile(tempPath, []byte(checksumContent), 0600); err != nil {
		return fmt.Errorf("failed to write temporary checksum file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, checksumPath); err != nil {
		_ = os.Remove(tempPath) // Clean up temp file, ignore error
		return fmt.Errorf("failed to rename checksum file: %w", err)
	}

	return nil
}

// calculateFileHash calculates SHA-256 hash of a file
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath) // #nosec G304
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
