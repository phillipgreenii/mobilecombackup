package sms

import (
	"strings"
	"testing"
)

// FuzzXMLSMSReader_StreamMessages tests the XML parser with random input to find crashes and edge cases
func FuzzXMLSMSReader_StreamMessages(f *testing.F) {
	// Seed with some known valid XML structures
	validSeeds := []string{
		`<?xml version='1.0' encoding='UTF-8'?><smses count="1"><sms address="123" date="1234567890" type="1" body="test"/></smses>`,
		`<smses count="2"><sms address="555-1234" date="1640995200000" type="2" body="Hello"/><sms address="555-5678" date="1640995300000" type="1" body="World"/></smses>`,
		`<smses count="1"><mms date="1640995200000" rr="0" sub="Test" ct_t="application/vnd.wap.multipart.related" msg_box="1"><parts><part seq="0" ct="text/plain" text="Hello MMS"/></parts></mms></smses>`,
		`<smses><sms address="" date="null" type="0" body=""/></smses>`,
		`<smses count="0"></smses>`,
	}

	for _, seed := range validSeeds {
		f.Add(seed)
	}

	// Add some edge case seeds
	edgeCases := []string{
		``,                                  // Empty input
		`<`,                                 // Incomplete XML
		`<smses`,                            // Unclosed tag
		`<smses></sms>`,                     // Mismatched tags
		`<smses count="invalid"></smses>`,   // Invalid count
		`<smses><sms/></smses>`,             // Self-closing SMS
		`<smses><unknown_element/></smses>`, // Unknown elements
		strings.Repeat("a", 10000),          // Very long input
		`<smses>` + strings.Repeat("<sms address='test' date='123' type='1' body='msg'/>", 1000) + `</smses>`, // Many messages
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

		reader := strings.NewReader(xmlData)
		xmlReader := NewXMLSMSReader("/tmp") // Use dummy repo root

		// Try to stream messages - this should handle any input gracefully
		var messages []Message

		// Use callback to collect messages and errors
		err := xmlReader.StreamMessagesFromReader(reader, func(msg Message) error {
			messages = append(messages, msg)
			return nil
		})

		// The parser should either succeed or fail gracefully with an error
		// It should never panic or crash
		_ = err // Error is acceptable for malformed input - just ignore it

		// Validate that any successfully parsed messages have reasonable values
		for i, msg := range messages {
			// Basic sanity checks - these shouldn't cause panics
			_ = msg.GetDate()
			_ = msg.GetAddress()
			_ = msg.GetType()
			_ = msg.GetReadableDate()
			_ = msg.GetContactName()

			// Check for obviously invalid data that should have been caught
			if msg.GetDate() < 0 {
				t.Errorf("Message %d has negative timestamp: %d", i, msg.GetDate())
			}

			// Message type should be within reasonable bounds
			if int(msg.GetType()) < 0 || int(msg.GetType()) > 10 {
				t.Logf("Message %d has unusual type: %d (input: %q)", i, msg.GetType(), xmlData)
			}

			// Type-specific validation
			switch m := msg.(type) {
			case SMS:
				// SMS-specific validation
				if len(m.Body) > 1000000 { // 1MB limit for SMS body
					t.Errorf("SMS %d has excessively large body: %d chars", i, len(m.Body))
				}
			case MMS:
				// MMS-specific validation - check parts don't cause issues
				for j, part := range m.Parts {
					if len(part.ContentType) > 1000 {
						t.Errorf("MMS Message %d, Part %d has excessively long content type", i, j)
					}
					if len(part.Data) > 100*1024*1024 { // 100MB limit
						t.Errorf("MMS Part %d data excessively large: %d bytes", j, len(part.Data))
					}
					if len(part.Text) > 1024*1024 { // 1MB text limit
						t.Errorf("MMS Part %d text excessively large: %d chars", j, len(part.Text))
					}
				}
			}
		}
	})
}

// FuzzAttachmentExtraction tests attachment extraction with random data
func FuzzAttachmentExtraction(f *testing.F) {
	// Seed with valid MMS structures containing parts
	f.Add(`<smses><mms><parts><part ct="image/png" data="iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="/></parts></mms></smses>`)
	f.Add(`<smses><mms><parts><part ct="text/plain" text="hello"/></parts></mms></smses>`)
	f.Add(`<smses><mms><parts><part ct="application/smil" data="invalid_base64"/></parts></mms></smses>`)

	f.Fuzz(func(t *testing.T, xmlData string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Attachment extraction panicked with input %q: %v", xmlData, r)
			}
		}()

		reader := strings.NewReader(xmlData)
		xmlReader := NewXMLSMSReader("/tmp")

		// Try to read messages with potential attachments
		err := xmlReader.StreamMessagesFromReader(reader, func(msg Message) error {
			// Process MMS parts safely if it's an MMS message
			if mms, ok := msg.(MMS); ok {
				for _, part := range mms.Parts {
					// These operations should not panic
					_ = part.ContentType
					_ = part.Data
					_ = part.Text

					// Validate data lengths are reasonable
					if len(part.Data) > 100*1024*1024 { // 100MB limit
						t.Errorf("Part data excessively large: %d bytes", len(part.Data))
					}
					if len(part.Text) > 1024*1024 { // 1MB text limit
						t.Errorf("Part text excessively large: %d chars", len(part.Text))
					}
				}
			}
			return nil
		})

		// Error is acceptable, panic is not
		_ = err
	})
}
