// Package sms provides SMS and MMS message processing capabilities.
package sms

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/coalescer"
)

// MessageEntry wraps a Message to implement the coalescer.Entry interface
type MessageEntry struct {
	Message
}

// Hash returns a unique identifier for deduplication
// Excludes readable_date and contact_name fields as per requirement
func (m MessageEntry) Hash() string {
	h := sha256.New()

	// Common fields for both SMS and MMS
	_, _ = fmt.Fprintf(h, "address:%s|", m.GetAddress())
	_, _ = fmt.Fprintf(h, "date:%d|", m.GetDate())
	_, _ = fmt.Fprintf(h, "type:%d|", m.GetType())

	// Type-specific fields
	switch msg := m.Message.(type) {
	case *SMS:
		_, _ = fmt.Fprintf(h, "msgtype:sms|")
		_, _ = fmt.Fprintf(h, "body:%s|", msg.Body)
		_, _ = fmt.Fprintf(h, "protocol:%s|", msg.Protocol)
		_, _ = fmt.Fprintf(h, "subject:%s|", msg.Subject)
		_, _ = fmt.Fprintf(h, "service_center:%s|", msg.ServiceCenter)
		_, _ = fmt.Fprintf(h, "read:%d|", msg.Read)
		_, _ = fmt.Fprintf(h, "status:%d|", msg.Status)
		_, _ = fmt.Fprintf(h, "locked:%d|", msg.Locked)
		_, _ = fmt.Fprintf(h, "date_sent:%d|", msg.DateSent)

	case *MMS:
		_, _ = fmt.Fprintf(h, "msgtype:mms|")
		_, _ = fmt.Fprintf(h, "msg_box:%d|", msg.MsgBox)
		_, _ = fmt.Fprintf(h, "m_id:%s|", msg.MId)
		_, _ = fmt.Fprintf(h, "m_type:%d|", msg.MType)
		// Include parts in hash for MMS uniqueness
		for i, part := range msg.Parts {
			_, _ = fmt.Fprintf(h, "part%d_seq:%d|", i, part.Seq)
			_, _ = fmt.Fprintf(h, "part%d_ct:%s|", i, part.ContentType)
			_, _ = fmt.Fprintf(h, "part%d_name:%s|", i, part.Name)
			_, _ = fmt.Fprintf(h, "part%d_text:%s|", i, part.Text)

			// Include attachment path if extracted, otherwise indicate presence of data
			if part.Path != "" {
				_, _ = fmt.Fprintf(h, "part%d_path:%s|", i, part.Path)
			} else if part.Data != "" {
				_, _ = fmt.Fprintf(h, "part%d_has_data:true|", i)
			}
		}
		// Include addresses for group messages
		for i, addr := range msg.Addresses {
			_, _ = fmt.Fprintf(h, "addr%d_address:%s|", i, addr.Address)
			_, _ = fmt.Fprintf(h, "addr%d_type:%d|", i, addr.Type)
			_, _ = fmt.Fprintf(h, "addr%d_charset:%d|", i, addr.Charset)
		}
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

// Timestamp returns the entry's timestamp for sorting
func (m MessageEntry) Timestamp() time.Time {
	return time.Unix(m.GetDate()/1000, (m.GetDate()%1000)*int64(time.Millisecond))
}

// Year returns the year for partitioning
func (m MessageEntry) Year() int {
	return m.Timestamp().UTC().Year()
}

// NewMessageCoalescer creates a new coalescer for messages
func NewMessageCoalescer() coalescer.Coalescer[MessageEntry] {
	return coalescer.NewCoalescer[MessageEntry]()
}

// NewMessageEntry creates a new MessageEntry from a Message
func NewMessageEntry(msg Message) MessageEntry {
	return MessageEntry{Message: msg}
}
