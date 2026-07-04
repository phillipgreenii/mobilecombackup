// dashboard_demo.go provides a demonstration of the quality dashboard
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/analyzer/core"
)

func main() {
	// Create dashboard instance
	dashboard := core.NewQualityDashboard("mobilecombackup-analyzer", "./dashboard-metrics")

	fmt.Println("🔍 Generating Quality Dashboard Report...")
	fmt.Println("=====================================")

	// Generate comprehensive report
	report, err := dashboard.GenerateReport()
	if err != nil {
		fmt.Printf("❌ Failed to generate report: %v\n", err)
		os.Exit(1)
	}

	// Display summary information
	displaySummary(report)

	// Display detailed metrics
	displayDetailedMetrics(report)

	// Save report to file
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("quality_report_%s.json", timestamp)

	fmt.Printf("\n💾 Saving report to %s...\n", filename)
	if err := dashboard.SaveReport(report, filename); err != nil {
		fmt.Printf("❌ Failed to save report: %v\n", err)
	} else {
		fmt.Printf("✅ Report saved successfully\n")
	}

	// Track current metrics for historical analysis
	metrics := core.TestMetrics{
		ExecutionTime:   report.TestExecution.ExecutionTime,
		MemoryUsage:     int64(report.Performance.MemoryUsage.AllocBytes), //nolint:gosec
		CoveragePercent: report.Coverage.OverallCoverage,
		PassedTests:     report.TestExecution.PassedTests,
		FailedTests:     report.TestExecution.FailedTests,
		SkippedTests:    report.TestExecution.SkippedTests,
	}

	fmt.Println("\n📊 Tracking metrics for trend analysis...")
	if err := dashboard.TrackMetrics(metrics); err != nil {
		fmt.Printf("❌ Failed to track metrics: %v\n", err)
	} else {
		fmt.Printf("✅ Metrics tracked successfully\n")
	}

	fmt.Println("\n🎯 Dashboard generation completed!")
}

func displaySummary(report *core.DashboardReport) {
	summary := report.Summary

	fmt.Printf("\n📋 PROJECT SUMMARY\n")
	fmt.Printf("==================\n")
	fmt.Printf("Project:      %s\n", report.ProjectName)
	fmt.Printf("Generated:    %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Status:       %s\n", getStatusEmoji(summary.Status))
	fmt.Printf("Overall Score: %.1f/100\n", summary.OverallScore)

	fmt.Printf("\n📈 KEY METRICS\n")
	fmt.Printf("==============\n")
	for metric, value := range summary.KeyMetrics {
		switch v := value.(type) {
		case float64:
			if metric == "execution_time" {
				fmt.Printf("%-18s: %.2fs\n", formatMetricName(metric), v)
			} else {
				fmt.Printf("%-18s: %.2f\n", formatMetricName(metric), v)
			}
		case int:
			fmt.Printf("%-18s: %d\n", formatMetricName(metric), v)
		default:
			fmt.Printf("%-18s: %v\n", formatMetricName(metric), v)
		}
	}

	if len(summary.CriticalIssues) > 0 {
		fmt.Printf("\n🚨 CRITICAL ISSUES\n")
		fmt.Printf("==================\n")
		for _, issue := range summary.CriticalIssues {
			fmt.Printf("• %s\n", issue)
		}
	}

	if len(summary.Recommendations) > 0 {
		fmt.Printf("\n💡 RECOMMENDATIONS\n")
		fmt.Printf("==================\n")
		for _, rec := range summary.Recommendations {
			fmt.Printf("• %s\n", rec)
		}
	}
}

