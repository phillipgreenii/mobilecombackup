// Package core provides quality dashboard functionality for testing metrics
package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// QualityDashboard provides comprehensive testing and quality metrics visualization
type QualityDashboard struct {
	projectName string
	metricsDir  string
}

// NewQualityDashboard creates a new quality dashboard instance
func NewQualityDashboard(projectName, metricsDir string) *QualityDashboard {
	return &QualityDashboard{
		projectName: projectName,
		metricsDir:  metricsDir,
	}
}

// DashboardReport represents a complete quality dashboard report
type DashboardReport struct {
	ProjectName   string              `json:"project_name"`
	GeneratedAt   time.Time           `json:"generated_at"`
	Coverage      CoverageReport      `json:"coverage"`
	Performance   PerformanceReport   `json:"performance"`
	Quality       QualityReport       `json:"quality"`
	TestExecution TestExecutionReport `json:"test_execution"`
	Trends        TrendAnalysis       `json:"trends"`
	Summary       DashboardSummary    `json:"summary"`
}

// CoverageReport tracks test coverage metrics
type CoverageReport struct {
	OverallCoverage   float64            `json:"overall_coverage"`
	PackageCoverage   map[string]float64 `json:"package_coverage"`
	LineCoverage      map[string]int     `json:"line_coverage"`
	BranchCoverage    map[string]int     `json:"branch_coverage"`
	CoverageThreshold float64            `json:"coverage_threshold"`
	MeetingThreshold  bool               `json:"meeting_threshold"`
	UncoveredLines    map[string][]int   `json:"uncovered_lines"`
}

// PerformanceReport tracks performance metrics and regressions
type PerformanceReport struct {
	CurrentMetrics    PerformanceMetrics      `json:"current_metrics"`
	BenchmarkResults  []BenchmarkResult       `json:"benchmark_results"`
	MemoryUsage       MemoryStats             `json:"memory_usage"`
	ThroughputMetrics ThroughputReport        `json:"throughput_metrics"`
	RegressionAlerts  []PerformanceRegression `json:"regression_alerts"`
}

// QualityReport tracks code quality metrics
type QualityReport struct {
	QualityMetrics  QualityMetrics      `json:"quality_metrics"`
	ComplexityTrend []float64           `json:"complexity_trend"`
	TechnicalDebt   TechnicalDebtReport `json:"technical_debt"`
	CodeSmells      CodeSmellReport     `json:"code_smells"`
	QualityGates    []QualityGate       `json:"quality_gates"`
}

// TestExecutionReport tracks test execution statistics
type TestExecutionReport struct {
	TotalTests    int               `json:"total_tests"`
	PassedTests   int               `json:"passed_tests"`
	FailedTests   int               `json:"failed_tests"`
	SkippedTests  int               `json:"skipped_tests"`
	ExecutionTime time.Duration     `json:"execution_time"`
	TestSuites    []TestSuiteResult `json:"test_suites"`
	FlakyTests    []FlakyTestReport `json:"flaky_tests"`
	SuccessRate   float64           `json:"success_rate"`
}

// TrendAnalysis provides historical trend analysis
type TrendAnalysis struct {
	CoverageTrend    []TrendPoint `json:"coverage_trend"`
	PerformanceTrend []TrendPoint `json:"performance_trend"`
	QualityTrend     []TrendPoint `json:"quality_trend"`
	TestCountTrend   []TrendPoint `json:"test_count_trend"`
	TimeRange        TimeRange    `json:"time_range"`
}

// DashboardSummary provides key dashboard insights
type DashboardSummary struct {
	Status          string                 `json:"status"` // "excellent", "good", "needs_attention", "critical"
	KeyMetrics      map[string]interface{} `json:"key_metrics"`
	Recommendations []string               `json:"recommendations"`
	CriticalIssues  []string               `json:"critical_issues"`
	OverallScore    float64                `json:"overall_score"`
}

// Supporting types for detailed reporting

type BenchmarkResult struct {
	Name             string        `json:"name"`
	Duration         time.Duration `json:"duration"`
	MemoryAllocs     int64         `json:"memory_allocs"`
	MemoryBytes      int64         `json:"memory_bytes"`
	OperationsPerSec float64       `json:"operations_per_sec"`
	Iterations       int           `json:"iterations"`
}

