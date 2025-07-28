package validation

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestChecksumValidatorImpl_CalculateFileChecksum(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewChecksumValidator(tempDir)
	
	// Test with non-existent file
	_, err := validator.CalculateFileChecksum(filepath.Join(tempDir, "nonexistent.txt"))
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	
	// Create test file with known content
	testContent := []byte("Hello, World!")
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Calculate expected checksum
	hasher := sha256.New()
	hasher.Write(testContent)
	expectedChecksum := fmt.Sprintf("%x", hasher.Sum(nil))
	
	// Test checksum calculation
	actualChecksum, err := validator.CalculateFileChecksum(testFile)
	if err != nil {
		t.Fatalf("Failed to calculate checksum: %v", err)
	}
	
	if actualChecksum != expectedChecksum {
		t.Errorf("Expected checksum %s, got %s", expectedChecksum, actualChecksum)
	}
	
	// Verify checksum format (64 hex characters)
	if len(actualChecksum) != 64 {
		t.Errorf("Expected checksum length 64, got %d", len(actualChecksum))
	}
}

func TestChecksumValidatorImpl_VerifyFileChecksum(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewChecksumValidator(tempDir)
	
	// Create test file with known content
	testContent := []byte("Hello, World!")
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Calculate correct checksum
	hasher := sha256.New()
	hasher.Write(testContent)
	correctChecksum := fmt.Sprintf("%x", hasher.Sum(nil))
	
	// Test with correct checksum
	err = validator.VerifyFileChecksum(testFile, correctChecksum)
	if err != nil {
		t.Errorf("Expected no error with correct checksum, got: %v", err)
	}
	
	// Test with incorrect checksum
	wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"
	err = validator.VerifyFileChecksum(testFile, wrongChecksum)
	if err == nil {
		t.Error("Expected error with wrong checksum")
	}
	
	// Test with non-existent file
	err = validator.VerifyFileChecksum(filepath.Join(tempDir, "nonexistent.txt"), correctChecksum)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestChecksumValidatorImpl_ValidateManifestChecksums(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewChecksumValidator(tempDir)
	
	// Create test files with known content
	files := map[string]string{
		"file1.txt": "Content of file 1",
		"file2.txt": "Content of file 2",
		"subdir/file3.txt": "Content of file 3",
	}
	
	var manifestEntries []FileEntry
	for fileName, content := range files {
		fullPath := filepath.Join(tempDir, fileName)
		
		// Create directory if needed
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		
		// Write file
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", fileName, err)
		}
		
		// Calculate correct checksum
		hasher := sha256.New()
		hasher.Write([]byte(content))
		checksum := fmt.Sprintf("%x", hasher.Sum(nil))
		
		manifestEntries = append(manifestEntries, FileEntry{
			File:      fileName,
			SHA256:    checksum,
			SizeBytes: int64(len(content)),
		})
	}
	
	// Test with all correct checksums
	manifest := &FileManifest{Files: manifestEntries}
	violations := validator.ValidateManifestChecksums(manifest)
	if len(violations) != 0 {
		t.Errorf("Expected no violations with correct checksums, got %d: %v", len(violations), violations)
	}
	
	// Test with incorrect checksum
	manifestWithWrongChecksum := &FileManifest{
		Files: []FileEntry{
			{
				File:      "file1.txt",
				SHA256:    "0000000000000000000000000000000000000000000000000000000000000000",
				SizeBytes: int64(len("Content of file 1")),
			},
		},
	}
	
	violations = validator.ValidateManifestChecksums(manifestWithWrongChecksum)
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation with wrong checksum, got %d", len(violations))
	}
	
	if len(violations) > 0 && violations[0].Type != ChecksumMismatch {
		t.Errorf("Expected ChecksumMismatch violation, got %s", violations[0].Type)
	}
	
	// Test with incorrect size
	manifestWithWrongSize := &FileManifest{
		Files: []FileEntry{
			{
				File:      "file1.txt",
				SHA256:    manifestEntries[0].SHA256, // Correct checksum
				SizeBytes: 999,                       // Wrong size
			},
		},
	}
	
	violations = validator.ValidateManifestChecksums(manifestWithWrongSize)
	if len(violations) < 1 {
		t.Errorf("Expected at least 1 violation with wrong size, got %d", len(violations))
	}
	
	// Should have a size mismatch violation (and possibly checksum mismatch too)
	foundSizeMismatch := false
	for _, violation := range violations {
		if violation.Type == SizeMismatch {
			foundSizeMismatch = true
			break
		}
	}
	
	if !foundSizeMismatch {
		t.Error("Expected to find SizeMismatch violation")
	}
	
	// Test with missing file
	manifestWithMissingFile := &FileManifest{
		Files: []FileEntry{
			{
				File:      "nonexistent.txt",
				SHA256:    "abc123",
				SizeBytes: 100,
			},
		},
	}
	
	violations = validator.ValidateManifestChecksums(manifestWithMissingFile)
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation with missing file, got %d", len(violations))
	}
	
	if len(violations) > 0 && violations[0].Type != MissingFile {
		t.Errorf("Expected MissingFile violation, got %s", violations[0].Type)
	}
}

