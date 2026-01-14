package sms

import (
	"encoding/xml"
	"strings"
	"testing"
)

// Tests for extractHashFromAttachmentPath (0% coverage)

func TestExtractHashFromAttachmentPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "valid path with filename",
			path:     "attachments/ab/abc123def456abc123def456abc123def456abc123def456abc123def456abc1/image.jpg",
			expected: "abc123def456abc123def456abc123def456abc123def456abc123def456abc1",
		},
		{
			name:     "valid path without filename",
			path:     "attachments/ab/abc123def456abc123def456abc123def456abc123def456abc123def456abc1",
			expected: "abc123def456abc123def456abc123def456abc123def456abc123def456abc1",
		},
		{
			name:     "empty path",
			path:     "",
			expected: "",
		},
		{
			name:     "invalid format - no attachments prefix",
			path:     "files/ab/abc123/image.jpg",
			expected: "",
		},
		{
			name:     "invalid format - too few parts",
			path:     "attachments/ab",
			expected: "",
		},
		{
			name:     "invalid format - hash too short",
			path:     "attachments/ab/abc123/image.jpg",
			expected: "",
		},
		{
			name:     "invalid format - hash too long",
			path:     "attachments/ab/abc123def456abc123def456abc123def456abc123def456abc123def456abc1234/image.jpg",
			expected: "",
		},
		{
			name:     "invalid format - hash with uppercase",
			path:     "attachments/ab/ABC123def456abc123def456abc123def456abc123def456abc123def456abc1/image.jpg",
			expected: "",
		},
		{
			name:     "invalid format - hash with non-hex characters",
			path:     "attachments/ab/zzz123def456abc123def456abc123def456abc123def456abc123def456abc1/image.jpg",
			expected: "",
		},
		{
			name:     "invalid format - hash with special characters",
			path:     "attachments/ab/abc-23def456abc123def456abc123def456abc123def456abc123def456abc1/image.jpg",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := extractHashFromAttachmentPath(tc.path)
			if result != tc.expected {
				t.Errorf("extractHashFromAttachmentPath(%q) = %q, expected %q", tc.path, result, tc.expected)
			}
		})
	}
}

// Tests for parseMMSPartChildElement (0% coverage)

func TestParseMMSPartChildElement_AttachmentRef(t *testing.T) {
	t.Parallel()

	xmlData := `<AttachmentRef>attachments/ab/abc123def456abc123def456abc123def456abc123def456abc123def456abc1/image.jpg</AttachmentRef>`
	decoder := xml.NewDecoder(strings.NewReader(xmlData))

	// Read the start element
	var startElement xml.StartElement
	for {
		token, err := decoder.Token()
		if err != nil {
			t.Fatal(err)
		}
		if se, ok := token.(xml.StartElement); ok {
			startElement = se
			break
		}
	}

	reader := NewXMLSMSReader("/test")
	part := &MMSPart{}

	err := reader.parseMMSPartChildElement(decoder, part, startElement)
	if err != nil {
		t.Fatalf("parseMMSPartChildElement failed: %v", err)
	}

	expectedRef := "attachments/ab/abc123def456abc123def456abc123def456abc123def456abc123def456abc1/image.jpg"
	if part.AttachmentRef != expectedRef {
		t.Errorf("Expected AttachmentRef %q, got %q", expectedRef, part.AttachmentRef)
	}
}

func TestParseMMSPartChildElement_UnknownElement(t *testing.T) {
	t.Parallel()

	xmlData := `<UnknownElement>some data</UnknownElement>`
	decoder := xml.NewDecoder(strings.NewReader(xmlData))

	// Read the start element
	var startElement xml.StartElement
	for {
		token, err := decoder.Token()
		if err != nil {
			t.Fatal(err)
		}
		if se, ok := token.(xml.StartElement); ok {
			startElement = se
			break
		}
	}

	reader := NewXMLSMSReader("/test")
	part := &MMSPart{}

	err := reader.parseMMSPartChildElement(decoder, part, startElement)
	if err != nil {
		t.Fatalf("parseMMSPartChildElement should handle unknown elements: %v", err)
	}

	// AttachmentRef should still be empty
	if part.AttachmentRef != "" {
		t.Error("AttachmentRef should remain empty for unknown elements")
	}
}

// Tests for parseMMSPartNumericAttributes (33.3% coverage)

func TestParseMMSPartNumericAttributes_AllAttributes(t *testing.T) {
	t.Parallel()

	part := &MMSPart{}
	reader := NewXMLSMSReader("/test")

	// Test ctt_s
	err := reader.parseMMSPartNumericAttributes(part, "ctt_s", "100")
	if err != nil {
		t.Fatalf("parseMMSPartNumericAttributes failed for ctt_s: %v", err)
	}
	if part.CttS != 100 {
		t.Errorf("Expected CttS 100, got %d", part.CttS)
	}

	// Test ctt_t
	err = reader.parseMMSPartNumericAttributes(part, "ctt_t", "200")
	if err != nil {
		t.Fatalf("parseMMSPartNumericAttributes failed for ctt_t: %v", err)
	}
	if part.CttT != 200 {
		t.Errorf("Expected CttT 200, got %d", part.CttT)
	}
}

