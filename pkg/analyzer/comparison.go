package analyzer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/phillipgreenii/mobilecombackup/pkg/types"
)

// DefaultComparisonEngine implements comparison between code and documentation
type DefaultComparisonEngine struct {
	logger Logger
}

// NewDefaultComparisonEngine creates a new comparison engine
func NewDefaultComparisonEngine(logger Logger) *DefaultComparisonEngine {
	return &DefaultComparisonEngine{
		logger: logger,
	}
}

// DetectInconsistencies finds inconsistencies between code and documentation
func (dce *DefaultComparisonEngine) DetectInconsistencies(codeSymbols []CodeSymbol, docSections []DocSection) types.Result[[]Inconsistency] {
	dce.logger.Debug("Detecting inconsistencies", "code_symbols", len(codeSymbols), "doc_sections", len(docSections))

	var inconsistencies []Inconsistency

	// Check for missing documentation
	missingDocs := dce.findMissingDocumentation(codeSymbols)
	inconsistencies = append(inconsistencies, missingDocs...)

	// Check for outdated documentation
	outdatedDocs := dce.findOutdatedDocumentation(codeSymbols, docSections)
	inconsistencies = append(inconsistencies, outdatedDocs...)

	// Check for documentation without corresponding code
	orphanedDocs := dce.findOrphanedDocumentation(codeSymbols, docSections)
	inconsistencies = append(inconsistencies, orphanedDocs...)

	dce.logger.Debug("Inconsistency detection completed", "total", len(inconsistencies))
	return types.NewResult(inconsistencies)
}

// CompareSignatures compares function signatures between code and docs
func (dce *DefaultComparisonEngine) CompareSignatures(codeSymbol CodeSymbol, docContent string) types.Result[*Inconsistency] {
	dce.logger.Debug("Comparing signatures", "symbol", codeSymbol.Name)

	// Extract signature from documentation
	docSignature := dce.extractSignatureFromDoc(docContent, codeSymbol.Name)

	if docSignature == "" {
		// No signature found in documentation
		return types.NewResult((*Inconsistency)(nil))
	}

	// Compare signatures
	if !dce.signaturesMatch(codeSymbol.Signature, docSignature) {
		inconsistency := &Inconsistency{
			ID:          fmt.Sprintf("signature-mismatch-%s", codeSymbol.Name),
			Type:        InconsistencyTypeMismatch,
			Severity:    SeverityHigh,
			Title:       "Function signature mismatch",
			Description: fmt.Sprintf("Function signature in documentation doesn't match code for '%s'", codeSymbol.Name),
			CodeFile:    codeSymbol.File,
			CodeLine:    codeSymbol.Line,
			Expected:    codeSymbol.Signature,
			Actual:      docSignature,
			Suggestion:  "Update documentation to match the current function signature",
			Context: InconsistencyContext{
				CodeSymbol: codeSymbol,
			},
		}
		return types.NewResult(inconsistency)
	}

	return types.NewResult((*Inconsistency)(nil))
}

// CheckDocumentationCoverage checks if all exported symbols are documented
func (dce *DefaultComparisonEngine) CheckDocumentationCoverage(symbols []CodeSymbol) types.Result[[]Inconsistency] {
	dce.logger.Debug("Checking documentation coverage", "symbols", len(symbols))

	var inconsistencies []Inconsistency

	for _, symbol := range symbols {
		if symbol.Exported && symbol.Comment == "" {
			severity := dce.determineSeverity(symbol)

			inconsistency := Inconsistency{
				ID:          fmt.Sprintf("missing-doc-%s", symbol.Name),
				Type:        InconsistencyMissingDoc,
				Severity:    severity,
				Title:       "Missing documentation",
				Description: fmt.Sprintf("Exported %s '%s' is missing documentation", symbol.Type, symbol.Name),
				CodeFile:    symbol.File,
				CodeLine:    symbol.Line,
				Suggestion:  fmt.Sprintf("Add documentation comment for %s '%s'", symbol.Type, symbol.Name),
				Context: InconsistencyContext{
					CodeSymbol: symbol,
				},
				Metadata: map[string]interface{}{
					"symbol_type": symbol.Type,
					"exported":    symbol.Exported,
				},
			}
			inconsistencies = append(inconsistencies, inconsistency)
		}
	}

	return types.NewResult(inconsistencies)
}

