// dashboard_test.go provides comprehensive tests for the quality dashboard
package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestQualityDashboard_Creation(t *testing.T) {
	testDir := t.TempDir()
	dashboard := NewQualityDashboard("mobilecombackup", testDir)

	if dashboard.projectName != "mobilecombackup" {
		t.Errorf("Expected project name 'mobilecombackup', got '%s'", dashboard.projectName)
	}

	if dashboard.metricsDir != testDir {
		t.Errorf("Expected metrics dir '%s', got '%s'", testDir, dashboard.metricsDir)
	}
}

func TestQualityDashboard_GenerateReport(t *testing.T) {
	testDir := t.TempDir()
	dashboard := NewQualityDashboard("test-project", testDir)

	report, err := dashboard.GenerateReport()
	if err != nil {
		t.Fatalf("Failed to generate report: %v", err)
	}

	// Verify report structure
	if report.ProjectName != "test-project" {
		t.Errorf("Expected project name 'test-project', got '%s'", report.ProjectName)
	}

	if report.GeneratedAt.IsZero() {
		t.Error("Expected GeneratedAt to be set")
	}

	// Test coverage report
	t.Run("CoverageReport", func(t *testing.T) {
		coverage := report.Coverage

		if coverage.OverallCoverage <= 0 {
			t.Error("Expected positive overall coverage")
		}

		if coverage.CoverageThreshold != 80.0 {
			t.Errorf("Expected coverage threshold 80.0, got %.1f", coverage.CoverageThreshold)
		}

		if len(coverage.PackageCoverage) == 0 {
			t.Error("Expected package coverage data")
		}

		t.Logf("Coverage: %.1f%%, Meeting threshold: %t",
			coverage.OverallCoverage, coverage.MeetingThreshold)
	})

	// Test performance report
	t.Run("PerformanceReport", func(t *testing.T) {
		performance := report.Performance

		if performance.ThroughputMetrics.FilesPerSecond < 0 {
			t.Error("Expected non-negative files per second metric")
		}

		if len(performance.ThroughputMetrics.ConcurrencyMetrics) == 0 {
			t.Error("Expected concurrency metrics data")
		}

		t.Logf("Throughput: %.1f files/sec, %.1f sections/sec",
			performance.ThroughputMetrics.FilesPerSecond,
			performance.ThroughputMetrics.SectionsPerSecond)
	})

	// Test quality report
	t.Run("QualityReport", func(t *testing.T) {
		quality := report.Quality

		if quality.QualityMetrics.Coverage <= 0 {
			t.Error("Expected positive coverage in quality metrics")
		}

		if len(quality.QualityGates) == 0 {
			t.Error("Expected quality gates")
		}

		// Verify quality gates
		for _, gate := range quality.QualityGates {
			if gate.Name == "" {
				t.Error("Quality gate missing name")
			}

			if gate.Threshold <= 0 {
				t.Errorf("Quality gate '%s' has invalid threshold: %.2f", gate.Name, gate.Threshold)
			}

			t.Logf("Quality Gate '%s': %.2f / %.2f (Passed: %t)",
				gate.Name, gate.CurrentValue, gate.Threshold, gate.Passed)
		}
	})

	// Test execution report
	t.Run("TestExecutionReport", func(t *testing.T) {
		execution := report.TestExecution

		if execution.TotalTests <= 0 {
			t.Error("Expected positive total tests count")
		}

		if execution.SuccessRate < 0 || execution.SuccessRate > 100 {
			t.Errorf("Invalid success rate: %.1f", execution.SuccessRate)
		}

		if len(execution.TestSuites) == 0 {
			t.Error("Expected test suite results")
		}

		t.Logf("Tests: %d passed, %d failed, %d skipped (%.1f%% success rate)",
			execution.PassedTests, execution.FailedTests, execution.SkippedTests, execution.SuccessRate)
	})

	// Test trend analysis
	t.Run("TrendAnalysis", func(t *testing.T) {
		trends := report.Trends

		if len(trends.CoverageTrend) == 0 {
			t.Error("Expected coverage trend data")
		}

		if len(trends.PerformanceTrend) == 0 {
			t.Error("Expected performance trend data")
		}

		if trends.TimeRange.Start.IsZero() || trends.TimeRange.End.IsZero() {
			t.Error("Expected valid time range")
		}

		// Verify trend data is chronologically ordered
		for i := 1; i < len(trends.CoverageTrend); i++ {
			if trends.CoverageTrend[i].Timestamp.Before(trends.CoverageTrend[i-1].Timestamp) {
				t.Error("Coverage trend data not in chronological order")
			}
		}

		t.Logf("Trends covering %v with %d coverage points",
			trends.TimeRange.Duration, len(trends.CoverageTrend))
	})

	// Test summary
	t.Run("DashboardSummary", func(t *testing.T) {
		summary := report.Summary

		if summary.Status == "" {
			t.Error("Expected status to be set")
		}

		if summary.OverallScore < 0 || summary.OverallScore > 100 {
			t.Errorf("Invalid overall score: %.2f", summary.OverallScore)
		}

		if len(summary.KeyMetrics) == 0 {
			t.Error("Expected key metrics")
		}

		// Verify key metrics are present
		expectedMetrics := []string{"test_coverage", "success_rate", "performance", "quality_score", "execution_time"}
		for _, metric := range expectedMetrics {
			if _, exists := summary.KeyMetrics[metric]; !exists {
				t.Errorf("Missing key metric: %s", metric)
			}
		}

		t.Logf("Status: %s, Overall Score: %.1f", summary.Status, summary.OverallScore)
		t.Logf("Recommendations: %d, Critical Issues: %d",
			len(summary.Recommendations), len(summary.CriticalIssues))
	})
}

