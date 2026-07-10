package display

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/repository/stats"
)

func TestFormatNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int
		expected string
	}{
		{"zero", 0, "0"},
		{"small", 123, "123"},
		{"under thousand", 999, "999"},
		{"thousand", 1000, "1,000"},
		{"thousands", 1234, "1,234"},
		{"ten thousands", 12345, "12,345"},
		{"hundred thousands", 123456, "123,456"},
		{"millions", 1234567, "1,234,567"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatNumber(tt.input)
			if result != tt.expected {
				t.Errorf("FormatNumber(%d) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{"zero", 0, "0 B"},
		{"bytes", 512, "512 B"},
		{"under KB", 1023, "1023 B"},
		{"1 KB", 1024, "1.0 KB"},
		{"KB", 5120, "5.0 KB"},
		{"MB", 5242880, "5.0 MB"},
		{"GB", 5368709120, "5.0 GB"},
		{"large", 245300000, "233.9 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.input)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAddCommas(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"single digit", "5", "5"},
		{"two digits", "12", "12"},
		{"three digits", "123", "123"},
		{"four digits", "1234", "1,234"},
		{"five digits", "12345", "12,345"},
		{"six digits", "123456", "123,456"},
		{"seven digits", "1234567", "1,234,567"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addCommas(tt.input)
			if result != tt.expected {
				t.Errorf("addCommas(%s) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatter_FormatRepositoryInfo(t *testing.T) {
	t.Parallel()

	info := &RepositoryInfo{
		Version:   "1",
		CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Calls: map[string]stats.YearInfo{
			"2023": {
				Count:    1234,
				Earliest: time.Date(2023, 1, 5, 10, 0, 0, 0, time.UTC),
				Latest:   time.Date(2023, 12, 28, 15, 30, 0, 0, time.UTC),
			},
		},
		SMS: map[string]stats.MessageInfo{
			"2023": {
				TotalCount: 5432,
				SMSCount:   4321,
				MMSCount:   1111,
				Earliest:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				Latest:     time.Date(2023, 12, 31, 23, 59, 0, 0, time.UTC),
			},
		},
		Attachments: AttachmentInfo{
			Count:         1456,
			TotalSize:     245300000,
			OrphanedCount: 12,
			ByType: map[string]int{
				"image/jpeg": 1200,
				"image/png":  200,
				"video/mp4":  56,
			},
		},
		Contacts:     ContactInfo{Count: 123},
		ValidationOK: true,
	}

	var buf bytes.Buffer
	formatter := NewFormatter(&buf, false)
	formatter.FormatRepositoryInfo(info, "/test/repo")

	output := buf.String()

	// Verify key elements are present
	expectedStrings := []string{
		"Repository: /test/repo",
		"Version: 1",
		"Created: 2024-01-15T10:30:00Z",
		"Calls:",
		"2023: 1,234 calls",
		"Total: 1,234 calls",
		"Messages:",
		"2023: 5,432 messages (4,321 SMS, 1,111 MMS)",
		"Total: 5,432 messages (4,321 SMS, 1,111 MMS)",
		"Attachments:",
		"Count: 1,456",
		"Total Size: 233.9 MB",
		"Types:",
		"image/jpeg: 1,200",
		"image/png: 200",
		"video/mp4: 56",
		"Orphaned: 12",
		"Contacts: 123",
		"Validation: OK",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, but it didn't.\nFull output:\n%s", expected, output)
		}
	}
}

func TestFormatter_QuietMode(t *testing.T) {
	t.Parallel()

	info := &RepositoryInfo{
		Version:      "1",
		Calls:        map[string]stats.YearInfo{},
		SMS:          map[string]stats.MessageInfo{},
		ValidationOK: true,
	}

	var buf bytes.Buffer
	formatter := NewFormatter(&buf, true) // quiet mode
	formatter.FormatRepositoryInfo(info, "/test/repo")

	output := buf.String()
	if output != "" {
		t.Errorf("Expected no output in quiet mode, got: %s", output)
	}
}

func TestFormatter_EmptyRepository(t *testing.T) {
	t.Parallel()

	info := &RepositoryInfo{
		Calls:        map[string]stats.YearInfo{},
		SMS:          map[string]stats.MessageInfo{},
		Attachments:  AttachmentInfo{},
		Contacts:     ContactInfo{},
		ValidationOK: true,
	}

	var buf bytes.Buffer
	formatter := NewFormatter(&buf, false)
	formatter.FormatRepositoryInfo(info, "/empty/repo")

	output := buf.String()

	expectedStrings := []string{
		"Repository: /empty/repo",
		"Total: 0 calls",
		"Total: 0 messages (0 SMS, 0 MMS)",
		"Count: 0",
		"Total Size: 0 B",
		"Contacts: 0",
		"Validation: OK",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, but it didn't.\nFull output:\n%s", expected, output)
		}
	}
}

func TestFormatter_WithIssues(t *testing.T) {
	t.Parallel()

	info := &RepositoryInfo{
		Calls:        map[string]stats.YearInfo{},
		SMS:          map[string]stats.MessageInfo{},
		Rejections:   map[string]int{"calls": 5, "sms": 3},
		Errors:       map[string]int{"attachments": 2},
		ValidationOK: false,
	}

	var buf bytes.Buffer
	formatter := NewFormatter(&buf, false)
	formatter.FormatRepositoryInfo(info, "/test/repo")

	output := buf.String()

	expectedStrings := []string{
		"Issues:",
		"Rejections (calls): 5",
		"Rejections (sms): 3",
		"Errors (attachments): 2",
		"Validation: Issues detected",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, but it didn't.\nFull output:\n%s", expected, output)
		}
	}
}

func TestGetSortedCallsYears(t *testing.T) {
	t.Parallel()

	calls := map[string]stats.YearInfo{
		"2023": {Count: 100},
		"2021": {Count: 50},
		"2022": {Count: 75},
	}

	years := getSortedCallsYears(calls)

	expected := []string{"2021", "2022", "2023"}
	if len(years) != len(expected) {
		t.Fatalf("Expected %d years, got %d", len(expected), len(years))
	}

	for i, year := range years {
		if year != expected[i] {
			t.Errorf("Year[%d] = %s; want %s", i, year, expected[i])
		}
	}
}

func TestGetSortedMessageYears(t *testing.T) {
	t.Parallel()

	sms := map[string]stats.MessageInfo{
		"2024": {TotalCount: 100},
		"2022": {TotalCount: 50},
		"2023": {TotalCount: 75},
	}

	years := getSortedMessageYears(sms)

	expected := []string{"2022", "2023", "2024"}
	if len(years) != len(expected) {
		t.Fatalf("Expected %d years, got %d", len(expected), len(years))
	}

	for i, year := range years {
		if year != expected[i] {
			t.Errorf("Year[%d] = %s; want %s", i, year, expected[i])
		}
	}
}

func TestCalculateMessageTotals(t *testing.T) {
	t.Parallel()

	info := &RepositoryInfo{
		SMS: map[string]stats.MessageInfo{
			"2022": {TotalCount: 100, SMSCount: 80, MMSCount: 20},
			"2023": {TotalCount: 200, SMSCount: 150, MMSCount: 50},
		},
	}

	totalMessages, totalSMS, totalMMS := calculateMessageTotals(info)

	if totalMessages != 300 {
		t.Errorf("totalMessages = %d; want 300", totalMessages)
	}
	if totalSMS != 230 {
		t.Errorf("totalSMS = %d; want 230", totalSMS)
	}
	if totalMMS != 70 {
		t.Errorf("totalMMS = %d; want 70", totalMMS)
	}
}

func TestGetSortedAttachmentTypes(t *testing.T) {
	t.Parallel()

	byType := map[string]int{
		"image/jpeg": 1200,
		"image/png":  200,
		"video/mp4":  56,
		"audio/amr":  300,
	}

	types := getSortedAttachmentTypes(byType)

	// Should be sorted by count descending
	expected := []string{"image/jpeg", "audio/amr", "image/png", "video/mp4"}
	if len(types) != len(expected) {
		t.Fatalf("Expected %d types, got %d", len(expected), len(types))
	}

	for i, typ := range types {
		if typ != expected[i] {
			t.Errorf("Type[%d] = %s; want %s", i, typ, expected[i])
		}
	}
}

func TestFormatter_MultipleYears(t *testing.T) {
	t.Parallel()

	info := &RepositoryInfo{
		Calls: map[string]stats.YearInfo{
			"2021": {Count: 500},
			"2022": {Count: 600},
			"2023": {Count: 700},
		},
		SMS: map[string]stats.MessageInfo{
			"2021": {TotalCount: 1000, SMSCount: 800, MMSCount: 200},
			"2022": {TotalCount: 1100, SMSCount: 900, MMSCount: 200},
		},
		ValidationOK: true,
	}

	var buf bytes.Buffer
	formatter := NewFormatter(&buf, false)
	formatter.FormatRepositoryInfo(info, "/test/repo")

	output := buf.String()

	// Verify totals are calculated correctly
	if !strings.Contains(output, "Total: 1,800 calls") {
		t.Error("Expected total calls to be 1,800")
	}
	if !strings.Contains(output, "Total: 2,100 messages (1,700 SMS, 400 MMS)") {
		t.Error("Expected total messages to be 2,100")
	}

	// Verify years appear in sorted order
	callsIndex := strings.Index(output, "Calls:")
	year2021 := strings.Index(output[callsIndex:], "2021:")
	year2022 := strings.Index(output[callsIndex:], "2022:")
	year2023 := strings.Index(output[callsIndex:], "2023:")

	if year2021 == -1 || year2022 == -1 || year2023 == -1 {
		t.Error("Expected all years to be present")
	}
	if year2021 >= year2022 || year2022 >= year2023 {
		t.Error("Expected years to be in ascending order")
	}
}