// ValidateExamples validates code examples in documentation
func (dce *DefaultComparisonEngine) ValidateExamples(examples []string, projectPath string) types.Result[[]Inconsistency] {
	dce.logger.Debug("Validating code examples", "examples", len(examples))

	var inconsistencies []Inconsistency

	for i, example := range examples {
		// Basic syntax validation
		if !dce.isValidGoCode(example) {
			inconsistency := Inconsistency{
				ID:          fmt.Sprintf("invalid-example-%d", i),
				Type:        InconsistencyIncorrectDoc,
				Severity:    SeverityMedium,
				Title:       "Invalid code example",
				Description: "Code example contains syntax errors or is not valid Go code",
				Actual:      example,
				Suggestion:  "Fix the syntax errors in the code example",
				Metadata: map[string]interface{}{
					"example_index": i,
				},
			}
			inconsistencies = append(inconsistencies, inconsistency)
		}
	}

	return types.NewResult(inconsistencies)
}

// Helper methods

func (dce *DefaultComparisonEngine) findMissingDocumentation(codeSymbols []CodeSymbol) []Inconsistency {
	var inconsistencies []Inconsistency

	for _, symbol := range codeSymbols {
		if symbol.Exported && strings.TrimSpace(symbol.Comment) == "" {
			severity := dce.determineSeverity(symbol)

			inconsistency := Inconsistency{
				ID:          fmt.Sprintf("missing-doc-%s", symbol.Name),
				Type:        InconsistencyMissingDoc,
				Severity:    severity,
				Title:       "Missing documentation",
				Description: fmt.Sprintf("Exported %s '%s' lacks documentation", symbol.Type, symbol.Name),
				CodeFile:    symbol.File,
				CodeLine:    symbol.Line,
				Suggestion:  dce.generateDocumentationSuggestion(symbol),
				Context: InconsistencyContext{
					CodeSymbol: symbol,
				},
				Metadata: map[string]interface{}{
					"symbol_type": symbol.Type,
					"exported":    symbol.Exported,
				},
			}
			inconsistencies = append(inconsistencies, inconsistency)
		}
	}

	return inconsistencies
}

func (dce *DefaultComparisonEngine) findOutdatedDocumentation(codeSymbols []CodeSymbol, docSections []DocSection) []Inconsistency {
	var inconsistencies []Inconsistency

	// Create a map of documented functions for quick lookup
	documentedFunctions := make(map[string]DocSection)
	for _, section := range docSections {
		// Extract function references from documentation
		refs, _ := dce.extractCodeReferences(section.Content)
		for _, ref := range refs {
			documentedFunctions[ref] = section
		}
	}

	for _, symbol := range codeSymbols {
		if docSection, exists := documentedFunctions[symbol.Name]; exists {
			// Check if documentation seems outdated
			if dce.isDocumentationOutdated(symbol, docSection) {
				inconsistency := Inconsistency{
					ID:          fmt.Sprintf("outdated-doc-%s", symbol.Name),
					Type:        InconsistencyOutdatedDoc,
					Severity:    SeverityMedium,
					Title:       "Outdated documentation",
					Description: fmt.Sprintf("Documentation for '%s' appears to be outdated", symbol.Name),
					CodeFile:    symbol.File,
					CodeLine:    symbol.Line,
					DocFile:     docSection.File,
					DocLine:     docSection.Line,
					Suggestion:  "Review and update the documentation to match current implementation",
					Context: InconsistencyContext{
						CodeSymbol: symbol,
						DocSection: docSection,
					},
				}
				inconsistencies = append(inconsistencies, inconsistency)
			}
		}
	}

	return inconsistencies
}

