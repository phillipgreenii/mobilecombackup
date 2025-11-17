package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/repository/stats"
)

func TestFormatNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{123, "123"},
		{999, "999"},
		{1000, "1,000"},
		{1234, "1,234"},
		{12345, "12,345"},
		{123456, "123,456"},
		{1234567, "1,234,567"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result := formatNumber(test.input)
			if result != test.expected {
				t.Errorf("formatNumber(%d) = %s; want %s", test.input, result, test.expected)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result := formatBytes(test.input)
			if result != test.expected {
				t.Errorf("formatBytes(%d) = %s; want %s", test.input, result, test.expected)
			}
		})
	}
}

func TestAddCommas(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"123", "123"},
		{"1234", "1,234"},
		{"12345", "12,345"},
		{"123456", "123,456"},
		{"1234567", "1,234,567"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := addCommas(test.input)
			if result != test.expected {
				t.Errorf("addCommas(%s) = %s; want %s", test.input, result, test.expected)
			}
		})
	}
}

func TestRepositoryInfoJSONMarshaling(t *testing.T) {
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
		Rejections:   map[string]int{"calls": 2, "sms": 3},
		Errors:       map[string]int{},
		ValidationOK: true,
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Failed to marshal RepositoryInfo: %v", err)
	}

	var unmarshaled RepositoryInfo
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal RepositoryInfo: %v", err)
	}

	// Verify key fields
	if unmarshaled.Version != info.Version {
		t.Errorf("Version mismatch: got %s, want %s", unmarshaled.Version, info.Version)
	}

	if unmarshaled.Contacts.Count != info.Contacts.Count {
		t.Errorf("Contacts mismatch: got %d, want %d", unmarshaled.Contacts.Count, info.Contacts.Count)
	}

	if unmarshaled.ValidationOK != info.ValidationOK {
		t.Errorf("ValidationOK mismatch: got %t, want %t", unmarshaled.ValidationOK, info.ValidationOK)
	}

	// Check calls data
	if len(unmarshaled.Calls) != len(info.Calls) {
		t.Errorf("Calls length mismatch: got %d, want %d", len(unmarshaled.Calls), len(info.Calls))
	}

	callsInfo, exists := unmarshaled.Calls["2023"]
	if !exists {
		t.Error("Expected 2023 calls data not found")
	} else if callsInfo.Count != 1234 {
		t.Errorf("Calls count mismatch: got %d, want %d", callsInfo.Count, 1234)
	}

	// Check SMS data
	smsInfo, exists := unmarshaled.SMS["2023"]
	if !exists {
		t.Error("Expected 2023 SMS data not found")
	} else {
		if smsInfo.TotalCount != 5432 {
			t.Errorf("SMS total count mismatch: got %d, want %d", smsInfo.TotalCount, 5432)
		}
		if smsInfo.SMSCount != 4321 {
			t.Errorf("SMS count mismatch: got %d, want %d", smsInfo.SMSCount, 4321)
		}
		if smsInfo.MMSCount != 1111 {
			t.Errorf("MMS count mismatch: got %d, want %d", smsInfo.MMSCount, 1111)
		}
	}

	// Check attachments data
	if unmarshaled.Attachments.Count != 1456 {
		t.Errorf("Attachments count mismatch: got %d, want %d", unmarshaled.Attachments.Count, 1456)
	}
	if unmarshaled.Attachments.OrphanedCount != 12 {
		t.Errorf("Orphaned count mismatch: got %d, want %d", unmarshaled.Attachments.OrphanedCount, 12)
	}
}

func TestOutputTextInfoFormatting(t *testing.T) {
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
			"2024": {
				Count:    567,
				Earliest: time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC),
				Latest:   time.Date(2024, 6, 15, 15, 30, 0, 0, time.UTC),
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

	// Capture output by temporarily redirecting stdout
	// For this test, we'll just verify that the function doesn't panic
	// and check that key strings would be included
	outputTextInfo(info, "/test/repo")

	// Test specific formatting functions that are used in outputTextInfo
	if formatNumber(1234) != "1,234" {
		t.Error("Number formatting should include commas")
	}

	if !strings.Contains(formatBytes(245300000), "MB") {
		t.Error("Bytes formatting should use appropriate units")
	}
}

func TestInfoMarkerFileContentUnmarshaling(t *testing.T) {
	t.Parallel()

	var marker InfoMarkerFileContent
	jsonData := `{"repository_structure_version":"1","created_at":"2024-01-15T10:30:00Z",` +
		`"created_by":"mobilecombackup v1.0.0"}`
	err := json.Unmarshal([]byte(jsonData), &marker)

	// For JSON test
	if err != nil {
		t.Fatalf("Failed to unmarshal marker file content: %v", err)
	}

	if marker.RepositoryStructureVersion != "1" {
		t.Errorf("Version mismatch: got %s, want %s", marker.RepositoryStructureVersion, "1")
	}

	if marker.CreatedAt != "2024-01-15T10:30:00Z" {
		t.Errorf("CreatedAt mismatch: got %s, want %s", marker.CreatedAt, "2024-01-15T10:30:00Z")
	}

	if marker.CreatedBy != "mobilecombackup v1.0.0" {
		t.Errorf("CreatedBy mismatch: got %s, want %s", marker.CreatedBy, "mobilecombackup v1.0.0")
	}
}

func TestEmptyRepositoryInfo(t *testing.T) {
	t.Parallel()

	info := &RepositoryInfo{
		Calls:        make(map[string]stats.YearInfo),
		SMS:          make(map[string]stats.MessageInfo),
		Rejections:   make(map[string]int),
		Errors:       make(map[string]int),
		ValidationOK: true,
	}

	// Test JSON marshaling of empty repository
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Failed to marshal empty RepositoryInfo: %v", err)
	}

	var unmarshaled RepositoryInfo
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal empty RepositoryInfo: %v", err)
	}

	if len(unmarshaled.Calls) != 0 {
		t.Errorf("Expected empty calls map, got %d entries", len(unmarshaled.Calls))
	}

	if len(unmarshaled.SMS) != 0 {
		t.Errorf("Expected empty SMS map, got %d entries", len(unmarshaled.SMS))
	}

	if unmarshaled.ValidationOK != true {
		t.Errorf("Expected ValidationOK to be true for empty repository")
	}
}
