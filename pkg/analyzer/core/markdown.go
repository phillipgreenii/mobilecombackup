// Package core provides core analyzer functionality for testing
package core

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/phillipgreenii/mobilecombackup/pkg/types"
)

// Logger interface for analyzer logging
type Logger interface {
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
}

// DocSection represents a section in documentation
type DocSection struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Level       int    `json:"level"`
	Anchor      string `json:"anchor"`
	Type        string `json:"type"`
	Fingerprint string `json:"fingerprint"`
	LastUpdated int64  `json:"last_updated"`
}

// SimpleMarkdownAnalyzer implements basic documentation analysis for Markdown files
type SimpleMarkdownAnalyzer struct {
	logger Logger
}

// NewSimpleMarkdownAnalyzer creates a new markdown analyzer
func NewSimpleMarkdownAnalyzer(logger Logger) *SimpleMarkdownAnalyzer {
	return &SimpleMarkdownAnalyzer{
		logger: logger,
	}
}

// ParseMarkdown parses markdown files and extracts sections
func (ma *SimpleMarkdownAnalyzer) ParseMarkdown(filePath string) types.Result[[]DocSection] { //nolint:funlen
	ma.logger.Debug("Parsing markdown file", "file", filePath)

	file, err := os.Open(filePath) //nolint:gosec
	if err != nil {
		return types.NewResultError[[]DocSection](fmt.Errorf("failed to open file %s: %w", filePath, err))
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			ma.logger.Warn("Failed to close file", "file", filePath, "error", closeErr)
		}
	}()

	// Get file modification time for all sections
	var fileModTime int64
	if fileInfo, err := os.Stat(filePath); err == nil {
		fileModTime = fileInfo.ModTime().Unix()
	}

	var sections []DocSection

	// Create scanner with increased buffer size to handle large content
	scanner := bufio.NewScanner(file)
	const maxScanTokenSize = 64 * 1024 * 1024 // 64MB buffer to handle large files
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	lineNumber := 0
	currentSection := DocSection{
		File: filePath,
		Type: "content",
	}
	var contentLines []string

	// Regular expressions for markdown parsing
	headerRegex := regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	codeBlockRegex := regexp.MustCompile("^```")

	inCodeBlock := false

	// Helper function to finalize a section with fingerprinting
	finalizeSection := func(section *DocSection) {
		if section.Content != "" {
			// Calculate SHA-256 fingerprint of content
			hasher := sha256.New()
			hasher.Write([]byte(section.Content))
			section.Fingerprint = fmt.Sprintf("%x", hasher.Sum(nil))
			section.LastUpdated = fileModTime
		}
	}

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// Handle code blocks
		if codeBlockRegex.MatchString(line) {
			inCodeBlock = !inCodeBlock
			contentLines = append(contentLines, line)
			continue
		}

		if inCodeBlock {
			contentLines = append(contentLines, line)
			continue
		}

		// Check for headers
		if matches := headerRegex.FindStringSubmatch(line); matches != nil {
			// Save previous section if it has content
			if len(contentLines) > 0 || currentSection.Title != "" {
				currentSection.Content = strings.Join(contentLines, "\n")
				finalizeSection(&currentSection)
				sections = append(sections, currentSection)
			}

			// Start new section
			level := len(matches[1])
			title := matches[2]
			anchor := ma.generateAnchor(title)

			currentSection = DocSection{
				File:   filePath,
				Line:   lineNumber,
				Title:  title,
				Level:  level,
				Anchor: anchor,
				Type:   "section",
			}
			contentLines = []string{}
		} else {
			contentLines = append(contentLines, line)
		}
	}

	// Add final section
	if len(contentLines) > 0 || currentSection.Title != "" {
		currentSection.Content = strings.Join(contentLines, "\n")
		finalizeSection(&currentSection)
		sections = append(sections, currentSection)
	}

	if err := scanner.Err(); err != nil {
		return types.NewResultError[[]DocSection](fmt.Errorf("error reading file %s: %w", filePath, err))
	}

	ma.logger.Debug("Parsed markdown sections with fingerprints", "file", filePath, "sections", len(sections))
	return types.NewResult(sections)
}

