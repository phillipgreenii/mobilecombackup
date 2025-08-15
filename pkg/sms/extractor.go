package sms

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/attachments"
)

// Action constants for attachment extraction results
const (
	ActionExtracted  = "extracted"
	ActionReferenced = "referenced"
	ActionSkipped    = "skipped"
)

// Content type category constants
const (
	CategoryUnknown = "unknown"
)

// AttachmentExtractor handles extraction of attachments from SMS/MMS messages
type AttachmentExtractor struct {
	attachmentManager *attachments.AttachmentManager
	repoRoot          string
}

// NewAttachmentExtractor creates a new attachment extractor
func NewAttachmentExtractor(repoRoot string) *AttachmentExtractor {
	return &AttachmentExtractor{
		attachmentManager: attachments.NewAttachmentManager(repoRoot),
		repoRoot:          repoRoot,
	}
}

// BinaryContentTypes defines the explicit whitelist of binary content types for extraction
var BinaryContentTypes = map[string]bool{
	// Common image types - strict binary only
	"image/jpeg": true,
	"image/jpg":  true, // Some systems use this non-standard variant
	"image/png":  true,
	"image/gif":  true,
	"image/bmp":  true,
	"image/webp": true,
	"image/tiff": true,
	"image/tif":  true, // TIFF variant

	// Video types
	"video/mp4":       true,
	"video/3gpp":      true,
	"video/quicktime": true,
	"video/avi":       true,
	"video/mov":       true,
	"video/wmv":       true,
	"video/flv":       true,

	// Audio types
	"audio/mpeg": true,
	"audio/mp3":  true, // Non-standard but commonly used
	"audio/mp4":  true,
	"audio/amr":  true,
	"audio/wav":  true,
	"audio/ogg":  true,
	"audio/aac":  true,
	"audio/m4a":  true,

	// Document types
	"application/pdf": true,
	"application/zip": true,
	"application/rar": true,
	"application/7z":  true,

	// Microsoft Office binary formats
	"application/msword":            true, // .doc
	"application/vnd.ms-excel":      true, // .xls
	"application/vnd.ms-powerpoint": true, // .ppt

	// Microsoft Office XML formats
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   true, // .docx
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true, // .xlsx
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true, // .pptx

	// Other common binary formats
	"application/octet-stream": true, // Generic binary
}

// TextContentTypes defines text-based content that should remain inline (for logging)
var TextContentTypes = map[string]bool{
	"text/plain":                            true,
	"text/html":                             true,
	"text/xml":                              true,
	"text/css":                              true,
	"text/javascript":                       true,
	"text/csv":                              true,
	"text/rtf":                              true, // Rich Text Format
	"text/x-vcard":                          true,
	"text/vcard":                            true,
	"application/xml":                       true,
	"application/json":                      true,
	"application/javascript":                true,
	"application/smil":                      true,
	"application/xhtml+xml":                 true,
	"application/vnd.wap.multipart.related": true,
}

// ContentDecision represents the decision made about content extraction
type ContentDecision struct {
	ShouldExtract bool
	Reason        string
	ContentType   string
	Category      string // "binary", "text", CategoryUnknown
}

// ContentTypeConfig defines which content types to extract vs skip (maintained for backward compatibility)
type ContentTypeConfig struct {
	ExtractableTypes []string // Content types to extract
	SkippedTypes     []string // Content types to skip (leave inline)
}

// GetDefaultContentTypeConfig returns the default content type configuration (backward compatibility)
func GetDefaultContentTypeConfig() ContentTypeConfig {
	// Hardcode the original list to maintain exact backward compatibility
	return ContentTypeConfig{
		ExtractableTypes: []string{
			// Images
			"image/jpeg", "image/png", "image/gif", "image/bmp", "image/webp",
			// Videos
			"video/mp4", "video/3gpp", "video/quicktime", "video/avi",
			// Audio
			"audio/mpeg", "audio/mp4", "audio/amr", "audio/wav",
			// Documents
			"application/pdf", "application/msword",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		},
		SkippedTypes: []string{
			// System-generated content
			"application/smil",
			"text/plain",
			"text/x-vCard",
			"application/vnd.wap.multipart.related",
		},
	}
}

