package analyzer

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/phillipgreenii/mobilecombackup/pkg/types"
)

const symbolTypeFunction = "function"

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
func (dce *DefaultComparisonEngine) DetectInconsistencies(codeSymbols []CodeSymbol, docSections []DocSection) types.Result[[]Inconsistency] { //nolint:lll
	dce.logger.Debug("Detecting inconsistencies", "code_symbols", len(codeSymbols), "doc_sections", len(docSections))

	var inconsistencies []Inconsistency

	// Phase 1: Basic inconsistencies (already implemented)
	// Check for missing documentation
	missingDocs := dce.findMissingDocumentation(codeSymbols)
	inconsistencies = append(inconsistencies, missingDocs...)

	// Check for outdated documentation
	outdatedDocs := dce.findOutdatedDocumentation(codeSymbols, docSections)
	inconsistencies = append(inconsistencies, outdatedDocs...)

	// Check for documentation without corresponding code
	orphanedDocs := dce.findOrphanedDocumentation(codeSymbols, docSections)
	inconsistencies = append(inconsistencies, orphanedDocs...)

	// Phase 2: Advanced inconsistencies (newly implemented)
	// Check for missing code examples
	missingExamples := dce.detectMissingExamples(codeSymbols, docSections)
	inconsistencies = append(inconsistencies, missingExamples...)

	// Check for broken documentation links
	brokenLinks := dce.detectBrokenLinks(docSections, codeSymbols)
	inconsistencies = append(inconsistencies, brokenLinks...)

	// Check for parameter documentation mismatches
	paramMismatches := dce.detectParameterMismatches(codeSymbols, docSections)
	inconsistencies = append(inconsistencies, paramMismatches...)

	// Check for return value documentation mismatches
	returnMismatches := dce.detectReturnValueMismatches(codeSymbols, docSections)
	inconsistencies = append(inconsistencies, returnMismatches...)

	// Check for deprecated code with active documentation
	deprecatedActive := dce.detectDeprecatedCodeWithActiveDocumentation(codeSymbols, docSections)
	inconsistencies = append(inconsistencies, deprecatedActive...)

	// Check for new features without documentation
	newFeaturesUndocumented := dce.detectNewFeaturesWithoutDocumentation(codeSymbols)
	inconsistencies = append(inconsistencies, newFeaturesUndocumented...)

	dce.logger.Info("Inconsistency detection completed",
		"total_inconsistencies", len(inconsistencies),
		"missing_docs", len(missingDocs),
		"outdated_docs", len(outdatedDocs),
		"orphaned_docs", len(orphanedDocs),
		"missing_examples", len(missingExamples),
		"broken_links", len(brokenLinks),
		"param_mismatches", len(paramMismatches),
		"return_mismatches", len(returnMismatches),
		"deprecated_active", len(deprecatedActive),
		"new_undocumented", len(newFeaturesUndocumented))

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

func (dce *DefaultComparisonEngine) findOutdatedDocumentation(codeSymbols []CodeSymbol, docSections []DocSection) []Inconsistency { //nolint:lll
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

func (dce *DefaultComparisonEngine) findOrphanedDocumentation(codeSymbols []CodeSymbol, docSections []DocSection) []Inconsistency { //nolint:lll
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
	case symbolTypeFunction:
		if strings.HasPrefix(symbol.Name, "Test") {
			return SeverityLow
		}
		return SeverityMedium
	case symbolTypeType, "interface", "struct":
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
	case symbolTypeFunction:
		return fmt.Sprintf("Add a comment starting with '%s' describing what the function does", symbol.Name)
	case symbolTypeType:
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
	if symbol.Type == symbolTypeFunction {
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
	commonWords := []string{"the", "and", "for", "are", "but", "not", "you", "all", "can", "had", "her", "was", "one", "our", "out", "day", "get", "use", "man", "new", "now", "old", "see", "him", "two", "way", "who", "boy", "did", "its", "let", "put", "say", "she", "too", "use"} //nolint:lll
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

// detectMissingExamples identifies functions/types that should have code examples but don't
func (dce *DefaultComparisonEngine) detectMissingExamples(codeSymbols []CodeSymbol, docSections []DocSection) []Inconsistency {
	var inconsistencies []Inconsistency

	// Focus on exported functions and types that would benefit from examples
	for _, symbol := range codeSymbols {
		if !symbol.Exported || (symbol.Type != symbolTypeFunction && symbol.Type != symbolTypeMethod && symbol.Type != symbolTypeType) {
			continue
		}

		// Check if symbol should have an example (complex functions, public APIs)
		if dce.shouldHaveExample(symbol) && !dce.hasCodeExample(symbol, docSections) {
			inconsistency := Inconsistency{
				ID:          fmt.Sprintf("missing-example-%s", symbol.Name),
				Type:        InconsistencyMissingExample,
				Severity:    SeverityMedium,
				Title:       "Missing code example",
				Description: fmt.Sprintf("Exported %s '%s' should include a code example", symbol.Type, symbol.Name),
				CodeFile:    symbol.File,
				CodeLine:    symbol.Line,
				Suggestion:  fmt.Sprintf("Add a code example demonstrating how to use %s '%s'", symbol.Type, symbol.Name),
				Context: InconsistencyContext{
					CodeSymbol: symbol,
				},
				Metadata: map[string]interface{}{
					"symbol_type":    symbol.Type,
					"complexity":     dce.assessSymbolComplexity(symbol),
					"example_needed": true,
				},
			}
			inconsistencies = append(inconsistencies, inconsistency)
		}
	}

	return inconsistencies
}

// detectBrokenLinks identifies broken internal documentation links
func (dce *DefaultComparisonEngine) detectBrokenLinks(docSections []DocSection, codeSymbols []CodeSymbol) []Inconsistency {
	var inconsistencies []Inconsistency

	// Create a map of valid code symbols for quick lookup
	validSymbols := make(map[string]bool)
	for _, symbol := range codeSymbols {
		validSymbols[symbol.Name] = true
		if symbol.Package != "" {
			validSymbols[symbol.Package+"."+symbol.Name] = true
		}
	}

	// Check each documentation section for broken links
	for _, section := range docSections {
		brokenLinks := dce.findBrokenLinksInContent(section.Content, validSymbols)
		for _, link := range brokenLinks {
			inconsistency := Inconsistency{
				ID:          fmt.Sprintf("broken-link-%s-%d", filepath.Base(section.File), section.Line),
				Type:        InconsistencyBrokenLink,
				Severity:    SeverityMedium,
				Title:       "Broken documentation link",
				Description: fmt.Sprintf("Documentation contains broken link to '%s'", link),
				DocFile:     section.File,
				DocLine:     section.Line,
				Actual:      link,
				Suggestion:  "Update or remove the broken link reference",
				Context: InconsistencyContext{
					DocSection: section,
				},
				Metadata: map[string]interface{}{
					"broken_link": link,
					"section":     section.Title,
				},
			}
			inconsistencies = append(inconsistencies, inconsistency)
		}
	}

	return inconsistencies
}

// detectParameterMismatches identifies parameter documentation mismatches
func (dce *DefaultComparisonEngine) detectParameterMismatches(codeSymbols []CodeSymbol, docSections []DocSection) []Inconsistency { //nolint:funlen,lll
	var inconsistencies []Inconsistency

	for _, symbol := range codeSymbols {
		if symbol.Type != symbolTypeFunction && symbol.Type != symbolTypeMethod {
			continue
		}

		// Find documentation for this symbol
		docContent := dce.findDocumentationForSymbol(symbol, docSections)
		if docContent == "" {
			continue
		}

		// Extract parameters from code signature and documentation
		codeParams := dce.extractParametersFromSignature(symbol.Signature)
		docParams := dce.extractParametersFromDoc(docContent)

		// Compare parameters
		for _, codeParam := range codeParams {
			if docParam, exists := docParams[codeParam.Name]; !exists {
				// Missing parameter documentation
				inconsistency := Inconsistency{
					ID:          fmt.Sprintf("missing-param-doc-%s-%s", symbol.Name, codeParam.Name),
					Type:        InconsistencyParamMismatch,
					Severity:    SeverityMedium,
					Title:       "Missing parameter documentation",
					Description: fmt.Sprintf("Parameter '%s' in function '%s' is not documented", codeParam.Name, symbol.Name),
					CodeFile:    symbol.File,
					CodeLine:    symbol.Line,
					Expected:    fmt.Sprintf("Documentation for parameter '%s %s'", codeParam.Name, codeParam.Type),
					Suggestion:  fmt.Sprintf("Add documentation for parameter '%s'", codeParam.Name),
					Context: InconsistencyContext{
						CodeSymbol: symbol,
					},
					Metadata: map[string]interface{}{
						"parameter_name": codeParam.Name,
						"parameter_type": codeParam.Type,
					},
				}
				inconsistencies = append(inconsistencies, inconsistency)
			} else if docParam.Type != "" && codeParam.Type != docParam.Type {
				// Parameter type mismatch
				inconsistency := Inconsistency{
					ID:          fmt.Sprintf("param-type-mismatch-%s-%s", symbol.Name, codeParam.Name),
					Type:        InconsistencyParamMismatch,
					Severity:    SeverityHigh,
					Title:       "Parameter type mismatch",
					Description: fmt.Sprintf("Parameter '%s' type mismatch in function '%s'", codeParam.Name, symbol.Name),
					CodeFile:    symbol.File,
					CodeLine:    symbol.Line,
					Expected:    codeParam.Type,
					Actual:      docParam.Type,
					Suggestion:  "Update parameter type in documentation to match code",
					Context: InconsistencyContext{
						CodeSymbol: symbol,
					},
					Metadata: map[string]interface{}{
						"parameter_name": codeParam.Name,
						"code_type":      codeParam.Type,
						"doc_type":       docParam.Type,
					},
				}
				inconsistencies = append(inconsistencies, inconsistency)
			}
		}

		// Check for documented parameters that don't exist in code
		for paramName, docParam := range docParams {
			found := false
			for _, codeParam := range codeParams {
				if codeParam.Name == paramName {
					found = true
					break
				}
			}
			if !found {
				inconsistency := Inconsistency{
					ID:          fmt.Sprintf("extra-param-doc-%s-%s", symbol.Name, paramName),
					Type:        InconsistencyParamMismatch,
					Severity:    SeverityMedium,
					Title:       "Extra parameter documentation",
					Description: fmt.Sprintf("Documentation for parameter '%s' exists but parameter not found in function '%s'", paramName, symbol.Name), //nolint:lll
					CodeFile:    symbol.File,
					CodeLine:    symbol.Line,
					Actual:      fmt.Sprintf("Documentation for non-existent parameter '%s'", paramName),
					Suggestion:  "Remove documentation for non-existent parameter",
					Context: InconsistencyContext{
						CodeSymbol: symbol,
					},
					Metadata: map[string]interface{}{
						"parameter_name": paramName,
						"doc_type":       docParam.Type,
					},
				}
				inconsistencies = append(inconsistencies, inconsistency)
			}
		}
	}

	return inconsistencies
}

// detectReturnValueMismatches identifies return value documentation mismatches
func (dce *DefaultComparisonEngine) detectReturnValueMismatches(codeSymbols []CodeSymbol, docSections []DocSection) []Inconsistency { //nolint:funlen,lll
	var inconsistencies []Inconsistency

	for _, symbol := range codeSymbols {
		if symbol.Type != symbolTypeFunction && symbol.Type != symbolTypeMethod {
			continue
		}

		// Find documentation for this symbol
		docContent := dce.findDocumentationForSymbol(symbol, docSections)
		if docContent == "" {
			continue
		}

		// Extract return values from signature and documentation
		codeReturns := dce.extractReturnValuesFromSignature(symbol.Signature)
		docReturns := dce.extractReturnValuesFromDoc(docContent)

		// Check if function has return values
		if len(codeReturns) > 0 {
			if len(docReturns) == 0 {
				// Missing return value documentation
				inconsistency := Inconsistency{
					ID:          fmt.Sprintf("missing-return-doc-%s", symbol.Name),
					Type:        InconsistencyReturnMismatch,
					Severity:    SeverityMedium,
					Title:       "Missing return value documentation",
					Description: fmt.Sprintf("Function '%s' returns values but has no return documentation", symbol.Name),
					CodeFile:    symbol.File,
					CodeLine:    symbol.Line,
					Expected:    fmt.Sprintf("Documentation for return values: %s", strings.Join(codeReturns, ", ")),
					Suggestion:  "Add documentation describing the return values",
					Context: InconsistencyContext{
						CodeSymbol: symbol,
					},
					Metadata: map[string]interface{}{
						"return_types": codeReturns,
					},
				}
				inconsistencies = append(inconsistencies, inconsistency)
			} else if len(codeReturns) != len(docReturns) {
				// Return value count mismatch
				inconsistency := Inconsistency{
					ID:          fmt.Sprintf("return-count-mismatch-%s", symbol.Name),
					Type:        InconsistencyReturnMismatch,
					Severity:    SeverityHigh,
					Title:       "Return value count mismatch",
					Description: fmt.Sprintf("Function '%s' return value count mismatch", symbol.Name),
					CodeFile:    symbol.File,
					CodeLine:    symbol.Line,
					Expected:    fmt.Sprintf("%d return values", len(codeReturns)),
					Actual:      fmt.Sprintf("%d documented return values", len(docReturns)),
					Suggestion:  "Update return value documentation to match function signature",
					Context: InconsistencyContext{
						CodeSymbol: symbol,
					},
					Metadata: map[string]interface{}{
						"code_return_count": len(codeReturns),
						"doc_return_count":  len(docReturns),
						"code_returns":      codeReturns,
						"doc_returns":       docReturns,
					},
				}
				inconsistencies = append(inconsistencies, inconsistency)
			}
		} else if len(docReturns) > 0 {
			// Function doesn't return values but documentation claims it does
			inconsistency := Inconsistency{
				ID:          fmt.Sprintf("extra-return-doc-%s", symbol.Name),
				Type:        InconsistencyReturnMismatch,
				Severity:    SeverityMedium,
				Title:       "Extra return value documentation",
				Description: fmt.Sprintf("Function '%s' doesn't return values but has return documentation", symbol.Name),
				CodeFile:    symbol.File,
				CodeLine:    symbol.Line,
				Actual:      "Documentation describes return values for void function",
				Suggestion:  "Remove return value documentation for void function",
				Context: InconsistencyContext{
					CodeSymbol: symbol,
				},
				Metadata: map[string]interface{}{
					"doc_returns": docReturns,
				},
			}
			inconsistencies = append(inconsistencies, inconsistency)
		}
	}

	return inconsistencies
}

// detectDeprecatedCodeWithActiveDocumentation identifies deprecated code that still has active documentation
func (dce *DefaultComparisonEngine) detectDeprecatedCodeWithActiveDocumentation(codeSymbols []CodeSymbol, docSections []DocSection) []Inconsistency { //nolint:lll
	var inconsistencies []Inconsistency

	for _, symbol := range codeSymbols {
		if !symbol.Deprecated {
			continue
		}

		// Check if deprecated symbol has active (non-deprecated) documentation
		docContent := dce.findDocumentationForSymbol(symbol, docSections)
		if docContent != "" && !dce.isDocumentationDeprecated(docContent) {
			inconsistency := Inconsistency{
				ID:          fmt.Sprintf("deprecated-active-doc-%s", symbol.Name),
				Type:        InconsistencyDeprecated,
				Severity:    SeverityHigh,
				Title:       "Deprecated code with active documentation",
				Description: fmt.Sprintf("Deprecated %s '%s' still has active documentation", symbol.Type, symbol.Name),
				CodeFile:    symbol.File,
				CodeLine:    symbol.Line,
				Suggestion:  "Add deprecation notice to documentation or remove if fully deprecated",
				Context: InconsistencyContext{
					CodeSymbol: symbol,
				},
				Metadata: map[string]interface{}{
					"symbol_type": symbol.Type,
					"deprecated":  true,
				},
			}
			inconsistencies = append(inconsistencies, inconsistency)
		}
	}

	return inconsistencies
}

// detectNewFeaturesWithoutDocumentation identifies new code features that lack documentation
func (dce *DefaultComparisonEngine) detectNewFeaturesWithoutDocumentation(codeSymbols []CodeSymbol) []Inconsistency {
	var inconsistencies []Inconsistency

	for _, symbol := range codeSymbols {
		if !symbol.Exported {
			continue
		}

		// Check if this is a "new" feature (recently added, lacks documentation)
		if dce.isNewFeature(symbol) && strings.TrimSpace(symbol.Comment) == "" {
			severity := dce.determineSeverity(symbol)
			// Increase severity for new features as they're likely being actively used
			if severity == SeverityLow {
				severity = SeverityMedium
			}

			inconsistency := Inconsistency{
				ID:          fmt.Sprintf("new-feature-no-doc-%s", symbol.Name),
				Type:        InconsistencyNewFeature,
				Severity:    severity,
				Title:       "New feature without documentation",
				Description: fmt.Sprintf("New %s '%s' lacks documentation", symbol.Type, symbol.Name),
				CodeFile:    symbol.File,
				CodeLine:    symbol.Line,
				Suggestion:  fmt.Sprintf("Add comprehensive documentation for new %s '%s'", symbol.Type, symbol.Name),
				Context: InconsistencyContext{
					CodeSymbol: symbol,
				},
				Metadata: map[string]interface{}{
					"symbol_type":  symbol.Type,
					"new_feature":  true,
					"exported":     symbol.Exported,
					"needs_urgent": true,
				},
			}
			inconsistencies = append(inconsistencies, inconsistency)
		}
	}

	return inconsistencies
}

// Helper methods for advanced detection

func (dce *DefaultComparisonEngine) shouldHaveExample(symbol CodeSymbol) bool {
	// Public API functions, complex types, and constructors should have examples
	if symbol.Type == symbolTypeFunction || symbol.Type == symbolTypeMethod {
		// Check if function name suggests it's a constructor or important API
		if strings.HasPrefix(symbol.Name, "New") ||
			strings.Contains(symbol.Signature, "error") ||
			strings.Contains(strings.ToLower(symbol.Name), "create") ||
			strings.Contains(strings.ToLower(symbol.Name), "parse") {
			return true
		}
	}

	if symbol.Type == symbolTypeType || symbol.Type == "interface" {
		// Public types and interfaces should have examples
		return true
	}

	return false
}

func (dce *DefaultComparisonEngine) hasCodeExample(symbol CodeSymbol, docSections []DocSection) bool {
	symbolName := symbol.Name

	for _, section := range docSections {
		// Look for code blocks that mention the symbol
		if strings.Contains(section.Content, symbolName) &&
			(strings.Contains(section.Content, "```go") ||
				strings.Contains(section.Content, "```")) {
			return true
		}
	}

	// Also check for example files (example_test.go)
	return strings.Contains(symbol.File, "example") ||
		strings.Contains(symbol.Comment, "Example")
}

func (dce *DefaultComparisonEngine) assessSymbolComplexity(symbol CodeSymbol) string {
	// Simple complexity assessment based on signature
	sig := symbol.Signature

	if strings.Count(sig, "(") > 2 || strings.Count(sig, ",") > 3 {
		return "high"
	} else if strings.Contains(sig, "error") || strings.Count(sig, ",") > 1 {
		return "medium"
	}

	return "low"
}

func (dce *DefaultComparisonEngine) findBrokenLinksInContent(content string, validSymbols map[string]bool) []string {
	var brokenLinks []string

	// Look for markdown links [text](link) and inline code references `symbol`
	linkPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)|` + "`" + `([^` + "`" + `]+)` + "`")
	matches := linkPattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		var candidate string
		if match[2] != "" {
			// Markdown link
			candidate = match[2]
		} else if match[3] != "" {
			// Inline code
			candidate = match[3]
		}

		// Check if it looks like a code reference and if it's valid
		if dce.looksLikeCodeReference(candidate) && !validSymbols[candidate] {
			brokenLinks = append(brokenLinks, candidate)
		}
	}

	return brokenLinks
}

