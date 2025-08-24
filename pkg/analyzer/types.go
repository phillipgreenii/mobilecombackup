// Package analyzer provides documentation analysis and inconsistency detection capabilities
// for the FEAT-084 automated documentation synchronization system.
package analyzer

import (
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/types"
)

// AnalysisResult represents the result of a documentation analysis
type AnalysisResult struct {
	ID              string                 `json:"id"`
	Timestamp       int64                  `json:"timestamp"`
	ProjectPath     string                 `json:"project_path"`
	TotalFiles      int                    `json:"total_files"`
	AnalyzedFiles   int                    `json:"analyzed_files"`
	Inconsistencies []Inconsistency        `json:"inconsistencies"`
	QualityScore    float64                `json:"quality_score"`
	Coverage        DocumentationCoverage  `json:"coverage"`
	Summary         AnalysisSummary        `json:"summary"`
	Metadata        map[string]interface{} `json:"metadata"`
	Duration        time.Duration          `json:"duration"`
}

// Inconsistency represents a detected inconsistency between code and documentation
type Inconsistency struct {
	ID          string                 `json:"id"`
	Type        InconsistencyType      `json:"type"`
	Severity    SeverityLevel          `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	CodeFile    string                 `json:"code_file"`
	CodeLine    int                    `json:"code_line"`
	DocFile     string                 `json:"doc_file"`
	DocLine     int                    `json:"doc_line"`
	Expected    string                 `json:"expected"`
	Actual      string                 `json:"actual"`
	Suggestion  string                 `json:"suggestion"`
	Context     InconsistencyContext   `json:"context"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// InconsistencyType defines the type of inconsistency detected
type InconsistencyType string

const (
	InconsistencyMissingDoc     InconsistencyType = "missing_documentation"
	InconsistencyOutdatedDoc    InconsistencyType = "outdated_documentation"
	InconsistencyIncorrectDoc   InconsistencyType = "incorrect_documentation"
	InconsistencyMissingExample InconsistencyType = "missing_example"
	InconsistencyBrokenLink     InconsistencyType = "broken_link"
	InconsistencyTypeMismatch   InconsistencyType = "type_mismatch"
	InconsistencyParamMismatch  InconsistencyType = "parameter_mismatch"
	InconsistencyReturnMismatch InconsistencyType = "return_mismatch"
	InconsistencyDeprecated     InconsistencyType = "deprecated_feature"
	InconsistencyNewFeature     InconsistencyType = "undocumented_feature"
)

// SeverityLevel defines the severity of an inconsistency
type SeverityLevel string

const (
	SeverityLow      SeverityLevel = "low"
	SeverityMedium   SeverityLevel = "medium"
	SeverityHigh     SeverityLevel = "high"
	SeverityCritical SeverityLevel = "critical"
)

// InconsistencyContext provides additional context for an inconsistency
type InconsistencyContext struct {
	CodeSymbol     CodeSymbol   `json:"code_symbol"`
	DocSection     DocSection   `json:"doc_section"`
	RelatedSymbols []CodeSymbol `json:"related_symbols"`
	Dependencies   []string     `json:"dependencies"`
	AffectedFiles  []string     `json:"affected_files"`
}

// CodeSymbol represents a symbol in the codebase
type CodeSymbol struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Package    string                 `json:"package"`
	File       string                 `json:"file"`
	Line       int                    `json:"line"`
	Signature  string                 `json:"signature"`
	Comment    string                 `json:"comment"`
	Exported   bool                   `json:"exported"`
	Deprecated bool                   `json:"deprecated"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// DocSection represents a section in documentation
type DocSection struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Level   int    `json:"level"`
	Anchor  string `json:"anchor"`
	Type    string `json:"type"`
}

// DocumentationCoverage represents coverage metrics for documentation
type DocumentationCoverage struct {
	TotalSymbols      int                       `json:"total_symbols"`
	DocumentedSymbols int                       `json:"documented_symbols"`
	CoveragePercent   float64                   `json:"coverage_percent"`
	ByType            map[string]CoverageByType `json:"by_type"`
	ByPackage         map[string]CoverageByType `json:"by_package"`
}

// CoverageByType represents coverage statistics for a specific type
type CoverageByType struct {
	Total      int     `json:"total"`
	Documented int     `json:"documented"`
	Percent    float64 `json:"percent"`
}

// AnalysisSummary provides a high-level summary of the analysis
type AnalysisSummary struct {
	TotalInconsistencies int                       `json:"total_inconsistencies"`
	BySeverity           map[SeverityLevel]int     `json:"by_severity"`
	ByType               map[InconsistencyType]int `json:"by_type"`
	Recommendations      []Recommendation          `json:"recommendations"`
	QualityGrade         string                    `json:"quality_grade"`
	ImprovementAreas     []string                  `json:"improvement_areas"`
}

// Recommendation represents a suggestion for improving documentation
type Recommendation struct {
	ID              string   `json:"id"`
	Priority        string   `json:"priority"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Action          string   `json:"action"`
	Files           []string `json:"files"`
	EstimatedEffort string   `json:"estimated_effort"`
	Impact          string   `json:"impact"`
}

