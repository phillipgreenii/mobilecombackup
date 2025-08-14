package security

import (
	"strings"
	"testing"
)

func TestParseSize_ValidSizes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"bytes no unit", "1024", 1024},
		{"bytes with B", "1024B", 1024},
		{"kilobytes K", "1K", 1024},
		{"kilobytes KB", "1KB", 1024},
		{"megabytes M", "1M", 1024 * 1024},
		{"megabytes MB", "1MB", 1024 * 1024},
		{"gigabytes G", "1G", 1024 * 1024 * 1024},
		{"gigabytes GB", "1GB", 1024 * 1024 * 1024},
		{"terabytes T", "1T", 1024 * 1024 * 1024 * 1024},
		{"terabytes TB", "1TB", 1024 * 1024 * 1024 * 1024},
		{"decimal megabytes", "1.5MB", int64(1.5 * 1024 * 1024)},
		{"500 megabytes", "500MB", 500 * 1024 * 1024},
		{"10 megabytes", "10MB", 10 * 1024 * 1024},
		{"zero bytes", "0", 0},
		{"zero with unit", "0B", 0},
		{"spaces ignored", "  500  MB  ", 500 * 1024 * 1024},
		{"lowercase input", "500mb", 500 * 1024 * 1024},
		{"mixed case", "500Mb", 500 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSize(tt.input)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestParseSize_InvalidSizes(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedErr error
	}{
		{"empty string", "", ErrInvalidSizeFormat},
		{"invalid number", "abc", ErrInvalidSizeFormat},
		{"invalid unit", "100XB", ErrInvalidSizeFormat},
		{"negative number", "-100MB", ErrNegativeSize},
		{"no number", "MB", ErrInvalidSizeFormat},
		{"space in number", "10 0MB", ErrInvalidSizeFormat},
		{"multiple units", "100MBGB", ErrInvalidSizeFormat},
		{"special chars", "100@MB", ErrInvalidSizeFormat},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSize(tt.input)
			if err == nil {
				t.Fatalf("Expected error for input %q, got none", tt.input)
			}

			if !strings.Contains(err.Error(), tt.expectedErr.Error()) {
				t.Errorf("Expected error containing %q, got %v", tt.expectedErr, err)
			}
		})
	}
}

func TestParseSize_DefaultLimits(t *testing.T) {
	// Test default limits from BUG-051
	defaultXMLLimit := "500MB"
	defaultMessageLimit := "10MB"

	xmlSize, err := ParseSize(defaultXMLLimit)
	if err != nil {
		t.Errorf("Failed to parse default XML limit: %v", err)
	}
	expected := int64(500 * 1024 * 1024)
	if xmlSize != expected {
		t.Errorf("Expected XML limit %d, got %d", expected, xmlSize)
	}

	msgSize, err := ParseSize(defaultMessageLimit)
	if err != nil {
		t.Errorf("Failed to parse default message limit: %v", err)
	}
	expected = int64(10 * 1024 * 1024)
	if msgSize != expected {
		t.Errorf("Expected message limit %d, got %d", expected, msgSize)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{"zero bytes", 0, "0B"},
		{"bytes", 512, "512B"},
		{"kilobytes exact", 1024, "1KB"},
		{"kilobytes decimal", 1536, "1.5KB"},
		{"megabytes exact", 1024 * 1024, "1MB"},
		{"megabytes decimal", int64(1.5 * 1024 * 1024), "1.5MB"},
		{"gigabytes exact", 1024 * 1024 * 1024, "1GB"},
		{"500 megabytes", 500 * 1024 * 1024, "500MB"},
		{"large size", int64(2.5 * 1024 * 1024 * 1024), "2.5GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSize(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFileSizeLimitExceededError(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		limit     int64
		attempted int64
		context   string
		expected  string
	}{
		{
			name:      "basic error",
			filename:  "test.xml",
			limit:     500 * 1024 * 1024,
			attempted: 600 * 1024 * 1024,
			expected:  "file size limit exceeded for test.xml: limit 500MB, attempted 600MB",
		},
		{
			name:      "with context",
			filename:  "test.xml",
			limit:     500 * 1024 * 1024,
			attempted: 0,
			context:   "XML parsing",
			expected:  "XML parsing: file size limit exceeded for test.xml: limit 500MB",
		},
		{
			name:      "with context and attempted",
			filename:  "large.xml",
			limit:     10 * 1024 * 1024,
			attempted: 50 * 1024 * 1024,
			context:   "Message processing",
			expected:  "Message processing: file size limit exceeded for large.xml: limit 10MB, attempted 50MB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewFileSizeLimitExceededError(tt.filename, tt.limit, tt.attempted, tt.context)
			result := err.Error()
			if result != tt.expected {
				t.Errorf("Expected error %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestParseSize_EdgeCases(t *testing.T) {
	// Test boundary conditions
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"very large size", "9TB", false},        // Should work
		{"extremely large size", "1000TB", true}, // Should exceed MaxReasonableSize
		{"decimal precision", "1.999MB", false},
		{"zero decimal", "0.0MB", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSize(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for %q, got none", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for %q, got %v", tt.input, err)
				}
			}
		})
	}
}

// Benchmark size parsing performance
func BenchmarkParseSize(b *testing.B) {
	testSizes := []string{"500MB", "10GB", "1024", "1.5TB"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, size := range testSizes {
			_, _ = ParseSize(size) // Ignore error in benchmark
		}
	}
}

func BenchmarkFormatSize(b *testing.B) {
	testSizes := []int64{
		1024,
		1024 * 1024,
		500 * 1024 * 1024,
		int64(1.5 * 1024 * 1024 * 1024),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, size := range testSizes {
			FormatSize(size)
		}
	}
}
