package importer

import (
	"reflect"
	"testing"

	"github.com/phillipgreen/mobilecombackup/pkg/calls"
)

func TestCallValidator_Validate(t *testing.T) {
	validator := NewCallValidator()

	tests := []struct {
		name       string
		call       *calls.Call
		violations []string
	}{
		{
			name: "valid call",
			call: &calls.Call{
				Number:   "+1234567890",
				Duration: 120,
				Date:     1609459200000,
				Type:     calls.Incoming,
			},
			violations: nil,
		},
		{
			name: "missing timestamp",
			call: &calls.Call{
				Number:   "+1234567890",
				Duration: 120,
				Date:     0,
				Type:     calls.Incoming,
			},
			violations: []string{"missing-timestamp"},
		},
		{
			name: "negative timestamp",
			call: &calls.Call{
				Number:   "+1234567890",
				Duration: 120,
				Date:     -1,
				Type:     calls.Incoming,
			},
			violations: []string{"missing-timestamp"},
		},
		{
			name: "missing number",
			call: &calls.Call{
				Number:   "",
				Duration: 120,
				Date:     1609459200000,
				Type:     calls.Incoming,
			},
			violations: []string{"missing-number"},
		},
		{
			name: "whitespace only number",
			call: &calls.Call{
				Number:   "   ",
				Duration: 120,
				Date:     1609459200000,
				Type:     calls.Incoming,
			},
			violations: []string{"missing-number"},
		},
		{
			name: "invalid call type",
			call: &calls.Call{
				Number:   "+1234567890",
				Duration: 120,
				Date:     1609459200000,
				Type:     99, // Invalid type
			},
			violations: []string{"invalid-type: 99"},
		},
		{
			name: "negative duration",
			call: &calls.Call{
				Number:   "+1234567890",
				Duration: -10,
				Date:     1609459200000,
				Type:     calls.Incoming,
			},
			violations: []string{"negative-duration"},
		},
		{
			name: "multiple violations",
			call: &calls.Call{
				Number:   "",
				Duration: -10,
				Date:     0,
				Type:     99,
			},
			violations: []string{
				"missing-timestamp",
				"missing-number",
				"invalid-type: 99",
				"negative-duration",
			},
		},
		{
			name: "valid incoming call",
			call: &calls.Call{
				Number:   "5551234567",
				Duration: 0, // Missed call can have 0 duration
				Date:     1609459200000,
				Type:     calls.Incoming,
			},
			violations: nil,
		},
		{
			name: "valid outgoing call",
			call: &calls.Call{
				Number:   "+15551234567",
				Duration: 300,
				Date:     1609459200000,
				Type:     calls.Outgoing,
			},
			violations: nil,
		},
		{
			name: "valid missed call",
			call: &calls.Call{
				Number:   "1234567",
				Duration: 0,
				Date:     1609459200000,
				Type:     calls.Missed,
			},
			violations: nil,
		},
		{
			name: "valid voicemail",
			call: &calls.Call{
				Number:   "+15551234567",
				Duration: 45,
				Date:     1609459200000,
				Type:     calls.Voicemail,
			},
			violations: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := validator.Validate(tt.call)

			if !reflect.DeepEqual(violations, tt.violations) {
				t.Errorf("Expected violations %v, got %v", tt.violations, violations)
			}
		})
	}
}

func TestCallValidator_PhoneNumberFormats(t *testing.T) {
	validator := NewCallValidator()

	// Test various phone number formats that should be valid
	validNumbers := []string{
		"1234567",         // Short number
		"5551234567",      // 10-digit US
		"15551234567",     // 11-digit US
		"+15551234567",    // International format
		"+441234567890",   // UK number
		"911",             // Emergency
		"*67",             // Special code
		"#31#",            // Special code with hash
		"+86138000138000", // China mobile
	}

	for _, number := range validNumbers {
		call := &calls.Call{
			Number:   number,
			Duration: 60,
			Date:     1609459200000,
			Type:     calls.Incoming,
		}

		violations := validator.Validate(call)
		if len(violations) > 0 {
			t.Errorf("Phone number %q should be valid, got violations: %v", number, violations)
		}
	}
}
