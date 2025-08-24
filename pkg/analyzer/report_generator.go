package analyzer

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/types"
)

// DefaultReportGenerator implements report generation for analysis results
type DefaultReportGenerator struct {
	// Future: Add configuration options for report formatting
}

// NewDefaultReportGenerator creates a new default report generator
func NewDefaultReportGenerator() *DefaultReportGenerator {
	return &DefaultReportGenerator{}
}

// GenerateTextReport generates a human-readable text report
func (drg *DefaultReportGenerator) GenerateTextReport(result *AnalysisResult) types.Result[string] {
	var report strings.Builder

	// Header
	report.WriteString("DOCUMENTATION ANALYSIS REPORT\n")
	report.WriteString("=============================\n\n")

	// Summary
	report.WriteString(fmt.Sprintf("Project: %s\n", result.ProjectPath))
	report.WriteString(fmt.Sprintf("Analysis ID: %s\n", result.ID))
	report.WriteString(fmt.Sprintf("Timestamp: %s\n", time.Unix(result.Timestamp/1000, 0).Format(time.RFC3339)))
	report.WriteString(fmt.Sprintf("Duration: %v\n", result.Duration))
	report.WriteString(fmt.Sprintf("Quality Score: %.2f (%s)\n", result.QualityScore, result.Summary.QualityGrade))
	report.WriteString("\n")

	// Files analyzed
	report.WriteString("FILES ANALYZED\n")
	report.WriteString("--------------\n")
	report.WriteString(fmt.Sprintf("Total Files: %d\n", result.TotalFiles))
	report.WriteString(fmt.Sprintf("Analyzed: %d\n", result.AnalyzedFiles))
	report.WriteString("\n")

	// Coverage
	report.WriteString("DOCUMENTATION COVERAGE\n")
	report.WriteString("----------------------\n")
	report.WriteString(fmt.Sprintf("Overall: %.1f%% (%d/%d symbols)\n",
		result.Coverage.CoveragePercent,
		result.Coverage.DocumentedSymbols,
		result.Coverage.TotalSymbols))

	if len(result.Coverage.ByType) > 0 {
		report.WriteString("\nBy Symbol Type:\n")
		for symbolType, coverage := range result.Coverage.ByType {
			report.WriteString(fmt.Sprintf("  %-12s: %.1f%% (%d/%d)\n",
				symbolType, coverage.Percent, coverage.Documented, coverage.Total))
		}
	}
	report.WriteString("\n")

	// Inconsistencies summary
	report.WriteString("INCONSISTENCIES SUMMARY\n")
	report.WriteString("-----------------------\n")
	report.WriteString(fmt.Sprintf("Total Issues: %d\n", result.Summary.TotalInconsistencies))

	if len(result.Summary.BySeverity) > 0 {
		report.WriteString("By Severity:\n")
		severityOrder := []SeverityLevel{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow}
		for _, severity := range severityOrder {
			if count, exists := result.Summary.BySeverity[severity]; exists && count > 0 {
				report.WriteString(fmt.Sprintf("  %-10s: %d\n", string(severity), count))
			}
		}
	}

	if len(result.Summary.ByType) > 0 {
		report.WriteString("\nBy Type:\n")
		for incType, count := range result.Summary.ByType {
			if count > 0 {
				report.WriteString(fmt.Sprintf("  %-20s: %d\n", strings.ReplaceAll(string(incType), "_", " "), count))
			}
		}
	}
	report.WriteString("\n")

	// Detailed inconsistencies
	if len(result.Inconsistencies) > 0 {
		report.WriteString("DETAILED INCONSISTENCIES\n")
		report.WriteString("------------------------\n")

		// Group by severity
		bySeverity := make(map[SeverityLevel][]Inconsistency)
		for _, inc := range result.Inconsistencies {
			bySeverity[inc.Severity] = append(bySeverity[inc.Severity], inc)
		}

		severityOrder := []SeverityLevel{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow}
		for _, severity := range severityOrder {
			if issues, exists := bySeverity[severity]; exists && len(issues) > 0 {
				report.WriteString(fmt.Sprintf("\n%s SEVERITY (%d issues):\n", strings.ToUpper(string(severity)), len(issues)))
				report.WriteString(strings.Repeat("-", len(string(severity))+20) + "\n")

				for i, inc := range issues {
					report.WriteString(fmt.Sprintf("\n%d. %s\n", i+1, inc.Title))
					report.WriteString(fmt.Sprintf("   Type: %s\n", strings.ReplaceAll(string(inc.Type), "_", " ")))
					report.WriteString(fmt.Sprintf("   Description: %s\n", inc.Description))

					if inc.CodeFile != "" {
						report.WriteString(fmt.Sprintf("   Code: %s", inc.CodeFile))
						if inc.CodeLine > 0 {
							report.WriteString(fmt.Sprintf(":%d", inc.CodeLine))
						}
						report.WriteString("\n")
					}

					if inc.DocFile != "" {
						report.WriteString(fmt.Sprintf("   Doc: %s", inc.DocFile))
						if inc.DocLine > 0 {
							report.WriteString(fmt.Sprintf(":%d", inc.DocLine))
						}
						report.WriteString("\n")
					}

					if inc.Expected != "" && inc.Actual != "" {
						report.WriteString(fmt.Sprintf("   Expected: %s\n", inc.Expected))
						report.WriteString(fmt.Sprintf("   Actual: %s\n", inc.Actual))
					}

					if inc.Suggestion != "" {
						report.WriteString(fmt.Sprintf("   Suggestion: %s\n", inc.Suggestion))
					}
				}
			}
		}
	}

	// Recommendations
	if len(result.Summary.Recommendations) > 0 {
		report.WriteString("\nRECOMMendations\n")
		report.WriteString("---------------\n")

		for i, rec := range result.Summary.Recommendations {
			report.WriteString(fmt.Sprintf("\n%d. [%s] %s\n", i+1, strings.ToUpper(rec.Priority), rec.Title))
			report.WriteString(fmt.Sprintf("   %s\n", rec.Description))
			report.WriteString(fmt.Sprintf("   Action: %s\n", rec.Action))
			report.WriteString(fmt.Sprintf("   Effort: %s | Impact: %s\n", rec.EstimatedEffort, rec.Impact))
		}
	}

	// Improvement areas
	if len(result.Summary.ImprovementAreas) > 0 {
		report.WriteString("\nKEY IMPROVEMENT AREAS\n")
		report.WriteString("---------------------\n")
		for i, area := range result.Summary.ImprovementAreas {
			report.WriteString(fmt.Sprintf("%d. %s\n", i+1, area))
		}
	}

	return types.NewResult(report.String())
}

