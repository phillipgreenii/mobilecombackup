package analyzer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/types"
)

// DocumentationAnalyzer implements the Analyzer interface for comprehensive documentation analysis
type DocumentationAnalyzer struct {
	codeAnalyzer     *GoCodeAnalyzer
	docAnalyzer      *MarkdownAnalyzer
	comparisonEngine *DefaultComparisonEngine
	reportGenerator  *DefaultReportGenerator
	cache            map[string]*AnalysisResult
	cacheMutex       sync.RWMutex
	logger           Logger
	auditLogger      AuditLogger
}

// Logger interface for analyzer logging
type Logger interface {
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
}

// AuditLogger interface for audit logging
type AuditLogger interface {
	LogEvent(event types.AuditEvent)
}

// NewDocumentationAnalyzer creates a new documentation analyzer
func NewDocumentationAnalyzer(logger Logger, auditLogger AuditLogger) *DocumentationAnalyzer {
	analyzer := &DocumentationAnalyzer{
		codeAnalyzer:     NewGoCodeAnalyzer(logger),
		docAnalyzer:      NewMarkdownAnalyzer(logger),
		comparisonEngine: NewDefaultComparisonEngine(logger),
		reportGenerator:  NewDefaultReportGenerator(),
		cache:            make(map[string]*AnalysisResult),
		logger:           logger,
		auditLogger:      auditLogger,
	}

	return analyzer
}

// AnalyzeProject performs comprehensive analysis of a project
func (da *DocumentationAnalyzer) AnalyzeProject(config AnalysisConfig) types.Result[*AnalysisResult] {
	startTime := time.Now()
	da.logger.Info("Starting project analysis", "project", config.ProjectRoot)

	// Create unique analysis ID
	analysisID := fmt.Sprintf("analysis-%d", time.Now().UnixMilli())

	// Check cache if enabled
	if config.CacheEnabled {
		if cached := da.getCachedResult(config); cached != nil {
			da.logger.Info("Using cached analysis result")
			return types.NewResult(cached)
		}
	}

	// Audit log analysis start
	startEvent := types.AuditEvent{
		UserID:   "system",
		Action:   "documentation_analysis_start",
		Resource: "project",
		Result:   "initiated",
		Details: map[string]interface{}{
			"analysis_id":  analysisID,
			"project_root": config.ProjectRoot,
			"parallel":     config.Parallel,
		},
	}
	da.auditLogger.LogEvent(startEvent)

	// Initialize analysis result
	result := &AnalysisResult{
		ID:          analysisID,
		Timestamp:   time.Now().UTC().UnixMilli(),
		ProjectPath: config.ProjectRoot,
		Metadata:    make(map[string]interface{}),
	}

	// Find and analyze files
	codeFiles, docFiles, err := da.findProjectFiles(config)
	if err != nil {
		da.logger.Error("Failed to find project files", "error", err)
		return types.NewResultError[*AnalysisResult](fmt.Errorf("failed to find project files: %w", err))
	}

	result.TotalFiles = len(codeFiles) + len(docFiles)
	da.logger.Info("Found project files", "code_files", len(codeFiles), "doc_files", len(docFiles))

	// Analyze code files
	codeSymbols, err := da.analyzeCodeFiles(codeFiles, config)
	if err != nil {
		da.logger.Error("Failed to analyze code files", "error", err)
		return types.NewResultError[*AnalysisResult](fmt.Errorf("failed to analyze code files: %w", err))
	}

	// Analyze documentation files
	docSections, err := da.analyzeDocFiles(docFiles, config)
	if err != nil {
		da.logger.Error("Failed to analyze documentation files", "error", err)
		return types.NewResultError[*AnalysisResult](fmt.Errorf("failed to analyze documentation files: %w", err))
	}

	// Detect inconsistencies
	inconsistencies, err := da.detectInconsistencies(codeSymbols, docSections, config)
	if err != nil {
		da.logger.Error("Failed to detect inconsistencies", "error", err)
		return types.NewResultError[*AnalysisResult](fmt.Errorf("failed to detect inconsistencies: %w", err))
	}

	// Calculate coverage and quality metrics
	coverage := da.calculateCoverage(codeSymbols, docSections)
	qualityScore := da.calculateQualityScore(inconsistencies, coverage)
	summary := da.generateSummary(inconsistencies, qualityScore)

	// Complete result
	result.AnalyzedFiles = len(codeFiles) + len(docFiles)
	result.Inconsistencies = inconsistencies
	result.QualityScore = qualityScore
	result.Coverage = coverage
	result.Summary = summary
	result.Duration = time.Since(startTime)

	// Cache result if enabled
	if config.CacheEnabled {
		da.cacheResult(config, result)
	}

	// Audit log completion
	completedEvent := types.AuditEvent{
		UserID:   "system",
		Action:   "documentation_analysis_completed",
		Resource: "project",
		Result:   "success",
		Details: map[string]interface{}{
			"analysis_id":     analysisID,
			"total_files":     result.TotalFiles,
			"analyzed_files":  result.AnalyzedFiles,
			"inconsistencies": len(inconsistencies),
			"quality_score":   qualityScore,
			"duration_ms":     result.Duration.Milliseconds(),
		},
	}
	da.auditLogger.LogEvent(completedEvent)

	da.logger.Info("Analysis completed",
		"inconsistencies", len(inconsistencies),
		"quality_score", qualityScore,
		"duration", result.Duration)

	return types.NewResult(result)
}

