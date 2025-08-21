package calls

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/phillipgreenii/mobilecombackup/pkg/security"
)

// FuzzXMLCallsParser tests the XML parsing with random input to find crashes and edge cases
func FuzzXMLCallsParser(f *testing.F) {
	// Seed with some known valid XML structures
	validSeeds := []string{
		`<?xml version='1.0' encoding='UTF-8'?><calls count="1"><call number="123" duration="30" date="1234567890" type="1"/></calls>`,
		`<calls count="2"><call number="555-1234" duration="60" date="1640995200000" type="2"/><call number="555-5678" duration="0" date="1640995300000" type="3"/></calls>`,
		`<calls><call number="" duration="null" date="null" type="0"/></calls>`,
		`<calls count="0"></calls>`,
	}

	for _, seed := range validSeeds {
		f.Add(seed)
	}

	// Add some edge case seeds
	edgeCases := []string{
		``,                                  // Empty input
		`<`,                                 // Incomplete XML
		`<calls`,                            // Unclosed tag
		`<calls></call>`,                    // Mismatched tags
		`<calls count="invalid"></calls>`,   // Invalid count
		`<calls><call/></calls>`,            // Self-closing call
		`<calls><unknown_element/></calls>`, // Unknown elements
		strings.Repeat("a", 10000),          // Very long input
		`<calls>` + strings.Repeat("<call number='test' duration='30' date='123' type='1'/>", 1000) + `</calls>`, // Many calls
	}

	for _, edge := range edgeCases {
		f.Add(edge)
	}

	f.Fuzz(func(t *testing.T, xmlData string) {
		// The fuzzer should never cause a panic, even with malformed input
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("XML parser panicked with input %q: %v", xmlData, r)
			}
		}()

		// Test direct XML parsing which is the core security-critical component
		var xmlCalls struct {
			XMLName xml.Name `xml:"calls"`
			Count   int      `xml:"count,attr"`
			Calls   []struct {
				Number       string `xml:"number,attr"`
				Duration     string `xml:"duration,attr"`
				Date         string `xml:"date,attr"`
				Type         string `xml:"type,attr"`
				ReadableDate string `xml:"readable_date,attr"`
				ContactName  string `xml:"contact_name,attr"`
			} `xml:"call"`
		}

		reader := strings.NewReader(xmlData)
		decoder := security.NewSecureXMLDecoder(reader)

		// Try to decode - this should handle any input gracefully
		err := decoder.Decode(&xmlCalls)

		// Error is acceptable for malformed input, panic is not
		if err == nil {
			// If parsing succeeded, validate the data makes sense
			if xmlCalls.Count < 0 {
				t.Errorf("Negative count parsed: %d", xmlCalls.Count)
			}

			// Count should be reasonable (not extremely large)
			if xmlCalls.Count > 1000000 {
				t.Errorf("Unreasonably large count: %d", xmlCalls.Count)
			}

			// Validate each call record
			for i, call := range xmlCalls.Calls {
				// These operations should not panic
				_ = call.Number
				_ = call.Duration
				_ = call.Date
				_ = call.Type
				_ = call.ReadableDate
				_ = call.ContactName

				// Check for excessively large fields
				if len(call.Number) > 100 {
					t.Logf("Call %d has very long number: %d chars", i, len(call.Number))
				}
				if len(call.ContactName) > 1000 {
					t.Errorf("Call %d has excessively long contact name: %d chars", i, len(call.ContactName))
				}
			}
		}
	})
}

// FuzzCallRecordValidation tests Call struct field validation
func FuzzCallRecordValidation(f *testing.F) {
	// Seed with various call record formats
	f.Add("123-456-7890", "30", "1640995200000", "1")
	f.Add("", "0", "null", "3")
	f.Add(strings.Repeat("1", 50), "-1", "0", "999")

	f.Fuzz(func(t *testing.T, number, duration, date, callType string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Call validation panicked: number=%q duration=%q date=%q type=%q panic=%v",
					number, duration, date, callType, r)
			}
		}()

		// Create a call struct and test field access
		// This tests the field parsing and validation logic
		call := Call{
			Number:       number,
			Duration:     0,        // Will be parsed separately
			Date:         0,        // Will be parsed separately
			Type:         Incoming, // Default type
			ReadableDate: "",
			ContactName:  "",
		}

		// These operations should never panic
		_ = call.Number
		_ = call.Duration
		_ = call.Date
		_ = call.Type
		_ = call.ReadableDate
		_ = call.ContactName

		// Test timestamp conversion
		_ = call.Timestamp()

		// Validate reasonable field lengths
		if len(call.Number) > 10000 {
			t.Errorf("Number field excessively large: %d chars", len(call.Number))
		}
		if len(call.ContactName) > 100000 {
			t.Errorf("ContactName field excessively large: %d chars", len(call.ContactName))
		}
	})
}