// GenerateJSONReport generates a JSON report
func (drg *DefaultReportGenerator) GenerateJSONReport(result *AnalysisResult) types.Result[string] {
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return types.NewResultError[string](fmt.Errorf("failed to marshal analysis result to JSON: %w", err))
	}

	return types.NewResult(string(jsonData))
}

// GenerateMarkdownReport generates a markdown report
func (drg *DefaultReportGenerator) GenerateMarkdownReport(result *AnalysisResult) types.Result[string] {
	var report strings.Builder

	// Header
	report.WriteString("# Documentation Analysis Report\n\n")

	// Summary table
	report.WriteString("## Summary\n\n")
	report.WriteString("| Metric | Value |\n")
	report.WriteString("|--------|-------|\n")
	report.WriteString(fmt.Sprintf("| Project | %s |\n", result.ProjectPath))
	report.WriteString(fmt.Sprintf("| Analysis ID | `%s` |\n", result.ID))
	report.WriteString(fmt.Sprintf("| Timestamp | %s |\n", time.Unix(result.Timestamp/1000, 0).Format(time.RFC3339)))
	report.WriteString(fmt.Sprintf("| Duration | %v |\n", result.Duration))
	report.WriteString(fmt.Sprintf("| Quality Score | **%.2f** (%s) |\n", result.QualityScore, result.Summary.QualityGrade))
	report.WriteString(fmt.Sprintf("| Total Files | %d |\n", result.TotalFiles))
	report.WriteString(fmt.Sprintf("| Analyzed Files | %d |\n", result.AnalyzedFiles))
	report.WriteString(fmt.Sprintf("| Total Issues | **%d** |\n", result.Summary.TotalInconsistencies))
	report.WriteString("\n")

	// Coverage
	report.WriteString("## Documentation Coverage\n\n")
	report.WriteString(fmt.Sprintf("**Overall Coverage: %.1f%%** (%d/%d symbols)\n\n",
		result.Coverage.CoveragePercent,
		result.Coverage.DocumentedSymbols,
		result.Coverage.TotalSymbols))

	if len(result.Coverage.ByType) > 0 {
		report.WriteString("### Coverage by Symbol Type\n\n")
		report.WriteString("| Symbol Type | Coverage | Documented | Total |\n")
		report.WriteString("|-------------|----------|------------|-------|\n")

		// Sort by coverage percentage
		type typeStats struct {
			name     string
			coverage CoverageByType
		}
		var stats []typeStats
		for name, coverage := range result.Coverage.ByType {
			stats = append(stats, typeStats{name, coverage})
		}
		sort.Slice(stats, func(i, j int) bool {
			return stats[i].coverage.Percent > stats[j].coverage.Percent
		})

		for _, stat := range stats {
			report.WriteString(fmt.Sprintf("| %s | %.1f%% | %d | %d |\n",
				stat.name, stat.coverage.Percent, stat.coverage.Documented, stat.coverage.Total))
		}
		report.WriteString("\n")
	}

	// Issues by severity
	if len(result.Summary.BySeverity) > 0 {
		report.WriteString("## Issues by Severity\n\n")

		severityOrder := []SeverityLevel{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow}
		for _, severity := range severityOrder {
			if count, exists := result.Summary.BySeverity[severity]; exists && count > 0 {
				icon := drg.getSeverityIcon(severity)
				report.WriteString(fmt.Sprintf("- %s **%s**: %d issues\n", icon, strings.Title(string(severity)), count))
			}
		}
		report.WriteString("\n")
	}

	// Detailed issues
	if len(result.Inconsistencies) > 0 {
		report.WriteString("## Detailed Issues\n\n")

		// Group by severity
		bySeverity := make(map[SeverityLevel][]Inconsistency)
		for _, inc := range result.Inconsistencies {
			bySeverity[inc.Severity] = append(bySeverity[inc.Severity], inc)
		}

		severityOrder := []SeverityLevel{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow}
		for _, severity := range severityOrder {
			if issues, exists := bySeverity[severity]; exists && len(issues) > 0 {
				icon := drg.getSeverityIcon(severity)
				report.WriteString(fmt.Sprintf("### %s %s Severity\n\n", icon, strings.Title(string(severity))))

				for _, inc := range issues {
					report.WriteString(fmt.Sprintf("#### %s\n\n", inc.Title))
					report.WriteString(fmt.Sprintf("**Type**: %s  \n", strings.ReplaceAll(string(inc.Type), "_", " ")))
					report.WriteString(fmt.Sprintf("**Description**: %s\n\n", inc.Description))

					if inc.CodeFile != "" || inc.DocFile != "" {
						report.WriteString("**Location**:\n")
						if inc.CodeFile != "" {
							report.WriteString(fmt.Sprintf("- Code: `%s", inc.CodeFile))
							if inc.CodeLine > 0 {
								report.WriteString(fmt.Sprintf(":%d", inc.CodeLine))
							}
							report.WriteString("`\n")
						}
						if inc.DocFile != "" {
							report.WriteString(fmt.Sprintf("- Documentation: `%s", inc.DocFile))
							if inc.DocLine > 0 {
								report.WriteString(fmt.Sprintf(":%d", inc.DocLine))
							}
							report.WriteString("`\n")
						}
						report.WriteString("\n")
					}

					if inc.Expected != "" && inc.Actual != "" {
						report.WriteString("**Expected**:\n```\n")
						report.WriteString(inc.Expected)
						report.WriteString("\n```\n\n")
						report.WriteString("**Actual**:\n```\n")
						report.WriteString(inc.Actual)
						report.WriteString("\n```\n\n")
					}

					if inc.Suggestion != "" {
						report.WriteString(fmt.Sprintf("💡 **Suggestion**: %s\n\n", inc.Suggestion))
					}

					report.WriteString("---\n\n")
				}
			}
		}
	}

	// Recommendations
	if len(result.Summary.Recommendations) > 0 {
		report.WriteString("## Recommendations\n\n")

		for _, rec := range result.Summary.Recommendations {
			priority := drg.getPriorityIcon(rec.Priority)
			report.WriteString(fmt.Sprintf("### %s %s\n\n", priority, rec.Title))
			report.WriteString(fmt.Sprintf("%s\n\n", rec.Description))
			report.WriteString(fmt.Sprintf("**Action**: %s  \n", rec.Action))
			report.WriteString(fmt.Sprintf("**Estimated Effort**: %s  \n", rec.EstimatedEffort))
			report.WriteString(fmt.Sprintf("**Impact**: %s\n\n", rec.Impact))
		}
	}

	// Improvement areas
	if len(result.Summary.ImprovementAreas) > 0 {
		report.WriteString("## Key Improvement Areas\n\n")
		for _, area := range result.Summary.ImprovementAreas {
			report.WriteString(fmt.Sprintf("- %s\n", area))
		}
		report.WriteString("\n")
	}

	return types.NewResult(report.String())
}