// AnalyzeIncremental performs incremental analysis of changed files
func (da *DocumentationAnalyzer) AnalyzeIncremental(config AnalysisConfig, changedFiles []string) types.Result[*AnalysisResult] {
	da.logger.Info("Starting incremental analysis", "changed_files", len(changedFiles))

	// Filter changed files by patterns
	relevantFiles := da.filterFiles(changedFiles, config.IncludePatterns, config.ExcludePatterns)
	if len(relevantFiles) == 0 {
		da.logger.Info("No relevant files changed")
		// Return empty result
		return types.NewResult(&AnalysisResult{
			ID:              fmt.Sprintf("incremental-%d", time.Now().UnixMilli()),
			Timestamp:       time.Now().UTC().UnixMilli(),
			ProjectPath:     config.ProjectRoot,
			TotalFiles:      0,
			AnalyzedFiles:   0,
			Inconsistencies: []Inconsistency{},
			QualityScore:    1.0,
			Coverage:        DocumentationCoverage{},
			Summary:         AnalysisSummary{},
			Metadata:        make(map[string]interface{}),
			Duration:        0,
		})
	}

	// For incremental analysis, we analyze the changed files and their dependencies
	// This is a simplified implementation - a full implementation would track dependencies
	config.ProjectRoot = config.ProjectRoot // Ensure project root is set
	return da.AnalyzeProject(config)
}

// CompareCodeAndDocs compares specific code and documentation files
func (da *DocumentationAnalyzer) CompareCodeAndDocs(codeFile, docFile string) types.Result[[]Inconsistency] {
	da.logger.Info("Comparing code and documentation files", "code_file", codeFile, "doc_file", docFile)

	// Extract symbols from code file
	codeSymbolsResult := da.codeAnalyzer.ExtractSymbols(codeFile)
	if codeSymbolsResult.IsErr() {
		return types.NewResultError[[]Inconsistency](codeSymbolsResult.Error)
	}
	codeSymbols := codeSymbolsResult.Value

	// Parse documentation file
	docSectionsResult := da.docAnalyzer.ParseMarkdown(docFile)
	if docSectionsResult.IsErr() {
		return types.NewResultError[[]Inconsistency](docSectionsResult.Error)
	}
	docSections := docSectionsResult.Value

	// Detect inconsistencies
	return da.comparisonEngine.DetectInconsistencies(codeSymbols, docSections)
}

