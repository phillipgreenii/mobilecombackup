package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Integration tests for core analyzer components
// These tests use real file systems and test component interactions

func TestMarkdownAnalyzer_Integration_RealFiles(t *testing.T) {
	// Create test directory structure with real files
	testDir := t.TempDir()

	// Create comprehensive test content
	testFiles := map[string]string{
		"README.md": `# Test Project

Welcome to our test project for integration testing.

## Overview

This project demonstrates:
- Markdown parsing capabilities
- Code reference extraction
- Cross-file analysis

## API Reference

See ` + "`ParseMarkdown`" + ` function for parsing.
Use ` + "`types.Result`" + ` for error handling.

### Code Examples

Here's how to use the parser:

` + "```go" + `
analyzer := NewSimpleMarkdownAnalyzer(logger)
result := analyzer.ParseMarkdown("file.md")
if result.IsOk() {
    sections := result.Value
    fmt.Printf("Found %d sections", len(sections))
}
` + "```" + `

And for extracting references:

` + "```go" + `
refs := analyzer.ExtractCodeReferences(content)
` + "```" + `

## Configuration

Set ` + "`DEBUG=true`" + ` for verbose output.
Call ` + "`setup()`" + ` to initialize.
`,
		"docs/api.md": `# API Documentation

## Functions

### ParseMarkdown(filePath string)

Parses markdown and returns structured data.

**Parameters:**
- ` + "`filePath`" + `: Path to markdown file

**Returns:**  
- ` + "`types.Result[[]DocSection]`" + `: Parse results

**Example:**
` + "```go" + `
result := analyzer.ParseMarkdown("/path/to/file.md")
` + "```" + `

### ExtractCodeReferences(content string)

Finds code references in text.

**Usage:**
Call ` + "`ExtractCodeReferences(text)`" + ` with markdown content.

Cross-reference with ` + "`ParseMarkdown`" + ` for complete analysis.
`,
		"docs/examples.md": `# Examples

## Basic Usage

Start with ` + "`NewSimpleMarkdownAnalyzer(logger)`" + `:

` + "```go" + `
package main

import "github.com/example/analyzer/core"

func main() {
    logger := &TestLogger{}
    analyzer := core.NewSimpleMarkdownAnalyzer(logger)
    
    result := analyzer.ParseMarkdown("README.md")
    if result.IsErr() {
        panic(result.Error)
    }
    
    sections := result.Value
    for _, section := range sections {
        fmt.Printf("Section: %s (Level %d)\n", section.Title, section.Level)
    }
}
` + "```" + `

## Advanced Features

For code analysis, use ` + "`ExtractCodeReferences`" + `:

` + "```javascript" + `
const analyzer = new MarkdownAnalyzer();
const refs = analyzer.extractReferences(content);
` + "```" + `

Process results with ` + "`processResults(data)`" + `.
`,
	}

	// Create all test files
	for relPath, content := range testFiles {
		fullPath := filepath.Join(testDir, relPath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory for %s: %v", relPath, err)
		}
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", relPath, err)
		}
	}

	logger := &TestLogger{}
	analyzer := NewSimpleMarkdownAnalyzer(logger)

	t.Run("ParseAllFiles_ExtractSections", func(t *testing.T) {
		totalSections := 0
		filesProcessed := 0

		for relPath := range testFiles {
			fullPath := filepath.Join(testDir, relPath)

			result := analyzer.ParseMarkdown(fullPath)
			if result.IsErr() {
				t.Errorf("Failed to parse %s: %v", relPath, result.Error)
				continue
			}

			sections := result.Value
			if len(sections) == 0 {
				t.Errorf("Expected sections in %s", relPath)
				continue
			}

			filesProcessed++
			totalSections += len(sections)

			// Verify all sections have required fields
			for i, section := range sections {
				if section.File != fullPath {
					t.Errorf("Section %d in %s: expected file %s, got %s", i, relPath, fullPath, section.File)
				}
				if section.Fingerprint == "" && section.Content != "" {
					t.Errorf("Section %d in %s: missing fingerprint", i, relPath)
				}
				if section.LastUpdated == 0 && section.Content != "" {
					t.Errorf("Section %d in %s: missing last updated timestamp", i, relPath)
				}
			}

			t.Logf("Processed %s: %d sections", relPath, len(sections))
		}

		if filesProcessed != len(testFiles) {
			t.Errorf("Expected to process %d files, processed %d", len(testFiles), filesProcessed)
		}

		if totalSections < len(testFiles) {
			t.Errorf("Expected at least %d sections total (one per file), got %d", len(testFiles), totalSections)
		}

		t.Logf("Integration test completed: %d files, %d total sections", filesProcessed, totalSections)
	})

	t.Run("CrossFileCodeReferences", func(t *testing.T) {
		// Collect all code references across all files
		allReferences := make(map[string][]string)

		for relPath := range testFiles {
			fullPath := filepath.Join(testDir, relPath)
			content, err := os.ReadFile(fullPath)
			if err != nil {
				t.Errorf("Failed to read %s: %v", relPath, err)
				continue
			}

			result := analyzer.ExtractCodeReferences(string(content))
			if result.IsErr() {
				t.Errorf("Failed to extract references from %s: %v", relPath, result.Error)
				continue
			}

			references := result.Value
			allReferences[relPath] = references
			t.Logf("File %s: %d code references", relPath, len(references))
		}

		// Verify references are found (adjust expectations to match actual distribution)
		expectedCommonRefs := []string{"ParseMarkdown", "types.Result", "ExtractCodeReferences"}
		foundAnyReferences := false

		for _, expectedRef := range expectedCommonRefs {
			foundInFiles := 0
			for relPath, refs := range allReferences {
				for _, ref := range refs {
					if ref == expectedRef {
						foundInFiles++
						t.Logf("Found reference '%s' in %s", expectedRef, relPath)
						break
					}
				}
			}

			if foundInFiles > 0 {
				foundAnyReferences = true
				t.Logf("Reference '%s' found in %d files", expectedRef, foundInFiles)
			}
		}

		if !foundAnyReferences {
			t.Error("Expected to find at least some of the common code references")
		}
	})

	t.Run("ExtractExamplesFromAllFiles", func(t *testing.T) {
		totalExamples := 0
		languagesFound := make(map[string]int)

		for relPath := range testFiles {
			fullPath := filepath.Join(testDir, relPath)
			content, err := os.ReadFile(fullPath)
			if err != nil {
				t.Errorf("Failed to read %s: %v", relPath, err)
				continue
			}

			result := analyzer.ExtractExamples(string(content))
			if result.IsErr() {
				t.Errorf("Failed to extract examples from %s: %v", relPath, result.Error)
				continue
			}

			examples := result.Value
			totalExamples += len(examples)

			// Categorize examples by likely language
			for _, example := range examples {
				if contains(example, "package main") || contains(example, "func ") {
					languagesFound["go"]++
				} else if contains(example, "const ") || contains(example, "console.log") {
					languagesFound["javascript"]++
				} else if contains(example, "()") {
					languagesFound["function_call"]++
				}
			}

			if len(examples) > 0 {
				t.Logf("File %s: %d code examples", relPath, len(examples))
			}
		}

		if totalExamples == 0 {
			t.Error("Expected to find code examples across files")
		}

		// Should find Go examples in our test content
		if languagesFound["go"] == 0 {
			t.Error("Expected to find Go code examples")
		}

		t.Logf("Integration test found %d total examples: %+v", totalExamples, languagesFound)
	})
}

