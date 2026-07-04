package analyzer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Simple test focusing just on the basic MarkdownAnalyzer functionality
// without the complex sync infrastructure

func TestMarkdownAnalyzer_Basic_Functionality(t *testing.T) {
	// Create a minimal logger for testing
	logger := &SimpleTestLogger{}
	analyzer := NewMarkdownAnalyzer(logger)

	t.Run("ParseMarkdown_WithValidContent", func(t *testing.T) {
		// Create test file
		content := `# Test Document

This is a test section.

## Section 2

More content here.
`
		tmpFile := filepath.Join(t.TempDir(), "test.md")
		err := os.WriteFile(tmpFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Test parsing
		result := analyzer.ParseMarkdown(tmpFile)
		if result.IsErr() {
			t.Fatalf("Unexpected error: %v", result.Error)
		}

		sections := result.Value
		if len(sections) < 2 {
			t.Errorf("Expected at least 2 sections, got %d", len(sections))
		}

		// Check first section
		if sections[0].Title != "Test Document" {
			t.Errorf("Expected first section title 'Test Document', got '%s'", sections[0].Title)
		}

		if sections[0].Level != 1 {
			t.Errorf("Expected first section level 1, got %d", sections[0].Level)
		}
	})

	t.Run("ExtractCodeReferences_BasicTest", func(t *testing.T) {
		content := "This mentions `ParseMarkdown` and `types.Result` as code references."
		result := analyzer.ExtractCodeReferences(content)

		if result.IsErr() {
			t.Fatalf("Unexpected error: %v", result.Error)
		}

		references := result.Value
		if len(references) == 0 {
			t.Error("Expected to find code references")
		}

		// Check that we found some references
		found := make(map[string]bool)
		for _, ref := range references {
			found[ref] = true
		}

		expectedRefs := []string{"ParseMarkdown", "types.Result"}
		for _, expected := range expectedRefs {
			if !found[expected] {
				t.Errorf("Expected reference %s not found in %v", expected, references)
			}
		}
	})

	t.Run("ExtractExamples_CodeBlocks", func(t *testing.T) {
		content := `Here's a Go example:

` + "```go" + `
func main() {
    fmt.Println("Hello")
}
` + "```" + `

And some inline code: ` + "`example()`" + `
`
		result := analyzer.ExtractExamples(content)
		if result.IsErr() {
			t.Fatalf("Unexpected error: %v", result.Error)
		}

		examples := result.Value
		if len(examples) == 0 {
			t.Error("Expected to find code examples")
		}

		// Check that we found the Go code block
		foundGoExample := false
		for _, example := range examples {
			if strings.Contains(example, "fmt.Println") {
				foundGoExample = true
				break
			}
		}

		if !foundGoExample {
			t.Error("Expected to find Go example with fmt.Println")
		}
	})
}

func TestMarkdownAnalyzer_EdgeCases(t *testing.T) {
	logger := &SimpleTestLogger{}
	analyzer := NewMarkdownAnalyzer(logger)

	t.Run("ParseMarkdown_EmptyFile", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "empty.md")
		err := os.WriteFile(tmpFile, []byte(""), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		result := analyzer.ParseMarkdown(tmpFile)
		if result.IsErr() {
			t.Fatalf("Unexpected error: %v", result.Error)
		}

		sections := result.Value
		// Empty file should return empty sections or a default section
		if len(sections) > 1 {
			t.Errorf("Expected 0-1 sections for empty file, got %d", len(sections))
		}
	})

	t.Run("ParseMarkdown_NonexistentFile", func(t *testing.T) {
		result := analyzer.ParseMarkdown("/nonexistent/file.md")
		if result.IsOk() {
			t.Error("Expected error for nonexistent file")
		}
	})

	t.Run("ExtractCodeReferences_EmptyContent", func(t *testing.T) {
		result := analyzer.ExtractCodeReferences("")
		if result.IsErr() {
			t.Fatalf("Unexpected error: %v", result.Error)
		}

		references := result.Value
		if len(references) != 0 {
			t.Errorf("Expected 0 references for empty content, got %d", len(references))
		}
	})
}

// SimpleTestLogger implements the Logger interface for testing
type SimpleTestLogger struct {
	messages []string
}

func (l *SimpleTestLogger) Info(msg string, fields ...interface{}) {
	l.messages = append(l.messages, "INFO: "+msg)
}

func (l *SimpleTestLogger) Warn(msg string, fields ...interface{}) {
	l.messages = append(l.messages, "WARN: "+msg)
}

func (l *SimpleTestLogger) Error(msg string, fields ...interface{}) {
	l.messages = append(l.messages, "ERROR: "+msg)
}

func (l *SimpleTestLogger) Debug(msg string, fields ...interface{}) {
	l.messages = append(l.messages, "DEBUG: "+msg)
}

func (l *SimpleTestLogger) GetMessages() []string {
	return l.messages
}
