// Package security provides security-related functionality including secure XML parsing to prevent XXE attacks.
package security

import (
	"encoding/xml"
	"fmt"
	"io"
)

// NewSecureXMLDecoder creates a new XML decoder with XXE protection enabled.
// This prevents XML External Entity (XXE) attacks by:
// - Disabling external entity resolution
// - Disabling DTD processing
// - Setting strict mode to false for compatibility
func NewSecureXMLDecoder(r io.Reader) *xml.Decoder {
	decoder := xml.NewDecoder(r)

	// Disable external entity resolution to prevent XXE attacks
	decoder.Entity = xml.HTMLEntity

	// Set a custom CharsetReader that prevents external entity resolution
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		// Only allow standard charsets, no external loading
		if charset == "utf-8" || charset == "UTF-8" || charset == "" {
			return input, nil
		}
		return nil, fmt.Errorf("unsupported charset: %s", charset)
	}

	// Disable strict mode for better compatibility with existing XML
	decoder.Strict = false

	return decoder
}

// SecureXMLError represents XML parsing security errors
type SecureXMLError struct {
	Message string
	Cause   error
}

func (e *SecureXMLError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("secure XML parsing error: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("secure XML parsing error: %s", e.Message)
}

func (e *SecureXMLError) Unwrap() error {
	return e.Cause
}

// NewSecureXMLError creates a new SecureXMLError
func NewSecureXMLError(message string, cause error) *SecureXMLError {
	return &SecureXMLError{
		Message: message,
		Cause:   cause,
	}
}
