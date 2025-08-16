package validation

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestReportGeneratorImpl_GenerateReport_YAML(t *testing.T) {
	generator := NewReportGenerator()

	report := &Report{
		Timestamp:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		RepositoryPath: "/test/repo",
		Status:         Invalid,
		Violations: []ValidationViolation{
			{
				Type:     MissingFile,
				Severity: SeverityError,
				File:     "calls/calls-2024.xml",
				Message:  "Required file not found",
				Expected: "file should exist",
				Actual:   "file missing",
			},
			{
				Type:     InvalidFormat,
				Severity: SeverityWarning,
				File:     "contacts.yaml",
				Message:  "Phone number format issue",
			},
		},
	}

	result, err := generator.GenerateReport(report, FormatYAML, nil)
	if err != nil {
		t.Fatalf("Failed to generate YAML report: %v", err)
	}

	// Verify it's valid YAML by parsing it back
	var parsed Report
	err = yaml.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Fatalf("Generated YAML is invalid: %v", err)
	}

	// Verify key fields
	if parsed.Status != Invalid {
		t.Errorf("Expected status Invalid, got %s", parsed.Status)
	}
	if len(parsed.Violations) != 2 {
		t.Errorf("Expected 2 violations, got %d", len(parsed.Violations))
	}

	// Verify YAML contains expected content
	if !strings.Contains(result, "timestamp:") {
		t.Error("YAML should contain timestamp field")
	}
	if !strings.Contains(result, "violations:") {
		t.Error("YAML should contain violations field")
	}
}

func TestReportGeneratorImpl_GenerateReport_JSON(t *testing.T) {
	generator := NewReportGenerator()

	report := &Report{
		Timestamp:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		RepositoryPath: "/test/repo",
		Status:         Valid,
		Violations:     []ValidationViolation{},
	}

	result, err := generator.GenerateReport(report, FormatJSON, nil)
	if err != nil {
		t.Fatalf("Failed to generate JSON report: %v", err)
	}

	// Verify it's valid JSON by parsing it back
	var parsed Report
	err = json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Verify key fields
	if parsed.Status != Valid {
		t.Errorf("Expected status Valid, got %s", parsed.Status)
	}
	if len(parsed.Violations) != 0 {
		t.Errorf("Expected 0 violations, got %d", len(parsed.Violations))
	}

	// Verify JSON contains expected content
	if !strings.Contains(result, "\"Timestamp\"") {
		t.Error("JSON should contain Timestamp field")
	}
	if !strings.Contains(result, "\"Violations\"") {
		t.Error("JSON should contain Violations field")
	}
}

func TestReportGeneratorImpl_GenerateReport_Text(t *testing.T) {
	generator := NewReportGenerator()

	report := &Report{
		Timestamp:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		RepositoryPath: "/test/repo",
		Status:         Invalid,
		Violations: []ValidationViolation{
			{
				Type:     MissingFile,
				Severity: SeverityError,
				File:     "calls/calls-2024.xml",
				Message:  "Required file not found",
				Expected: "file should exist",
				Actual:   "file missing",
			},
			{
				Type:     InvalidFormat,
				Severity: SeverityWarning,
				File:     "contacts.yaml",
				Message:  "Phone number format issue",
			},
			{
				Type:     ChecksumMismatch,
				Severity: SeverityError,
				File:     "sms/sms-2024.xml",
				Message:  "File checksum doesn't match expected value",
			},
		},
	}

	result, err := generator.GenerateReport(report, FormatText, nil)
	if err != nil {
		t.Fatalf("Failed to generate text report: %v", err)
	}

	// Verify header content
	if !strings.Contains(result, "Repository Validation Report") {
		t.Error("Text report should contain header")
	}
	if !strings.Contains(result, "/test/repo") {
		t.Error("Text report should contain repository path")
	}
	if !strings.Contains(result, "INVALID") {
		t.Error("Text report should contain status")
	}
	if !strings.Contains(result, "Total Violations: 3") {
		t.Error("Text report should contain violation count")
	}

	// Verify summary sections
	if !strings.Contains(result, "Summary by Severity:") {
		t.Error("Text report should contain severity summary")
	}
	if !strings.Contains(result, "Summary by Type:") {
		t.Error("Text report should contain type summary")
	}
	if !strings.Contains(result, "Detailed Violations:") {
		t.Error("Text report should contain detailed violations")
	}

	// Verify severity indicators
	if !strings.Contains(result, "❌ ERROR: 2") {
		t.Error("Text report should show error count with icon")
	}
	if !strings.Contains(result, "⚠️ WARNING: 1") {
		t.Error("Text report should show warning count with icon")
	}

	// Verify violation details
	if !strings.Contains(result, "Required file not found") {
		t.Error("Text report should contain violation messages")
	}
	if !strings.Contains(result, "Expected: file should exist, Actual: file missing") {
		t.Error("Text report should contain expected/actual details")
	}
}

