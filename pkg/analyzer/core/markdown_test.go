package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSimpleMarkdownAnalyzer_Basic_Functionality(t *testing.T) {
	// Create a minimal logger for testing
	logger := &TestLogger{}
	analyzer := NewSimpleMarkdownAnalyzer(logger)

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

		// Check fingerprinting
		if sections[0].Fingerprint == "" {
			t.Error("Expected section to have fingerprint")
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
		foundInlineExample := false
		for _, example := range examples {
			if strings.Contains(example, "fmt.Println") {
				foundGoExample = true
			}
			if strings.Contains(example, "example()") {
				foundInlineExample = true
			}
		}

		if !foundGoExample {
			t.Error("Expected to find Go example with fmt.Println")
		}
		if !foundInlineExample {
			t.Error("Expected to find inline example with example()")
		}
	})
}

func TestSimpleMarkdownAnalyzer_EdgeCases(t *testing.T) {
	logger := &TestLogger{}
	analyzer := NewSimpleMarkdownAnalyzer(logger)

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

	t.Run("ParseMarkdown_ComplexContent", func(t *testing.T) {
		content := `# Main Title

Some content here.

## Code Section

Here's some code:

` + "```go" + `
func example() {
    fmt.Println("test")
}
` + "```" + `

### Subsection

More details in subsection.

## Another Section

Final content.
`
		tmpFile := filepath.Join(t.TempDir(), "complex.md")
		err := os.WriteFile(tmpFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		result := analyzer.ParseMarkdown(tmpFile)
		if result.IsErr() {
			t.Fatalf("Unexpected error: %v", result.Error)
		}

		sections := result.Value
		if len(sections) != 4 { // Main Title, Code Section, Subsection, Another Section
			t.Errorf("Expected 4 sections, got %d", len(sections))
		}

		// Verify section hierarchy
		levels := make([]int, len(sections))
		for i, section := range sections {
			levels[i] = section.Level
		}

		expectedLevels := []int{1, 2, 3, 2}
		for i, expected := range expectedLevels {
			if i < len(levels) && levels[i] != expected {
				t.Errorf("Section %d: expected level %d, got %d", i, expected, levels[i])
			}
		}

		// Check that code blocks are preserved in content
		codeSection := sections[1] // "Code Section"
		if !strings.Contains(codeSection.Content, "func example()") {
			t.Error("Expected code section to contain the Go code block")
		}
	})
}

// TestLogger implements the Logger interface for testing
type TestLogger struct {
	messages []string
}

func (l *TestLogger) Info(msg string, fields ...interface{}) {
	l.messages = append(l.messages, "INFO: "+msg)
}

func (l *TestLogger) Warn(msg string, fields ...interface{}) {
	l.messages = append(l.messages, "WARN: "+msg)
}

func (l *TestLogger) Error(msg string, fields ...interface{}) {
	l.messages = append(l.messages, "ERROR: "+msg)
}

func (l *TestLogger) Debug(msg string, fields ...interface{}) {
	l.messages = append(l.messages, "DEBUG: "+msg)
}

func (l *TestLogger) GetMessages() []string {
	return l.messages
}

func (l *TestLogger) Clear() {
	l.messages = nil
}

// Benchmark tests
func BenchmarkSimpleMarkdownAnalyzer_ParseMarkdown(b *testing.B) {
	logger := &TestLogger{}
	analyzer := NewSimpleMarkdownAnalyzer(logger)

	// Create test file
	content := `# Benchmark Test

This is benchmark content for testing performance.

## Section 1

Content here.

## Section 2 

More content.

### Subsection

Details.
`
	tmpFile := filepath.Join(b.TempDir(), "bench.md")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
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