func TestParseMMSPartNumericAttributes_EmptyValue(t *testing.T) {
	t.Parallel()

	part := &MMSPart{}
	reader := NewXMLSMSReader("/test")

	// Empty value should not error
	err := reader.parseMMSPartNumericAttributes(part, "ctt_s", "")
	if err != nil {
		t.Error("Empty value should not error")
	}
}

func TestParseMMSPartNumericAttributes_NullValue(t *testing.T) {
	t.Parallel()

	part := &MMSPart{}
	reader := NewXMLSMSReader("/test")

	// "null" value should not error
	err := reader.parseMMSPartNumericAttributes(part, "ctt_s", "null")
	if err != nil {
		t.Error("Null value should not error")
	}
}

func TestParseMMSPartNumericAttributes_InvalidCttS(t *testing.T) {
	t.Parallel()

	part := &MMSPart{}
	reader := NewXMLSMSReader("/test")

	err := reader.parseMMSPartNumericAttributes(part, "ctt_s", "invalid")
	if err == nil {
		t.Error("Expected error for invalid ctt_s value")
	}
}

func TestParseMMSPartNumericAttributes_InvalidCttT(t *testing.T) {
	t.Parallel()

	part := &MMSPart{}
	reader := NewXMLSMSReader("/test")

	err := reader.parseMMSPartNumericAttributes(part, "ctt_t", "not-a-number")
	if err == nil {
		t.Error("Expected error for invalid ctt_t value")
	}
}

// Tests for parseMMSOptionalStringAttributes (58.3% coverage)

func TestParseMMSOptionalStringAttributes_AllAttributes(t *testing.T) {
	t.Parallel()

	mms := &MMS{}
	reader := NewXMLSMSReader("/test")

	// Test all supported attributes
	reader.parseMMSOptionalStringAttributes(mms, "sub", "test subject")
	reader.parseMMSOptionalStringAttributes(mms, "sim_imsi", "123456789")
	reader.parseMMSOptionalStringAttributes(mms, "creator", "user@example.com")
	reader.parseMMSOptionalStringAttributes(mms, "ct_cls", "class")
	reader.parseMMSOptionalStringAttributes(mms, "ct_l", "location")
	reader.parseMMSOptionalStringAttributes(mms, "tr_id", "trans-67890")
	reader.parseMMSOptionalStringAttributes(mms, "m_cls", "message-class")
	reader.parseMMSOptionalStringAttributes(mms, "ct_t", "content-type")
	reader.parseMMSOptionalStringAttributes(mms, "resp_txt", "OK")

	if mms.Sub != "test subject" {
		t.Errorf("Expected Sub 'test subject', got %q", mms.Sub)
	}
	if mms.SimImsi != "123456789" {
		t.Errorf("Expected SimImsi '123456789', got %q", mms.SimImsi)
	}
	if mms.Creator != "user@example.com" {
		t.Errorf("Expected Creator 'user@example.com', got %q", mms.Creator)
	}
	if mms.CtCls != "class" {
		t.Errorf("Expected CtCls 'class', got %q", mms.CtCls)
	}
	if mms.CtL != "location" {
		t.Errorf("Expected CtL 'location', got %q", mms.CtL)
	}
	if mms.TrID != "trans-67890" {
		t.Errorf("Expected TrID 'trans-67890', got %q", mms.TrID)
	}
	if mms.MCls != "message-class" {
		t.Errorf("Expected MCls 'message-class', got %q", mms.MCls)
	}
	if mms.CtT != "content-type" {
		t.Errorf("Expected CtT 'content-type', got %q", mms.CtT)
	}
	if mms.RespTxt != "OK" {
		t.Errorf("Expected RespTxt 'OK', got %q", mms.RespTxt)
	}
}

func TestParseMMSOptionalStringAttributes_NullValues(t *testing.T) {
	t.Parallel()

	mms := &MMS{}
	reader := NewXMLSMSReader("/test")

	// Set initial values
	mms.Creator = "initial"
	mms.Sub = "initial"

	// "null" values should not update fields
	reader.parseMMSOptionalStringAttributes(mms, "creator", "null")
	reader.parseMMSOptionalStringAttributes(mms, "sub", "null")

	if mms.Creator != "initial" {
		t.Errorf("Expected Creator 'initial' (null should not change), got %q", mms.Creator)
	}
	if mms.Sub != "initial" {
		t.Errorf("Expected Sub 'initial' (null should not change), got %q", mms.Sub)
	}
}

// Tests for GetAllAttachmentRefs (58.8% coverage)
// Note: Integration tests for GetAllAttachmentRefs are covered in existing test files
