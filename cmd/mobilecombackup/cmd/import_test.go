package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/importer"
	"github.com/spf13/cobra"
)

func TestImportCommand(t *testing.T) {
	// Save original values
	origArgs := os.Args
	origRepoRoot := repoRoot
	defer func() {
		os.Args = origArgs
		repoRoot = origRepoRoot
	}()

	tests := []struct {
		name        string
		args        []string
		env         map[string]string
		wantErr     bool
		errContains string
	}{
		{
			name:        "no repository and no args",
			args:        []string{"mobilecombackup", "import"},
			wantErr:     true,
			errContains: "no repository specified",
		},
		// Note: Tests for invalid repository paths are covered in integration tests
		// since they call os.Exit() which terminates the test process
		{
			name:        "invalid filter value",
			args:        []string{"mobilecombackup", "import", "--repo-root", ".", "--filter", "invalid", "."},
			wantErr:     true,
			errContains: "invalid filter value",
		},
		{
			name:    "help flag",
			args:    []string{"mobilecombackup", "import", "--help"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset cobra command
			rootCmd = &cobra.Command{
				Use:   "mobilecombackup",
				Short: "mobilecombackup processes call logs and SMS/MMS messages",
			}
			// Re-register commands manually
			rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Suppress non-error output")
			rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose logging")
			rootCmd.PersistentFlags().StringVar(&repoRoot, "repo-root", ".", "Path to repository root")
			rootCmd.AddCommand(importCmd)

			// Set environment variables
			for k, v := range tt.env {
				_ = os.Setenv(k, v)
				defer func(key string) { _ = os.Unsetenv(key) }(k)
			}

			// Set command line arguments
			os.Args = tt.args

			// Capture output
			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)

			// Execute command
			err := rootCmd.Execute()

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error = %v, want error containing %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestResolveImportRepoRoot(t *testing.T) {
	// Save original values
	origRepoRoot := repoRoot
	defer func() {
		repoRoot = origRepoRoot
	}()

	tests := []struct {
		name     string
		flagVal  string
		envVal   string
		expected string
	}{
		{
			name:     "flag takes precedence",
			flagVal:  "/flag/path",
			envVal:   "/env/path",
			expected: "/flag/path",
		},
		{
			name:     "env when no flag",
			flagVal:  ".",
			envVal:   "/env/path",
			expected: "/env/path",
		},
		{
			name:     "default when neither",
			flagVal:  ".",
			envVal:   "",
			expected: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoRoot = tt.flagVal
			if tt.envVal != "" {
				_ = os.Setenv("MB_REPO_ROOT", tt.envVal)
				defer func() { _ = os.Unsetenv("MB_REPO_ROOT") }()
			}

			result := resolveImportRepoRoot()
			if result != tt.expected {
				t.Errorf("resolveImportRepoRoot() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConsoleProgressReporter(t *testing.T) {
	reporter := &consoleProgressReporter{}

	// Test StartFile
	reporter.StartFile("/path/to/backup.xml", 5, 1)
	if reporter.currentFile != "backup.xml" {
		t.Errorf("Expected currentFile to be 'backup.xml', got %s", reporter.currentFile)
	}
	if reporter.fileCount != 5 {
		t.Errorf("Expected fileCount to be 5, got %d", reporter.fileCount)
	}
	if reporter.fileIndex != 1 {
		t.Errorf("Expected fileIndex to be 1, got %d", reporter.fileIndex)
	}

	// Test UpdateProgress - should output on multiples of 100
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	reporter.UpdateProgress(100, 0)
	reporter.UpdateProgress(50, 0) // Should not output
	reporter.UpdateProgress(200, 0)

	_ = w.Close()
	os.Stdout = oldStdout

	// Read captured output
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "(100 records)") {
		t.Error("Expected output to contain '(100 records)'")
	}
	if !strings.Contains(output, "(200 records)") {
		t.Error("Expected output to contain '(200 records)'")
	}
	if strings.Contains(output, "(50 records)") {
		t.Error("Should not output for non-100 multiples")
	}
}

func TestImportCommandFlags(t *testing.T) {
	// Test that all expected flags are registered

	// Find the import command
	var importSubCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "import" {
			importSubCmd = cmd
			break
		}
	}

	if importSubCmd == nil {
		t.Fatal("Import command not found")
	}

	// Check flags exist
	expectedFlags := []string{
		"dry-run",
		"json",
		"filter",
		"no-error-on-rejects",
	}

	for _, flagName := range expectedFlags {
		flag := importSubCmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag %q not found", flagName)
		}
	}

	// Check global flags are available
	globalFlags := []string{
		"repo-root",
		"quiet",
		"verbose",
	}

	for _, flagName := range globalFlags {
		// Check in persistent flags (inherited from root)
		flag := rootCmd.PersistentFlags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected global flag %q not found", flagName)
		}
	}
}

func TestDisplaySummary(t *testing.T) {
	// Create a test summary with all entity types
	summary := &importer.ImportSummary{
		Calls: &importer.EntityStats{
			YearStats: map[int]*importer.YearStat{
				2014: {Final: 10, Added: 8, Duplicates: 2},
				2015: {Final: 15, Added: 15, Duplicates: 0},
			},
			Total: &importer.YearStat{
				Initial:    0,
				Final:      25,
				Added:      23,
				Duplicates: 2,
			},
		},
		SMS: &importer.EntityStats{
			YearStats: map[int]*importer.YearStat{
				2014: {Final: 25, Added: 20, Duplicates: 5},
				2015: {Final: 30, Added: 25, Duplicates: 5},
			},
			Total: &importer.YearStat{
				Initial:    0,
				Final:      55,
				Added:      45,
				Duplicates: 10,
			},
		},
		Attachments: &importer.AttachmentStats{
			Total: &importer.AttachmentStat{
				Total:      12,
				New:        10,
				Duplicates: 2,
			},
		},
		Rejections: map[string]*importer.RejectionStats{
			"missing-timestamp":    {Count: 1, Reason: "missing-timestamp"},
			"malformed-attachment": {Count: 2, Reason: "malformed-attachment"},
		},
		FilesProcessed: 5,
		Duration:       2 * time.Second,
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Save importFilter
	oldFilter := importFilter
	importFilter = ""
	defer func() { importFilter = oldFilter }()

	// Call displaySummary
	displaySummary(summary, false)

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Verify all entity types are displayed
	expectedStrings := []string{
		"Import Summary:",
		"Calls:",
		"2014: 10 entries (8 new, 2 duplicates)",
		"2015: 15 entries (15 new, 0 duplicates)",
		"Total: 25 entries (23 new, 2 duplicates)",
		"SMS:",
		"2014: 25 entries (20 new, 5 duplicates)",
		"2015: 30 entries (25 new, 5 duplicates)",
		"Total: 55 entries (45 new, 10 duplicates)",
		"Attachments:",
		"Total: 12 files (10 new, 2 duplicates)",
		"Rejections:",
		"Files processed: 5",
		"Time taken: 2.0s",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, but it didn't.\nOutput:\n%s", expected, output)
		}
	}
}

func TestDisplayJSONSummary(t *testing.T) {
	// Create a test summary
	summary := &importer.ImportSummary{
		Calls: &importer.EntityStats{
			YearStats: map[int]*importer.YearStat{
				2014: {Final: 10, Added: 8, Duplicates: 2},
			},
			Total: &importer.YearStat{
				Initial:    0,
				Final:      10,
				Added:      8,
				Duplicates: 2,
			},
		},
		SMS: &importer.EntityStats{
			YearStats: map[int]*importer.YearStat{
				2014: {Final: 25, Added: 20, Duplicates: 5},
			},
			Total: &importer.YearStat{
				Initial:    0,
				Final:      25,
				Added:      20,
				Duplicates: 5,
			},
		},
		Attachments: &importer.AttachmentStats{
			Total: &importer.AttachmentStat{
				Total:      5,
				New:        4,
				Duplicates: 1,
			},
		},
		Rejections: map[string]*importer.RejectionStats{
			"missing-timestamp": {Count: 1, Reason: "missing-timestamp"},
		},
		FilesProcessed: 2,
		Duration:       1 * time.Second,
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call displayJSONSummary
	displayJSONSummary(summary)

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Verify JSON structure contains expected fields
	expectedFields := []string{
		`"files_processed": 2`,
		`"calls"`,
		`"sms"`,
		`"attachments"`,
		`"rejections"`,
		`"total": 5`,
		`"new": 4`,
		`"duplicates": 1`,
	}

	for _, expected := range expectedFields {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected JSON output to contain %q, but it didn't.\nOutput:\n%s", expected, output)
		}
	}
}

// TestDisplaySummaryYearOrdering tests that years are displayed in ascending chronological order
func TestDisplaySummaryYearOrdering(t *testing.T) {
	// Create a test summary with years in mixed order
	summary := &importer.ImportSummary{
		Calls: &importer.EntityStats{
			YearStats: map[int]*importer.YearStat{
				2024: {Final: 100, Added: 80, Duplicates: 20},
				2020: {Final: 200, Added: 150, Duplicates: 50},
				2022: {Final: 150, Added: 120, Duplicates: 30},
			},
			Total: &importer.YearStat{
				Initial:    0,
				Final:      450,
				Added:      350,
				Duplicates: 100,
			},
		},
		SMS: &importer.EntityStats{
			YearStats: map[int]*importer.YearStat{
				2023: {Final: 300, Added: 250, Duplicates: 50},
				2021: {Final: 250, Added: 200, Duplicates: 50},
				2019: {Final: 180, Added: 150, Duplicates: 30},
			},
			Total: &importer.YearStat{
				Initial:    0,
				Final:      730,
				Added:      600,
				Duplicates: 130,
			},
		},
		Attachments: &importer.AttachmentStats{
			Total: &importer.AttachmentStat{
				Total:      10,
				New:        8,
				Duplicates: 2,
			},
		},
		FilesProcessed: 3,
		Duration:       1 * time.Second,
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Save importFilter
	oldFilter := importFilter
	importFilter = ""
	defer func() { importFilter = oldFilter }()

	// Call displaySummary
	displaySummary(summary, false)

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Verify years appear in ascending order for Calls
	callsIndex := strings.Index(output, "Calls:")
	if callsIndex == -1 {
		t.Fatal("Expected 'Calls:' section in output")
	}
	callsSection := output[callsIndex:]

	smsIndex := strings.Index(callsSection, "SMS:")
	if smsIndex == -1 {
		t.Fatal("Expected 'SMS:' section in output")
	}
	smsSection := callsSection[smsIndex:]

	// For calls section, verify 2020 appears before 2022, which appears before 2024
	index2020 := strings.Index(callsSection, "2020: 200 entries")
	index2022 := strings.Index(callsSection, "2022: 150 entries")
	index2024 := strings.Index(callsSection, "2024: 100 entries")

	if index2020 == -1 || index2022 == -1 || index2024 == -1 {
		t.Fatalf("Missing expected year entries in calls section:\nOutput:\n%s", output)
	}

	if index2020 > index2022 {
		t.Errorf("Year 2020 should appear before 2022 in calls section")
	}
	if index2022 > index2024 {
		t.Errorf("Year 2022 should appear before 2024 in calls section")
	}

	// For SMS section, verify 2019 appears before 2021, which appears before 2023
	index2019 := strings.Index(smsSection, "2019: 180 entries")
	index2021 := strings.Index(smsSection, "2021: 250 entries")
	index2023 := strings.Index(smsSection, "2023: 300 entries")

	if index2019 == -1 || index2021 == -1 || index2023 == -1 {
		t.Fatalf("Missing expected year entries in SMS section:\nOutput:\n%s", output)
	}

	if index2019 > index2021 {
		t.Errorf("Year 2019 should appear before 2021 in SMS section")
	}
	if index2021 > index2023 {
		t.Errorf("Year 2021 should appear before 2023 in SMS section")
	}
}

// TestDisplayJSONSummaryYearOrdering tests that years are ordered in JSON output
func TestDisplayJSONSummaryYearOrdering(t *testing.T) {
	// Create a test summary with years in mixed order
	summary := &importer.ImportSummary{
		Calls: &importer.EntityStats{
			YearStats: map[int]*importer.YearStat{
				2024: {Final: 100, Added: 80, Duplicates: 20},
				2020: {Final: 200, Added: 150, Duplicates: 50},
				2022: {Final: 150, Added: 120, Duplicates: 30},
			},
			Total: &importer.YearStat{
				Initial:    0,
				Final:      450,
				Added:      350,
				Duplicates: 100,
			},
		},
		SMS: &importer.EntityStats{
			YearStats: map[int]*importer.YearStat{
				2023: {Final: 300, Added: 250, Duplicates: 50},
				2021: {Final: 250, Added: 200, Duplicates: 50},
			},
			Total: &importer.YearStat{
				Initial:    0,
				Final:      550,
				Added:      450,
				Duplicates: 100,
			},
		},
		Attachments: &importer.AttachmentStats{
			Total: &importer.AttachmentStat{
				Total:      5,
				New:        4,
				Duplicates: 1,
			},
		},
		FilesProcessed: 2,
		Duration:       1 * time.Second,
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call displayJSONSummary
	displayJSONSummary(summary)

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Parse JSON to verify structure
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput:\n%s", err, output)
	}

	// Verify the JSON output has years in ascending order by checking the raw JSON string
	// Since JSON objects maintain insertion order in Go's json package, we can check the string position

	// For calls section, verify 2020 appears before 2022, which appears before 2024
	callsYearsStart := strings.Index(output, `"calls"`)
	if callsYearsStart == -1 {
		t.Fatal("calls section not found in JSON output")
	}
	callsSection := output[callsYearsStart:]

	smsYearsStart := strings.Index(callsSection, `"sms"`)
	if smsYearsStart != -1 {
		callsSection = callsSection[:smsYearsStart] // Limit to just calls section
	}

	index2020 := strings.Index(callsSection, `"2020"`)
	index2022 := strings.Index(callsSection, `"2022"`)
	index2024 := strings.Index(callsSection, `"2024"`)

	if index2020 == -1 || index2022 == -1 || index2024 == -1 {
		t.Fatalf("Missing expected year keys in calls section:\nOutput:\n%s", output)
	}

	if index2020 > index2022 {
		t.Errorf("Year 2020 should appear before 2022 in calls JSON section")
	}
	if index2022 > index2024 {
		t.Errorf("Year 2022 should appear before 2024 in calls JSON section")
	}

	// For SMS section, verify 2021 appears before 2023
	smsStart := strings.Index(output, `"sms"`)
	if smsStart == -1 {
		t.Fatal("sms section not found in JSON output")
	}
	smsSection := output[smsStart:]

	// Limit SMS section to just that section (before attachments)
	attachmentsStart := strings.Index(smsSection, `"attachments"`)
	if attachmentsStart != -1 {
		smsSection = smsSection[:attachmentsStart]
	}

	index2021 := strings.Index(smsSection, `"2021"`)
	index2023 := strings.Index(smsSection, `"2023"`)

	if index2021 == -1 || index2023 == -1 {
		t.Fatalf("Missing expected year keys in SMS section:\nOutput:\n%s", output)
	}

	if index2021 > index2023 {
		t.Errorf("Year 2021 should appear before 2023 in SMS JSON section")
	}
}

// TestSortYearStatsForJSON tests the helper function directly
func TestSortYearStatsForJSON(t *testing.T) {
	// Test with mixed order years
	yearStats := map[int]*importer.YearStat{
		2024: {Final: 100, Added: 80, Duplicates: 20},
		2020: {Final: 200, Added: 150, Duplicates: 50},
		2022: {Final: 150, Added: 120, Duplicates: 30},
		2021: {Final: 175, Added: 140, Duplicates: 35},
	}

	result := sortYearStatsForJSON(yearStats)

	// Since Go maps have non-deterministic iteration order,
	// we need to check ordering differently. The sortYearStatsForJSON function
	// should create an ordered map in the internal representation.
	// We'll verify by marshaling to JSON and checking the order in the JSON string.
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal result to JSON: %v", err)
	}
	jsonStr := string(jsonBytes)

	// Check positions of years in the JSON string
	index2020 := strings.Index(jsonStr, `"2020"`)
	index2021 := strings.Index(jsonStr, `"2021"`)
	index2022 := strings.Index(jsonStr, `"2022"`)
	index2024 := strings.Index(jsonStr, `"2024"`)

	if index2020 == -1 || index2021 == -1 || index2022 == -1 || index2024 == -1 {
		t.Fatalf("Missing expected years in JSON: %s", jsonStr)
	}

	// Verify ascending order
	if index2020 > index2021 || index2021 > index2022 || index2022 > index2024 {
		t.Errorf("Years not in ascending order in JSON. Positions: 2020=%d, 2021=%d, 2022=%d, 2024=%d\nJSON: %s",
			index2020, index2021, index2022, index2024, jsonStr)
	}

	// Verify we have all expected keys
	expectedOrder := []string{"2020", "2021", "2022", "2024"}
	if len(result) != len(expectedOrder) {
		t.Fatalf("Expected %d keys, got %d: %v", len(expectedOrder), len(result), result)
	}

	// Verify values are preserved correctly
	for year, originalStat := range yearStats {
		yearStr := fmt.Sprintf("%d", year)
		if resultStat, exists := result[yearStr]; !exists {
			t.Errorf("Missing year %s in result", yearStr)
		} else if resultStat != originalStat {
			t.Errorf("Year %s stat not preserved correctly", yearStr)
		}
	}
}

// TestSortYearStatsForJSON_EdgeCases tests edge cases for the helper function
func TestSortYearStatsForJSON_EdgeCases(t *testing.T) {
	// Test with empty map
	emptyResult := sortYearStatsForJSON(nil)
	if len(emptyResult) != 0 {
		t.Errorf("Expected empty result for nil input, got %v", emptyResult)
	}

	emptyMap := make(map[int]*importer.YearStat)
	emptyResult = sortYearStatsForJSON(emptyMap)
	if len(emptyResult) != 0 {
		t.Errorf("Expected empty result for empty input, got %v", emptyResult)
	}

	// Test with single year
	singleYear := map[int]*importer.YearStat{
		2023: {Final: 50, Added: 40, Duplicates: 10},
	}

	singleResult := sortYearStatsForJSON(singleYear)
	if len(singleResult) != 1 {
		t.Fatalf("Expected 1 result for single year, got %d", len(singleResult))
	}

	if _, exists := singleResult["2023"]; !exists {
		t.Errorf("Expected year 2023 in result, got keys: %v", singleResult)
	}
}
