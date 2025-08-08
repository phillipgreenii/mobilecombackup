package attachments

// Attachment represents a stored attachment file
type Attachment struct {
	Hash   string // SHA-256 hash in lowercase hex
	Path   string // Relative path: attachments/ab/ab54363e39
	Size   int64  // File size in bytes
	Exists bool   // Whether the file exists on disk
}

// AttachmentReader reads attachments from repository
type AttachmentReader interface {
	// GetAttachment retrieves attachment info by hash
	GetAttachment(hash string) (*Attachment, error)

	// ReadAttachment reads the actual file content
	ReadAttachment(hash string) ([]byte, error)

	// AttachmentExists checks if attachment exists
	AttachmentExists(hash string) (bool, error)

	// ListAttachments returns all attachments in repository
	ListAttachments() ([]*Attachment, error)

	// StreamAttachments streams attachment info for memory efficiency
	StreamAttachments(callback func(*Attachment) error) error

	// VerifyAttachment checks if file content matches its hash
	VerifyAttachment(hash string) (bool, error)

	// GetAttachmentPath returns the expected path for a hash
	GetAttachmentPath(hash string) string

	// FindOrphanedAttachments returns attachments not referenced by any messages
	// Requires a set of referenced attachment hashes from SMS reader
	FindOrphanedAttachments(referencedHashes map[string]bool) ([]*Attachment, error)

	// ValidateAttachmentStructure validates the directory structure
	ValidateAttachmentStructure() error
}

// AttachmentStats provides statistics about attachments
type AttachmentStats struct {
	TotalCount     int   // Total number of attachments
	TotalSize      int64 // Total size of all attachments in bytes
	OrphanedCount  int   // Number of attachments not referenced by any SMS
	CorruptedCount int   // Number of attachments with hash mismatches
}