// shouldExtractContentType determines if a content type should be extracted with detailed decision logging
func (ae *AttachmentExtractor) shouldExtractContentType(contentType string, isExplicitAttachment bool, config ContentTypeConfig) ContentDecision {
	decision := ContentDecision{
		ContentType: contentType,
	}

	// Handle edge cases first
	if contentType == "" {
		decision.ShouldExtract = false
		decision.Reason = "missing content type header"
		decision.Category = CategoryUnknown
		log.Printf("[ATTACHMENT] Content type decision: %+v", decision)
		return decision
	}

	// Normalize content type (remove parameters like charset)
	normalizedType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))

	if normalizedType == "" {
		decision.ShouldExtract = false
		decision.Reason = "empty content type after normalization"
		decision.Category = CategoryUnknown
		log.Printf("[ATTACHMENT] Content type decision: %+v", decision)
		return decision
	}

	// Check against binary whitelist first
	if BinaryContentTypes[normalizedType] {
		decision.ShouldExtract = true
		decision.Reason = "whitelisted binary type"
		decision.Category = "binary"
		log.Printf("[ATTACHMENT] Content type decision: %+v", decision)
		return decision
	}

	// Check if it's a known text type
	if TextContentTypes[normalizedType] {
		decision.ShouldExtract = false
		decision.Reason = "text content - keeping inline"
		decision.Category = "text"
		log.Printf("[ATTACHMENT] Content type decision: %+v", decision)
		return decision
	}

	// Unknown content type - reject with detailed logging for manual review
	decision.ShouldExtract = false
	decision.Reason = fmt.Sprintf("unknown content type: %s", normalizedType)
	decision.Category = CategoryUnknown
	log.Printf("[ATTACHMENT] Content type decision: %+v (ExplicitAttachment: %t)", decision, isExplicitAttachment)
	return decision
}

