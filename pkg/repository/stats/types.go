package stats

import "time"

// RepositoryStats contains comprehensive statistics about a repository
type RepositoryStats struct {
	Version      string
	CreatedAt    time.Time
	Calls        map[string]YearInfo
	SMS          map[string]MessageInfo
	Attachments  AttachmentInfo
	Contacts     ContactInfo
	Rejections   map[string]int
	Errors       map[string]int
	ValidationOK bool
}

// YearInfo contains statistics for a specific year of calls
type YearInfo struct {
	Count    int
	Earliest time.Time
	Latest   time.Time
}

// MessageInfo contains statistics for messages
type MessageInfo struct {
	TotalCount int
	SMSCount   int
	MMSCount   int
	Earliest   time.Time
	Latest     time.Time
}

// AttachmentInfo contains statistics about attachments
type AttachmentInfo struct {
	Count      int
	TotalSize  int64
	ByType     map[string]int
	Referenced int
	Orphaned   int
}

// ContactInfo contains statistics about contacts
type ContactInfo struct {
	Count      int
	WithNames  int
	PhoneOnly  int
	Unresolved int
}
