package analyzer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MockLogger implements the Logger interface for testing
type MockLogger struct {
	messages []string
}

func (m *MockLogger) Debug(msg string, args ...interface{}) {
	m.messages = append(m.messages, "DEBUG: "+msg)
}

func (m *MockLogger) Info(msg string, args ...interface{}) {
	m.messages = append(m.messages, "INFO: "+msg)
}

func (m *MockLogger) Warn(msg string, args ...interface{}) {
	m.messages = append(m.messages, "WARN: "+msg)
}

func (m *MockLogger) Error(msg string, args ...interface{}) {
	m.messages = append(m.messages, "ERROR: "+msg)
}

// Test fixtures
const testMarkdownContent = `# Test Document

This is a test markdown document for analyzer testing.

## Section 1

Some content with [a link](example.com) and inline code ` + "`example`" + `.

### Subsection 1.1

More content here.

## Section 2

` + "```go" + `
func example() {
    fmt.Println("Hello, World!")
}
` + "```" + `

End of document.
`

const testMarkdownSimple = `# Simple Document

Basic content.
`

func TestMarkdownAnalyzer_ParseMarkdown(t *testing.T) {
	logger := &MockLogger{}
	analyzer := NewMarkdownAnalyzer(logger)

	tests := []struct {
		name     string
		content  string
		wantErr  bool
		sections int
	}{
		{
			name:     "valid markdown with multiple sections",
			content:  testMarkdownContent,
			wantErr:  false,
			sections: 4, // # Test Document, ## Section 1, ### Subsection 1.1, ## Section 2
		},
		{
			name:     "simple markdown",
			content:  testMarkdownSimple,
			wantErr:  false,
			sections: 1,
		},
		{
			name:     "empty file",
			content:  "",
			wantErr:  false,
			sections: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary test file
			tmpFile := filepath.Join(t.TempDir(), "test.md")
			err := os.WriteFile(tmpFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Test parsing
			result := analyzer.ParseMarkdown(tmpFile)

			if tt.wantErr && result.IsOk() {
				t.Errorf("Expected error but got success")
			}
			if !tt.wantErr && result.IsErr() {
				t.Errorf("Expected success but got error: %v", result.Error)
			}

			if result.IsOk() {
				sections := result.Value
				if len(sections) != tt.sections {
					t.Errorf("Expected %d sections, got %d", tt.sections, len(sections))
				}

				// Verify sections have required fields
				for i, section := range sections {
					if section.File != tmpFile {
						t.Errorf("Section %d: expected file %s, got %s", i, tmpFile, section.File)
					}
					if section.Title == "" && section.Content != "" {
						t.Errorf("Section %d: missing title", i)
					}
					if section.Fingerprint == "" && section.Content != "" {
						t.Errorf("Section %d: missing fingerprint", i)
					}
				}
			}
		})
	}
}

func TestMarkdownAnalyzer_ParseMarkdown_NonexistentFile(t *testing.T) {
	logger := &MockLogger{}
	analyzer := NewMarkdownAnalyzer(logger)

	result := analyzer.ParseMarkdown("/nonexistent/file.md")
	if result.IsOk() {
		t.Error("Expected error for nonexistent file")
	}
}

func TestMarkdownAnalyzer_ExtractCodeReferences(t *testing.T) {
	logger := &MockLogger{}
	analyzer := NewMarkdownAnalyzer(logger)

	content := "This mentions `ParseMarkdown` and `types.Result` as code references."
	result := analyzer.ExtractCodeReferences(content)

	if result.IsErr() {
		t.Fatalf("Unexpected error: %v", result.Error)
	}

	references := result.Value
	expectedRefs := []string{"ParseMarkdown", "types.Result"}

	if len(references) < 2 {
		t.Errorf("Expected at least 2 references, got %d", len(references))
	}

	// Check that expected references are found
	found := make(map[string]bool)
	for _, ref := range references {
		found[ref] = true
	}

	for _, expected := range expectedRefs {
		if !found[expected] {
			t.Errorf("Expected reference %s not found in %v", expected, references)
		}
	}
}

