package sms

import "time"

// MessageType represents the type of message
type MessageType int

const (
	// ReceivedMessage represents a received message
	ReceivedMessage MessageType = 1
	// SentMessage represents a sent message
	SentMessage MessageType = 2
)

// Message is the base interface for SMS and MMS
type Message interface {
	GetDate() int64 // Epoch milliseconds
	GetAddress() string
	GetType() MessageType
	GetReadableDate() string
	GetContactName() string
}

// SMS represents a simple text message
type SMS struct {
	Protocol      string      `xml:"protocol,attr"`
	Address       string      `xml:"address,attr"`
	Date          int64       `xml:"date,attr"` // Epoch milliseconds
	Type          MessageType `xml:"type,attr"`
	Subject       string      `xml:"subject,attr"`
	Body          string      `xml:"body,attr"`
	ServiceCenter string      `xml:"service_center,attr"`
	Read          int         `xml:"read,attr"`
	Status        int         `xml:"status,attr"`
	Locked        int         `xml:"locked,attr"`
	DateSent      int64       `xml:"date_sent,attr"`
	ReadableDate  string      `xml:"readable_date,attr"`
	ContactName   string      `xml:"contact_name,attr"`
	Toa           string      `xml:"toa,attr"`
	ScToa         string      `xml:"sc_toa,attr"`
}

// GetDate implements Message interface
func (s SMS) GetDate() int64 {
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

// Timestamp returns the SMS timestamp as time.Time
func (s *SMS) Timestamp() time.Time {
	return time.Unix(s.Date/1000, (s.Date%1000)*int64(time.Millisecond))
}

// MMS represents a multimedia message
type MMS struct {
	Date         int64        `xml:"date,attr"` // Epoch milliseconds
	MsgBox       int          `xml:"msg_box,attr"`
	Address      string       `xml:"address,attr"` // Primary address
	MType        int          `xml:"m_type,attr"`
	MId          string       `xml:"m_id,attr"`
	ThreadID     string       `xml:"thread_id,attr"`
	TextOnly     int          `xml:"text_only,attr"`
	Sub          string       `xml:"sub,attr"`
	Parts        []MMSPart    `xml:"parts>part"`
	Addresses    []MMSAddress `xml:"addrs>addr"`
	ReadableDate string       `xml:"readable_date,attr"`
	ContactName  string       `xml:"contact_name,attr"`
	CallbackSet  int          `xml:"callback_set,attr"`
	RetrSt       int          `xml:"retr_st,attr"`
	CtCls        string       `xml:"ct_cls,attr"`
	SubCs        int          `xml:"sub_cs,attr"`
	Read         int          `xml:"read,attr"`
	CtL          string       `xml:"ct_l,attr"`
	TrID         string       `xml:"tr_id,attr"`
	St           int          `xml:"st,attr"`
	MCls         string       `xml:"m_cls,attr"`
	DTm          int64        `xml:"d_tm,attr"`
	ReadStatus   int          `xml:"read_status,attr"`
	CtT          string       `xml:"ct_t,attr"`
	RetrTxtCs    int          `xml:"retr_txt_cs,attr"`
	Deletable    int          `xml:"deletable,attr"`
	DRpt         int          `xml:"d_rpt,attr"`
	DateSent     int64        `xml:"date_sent,attr"`
	Seen         int          `xml:"seen,attr"`
	Reserved     int          `xml:"reserved,attr"`
	V            int          `xml:"v,attr"`
	Exp          int64        `xml:"exp,attr"`
	Pri          int          `xml:"pri,attr"`
	Hidden       int          `xml:"hidden,attr"`
	MsgID        int          `xml:"msg_id,attr"`
	Rr           int          `xml:"rr,attr"`
	AppID        int          `xml:"app_id,attr"`
	RespTxt      string       `xml:"resp_txt,attr"`
	RptA         int          `xml:"rpt_a,attr"`
	Locked       int          `xml:"locked,attr"`
	RetrTxt      string       `xml:"retr_txt,attr"`
	RespSt       int          `xml:"resp_st,attr"`
	MSize        int          `xml:"m_size,attr"`
	SimImsi      string       `xml:"sim_imsi,attr"`
	Creator      string       `xml:"creator,attr"`
	SubID        int          `xml:"sub_id,attr"`
	SimSlot      int          `xml:"sim_slot,attr"`
	SpamReport   int          `xml:"spam_report,attr"`
	SafeMessage  int          `xml:"safe_message,attr"`
}

// GetDate implements Message interface
func (m MMS) GetDate() int64 {
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

// Timestamp returns the MMS timestamp as time.Time
func (m *MMS) Timestamp() time.Time {
	return time.Unix(m.Date/1000, (m.Date%1000)*int64(time.Millisecond))
}

// MMSPart represents a content part of an MMS
type MMSPart struct {
	Seq            int    `xml:"seq,attr"`
	ContentType    string `xml:"ct,attr"`
	Name           string `xml:"name,attr"`
	Charset        string `xml:"chset,attr"`
	ContentDisp    string `xml:"cd,attr"`
	Filename       string `xml:"fn,attr"`
	ContentID      string `xml:"cid,attr"`
	ContentLoc     string `xml:"cl,attr"`
	CttS           int    `xml:"ctt_s,attr"`
	CttT           int    `xml:"ctt_t,attr"`
	Text           string `xml:"text,attr"`
	Data           string `xml:"data,attr"`            // Base64 encoded data
	Path           string `xml:"path,attr"`            // Repository-relative path to extracted attachment
	OriginalSize   int64  `xml:"original_size,attr"`   // Size of decoded attachment in bytes
	ExtractionDate string `xml:"extraction_date,attr"` // When attachment was extracted (ISO8601)
	AttachmentRef  string `xml:"AttachmentRef"`        // Attachment reference path for validation
}

// MMSAddress represents an address in an MMS
type MMSAddress struct {
	Address string `xml:"address,attr"`
	Type    int    `xml:"type,attr"`
	Charset int    `xml:"charset,attr"`
}