// GenerateHTMLReport generates an HTML report
func (drg *DefaultReportGenerator) GenerateHTMLReport(result *AnalysisResult) types.Result[string] {
	var report strings.Builder

	// HTML header
	report.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Documentation Analysis Report</title>
    <style>
        body { 
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; 
            margin: 0; 
            padding: 20px; 
            background-color: #f5f5f5; 
        }
        .container { 
            max-width: 1200px; 
            margin: 0 auto; 
            background-color: white; 
            padding: 30px; 
            border-radius: 8px; 
            box-shadow: 0 2px 10px rgba(0,0,0,0.1); 
        }
        h1, h2, h3 { color: #333; margin-top: 2em; }
        h1 { border-bottom: 3px solid #007acc; padding-bottom: 10px; }
        h2 { border-bottom: 2px solid #eee; padding-bottom: 8px; }
        .summary-table { 
            width: 100%; 
            border-collapse: collapse; 
            margin: 20px 0; 
            background-color: #f9f9f9; 
        }
        .summary-table th, .summary-table td { 
            padding: 12px; 
            text-align: left; 
            border: 1px solid #ddd; 
        }
        .summary-table th { 
            background-color: #007acc; 
            color: white; 
            font-weight: bold; 
        }
        .coverage-bar {
            width: 100%;
            height: 20px;
            background-color: #eee;
            border-radius: 10px;
            overflow: hidden;
            margin: 5px 0;
        }
        .coverage-fill {
            height: 100%;
            transition: width 0.3s ease;
        }
        .coverage-excellent { background-color: #28a745; }
        .coverage-good { background-color: #17a2b8; }
        .coverage-fair { background-color: #ffc107; }
        .coverage-poor { background-color: #dc3545; }
        .severity-critical { color: #dc3545; font-weight: bold; }
        .severity-high { color: #fd7e14; font-weight: bold; }
        .severity-medium { color: #ffc107; font-weight: bold; }
        .severity-low { color: #28a745; }
        .issue-card {
            border: 1px solid #ddd;
            border-radius: 6px;
            margin: 15px 0;
            padding: 15px;
            background-color: #fff;
        }
        .issue-card h4 { margin-top: 0; color: #495057; }
        .code-block {
            background-color: #f8f9fa;
            border: 1px solid #e9ecef;
            border-radius: 4px;
            padding: 10px;
            font-family: 'Courier New', monospace;
            overflow-x: auto;
        }
        .recommendation {
            background-color: #e7f3ff;
            border-left: 4px solid #007acc;
            padding: 15px;
            margin: 15px 0;
        }
    </style>
</head>
<body>
    <div class="container">
`)

	// Title
	report.WriteString("<h1>📊 Documentation Analysis Report</h1>\n")

	// Summary table
	report.WriteString("<h2>Summary</h2>\n")
	report.WriteString(`<table class="summary-table">`)
	report.WriteString("<tr><th>Metric</th><th>Value</th></tr>\n")
	report.WriteString(fmt.Sprintf("<tr><td>Project</td><td>%s</td></tr>\n", result.ProjectPath))
	report.WriteString(fmt.Sprintf("<tr><td>Analysis ID</td><td><code>%s</code></td></tr>\n", result.ID))
	report.WriteString(fmt.Sprintf("<tr><td>Timestamp</td><td>%s</td></tr>\n", time.Unix(result.Timestamp/1000, 0).Format(time.RFC3339)))
	report.WriteString(fmt.Sprintf("<tr><td>Duration</td><td>%v</td></tr>\n", result.Duration))
	report.WriteString(fmt.Sprintf("<tr><td>Quality Score</td><td><strong>%.2f (%s)</strong></td></tr>\n", result.QualityScore, result.Summary.QualityGrade))
	report.WriteString(fmt.Sprintf("<tr><td>Total Issues</td><td><strong>%d</strong></td></tr>\n", result.Summary.TotalInconsistencies))
	report.WriteString("</table>\n")

	// Coverage
	report.WriteString("<h2>📈 Documentation Coverage</h2>\n")
	coverageClass := drg.getCoverageClass(result.Coverage.CoveragePercent)
	report.WriteString(fmt.Sprintf(`
        <p><strong>Overall Coverage: %.1f%%</strong> (%d/%d symbols)</p>
        <div class="coverage-bar">
            <div class="coverage-fill %s" style="width: %.1f%%"></div>
        </div>
    `, result.Coverage.CoveragePercent, result.Coverage.DocumentedSymbols, result.Coverage.TotalSymbols, coverageClass, result.Coverage.CoveragePercent))

	// Issues summary
	if len(result.Summary.BySeverity) > 0 {
		report.WriteString("<h2>⚠️ Issues Summary</h2>\n")
		report.WriteString("<ul>\n")

		severityOrder := []SeverityLevel{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow}
		for _, severity := range severityOrder {
			if count, exists := result.Summary.BySeverity[severity]; exists && count > 0 {
				report.WriteString(fmt.Sprintf(`<li class="severity-%s">%s: %d issues</li>`,
					string(severity), strings.Title(string(severity)), count))
				report.WriteString("\n")
			}
		}
		report.WriteString("</ul>\n")
	}

	// Detailed issues
	if len(result.Inconsistencies) > 0 {
		report.WriteString("<h2>🔍 Detailed Issues</h2>\n")

		for _, inc := range result.Inconsistencies {
			report.WriteString(`<div class="issue-card">`)
			report.WriteString(fmt.Sprintf(`<h4 class="severity-%s">%s</h4>`, string(inc.Severity), inc.Title))
			report.WriteString(fmt.Sprintf("<p><strong>Type:</strong> %s</p>\n", strings.ReplaceAll(string(inc.Type), "_", " ")))
			report.WriteString(fmt.Sprintf("<p><strong>Description:</strong> %s</p>\n", inc.Description))

			if inc.CodeFile != "" || inc.DocFile != "" {
				report.WriteString("<p><strong>Location:</strong></p><ul>\n")
				if inc.CodeFile != "" {
					location := inc.CodeFile
					if inc.CodeLine > 0 {
						location = fmt.Sprintf("%s:%d", location, inc.CodeLine)
					}
					report.WriteString(fmt.Sprintf("<li>Code: <code>%s</code></li>\n", location))
				}
				if inc.DocFile != "" {
					location := inc.DocFile
					if inc.DocLine > 0 {
						location = fmt.Sprintf("%s:%d", location, inc.DocLine)
					}
					report.WriteString(fmt.Sprintf("<li>Documentation: <code>%s</code></li>\n", location))
				}
				report.WriteString("</ul>\n")
			}

			if inc.Suggestion != "" {
				report.WriteString(fmt.Sprintf("<p><strong>💡 Suggestion:</strong> %s</p>\n", inc.Suggestion))
			}

			report.WriteString("</div>\n")
		}
	}

	// Recommendations
	if len(result.Summary.Recommendations) > 0 {
		report.WriteString("<h2>💡 Recommendations</h2>\n")

		for _, rec := range result.Summary.Recommendations {
			report.WriteString(`<div class="recommendation">`)
			report.WriteString(fmt.Sprintf("<h4>%s</h4>\n", rec.Title))
			report.WriteString(fmt.Sprintf("<p>%s</p>\n", rec.Description))
			report.WriteString(fmt.Sprintf("<p><strong>Action:</strong> %s</p>\n", rec.Action))
			report.WriteString(fmt.Sprintf("<p><strong>Effort:</strong> %s | <strong>Impact:</strong> %s</p>\n", rec.EstimatedEffort, rec.Impact))
			report.WriteString("</div>\n")
		}
	}

	// Close HTML
	report.WriteString(`
    </div>
</body>
</html>`)

	return types.NewResult(report.String())
}

// GenerateSummary generates a concise summary of the analysis
func (drg *DefaultReportGenerator) GenerateSummary(result *AnalysisResult) types.Result[string] {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("Analysis of %s completed with quality score %.2f (%s). ",
		result.ProjectPath, result.QualityScore, result.Summary.QualityGrade))

	summary.WriteString(fmt.Sprintf("Found %d inconsistencies across %d analyzed files. ",
		result.Summary.TotalInconsistencies, result.AnalyzedFiles))

	summary.WriteString(fmt.Sprintf("Documentation coverage is %.1f%% (%d/%d symbols). ",
		result.Coverage.CoveragePercent, result.Coverage.DocumentedSymbols, result.Coverage.TotalSymbols))

	// Add severity breakdown
	if len(result.Summary.BySeverity) > 0 {
		var severityParts []string
		severityOrder := []SeverityLevel{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow}
		for _, severity := range severityOrder {
			if count, exists := result.Summary.BySeverity[severity]; exists && count > 0 {
				severityParts = append(severityParts, fmt.Sprintf("%d %s", count, string(severity)))
			}
		}
		if len(severityParts) > 0 {
			summary.WriteString(fmt.Sprintf("Issues by severity: %s. ", strings.Join(severityParts, ", ")))
		}
	}

	// Add top recommendation
	if len(result.Summary.Recommendations) > 0 {
		summary.WriteString(fmt.Sprintf("Top recommendation: %s", result.Summary.Recommendations[0].Action))
	}

	return types.NewResult(summary.String())
}

// Helper methods

func (drg *DefaultReportGenerator) getSeverityIcon(severity SeverityLevel) string {
	switch severity {
	case SeverityCritical:
		return "🚨"
	case SeverityHigh:
		return "⚠️"
	case SeverityMedium:
		return "⚡"
	case SeverityLow:
		return "ℹ️"
	default:
		return "❓"
	}
}

func (drg *DefaultReportGenerator) getPriorityIcon(priority string) string {
	switch strings.ToLower(priority) {
	case "critical":
		return "🚨"
	case "high":
		return "🔴"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "📋"
	}
}

func (drg *DefaultReportGenerator) getCoverageClass(percent float64) string {
	if percent >= 90 {
		return "coverage-excellent"
	} else if percent >= 70 {
		return "coverage-good"
	} else if percent >= 50 {
		return "coverage-fair"
	}
	return "coverage-poor"
}