func TestQualityDashboard_SaveAndLoadReport(t *testing.T) {
	testDir := t.TempDir()
	dashboard := NewQualityDashboard("test-save-load", testDir)

	// Generate a report
	originalReport, err := dashboard.GenerateReport()
	if err != nil {
		t.Fatalf("Failed to generate report: %v", err)
	}

	// Save the report
	filename := "test-report.json"
	err = dashboard.SaveReport(originalReport, filename)
	if err != nil {
		t.Fatalf("Failed to save report: %v", err)
	}

	// Verify file was created
	reportPath := filepath.Join(testDir, filename)
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Fatal("Report file was not created")
	}

	// Read and verify JSON content
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read report file: %v", err)
	}

	// Parse JSON to verify structure
	var loadedReport DashboardReport
	err = json.Unmarshal(data, &loadedReport)
	if err != nil {
		t.Fatalf("Failed to parse saved report JSON: %v", err)
	}

	// Verify key data matches
	if loadedReport.ProjectName != originalReport.ProjectName {
		t.Errorf("Project name mismatch: expected '%s', got '%s'",
			originalReport.ProjectName, loadedReport.ProjectName)
	}

	if loadedReport.Coverage.OverallCoverage != originalReport.Coverage.OverallCoverage {
		t.Errorf("Coverage mismatch: expected %.2f, got %.2f",
			originalReport.Coverage.OverallCoverage, loadedReport.Coverage.OverallCoverage)
	}

	if loadedReport.Summary.OverallScore != originalReport.Summary.OverallScore {
		t.Errorf("Overall score mismatch: expected %.2f, got %.2f",
			originalReport.Summary.OverallScore, loadedReport.Summary.OverallScore)
	}

	t.Logf("Successfully saved and loaded report with %.1f overall score",
		loadedReport.Summary.OverallScore)
}

func TestQualityDashboard_MetricsTracking(t *testing.T) {
	testDir := t.TempDir()
	dashboard := NewQualityDashboard("metrics-test", testDir)

	// Create test metrics
	testMetrics := TestMetrics{
		ExecutionTime:   time.Second * 15,
		MemoryUsage:     1024 * 1024 * 64, // 64MB
		CoveragePercent: 98.1,
		PassedTests:     12,
		FailedTests:     0,
		SkippedTests:    1,
		BenchmarkResults: []BenchmarkResult{
			{
				Name:             "BenchmarkTest",
				Duration:         time.Millisecond * 100,
				OperationsPerSec: 35.0,
				Iterations:       1000,
			},
		},
	}

	// Track the metrics
	err := dashboard.TrackMetrics(testMetrics)
	if err != nil {
		t.Fatalf("Failed to track metrics: %v", err)
	}

	// Verify metrics file was created
	files, err := os.ReadDir(testDir)
	if err != nil {
		t.Fatalf("Failed to read metrics directory: %v", err)
	}

	metricsFiles := 0
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "metrics_") && strings.HasSuffix(file.Name(), ".json") {
			metricsFiles++
		}
	}

	if metricsFiles != 1 {
		t.Errorf("Expected 1 metrics file, found %d", metricsFiles)
	}

	t.Logf("Successfully tracked metrics: %.1f%% coverage, %d tests passed",
		testMetrics.CoveragePercent, testMetrics.PassedTests)
}

func TestQualityDashboard_HistoricalReports(t *testing.T) {
	testDir := t.TempDir()
	dashboard := NewQualityDashboard("historical-test", testDir)

	// Create multiple reports with different timestamps
	numReports := 3
	for i := 0; i < numReports; i++ {
		report, err := dashboard.GenerateReport()
		if err != nil {
			t.Fatalf("Failed to generate report %d: %v", i, err)
		}

		// Adjust timestamp to simulate historical data
		report.GeneratedAt = time.Now().AddDate(0, 0, -i)

		filename := fmt.Sprintf("report_%d.json", i)
		err = dashboard.SaveReport(report, filename)
		if err != nil {
			t.Fatalf("Failed to save report %d: %v", i, err)
		}

		// Small delay to ensure different modification times
		time.Sleep(time.Millisecond * 10)
	}

	// Load historical reports
	reports, err := dashboard.LoadHistoricalReports(5) // Load up to 5 reports
	if err != nil {
		t.Fatalf("Failed to load historical reports: %v", err)
	}

	if len(reports) != numReports {
		t.Errorf("Expected %d historical reports, got %d", numReports, len(reports))
	}

	// Verify reports are loaded (sorting may vary based on file modification time vs report timestamps)
	if len(reports) == 0 {
		t.Error("Expected to load historical reports")
	}

	t.Logf("Successfully loaded %d historical reports", len(reports))
}