func TestReportGeneratorImpl_GenerateReport_Text_NoViolations(t *testing.T) {
	generator := NewReportGenerator()

	report := &Report{
		Timestamp:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		RepositoryPath: "/test/repo",
		Status:         Valid,
		Violations:     []ValidationViolation{},
	}

	result, err := generator.GenerateReport(report, FormatText, nil)
	if err != nil {
		t.Fatalf("Failed to generate text report: %v", err)
	}

	// Verify success message
	if !strings.Contains(result, "✅ No validation violations found!") {
		t.Error("Text report should contain success message when no violations")
	}
	if !strings.Contains(result, "Total Violations: 0") {
		t.Error("Text report should show 0 violations")
	}
}

func TestReportGeneratorImpl_GenerateReport_UnsupportedFormat(t *testing.T) {
	generator := NewReportGenerator()

	report := &Report{
		Timestamp:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		RepositoryPath: "/test/repo",
		Status:         Valid,
		Violations:     []ValidationViolation{},
	}

	_, err := generator.GenerateReport(report, ReportFormat("xml"), nil)
	if err == nil {
		t.Error("Expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported report format") {
		t.Errorf("Expected unsupported format error, got: %v", err)
	}
}

func TestReportGeneratorImpl_GenerateReport_NilReport(t *testing.T) {
	generator := NewReportGenerator()

	_, err := generator.GenerateReport(nil, FormatText, nil)
	if err == nil {
		t.Error("Expected error for nil report")
	}
	if !strings.Contains(err.Error(), "validation report cannot be nil") {
		t.Errorf("Expected nil report error, got: %v", err)
	}
}

func TestReportGeneratorImpl_GenerateSummary(t *testing.T) {
	generator := NewReportGenerator()

	report := &Report{
		Timestamp:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		RepositoryPath: "/test/repo",
		Status:         Invalid,
		Violations: []ValidationViolation{
			{Type: MissingFile, Severity: SeverityError},
			{Type: MissingFile, Severity: SeverityWarning},
			{Type: InvalidFormat, Severity: SeverityError},
			{Type: ChecksumMismatch, Severity: SeverityWarning},
			{Type: ChecksumMismatch, Severity: SeverityWarning},
		},
	}

	summary, err := generator.GenerateSummary(report)
	if err != nil {
		t.Fatalf("Failed to generate summary: %v", err)
	}

	// Verify basic fields
	if summary.TotalViolations != 5 {
		t.Errorf("Expected 5 total violations, got %d", summary.TotalViolations)
	}
	if summary.Status != Invalid {
		t.Errorf("Expected status Invalid, got %s", summary.Status)
	}
	if summary.RepositoryPath != "/test/repo" {
		t.Errorf("Expected repository path '/test/repo', got %s", summary.RepositoryPath)
	}

	// Verify violations by type
	expectedByType := map[ViolationType]int{
		MissingFile:      2,
		InvalidFormat:    1,
		ChecksumMismatch: 2,
	}
	for violationType, expected := range expectedByType {
		if actual := summary.ViolationsByType[violationType]; actual != expected {
			t.Errorf("Expected %d violations of type %s, got %d", expected, violationType, actual)
		}
	}

	// Verify violations by severity
	expectedBySeverity := map[Severity]int{
		SeverityError:   2,
		SeverityWarning: 3,
	}
	for severity, expected := range expectedBySeverity {
		if actual := summary.ViolationsBySeverity[severity]; actual != expected {
			t.Errorf("Expected %d violations of severity %s, got %d", expected, severity, actual)
		}
	}
}

func TestReportGeneratorImpl_GenerateSummary_NilReport(t *testing.T) {
	generator := NewReportGenerator()

	_, err := generator.GenerateSummary(nil)
	if err == nil {
		t.Error("Expected error for nil report")
	}
	if !strings.Contains(err.Error(), "validation report cannot be nil") {
		t.Errorf("Expected nil report error, got: %v", err)
	}
}

func TestReportGeneratorImpl_ApplyFilters_SeverityFilter(t *testing.T) {
	generator := &ReportGeneratorImpl{}

	report := &Report{
		Timestamp:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		RepositoryPath: "/test/repo",
		Status:         Invalid,
		Violations: []ValidationViolation{
			{Type: MissingFile, Severity: SeverityError, Message: "Error 1"},
			{Type: InvalidFormat, Severity: SeverityWarning, Message: "Warning 1"},
			{Type: ChecksumMismatch, Severity: SeverityError, Message: "Error 2"},
			{Type: OrphanedAttachment, Severity: SeverityWarning, Message: "Warning 2"},
		},
	}

	options := &ReportFilterOptions{
		IncludeSeverities: []Severity{SeverityError},
	}

	filtered := generator.applyFilters(report, options)

	if len(filtered.Violations) != 2 {
		t.Errorf("Expected 2 violations after severity filter, got %d", len(filtered.Violations))
	}

	// Verify only errors remain
	for _, violation := range filtered.Violations {
		if violation.Severity != SeverityError {
			t.Errorf("Expected only error violations, got %s", violation.Severity)
		}
	}
}

func TestReportGeneratorImpl_ApplyFilters_TypeFilter(t *testing.T) {
	generator := &ReportGeneratorImpl{}

	report := &Report{
		Timestamp:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		RepositoryPath: "/test/repo",
		Status:         Invalid,
		Violations: []ValidationViolation{
			{Type: MissingFile, Severity: SeverityError, Message: "Missing 1"},
			{Type: InvalidFormat, Severity: SeverityWarning, Message: "Format 1"},
			{Type: MissingFile, Severity: SeverityWarning, Message: "Missing 2"},
			{Type: ChecksumMismatch, Severity: SeverityError, Message: "Checksum 1"},
		},
	}

	options := &ReportFilterOptions{
		IncludeTypes: []ViolationType{MissingFile, ChecksumMismatch},
	}

	filtered := generator.applyFilters(report, options)

	if len(filtered.Violations) != 3 {
		t.Errorf("Expected 3 violations after type filter, got %d", len(filtered.Violations))
	}

	// Verify only specified types remain
	allowedTypes := map[ViolationType]bool{MissingFile: true, ChecksumMismatch: true}
	for _, violation := range filtered.Violations {
		if !allowedTypes[violation.Type] {
			t.Errorf("Unexpected violation type %s after filtering", violation.Type)
		}
	}
}

func TestReportGeneratorImpl_ApplyFilters_FileExcludeFilter(t *testing.T) {
	generator := &ReportGeneratorImpl{}

	report := &Report{
		Timestamp:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		RepositoryPath: "/test/repo",
		Status:         Invalid,
		Violations: []ValidationViolation{
			{Type: MissingFile, Severity: SeverityError, File: "calls/calls-2024.xml", Message: "Call file missing"},
			{Type: InvalidFormat, Severity: SeverityWarning, File: "sms/sms-2024.xml", Message: "SMS format issue"},
			{Type: ChecksumMismatch, Severity: SeverityError, File: "attachments/ab/abc123", Message: "Attachment checksum"},
			{Type: OrphanedAttachment, Severity: SeverityWarning, File: "contacts.yaml", Message: "Contact issue"},
		},
	}

	options := &ReportFilterOptions{
		ExcludeFiles: []string{"calls/", "attachments/"},
	}

	filtered := generator.applyFilters(report, options)

	if len(filtered.Violations) != 2 {
		t.Errorf("Expected 2 violations after file filter, got %d", len(filtered.Violations))
	}

	// Verify excluded files are not present
	for _, violation := range filtered.Violations {
		if strings.Contains(violation.File, "calls/") || strings.Contains(violation.File, "attachments/") {
			t.Errorf("Violation with excluded file pattern found: %s", violation.File)
		}
	}
}

func TestReportGeneratorImpl_ApplyFilters_MaxViolationsLimit(t *testing.T) {
	generator := &ReportGeneratorImpl{}

	report := &Report{
		Timestamp:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		RepositoryPath: "/test/repo",
		Status:         Invalid,
		Violations: []ValidationViolation{
			{Type: MissingFile, Severity: SeverityError, Message: "Missing 1"},
			{Type: MissingFile, Severity: SeverityWarning, Message: "Missing 2"},
			{Type: MissingFile, Severity: SeverityError, Message: "Missing 3"},
			{Type: InvalidFormat, Severity: SeverityWarning, Message: "Format 1"},
			{Type: InvalidFormat, Severity: SeverityError, Message: "Format 2"},
		},
	}

	options := &ReportFilterOptions{
		MaxViolations: 2,
	}

	filtered := generator.applyFilters(report, options)

	// Should have max 2 per type: 2 MissingFile + 2 InvalidFormat = 4 total
	if len(filtered.Violations) != 4 {
		t.Errorf("Expected 4 violations after max limit, got %d", len(filtered.Violations))
	}

	// Count violations by type to verify limit
	typeCount := make(map[ViolationType]int)
	for _, violation := range filtered.Violations {
		typeCount[violation.Type]++
	}

	for violationType, count := range typeCount {
		if count > 2 {
			t.Errorf("Type %s has %d violations, exceeding limit of 2", violationType, count)
		}
	}
}

func TestReportGeneratorImpl_ApplyFilters_CombinedFilters(t *testing.T) {
	generator := &ReportGeneratorImpl{}

	report := &Report{
		Timestamp:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		RepositoryPath: "/test/repo",
		Status:         Invalid,
		Violations: []ValidationViolation{
			{Type: MissingFile, Severity: SeverityError, File: "calls/calls-2024.xml", Message: "Missing call file"},
			{Type: MissingFile, Severity: SeverityWarning, File: "sms/sms-2024.xml", Message: "Missing SMS file"},
			{Type: InvalidFormat, Severity: SeverityError, File: "contacts.yaml", Message: "Invalid contact format"},
			{Type: ChecksumMismatch, Severity: SeverityWarning, File: "calls/calls-2023.xml", Message: "Call checksum mismatch"},
		},
	}

	options := &ReportFilterOptions{
		IncludeSeverities: []Severity{SeverityError},
		IncludeTypes:      []ViolationType{MissingFile, InvalidFormat},
		ExcludeFiles:      []string{"calls/"},
	}

	filtered := generator.applyFilters(report, options)

	// Should only have: InvalidFormat + SeverityError + not in calls/ = 1 violation
	if len(filtered.Violations) != 1 {
		t.Errorf("Expected 1 violation after combined filters, got %d", len(filtered.Violations))
	}

	if len(filtered.Violations) > 0 {
		violation := filtered.Violations[0]
		if violation.Type != InvalidFormat || violation.Severity != SeverityError || strings.Contains(violation.File, "calls/") {
			t.Errorf("Filtered violation doesn't match criteria: %+v", violation)
		}
	}
}

func TestReportGeneratorImpl_ApplyFilters_NoOptions(t *testing.T) {
	generator := &ReportGeneratorImpl{}

	report := &Report{
		Timestamp:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		RepositoryPath: "/test/repo",
		Status:         Invalid,
		Violations: []ValidationViolation{
			{Type: MissingFile, Severity: SeverityError, Message: "Error 1"},
			{Type: InvalidFormat, Severity: SeverityWarning, Message: "Warning 1"},
		},
	}

	// No filters should return original report
	filtered := generator.applyFilters(report, nil)

	if len(filtered.Violations) != len(report.Violations) {
		t.Errorf("Expected %d violations with no filters, got %d", len(report.Violations), len(filtered.Violations))
	}
}

func TestNewReportGenerator(t *testing.T) {
	generator := NewReportGenerator()
	if generator == nil {
		t.Fatal("Expected non-nil report generator")
	}

	// Verify it implements the interface
	// Note: generator is already of type ReportGenerator
	// This test is redundant but kept for clarity
	if generator == nil {
		t.Error("Expected non-nil ReportGenerator")
	}
}

func TestReportGeneratorImpl_GenerateReport_WithFilters(t *testing.T) {
	generator := NewReportGenerator()

	report := &Report{
		Timestamp:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		RepositoryPath: "/test/repo",
		Status:         Invalid,
		Violations: []ValidationViolation{
			{
				Type:     MissingFile,
				Severity: SeverityError,
				File:     "calls/calls-2024.xml",
				Message:  "Required file not found",
			},
			{
				Type:     InvalidFormat,
				Severity: SeverityWarning,
				File:     "contacts.yaml",
				Message:  "Phone number format issue",
			},
		},
	}

	options := &ReportFilterOptions{
		IncludeSeverities: []Severity{SeverityError},
	}

	result, err := generator.GenerateReport(report, FormatText, options)
	if err != nil {
		t.Fatalf("Failed to generate filtered report: %v", err)
	}

	// Should only show error violations
	if !strings.Contains(result, "Required file not found") {
		t.Error("Filtered report should contain error violation")
	}
	if strings.Contains(result, "Phone number format issue") {
		t.Error("Filtered report should not contain warning violation")
	}
	if !strings.Contains(result, "Total Violations: 1") {
		t.Error("Filtered report should show 1 violation")
	}
}
