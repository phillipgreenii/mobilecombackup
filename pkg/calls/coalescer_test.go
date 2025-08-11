package calls

import (
	"testing"
	"time"
)

func TestCallEntry_Hash(t *testing.T) {
	tests := []struct {
		name     string
		call1    *Call
		call2    *Call
		sameHash bool
	}{
		{
			name: "identical calls have same hash",
			call1: &Call{
				Number:       "+1234567890",
				Duration:     120,
				Date:         1609459200000, // 2021-01-01 00:00:00
				Type:         Incoming,
				ReadableDate: "Jan 1, 2021 12:00:00 AM",
				ContactName:  "John Doe",
			},
			call2: &Call{
				Number:       "+1234567890",
				Duration:     120,
				Date:         1609459200000,
				Type:         Incoming,
				ReadableDate: "Jan 1, 2021 12:00:00 AM",
				ContactName:  "John Doe",
			},
			sameHash: true,
		},
		{
			name: "different readable_date same hash",
			call1: &Call{
				Number:       "+1234567890",
				Duration:     120,
				Date:         1609459200000,
				Type:         Incoming,
				ReadableDate: "Jan 1, 2021 12:00:00 AM",
				ContactName:  "John Doe",
			},
			call2: &Call{
				Number:       "+1234567890",
				Duration:     120,
				Date:         1609459200000,
				Type:         Incoming,
				ReadableDate: "Jan 1, 2021 12:00:00 AM EST", // Different timezone format
				ContactName:  "John Doe",
			},
			sameHash: true,
		},
		{
			name: "different contact_name same hash",
			call1: &Call{
				Number:       "+1234567890",
				Duration:     120,
				Date:         1609459200000,
				Type:         Incoming,
				ReadableDate: "Jan 1, 2021 12:00:00 AM",
				ContactName:  "John Doe",
			},
			call2: &Call{
				Number:       "+1234567890",
				Duration:     120,
				Date:         1609459200000,
				Type:         Incoming,
				ReadableDate: "Jan 1, 2021 12:00:00 AM",
				ContactName:  "Jane Smith", // Different contact name
			},
			sameHash: true,
		},
		{
			name: "different number different hash",
			call1: &Call{
				Number:       "+1234567890",
				Duration:     120,
				Date:         1609459200000,
				Type:         Incoming,
				ReadableDate: "Jan 1, 2021 12:00:00 AM",
				ContactName:  "John Doe",
			},
			call2: &Call{
				Number:       "+0987654321", // Different number
				Duration:     120,
				Date:         1609459200000,
				Type:         Incoming,
				ReadableDate: "Jan 1, 2021 12:00:00 AM",
				ContactName:  "John Doe",
			},
			sameHash: false,
		},
		{
			name: "different duration different hash",
			call1: &Call{
				Number:       "+1234567890",
				Duration:     120,
				Date:         1609459200000,
				Type:         Incoming,
				ReadableDate: "Jan 1, 2021 12:00:00 AM",
				ContactName:  "John Doe",
			},
			call2: &Call{
				Number:       "+1234567890",
				Duration:     180, // Different duration
				Date:         1609459200000,
				Type:         Incoming,
				ReadableDate: "Jan 1, 2021 12:00:00 AM",
				ContactName:  "John Doe",
			},
			sameHash: false,
		},
		{
			name: "different date different hash",
			call1: &Call{
				Number:       "+1234567890",
				Duration:     120,
				Date:         1609459200000,
				Type:         Incoming,
				ReadableDate: "Jan 1, 2021 12:00:00 AM",
				ContactName:  "John Doe",
			},
			call2: &Call{
				Number:       "+1234567890",
				Duration:     120,
				Date:         1609545600000, // Different date
				Type:         Incoming,
				ReadableDate: "Jan 2, 2021 12:00:00 AM",
				ContactName:  "John Doe",
			},
			sameHash: false,
		},
		{
			name: "different type different hash",
			call1: &Call{
				Number:       "+1234567890",
				Duration:     120,
				Date:         1609459200000,
				Type:         Incoming,
				ReadableDate: "Jan 1, 2021 12:00:00 AM",
				ContactName:  "John Doe",
			},
			call2: &Call{
				Number:       "+1234567890",
				Duration:     120,
				Date:         1609459200000,
				Type:         Outgoing, // Different type
				ReadableDate: "Jan 1, 2021 12:00:00 AM",
				ContactName:  "John Doe",
			},
			sameHash: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry1 := CallEntry{Call: tt.call1}
			entry2 := CallEntry{Call: tt.call2}
			
			hash1 := entry1.Hash()
			hash2 := entry2.Hash()
			
			if tt.sameHash && hash1 != hash2 {
				t.Errorf("Expected same hash, got different: %s != %s", hash1, hash2)
			}
			if !tt.sameHash && hash1 == hash2 {
				t.Errorf("Expected different hash, got same: %s", hash1)
			}
		})
	}
}

func TestCallEntry_Timestamp(t *testing.T) {
	call := &Call{
		Date: 1609459200000, // 2021-01-01 00:00:00 UTC in milliseconds
	}
	
	entry := CallEntry{Call: call}
	ts := entry.Timestamp()
	
	expected := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	if !ts.Equal(expected) {
		t.Errorf("Expected timestamp %v, got %v", expected, ts)
	}
}

func TestCallEntry_Year(t *testing.T) {
	tests := []struct {
		name     string
		date     int64
		expected int
	}{
		{
			name:     "year 2021",
			date:     1609459200000, // 2021-01-01 00:00:00 UTC
			expected: 2021,
		},
		{
			name:     "year 2023",
			date:     1672531200000, // 2023-01-01 00:00:00 UTC
			expected: 2023,
		},
		{
			name:     "year boundary - end of 2022",
			date:     1672531199000, // 2022-12-31 23:59:59 UTC
			expected: 2022,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			call := &Call{Date: tt.date}
			entry := CallEntry{Call: call}
			
			year := entry.Year()
			if year != tt.expected {
				t.Errorf("Expected year %d, got %d", tt.expected, year)
			}
		})
	}
}

func TestCallEntry_HashConsistency(t *testing.T) {
	// Test that hash is consistent across multiple calls
	call := &Call{
		Number:       "+1234567890",
		Duration:     120,
		Date:         1609459200000,
		Type:         Incoming,
		ReadableDate: "Jan 1, 2021 12:00:00 AM",
		ContactName:  "John Doe",
	}
	
	entry := CallEntry{Call: call}
	
	hash1 := entry.Hash()
	hash2 := entry.Hash()
	
	if hash1 != hash2 {
		t.Errorf("Hash not consistent: %s != %s", hash1, hash2)
	}
	
	// Verify hash format (should be 64 character hex string)
	if len(hash1) != 64 {
		t.Errorf("Hash length should be 64, got %d", len(hash1))
	}
	
	// Verify it's valid hex
	for _, c := range hash1 {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Errorf("Hash contains invalid character: %c", c)
		}
	}
}