func (dce *DefaultComparisonEngine) findOrphanedDocumentation(codeSymbols []CodeSymbol, docSections []DocSection) []Inconsistency {
	var inconsistencies []Inconsistency

	// Create a set of existing code symbols for quick lookup
	codeSymbolNames := make(map[string]bool)
	for _, symbol := range codeSymbols {
		codeSymbolNames[symbol.Name] = true
		// Also add qualified names
		if symbol.Package != "" {
			codeSymbolNames[symbol.Package+"."+symbol.Name] = true
		}
	}

	for _, section := range docSections {
		// Extract code references from documentation
		refs, _ := dce.extractCodeReferences(section.Content)
		for _, ref := range refs {
			if !codeSymbolNames[ref] && dce.looksLikeCodeReference(ref) {
				inconsistency := Inconsistency{
					ID:          fmt.Sprintf("orphaned-doc-%s", ref),
					Type:        InconsistencyIncorrectDoc,
					Severity:    SeverityLow,
					Title:       "Orphaned documentation reference",
					Description: fmt.Sprintf("Documentation references '%s' which doesn't exist in code", ref),
					DocFile:     section.File,
					DocLine:     section.Line,
					Actual:      ref,
					Suggestion:  "Remove the reference or update it to match existing code",
					Context: InconsistencyContext{
						DocSection: section,
					},
				}
				inconsistencies = append(inconsistencies, inconsistency)
			}
		}
	}

	return inconsistencies
}

func (dce *DefaultComparisonEngine) determineSeverity(symbol CodeSymbol) SeverityLevel {
	switch symbol.Type {
	case "function":
		if strings.HasPrefix(symbol.Name, "Test") {
			return SeverityLow
		}
		return SeverityMedium
	case "type", "interface", "struct":
		return SeverityHigh
	case "constant":
		return SeverityLow
	case "variable":
		return SeverityLow
	default:
		return SeverityMedium
	}
}

func (dce *DefaultComparisonEngine) generateDocumentationSuggestion(symbol CodeSymbol) string {
	switch symbol.Type {
	case "function":
		return fmt.Sprintf("Add a comment starting with '%s' describing what the function does", symbol.Name)
	case "type":
		return fmt.Sprintf("Add a comment starting with '%s' describing the type's purpose", symbol.Name)
	case "constant":
		return fmt.Sprintf("Add a comment starting with '%s' explaining the constant's value and use", symbol.Name)
	case "variable":
		return fmt.Sprintf("Add a comment starting with '%s' describing the variable's purpose", symbol.Name)
	default:
		return fmt.Sprintf("Add documentation comment for %s '%s'", symbol.Type, symbol.Name)
	}
}

func (dce *DefaultComparisonEngine) extractSignatureFromDoc(docContent, symbolName string) string {
	// Look for code blocks or inline code that might contain the signature
	patterns := []string{
		fmt.Sprintf("```go[^`]*func[^`]*%s[^`]*```", regexp.QuoteMeta(symbolName)),
		fmt.Sprintf("`func[^`]*%s[^`]*`", regexp.QuoteMeta(symbolName)),
	}

	for _, pattern := range patterns {
		regex := regexp.MustCompile(pattern)
		if match := regex.FindString(docContent); match != "" {
			// Clean up the match
			cleaned := strings.Trim(match, "`")
			cleaned = strings.ReplaceAll(cleaned, "```go", "")
			cleaned = strings.ReplaceAll(cleaned, "```", "")
			cleaned = strings.TrimSpace(cleaned)
			return cleaned
		}
	}

	return ""
}

func (dce *DefaultComparisonEngine) signaturesMatch(codeSignature, docSignature string) bool {
	// Normalize signatures for comparison
	normalize := func(sig string) string {
		// Remove extra whitespace
		sig = regexp.MustCompile(`\s+`).ReplaceAllString(sig, " ")
		// Remove receiver names but keep types
		sig = regexp.MustCompile(`\(\w+\s+`).ReplaceAllString(sig, "(")
		return strings.TrimSpace(sig)
	}

	normalizedCode := normalize(codeSignature)
	normalizedDoc := normalize(docSignature)

	return normalizedCode == normalizedDoc
}

