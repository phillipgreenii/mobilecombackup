package sms

import (
	"testing"
)

// Tests for Hash function with MMS messages

func TestMessageEntry_Hash_MMS(t *testing.T) {
	t.Parallel()

	t.Run("MMS with basic fields", func(t *testing.T) {
		mms := &MMS{
			Address: "1234567890",
			Date:    1234567890000,
			MsgBox:  1,
			MId:     "test-mms-id",
			MType:   132,
		}
		entry := NewMessageEntry(mms)
		hash := entry.Hash()

		if len(hash) != 64 {
			t.Errorf("Expected hash length 64, got %d", len(hash))
		}
	})

	t.Run("MMS with parts containing Path", func(t *testing.T) {
		mms := &MMS{
			Address: "1234567890",
			Date:    1234567890000,
			MsgBox:  1,
			MId:     "test-mms-id",
			MType:   132,
			Parts: []MMSPart{
				{
					Seq:         0,
					ContentType: "image/jpeg",
					Name:        "photo.jpg",
					Text:        "",
					Path:        "attachments/abc/abc123.jpg",
				},
			},
		}
		entry := NewMessageEntry(mms)
		hash1 := entry.Hash()

		// Create another MMS with different path
		mms2 := &MMS{
			Address: "1234567890",
			Date:    1234567890000,
			MsgBox:  1,
			MId:     "test-mms-id",
			MType:   132,
			Parts: []MMSPart{
				{
					Seq:         0,
					ContentType: "image/jpeg",
					Name:        "photo.jpg",
					Text:        "",
					Path:        "attachments/def/def456.jpg",
				},
			},
		}
		entry2 := NewMessageEntry(mms2)
		hash2 := entry2.Hash()

		if hash1 == hash2 {
			t.Error("Expected different hashes for MMS with different paths")
		}
	})

	t.Run("MMS with parts containing Data", func(t *testing.T) {
		mms := &MMS{
			Address: "1234567890",
			Date:    1234567890000,
			MsgBox:  1,
			MId:     "test-mms-id",
			MType:   132,
			Parts: []MMSPart{
				{
					Seq:         0,
					ContentType: "text/plain",
					Name:        "message.txt",
					Text:        "Hello World",
					Data:        "SGVsbG8gV29ybGQ=", // base64 encoded data
				},
			},
		}
		entry := NewMessageEntry(mms)
		hash1 := entry.Hash()

		// Create another MMS without data
		mms2 := &MMS{
			Address: "1234567890",
			Date:    1234567890000,
			MsgBox:  1,
			MId:     "test-mms-id",
			MType:   132,
			Parts: []MMSPart{
				{
					Seq:         0,
					ContentType: "text/plain",
					Name:        "message.txt",
					Text:        "Hello World",
				},
			},
		}
		entry2 := NewMessageEntry(mms2)
		hash2 := entry2.Hash()

		if hash1 == hash2 {
			t.Error("Expected different hashes for MMS with and without data")
		}
	})

	t.Run("MMS with multiple parts", func(t *testing.T) {
		mms := &MMS{
			Address: "1234567890",
			Date:    1234567890000,
			MsgBox:  1,
			MId:     "test-mms-id",
			MType:   132,
			Parts: []MMSPart{
				{
					Seq:         0,
					ContentType: "text/plain",
					Name:        "text.txt",
					Text:        "Message text",
				},
				{
					Seq:         1,
					ContentType: "image/jpeg",
					Name:        "image.jpg",
					Path:        "attachments/xyz/xyz789.jpg",
				},
			},
		}
		entry := NewMessageEntry(mms)
		hash := entry.Hash()

		if len(hash) != 64 {
			t.Errorf("Expected hash length 64, got %d", len(hash))
		}
	})

	t.Run("MMS with addresses", func(t *testing.T) {
		mms := &MMS{
			Address: "1234567890",
			Date:    1234567890000,
			MsgBox:  1,
			MId:     "test-mms-id",
			MType:   132,
			Addresses: []MMSAddress{
				{
					Address: "1111111111",
					Type:    137,
					Charset: 106,
				},
				{
					Address: "2222222222",
					Type:    151,
					Charset: 106,
				},
			},
		}
		entry := NewMessageEntry(mms)
		hash1 := entry.Hash()

		// Create MMS with different addresses
		mms2 := &MMS{
			Address: "1234567890",
			Date:    1234567890000,
			MsgBox:  1,
			MId:     "test-mms-id",
			MType:   132,
			Addresses: []MMSAddress{
				{
					Address: "3333333333",
					Type:    137,
					Charset: 106,
				},
			},
		}
		entry2 := NewMessageEntry(mms2)
		hash2 := entry2.Hash()

		if hash1 == hash2 {
			t.Error("Expected different hashes for MMS with different addresses")
		}
	})
}

