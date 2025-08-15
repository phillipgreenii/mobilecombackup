package importer

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const manifestVersion = "1.0"

// generateManifestFile creates a files.yaml manifest with checksums for all repository files
func generateManifestFile(repoRoot string, version string) error {
	entries, err := collectRepositoryFiles(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to collect repository files: %w", err)
	}

	manifest := &FileManifest{
		Version:   manifestVersion,
		Generated: time.Now().UTC().Format(time.RFC3339),
		Generator: fmt.Sprintf("mobilecombackup %s", version),
		Files:     entries,
	}

	// Write manifest to file
	manifestPath := filepath.Join(repoRoot, "files.yaml")
	file, err := os.Create(manifestPath) // #nosec G304
	if err != nil {
		return fmt.Errorf("failed to create manifest file: %w", err)
	}
	defer func() { _ = file.Close() }()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(manifest); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	// Also generate files.yaml.sha256
	if err := generateManifestChecksum(manifestPath); err != nil {
		return fmt.Errorf("failed to generate manifest checksum: %w", err)
	}

	return nil
}

// collectRepositoryFiles walks the repository and collects all files with their metadata
func collectRepositoryFiles(repoRoot string) ([]FileEntry, error) {
	var entries []FileEntry

	err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path from repository root
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Skip files.yaml and files.yaml.sha256 themselves
		if relPath == "files.yaml" || relPath == "files.yaml.sha256" {
			return nil
		}

		// Calculate checksum
		checksum, err := calculateFileChecksum(path)
		if err != nil {
			return fmt.Errorf("failed to calculate checksum for %s: %w", relPath, err)
		}

		entry := FileEntry{
			Name:     relPath,
			Size:     info.Size(),
			Checksum: fmt.Sprintf("sha256:%s", checksum),
			Modified: info.ModTime().UTC().Format(time.RFC3339),
		}

		entries = append(entries, entry)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort entries by name for consistent output
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	return entries, nil
}

// calculateFileChecksum calculates the SHA-256 checksum of a file
func calculateFileChecksum(path string) (string, error) {
	file, err := os.Open(path) // #nosec G304
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// generateManifestChecksum creates files.yaml.sha256 with the checksum of files.yaml
func generateManifestChecksum(manifestPath string) error {
	checksum, err := calculateFileChecksum(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to calculate manifest checksum: %w", err)
	}

	checksumPath := manifestPath + ".sha256"
	content := fmt.Sprintf("%s  %s\n", checksum, filepath.Base(manifestPath))

	if err := os.WriteFile(checksumPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write checksum file: %w", err)
	}

	return nil
}

// verifyManifest verifies that all files in the manifest exist and have correct checksums
func verifyManifest(repoRoot string) error {
	manifestPath := filepath.Join(repoRoot, "files.yaml")

	// Read manifest
	data, err := os.ReadFile(manifestPath) // #nosec G304
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest FileManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Verify each file
	for _, entry := range manifest.Files {
		filePath := filepath.Join(repoRoot, entry.Name)

		// Check file exists
		info, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("file %s not found: %w", entry.Name, err)
		}

		// Check size
		if info.Size() != entry.Size {
			return fmt.Errorf("file %s size mismatch: expected %d, got %d",
				entry.Name, entry.Size, info.Size())
		}

		// Check checksum
		actualChecksum, err := calculateFileChecksum(filePath)
		if err != nil {
			return fmt.Errorf("failed to calculate checksum for %s: %w", entry.Name, err)
		}

		expectedChecksum := strings.TrimPrefix(entry.Checksum, "sha256:")
		if actualChecksum != expectedChecksum {
			return fmt.Errorf("file %s checksum mismatch: expected %s, got %s",
				entry.Name, expectedChecksum, actualChecksum)
		}
	}

	return nil
}
