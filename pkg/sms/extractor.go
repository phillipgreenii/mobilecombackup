package sms

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/attachments"
	"github.com/phillipgreen/mobilecombackup/pkg/logging"
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
	logger            logging.Logger
}

// NewAttachmentExtractor creates a new attachment extractor with default (null) logger
func NewAttachmentExtractor(repoRoot string) *AttachmentExtractor {
	return &AttachmentExtractor{
		attachmentManager: attachments.NewAttachmentManager(repoRoot),
		repoRoot:          repoRoot,
		logger:            logging.NewNullLogger(),
	}
}

// NewAttachmentExtractorWithLogger creates a new attachment extractor with a specific logger
func NewAttachmentExtractorWithLogger(repoRoot string, logger logging.Logger) *AttachmentExtractor {
	return &AttachmentExtractor{
		attachmentManager: attachments.NewAttachmentManager(repoRoot),
		repoRoot:          repoRoot,
		logger:            logger,
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

// determineExtractableContent determines what content to extract from the MMS part
func (ae *AttachmentExtractor) determineExtractableContent(part *MMSPart) (string, bool, *AttachmentExtractionResult) {
	// Log extraction attempt for debugging and auditing
	ae.logger.Debug().
		Str("content_type", part.ContentType).
		Int("data_len", len(part.Data)).
		Int("text_len", len(part.Text)).
		Str("content_disp", part.ContentDisp).
		Str("filename", part.Filename).
		Msg("Processing MMS part")

	// Check if this is explicitly marked as an attachment
	isExplicitAttachment := strings.ToLower(part.ContentDisp) == "attachment"

	// Determine what content we have to extract
	var contentToExtract string
	var isBase64 bool

	switch {
	case part.Data != "" && part.Data != "null":
		// Binary data (base64 encoded)
		contentToExtract = part.Data
		isBase64 = true
		ae.logger.Debug().Int("size_bytes", len(contentToExtract)).Msg("Found base64 data")
	case part.Text != "" && part.Text != "null" && isExplicitAttachment:
		// Text content marked as attachment
		contentToExtract = part.Text
		isBase64 = false
		ae.logger.Debug().Int("size_bytes", len(contentToExtract)).Msg("Found text content marked as attachment")
	default:
		// No extractable content - skip
		ae.logger.Debug().
			Str("data", part.Data).
			Str("text", part.Text).
			Bool("explicit_attachment", isExplicitAttachment).
			Msg("No extractable content found")
		return "", false, &AttachmentExtractionResult{
			Action:     ActionSkipped,
			Reason:     "no-data",
			UpdatePart: false,
		}
	}

	return contentToExtract, isBase64, nil
}

// validateExtractionRequirements validates content type and size requirements
func (ae *AttachmentExtractor) validateExtractionRequirements(
	part *MMSPart, contentToExtract string, isBase64 bool, config ContentTypeConfig,
) *AttachmentExtractionResult {
	// Check if this is explicitly marked as an attachment
	isExplicitAttachment := strings.ToLower(part.ContentDisp) == "attachment"

	// Check content type extraction decision with comprehensive logging
	decision := ae.shouldExtractContentType(part.ContentType, isExplicitAttachment, config)
	if !decision.ShouldExtract {
		ae.logger.Debug().
			Str("reason", decision.Reason).
			Str("content_type", part.ContentType).
			Str("category", decision.Category).
			Msg("Content type filtered out")
		return &AttachmentExtractionResult{
			Action:     ActionSkipped,
			Reason:     "content-type-filtered",
			UpdatePart: false,
		}
	}

	ae.logger.Debug().
		Str("content_type", part.ContentType).
		Str("reason", decision.Reason).
		Msg("Content type approved for extraction")

	// For text content, skip size check as text attachments can be small
	// For binary content, skip small files (likely metadata)
	if isBase64 && len(contentToExtract) < 1024 {
		// Too small - skip
		ae.logger.Debug().Int("size_bytes", len(contentToExtract)).Int("threshold", 1024).Msg("Skipping small binary content")
		return &AttachmentExtractionResult{
			Action:     ActionSkipped,
			Reason:     "too-small",
			UpdatePart: false,
		}
	}

	// Proceed with extraction
	ae.logger.Debug().Str("content_type", part.ContentType).Msg("Proceeding with extraction")
	return nil
}

// decodeContent decodes the content based on whether it's base64 or text
func (ae *AttachmentExtractor) decodeContent(contentToExtract string, isBase64 bool) ([]byte, error) {
	// Decode content based on type
	var decodedData []byte
	var err error

	if isBase64 {
		// Decode base64 data
		ae.logger.Debug().Int("chars", len(contentToExtract)).Msg("Decoding base64 data")
		decodedData, err = base64.StdEncoding.DecodeString(contentToExtract)
		if err != nil {
			// Failed to decode base64 data
			ae.logger.Debug().Err(err).Msg("Failed to decode base64 data")
			return nil, fmt.Errorf("failed to decode base64 data: %w", err)
		}
		ae.logger.Debug().Int("bytes", len(decodedData)).Msg("Successfully decoded base64 data")
	} else {
		// Use text content as-is (UTF-8 encoded)
		decodedData = []byte(contentToExtract)
		ae.logger.Debug().Int("bytes", len(decodedData)).Msg("Using text content as-is")
	}

	return decodedData, nil
}

// handleAttachmentStorage handles either referencing existing attachments or creating new ones
func (ae *AttachmentExtractor) handleAttachmentStorage(part *MMSPart, decodedData []byte) (*AttachmentExtractionResult, error) {
	// Calculate SHA-256 hash
	hasher := sha256.New()
	hasher.Write(decodedData)
	hash := fmt.Sprintf("%x", hasher.Sum(nil))
	ae.logger.Debug().Str("hash", hash).Msg("Calculated attachment hash")

	// Check if attachment already exists
	exists, err := ae.attachmentManager.AttachmentExists(hash)
	if err != nil {
		ae.logger.Debug().Err(err).Msg("Failed to check attachment existence")
		return nil, fmt.Errorf("failed to check attachment existence: %w", err)
	}

	if exists {
		return ae.referenceExistingAttachment(hash, decodedData)
	}

	return ae.createNewAttachment(part, decodedData, hash)
}

// referenceExistingAttachment creates a result for an existing attachment
func (ae *AttachmentExtractor) referenceExistingAttachment(hash string, decodedData []byte) (*AttachmentExtractionResult, error) {
	// Attachment already exists, get its current path
	attachment, err := ae.attachmentManager.GetAttachment(hash)
	if err != nil {
		ae.logger.Debug().Err(err).Msg("Failed to get existing attachment path")
		return nil, fmt.Errorf("failed to get existing attachment path: %w", err)
	}

	ae.logger.Debug().Str("path", attachment.Path).Msg("Attachment already exists, referencing")
	return &AttachmentExtractionResult{
		Action:         ActionReferenced,
		Hash:           hash,
		Path:           attachment.Path,
		OriginalSize:   int64(len(decodedData)),
		UpdatePart:     true,
		ExtractionDate: time.Now().UTC(),
	}, nil
}

// createNewAttachment creates and stores a new attachment
func (ae *AttachmentExtractor) createNewAttachment(
	part *MMSPart, decodedData []byte, hash string,
) (*AttachmentExtractionResult, error) {
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
		ae.logger.Debug().Err(err).Msg("Failed to store attachment")
		return nil, fmt.Errorf("failed to store attachment: %w", err)
	}

	// Get the path for the stored attachment
	attachmentPath, err := storage.GetPath(hash)
	if err != nil {
		ae.logger.Debug().Err(err).Msg("Failed to get stored attachment path")
		return nil, fmt.Errorf("failed to get stored attachment path: %w", err)
	}

	ae.logger.Debug().Str("path", attachmentPath).Int("bytes", len(decodedData)).Msg("Successfully extracted attachment")
	return &AttachmentExtractionResult{
		Action:         ActionExtracted,
		Hash:           hash,
		Path:           attachmentPath,
		OriginalSize:   int64(len(decodedData)),
		UpdatePart:     true,
		ExtractionDate: time.Now().UTC(),
	}, nil
}

// shouldExtractContentType determines if a content type should be extracted with detailed decision logging
func (ae *AttachmentExtractor) shouldExtractContentType(
	contentType string,
	isExplicitAttachment bool,
	_ ContentTypeConfig,
) ContentDecision {
	decision := ContentDecision{
		ContentType: contentType,
	}

	// Handle edge cases first
	if contentType == "" {
		decision.ShouldExtract = false
		decision.Reason = "missing content type header"
		decision.Category = CategoryUnknown
		ae.logger.Debug().Interface("decision", decision).Msg("Content type decision: missing content type header")
		return decision
	}

	// Normalize content type (remove parameters like charset)
	normalizedType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))

	if normalizedType == "" {
		decision.ShouldExtract = false
		decision.Reason = "empty content type after normalization"
		decision.Category = CategoryUnknown
		ae.logger.Debug().Interface("decision", decision).Msg("Content type decision: empty after normalization")
		return decision
	}

	// Check against binary whitelist first
	if BinaryContentTypes[normalizedType] {
		decision.ShouldExtract = true
		decision.Reason = "whitelisted binary type"
		decision.Category = "binary"
		ae.logger.Debug().Interface("decision", decision).Msg("Content type decision: whitelisted binary type")
		return decision
	}

	// Check if it's a known text type
	if TextContentTypes[normalizedType] {
		decision.ShouldExtract = false
		decision.Reason = "text content - keeping inline"
		decision.Category = "text"
		ae.logger.Debug().Interface("decision", decision).Msg("Content type decision: text content kept inline")
		return decision
	}

	// Unknown content type - reject with detailed logging for manual review
	decision.ShouldExtract = false
	decision.Reason = fmt.Sprintf("unknown content type: %s", normalizedType)
	decision.Category = CategoryUnknown
	ae.logger.Debug().
		Interface("decision", decision).
		Bool("explicit_attachment", isExplicitAttachment).
		Msg("Content type decision: unknown content type")
	return decision
}

// ExtractAttachmentFromPart extracts an attachment from an MMS part if needed
func (ae *AttachmentExtractor) ExtractAttachmentFromPart(
	part *MMSPart, config ContentTypeConfig,
) (*AttachmentExtractionResult, error) {
	// Determine extractable content
	contentToExtract, isBase64, skipResult := ae.determineExtractableContent(part)
	if skipResult != nil {
		return skipResult, nil
	}

	// Validate content type and size requirements
	if result := ae.validateExtractionRequirements(part, contentToExtract, isBase64, config); result != nil {
		return result, nil
	}

	// Decode the content
	decodedData, err := ae.decodeContent(contentToExtract, isBase64)
	if err != nil {
		return nil, err
	}

	// Handle attachment storage (either reference existing or create new)
	return ae.handleAttachmentStorage(part, decodedData)
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