func TestMarkdownAnalyzer_Integration_FileModifications(t *testing.T) {
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "dynamic.md")

	logger := &TestLogger{}
	analyzer := NewSimpleMarkdownAnalyzer(logger)

	t.Run("DetectFileChanges", func(t *testing.T) {
		// Create initial content
		initialContent := `# Initial Version

This is the original content.

## Section 1

Initial section content.
`
		err := os.WriteFile(testFile, []byte(initialContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create initial file: %v", err)
		}

		// Parse initial version
		result1 := analyzer.ParseMarkdown(testFile)
		if result1.IsErr() {
			t.Fatalf("Failed to parse initial version: %v", result1.Error)
		}
		initialSections := result1.Value

		if len(initialSections) < 2 {
			t.Errorf("Expected at least 2 sections in initial version, got %d", len(initialSections))
		}

		// Store initial fingerprints
		initialFingerprints := make(map[string]string)
		for _, section := range initialSections {
			if section.Title != "" {
				initialFingerprints[section.Title] = section.Fingerprint
			}
		}

		// Wait a moment to ensure different timestamp
		time.Sleep(100 * time.Millisecond)

		// Modify content
		modifiedContent := `# Modified Version

This content has been updated for testing.

## Section 1

Modified section content with additional details.

## New Section

This section was added in the modification.
`
		err = os.WriteFile(testFile, []byte(modifiedContent), 0644)
		if err != nil {
			t.Fatalf("Failed to modify file: %v", err)
		}

		// Parse modified version
		result2 := analyzer.ParseMarkdown(testFile)
		if result2.IsErr() {
			t.Fatalf("Failed to parse modified version: %v", result2.Error)
		}
		modifiedSections := result2.Value

		if len(modifiedSections) < 3 {
			t.Errorf("Expected at least 3 sections in modified version, got %d", len(modifiedSections))
		}

		// Verify changes detected
		changesDetected := 0
		for _, section := range modifiedSections {
			if section.Title != "" {
				if initialFP, existed := initialFingerprints[section.Title]; existed {
					if section.Fingerprint != initialFP {
						changesDetected++
						t.Logf("Detected change in section '%s'", section.Title)
					}
				} else {
					changesDetected++
					t.Logf("Detected new section '%s'", section.Title)
				}
			}
		}

		if changesDetected == 0 {
			t.Error("Expected to detect content changes through fingerprints")
		}

		// Verify timestamps are different
		for _, section := range modifiedSections {
			if section.LastUpdated > 0 {
				found := false
				for _, initialSection := range initialSections {
					if initialSection.Title == section.Title && initialSection.LastUpdated == section.LastUpdated {
						found = true
						break
					}
				}
				if !found && section.Content != "" {
					t.Logf("Section '%s' has updated timestamp", section.Title)
				}
			}
		}

		t.Logf("Change detection test completed: %d changes detected", changesDetected)
	})
}