func TestMessageEntry_Hash_SMS_AllFields(t *testing.T) {
	t.Parallel()

	t.Run("SMS with all optional fields", func(t *testing.T) {
		sms := &SMS{
			Address:       "1234567890",
			Date:          1234567890000,
			Type:          1,
			Body:          "Test message",
			Protocol:      "0",
			Subject:       "Test subject",
			ServiceCenter: "+12345678900",
			Read:          1,
			Status:        -1,
			Locked:        0,
			DateSent:      1234567890000,
		}
		entry := NewMessageEntry(sms)
		hash1 := entry.Hash()

		// Create SMS with different optional fields
		sms2 := &SMS{
			Address:       "1234567890",
			Date:          1234567890000,
			Type:          1,
			Body:          "Test message",
			Protocol:      "1", // Different protocol
			Subject:       "Test subject",
			ServiceCenter: "+12345678900",
			Read:          1,
			Status:        -1,
			Locked:        0,
			DateSent:      1234567890000,
		}
		entry2 := NewMessageEntry(sms2)
		hash2 := entry2.Hash()

		if hash1 == hash2 {
			t.Error("Expected different hashes for SMS with different protocol")
		}
	})

	t.Run("SMS with different status", func(t *testing.T) {
		sms1 := &SMS{
			Address: "1234567890",
			Date:    1234567890000,
			Type:    1,
			Body:    "Test",
			Status:  -1,
		}
		sms2 := &SMS{
			Address: "1234567890",
			Date:    1234567890000,
			Type:    1,
			Body:    "Test",
			Status:  0,
		}

		hash1 := NewMessageEntry(sms1).Hash()
		hash2 := NewMessageEntry(sms2).Hash()

		if hash1 == hash2 {
			t.Error("Expected different hashes for different status values")
		}
	})

	t.Run("SMS with different locked state", func(t *testing.T) {
		sms1 := &SMS{
			Address: "1234567890",
			Date:    1234567890000,
			Type:    1,
			Body:    "Test",
			Locked:  0,
		}
		sms2 := &SMS{
			Address: "1234567890",
			Date:    1234567890000,
			Type:    1,
			Body:    "Test",
			Locked:  1,
		}

		hash1 := NewMessageEntry(sms1).Hash()
		hash2 := NewMessageEntry(sms2).Hash()

		if hash1 == hash2 {
			t.Error("Expected different hashes for different locked states")
		}
	})

	t.Run("SMS with different DateSent", func(t *testing.T) {
		sms1 := &SMS{
			Address:  "1234567890",
			Date:     1234567890000,
			Type:     1,
			Body:     "Test",
			DateSent: 1234567890000,
		}
		sms2 := &SMS{
			Address:  "1234567890",
			Date:     1234567890000,
			Type:     1,
			Body:     "Test",
			DateSent: 1234567891000,
		}

		hash1 := NewMessageEntry(sms1).Hash()
		hash2 := NewMessageEntry(sms2).Hash()

		if hash1 == hash2 {
			t.Error("Expected different hashes for different DateSent values")
		}
	})
}