type ThroughputReport struct {
	FilesPerSecond     float64         `json:"files_per_second"`
	SectionsPerSecond  float64         `json:"sections_per_second"`
	ProcessingLatency  time.Duration   `json:"processing_latency"`
	ConcurrencyMetrics map[int]float64 `json:"concurrency_metrics"` // workers -> throughput
}

type PerformanceRegression struct {
	Metric        string    `json:"metric"`
	Previous      float64   `json:"previous"`
	Current       float64   `json:"current"`
	ChangePercent float64   `json:"change_percent"`
	Severity      string    `json:"severity"` // "minor", "moderate", "major", "critical"
	DetectedAt    time.Time `json:"detected_at"`
}

type TechnicalDebtReport struct {
	TotalDebtMinutes int            `json:"total_debt_minutes"`
	DebtByType       map[string]int `json:"debt_by_type"`
	DebtTrend        []int          `json:"debt_trend"`
	HighDebtFiles    []string       `json:"high_debt_files"`
}

type CodeSmellReport struct {
	TotalSmells    int            `json:"total_smells"`
	SmellsByType   map[string]int `json:"smells_by_type"`
	SmellsByFile   map[string]int `json:"smells_by_file"`
	CriticalSmells []string       `json:"critical_smells"`
}

type QualityGate struct {
	Name         string  `json:"name"`
	Threshold    float64 `json:"threshold"`
	CurrentValue float64 `json:"current_value"`
	Passed       bool    `json:"passed"`
	Importance   string  `json:"importance"` // "low", "medium", "high", "critical"
}

type TestSuiteResult struct {
	Name         string        `json:"name"`
	Duration     time.Duration `json:"duration"`
	TestCount    int           `json:"test_count"`
	PassedCount  int           `json:"passed_count"`
	FailedCount  int           `json:"failed_count"`
	SkippedCount int           `json:"skipped_count"`
	Coverage     float64       `json:"coverage"`
}

type FlakyTestReport struct {
	TestName        string    `json:"test_name"`
	FailureRate     float64   `json:"failure_rate"`
	RecentFailures  int       `json:"recent_failures"`
	LastFailure     time.Time `json:"last_failure"`
	FailurePatterns []string  `json:"failure_patterns"`
}

type TrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Label     string    `json:"label,omitempty"`
}

type TimeRange struct {
	Start    time.Time     `json:"start"`
	End      time.Time     `json:"end"`
	Duration time.Duration `json:"duration"`
}

// Dashboard Interface Methods

// GenerateReport creates a comprehensive dashboard report
func (qd *QualityDashboard) GenerateReport() (*DashboardReport, error) {
	report := &DashboardReport{
		ProjectName: qd.projectName,
		GeneratedAt: time.Now(),
	}

	// Generate each section of the report
	if err := qd.generateCoverageReport(&report.Coverage); err != nil {
		return nil, fmt.Errorf("failed to generate coverage report: %w", err)
	}

	if err := qd.generatePerformanceReport(&report.Performance); err != nil {
		return nil, fmt.Errorf("failed to generate performance report: %w", err)
	}

	if err := qd.generateQualityReport(&report.Quality); err != nil {
		return nil, fmt.Errorf("failed to generate quality report: %w", err)
	}

	if err := qd.generateTestExecutionReport(&report.TestExecution); err != nil {
		return nil, fmt.Errorf("failed to generate test execution report: %w", err)
	}

	if err := qd.generateTrendAnalysis(&report.Trends); err != nil {
		return nil, fmt.Errorf("failed to generate trend analysis: %w", err)
	}

	// Generate summary based on all metrics
	qd.generateSummary(&report.Summary, report)

	return report, nil
}

// SaveReport saves the dashboard report to JSON file
func (qd *QualityDashboard) SaveReport(report *DashboardReport, filename string) error {
	// Ensure metrics directory exists
	if err := os.MkdirAll(qd.metricsDir, 0750); err != nil {
		return fmt.Errorf("failed to create metrics directory: %w", err)
	}

	filepath := filepath.Join(qd.metricsDir, filename)

	// Marshal to JSON with indentation
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report to JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filepath, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	return nil
}

