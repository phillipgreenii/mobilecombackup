package validation

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ReportFormat specifies the output format for validation reports
type ReportFormat string

const (
	FormatYAML ReportFormat = "yaml"
	FormatJSON ReportFormat = "json"
	FormatText ReportFormat = "text"
)

// ReportFilterOptions controls which violations are included in the report
type ReportFilterOptions struct {
	// IncludeSeverities filters violations by severity level
	IncludeSeverities []Severity

	// IncludeTypes filters violations by violation type
	IncludeTypes []ViolationType

	// ExcludeFiles excludes violations from specific files (glob patterns supported)
	ExcludeFiles []string

	// MaxViolations limits the number of violations shown per type
	MaxViolations int
}

// ReportGenerator generates formatted validation reports
type ReportGenerator interface {
	// GenerateReport creates a formatted validation report
	GenerateReport(report *ValidationReport, format ReportFormat, options *ReportFilterOptions) (string, error)

	// GenerateSummary creates a brief summary of validation results
	GenerateSummary(report *ValidationReport) (*ValidationSummary, error)
}

// ValidationSummary provides high-level statistics about validation results
type ValidationSummary struct {
	TotalViolations      int                   `yaml:"total_violations" json:"total_violations"`
	ViolationsByType     map[ViolationType]int `yaml:"violations_by_type" json:"violations_by_type"`
	ViolationsBySeverity map[Severity]int      `yaml:"violations_by_severity" json:"violations_by_severity"`
	Status               ValidationStatus      `yaml:"status" json:"status"`
	Timestamp            time.Time             `yaml:"timestamp" json:"timestamp"`
	RepositoryPath       string                `yaml:"repository_path" json:"repository_path"`
}

// ReportGeneratorImpl implements ReportGenerator interface
type ReportGeneratorImpl struct{}

// NewReportGenerator creates a new validation report generator
func NewReportGenerator() ReportGenerator {
	return &ReportGeneratorImpl{}
}

// GenerateReport creates a formatted validation report
func (r *ReportGeneratorImpl) GenerateReport(report *ValidationReport, format ReportFormat, options *ReportFilterOptions) (string, error) {
	if report == nil {
		return "", fmt.Errorf("validation report cannot be nil")
	}

	// Apply filters to create a filtered report
	filteredReport := r.applyFilters(report, options)

	switch format {
	case FormatYAML:
		return r.generateYAMLReport(filteredReport)
	case FormatJSON:
		return r.generateJSONReport(filteredReport)
	case FormatText:
		return r.generateTextReport(filteredReport)
	default:
		return "", fmt.Errorf("unsupported report format: %s", format)
	}
}

// GenerateSummary creates a brief summary of validation results
func (r *ReportGeneratorImpl) GenerateSummary(report *ValidationReport) (*ValidationSummary, error) {
	if report == nil {
		return nil, fmt.Errorf("validation report cannot be nil")
	}

	summary := &ValidationSummary{
		TotalViolations:      len(report.Violations),
		ViolationsByType:     make(map[ViolationType]int),
		ViolationsBySeverity: make(map[Severity]int),
		Status:               report.Status,
		Timestamp:            report.Timestamp,
		RepositoryPath:       report.RepositoryPath,
	}

	// Count violations by type and severity
	for _, violation := range report.Violations {
		summary.ViolationsByType[violation.Type]++
		summary.ViolationsBySeverity[violation.Severity]++
	}

	return summary, nil
}

// applyFilters creates a filtered copy of the validation report
func (r *ReportGeneratorImpl) applyFilters(report *ValidationReport, options *ReportFilterOptions) *ValidationReport {
	if options == nil {
		return report
	}

	filteredReport := &ValidationReport{
		Timestamp:      report.Timestamp,
		RepositoryPath: report.RepositoryPath,
		Status:         report.Status,
		Violations:     []ValidationViolation{},
	}

	// Create severity filter map for efficient lookup
	severityFilter := make(map[Severity]bool)
	if len(options.IncludeSeverities) > 0 {
		for _, severity := range options.IncludeSeverities {
			severityFilter[severity] = true
		}
	}

	// Create type filter map for efficient lookup
	typeFilter := make(map[ViolationType]bool)
	if len(options.IncludeTypes) > 0 {
		for _, violationType := range options.IncludeTypes {
			typeFilter[violationType] = true
		}
	}

	// Track violations per type for maxViolations limit
	violationCounts := make(map[ViolationType]int)

	for _, violation := range report.Violations {
		// Apply severity filter
		if len(severityFilter) > 0 && !severityFilter[violation.Severity] {
			continue
		}

		// Apply type filter
		if len(typeFilter) > 0 && !typeFilter[violation.Type] {
			continue
		}

		// Apply file exclusion filter (simple string matching for now)
		if r.shouldExcludeFile(violation.File, options.ExcludeFiles) {
			continue
		}

		// Apply max violations per type limit
		if options.MaxViolations > 0 {
			if violationCounts[violation.Type] >= options.MaxViolations {
				continue
			}
			violationCounts[violation.Type]++
		}

		filteredReport.Violations = append(filteredReport.Violations, violation)
	}

	return filteredReport
}

// shouldExcludeFile checks if a file should be excluded based on exclusion patterns
func (r *ReportGeneratorImpl) shouldExcludeFile(file string, excludePatterns []string) bool {
	for _, pattern := range excludePatterns {
		// Simple string matching for now - could be extended to support glob patterns
		if strings.Contains(file, pattern) {
			return true
		}
	}
	return false
}

