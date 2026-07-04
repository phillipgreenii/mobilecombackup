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

// DocumentationScanState represents the persistent state of documentation scanning
type DocumentationScanState struct {
	Version      string                       `json:"version"`
	LastScanTime int64                        `json:"last_scan_time"`
	FileStates   map[string]DocumentFileState `json:"file_states"`
	Metadata     DocumentationStateMetadata   `json:"metadata"`
}

// DocumentFileState represents the state of a single documentation file
type DocumentFileState struct {
	Path         string       `json:"path"`
	LastModified int64        `json:"last_modified"`
	Checksum     string       `json:"checksum"`
	Sections     []DocSection `json:"sections"`
	LastScanned  int64        `json:"last_scanned"`
	ScanCount    int          `json:"scan_count"`
}

// DocumentationStateMetadata contains metadata about the documentation scan state
type DocumentationStateMetadata struct {
	Status        string `json:"status"`
	TotalFiles    int    `json:"total_files"`
	TotalSections int    `json:"total_sections"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
	Checksum      string `json:"checksum"`
}

// Event represents a system event in the documentation synchronization system
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Target    string                 `json:"target,omitempty"`
	Data      map[string]interface{} `json:"data"`
	Timestamp int64                  `json:"timestamp"`
	Priority  EventPriority          `json:"priority"`
}

// EventPriority defines the priority level of events
type EventPriority string

const (
	EventPriorityLow    EventPriority = "low"
	EventPriorityNormal EventPriority = "normal"
	EventPriorityHigh   EventPriority = "high"
)

// EventHandler defines a function that handles events
type EventHandler func(event Event) error

// EventBus defines the interface for event publishing and subscription
type EventBus interface {
	Subscribe(eventType string, handler EventHandler) types.Result[interface{}]
	Publish(event Event) types.Result[interface{}]
	Unsubscribe(eventType string, handler EventHandler) types.Result[interface{}]
	GetEventHistory(limit int) types.Result[[]Event]
}

// SyncConfig contains configuration for documentation synchronization
type SyncConfig struct {
	ProjectRoot     string                 `json:"project_root"`
	SyncMode        SyncMode               `json:"sync_mode"`
	EnabledAgents   []string               `json:"enabled_agents"`
	WatchEnabled    bool                   `json:"watch_enabled"`
	MaxWorkers      int                    `json:"max_workers"`
	MaxConcurrency  int                    `json:"max_concurrency"`
	BatchSize       int                    `json:"batch_size"`
	TimeoutMinutes  int                    `json:"timeout_minutes"`
	RetryAttempts   int                    `json:"retry_attempts"`
	IncludePatterns []string               `json:"include_patterns"`
	ExcludePatterns []string               `json:"exclude_patterns"`
	OutputFormat    string                 `json:"output_format"`
	DryRun          bool                   `json:"dry_run"`
	AutoFix         bool                   `json:"auto_fix"`
	Verbose         bool                   `json:"verbose"`
	LogLevel        string                 `json:"log_level"`
	AgentConfig     map[string]interface{} `json:"agent_config"`
	Options         map[string]interface{} `json:"options"`
}

// SyncMode defines the synchronization mode
type SyncMode string

const (
	SyncModeFull        SyncMode = "full"
	SyncModeIncremental SyncMode = "incremental"
	SyncModeWatch       SyncMode = "watch"
)

// WatchSession represents an active file watching session
type WatchSession struct {
	ID           string   `json:"id"`
	ProjectRoot  string   `json:"project_root"`
	StartTime    int64    `json:"start_time"`
	Active       bool     `json:"active"`
	IsActive     bool     `json:"is_active"`
	WatchedPaths []string `json:"watched_paths"`
	LastActivity int64    `json:"last_activity"`
}

// AgentStatus represents the status of a synchronization agent
type AgentStatus struct {
	Name         string                 `json:"name"`
	Status       string                 `json:"status"`
	State        AgentState             `json:"state"`
	Progress     float64                `json:"progress"`
	LastRun      int64                  `json:"last_run"`
	LastDuration int64                  `json:"last_duration"`
	CurrentTask  string                 `json:"current_task"`
	SuccessRate  float64                `json:"success_rate"`
	ErrorCount   int                    `json:"error_count"`
	Active       bool                   `json:"active"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// SyncResult contains the results of a synchronization operation
type SyncResult struct {
	Success            bool                    `json:"success"`
	StartTime          int64                   `json:"start_time"`
	EndTime            int64                   `json:"end_time"`
	Duration           int64                   `json:"duration"`
	SyncMode           SyncMode                `json:"sync_mode"`
	FilesScanned       int                     `json:"files_scanned"`
	Inconsistencies    []Inconsistency         `json:"inconsistencies"`
	ExecutedAgents     []string                `json:"executed_agents"`
	SkippedAgents      []string                `json:"skipped_agents"`
	FailedAgents       []string                `json:"failed_agents"`
	AgentResults       map[string]interface{}  `json:"agent_results"`
	DocumentationState *DocumentationScanState `json:"documentation_state,omitempty"`
	CodeSymbols        []CodeSymbol            `json:"code_symbols"`
	QualityMetrics     *QualityMetrics         `json:"quality_metrics,omitempty"`
	PerformanceMetrics *PerformanceMetrics     `json:"performance_metrics,omitempty"`
	Errors             []SyncError             `json:"errors"`
	Summary            string                  `json:"summary"`
	Metadata           map[string]interface{}  `json:"metadata"`
}

// SyncError represents an error that occurred during synchronization
type SyncError struct {
	Type        string                 `json:"type"`
	Message     string                 `json:"message"`
	Agent       string                 `json:"agent"`
	File        string                 `json:"file,omitempty"`
	Line        int                    `json:"line,omitempty"`
	Timestamp   int64                  `json:"timestamp"`
	Severity    SeverityLevel          `json:"severity"`
	Recoverable bool                   `json:"recoverable"`
	Context     map[string]interface{} `json:"context"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SyncState represents the state of a sync operation
type SyncState string

const (
	SyncStatePending    SyncState = "pending"
	SyncStateInProgress SyncState = "in_progress"
	SyncStateCompleted  SyncState = "completed"
	SyncStateFailed     SyncState = "failed"
)

// SyncStatus contains current synchronization status and progress
type SyncStatus struct {
	IsActive            bool               `json:"is_active"`
	CurrentMode         SyncMode           `json:"current_mode"`
	OverallProgress     float64            `json:"overall_progress"`
	RunningAgents       []string           `json:"running_agents"`
	CompletedAgents     []string           `json:"completed_agents"`
	FailedAgents        []string           `json:"failed_agents"`
	AgentProgress       map[string]float64 `json:"agent_progress"`
	LastSync            int64              `json:"last_sync"`
	EstimatedCompletion int64              `json:"estimated_completion"`
	WatchSession        *WatchSession      `json:"watch_session,omitempty"`
}

// QualityMetrics contains quality assessment metrics
type QualityMetrics struct {
	CoveragePercent      float64            `json:"coverage_percent"`
	InconsistencyCount   int                `json:"inconsistency_count"`
	TotalInconsistencies int                `json:"total_inconsistencies"`
	CriticalIssues       int                `json:"critical_issues"`
	HighPriorityIssues   int                `json:"high_priority_issues"`
	MediumPriorityIssues int                `json:"medium_priority_issues"`
	LowPriorityIssues    int                `json:"low_priority_issues"`
	QualityScore         float64            `json:"quality_score"`
	CoverageScore        float64            `json:"coverage_score"`
	ConsistencyScore     float64            `json:"consistency_score"`
	DocumentationScore   float64            `json:"documentation_score"`
	CodeQualityScore     float64            `json:"code_quality_score"`
	OverallScore         float64            `json:"overall_score"`
	ScoreBreakdown       map[string]float64 `json:"score_breakdown"`
	Grade                string             `json:"grade"`
}

// PerformanceMetrics contains performance measurement data
type PerformanceMetrics struct {
	ProcessingTime    time.Duration    `json:"processing_time"`
	TotalDuration     int64            `json:"total_duration"`
	FilesPerSecond    float64          `json:"files_per_second"`
	FilesProcessed    int              `json:"files_processed"`
	MemoryUsage       int64            `json:"memory_usage"`
	MemoryPeakMB      float64          `json:"memory_peak_mb"`
	MemoryAverageMB   float64          `json:"memory_average_mb"`
	ConcurrentWorkers int              `json:"concurrent_workers"`
	CPUAveragePercent float64          `json:"cpu_average_percent"`
	CacheHitRatio     float64          `json:"cache_hit_ratio"`
	AgentDurations    map[string]int64 `json:"agent_durations"`
}

// AgentState represents the state of an analysis agent
type AgentState string

const (
	AgentStateIdle      AgentState = "idle"
	AgentStateRunning   AgentState = "running"
	AgentStatePaused    AgentState = "paused"
	AgentStateComplete  AgentState = "complete"
	AgentStateCompleted AgentState = "completed"
	AgentStateFailed    AgentState = "failed"
)

// AgentExecutionState represents detailed execution state of an analysis agent
type AgentExecutionState struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	LastRun    int64  `json:"last_run"`
	RunCount   int    `json:"run_count"`
	ErrorCount int    `json:"error_count"`
	Active     bool   `json:"active"`
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