// ExtractCodeReferences finds code references in documentation
func (ma *SimpleMarkdownAnalyzer) ExtractCodeReferences(content string) types.Result[[]string] {
	ma.logger.Debug("Extracting code references from content")

	var references []string

	// Regular expressions for different types of code references
	patterns := []string{
		// Function calls: `functionName(`
		"`([a-zA-Z_][a-zA-Z0-9_]*)`",
		// Type references: `TypeName`
		"`([A-Z][a-zA-Z0-9_]*)`",
		// Package.Function: `pkg.Function`
		"`([a-z][a-zA-Z0-9_]*\\.[A-Z][a-zA-Z0-9_]*)`",
		// Code blocks with go
		"```go\\s+([^`]+)```",
		// Inline code with dots (likely qualified names)
		"`([a-zA-Z_][a-zA-Z0-9_]*\\.[a-zA-Z_][a-zA-Z0-9_]*)`",
	}

	for _, pattern := range patterns {
		regex := regexp.MustCompile(pattern)
		matches := regex.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				ref := strings.TrimSpace(match[1])
				if ref != "" {
					references = append(references, ref)
				}
			}
		}
	}

	// Remove duplicates
	unique := make(map[string]bool)
	var uniqueRefs []string
	for _, ref := range references {
		if !unique[ref] {
			unique[ref] = true
			uniqueRefs = append(uniqueRefs, ref)
		}
	}

	ma.logger.Debug("Extracted code references", "count", len(uniqueRefs))
	return types.NewResult(uniqueRefs)
}

// ExtractExamples extracts code examples from documentation
func (ma *SimpleMarkdownAnalyzer) ExtractExamples(content string) types.Result[[]string] {
	ma.logger.Debug("Extracting code examples from content")

	var examples []string

	// First, extract code blocks and remove them from content for inline processing
	codeBlockRegex := regexp.MustCompile("```(?:go|golang)?\\s*\\n([^`]+)```")
	matches := codeBlockRegex.FindAllStringSubmatch(content, -1)

	// Create a copy of content with code blocks removed
	contentForInline := content
	for _, match := range matches {
		if len(match) > 0 {
			// Remove the entire code block from content
			contentForInline = strings.ReplaceAll(contentForInline, match[0], "")
		}
		if len(match) > 1 {
			code := strings.TrimSpace(match[1])
			if code != "" {
				examples = append(examples, code)
			}
		}
	}

	// Extract inline code that looks like examples (from content with code blocks removed)
	inlineRegex := regexp.MustCompile("`([^`]+)`")
	inlineMatches := inlineRegex.FindAllStringSubmatch(contentForInline, -1)

	for _, match := range inlineMatches {
		if len(match) > 1 {
			code := strings.TrimSpace(match[1])
			// Only include if it looks like a function call or substantial code
			if strings.Contains(code, "(") && strings.Contains(code, ")") {
				examples = append(examples, code)
			}
		}
	}

	ma.logger.Debug("Extracted code examples", "count", len(examples))
	return types.NewResult(examples)
}

// Helper methods
func (ma *SimpleMarkdownAnalyzer) generateAnchor(title string) string {
	// Convert title to lowercase anchor
	anchor := strings.ToLower(title)
	// Replace spaces with hyphens
	anchor = regexp.MustCompile(`\s+`).ReplaceAllString(anchor, "-")
	// Remove special characters
	anchor = regexp.MustCompile(`[^a-z0-9\-]`).ReplaceAllString(anchor, "")
	// Remove multiple consecutive hyphens
	anchor = regexp.MustCompile(`-+`).ReplaceAllString(anchor, "-")
	// Trim hyphens from ends
	anchor = strings.Trim(anchor, "-")

	return anchor
}