// GenerateReport creates a formatted report from analysis results
func (da *DocumentationAnalyzer) GenerateReport(result *AnalysisResult, format string) types.Result[*AnalysisReport] {
	da.logger.Info("Generating report", "format", format)

	switch strings.ToLower(format) {
	case "json":
		contentResult := da.reportGenerator.GenerateJSONReport(result)
		if contentResult.IsErr() {
			return types.NewResultError[*AnalysisReport](contentResult.Error)
		}
		return types.NewResult(&AnalysisReport{
			Format:   "json",
			Content:  contentResult.Value,
			Summary:  fmt.Sprintf("Analysis found %d inconsistencies with quality score %.2f", len(result.Inconsistencies), result.QualityScore),
			Metadata: make(map[string]interface{}),
		})

	case "markdown", "md":
		contentResult := da.reportGenerator.GenerateMarkdownReport(result)
		if contentResult.IsErr() {
			return types.NewResultError[*AnalysisReport](contentResult.Error)
		}
		return types.NewResult(&AnalysisReport{
			Format:   "markdown",
			Content:  contentResult.Value,
			Summary:  fmt.Sprintf("Analysis found %d inconsistencies with quality score %.2f", len(result.Inconsistencies), result.QualityScore),
			Metadata: make(map[string]interface{}),
		})

	case "html":
		contentResult := da.reportGenerator.GenerateHTMLReport(result)
		if contentResult.IsErr() {
			return types.NewResultError[*AnalysisReport](contentResult.Error)
		}
		return types.NewResult(&AnalysisReport{
			Format:   "html",
			Content:  contentResult.Value,
			Summary:  fmt.Sprintf("Analysis found %d inconsistencies with quality score %.2f", len(result.Inconsistencies), result.QualityScore),
			Metadata: make(map[string]interface{}),
		})

	default: // text
		contentResult := da.reportGenerator.GenerateTextReport(result)
		if contentResult.IsErr() {
			return types.NewResultError[*AnalysisReport](contentResult.Error)
		}
		return types.NewResult(&AnalysisReport{
			Format:   "text",
			Content:  contentResult.Value,
			Summary:  fmt.Sprintf("Analysis found %d inconsistencies with quality score %.2f", len(result.Inconsistencies), result.QualityScore),
			Metadata: make(map[string]interface{}),
		})
	}
}

// ValidateDocumentation validates documentation structure and content
func (da *DocumentationAnalyzer) ValidateDocumentation(docFiles []string) types.Result[[]Inconsistency] {
	da.logger.Info("Validating documentation", "files", len(docFiles))

	var allInconsistencies []Inconsistency

	for _, docFile := range docFiles {
		// Validate links
		linkResult := da.docAnalyzer.ValidateLinks([]string{docFile})
		if linkResult.IsOk() {
			allInconsistencies = append(allInconsistencies, linkResult.Value...)
		}

		// Parse and validate structure
		sectionsResult := da.docAnalyzer.ParseMarkdown(docFile)
		if sectionsResult.IsErr() {
			// Create inconsistency for parsing failure
			inconsistency := Inconsistency{
				ID:          fmt.Sprintf("parse-error-%s", filepath.Base(docFile)),
				Type:        InconsistencyIncorrectDoc,
				Severity:    SeverityHigh,
				Title:       "Documentation parsing failed",
				Description: fmt.Sprintf("Failed to parse documentation file: %s", sectionsResult.Error.Error()),
				DocFile:     docFile,
				DocLine:     1,
				Suggestion:  "Fix markdown syntax errors",
			}
			allInconsistencies = append(allInconsistencies, inconsistency)
		}
	}

	return types.NewResult(allInconsistencies)
}

// GetCoverageMetrics calculates documentation coverage metrics
func (da *DocumentationAnalyzer) GetCoverageMetrics(projectPath string) types.Result[*DocumentationCoverage] {
	da.logger.Info("Calculating coverage metrics", "project", projectPath)

	config := DefaultAnalysisConfig()
	config.ProjectRoot = projectPath

	// Find code files
	codeFiles, _, err := da.findProjectFiles(config)
	if err != nil {
		return types.NewResultError[*DocumentationCoverage](fmt.Errorf("failed to find project files: %w", err))
	}

	// Analyze code symbols
	codeSymbols, err := da.analyzeCodeFiles(codeFiles, config)
	if err != nil {
		return types.NewResultError[*DocumentationCoverage](fmt.Errorf("failed to analyze code files: %w", err))
	}

	// Calculate coverage
	coverage := da.calculateCoverage(codeSymbols, nil)
	return types.NewResult(&coverage)
}

// Helper methods

