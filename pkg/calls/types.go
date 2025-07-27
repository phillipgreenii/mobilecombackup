package calls

import "time"

// CallType represents the type of call
type CallType int

const (
	IncomingCall  CallType = 1
	OutgoingCall  CallType = 2
	MissedCall    CallType = 3
	VoicemailCall CallType = 4
)

// Call represents a single call record
type Call struct {
	Number       string
	Duration     int
	Date         time.Time
	Type         CallType
	ReadableDate string
	ContactName  string
}