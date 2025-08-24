package analyzer

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/phillipgreenii/mobilecombackup/pkg/types"
)

// MarkdownAnalyzer implements documentation analysis for Markdown files
type MarkdownAnalyzer struct {
	logger Logger
}

// NewMarkdownAnalyzer creates a new markdown analyzer
func NewMarkdownAnalyzer(logger Logger) *MarkdownAnalyzer {
	return &MarkdownAnalyzer{
		logger: logger,
	}
}

// ParseMarkdown parses markdown files and extracts sections
func (ma *MarkdownAnalyzer) ParseMarkdown(filePath string) types.Result[[]DocSection] {
	ma.logger.Debug("Parsing markdown file", "file", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		return types.NewResultError[[]DocSection](fmt.Errorf("failed to open file %s: %w", filePath, err))
	}
	defer file.Close()

	var sections []DocSection
	scanner := bufio.NewScanner(file)
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
		sections = append(sections, currentSection)
	}

	if err := scanner.Err(); err != nil {
		return types.NewResultError[[]DocSection](fmt.Errorf("error reading file %s: %w", filePath, err))
	}

	ma.logger.Debug("Parsed markdown sections", "file", filePath, "sections", len(sections))
	return types.NewResult(sections)
}

// ExtractCodeReferences finds code references in documentation
func (ma *MarkdownAnalyzer) ExtractCodeReferences(content string) types.Result[[]string] {
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

// ValidateLinks validates all links in documentation
func (ma *MarkdownAnalyzer) ValidateLinks(docFiles []string) types.Result[[]Inconsistency] {
	ma.logger.Debug("Validating links in documentation", "files", len(docFiles))

	var inconsistencies []Inconsistency
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

	for _, docFile := range docFiles {
		file, err := os.Open(docFile)
		if err != nil {
			ma.logger.Warn("Failed to open file for link validation", "file", docFile, "error", err)
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNumber := 0

		for scanner.Scan() {
			lineNumber++
			line := scanner.Text()

			matches := linkRegex.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				if len(match) >= 3 {
					linkText := match[1]
					linkURL := match[2]

					// Validate link
					if err := ma.validateLink(linkURL); err != nil {
						inconsistency := Inconsistency{
							ID:          fmt.Sprintf("broken-link-%s-%d", docFile, lineNumber),
							Type:        InconsistencyBrokenLink,
							Severity:    SeverityMedium,
							Title:       "Broken link detected",
							Description: fmt.Sprintf("Link '%s' is broken: %s", linkText, err.Error()),
							DocFile:     docFile,
							DocLine:     lineNumber,
							Actual:      linkURL,
							Suggestion:  "Fix the link URL or remove the broken link",
							Context: InconsistencyContext{
								DocSection: DocSection{
									File:    docFile,
									Line:    lineNumber,
									Content: line,
								},
							},
							Metadata: map[string]interface{}{
								"link_text": linkText,
								"link_url":  linkURL,
							},
						}
						inconsistencies = append(inconsistencies, inconsistency)
					}
				}
			}
		}

		file.Close()
	}

	ma.logger.Debug("Link validation completed", "inconsistencies", len(inconsistencies))
	return types.NewResult(inconsistencies)
}

// ExtractExamples extracts code examples from documentation
func (ma *MarkdownAnalyzer) ExtractExamples(content string) types.Result[[]string] {
	ma.logger.Debug("Extracting code examples from content")

	var examples []string

	// Extract code blocks
	codeBlockRegex := regexp.MustCompile("```(?:go|golang)?\\s*\\n([^`]+)```")
	matches := codeBlockRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			code := strings.TrimSpace(match[1])
			if code != "" {
				examples = append(examples, code)
			}
		}
	}

	// Extract inline code that looks like examples
	inlineRegex := regexp.MustCompile("`([^`]+)`")
	inlineMatches := inlineRegex.FindAllStringSubmatch(content, -1)

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

func (ma *MarkdownAnalyzer) generateAnchor(title string) string {
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

func (ma *MarkdownAnalyzer) validateLink(linkURL string) error {
	// Simplified link validation
	// In a full implementation, this would check HTTP links, file existence, etc.

	// Check for obviously broken patterns
	if strings.TrimSpace(linkURL) == "" {
		return fmt.Errorf("empty link URL")
	}

	// Check for placeholder links
	placeholders := []string{"#", "#todo", "TODO", "FIXME", "example.com"}
	for _, placeholder := range placeholders {
		if strings.Contains(linkURL, placeholder) {
			return fmt.Errorf("placeholder link: %s", placeholder)
		}
	}

	// Check for internal file references
	if strings.HasPrefix(linkURL, "./") || strings.HasPrefix(linkURL, "../") || strings.HasSuffix(linkURL, ".md") {
		// For file links, check if file exists (simplified check)
		if !strings.HasPrefix(linkURL, "http") {
			// This would need more sophisticated file existence checking
			// For now, assume local files might exist
			return nil
		}
	}

	// Check for malformed URLs
	if strings.HasPrefix(linkURL, "http") {
		// Basic URL format validation
		if !strings.Contains(linkURL, "://") {
			return fmt.Errorf("malformed URL")
		}
	}

	return nil
}