// generateYAMLReport creates a YAML-formatted validation report
func (r *ReportGeneratorImpl) generateYAMLReport(report *ValidationReport) (string, error) {
	yamlData, err := yaml.Marshal(report)
	if err != nil {
		return "", fmt.Errorf("failed to marshal report to YAML: %v", err)
	}
	return string(yamlData), nil
}

// generateJSONReport creates a JSON-formatted validation report
func (r *ReportGeneratorImpl) generateJSONReport(report *ValidationReport) (string, error) {
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal report to JSON: %v", err)
	}
	return string(jsonData), nil
}

// generateTextReport creates a human-readable text validation report
func (r *ReportGeneratorImpl) generateTextReport(report *ValidationReport) (string, error) {
	var builder strings.Builder

	// Header
	fmt.Fprintf(&builder, "Repository Validation Report\n")
	fmt.Fprintf(&builder, "===========================\n\n")
	fmt.Fprintf(&builder, "Repository: %s\n", report.RepositoryPath)
	fmt.Fprintf(&builder, "Timestamp:  %s\n", report.Timestamp.Format("2006-01-02 15:04:05 UTC"))
	fmt.Fprintf(&builder, "Status:     %s\n", strings.ToUpper(string(report.Status)))
	fmt.Fprintf(&builder, "Total Violations: %d\n\n", len(report.Violations))

	if len(report.Violations) == 0 {
		fmt.Fprintf(&builder, "✅ No validation violations found!\n")
		return builder.String(), nil
	}

	// Group violations by severity and type
	violationsByType := r.groupViolationsByType(report.Violations)
	violationsBySeverity := r.groupViolationsBySeverity(report.Violations)

	// Summary by severity
	fmt.Fprintf(&builder, "Summary by Severity:\n")
	fmt.Fprintf(&builder, "-------------------\n")
	severities := []Severity{SeverityError, SeverityWarning}
	for _, severity := range severities {
		count := len(violationsBySeverity[severity])
		if count > 0 {
			icon := "⚠️"
			if severity == SeverityError {
				icon = "❌"
			}
			fmt.Fprintf(&builder, "%s %s: %d\n", icon, strings.ToUpper(string(severity)), count)
		}
	}
	fmt.Fprintf(&builder, "\n")

	// Summary by type
	fmt.Fprintf(&builder, "Summary by Type:\n")
	fmt.Fprintf(&builder, "---------------\n")
	// Sort types for consistent output
	types := make([]ViolationType, 0, len(violationsByType))
	for violationType := range violationsByType {
		types = append(types, violationType)
	}
	sort.Slice(types, func(i, j int) bool {
		return string(types[i]) < string(types[j])
	})

	for _, violationType := range types {
		violations := violationsByType[violationType]
		if len(violations) > 0 {
			fmt.Fprintf(&builder, "  %s: %d\n", strings.ReplaceAll(string(violationType), "_", " "), len(violations))
		}
	}
	fmt.Fprintf(&builder, "\n")

	// Detailed violations
	fmt.Fprintf(&builder, "Detailed Violations:\n")
	fmt.Fprintf(&builder, "===================\n\n")

	// Show errors first, then warnings
	for _, severity := range severities {
		violations := violationsBySeverity[severity]
		if len(violations) == 0 {
			continue
		}

		icon := "⚠️"
		if severity == SeverityError {
			icon = "❌"
		}

		fmt.Fprintf(&builder, "%s %s (%d)\n", icon, strings.ToUpper(string(severity)), len(violations))
		fmt.Fprintf(&builder, "%s\n", strings.Repeat("-", len(string(severity))+6))

		// Group by type within severity
		typeGroups := r.groupViolationsByType(violations)
		for _, violationType := range types {
			typeViolations := typeGroups[violationType]
			if len(typeViolations) == 0 {
				continue
			}

			fmt.Fprintf(&builder, "\n%s:\n", strings.ReplaceAll(string(violationType), "_", " "))
			for _, violation := range typeViolations {
				fmt.Fprintf(&builder, "  • %s", violation.Message)
				if violation.File != "" {
					fmt.Fprintf(&builder, " (in %s)", violation.File)
				}
				if violation.Expected != "" || violation.Actual != "" {
					fmt.Fprintf(&builder, "\n    Expected: %s, Actual: %s", violation.Expected, violation.Actual)
				}
				fmt.Fprintf(&builder, "\n")
			}
		}
		fmt.Fprintf(&builder, "\n")
	}

	return builder.String(), nil
}

// groupViolationsByType groups violations by their type
func (r *ReportGeneratorImpl) groupViolationsByType(violations []ValidationViolation) map[ViolationType][]ValidationViolation {
	groups := make(map[ViolationType][]ValidationViolation)
	for _, violation := range violations {
		groups[violation.Type] = append(groups[violation.Type], violation)
	}
	return groups
}

// groupViolationsBySeverity groups violations by their severity
func (r *ReportGeneratorImpl) groupViolationsBySeverity(violations []ValidationViolation) map[Severity][]ValidationViolation {
	groups := make(map[Severity][]ValidationViolation)
	for _, violation := range violations {
		groups[violation.Severity] = append(groups[violation.Severity], violation)
	}
	return groups
}