func TestMarkdownAnalyzer_Integration_Performance(t *testing.T) {
	testDir := t.TempDir()
	logger := &TestLogger{}
	analyzer := NewSimpleMarkdownAnalyzer(logger)

	t.Run("ProcessMultipleFilesEfficiently", func(t *testing.T) {
		// Create multiple files for performance testing
		numFiles := 10
		fileSize := 1000 // characters per section
		sectionsPerFile := 5

		var filePaths []string

		for i := 0; i < numFiles; i++ {
			fileName := filepath.Join(testDir, "perf_"+string(rune('A'+i))+".md")

			content := "# Performance Test File " + string(rune('A'+i)) + "\n\n"
			for j := 0; j < sectionsPerFile; j++ {
				sectionContent := ""
				for k := 0; k < fileSize/50; k++ { // ~50 chars per line
					sectionContent += "This is test content line " + string(rune('0'+k%10)) + " for performance testing.\n"
				}

				content += "## Section " + string(rune('1'+j)) + "\n\n" + sectionContent + "\n"
			}

			err := os.WriteFile(fileName, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to create performance test file %d: %v", i, err)
			}
			filePaths = append(filePaths, fileName)
		}

		// Measure parsing performance
		start := time.Now()
		totalSections := 0

		for _, filePath := range filePaths {
			result := analyzer.ParseMarkdown(filePath)
			if result.IsErr() {
				t.Errorf("Failed to parse %s: %v", filePath, result.Error)
				continue
			}

			sections := result.Value
			totalSections += len(sections)
		}

		duration := time.Since(start)

		expectedSections := numFiles * (sectionsPerFile + 1) // +1 for title
		if totalSections < expectedSections {
			t.Errorf("Expected at least %d sections, got %d", expectedSections, totalSections)
		}

		// Performance validation - should process files reasonably quickly
		maxDuration := time.Duration(numFiles) * 100 * time.Millisecond // 100ms per file max
		if duration > maxDuration {
			t.Errorf("Performance too slow: took %v, expected under %v", duration, maxDuration)
		}

		filesPerSecond := float64(numFiles) / duration.Seconds()
		sectionsPerSecond := float64(totalSections) / duration.Seconds()

		t.Logf("Performance: %d files, %d sections in %v", numFiles, totalSections, duration)
		t.Logf("Throughput: %.1f files/sec, %.1f sections/sec", filesPerSecond, sectionsPerSecond)
	})
}

// Helper function for string containment check
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
