package sms

import "time"

// MessageType represents the type of message
type MessageType int

const (
	ReceivedMessage MessageType = 1
	SentMessage     MessageType = 2
)

// Message is the base interface for SMS and MMS
type Message interface {
	GetDate() time.Time
	GetAddress() string
	GetType() MessageType
	GetReadableDate() string
	GetContactName() string
}

// SMS represents a simple text message
type SMS struct {
	Protocol      string
	Address       string
	Date          time.Time
	Type          MessageType
	Subject       string
	Body          string
	ServiceCenter string
	Read          bool
	Status        int
	Locked        bool
	DateSent      time.Time
	ReadableDate  string
	ContactName   string
}

// GetDate implements Message interface
func (s SMS) GetDate() time.Time {
	return s.Date
}

// GetAddress implements Message interface
func (s SMS) GetAddress() string {
	return s.Address
}

// GetType implements Message interface
func (s SMS) GetType() MessageType {
	return s.Type
}

// GetReadableDate implements Message interface
func (s SMS) GetReadableDate() string {
	return s.ReadableDate
}

// GetContactName implements Message interface
func (s SMS) GetContactName() string {
	return s.ContactName
}

// MMS represents a multimedia message
type MMS struct {
	Date         time.Time
	MsgBox       int
	Address      string // Primary address
	MType        int
	MId          string
	ThreadId     string
	TextOnly     bool
	Sub          string
	Parts        []MMSPart
	Addresses    []MMSAddress
	ReadableDate string
	ContactName  string
	CallbackSet  int
	RetrSt       string
	CtCls        string
	SubCs        string
	Read         bool
	CtL          string
	TrId         string
	St           string
	MCls         string
	DTm          string
	ReadStatus   string
	CtT          string
	RetrTxtCs    string
	Deletable    bool
	DRpt         int
	DateSent     time.Time
	Seen         bool
	Reserved     int
	V            int
	Exp          string
	Pri          int
	Hidden       bool
	MsgId        int
	Rr           int
	AppId        int
	RespTxt      string
	RptA         string
	Locked       bool
	RetrTxt      string
	RespSt       int
	MSize        int
}

// GetDate implements Message interface
func (m MMS) GetDate() time.Time {
	return m.Date
}

// GetAddress implements Message interface
func (m MMS) GetAddress() string {
	return m.Address
}

// GetType implements Message interface
func (m MMS) GetType() MessageType {
	if m.MsgBox == 1 {
		return ReceivedMessage
	}
	return SentMessage
}

// GetReadableDate implements Message interface
func (m MMS) GetReadableDate() string {
	return m.ReadableDate
}

// GetContactName implements Message interface
func (m MMS) GetContactName() string {
	return m.ContactName
}

// MMSPart represents a content part of an MMS
type MMSPart struct {
	Seq           int
	ContentType   string
	Name          string
	Charset       string
	ContentDisp   string
	Filename      string
	ContentId     string
	ContentLoc    string
	CttS          string
	CttT          string
	Text          string
	Data          string // Base64 encoded data
	AttachmentRef string // Path reference if attachment extracted
}

// MMSAddress represents an address in an MMS
type MMSAddress struct {
	Address string
	Type    int
	Charset int
}
