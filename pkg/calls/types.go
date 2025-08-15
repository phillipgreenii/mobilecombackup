package calls

import "time"

// CallType represents the type of call
type CallType int

const (
	// Incoming represents an incoming call
	Incoming CallType = 1
	// Outgoing represents an outgoing call
	Outgoing CallType = 2
	// Missed represents a missed call
	Missed CallType = 3
	// Voicemail represents a voicemail call
	Voicemail CallType = 4
)

// Call represents a single call record
type Call struct {
	Number       string   `xml:"number,attr"`
	Duration     int      `xml:"duration,attr"`
	Date         int64    `xml:"date,attr"` // Epoch milliseconds
	Type         CallType `xml:"type,attr"`
	ReadableDate string   `xml:"readable_date,attr"`
	ContactName  string   `xml:"contact_name,attr"`
}

// Timestamp returns the call's timestamp as time.Time
func (c *Call) Timestamp() time.Time {
	return time.Unix(c.Date/1000, (c.Date%1000)*int64(time.Millisecond))
}