func displayDetailedMetrics(report *core.DashboardReport) {
	fmt.Printf("\n🔬 DETAILED METRICS\n")
	fmt.Printf("===================\n")

	// Coverage Details
	fmt.Printf("\n📊 Test Coverage\n")
	fmt.Printf("Overall Coverage: %.1f%% (Threshold: %.1f%%)\n",
		report.Coverage.OverallCoverage, report.Coverage.CoverageThreshold)
	fmt.Printf("Meeting Threshold: %s\n", getBoolEmoji(report.Coverage.MeetingThreshold))

	// Performance Details
	fmt.Printf("\n⚡ Performance Metrics\n")
	perf := report.Performance.ThroughputMetrics
	fmt.Printf("Files/Second:     %.1f\n", perf.FilesPerSecond)
	fmt.Printf("Sections/Second:  %.1f\n", perf.SectionsPerSecond)
	fmt.Printf("Processing Delay: %v\n", perf.ProcessingLatency)

	// Concurrency Performance
	if len(perf.ConcurrencyMetrics) > 0 {
		fmt.Printf("\nConcurrency Performance:\n")
		for workers, throughput := range perf.ConcurrencyMetrics {
			fmt.Printf("  %d workers: %.1f files/sec\n", workers, throughput)
		}
	}

	// Quality Gates
	fmt.Printf("\n🎯 Quality Gates\n")
	for _, gate := range report.Quality.QualityGates {
		status := "❌"
		if gate.Passed {
			status = "✅"
		}
		fmt.Printf("%s %-20s: %.2f / %.2f (%s priority)\n",
			status, gate.Name, gate.CurrentValue, gate.Threshold, gate.Importance)
	}

	// Technical Debt
	debt := report.Quality.TechnicalDebt
	fmt.Printf("\n🔧 Technical Debt\n")
	fmt.Printf("Total Debt:       %d minutes\n", debt.TotalDebtMinutes)
	if len(debt.DebtByType) > 0 {
		fmt.Printf("Debt Breakdown:\n")
		for debtType, minutes := range debt.DebtByType {
			fmt.Printf("  %s: %d minutes\n", debtType, minutes)
		}
	}

	// Test Execution
	exec := report.TestExecution
	fmt.Printf("\n🧪 Test Execution\n")
	fmt.Printf("Total Tests:      %d\n", exec.TotalTests)
	fmt.Printf("Passed:           %d\n", exec.PassedTests)
	fmt.Printf("Failed:           %d\n", exec.FailedTests)
	fmt.Printf("Skipped:          %d\n", exec.SkippedTests)
	fmt.Printf("Success Rate:     %.1f%%\n", exec.SuccessRate)
	fmt.Printf("Execution Time:   %v\n", exec.ExecutionTime)

	// Trend Analysis
	trends := report.Trends
	if len(trends.CoverageTrend) > 0 {
		fmt.Printf("\n📈 Trend Analysis (%v)\n", trends.TimeRange.Duration)
		fmt.Printf("Coverage Trend:   ")
		displayTrend(trends.CoverageTrend)

		fmt.Printf("Performance Trend:")
		displayTrend(trends.PerformanceTrend)

		fmt.Printf("Quality Trend:    ")
		displayTrend(trends.QualityTrend)
	}
}

func displayTrend(points []core.TrendPoint) {
	if len(points) < 2 {
		fmt.Printf("Insufficient data\n")
		return
	}

	start := points[0].Value
	end := points[len(points)-1].Value
	change := end - start
	changePercent := (change / start) * 100

	var arrow string
	switch {
	case change > 0:
		arrow = "📈"
	case change < 0:
		arrow = "📉"
	default:
		arrow = "➡️"
	}

	fmt.Printf(" %.1f → %.1f (%+.1f%%) %s\n", start, end, changePercent, arrow)
}

func getStatusEmoji(status string) string {
	switch status {
	case "excellent":
		return "🟢 " + status
	case "good":
		return "🟡 " + status
	case "needs_attention":
		return "🟠 " + status
	case "critical":
		return "🔴 " + status
	default:
		return "⚪ " + status
	}
}

func getBoolEmoji(value bool) string {
	if value {
		return "✅ Yes"
	}
	return "❌ No"
}

func formatMetricName(name string) string {
	switch name {
	case "test_coverage":
		return "Test Coverage"
	case "success_rate":
		return "Success Rate"
	case "performance":
		return "Performance"
	case "quality_score":
		return "Quality Score"
	case "execution_time":
		return "Execution Time"
	default:
		return name
	}
}
