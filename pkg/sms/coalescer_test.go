package sms

import (
	"testing"
)

func TestMessageEntry_Hash(t *testing.T) {
	sms := &SMS{
		Address: "1234567890",
		Date:    1234567890000,
		Type:    1,
		Body:    "Test",
	}
	entry := NewMessageEntry(sms)
	hash := entry.Hash()
	if len(hash) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash))
	}
}

func TestMessageEntry_Timestamp(t *testing.T) {
	sms := &SMS{
		Date: 1234567890000,
	}
	entry := NewMessageEntry(sms)
	ts := entry.Timestamp()
	if ts.Unix() != 1234567890 {
		t.Errorf("Expected Unix timestamp 1234567890, got %d", ts.Unix())
	}
}

func TestMessageEntry_Year(t *testing.T) {
	sms := &SMS{
		Date: 1234567890000, // 2009
	}
	entry := NewMessageEntry(sms)
	year := entry.Year()
	if year != 2009 {
		t.Errorf("Expected year 2009, got %d", year)
	}
}

func TestNewMessageCoalescer(t *testing.T) {
	coalescer := NewMessageCoalescer()
	if coalescer == nil {
		t.Fatal("NewMessageCoalescer returned nil")
	}
}

func TestNewMessageEntry(t *testing.T) {
	sms := &SMS{
		Address: "test",
		Date:    1000000000000,
	}
	entry := NewMessageEntry(sms)
	if entry.Message == nil {
		t.Error("NewMessageEntry did not set Message")
	}
}