// ExtractAttachmentFromPart extracts an attachment from an MMS part if needed
func (ae *AttachmentExtractor) ExtractAttachmentFromPart(part *MMSPart, config ContentTypeConfig) (*AttachmentExtractionResult, error) {
	// Log extraction attempt for debugging and auditing
	log.Printf("[ATTACHMENT] Processing part: ContentType=%s, DataLen=%d, TextLen=%d, ContentDisp=%s, Filename=%s",
		part.ContentType, len(part.Data), len(part.Text), part.ContentDisp, part.Filename)

	// Check if this is explicitly marked as an attachment
	isExplicitAttachment := strings.ToLower(part.ContentDisp) == "attachment"

	// Determine what content we have to extract
	var contentToExtract string
	var isBase64 bool

	if part.Data != "" && part.Data != "null" {
		// Binary data (base64 encoded)
		contentToExtract = part.Data
		isBase64 = true
		log.Printf("[ATTACHMENT] Found base64 data, size: %d bytes", len(contentToExtract))
	} else if part.Text != "" && part.Text != "null" && isExplicitAttachment {
		// Text content marked as attachment
		contentToExtract = part.Text
		isBase64 = false
		log.Printf("[ATTACHMENT] Found text content marked as attachment, size: %d bytes", len(contentToExtract))
	} else {
		// No extractable content - skip
		log.Printf("[ATTACHMENT] No extractable content found - Data='%s', Text='%s', ExplicitAttachment=%t",
			part.Data, part.Text, isExplicitAttachment)
		return &AttachmentExtractionResult{
			Action:     ActionSkipped,
			Reason:     "no-data",
			UpdatePart: false,
		}, nil
	}

	// Check content type extraction decision with comprehensive logging
	decision := ae.shouldExtractContentType(part.ContentType, isExplicitAttachment, config)
	if !decision.ShouldExtract {
		log.Printf("[ATTACHMENT] Content type filtering: %s - %s (category: %s)",
			decision.Reason, part.ContentType, decision.Category)
		return &AttachmentExtractionResult{
			Action:     ActionSkipped,
			Reason:     "content-type-filtered",
			UpdatePart: false,
		}, nil
	}

	log.Printf("[ATTACHMENT] Content type approved for extraction: %s - %s",
		part.ContentType, decision.Reason)

	// For text content, skip size check as text attachments can be small
	// For binary content, skip small files (likely metadata)
	if isBase64 && len(contentToExtract) < 1024 {
		// Too small - skip
		log.Printf("[ATTACHMENT] Skipping small binary content: %d bytes (threshold: 1024)", len(contentToExtract))
		return &AttachmentExtractionResult{
			Action:     ActionSkipped,
			Reason:     "too-small",
			UpdatePart: false,
		}, nil
	}

	// Proceed with extraction
	log.Printf("[ATTACHMENT] Proceeding with extraction for content type: %s", part.ContentType)

	// Decode content based on type
	var decodedData []byte
	var err error

	if isBase64 {
		// Decode base64 data
		log.Printf("[ATTACHMENT] Decoding base64 data (%d chars)", len(contentToExtract))
		decodedData, err = base64.StdEncoding.DecodeString(contentToExtract)
		if err != nil {
			// Failed to decode base64 data
			log.Printf("[ATTACHMENT] Failed to decode base64 data: %v", err)
			return nil, fmt.Errorf("failed to decode base64 data: %w", err)
		}
		log.Printf("[ATTACHMENT] Decoded to %d bytes", len(decodedData))
	} else {
		// Use text content as-is (UTF-8 encoded)
		decodedData = []byte(contentToExtract)
		log.Printf("[ATTACHMENT] Using text content as-is: %d bytes", len(decodedData))
	}

	// Calculate SHA-256 hash
	hasher := sha256.New()
	hasher.Write(decodedData)
	hash := fmt.Sprintf("%x", hasher.Sum(nil))
	log.Printf("[ATTACHMENT] Calculated hash: %s", hash)

	// Check if attachment already exists
	exists, err := ae.attachmentManager.AttachmentExists(hash)
	if err != nil {
		log.Printf("[ATTACHMENT] Failed to check attachment existence: %v", err)
		return nil, fmt.Errorf("failed to check attachment existence: %w", err)
	}

	if exists {
		// Attachment already exists, get its current path
		attachment, err := ae.attachmentManager.GetAttachment(hash)
		if err != nil {
			log.Printf("[ATTACHMENT] Failed to get existing attachment path: %v", err)
			return nil, fmt.Errorf("failed to get existing attachment path: %w", err)
		}

		log.Printf("[ATTACHMENT] Attachment already exists, referencing: %s", attachment.Path)
		return &AttachmentExtractionResult{
			Action:         ActionReferenced,
			Hash:           hash,
			Path:           attachment.Path,
			OriginalSize:   int64(len(decodedData)),
			UpdatePart:     true,
			ExtractionDate: time.Now().UTC(),
		}, nil
	}

	// Use new directory-based storage for new attachments
	storage := attachments.NewDirectoryAttachmentStorage(ae.repoRoot)

	// Create metadata
	metadata := attachments.AttachmentInfo{
		Hash:         hash,
		OriginalName: part.Filename,
		MimeType:     part.ContentType,
		Size:         int64(len(decodedData)),
		CreatedAt:    time.Now().UTC(),
		SourceMMS:    "", // Could be populated with MMS ID if available
	}

	// Store attachment with new format
	if err := storage.Store(hash, decodedData, metadata); err != nil {
		log.Printf("[ATTACHMENT] Failed to store attachment: %v", err)
		return nil, fmt.Errorf("failed to store attachment: %w", err)
	}

	// Get the path for the stored attachment
	attachmentPath, err := storage.GetPath(hash)
	if err != nil {
		log.Printf("[ATTACHMENT] Failed to get stored attachment path: %v", err)
		return nil, fmt.Errorf("failed to get stored attachment path: %w", err)
	}

	log.Printf("[ATTACHMENT] Successfully extracted attachment: %s (%d bytes)", attachmentPath, len(decodedData))
	return &AttachmentExtractionResult{
		Action:         ActionExtracted,
		Hash:           hash,
		Path:           attachmentPath,
		OriginalSize:   int64(len(decodedData)),
		UpdatePart:     true,
		ExtractionDate: time.Now().UTC(),
	}, nil
}

// AttachmentExtractionResult contains the result of an attachment extraction attempt
type AttachmentExtractionResult struct {
	Action         string    // ActionExtracted, "referenced", ActionSkipped
	Reason         string    // For skipped actions: "no-data", "content-type-filtered", "too-small"
	Hash           string    // SHA-256 hash of content (if extracted/referenced)
	Path           string    // Repository-relative path (if extracted/referenced)
	OriginalSize   int64     // Size of decoded content in bytes
	UpdatePart     bool      // Whether the MMS part should be updated
	ExtractionDate time.Time // When extraction occurred (UTC)
}