func (da *DocumentationAnalyzer) findProjectFiles(config AnalysisConfig) (codeFiles, docFiles []string, err error) {
	err = filepath.WalkDir(config.ProjectRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Check if file matches patterns
		if da.matchesPatterns(path, config.IncludePatterns) && !da.matchesPatterns(path, config.ExcludePatterns) {
			if da.matchesPatterns(path, config.CodePatterns) {
				codeFiles = append(codeFiles, path)
			} else if da.matchesPatterns(path, config.DocPatterns) {
				docFiles = append(docFiles, path)
			}
		}

		return nil
	})

	return codeFiles, docFiles, err
}

func (da *DocumentationAnalyzer) matchesPatterns(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
		// Simple wildcard matching for directory patterns
		if strings.Contains(pattern, "**") {
			simplePattern := strings.ReplaceAll(pattern, "**", "*")
			if matched, _ := filepath.Match(simplePattern, path); matched {
				return true
			}
		}
	}
	return false
}

func (da *DocumentationAnalyzer) filterFiles(files []string, includePatterns, excludePatterns []string) []string {
	var filtered []string
	for _, file := range files {
		if da.matchesPatterns(file, includePatterns) && !da.matchesPatterns(file, excludePatterns) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

func (da *DocumentationAnalyzer) analyzeCodeFiles(codeFiles []string, config AnalysisConfig) ([]CodeSymbol, error) {
	var allSymbols []CodeSymbol

	if config.Parallel {
		// Parallel analysis
		type result struct {
			symbols []CodeSymbol
			err     error
		}

		results := make(chan result, len(codeFiles))
		var wg sync.WaitGroup

		for _, file := range codeFiles {
			wg.Add(1)
			go func(f string) {
				defer wg.Done()
				symbolsResult := da.codeAnalyzer.ExtractSymbols(f)
				if symbolsResult.IsOk() {
					results <- result{symbols: symbolsResult.Value}
				} else {
					results <- result{err: symbolsResult.Error}
				}
			}(file)
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		for res := range results {
			if res.err != nil {
				da.logger.Warn("Failed to analyze file", "error", res.err)
				continue
			}
			allSymbols = append(allSymbols, res.symbols...)
		}

	} else {
		// Sequential analysis
		for _, file := range codeFiles {
			symbolsResult := da.codeAnalyzer.ExtractSymbols(file)
			if symbolsResult.IsOk() {
				allSymbols = append(allSymbols, symbolsResult.Value...)
			} else {
				da.logger.Warn("Failed to analyze file", "file", file, "error", symbolsResult.Error)
			}
		}
	}

	return allSymbols, nil
}

func (da *DocumentationAnalyzer) analyzeDocFiles(docFiles []string, config AnalysisConfig) ([]DocSection, error) {
	var allSections []DocSection

	for _, file := range docFiles {
		sectionsResult := da.docAnalyzer.ParseMarkdown(file)
		if sectionsResult.IsOk() {
			allSections = append(allSections, sectionsResult.Value...)
		} else {
			da.logger.Warn("Failed to parse documentation file", "file", file, "error", sectionsResult.Error)
		}
	}

	return allSections, nil
}

func (da *DocumentationAnalyzer) detectInconsistencies(codeSymbols []CodeSymbol, docSections []DocSection, config AnalysisConfig) ([]Inconsistency, error) {
	inconsistenciesResult := da.comparisonEngine.DetectInconsistencies(codeSymbols, docSections)
	if inconsistenciesResult.IsErr() {
		return nil, inconsistenciesResult.Error
	}

	inconsistencies := inconsistenciesResult.Value

	// Filter by severity if specified
	if config.MinSeverity != SeverityLow {
		filtered := make([]Inconsistency, 0, len(inconsistencies))
		for _, inc := range inconsistencies {
			if da.severityLevel(inc.Severity) >= da.severityLevel(config.MinSeverity) {
				filtered = append(filtered, inc)
			}
		}
		inconsistencies = filtered
	}

	// Filter by enabled types
	if len(config.EnableTypes) > 0 {
		enabledTypes := make(map[InconsistencyType]bool)
		for _, t := range config.EnableTypes {
			enabledTypes[t] = true
		}

		filtered := make([]Inconsistency, 0, len(inconsistencies))
		for _, inc := range inconsistencies {
			if enabledTypes[inc.Type] {
				filtered = append(filtered, inc)
			}
		}
		inconsistencies = filtered
	}

	return inconsistencies, nil
}

func (da *DocumentationAnalyzer) calculateCoverage(codeSymbols []CodeSymbol, docSections []DocSection) DocumentationCoverage {
	// Count exported symbols
	exportedSymbols := make([]CodeSymbol, 0, len(codeSymbols))
	for _, symbol := range codeSymbols {
		if symbol.Exported {
			exportedSymbols = append(exportedSymbols, symbol)
		}
	}

	// Count documented symbols (simplified - checks if symbol has comment)
	documentedCount := 0
	for _, symbol := range exportedSymbols {
		if symbol.Comment != "" {
			documentedCount++
		}
	}

	totalSymbols := len(exportedSymbols)
	coveragePercent := 0.0
	if totalSymbols > 0 {
		coveragePercent = float64(documentedCount) / float64(totalSymbols) * 100
	}

	// Calculate by type
	byType := make(map[string]CoverageByType)
	typeStats := make(map[string]struct {
		total      int
		documented int
	})

	for _, symbol := range exportedSymbols {
		stats := typeStats[symbol.Type]
		stats.total++
		if symbol.Comment != "" {
			stats.documented++
		}
		typeStats[symbol.Type] = stats
	}

	for symbolType, stats := range typeStats {
		percent := 0.0
		if stats.total > 0 {
			percent = float64(stats.documented) / float64(stats.total) * 100
		}
		byType[symbolType] = CoverageByType{
			Total:      stats.total,
			Documented: stats.documented,
			Percent:    percent,
		}
	}

	return DocumentationCoverage{
		TotalSymbols:      totalSymbols,
		DocumentedSymbols: documentedCount,
		CoveragePercent:   coveragePercent,
		ByType:            byType,
		ByPackage:         make(map[string]CoverageByType), // TODO: Implement package-level coverage
	}
}

func (da *DocumentationAnalyzer) calculateQualityScore(inconsistencies []Inconsistency, coverage DocumentationCoverage) float64 {
	// Base score from coverage
	coverageScore := coverage.CoveragePercent / 100.0

	// Penalty from inconsistencies
	penalty := 0.0
	for _, inc := range inconsistencies {
		switch inc.Severity {
		case SeverityCritical:
			penalty += 0.2
		case SeverityHigh:
			penalty += 0.1
		case SeverityMedium:
			penalty += 0.05
		case SeverityLow:
			penalty += 0.01
		}
	}

	// Calculate final score (0.0 to 1.0)
	score := coverageScore - penalty
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

func (da *DocumentationAnalyzer) generateSummary(inconsistencies []Inconsistency, qualityScore float64) AnalysisSummary {
	// Count by severity
	bySeverity := make(map[SeverityLevel]int)
	byType := make(map[InconsistencyType]int)

	for _, inc := range inconsistencies {
		bySeverity[inc.Severity]++
		byType[inc.Type]++
	}

	// Generate recommendations
	recommendations := da.generateRecommendations(inconsistencies, qualityScore)

	// Determine quality grade
	qualityGrade := "F"
	if qualityScore >= 0.9 {
		qualityGrade = "A"
	} else if qualityScore >= 0.8 {
		qualityGrade = "B"
	} else if qualityScore >= 0.7 {
		qualityGrade = "C"
	} else if qualityScore >= 0.6 {
		qualityGrade = "D"
	}

	// Identify improvement areas
	improvementAreas := da.identifyImprovementAreas(byType)

	return AnalysisSummary{
		TotalInconsistencies: len(inconsistencies),
		BySeverity:           bySeverity,
		ByType:               byType,
		Recommendations:      recommendations,
		QualityGrade:         qualityGrade,
		ImprovementAreas:     improvementAreas,
	}
}

func (da *DocumentationAnalyzer) generateRecommendations(inconsistencies []Inconsistency, qualityScore float64) []Recommendation {
	var recommendations []Recommendation

	if qualityScore < 0.7 {
		recommendations = append(recommendations, Recommendation{
			ID:              "improve-coverage",
			Priority:        "high",
			Title:           "Improve documentation coverage",
			Description:     "Documentation coverage is below 70%. Focus on documenting public APIs.",
			Action:          "Add documentation comments to exported functions, types, and packages",
			EstimatedEffort: "2-4 hours",
			Impact:          "high",
		})
	}

	// Count critical and high severity issues
	criticalCount := 0
	highCount := 0
	for _, inc := range inconsistencies {
		if inc.Severity == SeverityCritical {
			criticalCount++
		} else if inc.Severity == SeverityHigh {
			highCount++
		}
	}

	if criticalCount > 0 {
		recommendations = append(recommendations, Recommendation{
			ID:              "fix-critical",
			Priority:        "critical",
			Title:           "Fix critical documentation issues",
			Description:     fmt.Sprintf("Found %d critical documentation issues that require immediate attention", criticalCount),
			Action:          "Review and fix critical inconsistencies immediately",
			EstimatedEffort: fmt.Sprintf("%d-%.0f hours", criticalCount, float64(criticalCount)*1.5),
			Impact:          "critical",
		})
	}

	if highCount > 0 {
		recommendations = append(recommendations, Recommendation{
			ID:              "fix-high-priority",
			Priority:        "high",
			Title:           "Address high priority issues",
			Description:     fmt.Sprintf("Found %d high priority issues that should be addressed soon", highCount),
			Action:          "Review and fix high priority inconsistencies",
			EstimatedEffort: fmt.Sprintf("%.0f-%.0f hours", float64(highCount)*0.5, float64(highCount)*1.0),
			Impact:          "high",
		})
	}

	return recommendations
}

func (da *DocumentationAnalyzer) identifyImprovementAreas(byType map[InconsistencyType]int) []string {
	var areas []string

	// Sort by count to identify top issues
	type typeCount struct {
		incType InconsistencyType
		count   int
	}

	var typeCounts []typeCount
	for incType, count := range byType {
		typeCounts = append(typeCounts, typeCount{incType, count})
	}

	sort.Slice(typeCounts, func(i, j int) bool {
		return typeCounts[i].count > typeCounts[j].count
	})

	// Add top 3 improvement areas
	for i, tc := range typeCounts {
		if i >= 3 {
			break
		}
		switch tc.incType {
		case InconsistencyMissingDoc:
			areas = append(areas, "Missing documentation for public APIs")
		case InconsistencyOutdatedDoc:
			areas = append(areas, "Outdated documentation needs updating")
		case InconsistencyBrokenLink:
			areas = append(areas, "Broken links in documentation")
		case InconsistencyMissingExample:
			areas = append(areas, "Missing code examples")
		default:
			areas = append(areas, fmt.Sprintf("Address %s issues", strings.ReplaceAll(string(tc.incType), "_", " ")))
		}
	}

	return areas
}

func (da *DocumentationAnalyzer) severityLevel(severity SeverityLevel) int {
	switch severity {
	case SeverityLow:
		return 1
	case SeverityMedium:
		return 2
	case SeverityHigh:
		return 3
	case SeverityCritical:
		return 4
	default:
		return 0
	}
}

// Cache management methods

func (da *DocumentationAnalyzer) getCachedResult(config AnalysisConfig) *AnalysisResult {
	da.cacheMutex.RLock()
	defer da.cacheMutex.RUnlock()

	key := da.generateCacheKey(config)
	if result, exists := da.cache[key]; exists {
		// Check if cache is still valid
		cacheAge := time.Since(time.Unix(result.Timestamp/1000, 0))
		if cacheAge < config.CacheTTL {
			return result
		}
		// Remove expired cache
		delete(da.cache, key)
	}

	return nil
}

func (da *DocumentationAnalyzer) cacheResult(config AnalysisConfig, result *AnalysisResult) {
	da.cacheMutex.Lock()
	defer da.cacheMutex.Unlock()

	key := da.generateCacheKey(config)
	da.cache[key] = result
}

func (da *DocumentationAnalyzer) generateCacheKey(config AnalysisConfig) string {
	// Create hash of configuration to use as cache key
	configJSON, _ := json.Marshal(config)
	hash := sha256.Sum256(configJSON)
	return fmt.Sprintf("%x", hash[:8])
}