// LoadHistoricalReports loads historical reports for trend analysis
func (qd *QualityDashboard) LoadHistoricalReports(limit int) ([]*DashboardReport, error) {
	files, err := os.ReadDir(qd.metricsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read metrics directory: %w", err)
	}

	var reports []*DashboardReport
	var reportFiles []os.DirEntry

	// Filter for JSON report files
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			reportFiles = append(reportFiles, file)
		}
	}

	// Sort by modification time (newest first)
	sort.Slice(reportFiles, func(i, j int) bool {
		infoI, _ := reportFiles[i].Info()
		infoJ, _ := reportFiles[j].Info()
		return infoI.ModTime().After(infoJ.ModTime())
	})

	// Load up to limit reports
	loadLimit := limit
	if len(reportFiles) < loadLimit {
		loadLimit = len(reportFiles)
	}

	for i := 0; i < loadLimit; i++ {
		filePath := filepath.Join(qd.metricsDir, reportFiles[i].Name())

		data, err := os.ReadFile(filePath) //nolint:gosec
		if err != nil {
			continue // Skip files that can't be read
		}

		var report DashboardReport
		if err := json.Unmarshal(data, &report); err != nil {
			continue // Skip files that can't be parsed
		}

		reports = append(reports, &report)
	}

	return reports, nil
}

// TrackMetrics records new metrics for tracking over time
func (qd *QualityDashboard) TrackMetrics(metrics TestMetrics) error {
	// Create a timestamped metrics entry
	entry := MetricsEntry{
		Timestamp: time.Now(),
		Metrics:   metrics,
	}

	// Save to metrics history
	return qd.saveMetricsEntry(entry)
}

// MetricsEntry represents a timestamped metrics recording
type MetricsEntry struct {
	Timestamp time.Time   `json:"timestamp"`
	Metrics   TestMetrics `json:"metrics"`
}

// PerformanceMetrics captures detailed performance data (from performance_test.go)
type PerformanceMetrics struct {
	TotalFiles     int           `json:"total_files"`
	TotalSections  int           `json:"total_sections"`
	ProcessingTime time.Duration `json:"processing_time"`
	FilesPerSecond float64       `json:"files_per_second"`
	SectionsPerSec float64       `json:"sections_per_second"`
	MemoryUsage    MemoryStats   `json:"memory_usage"`
	CPUUsage       float64       `json:"cpu_usage_percent"`
}

// MemoryStats captures memory usage information (from performance_test.go)
type MemoryStats struct {
	AllocBytes      uint64 `json:"alloc_bytes"`
	TotalAllocBytes uint64 `json:"total_alloc_bytes"`
	SysBytes        uint64 `json:"sys_bytes"`
	NumGC           uint32 `json:"num_gc"`
	HeapObjects     uint64 `json:"heap_objects"`
}

// QualityMetrics captures quality assessment data (from performance_test.go)
type QualityMetrics struct {
	Coverage           float64 `json:"coverage_percent"`
	ComplexityScore    float64 `json:"complexity_score"`
	MaintainabilityIdx float64 `json:"maintainability_index"`
	TechnicalDebt      int     `json:"technical_debt_minutes"`
	CodeSmells         int     `json:"code_smells"`
}

// TestMetrics captures comprehensive test execution metrics
type TestMetrics struct {
	ExecutionTime    time.Duration     `json:"execution_time"`
	MemoryUsage      int64             `json:"memory_usage"`
	CoveragePercent  float64           `json:"coverage_percent"`
	PassedTests      int               `json:"passed_tests"`
	FailedTests      int               `json:"failed_tests"`
	SkippedTests     int               `json:"skipped_tests"`
	BenchmarkResults []BenchmarkResult `json:"benchmark_results"`
}

// saveMetricsEntry saves a metrics entry to the historical data
func (qd *QualityDashboard) saveMetricsEntry(entry MetricsEntry) error {
	filename := fmt.Sprintf("metrics_%s.json", entry.Timestamp.Format("20060102_150405"))
	filepath := filepath.Join(qd.metricsDir, filename)

	jsonData, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics entry: %w", err)
	}

	return os.WriteFile(filepath, jsonData, 0600)
}
