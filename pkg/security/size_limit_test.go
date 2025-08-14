package security

import (
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func TestIOLimitedReader_PreventDoS(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		limit     int64
		wantError bool
		wantEOF   bool
	}{
		{
			name:      "small content within limit",
			content:   "<calls><call number=\"123\" date=\"1234567890000\" type=\"1\" /></calls>",
			limit:     1000,
			wantError: false,
			wantEOF:   false,
		},
		{
			name:      "content exactly at limit",
			content:   strings.Repeat("a", 100),
			limit:     100,
			wantError: false,
			wantEOF:   false,
		},
		{
			name:      "content exceeds limit",
			content:   strings.Repeat("a", 150),
			limit:     100,
			wantError: false, // LimitedReader doesn't error, it just stops reading
			wantEOF:   true,  // Should reach EOF when limit is hit
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.content)
			limitedReader := &io.LimitedReader{R: reader, N: tt.limit}

			// Try to read all content
			content, err := io.ReadAll(limitedReader)

			if tt.wantError && err == nil {
				t.Errorf("Expected error, got none")
			}
			if !tt.wantError && err != nil && err != io.EOF {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.wantEOF && limitedReader.N != 0 {
				t.Errorf("Expected to hit limit (N should be 0), but N = %d", limitedReader.N)
			}

			if !tt.wantEOF && len(content) != len(tt.content) {
				t.Errorf("Content was truncated unexpectedly: got %d bytes, want %d", len(content), len(tt.content))
			}
		})
	}
}

func TestXMLParsingWithSizeLimit(t *testing.T) {
	// Test XML parsing with size limits similar to importer usage
	tests := []struct {
		name               string
		xmlContent         string
		limit              int64
		expectError        bool
		expectLimitReached bool
	}{
		{
			name: "small valid XML",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<calls count="1">
  <call number="123456789" date="1234567890000" type="1" duration="60" />
</calls>`,
			limit:              1000,
			expectError:        false,
			expectLimitReached: false,
		},
		{
			name: "large XML exceeding limit",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<calls count="1000">` +
				strings.Repeat(`<call number="123456789012345678901234567890" date="1234567890000" type="1" duration="60" />`, 1000) +
				`</calls>`,
			limit:              500,  // Very small limit to force truncation
			expectError:        true, // XML parsing should fail on truncated content
			expectLimitReached: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.xmlContent)
			limitedReader := &io.LimitedReader{R: reader, N: tt.limit}

			// Try to parse XML (similar to what importer does)
			decoder := xml.NewDecoder(limitedReader)

			var root struct {
				XMLName xml.Name `xml:"calls"`
				Count   string   `xml:"count,attr"`
				Calls   []struct {
					Number   string `xml:"number,attr"`
					Date     string `xml:"date,attr"`
					Type     string `xml:"type,attr"`
					Duration string `xml:"duration,attr"`
				} `xml:"call"`
			}

			err := decoder.Decode(&root)

			if tt.expectError && err == nil {
				t.Errorf("Expected parsing error due to size limit, got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected parsing error: %v", err)
			}

			if tt.expectLimitReached && limitedReader.N != 0 {
				t.Errorf("Expected to hit size limit, but still have %d bytes remaining", limitedReader.N)
			}
		})
	}
}

func TestFileSizeLimitExceededError_Integration(t *testing.T) {
	// Test the error type we use in the importer
	limit := int64(100)
	attempted := int64(200)

	err := NewFileSizeLimitExceededError("test.xml", limit, attempted, "XML parsing")
	expectedMsg := "XML parsing: file size limit exceeded for test.xml: limit 100B, attempted 200B"

	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}

	// Test error detection pattern used in importer
	reader := strings.NewReader(strings.Repeat("a", 200))
	limitedReader := &io.LimitedReader{R: reader, N: 100}

	// Read all content to simulate what XML decoder would do
	_, readErr := io.ReadAll(limitedReader)

	// Check if we can detect the limit was reached (like in importer)
	// When reading more data than the limit, ReadAll returns nil error but N becomes 0
	if limitedReader.N == 0 {
		// This is the condition we check in importer to detect size limit exceeded
		t.Logf("Successfully detected size limit exceeded condition")
	} else {
		t.Errorf("Failed to detect size limit condition: err=%v, N=%d", readErr, limitedReader.N)
	}
}

// Benchmark the overhead of LimitedReader
func BenchmarkLimitedReaderOverhead(b *testing.B) {
	content := strings.Repeat("benchmark data ", 1000)

	b.Run("without_limit", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			reader := strings.NewReader(content)
			buf := make([]byte, 4096)
			for {
				_, err := reader.Read(buf)
				if err == io.EOF {
					break
				}
			}
		}
	})

	b.Run("with_limit", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			reader := strings.NewReader(content)
			limitedReader := &io.LimitedReader{R: reader, N: int64(len(content) + 1000)}
			buf := make([]byte, 4096)
			for {
				_, err := limitedReader.Read(buf)
				if err == io.EOF {
					break
				}
			}
		}
	})
}