func (dce *DefaultComparisonEngine) isDocumentationOutdated(symbol CodeSymbol, docSection DocSection) bool {
	// Simple heuristics to detect outdated documentation
	content := strings.ToLower(docSection.Content)

	// Look for deprecated markers
	if strings.Contains(content, "deprecated") && !symbol.Deprecated {
		return true
	}

	// Look for version-specific information that might be outdated
	versionPatterns := []string{"version", "v1.", "v2.", "since"}
	for _, pattern := range versionPatterns {
		if strings.Contains(content, pattern) {
			// This is a simple heuristic - in practice, you'd need more sophisticated checks
			return false // Assume it might be outdated but don't flag without more evidence
		}
	}

	// Check if function signature in docs matches code
	if symbol.Type == "function" {
		docSig := dce.extractSignatureFromDoc(docSection.Content, symbol.Name)
		if docSig != "" && !dce.signaturesMatch(symbol.Signature, docSig) {
			return true
		}
	}

	return false
}

func (dce *DefaultComparisonEngine) extractCodeReferences(content string) ([]string, error) {
	var references []string

	patterns := []string{
		"`([a-zA-Z_][a-zA-Z0-9_]*)`",                  // Simple identifiers
		"`([A-Z][a-zA-Z0-9_]*)`",                      // Type names
		"`([a-z][a-zA-Z0-9_]*\\.[A-Z][a-zA-Z0-9_]*)`", // Qualified names
		"`([a-zA-Z_][a-zA-Z0-9_]*\\([^)]*\\))`",       // Function calls
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

	return references, nil
}

func (dce *DefaultComparisonEngine) looksLikeCodeReference(ref string) bool {
	// Heuristics to determine if a string looks like a code reference

	// Skip very short references
	if len(ref) < 3 {
		return false
	}

	// Skip common English words
	commonWords := []string{"the", "and", "for", "are", "but", "not", "you", "all", "can", "had", "her", "was", "one", "our", "out", "day", "get", "use", "man", "new", "now", "old", "see", "him", "two", "way", "who", "boy", "did", "its", "let", "put", "say", "she", "too", "use"}
	for _, word := range commonWords {
		if strings.EqualFold(ref, word) {
			return false
		}
	}

	// Look for programming patterns
	if strings.Contains(ref, ".") || strings.Contains(ref, "(") || strings.Contains(ref, "_") {
		return true
	}

	// Check if it starts with uppercase (likely a type)
	if len(ref) > 0 && ref[0] >= 'A' && ref[0] <= 'Z' {
		return true
	}

	// Check if it looks like a constant (ALL_CAPS)
	if strings.ToUpper(ref) == ref && strings.Contains(ref, "_") {
		return true
	}

	return false
}

func (dce *DefaultComparisonEngine) isValidGoCode(code string) bool {
	// Very basic Go code validation
	code = strings.TrimSpace(code)

	if code == "" {
		return false
	}

	// Check for basic Go syntax elements
	goKeywords := []string{"func", "type", "var", "const", "package", "import", "if", "for", "range", "return"}
	hasGoSyntax := false

	for _, keyword := range goKeywords {
		if strings.Contains(code, keyword) {
			hasGoSyntax = true
			break
		}
	}

	// If it has Go keywords, do basic bracket matching
	if hasGoSyntax {
		return dce.hasBalancedBrackets(code)
	}

	// For simple expressions, check basic syntax
	return !strings.ContainsAny(code, "[]{}") || dce.hasBalancedBrackets(code)
}

func (dce *DefaultComparisonEngine) hasBalancedBrackets(code string) bool {
	brackets := map[rune]rune{
		'(': ')',
		'[': ']',
		'{': '}',
	}

	stack := []rune{}

	for _, char := range code {
		if _, isOpening := brackets[char]; isOpening {
			stack = append(stack, char)
		} else {
			for opening, closing := range brackets {
				if char == closing {
					if len(stack) == 0 || stack[len(stack)-1] != opening {
						return false
					}
					stack = stack[:len(stack)-1]
					break
				}
			}
		}
	}

	return len(stack) == 0
}