func TestChecksumValidatorImpl_ValidateManifestChecksums_MultipleViolations(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewChecksumValidator(tempDir)
	
	// Create one valid file
	testContent := []byte("Valid content")
	testFile := filepath.Join(tempDir, "valid.txt")
	err := os.WriteFile(testFile, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	hasher := sha256.New()
	hasher.Write(testContent)
	correctChecksum := fmt.Sprintf("%x", hasher.Sum(nil))
	
	// Create manifest with multiple issues
	manifest := &FileManifest{
		Files: []FileEntry{
			{
				File:      "valid.txt",
				SHA256:    correctChecksum,
				SizeBytes: int64(len(testContent)),
			}, // Valid entry
			{
				File:      "valid.txt",
				SHA256:    "wrong_checksum_here",
				SizeBytes: int64(len(testContent)),
			}, // Wrong checksum
			{
				File:      "valid.txt",
				SHA256:    correctChecksum,
				SizeBytes: 999,
			}, // Wrong size
			{
				File:      "missing.txt",
				SHA256:    "abc123",
				SizeBytes: 100,
			}, // Missing file
		},
	}
	
	violations := validator.ValidateManifestChecksums(manifest)
	
	// Should have 3 violations (wrong checksum, wrong size, missing file)
	// The first entry should be valid with no violations
	if len(violations) != 3 {
		t.Errorf("Expected 3 violations, got %d: %v", len(violations), violations)
	}
	
	// Check that we have the expected violation types
	violationTypes := make(map[ViolationType]int)
	for _, violation := range violations {
		violationTypes[violation.Type]++
	}
	
	expectedTypes := map[ViolationType]int{
		ChecksumMismatch: 1,
		SizeMismatch:     1,
		MissingFile:      1,
	}
	
	for expectedType, expectedCount := range expectedTypes {
		if violationTypes[expectedType] != expectedCount {
			t.Errorf("Expected %d violations of type %s, got %d", 
				expectedCount, expectedType, violationTypes[expectedType])
		}
	}
}

func TestChecksumValidatorImpl_LargeFilePerformance(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewChecksumValidator(tempDir)
	
	// Create a moderately large file (1MB) to test performance
	largeContent := make([]byte, 1024*1024) // 1MB
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	
	largeFile := filepath.Join(tempDir, "large.txt")
	err := os.WriteFile(largeFile, largeContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create large test file: %v", err)
	}
	
	// Calculate checksum - this should complete in reasonable time
	checksum, err := validator.CalculateFileChecksum(largeFile)
	if err != nil {
		t.Fatalf("Failed to calculate checksum for large file: %v", err)
	}
	
	// Verify checksum format
	if len(checksum) != 64 {
		t.Errorf("Expected checksum length 64, got %d", len(checksum))
	}
	
	// Verify the checksum by recalculating
	checksum2, err := validator.CalculateFileChecksum(largeFile)
	if err != nil {
		t.Fatalf("Failed to recalculate checksum: %v", err)
	}
	
	if checksum != checksum2 {
		t.Error("Checksum calculation is not deterministic")
	}
}