type Parameter struct {
	Name string
	Type string
}

func (dce *DefaultComparisonEngine) extractParametersFromSignature(signature string) []Parameter {
	var params []Parameter

	// Simple parameter extraction from Go function signature
	// This is a simplified implementation - a full parser would be more robust
	paramPattern := regexp.MustCompile(`\(([^)]*)\)`)
	matches := paramPattern.FindStringSubmatch(signature)

	if len(matches) > 1 {
		paramStr := matches[1]
		// Split by comma and parse each parameter
		for _, param := range strings.Split(paramStr, ",") {
			param = strings.TrimSpace(param)
			if param == "" {
				continue
			}

			// Extract name and type (simplified)
			parts := strings.Fields(param)
			if len(parts) >= 2 {
				params = append(params, Parameter{
					Name: parts[0],
					Type: strings.Join(parts[1:], " "),
				})
			}
		}
	}

	return params
}

func (dce *DefaultComparisonEngine) extractParametersFromDoc(docContent string) map[string]Parameter {
	params := make(map[string]Parameter)

	// Look for parameter documentation patterns like "- param: description"
	paramPattern := regexp.MustCompile(`(?m)^[-*]\s+(\w+)\s*(\w+)?:\s*(.+)`)
	matches := paramPattern.FindAllStringSubmatch(docContent, -1)

	for _, match := range matches {
		paramName := match[1]
		paramType := match[2]

		params[paramName] = Parameter{
			Name: paramName,
			Type: paramType,
		}
	}

	return params
}