// AnalysisConfig defines configuration for analysis operations
type AnalysisConfig struct {
	ProjectRoot     string                 `yaml:"project_root"`
	IncludePatterns []string               `yaml:"include_patterns"`
	ExcludePatterns []string               `yaml:"exclude_patterns"`
	DocPatterns     []string               `yaml:"doc_patterns"`
	CodePatterns    []string               `yaml:"code_patterns"`
	MinSeverity     SeverityLevel          `yaml:"min_severity"`
	EnableTypes     []InconsistencyType    `yaml:"enable_types"`
	MaxDepth        int                    `yaml:"max_depth"`
	Parallel        bool                   `yaml:"parallel"`
	CacheEnabled    bool                   `yaml:"cache_enabled"`
	CacheTTL        time.Duration          `yaml:"cache_ttl"`
	OutputFormat    string                 `yaml:"output_format"`
	Verbose         bool                   `yaml:"verbose"`
	Options         map[string]interface{} `yaml:"options"`
}

// AnalysisReport contains the formatted report of analysis results
type AnalysisReport struct {
	Format   string                 `json:"format"`
	Content  string                 `json:"content"`
	Summary  string                 `json:"summary"`
	Metadata map[string]interface{} `json:"metadata"`
}

// Analyzer defines the interface for documentation analysis
type Analyzer interface {
	// AnalyzeProject performs comprehensive analysis of a project
	AnalyzeProject(config AnalysisConfig) types.Result[*AnalysisResult]

	// AnalyzeIncremental performs incremental analysis of changed files
	AnalyzeIncremental(config AnalysisConfig, changedFiles []string) types.Result[*AnalysisResult]

	// CompareCodeAndDocs compares specific code and documentation files
	CompareCodeAndDocs(codeFile, docFile string) types.Result[[]Inconsistency]

	// GenerateReport creates a formatted report from analysis results
	GenerateReport(result *AnalysisResult, format string) types.Result[*AnalysisReport]

	// ValidateDocumentation validates documentation structure and content
	ValidateDocumentation(docFiles []string) types.Result[[]Inconsistency]

	// GetCoverageMetrics calculates documentation coverage metrics
	GetCoverageMetrics(projectPath string) types.Result[*DocumentationCoverage]
}

// CodeAnalyzer defines the interface for code analysis operations
type CodeAnalyzer interface {
	// AnalyzePackage analyzes a Go package and extracts symbols
	AnalyzePackage(packagePath string) types.Result[[]CodeSymbol]

	// ExtractSymbols extracts symbols from a Go source file
	ExtractSymbols(filePath string) types.Result[[]CodeSymbol]

	// GetPackageDocs extracts package-level documentation
	GetPackageDocs(packagePath string) types.Result[string]

	// FindReferences finds references to a symbol across the codebase
	FindReferences(symbol CodeSymbol) types.Result[[]CodeSymbol]
}

// DocAnalyzer defines the interface for documentation analysis operations
type DocAnalyzer interface {
	// ParseMarkdown parses markdown files and extracts sections
	ParseMarkdown(filePath string) types.Result[[]DocSection]

	// ExtractCodeReferences finds code references in documentation
	ExtractCodeReferences(content string) types.Result[[]string]

	// ValidateLinks validates all links in documentation
	ValidateLinks(docFiles []string) types.Result[[]Inconsistency]

	// ExtractExamples extracts code examples from documentation
	ExtractExamples(content string) types.Result[[]string]
}

// ComparisonEngine defines the interface for comparing code and documentation
type ComparisonEngine interface {
	// DetectInconsistencies finds inconsistencies between code and documentation
	DetectInconsistencies(codeSymbols []CodeSymbol, docSections []DocSection) types.Result[[]Inconsistency]

	// CompareSignatures compares function signatures between code and docs
	CompareSignatures(codeSymbol CodeSymbol, docContent string) types.Result[*Inconsistency]

	// CheckDocumentationCoverage checks if all exported symbols are documented
	CheckDocumentationCoverage(symbols []CodeSymbol) types.Result[[]Inconsistency]

	// ValidateExamples validates code examples in documentation
	ValidateExamples(examples []string, projectPath string) types.Result[[]Inconsistency]
}

// ReportGenerator defines the interface for report generation
type ReportGenerator interface {
	// GenerateTextReport generates a human-readable text report
	GenerateTextReport(result *AnalysisResult) types.Result[string]

	// GenerateJSONReport generates a JSON report
	GenerateJSONReport(result *AnalysisResult) types.Result[string]

	// GenerateMarkdownReport generates a markdown report
	GenerateMarkdownReport(result *AnalysisResult) types.Result[string]

	// GenerateHTMLReport generates an HTML report
	GenerateHTMLReport(result *AnalysisResult) types.Result[string]

	// GenerateSummary generates a concise summary of the analysis
	GenerateSummary(result *AnalysisResult) types.Result[string]
}

// DefaultAnalysisConfig returns the default configuration for analysis
func DefaultAnalysisConfig() AnalysisConfig {
	return AnalysisConfig{
		IncludePatterns: []string{"**/*.go", "**/*.md"},
		ExcludePatterns: []string{"vendor/**", "node_modules/**", ".git/**", "**/testdata/**"},
		DocPatterns:     []string{"**/*.md", "**/doc.go"},
		CodePatterns:    []string{"**/*.go"},
		MinSeverity:     SeverityLow,
		EnableTypes: []InconsistencyType{
			InconsistencyMissingDoc,
			InconsistencyOutdatedDoc,
			InconsistencyIncorrectDoc,
			InconsistencyMissingExample,
			InconsistencyBrokenLink,
			InconsistencyTypeMismatch,
			InconsistencyParamMismatch,
			InconsistencyReturnMismatch,
			InconsistencyDeprecated,
			InconsistencyNewFeature,
		},
		MaxDepth:     10,
		Parallel:     true,
		CacheEnabled: true,
		CacheTTL:     24 * time.Hour,
		OutputFormat: "text",
		Verbose:      false,
		Options:      make(map[string]interface{}),
	}
}
