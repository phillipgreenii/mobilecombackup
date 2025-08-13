package attachments

import (
	"strings"
	"time"
)

// Attachment represents a stored attachment file
type Attachment struct {
	Hash   string // SHA-256 hash in lowercase hex
	Path   string // Relative path: attachments/ab/ab54363e39/attachment.ext
	Size   int64  // File size in bytes
	Exists bool   // Whether the file exists on disk
}

// AttachmentInfo represents attachment metadata
type AttachmentInfo struct {
	Hash         string    `yaml:"hash"`
	OriginalName string    `yaml:"original_name,omitempty"`
	MimeType     string    `yaml:"mime_type"`
	Size         int64     `yaml:"size"`
	CreatedAt    time.Time `yaml:"created_at"`
	SourceMMS    string    `yaml:"source_mms,omitempty"`
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

// AttachmentStorage interface with directory support
type AttachmentStorage interface {
	Store(hash string, data []byte, metadata AttachmentInfo) error
	GetPath(hash string) (string, error)
	GetMetadata(hash string) (AttachmentInfo, error)
	Exists(hash string) bool
}

// AttachmentStats provides statistics about attachments
type AttachmentStats struct {
	TotalCount     int   // Total number of attachments
	TotalSize      int64 // Total size of all attachments in bytes
	OrphanedCount  int   // Number of attachments not referenced by any SMS
	CorruptedCount int   // Number of attachments with hash mismatches
}

// MIME type to extension mapping
var MimeExtensions = map[string]string{
	// Images
	"image/png":  "png",
	"image/jpeg": "jpg",
	"image/jpg":  "jpg", // Non-standard variant
	"image/gif":  "gif",
	"image/bmp":  "bmp",
	"image/webp": "webp",
	"image/tiff": "tiff",
	"image/tif":  "tif",

	// Videos
	"video/mp4":       "mp4",
	"video/3gpp":      "3gp",
	"video/quicktime": "mov",
	"video/avi":       "avi",
	"video/mov":       "mov",
	"video/wmv":       "wmv",
	"video/flv":       "flv",

	// Audio
	"audio/mpeg": "mp3",
	"audio/mp3":  "mp3", // Non-standard but commonly used
	"audio/mp4":  "m4a",
	"audio/amr":  "amr",
	"audio/wav":  "wav",
	"audio/ogg":  "ogg",
	"audio/aac":  "aac",
	"audio/m4a":  "m4a",

	// Documents
	"application/pdf": "pdf",
	"application/zip": "zip",
	"application/rar": "rar",
	"application/7z":  "7z",

	// Microsoft Office
	"application/msword":            "doc",
	"application/vnd.ms-excel":      "xls",
	"application/vnd.ms-powerpoint": "ppt",

	// Microsoft Office XML formats
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   "docx",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         "xlsx",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": "pptx",

	// Generic binary
	"application/octet-stream": "bin",
}

// GetFileExtension returns the appropriate file extension for a MIME type
func GetFileExtension(mimeType string) string {
	// Normalize MIME type (remove parameters like charset)
	normalizedType := strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0]))

	if ext, exists := MimeExtensions[normalizedType]; exists {
		return ext
	}

	// Default to .bin for unknown types
	return "bin"
}

// GenerateFilename creates a filename based on original name or MIME type
func GenerateFilename(originalName, mimeType string) string {
	if originalName != "" && originalName != "null" {
		// Use original filename if available
		return originalName
	}

	// Generate generic filename with proper extension
	ext := GetFileExtension(mimeType)
	return "attachment." + ext
}