func TestMarkdownAnalyzer_ValidateLinks(t *testing.T) {
	logger := &MockLogger{}
	analyzer := NewMarkdownAnalyzer(logger)

	// Create test markdown files with links
	testDir := t.TempDir()

	// File with valid internal link
	validContent := `# Test
[Valid link](valid.md)
`
	validFile := filepath.Join(testDir, "valid.md")
	err := os.WriteFile(validFile, []byte(validContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create valid test file: %v", err)
	}

	// Create the target file
	targetFile := filepath.Join(testDir, "valid.md")
	err = os.WriteFile(targetFile, []byte("# Target"), 0644)
	if err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// File with broken link (placeholder that the validator detects)
	brokenContent := `# Test
[Broken link](#todo)
`
	brokenFile := filepath.Join(testDir, "broken.md")
	err = os.WriteFile(brokenFile, []byte(brokenContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create broken test file: %v", err)
	}

	result := analyzer.ValidateLinks([]string{validFile, brokenFile})
	if result.IsErr() {
		t.Fatalf("Unexpected error: %v", result.Error)
	}

	inconsistencies := result.Value
	// Should find broken link inconsistency
	if len(inconsistencies) == 0 {
		t.Error("Expected to find at least one broken link inconsistency")
	}

	// Check that we found a broken link inconsistency
	foundBrokenLink := false
	for _, inc := range inconsistencies {
		if inc.Type == InconsistencyBrokenLink {
			foundBrokenLink = true
			if !strings.Contains(inc.DocFile, "broken.md") {
				t.Errorf("Expected broken link in broken.md, found in %s", inc.DocFile)
			}
		}
	}

	if !foundBrokenLink {
		t.Error("Expected to find broken link inconsistency")
	}
}

func TestMarkdownAnalyzer_ExtractExamples(t *testing.T) {
	logger := &MockLogger{}
	analyzer := NewMarkdownAnalyzer(logger)

	content := `# Test Document

Here's a Go example:

` + "```go" + `
func main() {
    fmt.Println("Hello")
}
` + "```" + `

And a JavaScript example:

` + "```javascript" + `
console.log("Hello");
` + "```" + `
`

	result := analyzer.ExtractExamples(content)
	if result.IsErr() {
		t.Fatalf("Unexpected error: %v", result.Error)
	}

	examples := result.Value
	if len(examples) != 2 {
		t.Errorf("Expected 2 examples, got %d", len(examples))
	}

	// Check that examples contain expected code
	goFound := false
	jsFound := false
	for _, example := range examples {
		if strings.Contains(example, "fmt.Println") {
			goFound = true
		}
		if strings.Contains(example, "console.log") {
			jsFound = true
		}
	}

	if !goFound {
		t.Error("Expected to find Go example")
	}
	if !jsFound {
		t.Error("Expected to find JavaScript example")
	}
}

func TestDocumentationStateManager_NewDocumentationStateManager(t *testing.T) {
	logger := &MockLogger{}
	tmpFile := filepath.Join(t.TempDir(), "state.json")

	manager := NewDocumentationStateManager(tmpFile, logger)
	if manager == nil {
		t.Fatal("Expected non-nil DocumentationStateManager")
	}

	// Test basic functionality
	state := manager.GetState()
	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	// Initially should have empty state
	if state.Version == "" {
		state.Version = "1.0"
	}
	if len(state.FileStates) != 0 {
		t.Errorf("Expected empty file states, got %d", len(state.FileStates))
	}
}

func TestConcurrentDocumentScanner_Basic(t *testing.T) {
	logger := &MockLogger{}
	stateManager := NewDocumentationStateManager(filepath.Join(t.TempDir(), "state.json"), logger)
	markdownAnalyzer := NewMarkdownAnalyzer(logger)

	scanner := NewConcurrentDocumentScanner(markdownAnalyzer, stateManager, logger)
	if scanner == nil {
		t.Fatal("Expected non-nil ConcurrentDocumentScanner")
	}

	// Test configuration
	config := ScanConfig{
		MaxWorkers:     2,
		BatchSize:      10,
		ReportProgress: true,
	}
	scanner.SetConfig(config)

	// Create test files
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "test.md")
	err := os.WriteFile(testFile, []byte(testMarkdownSimple), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test scanning
	result := scanner.ScanDocuments([]string{testFile})
	if result.IsErr() {
		t.Fatalf("Unexpected error: %v", result.Error)
	}

	scanResult := result.Value
	if scanResult.Progress.TotalFiles != 1 {
		t.Errorf("Expected 1 file scanned, got %d", scanResult.Progress.TotalFiles)
	}
	if scanResult.Progress.ProcessedFiles != 1 {
		t.Errorf("Expected 1 file processed, got %d", scanResult.Progress.ProcessedFiles)
	}
}

// Benchmark tests
func BenchmarkMarkdownAnalyzer_ParseMarkdown(b *testing.B) {
	logger := &MockLogger{}
	analyzer := NewMarkdownAnalyzer(logger)

	// Create test file
	tmpFile := filepath.Join(b.TempDir(), "bench.md")
	err := os.WriteFile(tmpFile, []byte(testMarkdownContent), 0644)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := analyzer.ParseMarkdown(tmpFile)
		if result.IsErr() {
			b.Fatalf("Unexpected error: %v", result.Error)
		}
	}
}