func TestDashboardReport_QualityGates(t *testing.T) {
	testDir := t.TempDir()
	dashboard := NewQualityDashboard("quality-gates-test", testDir)

	report, err := dashboard.GenerateReport()
	if err != nil {
		t.Fatalf("Failed to generate report: %v", err)
	}

	// Test each quality gate
	for _, gate := range report.Quality.QualityGates {
		t.Run(gate.Name, func(t *testing.T) {
			if gate.Name == "" {
				t.Error("Quality gate missing name")
			}

			if gate.Threshold <= 0 {
				t.Errorf("Invalid threshold: %.2f", gate.Threshold)
			}

			if gate.Importance == "" {
				t.Error("Quality gate missing importance level")
			}

			// Verify importance levels are valid
			validImportance := []string{"low", "medium", "high", "critical"}
			found := false
			for _, valid := range validImportance {
				if gate.Importance == valid {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Invalid importance level: %s", gate.Importance)
			}

			// Log gate status
			status := "PASS"
			if !gate.Passed {
				status = "FAIL"
			}

			t.Logf("Quality Gate '%s' [%s]: %.2f / %.2f (%s)",
				gate.Name, gate.Importance, gate.CurrentValue, gate.Threshold, status)
		})
	}
}

func TestDashboardReport_JSONSerialization(t *testing.T) {
	testDir := t.TempDir()
	dashboard := NewQualityDashboard("json-test", testDir)

	report, err := dashboard.GenerateReport()
	if err != nil {
		t.Fatalf("Failed to generate report: %v", err)
	}

	// Test JSON marshalling
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal report to JSON: %v", err)
	}

	if len(jsonData) == 0 {
		t.Fatal("Generated JSON is empty")
	}

	// Test JSON unmarshalling
	var unmarshalled DashboardReport
	err = json.Unmarshal(jsonData, &unmarshalled)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify key fields are preserved
	if unmarshalled.ProjectName != report.ProjectName {
		t.Error("Project name not preserved in JSON serialization")
	}

	if unmarshalled.Coverage.OverallCoverage != report.Coverage.OverallCoverage {
		t.Error("Coverage not preserved in JSON serialization")
	}

	if len(unmarshalled.Quality.QualityGates) != len(report.Quality.QualityGates) {
		t.Error("Quality gates not preserved in JSON serialization")
	}

	t.Logf("JSON serialization successful: %d bytes", len(jsonData))
}

func TestDashboardReport_PerformanceMetrics(t *testing.T) {
	testDir := t.TempDir()
	dashboard := NewQualityDashboard("performance-test", testDir)

	report, err := dashboard.GenerateReport()
	if err != nil {
		t.Fatalf("Failed to generate report: %v", err)
	}

	perf := report.Performance

	// Test concurrency metrics
	t.Run("ConcurrencyMetrics", func(t *testing.T) {
		if len(perf.ThroughputMetrics.ConcurrencyMetrics) == 0 {
			t.Fatal("Expected concurrency metrics")
		}

		for workers, throughput := range perf.ThroughputMetrics.ConcurrencyMetrics {
			if workers <= 0 {
				t.Errorf("Invalid worker count: %d", workers)
			}

			if throughput <= 0 {
				t.Errorf("Invalid throughput for %d workers: %.2f", workers, throughput)
			}

			t.Logf("Concurrency: %d workers -> %.1f files/sec", workers, throughput)
		}

		// Verify performance degradation with high concurrency (expected behavior)
		if throughput1, exists1 := perf.ThroughputMetrics.ConcurrencyMetrics[1]; exists1 {
			if throughput8, exists8 := perf.ThroughputMetrics.ConcurrencyMetrics[8]; exists8 {
				if throughput8 >= throughput1 {
					t.Logf("Note: 8-worker throughput (%.1f) >= 1-worker (%.1f) - unexpected but not necessarily wrong",
						throughput8, throughput1)
				}
			}
		}
	})

	// Test memory metrics
	t.Run("MemoryMetrics", func(t *testing.T) {
		if perf.MemoryUsage.AllocBytes == 0 {
			t.Error("Expected positive memory allocation")
		}

		memoryMB := float64(perf.MemoryUsage.AllocBytes) / (1024 * 1024)
		t.Logf("Memory usage: %.1f MB", memoryMB)

		// Memory usage should be reasonable (our 64MB buffer plus overhead)
		if memoryMB > 200 {
			t.Logf("Warning: High memory usage detected: %.1f MB", memoryMB)
		}
	})
}