func (dce *DefaultComparisonEngine) extractReturnValuesFromSignature(signature string) []string {
	var returns []string

	// Extract return types from function signature
	// Look for return values after the last )
	lastParen := strings.LastIndex(signature, ")")
	if lastParen != -1 && lastParen < len(signature)-1 {
		returnPart := strings.TrimSpace(signature[lastParen+1:])
		if returnPart != "" {
			// Handle multiple return values (type1, type2) or single type
			if strings.HasPrefix(returnPart, "(") && strings.HasSuffix(returnPart, ")") {
				returnPart = returnPart[1 : len(returnPart)-1]
			}

			for _, ret := range strings.Split(returnPart, ",") {
				returns = append(returns, strings.TrimSpace(ret))
			}
		}
	}

	return returns
}

func (dce *DefaultComparisonEngine) extractReturnValuesFromDoc(docContent string) []string {
	var returns []string

	// Look for return value documentation
	returnPattern := regexp.MustCompile(`(?i)returns?\s*:?\s*(.+)`)
	matches := returnPattern.FindAllStringSubmatch(docContent, -1)

	for _, match := range matches {
		if len(match) > 1 {
			returns = append(returns, strings.TrimSpace(match[1]))
		}
	}

	return returns
}

func (dce *DefaultComparisonEngine) findDocumentationForSymbol(symbol CodeSymbol, docSections []DocSection) string {
	for _, section := range docSections {
		if strings.Contains(section.Content, symbol.Name) {
			return section.Content
		}
	}
	return ""
}