// UpdatePartWithExtraction updates an MMS part based on extraction result
func UpdatePartWithExtraction(part *MMSPart, result *AttachmentExtractionResult) {
	if !result.UpdatePart {
		return
	}

	// Remove original content and replace with path reference
	part.Data = ""
	part.Text = "" // Clear text content if it was extracted as attachment
	part.Path = result.Path
	part.OriginalSize = result.OriginalSize
	part.ExtractionDate = result.ExtractionDate.Format(time.RFC3339)
	part.AttachmentRef = result.Path // Also set internal tracking field
}

// ExtractAttachmentsFromMMS processes all parts of an MMS message for attachment extraction
func (ae *AttachmentExtractor) ExtractAttachmentsFromMMS(mms *MMS, config ContentTypeConfig) (*MMSExtractionSummary, error) {
	// Extract attachments from MMS message

	summary := &MMSExtractionSummary{
		MessageDate: mms.GetDate(),
		TotalParts:  len(mms.Parts),
		Results:     make([]*AttachmentExtractionResult, 0),
	}

	for i := range mms.Parts {
		// Process part for attachment extraction
		result, err := ae.ExtractAttachmentFromPart(&mms.Parts[i], config)
		if err != nil {
			// Error extracting part
			return nil, fmt.Errorf("failed to extract attachment from part %d: %w", i, err)
		}

		// Update the part if needed
		UpdatePartWithExtraction(&mms.Parts[i], result)
		// Part extraction complete

		summary.Results = append(summary.Results, result)

		// Update counters
		switch result.Action {
		case ActionExtracted:
			summary.ExtractedCount++
			summary.TotalExtractedSize += result.OriginalSize
		case ActionReferenced:
			summary.ReferencedCount++
			summary.TotalReferencedSize += result.OriginalSize
		case ActionSkipped:
			summary.SkippedCount++
		}
	}

	return summary, nil
}

// MMSExtractionSummary summarizes attachment extraction from a single MMS
type MMSExtractionSummary struct {
	MessageDate         int64                         // Message timestamp
	TotalParts          int                           // Total number of parts processed
	ExtractedCount      int                           // Number of new attachments extracted
	ReferencedCount     int                           // Number of existing attachments referenced
	SkippedCount        int                           // Number of parts skipped
	TotalExtractedSize  int64                         // Total bytes of new attachments
	TotalReferencedSize int64                         // Total bytes of referenced attachments
	Results             []*AttachmentExtractionResult // Detailed results per part
}

// AttachmentExtractionStats provides overall extraction statistics
type AttachmentExtractionStats struct {
	TotalMessages       int                          // Total messages processed
	MessagesWithParts   int                          // Messages that had parts
	TotalParts          int                          // Total parts processed
	ExtractedCount      int                          // New attachments extracted
	ReferencedCount     int                          // Existing attachments referenced
	SkippedCount        int                          // Parts skipped
	TotalExtractedSize  int64                        // Total bytes extracted
	TotalReferencedSize int64                        // Total bytes referenced
	ContentTypeStats    map[string]*ContentTypeStats // Stats by content type
	ErrorCount          int                          // Number of extraction errors
}

// ContentTypeStats provides statistics for a specific content type
type ContentTypeStats struct {
	ContentType     string // The content type
	ExtractedCount  int    // New attachments of this type
	ReferencedCount int    // Referenced attachments of this type
	SkippedCount    int    // Skipped parts of this type
	TotalSize       int64  // Total bytes for this type
}

// AddMMSExtractionSummary incorporates an MMS extraction summary into overall stats
func (stats *AttachmentExtractionStats) AddMMSExtractionSummary(summary *MMSExtractionSummary) {
	stats.TotalMessages++
	if summary.TotalParts > 0 {
		stats.MessagesWithParts++
	}

	stats.TotalParts += summary.TotalParts
	stats.ExtractedCount += summary.ExtractedCount
	stats.ReferencedCount += summary.ReferencedCount
	stats.SkippedCount += summary.SkippedCount
	stats.TotalExtractedSize += summary.TotalExtractedSize
	stats.TotalReferencedSize += summary.TotalReferencedSize
}

// NewAttachmentExtractionStats creates a new stats tracker
func NewAttachmentExtractionStats() *AttachmentExtractionStats {
	return &AttachmentExtractionStats{
		ContentTypeStats: make(map[string]*ContentTypeStats),
	}
}
