package sms

import (
	"crypto/sha256"
	"fmt"
	"time"
	
	"github.com/phillipgreen/mobilecombackup/pkg/coalescer"
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
	fmt.Fprintf(h, "address:%s|", m.GetAddress())
	fmt.Fprintf(h, "date:%d|", m.GetDate())
	fmt.Fprintf(h, "type:%d|", m.GetType())
	
	// Type-specific fields
	switch msg := m.Message.(type) {
	case *SMS:
		fmt.Fprintf(h, "msgtype:sms|")
		fmt.Fprintf(h, "body:%s|", msg.Body)
		fmt.Fprintf(h, "protocol:%s|", msg.Protocol)
		fmt.Fprintf(h, "subject:%s|", msg.Subject)
		fmt.Fprintf(h, "service_center:%s|", msg.ServiceCenter)
		fmt.Fprintf(h, "read:%d|", msg.Read)
		fmt.Fprintf(h, "status:%d|", msg.Status)
		fmt.Fprintf(h, "locked:%d|", msg.Locked)
		fmt.Fprintf(h, "date_sent:%d|", msg.DateSent)
		
	case *MMS:
		fmt.Fprintf(h, "msgtype:mms|")
		fmt.Fprintf(h, "msg_box:%d|", msg.MsgBox)
		fmt.Fprintf(h, "m_id:%s|", msg.MId)
		fmt.Fprintf(h, "m_type:%d|", msg.MType)
		// Include parts in hash for MMS uniqueness
		for i, part := range msg.Parts {
			fmt.Fprintf(h, "part%d_seq:%d|", i, part.Seq)
			fmt.Fprintf(h, "part%d_ct:%s|", i, part.ContentType)
			fmt.Fprintf(h, "part%d_name:%s|", i, part.Name)
			fmt.Fprintf(h, "part%d_text:%s|", i, part.Text)
			// Don't include actual data content in hash (too large)
			if part.Data != "" {
				fmt.Fprintf(h, "part%d_has_data:true|", i)
			}
		}
		// Include addresses for group messages
		for i, addr := range msg.Addresses {
			fmt.Fprintf(h, "addr%d_address:%s|", i, addr.Address)
			fmt.Fprintf(h, "addr%d_type:%d|", i, addr.Type)
			fmt.Fprintf(h, "addr%d_charset:%d|", i, addr.Charset)
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