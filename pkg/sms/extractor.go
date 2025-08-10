package sms

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/attachments"
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

// ContentTypeConfig defines which content types to extract vs skip
type ContentTypeConfig struct {
	ExtractableTypes []string // Content types to extract
	SkippedTypes     []string // Content types to skip (leave inline)
}

// GetDefaultContentTypeConfig returns the default content type configuration
func GetDefaultContentTypeConfig() ContentTypeConfig {
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

// shouldExtractContentType determines if a content type should be extracted
func (ae *AttachmentExtractor) shouldExtractContentType(contentType string, config ContentTypeConfig) bool {
	// Normalize content type (remove parameters like charset)
	contentType = strings.ToLower(strings.Split(contentType, ";")[0])
	
	// Check if explicitly skipped
	for _, skipped := range config.SkippedTypes {
		if contentType == strings.ToLower(skipped) {
			return false
		}
	}
	
	// Check if explicitly extractable
	for _, extractable := range config.ExtractableTypes {
		if contentType == strings.ToLower(extractable) {
			return true
		}
	}
	
	// Default: don't extract unknown types
	return false
}

// ExtractAttachmentFromPart extracts an attachment from an MMS part if needed
func (ae *AttachmentExtractor) ExtractAttachmentFromPart(part *MMSPart, config ContentTypeConfig) (*AttachmentExtractionResult, error) {
	// Skip if no data to extract
	if part.Data == "" || part.Data == "null" {
		return &AttachmentExtractionResult{
			Action:     "skipped",
			Reason:     "no-data",
			UpdatePart: false,
		}, nil
	}
	
	// Skip if content type should not be extracted
	if !ae.shouldExtractContentType(part.ContentType, config) {
		return &AttachmentExtractionResult{
			Action:     "skipped", 
			Reason:     "content-type-filtered",
			UpdatePart: false,
		}, nil
	}
	
	// Skip small files (likely metadata)
	if len(part.Data) < 1024 { // Less than 1KB when base64 encoded
		return &AttachmentExtractionResult{
			Action:     "skipped",
			Reason:     "too-small",
			UpdatePart: false,
		}, nil
	}
	
	// Decode base64 data
	decodedData, err := base64.StdEncoding.DecodeString(part.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 data: %w", err)
	}
	
	// Calculate SHA-256 hash
	hasher := sha256.New()
	hasher.Write(decodedData)
	hash := fmt.Sprintf("%x", hasher.Sum(nil))
	
	// Check if attachment already exists
	exists, err := ae.attachmentManager.AttachmentExists(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to check attachment existence: %w", err)
	}
	
	attachmentPath := ae.attachmentManager.GetAttachmentPath(hash)
	
	if exists {
		// Attachment already exists, just reference it
		return &AttachmentExtractionResult{
			Action:         "referenced",
			Hash:           hash,
			Path:           attachmentPath,
			OriginalSize:   int64(len(decodedData)),
			UpdatePart:     true,
			ExtractionDate: time.Now().UTC(),
		}, nil
	}
	
	// Write attachment to disk
	fullPath := filepath.Join(ae.repoRoot, attachmentPath)
	
	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(fullPath), 0750); err != nil {
		return nil, fmt.Errorf("failed to create attachment directory: %w", err)
	}
	
	// Write file
	if err := os.WriteFile(fullPath, decodedData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write attachment file: %w", err)
	}
	
	return &AttachmentExtractionResult{
		Action:         "extracted",
		Hash:           hash,
		Path:           attachmentPath,
		OriginalSize:   int64(len(decodedData)),
		UpdatePart:     true,
		ExtractionDate: time.Now().UTC(),
	}, nil
}

// AttachmentExtractionResult contains the result of an attachment extraction attempt
type AttachmentExtractionResult struct {
	Action         string    // "extracted", "referenced", "skipped"
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
	
	// Remove base64 data and replace with path reference
	part.Data = ""
	part.Path = result.Path
	part.OriginalSize = result.OriginalSize
	part.ExtractionDate = result.ExtractionDate.Format(time.RFC3339)
	part.AttachmentRef = result.Path // Also set internal tracking field
}

// ExtractAttachmentsFromMMS processes all parts of an MMS message for attachment extraction  
func (ae *AttachmentExtractor) ExtractAttachmentsFromMMS(mms *MMS, config ContentTypeConfig) (*MMSExtractionSummary, error) {
	summary := &MMSExtractionSummary{
		MessageDate:    mms.GetDate(),
		TotalParts:     len(mms.Parts),
		Results:        make([]*AttachmentExtractionResult, 0),
	}
	
	for i := range mms.Parts {
		result, err := ae.ExtractAttachmentFromPart(&mms.Parts[i], config)
		if err != nil {
			return nil, fmt.Errorf("failed to extract attachment from part %d: %w", i, err)
		}
		
		// Update the part if needed
		UpdatePartWithExtraction(&mms.Parts[i], result)
		
		summary.Results = append(summary.Results, result)
		
		// Update counters
		switch result.Action {
		case "extracted":
			summary.ExtractedCount++
			summary.TotalExtractedSize += result.OriginalSize
		case "referenced": 
			summary.ReferencedCount++
			summary.TotalReferencedSize += result.OriginalSize
		case "skipped":
			summary.SkippedCount++
		}
	}
	
	return summary, nil
}

// MMSExtractionSummary summarizes attachment extraction from a single MMS
type MMSExtractionSummary struct {
	MessageDate          int64                        // Message timestamp
	TotalParts           int                          // Total number of parts processed
	ExtractedCount       int                          // Number of new attachments extracted
	ReferencedCount      int                          // Number of existing attachments referenced
	SkippedCount         int                          // Number of parts skipped
	TotalExtractedSize   int64                        // Total bytes of new attachments
	TotalReferencedSize  int64                        // Total bytes of referenced attachments
	Results              []*AttachmentExtractionResult // Detailed results per part
}

// AttachmentExtractionStats provides overall extraction statistics
type AttachmentExtractionStats struct {
	TotalMessages        int                             // Total messages processed
	MessagesWithParts    int                             // Messages that had parts
	TotalParts           int                             // Total parts processed
	ExtractedCount       int                             // New attachments extracted
	ReferencedCount      int                             // Existing attachments referenced
	SkippedCount         int                             // Parts skipped
	TotalExtractedSize   int64                           // Total bytes extracted
	TotalReferencedSize  int64                           // Total bytes referenced
	ContentTypeStats     map[string]*ContentTypeStats    // Stats by content type
	ErrorCount           int                             // Number of extraction errors
}

// ContentTypeStats provides statistics for a specific content type
type ContentTypeStats struct {
	ContentType      string // The content type
	ExtractedCount   int    // New attachments of this type
	ReferencedCount  int    // Referenced attachments of this type
	SkippedCount     int    // Skipped parts of this type
	TotalSize        int64  // Total bytes for this type
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