func (dce *DefaultComparisonEngine) isDocumentationDeprecated(docContent string) bool {
	deprecatedKeywords := []string{"deprecated", "deprecate", "obsolete", "legacy"}
	lowerContent := strings.ToLower(docContent)

	for _, keyword := range deprecatedKeywords {
		if strings.Contains(lowerContent, keyword) {
			return true
		}
	}

	return false
}

func (dce *DefaultComparisonEngine) isNewFeature(symbol CodeSymbol) bool {
	// Heuristics to determine if this is a "new" feature
	// This could be enhanced with git history, version tags, etc.

	// Check metadata for version information
	if metadata := symbol.Metadata; metadata != nil {
		if version, ok := metadata["version"]; ok {
			// If versioned, check if it's a recent version
			if versionStr, isStr := version.(string); isStr && versionStr != "" {
				// Simple check for recent versions (this could be more sophisticated)
				return strings.Contains(versionStr, "beta") ||
					strings.Contains(versionStr, "alpha") ||
					strings.Contains(versionStr, "rc")
			}
		}
	}

	// If no version info, assume exported symbols without comments are potentially new
	return symbol.Exported && strings.TrimSpace(symbol.Comment) == ""
}

// TEMP: Force filepath import - will be removed after auto-formatting
var _ = filepath.Base
